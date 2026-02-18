package types

type (
	DegMag uint16 // 1-360, degrees magnetic
	Knots  int
	Feet   int
	Mi     float64
	InHg   float64
)

type Timestamp struct {
	Age   int   `json:"age_minutes"`
	Epoch int64 `json:"epoch"`
	Local Time  `json:"local"`
	Zulu  Time  `json:"zulu"`
}

type Time struct {
	Day  uint8 `json:"day"`  // validate: 1-31
	Hour uint8 `json:"hour"` // validate: 0-23
}
