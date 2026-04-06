package climgen

import (
	"encoding/json"
	"fmt"
	"os"
)

type RiverRouteModeSettings struct {
	Name                 string  `json:"name"`
	MinChannelStrength   float64 `json:"minChannelStrength"`
	MinRunoff            float64 `json:"minRunoff"`
	MinNavigability      float64 `json:"minNavigability"`
	DailyRange           float64 `json:"dailyRange"`
	PayloadCapacity      float64 `json:"payloadCapacity"`
	LongHaulTolerance    float64 `json:"longHaulTolerance"`
	UpstreamPenalty      float64 `json:"upstreamPenalty"`
	DownstreamBonus      float64 `json:"downstreamBonus"`
	LakeBonus            float64 `json:"lakeBonus"`
	PortageTolerance     float64 `json:"portageTolerance"`
	FloodTolerance       float64 `json:"floodTolerance"`
	TowpathBenefit       float64 `json:"towpathBenefit"`
	TowpathReliefLimit   float64 `json:"towpathReliefLimit"`
	TransferSupportFloor float64 `json:"transferSupportFloor"`
}

type RiverRouteSettings struct {
	DefaultMode string                   `json:"defaultMode"`
	Modes       []RiverRouteModeSettings `json:"modes"`
}

func DefaultRiverRouteSettings() RiverRouteSettings {
	return RiverRouteSettings{
		DefaultMode: "barge",
		Modes: []RiverRouteModeSettings{
			{
				Name:                 "barge",
				MinChannelStrength:   0.95,
				MinRunoff:            18.0,
				MinNavigability:      0.34,
				DailyRange:           0.96,
				PayloadCapacity:      0.92,
				LongHaulTolerance:    0.88,
				UpstreamPenalty:      0.46,
				DownstreamBonus:      0.22,
				LakeBonus:            0.12,
				PortageTolerance:     0.18,
				FloodTolerance:       0.54,
				TowpathBenefit:       0.32,
				TowpathReliefLimit:   260.0,
				TransferSupportFloor: 0.20,
			},
			{
				Name:                 "canoe",
				MinChannelStrength:   0.55,
				MinRunoff:            8.0,
				MinNavigability:      0.24,
				DailyRange:           0.82,
				PayloadCapacity:      0.42,
				LongHaulTolerance:    0.48,
				UpstreamPenalty:      0.30,
				DownstreamBonus:      0.12,
				LakeBonus:            0.08,
				PortageTolerance:     0.62,
				FloodTolerance:       0.60,
				TowpathBenefit:       0.10,
				TowpathReliefLimit:   180.0,
				TransferSupportFloor: 0.14,
			},
		},
	}
}

func (s RiverRouteSettings) Validate() error {
	if len(s.Modes) == 0 {
		return fmt.Errorf("at least one river route mode is required")
	}
	names := map[string]struct{}{}
	for _, mode := range s.Modes {
		if mode.Name == "" {
			return fmt.Errorf("river route mode name is required")
		}
		if _, exists := names[mode.Name]; exists {
			return fmt.Errorf("duplicate river route mode %q", mode.Name)
		}
		names[mode.Name] = struct{}{}
		if mode.MinChannelStrength < 0 || mode.MinRunoff < 0 || mode.MinNavigability < 0 {
			return fmt.Errorf("river route mode %q has invalid thresholds", mode.Name)
		}
	}
	if s.DefaultMode == "" {
		s.DefaultMode = s.Modes[0].Name
	}
	if _, ok := s.ModeByName(s.DefaultMode); !ok {
		return fmt.Errorf("default river route mode %q not found", s.DefaultMode)
	}
	return nil
}

func (s RiverRouteSettings) ModeByName(name string) (RiverRouteModeSettings, bool) {
	for _, mode := range s.Modes {
		if mode.Name == name {
			return mode, true
		}
	}
	return RiverRouteModeSettings{}, false
}

func LoadRiverRouteSettings(path string) (RiverRouteSettings, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return RiverRouteSettings{}, err
	}
	var settings RiverRouteSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return RiverRouteSettings{}, err
	}
	if err := settings.Validate(); err != nil {
		return RiverRouteSettings{}, err
	}
	return settings, nil
}
