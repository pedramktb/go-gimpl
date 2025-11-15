package pgtestcontainers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pedramktb/go-gimpl/internal/pg"
	"github.com/testcontainers/testcontainers-go"
	postgresC "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func NewTestContainer(ctx context.Context) (container testcontainers.Container, ip, port string) {
	postgresContainer, err := postgresC.Run(ctx, "postgres:14.17",
		postgresC.WithUsername("testpsqluser"),
		postgresC.WithPassword("testpsqluser"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(time.Minute)),
	)
	if err != nil {
		panic(err)
	}

	ip, err = postgresContainer.Host(ctx)
	if err != nil {
		panic(err)
	}
	natPort, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		panic(err)
	}

	return postgresContainer, ip, natPort.Port()
}

func CreateTestDB(ctx context.Context, ip, port, dbName string) *sql.DB {
	db, err := pg.Pg(ctx, fmt.Sprintf("postgres://testpsqluser:testpsqluser@%s:%s/postgres", ip, port))
	if err != nil {
		panic(err)
	}

	_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %q", dbName))
	if err != nil {
		panic(err)
	}
	db.Close()

	db, err = pg.Pg(ctx, fmt.Sprintf("postgres://testpsqluser:testpsqluser@%s:%s/%s", ip, port, dbName))
	if err != nil {
		panic(err)
	}

	return db
}

func DropTestDB(ctx context.Context, db *sql.DB, ip, port, dbName string) {
	db.Close()
	db, _ = pg.Pg(ctx, fmt.Sprintf("postgres://testpsqluser:testpsqluser@%s:%s/postgres", ip, port))
	_, _ = db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE %q WITH (force)", dbName))
	db.Close()
}
