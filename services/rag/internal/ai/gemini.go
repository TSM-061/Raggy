package ai

import (
	"context"
	"fmt"

	"github.com/TSM-061/Raggy/rag/internal/rag"

	"go.opentelemetry.io/otel"
	"google.golang.org/genai"
)

type GeminiConfig struct {
	APIKey              string `env:"API_KEY"`
	EmbeddingModel      string `env:"EMBEDDING_MODEL" envDefault:"gemini-embedding-2"`
	EmbeddingDimensions *int32 `env:"EMBEDDING_DIMENSIONS" envDefault:"1536"`
	GenerationModel     string `env:"GENERATION_MODEL" envDefault:"gemini-3.5-flash"`
}

type GeminiClient struct {
	client              *genai.Client
	embeddingModel      string
	embeddingDimensions *int32
	generationModel     string
}

var tracer = otel.Tracer("github.com/TSM-061/Raggy/rag/internal/ai")

func NewGeminiClient(ctx context.Context, cfg *GeminiConfig) (*GeminiClient, error) {
	if err := ValidateProvider(ProviderGemini, cfg); err != nil {
		return nil, err
	}
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

// Embed converts text into a vector using the configured Gemini model.
func (g *GeminiClient) Embed(ctx context.Context, tokens string) ([]float32, error) {
	contents := []*genai.Content{
		genai.NewContentFromText(tokens, genai.RoleUser),
	}

	ctx, span := tracer.Start(ctx, fmt.Sprintf("embeddings %s", g.embeddingModel))
	result, err := g.client.Models.EmbedContent(ctx,
		g.embeddingModel,
		contents,
		&genai.EmbedContentConfig{OutputDimensionality: g.embeddingDimensions},
	)
	span.End()
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

// GenerateContent requests and decodes a structured answer from Gemini.
func (g *GeminiClient) GenerateContent(ctx context.Context, prompt string) (*rag.GeneratedAnswer, error) {
	result, err := g.client.Models.GenerateContent(
		ctx,
		g.generationModel,
		genai.Text(prompt),
		&genai.GenerateContentConfig{
			ResponseMIMEType: "application/json",
			ResponseSchema:   answerSchema(),
			ThinkingConfig: &genai.ThinkingConfig{
				ThinkingLevel: genai.ThinkingLevelLow,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("generate gemini response: %w", err)
	}

	if result == nil {
		return nil, rag.ErrInvalidGeneration
	}
	return rag.ParseGeneratedAnswer(result.Text())
}

// answerSchema defines the generated answer shape expected by citation validation.
func answerSchema() *genai.Schema {
	return &genai.Schema{Type: genai.TypeObject, Required: []string{"status", "answer", "references"}, Properties: map[string]*genai.Schema{
		"status":     {Type: genai.TypeString, Enum: []string{"answered", "insufficient_evidence"}},
		"answer":     {Type: genai.TypeString},
		"references": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
	}}
}
