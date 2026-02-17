package users

import "github.com/pedramktb/go-gimpl/example/shared"

// pgimpl:Entity, CreateEntity, UpdateEntity
type User struct {
	ID       int        `gimpl:"field:id" pgimpl:"identify"`
	Location shared.Geo `gimpl:"flat"`
}
