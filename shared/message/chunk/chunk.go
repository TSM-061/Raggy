package chunk

const TopicName string = "chunks"

type StreamMessage struct {
	UploadID       string `json:"uploadId"`
	UploadFilename string `json:"uploadFilename"`

	EventName EventName `json:"eventName"`

	ChunkIndex int `json:"chunkIndex"`
	ChunkTotal int `json:"chunkTotal"`

	Payload string `json:"payload"`
}

type EventName string

const (
	Stream EventName = "chunk.stream"
)
