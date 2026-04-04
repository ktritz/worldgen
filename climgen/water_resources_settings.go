package climgen

import (
	"encoding/json"
	"fmt"
	"os"
)

const WaterResourceSchemaVersion = "water-resources/v1"

type WaterResourceSettings struct {
	SchemaVersion string `json:"schemaVersion"`

	SurfaceReliabilityMultiplier   float64 `json:"surfaceReliabilityMultiplier"`
	SeasonalAvailabilityMultiplier float64 `json:"seasonalAvailabilityMultiplier"`
	GroundwaterMultiplier          float64 `json:"groundwaterMultiplier"`
	LakeAccessMultiplier           float64 `json:"lakeAccessMultiplier"`

	SurfacePrimaryBias     float64 `json:"surfacePrimaryBias"`
	SeasonalPrimaryBias    float64 `json:"seasonalPrimaryBias"`
	GroundwaterPrimaryBias float64 `json:"groundwaterPrimaryBias"`
	LakePrimaryBias        float64 `json:"lakePrimaryBias"`
}

func DefaultWaterResourceSettings() WaterResourceSettings {
	return WaterResourceSettings{
		SchemaVersion: WaterResourceSchemaVersion,

		SurfaceReliabilityMultiplier:   1.00,
		SeasonalAvailabilityMultiplier: 0.88,
		GroundwaterMultiplier:          0.95,
		LakeAccessMultiplier:           0.82,

		SurfacePrimaryBias:     0.02,
		SeasonalPrimaryBias:    0.00,
		GroundwaterPrimaryBias: 0.00,
		LakePrimaryBias:        0.02,
	}
}

func LoadWaterResourceSettings(path string) (WaterResourceSettings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return WaterResourceSettings{}, err
	}
	var settings WaterResourceSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return WaterResourceSettings{}, fmt.Errorf("decode water resource settings: %w", err)
	}
	if err := ValidateWaterResourceSettings(settings); err != nil {
		return WaterResourceSettings{}, err
	}
	return settings, nil
}

func ValidateWaterResourceSettings(settings WaterResourceSettings) error {
	if settings.SchemaVersion == "" {
		return fmt.Errorf("water resource schemaVersion is required")
	}
	if settings.SchemaVersion != WaterResourceSchemaVersion {
		return fmt.Errorf("unsupported water resource schemaVersion %q", settings.SchemaVersion)
	}
	values := map[string]float64{
		"surfaceReliabilityMultiplier":   settings.SurfaceReliabilityMultiplier,
		"seasonalAvailabilityMultiplier": settings.SeasonalAvailabilityMultiplier,
		"groundwaterMultiplier":          settings.GroundwaterMultiplier,
		"lakeAccessMultiplier":           settings.LakeAccessMultiplier,
	}
	for name, value := range values {
		if value < 0 {
			return fmt.Errorf("%s cannot be negative", name)
		}
	}
	return nil
}
