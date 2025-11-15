package gimpl

// Entity is a domain model that is allowed to use common implementations
type Entity interface {
	// Sortable returns whether a given field is sortable
	Sortable(field string) bool
	// Filterable returns whether a given field is filterable
	Filterable(field string) bool
	// Pointer returns a pointer to a given field and its nil if the field does not exist
	Pointer(field string) (value any)
}

// UpdateEntity is the Entity augment used with update operations
type UpdateEntity interface {
	// Pointer returns a pointer to a given field and its nil if the field does not exist
	Pointer(field string) any
}
