package models

type Geodata struct {
	Geometry   Geometry   `json:"geometry" bson:"geometry"`
	Properties Properties `json:"properties" bson:"properties"`
}
type Geometry struct {
	Type        string    `json:"type" bson:"type"`
	Coordinates []float64 `json:"coordinates" bson:"coordinates"`
}

type Properties struct {
	Region              string `json:"Region" bson:"Region"`
	Postcode            string `json:"Postcode" bson:"Postcode"`
	City                string `json:"City" bson:"City"`
	Country             string `json:"Country" bson:"Country"`
	CountryAbbreviation string `json:"Country_abbreviation" bson:"country_abbreviation"`
	CountryCode         string `json:"Country_code" bson:"Country_ccode"`
	LocalAuthorityCode  string `json:"Local_authority_code" bson:"Local_authority_code"`
	RandNum             int32  `json:"Rand_num,omitempty" bson:"Rand_num,omitempty"`
}
