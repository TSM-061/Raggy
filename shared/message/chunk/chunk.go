package chunk

import "encoding/json"

const TopicName string = "chunks"

type Message struct {
	UploadID string `json:"uploadId"`

	Type Type `json:"type"`

	Payload json.RawMessage `json:"payload"`
}

type StreamPayload struct {
	Index int `json:"index"`
	Total int `json:"total"`

	// Content formatted as json metadata
	Content json.RawMessage `json:"content"`
	// Content formatted as singular string for embedding
	ContentString string `json:"contentString"`
}

type Type string

const (
	StreamEvent Type = "chunk.stream"
)
