package upload

const TopicName string = "uploads"

type EventType string

const (
	Completed EventType = "upload.completed"
	Error     EventType = "upload.failed"
)

type ProfileHint string

const (
	ProjectReportMd ProfileHint = "profile-hint.project-report-md"
)

type CompletedMessage struct {
	UploadID string `json:"uploadId"`

	EventType   EventType   `json:"eventName"`
	ProfileHint ProfileHint `json:"profileHint"`
}

type ErrorMessage struct {
	UploadID string `json:"uploadId"`

	EventType EventType `json:"eventName"`
	Stage     string    `json:"stage"`
	Error     string    `json:"error"`
}
