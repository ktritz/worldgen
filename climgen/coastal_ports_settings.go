package climgen

import (
	"encoding/json"
	"fmt"
	"os"

	worldgen "worldgen"
)

const MaritimePortSchemaVersion = "maritime-ports/v2"

type MaritimePortSettings struct {
	SchemaVersion string `json:"schemaVersion"`

	HarborShelterWeight      float64 `json:"harborShelterWeight"`
	EstuaryWeight            float64 `json:"estuaryWeight"`
	RiverTransferWeight      float64 `json:"riverTransferWeight"`
	StopoverWeight           float64 `json:"stopoverWeight"`
	BeachingWeight           float64 `json:"beachingWeight"`
	ShallowDraftWeight       float64 `json:"shallowDraftWeight"`
	DeepDraftHarborWeight    float64 `json:"deepDraftHarborWeight"`
	DeepDraftEstuaryWeight   float64 `json:"deepDraftEstuaryWeight"`
	BeachLandingWeight       float64 `json:"beachLandingWeight"`
	DeepDraftExposurePenalty float64 `json:"deepDraftExposurePenalty"`
	DeepwaterAccessWeight    float64 `json:"deepwaterAccessWeight"`
	DeepwaterHarborWeight    float64 `json:"deepwaterHarborWeight"`
	DeepwaterEstuaryWeight   float64 `json:"deepwaterEstuaryWeight"`
	DeepwaterTransferWeight  float64 `json:"deepwaterTransferWeight"`
	DeepwaterStormPenalty    float64 `json:"deepwaterStormPenalty"`
	ExposurePenalty          float64 `json:"exposurePenalty"`
	StormPenalty             float64 `json:"stormPenalty"`
	PortSuitabilityFloor     float64 `json:"portSuitabilityFloor"`

	PortSuitabilityWeight          float64                           `json:"portSuitabilityWeight"`
	NodeFeatureWeight              float64                           `json:"nodeFeatureWeight"`
	NodeFeatureHarborWeight        float64                           `json:"nodeFeatureHarborWeight"`
	NodeFeatureEstuaryWeight       float64                           `json:"nodeFeatureEstuaryWeight"`
	NodeFeatureRiverTransferWeight float64                           `json:"nodeFeatureRiverTransferWeight"`
	NodeFeatureStopoverWeight      float64                           `json:"nodeFeatureStopoverWeight"`
	NodeCatchmentDecay             float64                           `json:"nodeCatchmentDecay"`
	NodeCatchmentHops              int                               `json:"nodeCatchmentHops"`
	NodeScoreWeight                float64                           `json:"nodeScoreWeight"`
	TradeCentralityWeight          float64                           `json:"tradeCentralityWeight"`
	RiverCentralityWeight          float64                           `json:"riverCentralityWeight"`
	MajorHubBonus                  float64                           `json:"majorHubBonus"`
	RiverHandoffBonus              float64                           `json:"riverHandoffBonus"`
	RegionalAnchorBonus            float64                           `json:"regionalAnchorBonus"`
	DistrictAnchorBonus            float64                           `json:"districtAnchorBonus"`
	LocalAnchorBonus               float64                           `json:"localAnchorBonus"`
	MajorPortThreshold             float64                           `json:"majorPortThreshold"`
	MajorDeepwaterPortThreshold    float64                           `json:"majorDeepwaterPortThreshold"`
	DeepwaterNodeWeight            float64                           `json:"deepwaterNodeWeight"`
	RegionalMinCentrality          float64                           `json:"regionalMinCentrality"`
	DistrictMinCentrality          float64                           `json:"districtMinCentrality"`
	LocalMinCentrality             float64                           `json:"localMinCentrality"`
	StopoverSelection              MaritimeStopoverSelectionSettings `json:"stopoverSelection"`
}

type MaritimeStopoverSelectionSettings struct {
	MinStopoverValue        float64 `json:"minStopoverValue"`
	MinPortSuitability      float64 `json:"minPortSuitability"`
	ScoreFloor              float64 `json:"scoreFloor"`
	StopoverValueWeight     float64 `json:"stopoverValueWeight"`
	PortSuitabilityWeight   float64 `json:"portSuitabilityWeight"`
	OceanExposureWeight     float64 `json:"oceanExposureWeight"`
	LandScarcityWeight      float64 `json:"landScarcityWeight"`
	FullComponentAreaEq     float64 `json:"fullComponentAreaEq"`
	MinComponentScoreFactor float64 `json:"minComponentScoreFactor"`
	ComponentTaperPower     float64 `json:"componentTaperPower"`
}

