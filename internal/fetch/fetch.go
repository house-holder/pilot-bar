package fetch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/house-holder/pilot-bar/pkg/types"
)

const baseURL = "https://aviationweather.gov/api/data"

// FetchMETAR loads full report into a default-shaped struct
func GetMETAR(icao string, maxAttempts int) (types.METARresponse, error) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	metarURL := fmt.Sprintf("%s/metar?ids=%s&format=json", baseURL, icao)
	client := &http.Client{Timeout: 10 * time.Second}
	startTime := time.Now()

	var payload []types.METARresponse
	err := doWithRetry(maxAttempts, func(attempt int) (bool, error) {
		if attempt > 1 {
			slog.Info(fmt.Sprintf("Fetch METAR retry (%d of %d)", attempt, maxAttempts))
		} else {
			slog.Info("Fetching METAR")
		}

		resp, err := client.Get(metarURL)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				slog.Warn("Fetch timeout", "attempt", attempt, "max", maxAttempts)
				return true, err
			}
			return false, fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		if statusRetryOK(resp.StatusCode) {
			slog.Warn("OK to retry", "status", resp.Status, "attempt", attempt)
			return true, fmt.Errorf("status %d: %s", resp.StatusCode, resp.Status)
		}

		if resp.StatusCode != http.StatusOK {
			return false, fmt.Errorf("status: %s", resp.Status)
		}

		var decoded []types.METARresponse
		if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
			return false, fmt.Errorf("decode failed: %w", err)
		}
		if len(decoded) == 0 {
			return false, fmt.Errorf("no METAR data for %s", icao)
		}

		payload = decoded
		return false, nil
	})

	if err != nil {
		return types.METARresponse{}, err
	}

	fetchDuration := time.Since(startTime).Seconds()
	slog.Info("Fetch OK", "took", fmt.Sprintf("%.3fs", fetchDuration))
	return payload[0], nil
}

func GetTAF(icao string, maxAttempts int) (types.TAFresponse, error) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	tafURL := fmt.Sprintf("%s/taf?ids=%s&format=json", baseURL, icao)
	client := &http.Client{Timeout: 10 * time.Second}
	startTime := time.Now()

	var payload []types.TAFresponse
	err := doWithRetry(maxAttempts, func(attempt int) (bool, error) {
		if attempt > 1 {
			slog.Info(fmt.Sprintf("Fetch TAF retry (%d of %d)", attempt, maxAttempts))
		} else {
			slog.Info("Fetching TAF")
		}

		resp, err := client.Get(tafURL)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				slog.Warn("Fetch timeout", "attempt", attempt, "max", maxAttempts)
				return true, err
			}
			return false, fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		if statusRetryOK(resp.StatusCode) {
			slog.Warn("OK to retry", "status", resp.Status, "attempt", attempt)
			return true, fmt.Errorf("status %d: %s", resp.StatusCode, resp.Status)
		}

		if resp.StatusCode != http.StatusOK {
			return false, fmt.Errorf("status: %s", resp.Status)
		}

		var decoded []types.TAFresponse
		if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
			return false, fmt.Errorf("decode failed: %w", err)
		}
		if len(decoded) == 0 {
			return false, fmt.Errorf("no TAF data for %s", icao)
		}

		payload = decoded
		return false, nil
	})

	if err != nil {
		return types.TAFresponse{}, err
	}

	fetchDuration := time.Since(startTime).Seconds()
	slog.Info("TAF OK", "took", fmt.Sprintf("%.3fs", fetchDuration))
	return payload[0], nil
}

func LookupCWA(lat, lon float64) (string, error) {
	url := fmt.Sprintf("https://api.weather.gov/points/%.4f,%.4f", lat, lon)
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "pilot-bar")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("CWA lookup failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("CWA lookup status: %s", resp.Status)
	}

	var result struct {
		Properties struct {
			CWA string `json:"cwa"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("CWA decode failed: %w", err)
	}
	if result.Properties.CWA == "" {
		return "", fmt.Errorf("no CWA found for %.4f,%.4f", lat, lon)
	}

	slog.Info("CWA resolved", "cwa", result.Properties.CWA)
	return result.Properties.CWA, nil
}

func GetAFD(cwa string) (string, error) {
	wfo := "k" + strings.ToLower(cwa)
	url := fmt.Sprintf("%s/fcstdisc?cwa=%s&type=afd", baseURL, wfo)
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("AFD fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AFD fetch status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("AFD read failed: %w", err)
	}

	text := strings.TrimSpace(string(body))
	if text == "" {
		return "", fmt.Errorf("empty AFD for CWA %s", cwa)
	}

	slog.Info("AFD OK")
	return text, nil
}

func doWithRetry(maxAttempts int, op func(attempt int) (bool, error)) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		retry, err := op(attempt)
		if err == nil {
			return nil
		}

		lastErr = err
		if !retry || attempt == maxAttempts {
			break
		}
		time.Sleep(2 * time.Second) // backoff delay
	}

	return lastErr
}

func statusRetryOK(code int) bool {
	switch code {
	case http.StatusRequestTimeout, http.StatusTooManyRequests,
		http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}
