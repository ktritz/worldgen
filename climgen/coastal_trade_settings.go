package climgen

import (
	"encoding/json"
	"fmt"
	"os"

	worldgen "worldgen"
)

const CoastalTradeSchemaVersion = "coastal-trade/v1"

type CoastalTradeSettings struct {
	SchemaVersion string `json:"schemaVersion"`

	CandidatePortThreshold        float64 `json:"candidatePortThreshold"`
	CandidatePortSuitabilityFloor float64 `json:"candidatePortSuitabilityFloor"`
	CandidatePortFeatureFloor     float64 `json:"candidatePortFeatureFloor"`
	MaxPartnersPerPort            int     `json:"maxPartnersPerPort"`
	MaxPartnersPerCivilization    int     `json:"maxPartnersPerCivilization"`
	BaseLegCost                   float64 `json:"baseLegCost"`
	LegScale                      float64 `json:"legScale"`
	MaxRouteCost                  float64 `json:"maxRouteCost"`
	RouteBudgetBaseFactor         float64 `json:"routeBudgetBaseFactor"`
	RouteBudgetLongHaulWeight     float64 `json:"routeBudgetLongHaulWeight"`
	RouteBudgetCoastalWeight      float64 `json:"routeBudgetCoastalWeight"`
	RouteBudgetOpenOceanWeight    float64 `json:"routeBudgetOpenOceanWeight"`
	RouteBudgetDailyRangeWeight   float64 `json:"routeBudgetDailyRangeWeight"`
	RouteBudgetStopoverWeight     float64 `json:"routeBudgetStopoverWeight"`
	RouteBudgetLegWeight          float64 `json:"routeBudgetLegWeight"`
	MinFlow                       float64 `json:"minFlow"`
	RegionalFlow                  float64 `json:"regionalFlow"`
	PrimaryFlow                   float64 `json:"primaryFlow"`
}

func DefaultCoastalTradeSettings() CoastalTradeSettings {
	settings, err := loadCoastalTradeSettingsData(worldgen.EmbeddedCoastalTradeSettings())
	if err == nil {
		return settings
	}
	return CoastalTradeSettings{
		SchemaVersion:                 CoastalTradeSchemaVersion,
		CandidatePortThreshold:        0.42,
		CandidatePortSuitabilityFloor: 0.29,
		CandidatePortFeatureFloor:     0.22,
		MaxPartnersPerPort:            3,
		MaxPartnersPerCivilization:    3,
		BaseLegCost:                   4.0,
		LegScale:                      42.0,
		MaxRouteCost:                  18.0,
		RouteBudgetBaseFactor:         0.70,
		RouteBudgetLongHaulWeight:     0.85,
		RouteBudgetCoastalWeight:      0.20,
		RouteBudgetOpenOceanWeight:    0.55,
		RouteBudgetDailyRangeWeight:   0.20,
		RouteBudgetStopoverWeight:     0.15,
		RouteBudgetLegWeight:          8.0,
		MinFlow:                       0.04,
		RegionalFlow:                  0.10,
		PrimaryFlow:                   0.18,
	}
}

func LoadCoastalTradeSettings(path string) (CoastalTradeSettings, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return CoastalTradeSettings{}, err
	}
	return loadCoastalTradeSettingsData(raw)
}

func loadCoastalTradeSettingsData(data []byte) (CoastalTradeSettings, error) {
	var settings CoastalTradeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return CoastalTradeSettings{}, fmt.Errorf("decode coastal trade settings: %w", err)
	}
	if err := settings.Validate(); err != nil {
		return CoastalTradeSettings{}, err
	}
	return settings, nil
}

func (s CoastalTradeSettings) Validate() error {
	if s.SchemaVersion == "" {
		return fmt.Errorf("coastal trade schemaVersion is required")
	}
	if s.SchemaVersion != CoastalTradeSchemaVersion {
		return fmt.Errorf("unsupported coastal trade schemaVersion %q", s.SchemaVersion)
	}
	if s.CandidatePortThreshold < 0 || s.CandidatePortSuitabilityFloor < 0 || s.CandidatePortFeatureFloor < 0 || s.BaseLegCost < 0 || s.LegScale < 0 || s.MaxRouteCost < 0 {
		return fmt.Errorf("coastal trade thresholds cannot be negative")
	}
	if s.RouteBudgetBaseFactor < 0 || s.RouteBudgetLongHaulWeight < 0 || s.RouteBudgetCoastalWeight < 0 || s.RouteBudgetOpenOceanWeight < 0 || s.RouteBudgetDailyRangeWeight < 0 || s.RouteBudgetStopoverWeight < 0 || s.RouteBudgetLegWeight < 0 {
		return fmt.Errorf("coastal trade thresholds cannot be negative")
	}
	if s.MaxPartnersPerPort < 1 || s.MaxPartnersPerCivilization < 1 {
		return fmt.Errorf("coastal trade partner limits must be >= 1")
	}
	return nil
}
