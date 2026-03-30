package gaggiuino

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

const defaultBaseURL string = "http://gaggiuino.local"

type status struct {
	Uptime 				int 	`json:"upTime"`
	ProfileId 			int 	`json:"profileId"`
	ProfileName 		string	`json:"profileName"`
	TargetTemperature 	float64 `json:"targetTemperature"`
	Temperature 		float64 `json:"temperature"`
	Pressure			float64 `json:"pressure"`
	WaterLevel			int		`json:"waterLevel"`
	Weight				float64	`json:"weight"`
	BrewSwitchState		bool	`json:"brewSwitchState"`
	SteamSwitchState	bool	`json:"steamSwitchState"`
	}
	// UnmarshalJSON implements custom unmarshaling for status to handle string fields in JSON.
	func (s *status) UnmarshalJSON(data []byte) error {
		var aux struct {
			Uptime            string `json:"upTime"`
			ProfileId         string `json:"profileId"`
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
		if s.ProfileId, err = strconv.Atoi(aux.ProfileId); err != nil {
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


func GetBaseUrl() string {
	/* 	The base URL of the Gaggiuino API can be set via the Gaggiuino_BASE_URL environment variable.
	If not set, it defaults to http://gaggiuino.local.
	We strip any authentication info from the URL, as we don't want to log it.
	*/

	var baseURL = os.Getenv("Gaggiuino_BASE_URL")
	//
	if baseURL != "" {
		u, err := url.Parse(baseURL)
		if err != nil {
			log.Printf("Invalid base URL from env %s: %v", baseURL, err)
			return defaultBaseURL
		}
		log.Printf("Using base URL from env %s:%s", u.Scheme, u.Host)
		return u.String()
	}
	log.Printf("Using default base URL: %s", defaultBaseURL)
	return defaultBaseURL
}

func GetState(baseURL string) (status, error) {
	stateURL := baseURL + "/api/system/status"
	resp, err := http.Get(stateURL)
	if err != nil {
		return status{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return status{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return status{}, err
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


func GetLastShot(baseURL string) (int, error) {
	/*   Get the last shot from the Gaggiuino API.
	 */
	lastShotURL := baseURL + "/api/shots/latest"
	resp, err := http.Get(lastShotURL)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, err
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

func GetShot(baseURL string, shotID int) (*LastShot, error) {
	shotURL := fmt.Sprintf("%s/api/shots/%d", baseURL, shotID)

	resp, err := http.Get(shotURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get last shot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var lastShot LastShot
	if err := json.Unmarshal(body, &lastShot); err != nil {
		return nil, fmt.Errorf("failed to parse getLastShot response: %w", err)
	}

	return &lastShot, nil
}

