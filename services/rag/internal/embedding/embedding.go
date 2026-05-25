package embedding

import "context"

type Embedder interface {
	Embed(ctx context.Context, tokens string) []float32
}
