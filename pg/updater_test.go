package pgimpl_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	pgimpl "github.com/pedramktb/go-gimpl/pg"
	"github.com/pedramktb/go-tagerr"
	"github.com/stretchr/testify/require"
)

type TestUpdateEntity struct {
	ID   uuid.UUID
	Name string
}

func (e TestUpdateEntity) IdentifyPgColumns() []string {
	return []string{"pg_id"}
}
func (e TestUpdateEntity) IdentifyPgColumnVals() []any {
	return []any{e.ID}
}
func (e TestUpdateEntity) UpdatePgColumns() []string {
	return []string{"pg_name"}
}
func (e TestUpdateEntity) UpdatePgColumnVals() []any {
	return []any{e.Name}
}

func Test_Update(t *testing.T) {
	ctx := context.Background()
	db, cleanup := DB(ctx, "pgimpl_test_update")
	defer cleanup()

	updater := pgimpl.Updater[TestUpdateEntity](db, "test_entity")

	tests := []struct {
		name    string
		setup   func() []uuid.UUID
		items   func(ids []uuid.UUID) []TestUpdateEntity
		wantErr error
	}{
		{
			name: "update existing items",
			setup: func() []uuid.UUID {
				id1, id2 := uuid.New(), uuid.New()
				_, err := db.ExecContext(ctx, "INSERT INTO test_entity (pg_id, pg_name) VALUES ($1, $2), ($3, $4)",
					id1, "Old Name 1", id2, "Old Name 2")
				require.NoError(t, err)
				return []uuid.UUID{id1, id2}
			},
			items: func(ids []uuid.UUID) []TestUpdateEntity {
				return []TestUpdateEntity{
					{ID: ids[0], Name: "Updated Name 1"},
					{ID: ids[1], Name: "Updated Name 2"},
				}
			},
			wantErr: nil,
		},
		{
			name: "update single item",
			setup: func() []uuid.UUID {
				id := uuid.New()
				_, err := db.ExecContext(ctx, "INSERT INTO test_entity (pg_id, pg_name) VALUES ($1, $2)", id, "Original")
				require.NoError(t, err)
				return []uuid.UUID{id}
			},
			items: func(ids []uuid.UUID) []TestUpdateEntity {
				return []TestUpdateEntity{
					{ID: ids[0], Name: "Modified"},
				}
			},
			wantErr: nil,
		},
		{
			name: "update non-existent item",
			setup: func() []uuid.UUID {
				return []uuid.UUID{}
			},
			items: func(ids []uuid.UUID) []TestUpdateEntity {
				return []TestUpdateEntity{
					{ID: uuid.New(), Name: "Non-existent"},
				}
			},
			wantErr: tagerr.ErrNotFound,
		},
		{
			name: "update with partial non-existent items",
			setup: func() []uuid.UUID {
				id := uuid.New()
				_, err := db.ExecContext(ctx, "INSERT INTO test_entity (pg_id, pg_name) VALUES ($1, $2)", id, "Exists")
				require.NoError(t, err)
				return []uuid.UUID{id}
			},
			items: func(ids []uuid.UUID) []TestUpdateEntity {
				return []TestUpdateEntity{
					{ID: ids[0], Name: "Exists Updated"},
					{ID: uuid.New(), Name: "Does Not Exist"},
				}
			},
			wantErr: tagerr.ErrNotFound,
		},
		{
			name: "update with empty items",
			setup: func() []uuid.UUID {
				return []uuid.UUID{}
			},
			items: func(ids []uuid.UUID) []TestUpdateEntity {
				return []TestUpdateEntity{}
			},
			wantErr: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ids := tc.setup()
			items := tc.items(ids)

			err := updater.Update(ctx, items)
			if tc.wantErr != nil {
				require.True(t, tagerr.Is(err, tc.wantErr))
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
