package upload

type Status string

const (
	StatusPending    Status = "pending"
	StatusUploaded   Status = "uploaded"
	StatusProcessing Status = "processing"
	StatusProcessed  Status = "processed"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusUploaded, StatusProcessing, StatusProcessed:
		return true
	default:
		return false
	}
}
