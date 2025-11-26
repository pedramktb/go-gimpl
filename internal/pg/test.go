package pg

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	postgresC "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestContainer(ctx context.Context) testcontainers.Container {
	c, err := postgresC.Run(ctx, "postgres:latest",
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

	return c
}

func TestDB(ctx context.Context, postgresContainer testcontainers.Container, dbName string) (db *sql.DB, cleanup func()) {
	ip, err := postgresContainer.Host(ctx)
	if err != nil {
		panic(err)
	}
	natPort, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		panic(err)
	}
	db, err = Pg(ctx, fmt.Sprintf("postgres://testpsqluser:testpsqluser@%s:%s/postgres", ip, natPort.Port()))
	if err != nil {
		panic(err)
	}
	_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %q", dbName))
	if err != nil {
		panic(err)
	}
	db.Close()
	db, err = Pg(ctx, fmt.Sprintf("postgres://testpsqluser:testpsqluser@%s:%s/%s", ip, natPort.Port(), dbName))
	if err != nil {
		panic(err)
	}
	return db, func() {
		db.Close()
		db, err := Pg(ctx, fmt.Sprintf("postgres://testpsqluser:testpsqluser@%s:%s/postgres", ip, natPort.Port()))
		if err != nil {
			panic(err)
		}
		_, err = db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE %q WITH (force)", dbName))
		if err != nil {
			panic(err)
		}
		db.Close()
	}
}
