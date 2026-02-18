package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/house-holder/pilot-bar/pkg/types"
)

const cacheFile = "currentWX.json"

func dir() (string, error) {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cache: home dir: %w", err)
		}
		cacheDir = filepath.Join(home, ".cache")
	}
	return filepath.Join(cacheDir, "pilot-bar"), nil
}

func Read() (types.Airport, error) {
	d, err := dir()
	if err != nil {
		return types.Airport{}, err
	}
	data, err := os.ReadFile(filepath.Join(d, cacheFile))
	if err != nil {
		return types.Airport{}, err
	}
	var airport types.Airport
	if err := json.Unmarshal(data, &airport); err != nil {
		return types.Airport{}, fmt.Errorf("cache: unmarshal: %w", err)
	}
	return airport, nil
}

func Write(airport types.Airport) error {
	d, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0755); err != nil {
		return fmt.Errorf("cache: mkdir: %w", err)
	}
	data, err := json.MarshalIndent(airport, "", "  ")
	if err != nil {
		return fmt.Errorf("cache: marshal: %w", err)
	}
	return os.WriteFile(filepath.Join(d, cacheFile), data, 0644)
}

func ReadICAO() (string, error) {
	airport, err := Read()
	if err != nil {
		return "", err
	}
	return airport.ICAO, nil
}

func EnsureExists(icao string) error {
	d, err := dir()
	if err != nil {
		return err
	}
	p := filepath.Join(d, cacheFile)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return Write(types.Airport{
			ICAO: icao,
			METAR: types.METAR{
				Reported: types.Timestamp{Epoch: 0},
			},
		})
	}
	return nil
}
