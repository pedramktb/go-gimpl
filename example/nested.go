package example

// pgimpl:Entity, SaveEntity
type Address struct {
	Street  string  `gimpl:"field:street;sort" pgimpl:"create;update"`
	City    string  `gimpl:"field:city" pgimpl:"create;update"`
	Zip     string  `gimpl:"field:zip" pgimpl:"create;update"`
	Country Country `gimpl:"flat;prefix:country_"`
}

// pgimpl:Entity, SaveEntity
type Country struct {
	Code string `gimpl:"field:code;sort" pgimpl:"create;update"`
	Name string `gimpl:"field:name" pgimpl:"create;update"`
}

// pgimpl:Entity, SaveEntity
type User struct {
	ID      int     `gimpl:"field:id;sort;filter" pgimpl:"identify"`
	Name    string  `gimpl:"field:name;sort;filter"`
	Address Address `gimpl:"flat;prefix:address_"`
}
