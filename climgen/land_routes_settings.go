package climgen

import (
	"encoding/json"
	"fmt"
	"os"
)

type LandRouteModeSettings struct {
	Name                   string  `json:"name"`
	PayloadCapacity        float64 `json:"payloadCapacity"`
	DailyRange             float64 `json:"dailyRange"`
	LongHaulTolerance      float64 `json:"longHaulTolerance"`
	InterCivilizationFlow  float64 `json:"interCivilizationFlow"`
	InternalFlow           float64 `json:"internalFlow"`
	FeederFlow             float64 `json:"feederFlow"`
	FeederReach            float64 `json:"feederReach"`
	WaterNeed              float64 `json:"waterNeed"`
	ForageNeed             float64 `json:"forageNeed"`
	RoadDependence         float64 `json:"roadDependence"`
	BridgeDependence       float64 `json:"bridgeDependence"`
	FordTolerance          float64 `json:"fordTolerance"`
	SlopeLimit             float64 `json:"slopeLimit"`
	MarshPassability       float64 `json:"marshPassability"`
	SnowPassability        float64 `json:"snowPassability"`
	BanditVulnerability    float64 `json:"banditVulnerability"`
	EscortNeed             float64 `json:"escortNeed"`
	SeasonalityTolerance   float64 `json:"seasonalityTolerance"`
	SpeedMultiplier        float64 `json:"speedMultiplier"`
	EnduranceMultiplier    float64 `json:"enduranceMultiplier"`
	HeatTolerance          float64 `json:"heatTolerance"`
	ColdTolerance          float64 `json:"coldTolerance"`
	BaseCostMultiplier     float64 `json:"baseCostMultiplier"`
	ReliefMultiplier       float64 `json:"reliefMultiplier"`
	RockinessMultiplier    float64 `json:"rockinessMultiplier"`
	WetlandMultiplier      float64 `json:"wetlandMultiplier"`
	AridityMultiplier      float64 `json:"aridityMultiplier"`
	ForestMultiplier       float64 `json:"forestMultiplier"`
	IceMultiplier          float64 `json:"iceMultiplier"`
	RiverBonusMultiplier   float64 `json:"riverBonusMultiplier"`
	CoastalBonusMultiplier float64 `json:"coastalBonusMultiplier"`
	FloodRiskMultiplier    float64 `json:"floodRiskMultiplier"`
	DesertRiskMultiplier   float64 `json:"desertRiskMultiplier"`
	MarshRiskMultiplier    float64 `json:"marshRiskMultiplier"`
	WildlifeRiskMultiplier float64 `json:"wildlifeRiskMultiplier"`
	ColdRiskMultiplier     float64 `json:"coldRiskMultiplier"`
	ForageSupportWeight    float64 `json:"forageSupportWeight"`
	WaterSupportWeight     float64 `json:"waterSupportWeight"`
	MarshAdaptation        float64 `json:"marshAdaptation"`
	AridAdaptation         float64 `json:"aridAdaptation"`
}

type LandRouteSettings struct {
	DefaultMode string                  `json:"defaultMode"`
	Modes       []LandRouteModeSettings `json:"modes"`
}

