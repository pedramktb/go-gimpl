package gimpl

import (
	"errors"

	"github.com/pedramktb/go-tagerr"
)

var (
	ErrDBUnhandled = tagerr.ErrInternal.Wrap(&tagerr.Err{
		Err: errors.New("db unhandled error"),
		Tag: "unhandled_database_error",
	})
	ErrAPIUnhandled = tagerr.ErrInternal.Wrap(&tagerr.Err{
		Err: errors.New("api unhandled error"),
		Tag: "unhandled_api_error",
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

func ErrDBTx(err error) error {
	if err == nil {
		return nil
	}
	if err, ok := err.(*tagerr.Err); ok {
		return err
	}
	return ErrDBUnhandled.Wrap(err)
}
