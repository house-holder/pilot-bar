package main

import (
	"log/slog"

	"github.com/spf13/pflag"
)

type Flags struct {
	Airport *string
	Debug   *bool
	Info    *bool
	Update  *bool
	Verbose *bool
}

func setupFlags() Flags {
	info := pflag.BoolP("info", "i", false, "enable info logging")
	debug := pflag.BoolP("debug", "d", false, "enable debug logging")
	update := pflag.BoolP("update", "u", false, "force update cycle")
	verbose := pflag.BoolP("verbose", "v", false, "enable verbose output")

	defaultID, err := resolveAirport()
	if err != nil {
		slog.Error("failed to resolve default airport", "error", err)
		defaultID = "KCGI"
	}
	airport := pflag.StringP("airport", "a", defaultID, "target station ID")

	pflag.Parse()
	return Flags{
		Airport: airport,
		Debug:   debug,
		Info:    info,
		Update:  update,
		Verbose: verbose,
	}
}

func main() {
	flags := setupFlags()
	InitLogger(flags)

	if err := Update(flags); err != nil {
		slog.Error("Update", "error", err)
	}
}
