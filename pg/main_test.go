package pgimpl_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/pedramktb/go-gimpl/internal/pg"
	"github.com/testcontainers/testcontainers-go"
)

var container testcontainers.Container

// TestMain creates a Postgres container and tears it down when tests finish.
func TestMain(m *testing.M) {
	ctx := context.Background()
	container = pg.TestContainer(ctx)
	defer func() { _ = container.Terminate(ctx) }()
	os.Exit(m.Run())
}

func DB(ctx context.Context, dbName string) (*sql.DB, func()) {
	db, cleanup := pg.TestDB(ctx, container, dbName)
	_, err := db.ExecContext(ctx, "CREATE TABLE test_entity (pg_id UUID PRIMARY KEY, pg_name TEXT)")
	if err != nil {
		panic(err)
	}
	return db, cleanup
}

type TestEntity struct {
	ID   uuid.UUID
	Name string
}

func (e TestEntity) FilterPtr(path string) any {
	switch path {
	case "id":
		return &e.ID
	case "name":
		return &e.Name
	default:
		return nil
	}
}
func (e TestEntity) SortPtr(path string) any {
	switch path {
	case "id":
		return &e.ID
	case "name":
		return &e.Name
	default:
		return nil
	}
}
func (e TestEntity) PgPath(path string) []string {
	switch path {
	case "id":
		return []string{"pg_id"}
	case "name":
		return []string{"pg_name"}
	default:
		return nil
	}
}
func (e TestEntity) PgColumns() []string {
	return []string{"pg_id", "pg_name"}
}
func (e TestEntity) NewWithPgColumnPtrs() (any, []any) {
	return &e, []any{&e.ID, &e.Name}
}
