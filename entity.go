package gimpl

// Entity is a domain model that is allowed to use common implementations.
// DO NOT USE POINTER RECEIVERS
type Entity interface {
	// SortPtr returns a pointer to the field used for sorting and nil if the field does not exist or does not allow sorting
	// It will also be used to extract cursor values respective to sorts
	SortPtr(path string) any
	// FilterPtr returns a pointer to the field used for filtering and nil if the field does not exist or does not allow filtering
	FilterPtr(path string) any
}

// CreateEntity is the Entity augment used with creation operations.
// DO NOT USE POINTER RECEIVERS
type CreateEntity interface {
}

// UpdateEntity is the Entity augment used with update operations.
// DO NOT USE POINTER RECEIVERS
type UpdateEntity interface {
}

// SaveEntity is the Entity augment used with create or update operations.
// DO NOT USE POINTER RECEIVERS
type SaveEntity interface {
	CreateEntity
	UpdateEntity
}
