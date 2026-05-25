package embedding

import (
	"context"
	"log"

	"google.golang.org/genai"
)

type GeminiEmbedder struct {
	client     *genai.Client
	model      string
	dimensions *int32
}

func NewGeminiEmbedder(client *genai.Client, model string, dimensions *int32) *GeminiEmbedder {
	return &GeminiEmbedder{
		client:     client,
		model:      model,
		dimensions: dimensions,
	}
}

func (g *GeminiEmbedder) Embed(ctx context.Context, tokens string) []float32 {
	contents := []*genai.Content{
		genai.NewContentFromText(tokens, genai.RoleUser),
	}

	result, err := g.client.Models.EmbedContent(ctx,
		g.model,
		contents,
		&genai.EmbedContentConfig{OutputDimensionality: g.dimensions},
	)
	if err != nil {
		log.Fatal(err)
	}

	if len(result.Embeddings) != 1 {
		log.Fatalf(
			"Incorrect number of embeddings returned: wanted 1, got %d",
			len(result.Embeddings),
		)
	}

	return result.Embeddings[0].Values
}
