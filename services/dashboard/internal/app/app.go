package app

import (
	"github.com/TSM-061/Raggy/dashboard/internal/config"
	"github.com/TSM-061/Raggy/dashboard/internal/services"
	"github.com/TSM-061/Raggy/dashboard/internal/upload"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"gocloud.dev/blob"
)

type App struct {
	Config *config.Config

	Uploads upload.Repo
	Bucket  *blob.Bucket

	UploadService *services.UploadService
}

func New(
	cfg *config.Config,
	pool *pgxpool.Pool,
	bucket *blob.Bucket,
	profileKeys []string,
	v *validator.Validate,
) *App {
	uploadRepo := upload.NewPostgresRepo(pool)
	uploadService := services.NewUploadService(uploadRepo, bucket, cfg.UploadURLTTL, v)

	return &App{
		Config:        cfg,
		Uploads:       uploadRepo,
		Bucket:        bucket,
		UploadService: uploadService,
	}
}
