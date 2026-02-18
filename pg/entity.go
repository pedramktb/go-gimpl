package pgimpl

import "github.com/pedramktb/go-gimpl"

// Entity is an extension of gimpl.Entity for Postgres operations.
// DO NOT USE POINTER RECEIVERS
type Entity interface {
	gimpl.Entity
	// PgPath returns the postgres path steps for the given path.
	// If the Entity is a root entity, the zero index will be the column name.
	PgPath(path string) []string
	// PgColumns returns a slice of column names used for data retrieval.
	// Returned values should be static and not vary between calls
	PgColumns() []string
	// NewWithPgColumnPtrs returns a new entity and a slice of pointers to the fields used for writing to the entity during data retrieval
	// Returned values should not be dynamic and must correspond to the columns returned by PgColumns
	// The reason for creating a new entity is that without pointer receivers, the method cannot modify the original entity
	NewWithPgColumnPtrs() (any, []any)
}

// CreateEntity is an extension of gimpl.CreateEntity for Postgres creation operations.
// DO NOT USE POINTER RECEIVERS
type CreateEntity interface {
	gimpl.CreateEntity
	// CreatePgColumns returns a slice of column names used for creation operations.
	// Returned values should be static and not vary between calls
	CreatePgColumns() []string
	// CreatePgColumnVals returns a slice of the fields used for writing to the entity during creation operations
	// Returned values should not be dynamic and must correspond to the columns returned by CreatePgColumns
	CreatePgColumnVals() []any
}

// UpdateEntity is an extension of gimpl.UpdateEntity for Postgres update operations.
// DO NOT USE POINTER RECEIVERS
type UpdateEntity interface {
	gimpl.UpdateEntity
	// IdentifyPgColumns returns a slice of column names used to identify unique records for update operations.
	// Returned values should be static and not vary between calls
	IdentifyPgColumns() []string
	// IdentifyPgColumnVals returns a slice of the fields used to identify unique records for update operations
	// Returned values should not be dynamic and must correspond to the columns returned by IdentifyPgColumns
	IdentifyPgColumnVals() []any
	// UpdatePgColumns returns a slice of column names used for update operations.
	// Returned values should be static and not vary between calls
	UpdatePgColumns() []string
	// UpdatePgColumnVals returns a slice of the fields used for writing to the entity during update operations
	// Returned values should not be dynamic and must correspond to the columns returned by UpdatePgColumns
	UpdatePgColumnVals() []any
}

// SaveEntity is an extension of gimpl.SaveEntity for Postgres create or update operations.
// DO NOT USE POINTER RECEIVERS
type SaveEntity interface {
	CreateEntity
	UpdateEntity
}
