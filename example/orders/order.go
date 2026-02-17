package orders

import "github.com/pedramktb/go-gimpl/example/shared"

// pgimpl:Entity, CreateEntity
type Order struct {
	ID          int        `gimpl:"field:id" pgimpl:"identify"`
	Destination shared.Geo `gimpl:"flat;prefix:dest_"`
}
