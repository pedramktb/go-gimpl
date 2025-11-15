package gimpl

// Entity is a domain model that is allowed to use common implementations
type Entity interface {
	// GetPtrs returns a slice of pointers to the fields used for writing to the entity during data retrieval
	GetPtrs() []any
	// SortPtr returns a pointer to the field used for sorting and nil if the field does not exist or does not allow sorting
	// It will also be used to extract cursor values respective to sorts
	SortPtr(field string) any
	// FilterPtr returns a pointer to the field used for filtering and nil if the field does not exist or does not allow filtering
	FilterPtr(field string) any
}

// CreateEntity is the Entity augment used with creation operations
type CreateEntity interface {
	// CreatePtrs returns a slice of pointers to the fields used for writing to the data store during creation operations
	CreatePtrs() []any
}

// UpdateEntity is the Entity augment used with update operations
type UpdateEntity interface {
	// UpdatePtrs returns a slice of pointers to the fields used for writing to the data store during update operations
	UpdatePtrs() []any
}

// ReplaceEntity is the Entity augment used with replace operations
type ReplaceEntity interface {
	// ReplacePtrs returns a slice of pointers to the fields used for writing to the data store during replace operations
	ReplacePtrs() []any
}
