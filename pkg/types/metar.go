package types

type METARresponse struct { // the full data returned by the API
	IcaoID      string   `json:"icaoId"`
	ReceiptTime string   `json:"receiptTime"`
	ObsTime     int64    `json:"obsTime"`
	ReportTime  string   `json:"reportTime"`
	MetarType   string   `json:"metarType"`
	Temp        float64  `json:"temp"`
	Dewp        float64  `json:"dewp"`
	Wdir        any      `json:"wdir"`
	Wspd        any      `json:"wspd"`
	Visib       any      `json:"visib"`
	Altim       float64  `json:"altim"`
	WxString    string   `json:"wxString"`
	QcField     int      `json:"qcField"`
	Slp         float64  `json:"slp"`
	PresTend    *float64 `json:"presTend"`
	MaxT        *float64 `json:"maxT"`
	MinT        *float64 `json:"minT"`
	MaxT24      *float64 `json:"maxT24"`
	MinT24      *float64 `json:"minT24"`
	Precip      *float64 `json:"precip"`
	Pcp3hr      *float64 `json:"pcp3hr"`
	Pcp6hr      *float64 `json:"pcp6hr"`
	Pcp24hr     *float64 `json:"pcp24hr"`
	Snow        *float64 `json:"snow"`
	VertVis     *float64 `json:"vertVis"`
	RawOb       string   `json:"rawOb"`
	Lat         float64  `json:"lat"`
	Long        float64  `json:"lon"`
	Elev        int      `json:"elev"`
	Name        string   `json:"name"`
	Cover       string   `json:"cover"`
	Clouds      []struct {
		Cover string `json:"cover"`
		Base  int    `json:"base"`
	} `json:"clouds"`
	FltCat string `json:"fltCat"`
}

// component structs
type WindData struct {
	Direction DegMag `json:"direction"`
	Speed     Knots  `json:"speed"`
	Gusts     *Knots `json:"gusts"`
	Variable  bool   `json:"variable"`
	Calm      bool   `json:"calm"`
}

type CloudData struct {
	Base     Feet   `json:"base"`
	Coverage string `json:"coverage"`
}

type TempData struct {
	Ambient       int     `json:"ambient"`
	Dewpoint      int     `json:"dewpoint"`
	AmbientExact  float64 `json:"ambientExact"`
	DewpointExact float64 `json:"dewpointExact"`
}

// main internal struct
type METAR struct {
	Reported   Timestamp   `json:"reported"`
	FltCat     string      `json:"fltCat"`
	WxString   string      `json:"wxString"`
	Wind       WindData    `json:"wind"`
	Visibility Mi          `json:"visibility"`
	Clouds     []CloudData `json:"clouds"`
	Temp       TempData    `json:"temp"`
	Altimeter  InHg        `json:"altimeter"`
	Remarks    struct {
		Raw      []string `json:"raw"`
		Readable []string `json:"readable"`
	} `json:"remarks"`
}
