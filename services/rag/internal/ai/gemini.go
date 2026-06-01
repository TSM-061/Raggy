package ai

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

type GeminiEmbedderConfig struct {
	APIKey string

	EmbeddingModel      string
	EmbeddingDimensions *int32

	GenerationModel string
}

type GeminiClient struct {
	client              *genai.Client
	embeddingModel      string
	embeddingDimensions *int32
	generationModel     string
}

func NewGeminiClient(ctx context.Context, cfg *GeminiEmbedderConfig) (*GeminiClient, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: cfg.APIKey,
	})

	if err != nil {
		return nil, fmt.Errorf("initialize gemini client: %w", err)
	}

	return &GeminiClient{
		client:              client,
		embeddingModel:      cfg.EmbeddingModel,
		embeddingDimensions: cfg.EmbeddingDimensions,
		generationModel:     cfg.GenerationModel,
	}, nil
}

func (g *GeminiClient) Embed(ctx context.Context, tokens string) ([]float32, error) {
	contents := []*genai.Content{
		genai.NewContentFromText(tokens, genai.RoleUser),
	}

	result, err := g.client.Models.EmbedContent(ctx,
		g.embeddingModel,
		contents,
		&genai.EmbedContentConfig{OutputDimensionality: g.embeddingDimensions},
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

func (g *GeminiClient) GenerateContent(ctx context.Context, prompt string) (string, error) {
	result, err := g.client.Models.GenerateContent(
		ctx,
		g.generationModel,
		genai.Text(prompt),
		&genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				ThinkingLevel: genai.ThinkingLevelLow,
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("generate gemini response: %w", err)
	}

	return result.Text(), nil
}
