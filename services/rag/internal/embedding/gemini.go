package embedding

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

type GeminiEmbedderConfig struct {
	Model      string
	Dimensions *int32
	APIKey     string
}

type GeminiEmbedder struct {
	client     *genai.Client
	model      string
	dimensions *int32
}

func NewGeminiEmbedder(ctx context.Context, config *GeminiEmbedderConfig) (*GeminiEmbedder, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: config.APIKey,
	})

	if err != nil {
		return nil, fmt.Errorf("initialize gemini client: %w", err)
	}

	return &GeminiEmbedder{
		client:     client,
		model:      config.Model,
		dimensions: config.Dimensions,
	}, nil
}

func (g *GeminiEmbedder) Embed(ctx context.Context, tokens string) ([]float32, error) {
	contents := []*genai.Content{
		genai.NewContentFromText(tokens, genai.RoleUser),
	}

	result, err := g.client.Models.EmbedContent(ctx,
		g.model,
		contents,
		&genai.EmbedContentConfig{OutputDimensionality: g.dimensions},
	)
	if err != nil {
		return nil, fmt.Errorf("generate gemini embedding: %w", err)
	}

	if len(result.Embeddings) != 1 {
		return nil, fmt.Errorf(
			"unexpected embeddings result length: wanted 1, got %d",
			len(result.Embeddings),
		)
	}

	return result.Embeddings[0].Values, nil
}
