package projectreportmd

import "fmt"

type Content struct {
	Title      string `json:"title"`
	Subject    string `json:"subject"`
	Subheading string `json:"subheading"`
	Content    string `json:"content"`
}

func ContentAsString(context Content) string {
	return fmt.Sprintf(
		"Document: %s\nSubject: %s\nSection: %s\nContent: %s",
		context.Title,
		context.Subject,
		context.Subheading,
		context.Content,
	)
}
