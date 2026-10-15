package main

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"strings"

	"github.com/house-holder/pilot-bar/internal/cache"
	"github.com/house-holder/pilot-bar/internal/config"
	"github.com/house-holder/pilot-bar/pkg/types"
	"github.com/spf13/pflag"
)

type WaybarOutput struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
	Alt     string `json:"alt"`
}

func main() {
	format := pflag.StringP("format", "f", "", "override config format string")
	pflag.Parse()

	cfg := config.Load()
	barFormat := cfg.Format
	if pflag.Lookup("format").Changed {
		barFormat = *format
	}

	wx, err := cache.Read()
	if err != nil {
		os.Exit(0)
	}

	out := WaybarOutput{
		Text:    formatText(wx, barFormat, cfg.TempUnit),
		Tooltip: formatTooltip(wx),
		Class:   strings.ToLower(wx.METAR.FltCat),
		Alt:     wx.METAR.FltCat,
	}

	json.NewEncoder(os.Stdout).Encode(out)
}

const (
	visUnlimited = 99.0
	visThreshold = 6.0
)

var cloudIcons = map[string]string{
	"FEW": "\U000F0A9F", // 󰪟
	"SCT": "\U000F0AA1", // 󰪡
	"BKN": "\U000F0AA3", // 󰪣
	"OVC": "\U000F0AA5", // 󰪥
}

func convertTemps(ambient, dewpoint float64, unit string) (float64, float64) {
	if unit == "C" {
		return ambient, dewpoint
	}
	return (ambient*9.0/5.0 + 32), (dewpoint*9.0/5.0 + 32)
}

func formatText(wx types.Airport, format string, tempUnit string) string {
	m := wx.METAR
	icon, alt, hasCeiling := ceiling(m.Clouds)
	ambF, dewF := convertTemps(m.Temp.AmbientExact, m.Temp.DewpointExact, tempUnit)

	replacer := strings.NewReplacer(
		"{temps}", fmt.Sprintf("%.1f/%.1f", ambF, dewF),
		"{temp}", fmt.Sprintf("%.1f", ambF),
		"{dewp}", fmt.Sprintf("%.1f", dewF),
		"{winds}", fmtWind(m.Wind),
		"{cloud-icon}", fmtIf(hasCeiling, icon),
		"{clouds}", fmtIf(hasCeiling, fmt.Sprintf("%03d", alt)),
		"{vis}", fmtVis(m),
		"{wx}", m.WxString,
		"{station}", wx.ICAO,
		"{age}", fmt.Sprintf("%d", m.Reported.Age),
		"{cat}", m.FltCat,
		"{alt}", fmt.Sprintf("%.2f", float64(m.Altimeter)),
	)

	result := replacer.Replace(format)
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}
	return strings.TrimSpace(result)
}

func fmtIf(ok bool, val string) string {
	if ok {
		return val
	}
	return ""
}

func fmtWind(w types.WindData) string {
	if w.Calm {
		return "WC"
	}
	var s string
	if w.Variable {
		s = fmt.Sprintf("VRB %d", w.Speed)
	} else {
		s = fmt.Sprintf("%03d/%d", w.Direction, w.Speed)
	}
	if w.Gusts != nil {
		s += fmt.Sprintf("G%d", *w.Gusts)
	}
	return s
}

func fmtVis(m types.METAR) string {
	if m.RVR != nil {
		return *m.RVR
	}
	v := float64(m.Visibility)
	if v <= 0 || v >= visThreshold {
		return ""
	}
	return fmt.Sprintf("%gSM", v)
}

func ceiling(clouds []types.CloudData) (icon string, alt int, ok bool) {
	for _, layer := range clouds {
		if layer.Coverage == "BKN" || layer.Coverage == "OVC" {
			if ic, found := cloudIcons[layer.Coverage]; found {
				return ic, int(layer.Base) / 100, true
			}
		}
	}
	if len(clouds) > 0 {
		first := clouds[0]
		if ic, found := cloudIcons[first.Coverage]; found {
			return ic, int(first.Base) / 100, true
		}
	}
	return "", 0, false
}

func formatTooltip(wx types.Airport) string {
	var b strings.Builder
	metarHTML := html.EscapeString(wx.METAR.RawOb)

	fmt.Fprintf(&b, "<tt>%s</tt>", metarHTML)
	if wx.RawTAF != "" {
		tafHTML := html.EscapeString(wx.RawTAF)
		fmt.Fprintf(&b, "\n\n<tt>%s</tt>", wrapTAF(tafHTML))
	}
	if wx.RawAFD != "" {
		afdHTML := html.EscapeString(wx.RawAFD)
		fmt.Fprintf(&b, "\n\n%s", afdHTML)
	}
	return b.String()
}

func wrapTAF(raw string) string {
	r := strings.NewReplacer(
		" FM", "\n  FM",
		" TEMPO", "\n  TEMPO",
		" BECMG", "\n  BECMG",
		" PROB", "\n  PROB",
	)
	return r.Replace(raw)
}
