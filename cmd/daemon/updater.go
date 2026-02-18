package main

import (
	"log/slog"
	"time"

	"github.com/house-holder/pilot-bar/internal/cache"
	"github.com/house-holder/pilot-bar/internal/fetch"
	"github.com/house-holder/pilot-bar/internal/parse"
	"github.com/house-holder/pilot-bar/pkg/types"
)

const (
	MaxTries = 5

	// NOTE: seconds vals below, 1min for testing - must edit for prod
	UpdateInterval = 60
	IntervalMETAR  = 60
	IntervalTAF    = 60
	IntervalAFD    = 60
)

type UpdateData struct {
	cached        types.Airport
	requested     string
	now           int64
	intervalMETAR int64
	intervalTAF   int64
	intervalAFD   int64
}

func (d *UpdateData) ICAOChanged() bool {
	return d.cached.ICAO != d.requested
}

func (d *UpdateData) METAREmpty() bool {
	return d.cached.METAR.Reported.Epoch == 0
}

func (d *UpdateData) TimeExpired() bool {
	return d.now-d.cached.LastUpdateEpoch > UpdateInterval
}

func (d *UpdateData) NeedsAnyUpdate(force bool) bool {
	if force || d.ICAOChanged() || d.METAREmpty() || d.TimeExpired() {
		slog.Debug("Proceeding with update", "list", map[string]any{
			"New ID":       d.ICAOChanged(),
			"Timeout":      d.TimeExpired(),
			"Force update": force,
		})
		return true
	}
	slog.Debug("No update needed")
	return false
}

func Update(flags Flags) error {
	if err := cache.EnsureExists(*flags.Airport); err != nil {
		return err
	}

	cachedWX, err := cache.Read()
	if err != nil {
		return err
	}

	d := &UpdateData{
		cached:        cachedWX,
		requested:     *flags.Airport,
		now:           time.Now().Unix(),
		intervalMETAR: IntervalMETAR,
		intervalTAF:   IntervalTAF,
		intervalAFD:   IntervalAFD,
	}

	if !d.NeedsAnyUpdate(*flags.Update) {
		return nil
	}

	APImetar, err := fetch.GetMETAR(*flags.Airport, MaxTries)
	if err != nil {
		return err
	}

	if *flags.Verbose {
		displayMETAR(APImetar)
	} else {
		slog.Debug("", "metar", APImetar.RawOb)
	}

	if err := parse.BuildInternalMETAR(&APImetar, &cachedWX.METAR); err != nil {
		return err
	}

	if d.ICAOChanged() {
		cachedWX.ICAO = *flags.Airport
		cachedWX.Elevation = types.Feet(float64(APImetar.Elev) * 3.28084)
	}

	cachedWX.Name = APImetar.Name
	cachedWX.LastUpdateEpoch = time.Now().Unix()
	cachedWX.METAR.Reported.Epoch = int64(APImetar.ObsTime)

	return cache.Write(cachedWX)
}

func resolveAirport() (string, error) {
	icao, err := cache.ReadICAO()
	if err == nil && icao != "" {
		return icao, nil
	}
	slog.Warn("no cached or provided airport. using default: KCGI")
	return "KCGI", nil
}