func DefaultLandRouteSettings() LandRouteSettings {
	return LandRouteSettings{
		DefaultMode: "pack-mule",
		Modes: []LandRouteModeSettings{
			{
				Name:                   "porter",
				PayloadCapacity:        0.22,
				DailyRange:             0.78,
				LongHaulTolerance:      0.18,
				InterCivilizationFlow:  0.06,
				InternalFlow:           0.48,
				FeederFlow:             0.95,
				FeederReach:            0.42,
				WaterNeed:              0.48,
				ForageNeed:             0.05,
				RoadDependence:         0.05,
				BridgeDependence:       0.10,
				FordTolerance:          0.92,
				SlopeLimit:             0.96,
				MarshPassability:       0.45,
				SnowPassability:        0.55,
				BanditVulnerability:    0.72,
				EscortNeed:             0.18,
				SeasonalityTolerance:   0.55,
				SpeedMultiplier:        0.74,
				EnduranceMultiplier:    0.82,
				HeatTolerance:          0.62,
				ColdTolerance:          0.52,
				BaseCostMultiplier:     0.94,
				ReliefMultiplier:       0.72,
				RockinessMultiplier:    0.48,
				WetlandMultiplier:      0.68,
				AridityMultiplier:      0.84,
				ForestMultiplier:       0.22,
				IceMultiplier:          0.92,
				RiverBonusMultiplier:   0.90,
				CoastalBonusMultiplier: 0.45,
				FloodRiskMultiplier:    0.86,
				DesertRiskMultiplier:   0.92,
				MarshRiskMultiplier:    0.72,
				WildlifeRiskMultiplier: 0.82,
				ColdRiskMultiplier:     0.94,
				ForageSupportWeight:    0.15,
				WaterSupportWeight:     0.98,
				MarshAdaptation:        0.18,
				AridAdaptation:         0.08,
			},
			{
				Name:                   "pack-mule",
				PayloadCapacity:        0.58,
				DailyRange:             0.98,
				LongHaulTolerance:      0.62,
				InterCivilizationFlow:  0.92,
				InternalFlow:           1.00,
				FeederFlow:             0.34,
				FeederReach:            0.34,
				WaterNeed:              0.56,
				ForageNeed:             0.72,
				RoadDependence:         0.16,
				BridgeDependence:       0.24,
				FordTolerance:          0.76,
				SlopeLimit:             0.90,
				MarshPassability:       0.28,
				SnowPassability:        0.54,
				BanditVulnerability:    0.56,
				EscortNeed:             0.34,
				SeasonalityTolerance:   0.60,
				SpeedMultiplier:        0.96,
				EnduranceMultiplier:    1.04,
				HeatTolerance:          0.52,
				ColdTolerance:          0.62,
				BaseCostMultiplier:     1.00,
				ReliefMultiplier:       0.86,
				RockinessMultiplier:    0.64,
				WetlandMultiplier:      0.82,
				AridityMultiplier:      0.72,
				ForestMultiplier:       0.38,
				IceMultiplier:          0.88,
				RiverBonusMultiplier:   1.00,
				CoastalBonusMultiplier: 0.58,
				FloodRiskMultiplier:    0.84,
				DesertRiskMultiplier:   0.80,
				MarshRiskMultiplier:    0.84,
				WildlifeRiskMultiplier: 0.70,
				ColdRiskMultiplier:     0.88,
				ForageSupportWeight:    0.62,
				WaterSupportWeight:     0.94,
				MarshAdaptation:        0.12,
				AridAdaptation:         0.18,
			},
			{
				Name:                   "horse-wagon",
				PayloadCapacity:        0.86,
				DailyRange:             1.00,
				LongHaulTolerance:      0.72,
				InterCivilizationFlow:  1.00,
				InternalFlow:           0.94,
				FeederFlow:             0.10,
				FeederReach:            0.20,
				WaterNeed:              0.58,
				ForageNeed:             0.66,
				RoadDependence:         0.76,
				BridgeDependence:       0.82,
				FordTolerance:          0.34,
				SlopeLimit:             0.48,
				MarshPassability:       0.12,
				SnowPassability:        0.28,
				BanditVulnerability:    0.74,
				EscortNeed:             0.64,
				SeasonalityTolerance:   0.48,
				SpeedMultiplier:        1.00,
				EnduranceMultiplier:    0.92,
				HeatTolerance:          0.52,
				ColdTolerance:          0.42,
				BaseCostMultiplier:     1.04,
				ReliefMultiplier:       1.22,
				RockinessMultiplier:    1.08,
				WetlandMultiplier:      1.22,
				AridityMultiplier:      0.78,
				ForestMultiplier:       0.82,
				IceMultiplier:          1.08,
				RiverBonusMultiplier:   1.04,
				CoastalBonusMultiplier: 0.70,
				FloodRiskMultiplier:    0.90,
				DesertRiskMultiplier:   0.85,
				MarshRiskMultiplier:    1.10,
				WildlifeRiskMultiplier: 0.60,
				ColdRiskMultiplier:     1.00,
				ForageSupportWeight:    0.58,
				WaterSupportWeight:     0.90,
				MarshAdaptation:        0.00,
				AridAdaptation:         0.10,
			},
			{
				Name:                   "ox-cart",
				PayloadCapacity:        1.05,
				DailyRange:             0.84,
				LongHaulTolerance:      0.64,
				InterCivilizationFlow:  0.92,
				InternalFlow:           0.98,
				FeederFlow:             0.08,
				FeederReach:            0.18,
				WaterNeed:              0.52,
				ForageNeed:             0.58,
				RoadDependence:         0.84,
				BridgeDependence:       0.88,
				FordTolerance:          0.26,
				SlopeLimit:             0.42,
				MarshPassability:       0.08,
				SnowPassability:        0.22,
				BanditVulnerability:    0.68,
				EscortNeed:             0.60,
				SeasonalityTolerance:   0.52,
				SpeedMultiplier:        0.84,
				EnduranceMultiplier:    1.08,
				HeatTolerance:          0.56,
				ColdTolerance:          0.50,
				BaseCostMultiplier:     1.06,
				ReliefMultiplier:       1.28,
				RockinessMultiplier:    1.12,
				WetlandMultiplier:      1.28,
				AridityMultiplier:      0.74,
				ForestMultiplier:       0.88,
				IceMultiplier:          1.06,
				RiverBonusMultiplier:   1.02,
				CoastalBonusMultiplier: 0.64,
				FloodRiskMultiplier:    0.88,
				DesertRiskMultiplier:   0.82,
				MarshRiskMultiplier:    1.14,
				WildlifeRiskMultiplier: 0.62,
				ColdRiskMultiplier:     0.98,
				ForageSupportWeight:    0.60,
				WaterSupportWeight:     0.88,
				MarshAdaptation:        0.00,
				AridAdaptation:         0.08,
			},
			{
				Name:                   "camel-caravan",
				PayloadCapacity:        0.80,
				DailyRange:             1.10,
				LongHaulTolerance:      0.92,
				InterCivilizationFlow:  0.98,
				InternalFlow:           0.90,
				FeederFlow:             0.18,
				FeederReach:            0.24,
				WaterNeed:              0.25,
				ForageNeed:             0.40,
				RoadDependence:         0.15,
				BridgeDependence:       0.25,
				FordTolerance:          0.55,
				SlopeLimit:             0.75,
				MarshPassability:       0.05,
				SnowPassability:        0.10,
				BanditVulnerability:    0.60,
				EscortNeed:             0.50,
				SeasonalityTolerance:   0.65,
				SpeedMultiplier:        1.00,
				EnduranceMultiplier:    1.20,
				HeatTolerance:          0.95,
				ColdTolerance:          0.20,
				BaseCostMultiplier:     1.00,
				ReliefMultiplier:       0.85,
				RockinessMultiplier:    0.60,
				WetlandMultiplier:      1.20,
				AridityMultiplier:      0.35,
				ForestMultiplier:       0.45,
				IceMultiplier:          1.10,
				RiverBonusMultiplier:   0.80,
				CoastalBonusMultiplier: 0.55,
				FloodRiskMultiplier:    0.75,
				DesertRiskMultiplier:   0.35,
				MarshRiskMultiplier:    1.25,
				WildlifeRiskMultiplier: 0.65,
				ColdRiskMultiplier:     1.10,
				ForageSupportWeight:    0.45,
				WaterSupportWeight:     0.70,
				MarshAdaptation:        0.00,
				AridAdaptation:         0.90,
			},
			{
				Name:                   "pack-lizard",
				PayloadCapacity:        0.70,
				DailyRange:             0.95,
				LongHaulTolerance:      0.52,
				InterCivilizationFlow:  0.70,
				InternalFlow:           0.92,
				FeederFlow:             0.42,
				FeederReach:            0.34,
				WaterNeed:              0.45,
				ForageNeed:             0.35,
				RoadDependence:         0.10,
				BridgeDependence:       0.20,
				FordTolerance:          0.85,
				SlopeLimit:             0.70,
				MarshPassability:       0.95,
				SnowPassability:        0.05,
				BanditVulnerability:    0.58,
				EscortNeed:             0.42,
				SeasonalityTolerance:   0.60,
				SpeedMultiplier:        0.92,
				EnduranceMultiplier:    1.05,
				HeatTolerance:          0.75,
				ColdTolerance:          0.15,
				BaseCostMultiplier:     1.00,
				ReliefMultiplier:       0.85,
				RockinessMultiplier:    0.60,
				WetlandMultiplier:      0.35,
				AridityMultiplier:      1.05,
				ForestMultiplier:       0.40,
				IceMultiplier:          1.20,
				RiverBonusMultiplier:   0.95,
				CoastalBonusMultiplier: 0.65,
				FloodRiskMultiplier:    0.65,
				DesertRiskMultiplier:   1.05,
				MarshRiskMultiplier:    0.35,
				WildlifeRiskMultiplier: 0.75,
				ColdRiskMultiplier:     1.15,
				ForageSupportWeight:    0.50,
				WaterSupportWeight:     0.90,
				MarshAdaptation:        0.95,
				AridAdaptation:         0.00,
			},
		},
	}
}

