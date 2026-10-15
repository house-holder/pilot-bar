package parse

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/house-holder/pilot-bar/pkg/types"
)

type ParseContext struct {
	tokens []string
	input  *types.METARresponse
	output *types.METAR
}

type parseFunc func(c *ParseContext) error

func deriveFltCat(clouds []types.CloudData, vis types.Mi) string {
	ceiling := 99999
	for _, layer := range clouds {
		if layer.Coverage == "BKN" || layer.Coverage == "OVC" {
			if int(layer.Base) < ceiling {
				ceiling = int(layer.Base)
			}
		}
	}

	var cat string
	switch {
	case ceiling < 500 || vis < 1:
		cat = "LIFR"
	case ceiling < 1000 || vis < 3:
		cat = "IFR"
	case ceiling < 3000 || vis <= 5:
		cat = "MVFR"
	default:
		cat = "VFR"
	}
	return cat
}

func BuildInternalMETAR(data *types.METARresponse, output *types.METAR) error {
	output.RawOb = data.RawOb
	output.WxString = data.WxString

	output.Temp.AmbientExact = float64(data.Temp)
	output.Temp.DewpointExact = float64(data.Dewp)
	output.Temp.Ambient = int(data.Temp)
	output.Temp.Dewpoint = int(data.Dewp)
	output.Reported.Age = int(time.Since(time.Unix(data.ObsTime, 0)).Minutes())

	switch v := data.Visib.(type) {
	case float64:
		output.Visibility = types.Mi(v)
	case string:
		output.Visibility = types.Mi(99)
	}

	output.Clouds = make([]types.CloudData, 0)
	for _, layer := range data.Clouds {
		output.Clouds = append(output.Clouds, types.CloudData{
			Base:     types.Feet(layer.Base),
			Coverage: provideCloudCover(layer.Cover),
		})
	}

	output.Reported.Zulu = provideTimeData(data.ReportTime, "zulu")
	output.Reported.Local = provideTimeData(data.ReportTime, "local")

	parsers := []parseFunc{
		loadAltimeter, loadWind, loadWXString, loadRemarks, loadVisibility,
	}
	c := &ParseContext{
		tokens: strings.Split(data.RawOb, " "),
		output: output,
	}

	for _, parser := range parsers {
		err := parser(c)
		if err != nil {
			return err
		}
	}

	if data.FltCat == "" || data.FltCat == "null" {
		output.FltCat = deriveFltCat(output.Clouds, output.Visibility)
	} else {
		output.FltCat = data.FltCat
	}

	return nil
}

func provideTimeData(timeString string, request string) types.Time {
	timeValue, err := time.Parse("2006-01-02T15:04:05Z", timeString)
	if err != nil {
		slog.Error("provideTimeData failed", "error", err)
		return types.Time{}
	}
	if request == "zulu" {
		return types.Time{
			Day:  uint8(timeValue.In(time.UTC).Day()),
			Hour: uint8(timeValue.In(time.UTC).Hour()),
		}
	} else {
		return types.Time{
			Day:  uint8(timeValue.In(time.Local).Day()),
			Hour: uint8(timeValue.In(time.Local).Hour()),
		}
	}
}

func provideCloudCover(coverage string) string {
	return coverage
}

// TODO: slices refactor?
func loadAltimeter(ctx *ParseContext) error {
	for _, token := range ctx.tokens {
		if len(token) == 5 && strings.HasPrefix(token, "A") {
			altVal, err := strconv.ParseFloat(token[1:], 64)
			if err != nil {
				return fmt.Errorf("getAltimeter failed: %w", err)
			}
			ctx.output.Altimeter = types.InHg(altVal / 100.0)
			break
		}
	}
	return nil
}

// TODO: slices refactor?
func loadWind(ctx *ParseContext) error {
	for _, token := range ctx.tokens {
		if strings.HasSuffix(token, "KT") {
			if len(token) < 7 {
				continue // bad token length bailout ("180KT")
			}
			if token[:3] == "VRB" {
				ctx.output.Wind.Direction = types.DegMag(0)
				ctx.output.Wind.Variable = true
				speed, err := strconv.Atoi(token[3:5])
				if err != nil {
					return fmt.Errorf("getWind(0) failed: %w", err)
				}
				ctx.output.Wind.Speed = types.Knots(speed)
			} else {
				ctx.output.Wind.Variable = false
				direction, err := strconv.Atoi(token[0:3])
				if err != nil {
					return fmt.Errorf("getWind(1) failed: %w", err)
				}
				ctx.output.Wind.Direction = types.DegMag(direction)
				speed, err := strconv.Atoi(token[3:5])
				if err != nil {
					return fmt.Errorf("getWind(2) failed: %w", err)
				}
				ctx.output.Wind.Speed = types.Knots(speed)
			}

			if strings.Contains(token, "G") {
				gustsIdx := strings.Index(token, "G")
				gusts, err := strconv.Atoi(token[gustsIdx+1 : gustsIdx+3])
				if err != nil {
					return fmt.Errorf("getWind(3) failed: %w", err)
				}
				gustsValue := types.Knots(gusts)
				ctx.output.Wind.Gusts = &gustsValue
			} else {
				ctx.output.Wind.Gusts = nil
			}
		}
	}
	if ctx.output.Wind.Speed == 0 && ctx.output.Wind.Direction == 0 {
		ctx.output.Wind.Calm = true
	} else {
		ctx.output.Wind.Calm = false
	}
	return nil
}

