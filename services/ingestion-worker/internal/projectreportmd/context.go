package projectreportmd

import "fmt"

type Context struct {
	Title      string `json:"title"`
	Subject    string `json:"subject"`
	Subheading string `json:"subheading"`
	Content    string `json:"content"`
}

func ContextAsString(context Context) string {
	return fmt.Sprintf(
		"Document: %s\nSubject: %s\nSection: %s\nContent: %s",
		context.Title,
		context.Subject,
		context.Subheading,
		context.Content,
	)
}
