package testutil

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/TSM-061/Raggy/simple-auth-service/internal/migrations"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func StartDatabase(ctx context.Context) (*pgxpool.Pool, error) {
	dbName := "users"
	dbUser := "user"
	dbPassword := "password"

	postgresContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		postgres.BasicWaitStrategies(),
		testcontainers.WithReuseByName("raggy-auth-test-database"),
	)
	if err != nil {
		return nil, err
	}

	connectionString, err := postgresContainer.ConnectionString(ctx)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}

	if err := goose.Up(db, "."); err != nil {
		panic(err)
	}

	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		panic("failed to open database connection")
	}

	return pool, nil
}