func DefaultMaritimePortSettings() MaritimePortSettings {
	settings, err := loadMaritimePortSettingsData(worldgen.EmbeddedMaritimePortSettings())
	if err == nil {
		return settings
	}
	return MaritimePortSettings{
		SchemaVersion:                  MaritimePortSchemaVersion,
		HarborShelterWeight:            0.34,
		EstuaryWeight:                  0.18,
		RiverTransferWeight:            0.20,
		StopoverWeight:                 0.12,
		BeachingWeight:                 0.10,
		ShallowDraftWeight:             0.08,
		DeepDraftHarborWeight:          0.12,
		DeepDraftEstuaryWeight:         0.06,
		BeachLandingWeight:             0.08,
		DeepDraftExposurePenalty:       0.10,
		DeepwaterAccessWeight:          0.34,
		DeepwaterHarborWeight:          0.30,
		DeepwaterEstuaryWeight:         0.16,
		DeepwaterTransferWeight:        0.08,
		DeepwaterStormPenalty:          0.16,
		ExposurePenalty:                0.16,
		StormPenalty:                   0.14,
		PortSuitabilityFloor:           0.18,
		PortSuitabilityWeight:          0.46,
		NodeFeatureWeight:              0.30,
		NodeFeatureHarborWeight:        0.34,
		NodeFeatureEstuaryWeight:       0.28,
		NodeFeatureRiverTransferWeight: 0.22,
		NodeFeatureStopoverWeight:      0.16,
		NodeCatchmentDecay:             0.82,
		NodeCatchmentHops:              4,
		NodeScoreWeight:                0.24,
		TradeCentralityWeight:          0.16,
		RiverCentralityWeight:          0.12,
		MajorHubBonus:                  0.12,
		RiverHandoffBonus:              0.08,
		RegionalAnchorBonus:            0.12,
		DistrictAnchorBonus:            0.06,
		LocalAnchorBonus:               -0.02,
		MajorPortThreshold:             0.64,
		MajorDeepwaterPortThreshold:    0.58,
		DeepwaterNodeWeight:            0.36,
		RegionalMinCentrality:          0.00,
		DistrictMinCentrality:          0.04,
		LocalMinCentrality:             0.18,
		StopoverSelection:              DefaultMaritimeStopoverSelectionSettings(),
	}
}

func DefaultMaritimeStopoverSelectionSettings() MaritimeStopoverSelectionSettings {
	return MaritimeStopoverSelectionSettings{
		MinStopoverValue:        0.26,
		MinPortSuitability:      0.22,
		ScoreFloor:              0.28,
		StopoverValueWeight:     0.52,
		PortSuitabilityWeight:   0.22,
		OceanExposureWeight:     0.16,
		LandScarcityWeight:      0.10,
		FullComponentAreaEq:     8.00,
		MinComponentScoreFactor: 0.48,
		ComponentTaperPower:     0.70,
	}
}

func LoadMaritimePortSettings(path string) (MaritimePortSettings, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return MaritimePortSettings{}, err
	}
	return loadMaritimePortSettingsData(raw)
}

func loadMaritimePortSettingsData(data []byte) (MaritimePortSettings, error) {
	var settings MaritimePortSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return MaritimePortSettings{}, fmt.Errorf("decode maritime port settings: %w", err)
	}
	if err := settings.Validate(); err != nil {
		return MaritimePortSettings{}, err
	}
	return settings, nil
}

