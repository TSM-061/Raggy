package upload

type Status string

const (
	// The upload record exists, chunks are streaming in
	StatusProcessing Status = "processing"
	// All text chunks have successfully hit database
	StatusIngested Status = "ingested"
	// Background workers are currently calculating vector embeddings
	StatusEmbedding Status = "embedding"
	// Vector generation is complete
	StatusReady Status = "ready"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusProcessing, StatusIngested, StatusEmbedding, StatusReady:
		return true
	default:
		return false
	}
}
