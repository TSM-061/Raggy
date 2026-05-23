package upload

type Status string

const (
	StatusPending   Status = "pending"
	StatusUploaded  Status = "uploaded"
	StatusProcessed Status = "processed"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusUploaded, StatusProcessed:
		return true
	default:
		return false
	}
}