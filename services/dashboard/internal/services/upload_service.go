package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/TSM-061/Raggy/dashboard/internal/upload"
	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/TSM-061/Raggy/shared/storage"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Presigner interface {
	SignedPutURL(ctx context.Context, key string) (*storage.SignedUrl, error)
}

type Upload struct {
	uploads   upload.Repo
	presigner Presigner
	urlTTL    time.Duration
	validator *validator.Validate
}

func NewUploadService(
	uploads upload.Repo,
	bucket Presigner,
	urlTTL time.Duration,
	v *validator.Validate,
) *Upload {
	return &Upload{
		uploads:   uploads,
		presigner: bucket,
		urlTTL:    urlTTL,
		validator: v,
	}
}

const (
	defaultUploadsPage     = 1
	defaultUploadsPageSize = 20
)

type CreateUploadCommand struct {
	UploadedBy   uuid.UUID `validate:"required"`
	OriginalName string    `validate:"required"`
	ContentType  string    `validate:"required,contains=/"`
	ProfileHint  string    `validate:"required"`
	SizeBytes    int64     `validate:"gt=0"`
}

type CreateUploadResult struct {
	UploadURL     string
	SignedHeaders map[string]string
	ExpiresAt     time.Time
	Upload        *upload.Upload
}

func (s *Upload) CreateUpload(
	ctx context.Context,
	req *CreateUploadCommand,
) (*CreateUploadResult, error) {
	log := logger.FromContext(ctx)

	if req == nil {
		return nil, fmt.Errorf("%w: req is nil", serviceerr.InvalidInput)
	}

	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %v", serviceerr.InvalidInput, err)
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

	u, err := s.presigner.SignedPutURL(ctx, upload.ID.String())
	if err != nil {
		return nil, fmt.Errorf("presign upload url: %w", err)
	}

	expiresAt := time.Now().Add(s.urlTTL).UTC()

	log.InfoContext(
		ctx, "upload successfully persisted",
		slog.String("upload_id", upload.ID.String()),
		slog.String("uploaded_by", upload.UploadedBy.String()),
	)

	return &CreateUploadResult{
		UploadURL:     u.Url,
		SignedHeaders: u.SignedHeaders,
		ExpiresAt:     expiresAt,
		Upload:        upload,
	}, nil
}

func (s *Upload) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status upload.Status,
) error {
	log := logger.FromContext(ctx)

	if id == uuid.Nil {
		return fmt.Errorf("%w: id is uuid.Nil", serviceerr.InvalidInput)
	}

	if !status.IsValid() {
		return fmt.Errorf("%w: upload status '%s'", serviceerr.InvalidInput, status)
	}

	if err := s.uploads.UpdateStatus(ctx, id, status); err != nil {
		return err
	}

	log.InfoContext(ctx, "upload status updated",
		slog.String("upload_id", id.String()),
		slog.String("status", string(status)),
	)

	return nil
}

type ListUploadsQuery struct {
	Page     int `validate:"omitempty,gte=1"`
	PageSize int `validate:"omitempty,gte=1,lte=100"`
}

type GetUploadByIDQuery struct {
	UploadID uuid.UUID `validate:"required"`
}

type ListUploadsResult struct {
	Uploads    []upload.Upload
	PageNumber int
	PageSize   int
	Total      int64
}

func (s *Upload) ListUploads(
	ctx context.Context,
	query *ListUploadsQuery,
) (*ListUploadsResult, error) {
	log := logger.FromContext(ctx)

	if query == nil {
		return nil, fmt.Errorf("%w: request is nil", serviceerr.InvalidInput)
	}

	if err := s.validator.Struct(query); err != nil {
		return nil, fmt.Errorf("%w: %v", serviceerr.InvalidInput, err)
	}

	page := query.Page
	if page == 0 {
		page = defaultUploadsPage
	}

	pageSize := query.PageSize
	if pageSize == 0 {
		pageSize = defaultUploadsPageSize
	}

	offset := (page - 1) * pageSize

	uploads, total, err := s.uploads.List(ctx, pageSize, offset)
	if err != nil {
		return nil, err
	}

	log.InfoContext(
		ctx,
		"uploads listed",
		slog.Int("page_size", pageSize),
		slog.Int("count", len(uploads)),
		slog.Int64("total_count", total),
	)

	return &ListUploadsResult{
		Uploads:    uploads,
		PageNumber: page,
		PageSize:   pageSize,
		Total:      total,
	}, nil
}

func (s *Upload) GetUploadByID(
	ctx context.Context,
	query *GetUploadByIDQuery,
) (*upload.Upload, error) {
	if query == nil {
		return nil, fmt.Errorf("%w: req is nil", serviceerr.InvalidInput)
	}

	if err := s.validator.Struct(query); err != nil {
		return nil, fmt.Errorf("%w: %v", serviceerr.InvalidInput, err)
	}

	return s.uploads.GetByID(ctx, query.UploadID)
}

type DeleteUploadCommand struct {
	UploadID uuid.UUID `validate:"required"`
}

func (s *Upload) DeleteUpload(ctx context.Context, req *DeleteUploadCommand) error {
	if req == nil {
		return fmt.Errorf("%w: request is required", serviceerr.InvalidInput)
	}

	if err := s.validator.Struct(req); err != nil {
		return fmt.Errorf("%w: %v", serviceerr.InvalidInput, err)
	}

	return s.uploads.DeleteByID(ctx, req.UploadID)
}