func (s MaritimePortSettings) Validate() error {
	if s.SchemaVersion == "" {
		return fmt.Errorf("maritime port schemaVersion is required")
	}
	if s.SchemaVersion != MaritimePortSchemaVersion {
		return fmt.Errorf("unsupported maritime port schemaVersion %q", s.SchemaVersion)
	}
	values := map[string]float64{
		"harborShelterWeight":            s.HarborShelterWeight,
		"estuaryWeight":                  s.EstuaryWeight,
		"riverTransferWeight":            s.RiverTransferWeight,
		"stopoverWeight":                 s.StopoverWeight,
		"beachingWeight":                 s.BeachingWeight,
		"shallowDraftWeight":             s.ShallowDraftWeight,
		"deepDraftHarborWeight":          s.DeepDraftHarborWeight,
		"deepDraftEstuaryWeight":         s.DeepDraftEstuaryWeight,
		"beachLandingWeight":             s.BeachLandingWeight,
		"deepDraftExposurePenalty":       s.DeepDraftExposurePenalty,
		"deepwaterAccessWeight":          s.DeepwaterAccessWeight,
		"deepwaterHarborWeight":          s.DeepwaterHarborWeight,
		"deepwaterEstuaryWeight":         s.DeepwaterEstuaryWeight,
		"deepwaterTransferWeight":        s.DeepwaterTransferWeight,
		"deepwaterStormPenalty":          s.DeepwaterStormPenalty,
		"exposurePenalty":                s.ExposurePenalty,
		"stormPenalty":                   s.StormPenalty,
		"portSuitabilityFloor":           s.PortSuitabilityFloor,
		"portSuitabilityWeight":          s.PortSuitabilityWeight,
		"nodeFeatureWeight":              s.NodeFeatureWeight,
		"nodeFeatureHarborWeight":        s.NodeFeatureHarborWeight,
		"nodeFeatureEstuaryWeight":       s.NodeFeatureEstuaryWeight,
		"nodeFeatureRiverTransferWeight": s.NodeFeatureRiverTransferWeight,
		"nodeFeatureStopoverWeight":      s.NodeFeatureStopoverWeight,
		"nodeCatchmentDecay":             s.NodeCatchmentDecay,
		"nodeScoreWeight":                s.NodeScoreWeight,
		"tradeCentralityWeight":          s.TradeCentralityWeight,
		"riverCentralityWeight":          s.RiverCentralityWeight,
		"majorHubBonus":                  s.MajorHubBonus,
		"riverHandoffBonus":              s.RiverHandoffBonus,
		"regionalAnchorBonus":            s.RegionalAnchorBonus,
		"districtAnchorBonus":            s.DistrictAnchorBonus,
		"majorPortThreshold":             s.MajorPortThreshold,
		"majorDeepwaterPortThreshold":    s.MajorDeepwaterPortThreshold,
		"deepwaterNodeWeight":            s.DeepwaterNodeWeight,
	}
	for name, value := range values {
		if value < 0 {
			return fmt.Errorf("%s cannot be negative", name)
		}
	}
	if s.NodeCatchmentHops < 0 {
		return fmt.Errorf("nodeCatchmentHops cannot be negative")
	}
	if s.RegionalMinCentrality < 0 || s.DistrictMinCentrality < 0 || s.LocalMinCentrality < 0 {
		return fmt.Errorf("centrality floors cannot be negative")
	}
	if err := s.StopoverSelection.Validate(); err != nil {
		return err
	}
	return nil
}

func (s MaritimeStopoverSelectionSettings) Validate() error {
	values := map[string]float64{
		"stopoverSelection.minStopoverValue":        s.MinStopoverValue,
		"stopoverSelection.minPortSuitability":      s.MinPortSuitability,
		"stopoverSelection.scoreFloor":              s.ScoreFloor,
		"stopoverSelection.stopoverValueWeight":     s.StopoverValueWeight,
		"stopoverSelection.portSuitabilityWeight":   s.PortSuitabilityWeight,
		"stopoverSelection.oceanExposureWeight":     s.OceanExposureWeight,
		"stopoverSelection.landScarcityWeight":      s.LandScarcityWeight,
		"stopoverSelection.fullComponentAreaEq":     s.FullComponentAreaEq,
		"stopoverSelection.minComponentScoreFactor": s.MinComponentScoreFactor,
		"stopoverSelection.componentTaperPower":     s.ComponentTaperPower,
	}
	for name, value := range values {
		if value < 0 {
			return fmt.Errorf("%s cannot be negative", name)
		}
	}
	if s.FullComponentAreaEq <= 0 {
		return fmt.Errorf("stopoverSelection.fullComponentAreaEq must be positive")
	}
	if s.MinComponentScoreFactor > 1 {
		return fmt.Errorf("stopoverSelection.minComponentScoreFactor cannot exceed 1")
	}
	if s.ComponentTaperPower <= 0 {
		return fmt.Errorf("stopoverSelection.componentTaperPower must be positive")
	}
	totalWeight := s.StopoverValueWeight + s.PortSuitabilityWeight + s.OceanExposureWeight + s.LandScarcityWeight
	if totalWeight <= 0 {
		return fmt.Errorf("stopoverSelection score weights must have positive total")
	}
	return nil
}
