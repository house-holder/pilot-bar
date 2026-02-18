package types

type Airport struct {
	ICAO            string `json:"icao"`
	Name            string `json:"name"`
	LastUpdateEpoch int64  `json:"last_update"`
	Elevation       Feet   `json:"elevation"`
	METAR           METAR  `json:"metar"`
}
