package climgen

import (
	"encoding/json"
	"fmt"
	"os"
)

const CoastalResourceSchemaVersion = "coastal-resources/v1"

type CoastalResourceSettings struct {
	SchemaVersion string `json:"schemaVersion"`

	OpenFisheryMultiplier      float64 `json:"openFisheryMultiplier"`
	EstuarineFisheryMultiplier float64 `json:"estuarineFisheryMultiplier"`
	ShellfishMultiplier        float64 `json:"shellfishMultiplier"`
	SaltworksMultiplier        float64 `json:"saltworksMultiplier"`

	OpenFisheryPrimaryBias      float64 `json:"openFisheryPrimaryBias"`
	EstuarineFisheryPrimaryBias float64 `json:"estuarineFisheryPrimaryBias"`
	ShellfishPrimaryBias        float64 `json:"shellfishPrimaryBias"`
	SaltworksPrimaryBias        float64 `json:"saltworksPrimaryBias"`
}

func DefaultCoastalResourceSettings() CoastalResourceSettings {
	return CoastalResourceSettings{
		SchemaVersion: CoastalResourceSchemaVersion,

		OpenFisheryMultiplier:      1.05,
		EstuarineFisheryMultiplier: 1.00,
		ShellfishMultiplier:        1.05,
		SaltworksMultiplier:        0.90,

		OpenFisheryPrimaryBias:      0.00,
		EstuarineFisheryPrimaryBias: 0.01,
		ShellfishPrimaryBias:        0.00,
		SaltworksPrimaryBias:        0.00,
	}
}

func LoadCoastalResourceSettings(path string) (CoastalResourceSettings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CoastalResourceSettings{}, err
	}
	var settings CoastalResourceSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return CoastalResourceSettings{}, fmt.Errorf("decode coastal resource settings: %w", err)
	}
	if err := ValidateCoastalResourceSettings(settings); err != nil {
		return CoastalResourceSettings{}, err
	}
	return settings, nil
}

func ValidateCoastalResourceSettings(settings CoastalResourceSettings) error {
	if settings.SchemaVersion == "" {
		return fmt.Errorf("coastal resource schemaVersion is required")
	}
	if settings.SchemaVersion != CoastalResourceSchemaVersion {
		return fmt.Errorf("unsupported coastal resource schemaVersion %q", settings.SchemaVersion)
	}
	values := map[string]float64{
		"openFisheryMultiplier":      settings.OpenFisheryMultiplier,
		"estuarineFisheryMultiplier": settings.EstuarineFisheryMultiplier,
		"shellfishMultiplier":        settings.ShellfishMultiplier,
		"saltworksMultiplier":        settings.SaltworksMultiplier,
	}
	for name, value := range values {
		if value < 0 {
			return fmt.Errorf("%s cannot be negative", name)
		}
	}
	return nil
}
