package climgen

import (
	"encoding/json"
	"fmt"
	"os"
)

const ResourceAbundanceSchemaVersion = "resource-abundance/v1"

type ResourceAbundanceSettings struct {
	SchemaVersion string `json:"schemaVersion"`

	IronAffinityMultiplier       float64 `json:"ironAffinityMultiplier"`
	CopperAffinityMultiplier     float64 `json:"copperAffinityMultiplier"`
	LeadSilverAffinityMultiplier float64 `json:"leadSilverAffinityMultiplier"`
	GoldAffinityMultiplier       float64 `json:"goldAffinityMultiplier"`
	GemAffinityMultiplier        float64 `json:"gemAffinityMultiplier"`
	PlacerAffinityMultiplier     float64 `json:"placerAffinityMultiplier"`
	CoalAffinityMultiplier       float64 `json:"coalAffinityMultiplier"`
	OilGasAffinityMultiplier     float64 `json:"oilGasAffinityMultiplier"`
	EvaporiteAffinityMultiplier  float64 `json:"evaporiteAffinityMultiplier"`
	ClayAffinityMultiplier       float64 `json:"clayAffinityMultiplier"`
	StoneAffinityMultiplier      float64 `json:"stoneAffinityMultiplier"`

	IronPrimaryBias       float64 `json:"ironPrimaryBias"`
	CopperPrimaryBias     float64 `json:"copperPrimaryBias"`
	LeadSilverPrimaryBias float64 `json:"leadSilverPrimaryBias"`
	GoldPrimaryBias       float64 `json:"goldPrimaryBias"`
	GemPrimaryBias        float64 `json:"gemPrimaryBias"`
	PlacerPrimaryBias     float64 `json:"placerPrimaryBias"`
	CoalPrimaryBias       float64 `json:"coalPrimaryBias"`
	OilGasPrimaryBias     float64 `json:"oilGasPrimaryBias"`
	EvaporitePrimaryBias  float64 `json:"evaporitePrimaryBias"`
	ClayPrimaryBias       float64 `json:"clayPrimaryBias"`
	StonePrimaryBias      float64 `json:"stonePrimaryBias"`
}

func DefaultResourceAbundanceSettings() ResourceAbundanceSettings {
	return ResourceAbundanceSettings{
		SchemaVersion: ResourceAbundanceSchemaVersion,

		IronAffinityMultiplier:       1.00,
		CopperAffinityMultiplier:     0.92,
		LeadSilverAffinityMultiplier: 0.78,
		GoldAffinityMultiplier:       0.70,
		GemAffinityMultiplier:        0.62,
		PlacerAffinityMultiplier:     1.00,
		CoalAffinityMultiplier:       1.00,
		OilGasAffinityMultiplier:     1.00,
		EvaporiteAffinityMultiplier:  1.00,
		ClayAffinityMultiplier:       1.00,
		StoneAffinityMultiplier:      1.00,

		IronPrimaryBias:       0.00,
		CopperPrimaryBias:     0.00,
		LeadSilverPrimaryBias: 0.02,
		GoldPrimaryBias:       0.04,
		GemPrimaryBias:        0.03,
		PlacerPrimaryBias:     0.00,
		CoalPrimaryBias:       0.00,
		OilGasPrimaryBias:     0.00,
		EvaporitePrimaryBias:  0.00,
		ClayPrimaryBias:       0.00,
		StonePrimaryBias:      0.00,
	}
}

func LoadResourceAbundanceSettings(path string) (ResourceAbundanceSettings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ResourceAbundanceSettings{}, err
	}
	var settings ResourceAbundanceSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return ResourceAbundanceSettings{}, fmt.Errorf("decode resource abundance settings: %w", err)
	}
	if err := ValidateResourceAbundanceSettings(settings); err != nil {
		return ResourceAbundanceSettings{}, err
	}
	return settings, nil
}

func ValidateResourceAbundanceSettings(settings ResourceAbundanceSettings) error {
	if settings.SchemaVersion == "" {
		return fmt.Errorf("resource abundance schemaVersion is required")
	}
	if settings.SchemaVersion != ResourceAbundanceSchemaVersion {
		return fmt.Errorf("unsupported resource abundance schemaVersion %q", settings.SchemaVersion)
	}
	values := map[string]float64{
		"ironAffinityMultiplier":       settings.IronAffinityMultiplier,
		"copperAffinityMultiplier":     settings.CopperAffinityMultiplier,
		"leadSilverAffinityMultiplier": settings.LeadSilverAffinityMultiplier,
		"goldAffinityMultiplier":       settings.GoldAffinityMultiplier,
		"gemAffinityMultiplier":        settings.GemAffinityMultiplier,
		"placerAffinityMultiplier":     settings.PlacerAffinityMultiplier,
		"coalAffinityMultiplier":       settings.CoalAffinityMultiplier,
		"oilGasAffinityMultiplier":     settings.OilGasAffinityMultiplier,
		"evaporiteAffinityMultiplier":  settings.EvaporiteAffinityMultiplier,
		"clayAffinityMultiplier":       settings.ClayAffinityMultiplier,
		"stoneAffinityMultiplier":      settings.StoneAffinityMultiplier,
	}
	for name, value := range values {
		if value < 0 {
			return fmt.Errorf("%s cannot be negative", name)
		}
	}
	return nil
}
