package gaggiuino

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestGetLastShot(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedID     int
		expectedErrMsg string
	}{
		{
			name:         "parses array response",
			statusCode:   http.StatusOK,
			responseBody: `[{"lastShotId": 644}]`,
			expectedID:   644,
		},
		{
			name:         "parses second array response",
			statusCode:   http.StatusOK,
			responseBody: `[{"lastShotId": 721}]`,
			expectedID:   721,
		},
		{
			name:           "returns error for non-200 status",
			statusCode:     http.StatusBadGateway,
			responseBody:   `{"error":"upstream unavailable"}`,
			expectedID:     -1,
			expectedErrMsg: "unexpected status code: 502",
		},
		{
			name:           "returns error for empty array",
			statusCode:     http.StatusOK,
			responseBody:   `[]`,
			expectedID:     -1,
			expectedErrMsg: "failed to parse last shot ID: empty response",
		},
		{
			name:           "returns error for invalid payload",
			statusCode:     http.StatusOK,
			responseBody:   `not-json`,
			expectedID:     -1,
			expectedErrMsg: "failed to parse last shot ID from response",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/shots/latest" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			id, err := GetLastShot(server.URL)

			if tt.expectedErrMsg == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				if id != tt.expectedID {
					t.Fatalf("expected id %d, got %d", tt.expectedID, id)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.expectedErrMsg)
			}
			if !strings.Contains(err.Error(), tt.expectedErrMsg) {
				t.Fatalf("expected error containing %q, got %q", tt.expectedErrMsg, err.Error())
			}
			if id != -1 {
				t.Fatalf("expected id -1 on error, got %d", id)
			}
		})
	}
}
func TestGetState(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedStatus status
		expectedErrMsg string
	}{
		{
			name:         "parses valid status array",
			statusCode:   http.StatusOK,
			responseBody: `[{"upTime": "67", "profileId": "10", "profileName": "[UT] Boiler Off (community)", "targetTemperature": "1.000000", "temperature": "23.501986", "pressure": "0.168047", "waterLevel": "29", "weight": "1.100000", "brewSwitchState": false, "steamSwitchState": false}]`,
			expectedStatus: status{
				Uptime:            67,
				ProfileId:         10,
				ProfileName:       "[UT] Boiler Off (community)",
				TargetTemperature: 1.0,
				Temperature:       23.501986,
				Pressure:          0.168047,
				WaterLevel:        29,
				Weight:            1.1,
				BrewSwitchState:   false,
				SteamSwitchState:  false,
			},
		},
		{
			name:           "returns error for non-200 status",
			statusCode:     http.StatusBadGateway,
			responseBody:   `[{"error":"upstream unavailable"}]`,
			expectedErrMsg: "unexpected status code: 502",
		},
		{
			name:           "returns error for invalid payload",
			statusCode:     http.StatusOK,
			responseBody:   `not-json`,
			expectedErrMsg: "failed to parse status from response",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/system/status" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			st, err := GetState(server.URL)

			if tt.expectedErrMsg == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				if st != tt.expectedStatus {
					t.Fatalf("expected status %+v, got %+v", tt.expectedStatus, st)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.expectedErrMsg)
			}
			if !strings.Contains(err.Error(), tt.expectedErrMsg) {
				t.Fatalf("expected error containing %q, got %q", tt.expectedErrMsg, err.Error())
			}
		})
	}
}

func TestGetShot(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		shotID         int
		expectedErrMsg string
		expectedShotID int
	}{
		{
			name:       "returns parsed shot",
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 644,
				"timestamp": 1711111111,
				"duration": 312,
				"datapoints": {
					"timeInShot": [0, 1],
					"pressure": [1, 2]
				}
			}`,
			shotID:         644,
			expectedShotID: 644,
		},
		{
			name:           "returns error for non-200 status",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error":"failed"}`,
			shotID:         123,
			expectedErrMsg: "unexpected status code: 500",
		},
		{
			name:           "returns error for invalid shot payload",
			statusCode:     http.StatusOK,
			responseBody:   `{"id":`,
			shotID:         7,
			expectedErrMsg: "failed to parse getLastShot response",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/api/shots/" + strconv.Itoa(tt.shotID)
				if r.URL.Path != expectedPath {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			shot, err := GetShot(server.URL, tt.shotID)

			if tt.expectedErrMsg == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				if shot == nil {
					t.Fatalf("expected shot, got nil")
				}
				if shot.ID != tt.expectedShotID {
					t.Fatalf("expected shot id %d, got %d", tt.expectedShotID, shot.ID)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.expectedErrMsg)
			}
			if !strings.Contains(err.Error(), tt.expectedErrMsg) {
				t.Fatalf("expected error containing %q, got %q", tt.expectedErrMsg, err.Error())
			}
			if shot != nil {
				t.Fatalf("expected nil shot on error, got %+v", shot)
			}
		})
	}
}
