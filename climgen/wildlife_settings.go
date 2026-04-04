package climgen

import (
	"encoding/json"
	"fmt"
	"os"
)

const WildlifeProductivitySchemaVersion = "wildlife-productivity/v1"

type WildlifeProductivitySettings struct {
	SchemaVersion string `json:"schemaVersion"`

	GameMultiplier        float64 `json:"gameMultiplier"`
	GrazingMultiplier     float64 `json:"grazingMultiplier"`
	ForestGameMultiplier  float64 `json:"forestGameMultiplier"`
	WetlandGameMultiplier float64 `json:"wetlandGameMultiplier"`
	PeltMultiplier        float64 `json:"peltMultiplier"`
	TimberMultiplier      float64 `json:"timberMultiplier"`

	FurredAnimalsPresent bool `json:"furredAnimalsPresent"`
	TimberPresent        bool `json:"timberPresent"`

	GrazingPrimaryBias     float64 `json:"grazingPrimaryBias"`
	ForestGamePrimaryBias  float64 `json:"forestGamePrimaryBias"`
	WetlandGamePrimaryBias float64 `json:"wetlandGamePrimaryBias"`
	PeltPrimaryBias        float64 `json:"peltPrimaryBias"`
	TimberPrimaryBias      float64 `json:"timberPrimaryBias"`
}

func DefaultWildlifeProductivitySettings() WildlifeProductivitySettings {
	return WildlifeProductivitySettings{
		SchemaVersion: WildlifeProductivitySchemaVersion,

		GameMultiplier:        1.00,
		GrazingMultiplier:     1.00,
		ForestGameMultiplier:  1.00,
		WetlandGameMultiplier: 1.00,
		PeltMultiplier:        0.90,
		TimberMultiplier:      1.00,

		FurredAnimalsPresent: true,
		TimberPresent:        true,

		GrazingPrimaryBias:     0.00,
		ForestGamePrimaryBias:  0.00,
		WetlandGamePrimaryBias: 0.00,
		PeltPrimaryBias:        0.02,
		TimberPrimaryBias:      0.02,
	}
}

func LoadWildlifeProductivitySettings(path string) (WildlifeProductivitySettings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return WildlifeProductivitySettings{}, err
	}
	var settings WildlifeProductivitySettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return WildlifeProductivitySettings{}, fmt.Errorf("decode wildlife productivity settings: %w", err)
	}
	if err := ValidateWildlifeProductivitySettings(settings); err != nil {
		return WildlifeProductivitySettings{}, err
	}
	return settings, nil
}

func ValidateWildlifeProductivitySettings(settings WildlifeProductivitySettings) error {
	if settings.SchemaVersion == "" {
		return fmt.Errorf("wildlife productivity schemaVersion is required")
	}
	if settings.SchemaVersion != WildlifeProductivitySchemaVersion {
		return fmt.Errorf("unsupported wildlife productivity schemaVersion %q", settings.SchemaVersion)
	}
	for name, value := range map[string]float64{
		"gameMultiplier":        settings.GameMultiplier,
		"grazingMultiplier":     settings.GrazingMultiplier,
		"forestGameMultiplier":  settings.ForestGameMultiplier,
		"wetlandGameMultiplier": settings.WetlandGameMultiplier,
		"peltMultiplier":        settings.PeltMultiplier,
		"timberMultiplier":      settings.TimberMultiplier,
	} {
		if value < 0 {
			return fmt.Errorf("%s cannot be negative", name)
		}
	}
	return nil
}
