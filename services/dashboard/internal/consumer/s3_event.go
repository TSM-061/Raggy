package consumer

type S3Record struct {
	EventName string `json:"eventName"`
	S3        struct {
		Bucket struct {
			Name string `json:"name"`
		} `json:"bucket"`
		Object struct {
			Key         string `json:"key"`
			Size        int64  `json:"size"`
			ContentType string `json:"contentType"`
		} `json:"object"`
	} `json:"s3"`
}

type S3Message struct {
	Records []S3Record `json:"Records"`
}
