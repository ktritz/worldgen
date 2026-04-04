package climgen

import (
	"encoding/json"
	"fmt"
	"os"
)

const AgricultureProductivitySchemaVersion = "agriculture-productivity/v1"

type AgricultureProductivitySettings struct {
	SchemaVersion string `json:"schemaVersion"`

	CropMultiplier       float64 `json:"cropMultiplier"`
	PastureMultiplier    float64 `json:"pastureMultiplier"`
	IrrigationMultiplier float64 `json:"irrigationMultiplier"`
	FloodplainMultiplier float64 `json:"floodplainMultiplier"`

	DryFarmingThreshold   float64 `json:"dryFarmingThreshold"`
	MixedFarmingThreshold float64 `json:"mixedFarmingThreshold"`
	IntensiveThreshold    float64 `json:"intensiveThreshold"`
	PastoralThreshold     float64 `json:"pastoralThreshold"`
	FloodplainThreshold   float64 `json:"floodplainThreshold"`
}

func DefaultAgricultureProductivitySettings() AgricultureProductivitySettings {
	return AgricultureProductivitySettings{
		SchemaVersion: AgricultureProductivitySchemaVersion,

		CropMultiplier:       0.90,
		PastureMultiplier:    1.12,
		IrrigationMultiplier: 1.00,
		FloodplainMultiplier: 1.00,

		DryFarmingThreshold:   0.48,
		MixedFarmingThreshold: 0.58,
		IntensiveThreshold:    0.74,
		PastoralThreshold:     0.42,
		FloodplainThreshold:   0.58,
	}
}

func LoadAgricultureProductivitySettings(path string) (AgricultureProductivitySettings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AgricultureProductivitySettings{}, err
	}
	var settings AgricultureProductivitySettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return AgricultureProductivitySettings{}, fmt.Errorf("decode agriculture productivity settings: %w", err)
	}
	if err := ValidateAgricultureProductivitySettings(settings); err != nil {
		return AgricultureProductivitySettings{}, err
	}
	return settings, nil
}

func ValidateAgricultureProductivitySettings(settings AgricultureProductivitySettings) error {
	if settings.SchemaVersion == "" {
		return fmt.Errorf("agriculture productivity schemaVersion is required")
	}
	if settings.SchemaVersion != AgricultureProductivitySchemaVersion {
		return fmt.Errorf("unsupported agriculture productivity schemaVersion %q", settings.SchemaVersion)
	}
	for name, value := range map[string]float64{
		"cropMultiplier":        settings.CropMultiplier,
		"pastureMultiplier":     settings.PastureMultiplier,
		"irrigationMultiplier":  settings.IrrigationMultiplier,
		"floodplainMultiplier":  settings.FloodplainMultiplier,
		"dryFarmingThreshold":   settings.DryFarmingThreshold,
		"mixedFarmingThreshold": settings.MixedFarmingThreshold,
		"intensiveThreshold":    settings.IntensiveThreshold,
		"pastoralThreshold":     settings.PastoralThreshold,
		"floodplainThreshold":   settings.FloodplainThreshold,
	} {
		if value < 0 {
			return fmt.Errorf("%s cannot be negative", name)
		}
	}
	return nil
}
