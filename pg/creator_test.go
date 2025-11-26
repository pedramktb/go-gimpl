package pgimpl_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	pgimpl "github.com/pedramktb/go-gimpl/pg"
	"github.com/pedramktb/go-tagerr"
	"github.com/stretchr/testify/require"
)

type TestCreateEntity struct {
	ID   uuid.UUID
	Name string
}

func (e TestCreateEntity) CreateColumns() []string {
	return []string{"pg_id", "pg_name"}
}
func (e TestCreateEntity) CreateColumnVals() []any {
	return []any{e.ID, e.Name}
}

func Test_Create(t *testing.T) {
	ctx := context.Background()
	db, cleanup := DB(ctx, "pgimpl_test_create")
	defer cleanup()
	creator := pgimpl.Creator[TestCreateEntity](db, "test_entity")
	tests := []struct {
		name    string
		items   []TestCreateEntity
		want    pgimpl.Entity
		wantErr error
	}{
		{
			name: "valid create",
			items: []TestCreateEntity{
				{ID: uuid.New(), Name: "Test 1"},
				{ID: uuid.New(), Name: "Test 2"},
			},
			wantErr: nil,
		},
		{
			name:    "empty items",
			items:   []TestCreateEntity{},
			wantErr: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := creator.Create(ctx, tc.items)
			if tc.wantErr != nil {
				require.True(t, tagerr.Is(err, tc.wantErr))
			} else {
				require.NoError(t, err)

				// Verify all items exist with correct names
				for _, item := range tc.items {
					var name string
					err := db.QueryRowContext(ctx, "SELECT pg_name FROM test_entity WHERE pg_id = $1", item.ID).Scan(&name)
					require.NoError(t, err)
					require.Equal(t, item.Name, name)
				}
			}
		})
	}
}
