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

// --------------------------------------------------------------------------------------
type parseFunc func(c *ParseContext) error

func BuildInternalMETAR(data *types.METARresponse, output *types.METAR) error {
	output.Temp.AmbientExact = float64(data.Temp)
	output.Temp.DewpointExact = float64(data.Dewp)
	output.Temp.Ambient = int(data.Temp)
	output.Temp.Dewpoint = int(data.Dewp)
	output.Reported.Age = int(time.Since(time.Unix(data.ObsTime, 0)).Minutes())

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
		loadAltimeter, loadWind, loadWXString, loadRemarks,
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
	switch coverage {
	case "FEW":
		return "few"
	case "SCT":
		return "scattered"
	case "BKN":
		return "broken"
	case "OVC":
		return "overcast"
	default:
		return coverage
	}
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

			if ctx.output.Wind.Speed == 0 && ctx.output.Wind.Direction == 0 {
				ctx.output.Wind.Calm = true
			} else {
				ctx.output.Wind.Calm = false
			}
		}
	}
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
