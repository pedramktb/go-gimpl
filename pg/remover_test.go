package pgimpl_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pedramktb/go-gimpl"
	pgimpl "github.com/pedramktb/go-gimpl/pg"
	"github.com/pedramktb/go-tagerr"
	"github.com/stretchr/testify/require"
)

func Test_RemoveOne(t *testing.T) {
	ctx := context.Background()
	db, cleanup := DB(ctx, "pgimpl_test_delete_one")
	defer cleanup()

	remover := pgimpl.Remover[TestEntity](db, "test_entity")

	tests := []struct {
		name    string
		setup   func() uuid.UUID
		filter  func(uuid.UUID) gimpl.Expr
		wantErr error
	}{
		{
			name: "valid delete one",
			setup: func() uuid.UUID {
				id := uuid.New()
				_, err := db.ExecContext(ctx, "INSERT INTO test_entity (pg_id, pg_name) VALUES ($1, $2)", id, "Delete Me One")
				require.NoError(t, err)
				return id
			},
			filter: func(id uuid.UUID) gimpl.Expr {
				return gimpl.Expr{Sample: TestEntity{}, Expr: gimpl.CondExpr{
					Path: "id",
					Op:   gimpl.CondOpEQ,
					Val:  id,
				}}
			},
			wantErr: nil,
		},
		{
			name:  "delete non-existent entity",
			setup: func() uuid.UUID { return uuid.Nil },
			filter: func(id uuid.UUID) gimpl.Expr {
				return gimpl.Expr{Sample: TestEntity{}, Expr: gimpl.CondExpr{
					Path: "id",
					Op:   gimpl.CondOpEQ,
					Val:  uuid.New(),
				}}
			},
			wantErr: tagerr.ErrNotFound,
		},
		{
			name: "delete without filter",
			setup: func() uuid.UUID {
				id := uuid.New()
				_, err := db.ExecContext(ctx, "INSERT INTO test_entity (pg_id, pg_name) VALUES ($1, $2)", id, "Test Entity")
				require.NoError(t, err)
				return id
			},
			filter:  func(id uuid.UUID) gimpl.Expr { return gimpl.Expr{} },
			wantErr: gimpl.ErrDatastoreUnhandled,
		},
		{
			name: "delete one when multiple match - should error",
			setup: func() uuid.UUID {
				id1, id2 := uuid.New(), uuid.New()
				_, err := db.ExecContext(ctx, "INSERT INTO test_entity (pg_id, pg_name) VALUES ($1, $2), ($3, $4)",
					id1, "Same Name", id2, "Same Name")
				require.NoError(t, err)
				return uuid.Nil
			},
			filter: func(id uuid.UUID) gimpl.Expr {
				return gimpl.Expr{Sample: TestEntity{}, Expr: gimpl.CondExpr{
					Path: "name",
					Op:   gimpl.CondOpEQ,
					Val:  "Same Name",
				}}
			},
			wantErr: gimpl.ErrDatastoreUnhandled,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id := tc.setup()
			filter := tc.filter(id)

			err := remover.RemoveOne(ctx, filter)
			if tc.wantErr != nil {
				require.Error(t, err)
				require.True(t, tagerr.Is(err, tc.wantErr))
			} else {
				require.NoError(t, err)

				// Verify deletion
				var count int
				err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_entity WHERE pg_id = $1", id).Scan(&count)
				require.NoError(t, err)
				require.Equal(t, 0, count)
			}
		})
	}
}

func Test_Remove(t *testing.T) {
	ctx := context.Background()
	db, cleanup := DB(ctx, "pgimpl_test_delete")
	defer cleanup()

	id1, id2 := uuid.New(), uuid.New()

	remover := pgimpl.Remover[TestEntity](db, "test_entity")

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func()
		filter  gimpl.Expr
		wantErr error
	}{
		{
			name: "valid delete",
			id:   id1,
			setup: func() {
				_, err := db.ExecContext(ctx, "INSERT INTO test_entity (pg_id, pg_name) VALUES ($1, $2)", id1, "Delete Me")
				require.NoError(t, err)
			},
			filter:  gimpl.Expr{Sample: TestEntity{}, Expr: gimpl.CondExpr{Path: "id", Op: gimpl.CondOpEQ, Val: id1}},
			wantErr: nil,
		},
		{
			name:    "delete non-existent",
			id:      id2,
			setup:   func() {},
			filter:  gimpl.Expr{Sample: TestEntity{}, Expr: gimpl.CondExpr{Path: "id", Op: gimpl.CondOpEQ, Val: id2}},
			wantErr: nil,
		},
		{
			name:    "delete without filter",
			setup:   func() {},
			wantErr: gimpl.ErrDatastoreUnhandled,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			err := remover.Remove(ctx, tc.filter)
			if tc.wantErr != nil {
				require.True(t, tagerr.Is(err, tc.wantErr))
			} else {
				require.NoError(t, err)

				// Verify deletion
				var count int
				err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_entity WHERE pg_id = $1", tc.id).Scan(&count)
				require.NoError(t, err)
				require.Equal(t, 0, count)
			}
		})
	}
}
