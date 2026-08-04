package climgen

import (
	"encoding/json"
	"fmt"
	"os"
)

const PopulationSupportSchemaVersion = "population-support/v2"

type PopulationSupportSettings struct {
	SchemaVersion string `json:"schemaVersion"`

	FoodMultiplier     float64 `json:"foodMultiplier"`
	WaterMultiplier    float64 `json:"waterMultiplier"`
	TradeMultiplier    float64 `json:"tradeMultiplier"`
	ResourceMultiplier float64 `json:"resourceMultiplier"`
	UrbanMultiplier    float64 `json:"urbanMultiplier"`
	CatchmentHops      int     `json:"catchmentHops"`
	CatchmentBlend     float64 `json:"catchmentBlend"`

	SparseThreshold     float64 `json:"sparseThreshold"`
	RuralThreshold      float64 `json:"ruralThreshold"`
	DenseRuralThreshold float64 `json:"denseRuralThreshold"`
	UrbanThreshold      float64 `json:"urbanThreshold"`
}

func DefaultPopulationSupportSettings() PopulationSupportSettings {
	return PopulationSupportSettings{
		SchemaVersion: PopulationSupportSchemaVersion,

		FoodMultiplier:     1.00,
		WaterMultiplier:    1.00,
		TradeMultiplier:    1.00,
		ResourceMultiplier: 0.96,
		UrbanMultiplier:    1.08,
		CatchmentHops:      1,
		CatchmentBlend:     0.0,

		SparseThreshold:     0.18,
		RuralThreshold:      0.34,
		DenseRuralThreshold: 0.52,
		UrbanThreshold:      0.56,
	}
}

func LoadPopulationSupportSettings(path string) (PopulationSupportSettings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PopulationSupportSettings{}, err
	}
	var settings PopulationSupportSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return PopulationSupportSettings{}, fmt.Errorf("decode population support settings: %w", err)
	}
	if err := ValidatePopulationSupportSettings(settings); err != nil {
		return PopulationSupportSettings{}, err
	}
	return settings, nil
}

func ValidatePopulationSupportSettings(settings PopulationSupportSettings) error {
	if settings.SchemaVersion == "" {
		return fmt.Errorf("population support schemaVersion is required")
	}
	if settings.SchemaVersion != PopulationSupportSchemaVersion {
		return fmt.Errorf("unsupported population support schemaVersion %q", settings.SchemaVersion)
	}
	for name, value := range map[string]float64{
		"foodMultiplier":      settings.FoodMultiplier,
		"waterMultiplier":     settings.WaterMultiplier,
		"tradeMultiplier":     settings.TradeMultiplier,
		"resourceMultiplier":  settings.ResourceMultiplier,
		"urbanMultiplier":     settings.UrbanMultiplier,
		"catchmentBlend":      settings.CatchmentBlend,
		"sparseThreshold":     settings.SparseThreshold,
		"ruralThreshold":      settings.RuralThreshold,
		"denseRuralThreshold": settings.DenseRuralThreshold,
		"urbanThreshold":      settings.UrbanThreshold,
	} {
		if value < 0 {
			return fmt.Errorf("%s cannot be negative", name)
		}
	}
	if settings.RuralThreshold <= settings.SparseThreshold {
		return fmt.Errorf("ruralThreshold must exceed sparseThreshold")
	}
	if settings.DenseRuralThreshold <= settings.RuralThreshold {
		return fmt.Errorf("denseRuralThreshold must exceed ruralThreshold")
	}
	if settings.UrbanThreshold <= settings.RuralThreshold {
		return fmt.Errorf("urbanThreshold must exceed ruralThreshold")
	}
	if settings.CatchmentHops < 0 {
		return fmt.Errorf("catchmentHops cannot be negative")
	}
	if settings.CatchmentBlend < 0 || settings.CatchmentBlend > 1 {
		return fmt.Errorf("catchmentBlend must be between 0 and 1")
	}
	return nil
}
