package pgimpl_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pedramktb/go-gimpl"
	pgimpl "github.com/pedramktb/go-gimpl/pg"
	"github.com/stretchr/testify/require"
)

func Test_Get(t *testing.T) {
	ctx := context.Background()
	db, cleanup := DB(ctx, "pgimpl_test_get")
	defer cleanup()

	// Insert test data
	id1, id2, id3 := uuid.New(), uuid.New(), uuid.New()
	_, err := db.ExecContext(ctx, "INSERT INTO test_entity (pg_id, pg_name) VALUES ($1, $2), ($3, $4), ($5, $6)",
		id1, "Test 1", id2, "Test 2", id3, "Test 3")
	require.NoError(t, err)

	getter := pgimpl.Getter[TestEntity](db, "test_entity")

	tests := []struct {
		name           string
		locateOpts     []gimpl.LocateOpt
		paginateOpt    []gimpl.PaginateOpt
		wantTotalCount uint64
		want           []TestEntity
	}{
		{
			name:           "get all items",
			locateOpts:     []gimpl.LocateOpt{},
			paginateOpt:    []gimpl.PaginateOpt{gimpl.WithLimit(10)},
			wantTotalCount: 3,
			want: []TestEntity{
				{ID: id1, Name: "Test 1"},
				{ID: id2, Name: "Test 2"},
				{ID: id3, Name: "Test 3"},
			},
		},
		{
			name:           "get with limit",
			locateOpts:     []gimpl.LocateOpt{},
			paginateOpt:    []gimpl.PaginateOpt{gimpl.WithLimit(2)},
			wantTotalCount: 3,
			want: []TestEntity{
				{ID: id1, Name: "Test 1"},
				{ID: id2, Name: "Test 2"},
			},
		},
		{
			name: "get with filter",
			locateOpts: []gimpl.LocateOpt{gimpl.WithFilter(gimpl.Expr{Sample: TestEntity{}, Expr: gimpl.CondExpr{
				Field: "id",
				Op:    gimpl.CondOpEQ,
				Val:   id1,
			}})},
			paginateOpt:    []gimpl.PaginateOpt{gimpl.WithLimit(10)},
			wantTotalCount: 1,
			want: []TestEntity{
				{ID: id1, Name: "Test 1"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := getter(ctx, tc.locateOpts, tc.paginateOpt)
			require.NoError(t, err)
			require.Len(t, result.Items, len(tc.want))
			require.Equal(t, tc.wantTotalCount, result.Meta.Total)
		})
	}
}
