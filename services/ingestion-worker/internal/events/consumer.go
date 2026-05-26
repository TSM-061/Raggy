package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/TSM-061/Raggy/shared/message/chunk"
	"github.com/TSM-061/Raggy/shared/message/upload"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/frontmatter"

	"gocloud.dev/blob"
)

type Consumer struct {
	client        *kgo.Client
	uploadsBucket *blob.Bucket
}

func NewConsumer(client *kgo.Client, uploadsBucket *blob.Bucket) *Consumer {
	return &Consumer{
		uploadsBucket: uploadsBucket,
		client:        client,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	log.Println("Kafka ready")

	for {
		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			log.Printf("Kafka poll errors encountered: %v", errs)
		}

		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()

			switch record.Topic {
			case "uploads":
				c.handleUploadEvent(ctx, record)
			}
		}
	}
}

func (c *Consumer) handleUploadEvent(ctx context.Context, record *kgo.Record) error {
	var msg upload.CompletedMessage

	if err := json.Unmarshal(record.Value, &msg); err != nil {
		log.Printf("failed to decode s3 event payload: %v", err)
		return err
	}

	if msg.EventType != upload.Completed {
		return nil
	}

	data, err := c.uploadsBucket.ReadAll(ctx, msg.UploadID)
	if err != nil {
		log.Printf("failed to read uploads bucket: %v", err)
		return err
	}

	switch msg.ProfileHint {
	case upload.ProjectReportMarkdown:
		c.handleProjectReportMarkdown(ctx, msg.UploadID, data)
	}

	return nil
}

type section struct {
	Subheading string
	Content    string
}

func (c *Consumer) handleProjectReportMarkdown(ctx context.Context, uploadID string, data []byte) error {
	markdown := goldmark.New(goldmark.WithExtensions(&frontmatter.Extender{}))
	reader := text.NewReader(data)
	doc := markdown.Parser().Parse(reader)

	var sections []section
	currHeading := ""

	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		switch node := child.(type) {

		case *ast.Heading:
			currHeading = string(node.Lines().Value(data))

		case *ast.Paragraph:
			paragraphText := extractText(node, data)

			sections = append(sections, section{
				Subheading: currHeading,
				Content:    paragraphText,
			})

		case *ast.List:
			// Goldmark treats lists as distinct blocks, not paragraphs.
			listText := extractListText(node, data)

			sections = append(sections, section{
				Subheading: currHeading,
				Content:    listText,
			})
		}
	}

	messages := make([]*kgo.Record, len(sections))
	for i, s := range sections {
		msg, err := json.Marshal(chunk.StreamMessage{
			UploadID:       uploadID,
			UploadFilename: "TODO",
			EventName:      chunk.Stream,
			ChunkIndex:     i,
			ChunkTotal:     len(sections),
			Payload:        fmt.Sprintf("Subheading: %s\nContent:%s", s.Subheading, s.Content),
		})
		if err != nil {
			return err
		}

		messages[i] = &kgo.Record{
			Key:   []byte(uploadID),
			Topic: chunk.TopicName,
			Value: msg,
		}
	}

	if err := c.client.ProduceSync(ctx, messages...).FirstErr(); err != nil {
		log.Printf("failed to produce messages for %s: %v", uploadID, err)
		return err
	}

	return nil
}

func extractText(node ast.Node, source []byte) string {
	var buf []byte

	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		buf = append(buf, line.Value(source)...)
	}

	return string(bytes.ReplaceAll(bytes.TrimSpace(buf), []byte("\n"), []byte(" ")))
}

func extractListText(list *ast.List, source []byte) string {
	var buf bytes.Buffer

	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		if item.Kind() != ast.KindListItem {
			continue
		}

		// Can contain paragraphs or raw text lines
		for child := item.FirstChild(); child != nil; child = child.NextSibling() {
			buf.WriteString(extractText(child, source))
		}

		if item.NextSibling() != nil {
			// Separate bullet points
			buf.WriteString(" ")
		}
	}
	return buf.String()
}
