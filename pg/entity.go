package pgimpl

import "github.com/pedramktb/go-gimpl"

// Entity is an extension of gimpl.Entity for Postgres operations.
// DO NOT USE POINTER RECEIVERS
type Entity interface {
	gimpl.Entity
	// Column returns the column name used for the given field
	Column(field string) string
	// Columns returns a slice of column names used for data retrieval, columns must match the order of pointers returned by GetColumnPtrs
	// Returned values should be static and not vary between calls
	Columns() []string
	// NewWithColumnPtrs returns a new entity and a slice of pointers to the fields used for writing to the entity during data retrieval
	// Returned values should not be dynamic and must correspond to the columns returned by Columns
	// The reason for creating a new entity is that without pointer receivers, the method cannot modify the original entity
	NewWithColumnPtrs() (Entity, []any)
}

// CreateEntity is an extension of gimpl.CreateEntity for Postgres creation operations.
// DO NOT USE POINTER RECEIVERS
type CreateEntity interface {
	gimpl.CreateEntity
	// CreateColumns returns a slice of column names used for creation operations, columns must match the order of pointers returned by CreateColumnPtrs
	// Returned values should be static and not vary between calls
	CreateColumns() []string
	// CreateColumnPtrs returns a slice of pointers to the fields used for writing to the entity during creation operations
	// Returned values should not be dynamic and must correspond to the columns returned by CreateColumns
	CreateColumnPtrs() []any
}

// UpdateEntity is an extension of gimpl.UpdateEntity for Postgres update operations.
// DO NOT USE POINTER RECEIVERS
type UpdateEntity interface {
	gimpl.UpdateEntity
	// IdentifyColumns returns a slice of column names used to identify unique records for update operations, columns must match the order of pointers returned by IdentifyColumnPtrs
	// Returned values should be static and not vary between calls
	IdentifyColumns() []string
	// IdentifyColumnPtrs returns a slice of pointers to the fields used to identify unique records for update operations
	// Returned values should not be dynamic and must correspond to the columns returned by IdentifyColumns
	IdentifyColumnPtrs() []any
	// UpdateColumns returns a slice of column names used for update operations, columns must match the order of pointers returned by UpdateColumnPtrs
	// Returned values should be static and not vary between calls
	UpdateColumns() []string
	// UpdateColumnPtrs returns a slice of pointers to the fields used for writing to the entity during update operations
	// Returned values should not be dynamic and must correspond to the columns returned by UpdateColumns
	UpdateColumnPtrs() []any
}

// CreateOrUpdateEntity is an extension of gimpl.CreateOrUpdateEntity for Postgres create or update operations.
// DO NOT USE POINTER RECEIVERS
type CreateOrUpdateEntity interface {
	CreateEntity
	UpdateEntity
}
