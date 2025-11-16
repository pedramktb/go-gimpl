package gimpl

import (
	"errors"

	"github.com/pedramktb/go-tagerr"
)

var (
	ErrDatastoreUnhandled = tagerr.ErrInternal.Wrap(&tagerr.Err{
		Err: errors.New("datastore unhandled error"),
		Tag: "unhandled_datastore_error",
	})
	ErrInUse = tagerr.ErrFailedPreCond.Wrap(&tagerr.Err{
		Err: errors.New("entity in use and is referenced by other entities"),
		Tag: "entity_in_use",
	})
	ErrInvalidField = tagerr.ErrInvalidReq.Wrap(&tagerr.Err{
		Err: errors.New("invalid field"),
		Tag: "invalid_field",
	})
	ErrInvalidCursor = tagerr.ErrInvalidReq.Wrap(&tagerr.Err{
		Err: errors.New("invalid cursor"),
		Tag: "invalid_cursor",
	})
	ErrMismatchInSortAndCursor = ErrInvalidCursor.Wrap(&tagerr.Err{
		Err: errors.New("mismatch in sort and cursor"),
		Tag: "mismatch_in_sort_and_cursor",
	})
	ErrInvalidExpr = tagerr.ErrInvalidReq.Wrap(&tagerr.Err{
		Err: errors.New("invalid expression"),
		Tag: "invalid_expr",
	})
)
