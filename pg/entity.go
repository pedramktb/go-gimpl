package pgimpl

import "github.com/pedramktb/go-gimpl"

type Entity interface {
	gimpl.Entity
	// GetColumns returns a slice of column names used for data retrieval, columns must match the order of pointers returned by GetColumnPtrs
	GetColumns() []string
	// ColumnPtrs returns a slice of pointers to the fields used for writing to the entity during data retrieval
	// Implementation must have a pointer receiver, otherwise scanning will return an empty value
	GetColumnPtrs() []any
}

type CreateEntity interface {
	gimpl.CreateEntity
	// CreateColumns returns a slice of column names used for creation operations, columns must match the order of pointers returned by CreateColumnPtrs
	CreateColumns() []string
	// CreateColumnPtrs returns a slice of pointers to the fields used for writing to the entity during creation operations
	CreateColumnPtrs() []any
}

type ReplaceEntity interface {
	gimpl.ReplaceEntity
	// IdentifyColumns returns a slice of column names used to identify unique records for replace operations
	IdentifyColumns() []string
	// IdentifyColumnPtrs returns a slice of pointers to the fields used to identify unique records for replace operations
	IdentifyColumnPtrs() []any
	// ReplaceColumns returns a slice of column names used for replace operations, columns must match the order of pointers returned by ReplaceColumnPtrs
	ReplaceColumns() []string
	// ReplaceColumnPtrs returns a slice of pointers to the fields used for writing to the entity during replace operations
	ReplaceColumnPtrs() []any
}

type CreateOrReplaceEntity interface {
	CreateEntity
	ReplaceEntity
}

type UpdateEntity interface {
	gimpl.UpdateEntity
	// IdentifyColumns returns a slice of column names used to identify unique records for update operations, columns must match the order of pointers returned by IdentifyColumnPtrs
	IdentifyColumns() []string
	// IdentifyColumnPtrs returns a slice of pointers to the fields used to identify unique records for update operations
	IdentifyColumnPtrs() []any
	// UpdateColumns returns a slice of column names used for update operations, columns must match the order of pointers returned by UpdateColumnPtrs
	UpdateColumns() []string
	// UpdateColumnPtrs returns a slice of pointers to the fields used for writing to the entity during update operations
	UpdateColumnPtrs() []any
}
