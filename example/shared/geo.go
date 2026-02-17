package shared

// pgimpl:Entity, CreateEntity, UpdateEntity
type Geo struct {
	Lat float64 `gimpl:"field:lat" pgimpl:"column:pg_lat;create;update"`
	Lon float64 `gimpl:"field:lon" pgimpl:"column:pg_lon;create;update"`
}