func (s LandRouteSettings) ModeByName(name string) (LandRouteModeSettings, bool) {
	for _, mode := range s.Modes {
		if mode.Name == name {
			return mode, true
		}
	}
	return LandRouteModeSettings{}, false
}

func ValidateLandRouteSettings(settings LandRouteSettings) error {
	if settings.DefaultMode == "" {
		return fmt.Errorf("defaultMode is required")
	}
	if len(settings.Modes) == 0 {
		return fmt.Errorf("at least one land route mode is required")
	}
	seen := map[string]struct{}{}
	for _, mode := range settings.Modes {
		if mode.Name == "" {
			return fmt.Errorf("land route mode name is required")
		}
		if _, ok := seen[mode.Name]; ok {
			return fmt.Errorf("duplicate land route mode %q", mode.Name)
		}
		seen[mode.Name] = struct{}{}
	}
	if _, ok := settings.ModeByName(settings.DefaultMode); !ok {
		return fmt.Errorf("defaultMode %q not found in modes", settings.DefaultMode)
	}
	return nil
}

func LoadLandRouteSettings(path string) (LandRouteSettings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LandRouteSettings{}, err
	}
	var settings LandRouteSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return LandRouteSettings{}, err
	}
	if err := ValidateLandRouteSettings(settings); err != nil {
		return LandRouteSettings{}, err
	}
	return settings, nil
}
