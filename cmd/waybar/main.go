package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/house-holder/pilot-bar/internal/cache"
	"github.com/house-holder/pilot-bar/pkg/types"
)

type WaybarOutput struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
	Alt     string `json:"alt"`
}

func main() {
	wx, err := cache.Read()
	if err != nil {
		os.Exit(0)
	}

	out := WaybarOutput{
		Text:    formatText(wx),
		Tooltip: formatTooltip(wx),
		Class:   strings.ToLower(wx.METAR.FltCat),
		Alt:     wx.METAR.FltCat,
	}

	json.NewEncoder(os.Stdout).Encode(out)
}

const visUnlimited = 99.0

var cloudIcons = map[string]string{
	"FEW": "\U000F0A9F",
	"SCT": "\U000F0AA1",
	"BKN": "\U000F0AA3",
	"OVC": "\U000F0AA5",
}

func formatText(wx types.Airport) string {
	m := wx.METAR
	var parts []string

	parts = append(parts, fmt.Sprintf("%d/%d", m.Temp.Ambient, m.Temp.Dewpoint))

	if float64(m.Visibility) > 0 && float64(m.Visibility) < 6 {
		parts = append(parts, fmt.Sprintf("%gSM", float64(m.Visibility)))
	}

	if icon, alt, ok := ceiling(m.Clouds); ok {
		parts = append(parts, fmt.Sprintf("%s %03d", icon, alt))
	}

	if m.WxString != "" {
		parts = append(parts, m.WxString)
	}

	return strings.Join(parts, " ")
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
	m := wx.METAR
	var b strings.Builder

	fmt.Fprintf(&b, "<b>%s</b>", wx.ICAO)
	if wx.Name != "" {
		fmt.Fprintf(&b, " — %s", wx.Name)
	}
	b.WriteString("\n")

	if m.Wind.Calm {
		b.WriteString("Wind: Calm\n")
	} else if m.Wind.Variable {
		fmt.Fprintf(&b, "Wind: Variable at %d kt\n", m.Wind.Speed)
	} else {
		fmt.Fprintf(&b, "Wind: %03d° at %d kt", m.Wind.Direction, m.Wind.Speed)
		if m.Wind.Gusts != nil {
			fmt.Fprintf(&b, " G%d", *m.Wind.Gusts)
		}
		b.WriteString("\n")
	}

	if float64(m.Visibility) > 0 && float64(m.Visibility) < visUnlimited {
		fmt.Fprintf(&b, "Visibility: %g SM\n", float64(m.Visibility))
	}

	for _, layer := range m.Clouds {
		fmt.Fprintf(&b, "%s %03d\n", layer.Coverage, int(layer.Base)/100)
	}

	fmt.Fprintf(&b, "Temp: %d°C / Dewpoint: %d°C\n", m.Temp.Ambient, m.Temp.Dewpoint)
	fmt.Fprintf(&b, "Altimeter: %.2f inHg\n", m.Altimeter)
	fmt.Fprintf(&b, "\n<i>%d min ago</i>", m.Reported.Age)

	return b.String()
}
