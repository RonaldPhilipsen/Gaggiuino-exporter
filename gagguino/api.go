package gaggiuino

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const defaultHTTPTimeout = 10 * time.Second
const httpTimeoutEnvVar = "GAGGIUINO_HTTP_TIMEOUT"

type status struct {
	Uptime            int     `json:"upTime"`
	ProfileID         int     `json:"profileId"`
	ProfileName       string  `json:"profileName"`
	TargetTemperature float64 `json:"targetTemperature"`
	Temperature       float64 `json:"temperature"`
	Pressure          float64 `json:"pressure"`
	WaterLevel        int     `json:"waterLevel"`
	Weight            float64 `json:"weight"`
	BrewSwitchState   bool    `json:"brewSwitchState"`
	SteamSwitchState  bool    `json:"steamSwitchState"`
}

// UnmarshalJSON implements custom unmarshaling for status to handle string fields in JSON.
func (s *status) UnmarshalJSON(data []byte) error {
	var aux struct {
		Uptime            string `json:"upTime"`
		ProfileID         string `json:"profileId"`
		ProfileName       string `json:"profileName"`
		TargetTemperature string `json:"targetTemperature"`
		Temperature       string `json:"temperature"`
		Pressure          string `json:"pressure"`
		WaterLevel        string `json:"waterLevel"`
		Weight            string `json:"weight"`
		BrewSwitchState   bool   `json:"brewSwitchState"`
		SteamSwitchState  bool   `json:"steamSwitchState"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	if s.Uptime, err = strconv.Atoi(aux.Uptime); err != nil {
		return err
	}
	if s.ProfileID, err = strconv.Atoi(aux.ProfileID); err != nil {
		return err
	}
	s.ProfileName = aux.ProfileName
	if s.TargetTemperature, err = strconv.ParseFloat(aux.TargetTemperature, 64); err != nil {
		return err
	}
	if s.Temperature, err = strconv.ParseFloat(aux.Temperature, 64); err != nil {
		return err
	}
	if s.Pressure, err = strconv.ParseFloat(aux.Pressure, 64); err != nil {
		return err
	}
	if s.WaterLevel, err = strconv.Atoi(aux.WaterLevel); err != nil {
		return err
	}
	if s.Weight, err = strconv.ParseFloat(aux.Weight, 64); err != nil {
		return err
	}
	s.BrewSwitchState = aux.BrewSwitchState
	s.SteamSwitchState = aux.SteamSwitchState
	return nil
}

func getHTTPTimeout() time.Duration {
	raw := os.Getenv(httpTimeoutEnvVar)
	if raw == "" {
		return defaultHTTPTimeout
	}

	timeout, err := time.ParseDuration(raw)
	if err != nil || timeout <= 0 {
		log.Printf("Invalid %s=%q, using default %s", httpTimeoutEnvVar, raw, defaultHTTPTimeout)
		return defaultHTTPTimeout
	}

	return timeout
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: getHTTPTimeout()}
}

func doRequest(ctx context.Context, path string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := newHTTPClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return body, resp.StatusCode, nil
}

func GetStateWithContext(ctx context.Context, baseURL string) (status, error) {
	stateURL := baseURL + "/api/system/status"
	body, statusCode, err := doRequest(ctx, stateURL)
	if err != nil {
		return status{}, err
	}

	if statusCode != http.StatusOK {
		return status{}, fmt.Errorf("unexpected status code: %d", statusCode)
	}

	var arr []status
	if err := json.Unmarshal(body, &arr); err == nil {
		if len(arr) > 0 {
			return arr[0], nil
		}
		return status{}, fmt.Errorf("failed to parse status: empty array")
	}
	return status{}, fmt.Errorf("failed to parse status from response: %s", string(body))
}

func GetLastShotWithContext(ctx context.Context, baseURL string) (int, error) {
	/*   Get the last shot from the Gaggiuino API.
	 */
	lastShotURL := baseURL + "/api/shots/latest"
	body, statusCode, err := doRequest(ctx, lastShotURL)
	if err != nil {
		return -1, err
	}

	if statusCode != http.StatusOK {
		return -1, fmt.Errorf("unexpected status code: %d", statusCode)
	}

	type lastShotIDResponse struct {
		LastShotID int `json:"lastShotId"`
	}

	var responseData []lastShotIDResponse
	if err := json.Unmarshal(body, &responseData); err == nil {
		if len(responseData) == 0 {
			return -1, fmt.Errorf("failed to parse last shot ID: empty response")
		}
		return responseData[0].LastShotID, nil
	}

	return -1, fmt.Errorf("failed to parse last shot ID from response: %s", string(body))
}

func GetShotWithContext(ctx context.Context, baseURL string, shotID int) (*LastShot, error) {
	shotURL := fmt.Sprintf("%s/api/shots/%d", baseURL, shotID)

	body, statusCode, err := doRequest(ctx, shotURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get last shot: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", statusCode)
	}

	var lastShot LastShot
	if err := json.Unmarshal(body, &lastShot); err != nil {
		return nil, fmt.Errorf("failed to parse getLastShot response: %w", err)
	}

	return &lastShot, nil
}
