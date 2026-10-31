package upload

import "fmt"

const ProgressTopic = "upload-progress"
const Progress MessageType = "upload.progress"

// ProgressMessage is a cumulative snapshot of successfully persisted embeddings.
// Counts are scoped to an immutable upload; reprocessing requires a new upload ID.
type ProgressMessage struct {
	Type            MessageType `json:"type"`
	UploadID        string      `json:"uploadId"`
	ChunkTotal      int         `json:"chunkTotal"`
	ChunksProcessed int         `json:"chunksProcessed"`
}

// Validate checks that a progress snapshot has a supported type and consistent counts.
func (m ProgressMessage) Validate() error {
	if m.Type != Progress || m.ChunkTotal < 0 || m.ChunksProcessed < 0 || m.ChunksProcessed > m.ChunkTotal {
		return fmt.Errorf("invalid upload progress counts or event type")
	}
	return nil
}
