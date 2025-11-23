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

func Test_Delete(t *testing.T) {
	ctx := context.Background()
	db, cleanup := DB(ctx, "pgimpl_test_delete")
	defer cleanup()

	deleter := pgimpl.Deleter[TestEntity](db, "test_entity")

	tests := []struct {
		name       string
		setup      func() uuid.UUID
		locateOpts []gimpl.LocateOpt
		wantErr    error
	}{
		{
			name: "valid delete",
			setup: func() uuid.UUID {
				id := uuid.New()
				_, err := db.ExecContext(ctx, "INSERT INTO test_entity (pg_id, pg_name) VALUES ($1, $2)", id, "Delete Me")
				require.NoError(t, err)
				return id
			},
			locateOpts: nil,
			wantErr:    nil,
		},
		{
			name: "delete non-existent",
			setup: func() uuid.UUID {
				return uuid.New()
			},
			locateOpts: nil,
			wantErr:    tagerr.ErrNotFound,
		},
		{
			name: "delete without filter",
			setup: func() uuid.UUID {
				return uuid.New()
			},
			locateOpts: []gimpl.LocateOpt{},
			wantErr:    gimpl.ErrDatastoreUnhandled,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id := tc.setup()
			opts := tc.locateOpts
			if opts == nil {
				opts = []gimpl.LocateOpt{gimpl.WithFilter(gimpl.Expr{Sample: TestEntity{}, Expr: gimpl.CondExpr{Field: "id", Op: gimpl.CondOpEQ, Val: id}})}
			}

			err := deleter(ctx, opts)
			if tc.wantErr != nil {
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
