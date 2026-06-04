package serviceerr

import (
	"errors"
	"net/http"
)

type HTTPError struct {
	StatusCode int
	Message    string
}

// TODO fix this initial implementation

func MapToHTTPError(err error) HTTPError {
	switch {
	case errors.Is(err, InvalidInput):
		return HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid input"}
	case errors.Is(err, NotFound):
		return HTTPError{StatusCode: http.StatusNotFound, Message: "resource not found"}
	case errors.Is(err, Unauthorized):
		return HTTPError{StatusCode: http.StatusUnauthorized, Message: "unauthorized"}
	case errors.Is(err, Conflict):
		return HTTPError{StatusCode: http.StatusConflict, Message: "resource already exists"}
	default:
		return HTTPError{StatusCode: http.StatusInternalServerError, Message: "internal error"}
	}
}

func WriteHTTPError(w http.ResponseWriter, err error) {
	httpErr := MapToHTTPError(err)
	http.Error(w, httpErr.Message, httpErr.StatusCode)
}
