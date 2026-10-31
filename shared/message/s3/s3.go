package s3

const TopicName string = "s3-events"

const ObjectCreatedPut string = "s3:ObjectCreated:Put"

// Record describes an object notification emitted by S3-compatible storage.
type Record struct {
	EventName string `json:"eventName"`
	S3        struct {
		Bucket struct {
			Name string `json:"name"`
		} `json:"bucket"`
		Object struct {
			Key          string            `json:"key"`
			Size         int64             `json:"size"`
			ContentType  string            `json:"contentType"`
			UserMetadata map[string]string `json:"userMetadata"`
		} `json:"object"`
	} `json:"s3"`
}

// Message wraps the records delivered in a storage notification.
type Message struct {
	Records []Record `json:"Records"`
}
