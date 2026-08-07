package climgen

import (
	"encoding/json"
	"fmt"
	"os"

	worldgen "worldgen"
)

const OceanTradeSchemaVersion = "ocean-trade/v2"

type OceanTradeSettings struct {
	SchemaVersion string `json:"schemaVersion"`

	CandidatePortThreshold          float64 `json:"candidatePortThreshold"`
	CandidateSecondaryPortFloor     float64 `json:"candidateSecondaryPortFloor"`
	CandidatePhysicalDeepwaterFloor float64 `json:"candidatePhysicalDeepwaterFloor"`
	MinOpenOceanCapability          float64 `json:"minOpenOceanCapability"`
	MaxStopovers                    int     `json:"maxStopovers"`
	StopoverScoreFloor              float64 `json:"stopoverScoreFloor"`
	StopoverSpacingHops             int     `json:"stopoverSpacingHops"`
	MaxCandidatePortsPerCiv         int     `json:"maxCandidatePortsPerCivilization"`
	MaxPartnersPerPort              int     `json:"maxPartnersPerPort"`
	MaxPartnersPerCivilization      int     `json:"maxPartnersPerCivilization"`
	BaseLegCost                     float64 `json:"baseLegCost"`
	LegScale                        float64 `json:"legScale"`
	MaxRouteCost                    float64 `json:"maxRouteCost"`
	RouteBudgetBaseFactor           float64 `json:"routeBudgetBaseFactor"`
	RouteBudgetLongHaulWeight       float64 `json:"routeBudgetLongHaulWeight"`
	RouteBudgetOpenOceanWeight      float64 `json:"routeBudgetOpenOceanWeight"`
	RouteBudgetDailyRangeWeight     float64 `json:"routeBudgetDailyRangeWeight"`
	RouteBudgetStopoverWeight       float64 `json:"routeBudgetStopoverWeight"`
	RouteBudgetLegWeight            float64 `json:"routeBudgetLegWeight"`
	MinFlow                         float64 `json:"minFlow"`
	RegionalFlow                    float64 `json:"regionalFlow"`
	PrimaryFlow                     float64 `json:"primaryFlow"`
}

func DefaultOceanTradeSettings() OceanTradeSettings {
	settings, err := loadOceanTradeSettingsData(worldgen.EmbeddedOceanTradeSettings())
	if err == nil {
		return settings
	}
	return OceanTradeSettings{
		SchemaVersion:                   OceanTradeSchemaVersion,
		CandidatePortThreshold:          0.48,
		CandidateSecondaryPortFloor:     0.44,
		CandidatePhysicalDeepwaterFloor: 0.16,
		MinOpenOceanCapability:          0.35,
		MaxStopovers:                    56,
		StopoverScoreFloor:              0.36,
		StopoverSpacingHops:             4,
		MaxCandidatePortsPerCiv:         2,
		MaxPartnersPerPort:              3,
		MaxPartnersPerCivilization:      3,
		BaseLegCost:                     8.0,
		LegScale:                        220.0,
		MaxRouteCost:                    42.0,
		RouteBudgetBaseFactor:           0.45,
		RouteBudgetLongHaulWeight:       1.10,
		RouteBudgetOpenOceanWeight:      1.25,
		RouteBudgetDailyRangeWeight:     0.35,
		RouteBudgetStopoverWeight:       0.35,
		RouteBudgetLegWeight:            18.0,
		MinFlow:                         0.035,
		RegionalFlow:                    0.09,
		PrimaryFlow:                     0.16,
	}
}

func LoadOceanTradeSettings(path string) (OceanTradeSettings, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return OceanTradeSettings{}, err
	}
	return loadOceanTradeSettingsData(raw)
}

func loadOceanTradeSettingsData(data []byte) (OceanTradeSettings, error) {
	var settings OceanTradeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return OceanTradeSettings{}, fmt.Errorf("decode ocean trade settings: %w", err)
	}
	if err := settings.Validate(); err != nil {
		return OceanTradeSettings{}, err
	}
	return settings, nil
}

func (s OceanTradeSettings) Validate() error {
	if s.SchemaVersion == "" {
		return fmt.Errorf("ocean trade schemaVersion is required")
	}
	if s.SchemaVersion != OceanTradeSchemaVersion {
		return fmt.Errorf("unsupported ocean trade schemaVersion %q", s.SchemaVersion)
	}
	values := map[string]float64{
		"candidatePortThreshold":          s.CandidatePortThreshold,
		"candidateSecondaryPortFloor":     s.CandidateSecondaryPortFloor,
		"candidatePhysicalDeepwaterFloor": s.CandidatePhysicalDeepwaterFloor,
		"minOpenOceanCapability":          s.MinOpenOceanCapability,
		"stopoverScoreFloor":              s.StopoverScoreFloor,
		"baseLegCost":                     s.BaseLegCost,
		"legScale":                        s.LegScale,
		"maxRouteCost":                    s.MaxRouteCost,
		"routeBudgetBaseFactor":           s.RouteBudgetBaseFactor,
		"routeBudgetLongHaulWeight":       s.RouteBudgetLongHaulWeight,
		"routeBudgetOpenOceanWeight":      s.RouteBudgetOpenOceanWeight,
		"routeBudgetDailyRangeWeight":     s.RouteBudgetDailyRangeWeight,
		"routeBudgetStopoverWeight":       s.RouteBudgetStopoverWeight,
		"routeBudgetLegWeight":            s.RouteBudgetLegWeight,
		"minFlow":                         s.MinFlow,
		"regionalFlow":                    s.RegionalFlow,
		"primaryFlow":                     s.PrimaryFlow,
	}
	for name, value := range values {
		if value < 0 {
			return fmt.Errorf("%s cannot be negative", name)
		}
	}
	if s.MaxStopovers < 0 || s.StopoverSpacingHops < 0 || s.MaxCandidatePortsPerCiv < 1 || s.MaxPartnersPerPort < 1 || s.MaxPartnersPerCivilization < 1 {
		return fmt.Errorf("ocean trade counts and partner limits must be valid")
	}
	return nil
}
