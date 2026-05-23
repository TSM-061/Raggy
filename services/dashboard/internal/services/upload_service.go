package services

import (
	"context"
	"fmt"
	"time"

	"github.com/TSM-061/Raggy/dashboard/internal/upload"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gocloud.dev/blob"
)

type UploadService struct {
	uploads   upload.Repo
	bucket    *blob.Bucket
	urlTTL    time.Duration
	profiles  *upload.ProfileRegistry
	validator *validator.Validate
}

func NewUploadService(
	uploads upload.Repo,
	bucket *blob.Bucket,
	urlTTL time.Duration,
	profiles *upload.ProfileRegistry,
	v *validator.Validate,
) *UploadService {
	return &UploadService{
		uploads:   uploads,
		bucket:    bucket,
		urlTTL:    urlTTL,
		profiles:  profiles,
		validator: v,
	}
}

const (
	defaultUploadsPage     = 1
	defaultUploadsPageSize = 20
	maxUploadsPageSize     = 100
)

type CreateUploadCommand struct {
	UploadedBy   uuid.UUID `validate:"required"`
	OriginalName string    `validate:"required"`
	ContentType  string    `validate:"required,contains=/"`
	ProfileHint  string    `validate:"required"`
	SizeBytes    int64     `validate:"gt=0"`
}

type CreateUploadResult struct {
	UploadURL          string
	UploadURLExpiresAt time.Time
	Upload             *upload.Upload
}

func (s *UploadService) CreateUpload(
	ctx context.Context,
	req *CreateUploadCommand,
) (*CreateUploadResult, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is required", serviceerr.InvalidInput)
	}

	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %v", serviceerr.InvalidInput, err)
	}

	if !s.profiles.IsRegistered(req.ProfileHint) {
		return nil, fmt.Errorf("%w: unknown profile hint '%s'", serviceerr.InvalidInput, req.ProfileHint)
	}

	upload := &upload.Upload{
		UploadedBy:   req.UploadedBy,
		OriginalName: req.OriginalName,
		ContentType:  req.ContentType,
		ProfileHint:  req.ProfileHint,
		SizeBytes:    req.SizeBytes,
		Status:       upload.StatusPending,
	}

	if err := s.uploads.Create(ctx, upload); err != nil {
		return nil, err
	}

	u, err := s.bucket.SignedURL(ctx, upload.ID.String(), &blob.SignedURLOptions{
		Method: "PUT",
		Expiry: s.urlTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign upload URL: %w", err)
	}

	expiresAt := time.Now().Add(s.urlTTL).UTC()

	return &CreateUploadResult{
		UploadURL:          u,
		UploadURLExpiresAt: expiresAt,
		Upload:             upload,
	}, nil
}

func (s *UploadService) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status upload.Status,
) error {
	if id == uuid.Nil {
		return fmt.Errorf("%w: upload id is required", serviceerr.InvalidInput)
	}

	if !status.IsValid() {
		return fmt.Errorf("%w: invalid upload status '%s'", serviceerr.InvalidInput, status)
	}

	return s.uploads.UpdateStatus(ctx, id, status)
}

type ListUploadsCommand struct {
	Page     int `validate:"omitempty,gte=1"`
	PageSize int `validate:"omitempty,gte=1,lte=100"`
}

type ListUploadsResult struct {
	Uploads    []upload.Upload
	PageNumber int
	PageSize   int
	Total      int64
}

func (s *UploadService) ListUploads(
	ctx context.Context,
	req *ListUploadsCommand,
) (*ListUploadsResult, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is required", serviceerr.InvalidInput)
	}

	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %v", serviceerr.InvalidInput, err)
	}

	page := req.Page
	if page == 0 {
		page = defaultUploadsPage
	}

	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = defaultUploadsPageSize
	}

	offset := (page - 1) * pageSize
	uploads, total, err := s.uploads.List(ctx, pageSize, offset)
	if err != nil {
		return nil, err
	}

	return &ListUploadsResult{
		Uploads:    uploads,
		PageNumber: page,
		PageSize:   pageSize,
		Total:      total,
	}, nil
}

type DeleteUploadCommand struct {
	UploadID uuid.UUID `validate:"required"`
}

func (s *UploadService) DeleteUpload(ctx context.Context, req *DeleteUploadCommand) error {
	if req == nil {
		return fmt.Errorf("%w: request is required", serviceerr.InvalidInput)
	}

	if err := s.validator.Struct(req); err != nil {
		return fmt.Errorf("%w: %v", serviceerr.InvalidInput, err)
	}

	return s.uploads.DeleteByID(ctx, req.UploadID)
}
