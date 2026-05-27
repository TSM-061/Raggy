package projectreportmd

import (
	"bytes"
	"context"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/frontmatter"
)

type ProjectReport struct {
	Frontmatter Frontmatter
	Sections    []Section
}

type Frontmatter struct {
	Title   string `yaml:"title"`
	Subject string `yaml:"subject"`
}

type Section struct {
	Subheading string
	Content    string
}

func Parse(ctx context.Context, data []byte) (*ProjectReport, error) {
	markdown := goldmark.New(goldmark.WithExtensions(&frontmatter.Extender{}))
	reader := text.NewReader(data)
	parserCtx := parser.NewContext()
	doc := markdown.Parser().Parse(reader, parser.WithContext(parserCtx))

	fmData := frontmatter.Get(parserCtx)
	var fm Frontmatter
	if err := fmData.Decode(&fm); err != nil {
		return nil, fmt.Errorf("could not parse fontmatter: %v", err)
	}

	var sections []Section
	currHeading := ""

	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		switch node := child.(type) {

		case *ast.Heading:
			currHeading = string(node.Lines().Value(data))

		case *ast.Paragraph:
			paragraphText := extractText(node, data)

			sections = append(sections, Section{
				Subheading: currHeading,
				Content:    paragraphText,
			})

		case *ast.List:
			// Goldmark treats lists as distinct blocks, not paragraphs.
			listText := extractListText(node, data)

			sections = append(sections, Section{
				Subheading: currHeading,
				Content:    listText,
			})
		}
	}

	return &ProjectReport{
		Frontmatter: fm,
		Sections:    sections,
	}, nil
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
