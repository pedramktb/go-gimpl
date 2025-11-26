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

func Test_FindOne(t *testing.T) {
	ctx := context.Background()
	db, cleanup := DB(ctx, "pgimpl_test_get_one")
	defer cleanup()

	// Insert test data
	id1, id2, id3 := uuid.New(), uuid.New(), uuid.New()
	_, err := db.ExecContext(ctx, "INSERT INTO test_entity (pg_id, pg_name) VALUES ($1, $2), ($3, $4), ($5, $6)",
		id1, "Test 1", id2, "Test 2", id3, "Test 3")
	require.NoError(t, err)

	finder := pgimpl.Finder[TestEntity](db, "test_entity")

	tests := []struct {
		name    string
		filter  gimpl.Expr
		want    TestEntity
		wantErr error
	}{
		{
			name: "find existing entity by id",
			filter: gimpl.Expr{Sample: TestEntity{}, Expr: gimpl.CondExpr{
				Field: "id",
				Op:    gimpl.CondOpEQ,
				Val:   id1,
			}},
			want:    TestEntity{ID: id1, Name: "Test 1"},
			wantErr: nil,
		},
		{
			name: "find existing entity by name",
			filter: gimpl.Expr{Sample: TestEntity{}, Expr: gimpl.CondExpr{
				Field: "name",
				Op:    gimpl.CondOpEQ,
				Val:   "Test 2",
			}},
			want:    TestEntity{ID: id2, Name: "Test 2"},
			wantErr: nil,
		},
		{
			name: "find non-existent entity",
			filter: gimpl.Expr{Sample: TestEntity{}, Expr: gimpl.CondExpr{
				Field: "id",
				Op:    gimpl.CondOpEQ,
				Val:   uuid.New(),
			}},
			want:    TestEntity{},
			wantErr: tagerr.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := finder.FindOne(ctx, tc.filter)
			if tc.wantErr != nil {
				require.Error(t, err)
				require.True(t, tagerr.Is(err, tc.wantErr))
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, result)
			}
		})
	}
}

func Test_Find(t *testing.T) {
	ctx := context.Background()
	db, cleanup := DB(ctx, "pgimpl_test_get")
	defer cleanup()

	// Insert test data
	id1, id2, id3 := uuid.New(), uuid.New(), uuid.New()
	_, err := db.ExecContext(ctx, "INSERT INTO test_entity (pg_id, pg_name) VALUES ($1, $2), ($3, $4), ($5, $6)",
		id1, "Test 1", id2, "Test 2", id3, "Test 3")
	require.NoError(t, err)

	finder := pgimpl.Finder[TestEntity](db, "test_entity")

	tests := []struct {
		name           string
		filter         gimpl.Expr
		paginateOpt    []gimpl.PaginateOpt
		wantTotalCount uint64
		want           []TestEntity
	}{
		{
			name:           "get all items",
			filter:         gimpl.Expr{},
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
			filter:         gimpl.Expr{},
			paginateOpt:    []gimpl.PaginateOpt{gimpl.WithLimit(2)},
			wantTotalCount: 3,
			want: []TestEntity{
				{ID: id1, Name: "Test 1"},
				{ID: id2, Name: "Test 2"},
			},
		},
		{
			name: "get with filter",
			filter: gimpl.Expr{Sample: TestEntity{}, Expr: gimpl.CondExpr{
				Field: "id",
				Op:    gimpl.CondOpEQ,
				Val:   id1,
			}},
			paginateOpt:    []gimpl.PaginateOpt{gimpl.WithLimit(10)},
			wantTotalCount: 1,
			want: []TestEntity{
				{ID: id1, Name: "Test 1"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := finder.Find(ctx, tc.filter, tc.paginateOpt)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.want, result.Items)
			require.Equal(t, tc.wantTotalCount, result.Meta.Total)
		})
	}
}
