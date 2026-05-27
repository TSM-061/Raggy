package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/TSM-061/Raggy/dashboard/internal/services"
	"github.com/TSM-061/Raggy/dashboard/internal/upload"
	"github.com/TSM-061/Raggy/shared/auth"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/google/uuid"
)

type createUploadRequest struct {
	OriginalName string `json:"originalName" validate:"required"`
	ContentType  string `json:"contentType" validate:"required"`
	ProfileHint  string `json:"profileHint" validate:"required"`
	SizeBytes    int64  `json:"sizeBytes" validate:"gt=0"`
}

type createUploadResponse struct {
	Upload    uploadResponse `json:"upload"`
	UploadURL string `json:"uploadUrl"`
}

type listUploadsResponse struct {
	Uploads    []uploadResponse `json:"uploads"`
	PageNumber int              `json:"pageNumber"`
	PageSize   int              `json:"pageSize"`
	Total      int64            `json:"total"`
}

type uploadResponse struct {
	ID           string    `json:"id"`
	UploadedBy   string    `json:"uploadedBy"`
	OriginalName string    `json:"originalName"`
	ContentType  string    `json:"contentType"`
	ProfileHint  string    `json:"profileHint"`
	SizeBytes    int64     `json:"sizeBytes"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func toUploadResponse(u *upload.Upload) uploadResponse {
	return uploadResponse{
		ID:           u.ID.String(),
		UploadedBy:   u.UploadedBy.String(),
		OriginalName: u.OriginalName,
		ContentType:  u.ContentType,
		ProfileHint:  u.ProfileHint,
		SizeBytes:    u.SizeBytes,
		Status:       string(u.Status),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func (s *Server) HandleUpload(w http.ResponseWriter, r *http.Request) {
	uploaderID := auth.MustGetUserID(r.Context())

	var body createUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.validator.Struct(body); err != nil {

		if !ValidationProblem(w, err) {
			serviceerr.WriteHTTPError(w, err)
		}
		return
	}

	result, err := s.Application.UploadService.CreateUpload(r.Context(), &services.CreateUploadCommand{
		UploadedBy:   uploaderID,
		OriginalName: body.OriginalName,
		ContentType:  body.ContentType,
		ProfileHint:  body.ProfileHint,
		SizeBytes:    body.SizeBytes,
	})
	if err != nil {
		log.Printf("%v", err)
		serviceerr.WriteHTTPError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(&createUploadResponse{
		Upload:    toUploadResponse(result.Upload),
		UploadURL: result.UploadURL,
	}); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

func (s *Server) HandleListUploads(w http.ResponseWriter, r *http.Request) {
	page, err := parseOptionalIntQuery(r, "page")
	if err != nil {
		serviceerr.WriteHTTPError(w, err)
		return
	}

	pageSize, err := parseOptionalIntQuery(r, "pageSize")
	if err != nil {
		serviceerr.WriteHTTPError(w, err)
		return
	}

	result, err := s.Application.UploadService.ListUploads(r.Context(), &services.ListUploadsQuery{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		log.Printf("%v", err)
		serviceerr.WriteHTTPError(w, err)
		return
	}

	uploads := make([]uploadResponse, 0, len(result.Uploads))
	for index := range result.Uploads {
		uploads = append(uploads, toUploadResponse(&result.Uploads[index]))
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&listUploadsResponse{
		Uploads:    uploads,
		PageNumber: result.PageNumber,
		PageSize:   result.PageSize,
		Total:      result.Total,
	}); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

func (s *Server) HandleDeleteUpload(w http.ResponseWriter, r *http.Request) {
	uploadID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		serviceerr.WriteHTTPError(w, fmt.Errorf("%w: invalid upload id", serviceerr.InvalidInput))
		return
	}

	err = s.Application.UploadService.DeleteUpload(r.Context(), &services.DeleteUploadCommand{
		UploadID: uploadID,
	})
	if err != nil {
		log.Printf("%v", err)
		serviceerr.WriteHTTPError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseOptionalIntQuery(r *http.Request, key string) (int, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%w: query '%s' must be an integer", serviceerr.InvalidInput, key)
	}

	return parsed, nil
}
