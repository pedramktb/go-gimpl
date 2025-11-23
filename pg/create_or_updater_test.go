package pgimpl_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	pgimpl "github.com/pedramktb/go-gimpl/pg"
	"github.com/stretchr/testify/require"
)

type TestCreateOrUpdateEntity struct {
	ID   uuid.UUID
	Name string
}

func (e TestCreateOrUpdateEntity) CreateColumns() []string {
	return []string{"pg_id", "pg_name"}
}

func (e TestCreateOrUpdateEntity) CreateColumnVals() []any {
	return []any{e.ID, e.Name}
}

func (e TestCreateOrUpdateEntity) IdentifyColumns() []string {
	return []string{"pg_id"}
}

func (e TestCreateOrUpdateEntity) IdentifyColumnVals() []any {
	return []any{e.ID}
}

func (e TestCreateOrUpdateEntity) UpdateColumns() []string {
	return []string{"pg_name"}
}

func (e TestCreateOrUpdateEntity) UpdateColumnVals() []any {
	return []any{e.Name}
}

func Test_CreateOrUpdate(t *testing.T) {
	ctx := context.Background()
	db, cleanup := DB(ctx, "pgimpl_test_create_or_update")
	defer cleanup()

	createOrUpdater := pgimpl.CreateOrUpdater[TestCreateOrUpdateEntity](db, "test_entity")

	tests := []struct {
		name    string
		setup   func() []uuid.UUID
		items   func(ids []uuid.UUID) []TestCreateOrUpdateEntity
		wantErr bool
	}{
		{
			name: "create new items",
			setup: func() []uuid.UUID {
				return []uuid.UUID{}
			},
			items: func(ids []uuid.UUID) []TestCreateOrUpdateEntity {
				return []TestCreateOrUpdateEntity{
					{ID: uuid.New(), Name: "New Item 1"},
					{ID: uuid.New(), Name: "New Item 2"},
				}
			},
			wantErr: false,
		},
		{
			name: "replace existing items",
			setup: func() []uuid.UUID {
				id1, id2 := uuid.New(), uuid.New()
				_, err := db.ExecContext(ctx, "INSERT INTO test_entity (pg_id, pg_name) VALUES ($1, $2), ($3, $4)",
					id1, "Old Name 1", id2, "Old Name 2")
				require.NoError(t, err)
				return []uuid.UUID{id1, id2}
			},
			items: func(ids []uuid.UUID) []TestCreateOrUpdateEntity {
				return []TestCreateOrUpdateEntity{
					{ID: ids[0], Name: "Updated Name 1"},
					{ID: ids[1], Name: "Updated Name 2"},
				}
			},
			wantErr: false,
		},
		{
			name: "mixed create and replace",
			setup: func() []uuid.UUID {
				id := uuid.New()
				_, err := db.ExecContext(ctx, "INSERT INTO test_entity (pg_id, pg_name) VALUES ($1, $2)", id, "Existing")
				require.NoError(t, err)
				return []uuid.UUID{id}
			},
			items: func(ids []uuid.UUID) []TestCreateOrUpdateEntity {
				return []TestCreateOrUpdateEntity{
					{ID: ids[0], Name: "Updated Existing"},
					{ID: uuid.New(), Name: "New Item"},
				}
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ids := tc.setup()
			items := tc.items(ids)

			err := createOrUpdater(ctx, items)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Verify all items exist with correct names
				for _, item := range items {
					var name string
					err := db.QueryRowContext(ctx, "SELECT pg_name FROM test_entity WHERE pg_id = $1", item.ID).Scan(&name)
					require.NoError(t, err)
					require.Equal(t, item.Name, name)
				}
			}
		})
	}
}
