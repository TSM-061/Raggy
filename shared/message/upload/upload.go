package upload

const TopicName string = "uploads"

type MessageType string

const (
	Completed MessageType = "upload.completed"
	Error     MessageType = "upload.failed"
)

type ProfileHint string

const (
	ProjectReportMd ProfileHint = "profile-hint.project-report-md"
)

type CompletedMessage struct {
	UploadID string `json:"uploadId"`

	Type        MessageType `json:"type"`
	ProfileHint ProfileHint `json:"profileHint"`
}

type ErrorMessage struct {
	UploadID string `json:"uploadId"`

	Type  MessageType `json:"type"`
	Stage string      `json:"stage"`
	Error string      `json:"error"`
}