func loadVisibility(ctx *ParseContext) error {
	lowestRVR := -1
	lowestStr := ""

	for _, token := range ctx.tokens {
		if !strings.HasPrefix(token, "R") ||
			!strings.Contains(token, "/") ||
			!strings.HasSuffix(token, "FT") {
			continue
		}

		parts := strings.SplitN(token, "/", 2)
		if len(parts) != 2 {
			continue
		}
		runway := strings.TrimPrefix(parts[0], "R")
		valuePart := strings.TrimSuffix(parts[1], "FT")

		prefix := ""
		if strings.HasPrefix(valuePart, "M") {
			prefix = "<"
			valuePart = strings.TrimPrefix(valuePart, "M")
		}

		val, err := strconv.Atoi(valuePart)
		if err != nil {
			continue
		}

		if lowestRVR == -1 || val < lowestRVR {
			lowestRVR = val
			lowestStr = fmt.Sprintf("%s/%s%dRVR", runway, prefix, val)
		}
	}

	if lowestRVR != -1 {
		ctx.output.Visibility = types.Mi(0.24)
		ctx.output.RVR = &lowestStr
		return nil
	}

	// no RVR — proceed with visibility token
	for i, token := range ctx.tokens {
		token = strings.ReplaceAll(token, "SMSM", "SM")

		if !strings.HasSuffix(token, "SM") {
			continue
		}

		if strings.HasPrefix(token, "P") {
			ctx.output.Visibility = types.Mi(99)
			return nil
		}

		if strings.HasPrefix(token, "M") {
			ctx.output.Visibility = types.Mi(0.24)
			return nil
		}

		if strings.Contains(token, "/") {
			parts := strings.Split(strings.TrimSuffix(token, "SM"), "/")
			if len(parts) == 2 {
				num, err1 := strconv.Atoi(parts[0])
				den, err2 := strconv.Atoi(parts[1])
				if err1 != nil || err2 != nil || den == 0 {
					return fmt.Errorf("loadVisibility: bad parsing %q", token)
				}
				frac := float64(num) / float64(den)
				if i > 0 {
					if whole, err := strconv.Atoi(ctx.tokens[i-1]); err == nil {
						frac += float64(whole)
					}
				}
				ctx.output.Visibility = types.Mi(frac)
				return nil
			}
		}

		val, err := strconv.ParseFloat(strings.TrimSuffix(token, "SM"), 64)
		if err != nil {
			return fmt.Errorf("loadVisibility: bad parsing %q: %w", token, err)
		}
		ctx.output.Visibility = types.Mi(val)
		return nil
	}

	ctx.output.Visibility = types.Mi(99)
	return nil
}

func loadWXString(ctx *ParseContext) error {
	return nil
}

func loadRemarks(ctx *ParseContext) error {
	idx := slices.Index(ctx.tokens, "RMK")
	if idx != -1 && idx+1 < len(ctx.tokens) {
		rmkTokens := ctx.tokens[idx+1:]
		raw, readable := processRemarks(rmkTokens)
		ctx.output.Remarks.Raw = raw
		ctx.output.Remarks.Readable = readable
	}
	return nil
}

var matchCases = [2]string{"AO2", "$"}

func processRemarks(remarkTokens []string) (raw []string, readable []string) {
	// TODO: if token meets certain criteria, make it human-readable
	// for now: arbitary creation of test strings
	for _, token := range remarkTokens {
		if slices.Contains(matchCases[:], token) {
			slog.Debug("match", "token", token)
			readable = append(readable, createReadableRemark(token))
		} else {
			slog.Debug("non-match", "token", token)
			raw = append(raw, token)
		}
	}
	return raw, readable
}

func createReadableRemark(token string) string {
	// TODO: need custom parsing. for now just output a string placedholder
	return fmt.Sprintf("%s match - createReadableRemark", token)
}
