package climgen

import (
	"encoding/json"
	"fmt"
	"os"

	worldgen "worldgen"
)

type MaritimeVesselSettings struct {
	Name                  string  `json:"name"`
	TechLevel             string  `json:"techLevel"`
	Propulsion            string  `json:"propulsion"`
	RouteClass            string  `json:"routeClass"`
	PayloadCapacity       float64 `json:"payloadCapacity"`
	DailyRange            float64 `json:"dailyRange"`
	LongHaulTolerance     float64 `json:"longHaulTolerance"`
	RiverCapability       float64 `json:"riverCapability"`
	CoastalCapability     float64 `json:"coastalCapability"`
	OpenOceanCapability   float64 `json:"openOceanCapability"`
	MaxCoastalLeg         float64 `json:"maxCoastalLeg"`
	MaxOpenWaterLeg       float64 `json:"maxOpenWaterLeg"`
	StopoverNeed          float64 `json:"stopoverNeed"`
	HarborDependence      float64 `json:"harborDependence"`
	BeachingCapability    float64 `json:"beachingCapability"`
	ShallowDraft          float64 `json:"shallowDraft"`
	CurrentAssist         float64 `json:"currentAssist"`
	AdverseCurrentPenalty float64 `json:"adverseCurrentPenalty"`
	WindAssist            float64 `json:"windAssist"`
	UpwindPenalty         float64 `json:"upwindPenalty"`
	StormTolerance        float64 `json:"stormTolerance"`
	SeasonalityTolerance  float64 `json:"seasonalityTolerance"`
	RepairIndependence    float64 `json:"repairIndependence"`
	CrewEfficiency        float64 `json:"crewEfficiency"`
}

type MaritimeRouteSettings struct {
	DefaultVessel string                   `json:"defaultVessel"`
	Vessels       []MaritimeVesselSettings `json:"vessels"`
}

func DefaultMaritimeRouteSettings() MaritimeRouteSettings {
	settings, err := loadMaritimeRouteSettingsData(worldgen.EmbeddedMaritimeRouteSettings())
	if err == nil {
		return settings
	}
	return MaritimeRouteSettings{
		DefaultVessel: "coastal-sloop",
		Vessels: []MaritimeVesselSettings{
			{
				Name:                "coastal-sloop",
				TechLevel:           "medieval",
				Propulsion:          "sail",
				RouteClass:          "coastal",
				PayloadCapacity:     0.46,
				DailyRange:          0.72,
				LongHaulTolerance:   0.46,
				CoastalCapability:   0.88,
				OpenOceanCapability: 0.24,
			},
		},
	}
}

func (s MaritimeRouteSettings) Validate() error {
	if len(s.Vessels) == 0 {
		return fmt.Errorf("at least one maritime vessel is required")
	}
	names := map[string]struct{}{}
	for _, vessel := range s.Vessels {
		if vessel.Name == "" {
			return fmt.Errorf("maritime vessel name is required")
		}
		if _, exists := names[vessel.Name]; exists {
			return fmt.Errorf("duplicate maritime vessel %q", vessel.Name)
		}
		names[vessel.Name] = struct{}{}
		if vessel.TechLevel == "" {
			return fmt.Errorf("maritime vessel %q techLevel is required", vessel.Name)
		}
		if vessel.RouteClass == "" {
			return fmt.Errorf("maritime vessel %q routeClass is required", vessel.Name)
		}
		if vessel.Propulsion == "" {
			return fmt.Errorf("maritime vessel %q propulsion is required", vessel.Name)
		}
		if vessel.PayloadCapacity < 0 || vessel.DailyRange < 0 || vessel.LongHaulTolerance < 0 {
			return fmt.Errorf("maritime vessel %q has invalid capacity/range values", vessel.Name)
		}
	}
	if s.DefaultVessel == "" {
		s.DefaultVessel = s.Vessels[0].Name
	}
	if _, ok := s.VesselByName(s.DefaultVessel); !ok {
		return fmt.Errorf("default maritime vessel %q not found", s.DefaultVessel)
	}
	return nil
}

func (s MaritimeRouteSettings) VesselByName(name string) (MaritimeVesselSettings, bool) {
	for _, vessel := range s.Vessels {
		if vessel.Name == name {
			return vessel, true
		}
	}
	return MaritimeVesselSettings{}, false
}

func LoadMaritimeRouteSettings(path string) (MaritimeRouteSettings, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return MaritimeRouteSettings{}, err
	}
	return loadMaritimeRouteSettingsData(raw)
}

func loadMaritimeRouteSettingsData(data []byte) (MaritimeRouteSettings, error) {
	var settings MaritimeRouteSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return MaritimeRouteSettings{}, err
	}
	if err := settings.Validate(); err != nil {
		return MaritimeRouteSettings{}, err
	}
	return settings, nil
}
