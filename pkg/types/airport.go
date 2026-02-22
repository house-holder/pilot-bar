package types

type Airport struct {
	ICAO            string `json:"icao"`
	Name            string `json:"name"`
	CWA             string `json:"cwa"`
	LastUpdateEpoch int64  `json:"last_update"`
	Elevation       Feet   `json:"elevation"`
	METAR           METAR  `json:"metar"`
	RawTAF          string `json:"rawTAF"`
	RawAFD          string `json:"rawAFD"`
}
