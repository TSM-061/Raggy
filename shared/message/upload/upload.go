package upload

const TopicName string = "uploads"

type CompletedMessage struct {
	UploadID string `json:"uploadId"`

	EventType   EventType   `json:"eventName"`
	ProfileHint ProfileHint `json:"profileHint"`
}

type EventType string

const (
	Completed EventType = "upload.completed"
)

type ProfileHint string

const (
	ProjectReportMarkdown ProfileHint = "profile-hint.project-report-markdown"
)
