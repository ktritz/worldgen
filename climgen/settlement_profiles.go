package climgen

import (
	"encoding/json"
	"fmt"
	"os"
)

type SettlementPreferenceProfile struct {
	Name                string
	ClimateWeight       float64
	WaterWeight         float64
	TerrainWeight       float64
	SoilWeight          float64
	AccessWeight        float64
	ResourceWeight      float64
	HazardWeight        float64
	RiverBias           float64
	CoastalBias         float64
	AlluvialBias        float64
	FertilityBias       float64
	ForestBias          float64
	WetlandBias         float64
	RockBias            float64
	ElevationBias       float64
	ColdTolerance       float64
	AridityTolerance    float64
	FavorableThreshold  float64
	PrimeThreshold      float64
}

type SettlementPreferenceResult struct {
	Profile     SettlementPreferenceProfile
	Suitability []float64
	Classes     []SettlementClass
}

type SettlementPreferenceProfileFile struct {
	Profiles []SettlementPreferenceProfile `json:"profiles"`
}

func DefaultFantasySettlementProfiles() []SettlementPreferenceProfile {
	return []SettlementPreferenceProfile{
		{
			Name:               "Human",
			ClimateWeight:      1.00,
			WaterWeight:        1.00,
			TerrainWeight:      1.00,
			SoilWeight:         1.00,
			AccessWeight:       1.00,
			ResourceWeight:     1.00,
			HazardWeight:       1.00,
			RiverBias:          0.04,
			CoastalBias:        0.04,
			AlluvialBias:       0.05,
			FertilityBias:      0.04,
			ForestBias:         0.00,
			WetlandBias:       -0.03,
			RockBias:          -0.03,
			ElevationBias:     -0.02,
			ColdTolerance:      0.00,
			AridityTolerance:   0.00,
			FavorableThreshold: 0.42,
			PrimeThreshold:     0.66,
		},
		{
			Name:               "Elf",
			ClimateWeight:      1.05,
			WaterWeight:        1.00,
			TerrainWeight:      0.95,
			SoilWeight:         0.95,
			AccessWeight:       0.85,
			ResourceWeight:     0.65,
			HazardWeight:       0.90,
			RiverBias:          0.02,
			CoastalBias:       -0.01,
			AlluvialBias:       0.01,
			FertilityBias:      0.02,
			ForestBias:         0.10,
			WetlandBias:       -0.01,
			RockBias:          -0.04,
			ElevationBias:     -0.01,
			ColdTolerance:      0.02,
			AridityTolerance:  -0.04,
			FavorableThreshold: 0.41,
			PrimeThreshold:     0.64,
		},
		{
			Name:               "Dwarf",
			ClimateWeight:      0.80,
			WaterWeight:        0.80,
			TerrainWeight:      1.20,
			SoilWeight:         0.65,
			AccessWeight:       0.70,
			ResourceWeight:     1.35,
			HazardWeight:       0.70,
			RiverBias:         -0.01,
			CoastalBias:       -0.05,
			AlluvialBias:      -0.02,
			FertilityBias:     -0.04,
			ForestBias:        -0.03,
			WetlandBias:       -0.08,
			RockBias:           0.09,
			ElevationBias:      0.08,
			ColdTolerance:      0.06,
			AridityTolerance:   0.03,
			FavorableThreshold: 0.38,
			PrimeThreshold:     0.61,
		},
		{
			Name:               "Halfling",
			ClimateWeight:      1.05,
			WaterWeight:        1.10,
			TerrainWeight:      0.90,
			SoilWeight:         1.15,
			AccessWeight:       1.00,
			ResourceWeight:     0.75,
			HazardWeight:       1.00,
			RiverBias:          0.06,
			CoastalBias:        0.02,
			AlluvialBias:       0.08,
			FertilityBias:      0.08,
			ForestBias:         0.01,
			WetlandBias:       -0.05,
			RockBias:          -0.07,
			ElevationBias:     -0.06,
			ColdTolerance:     -0.01,
			AridityTolerance:  -0.03,
			FavorableThreshold: 0.40,
			PrimeThreshold:     0.62,
		},
	}
}

func LoadSettlementPreferenceProfiles(path string) ([]SettlementPreferenceProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file SettlementPreferenceProfileFile
	if err := json.Unmarshal(data, &file); err == nil && len(file.Profiles) > 0 {
		if err := validateSettlementPreferenceProfiles(file.Profiles); err != nil {
			return nil, err
		}
		return file.Profiles, nil
	}
	var profiles []SettlementPreferenceProfile
	if err := json.Unmarshal(data, &profiles); err != nil {
		return nil, fmt.Errorf("decode settlement profiles: %w", err)
	}
	if err := validateSettlementPreferenceProfiles(profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

func validateSettlementPreferenceProfiles(profiles []SettlementPreferenceProfile) error {
	if len(profiles) == 0 {
		return fmt.Errorf("no settlement profiles provided")
	}
	seen := make(map[string]struct{}, len(profiles))
	for _, profile := range profiles {
		if profile.Name == "" {
			return fmt.Errorf("settlement profile name cannot be empty")
		}
		if _, ok := seen[profile.Name]; ok {
			return fmt.Errorf("duplicate settlement profile %q", profile.Name)
		}
		seen[profile.Name] = struct{}{}
		if profile.FavorableThreshold <= 0 || profile.PrimeThreshold <= 0 || profile.PrimeThreshold <= profile.FavorableThreshold {
			return fmt.Errorf("invalid thresholds for profile %q", profile.Name)
		}
	}
	return nil
}

func ClassifySettlementPreference(
	base *SettlementResult,
	biomes *BiomeResult,
	soils *SoilResult,
	vegetation *VegetationResult,
	elevation []float64,
	seaLevel float64,
	profile SettlementPreferenceProfile,
) *SettlementPreferenceResult {
	n := len(elevation)
	out := &SettlementPreferenceResult{
		Profile:     profile,
		Suitability: make([]float64, n),
		Classes:     make([]SettlementClass, n),
	}
	if base == nil || base.Diagnostics == nil || biomes == nil || biomes.Diagnostics == nil || soils == nil || soils.Diagnostics == nil {
		return out
	}
	bdiag := biomes.Diagnostics
	sdiag := soils.Diagnostics
	for i := 0; i < n; i++ {
		if elevation[i] < seaLevel {
			out.Classes[i] = SettlementOcean
			continue
		}
		score := 0.0
		score += 0.23 * profile.ClimateWeight * base.Diagnostics.ClimateScore[i]
		score += 0.21 * profile.WaterWeight * base.Diagnostics.WaterScore[i]
		score += 0.17 * profile.TerrainWeight * base.Diagnostics.TerrainScore[i]
		score += 0.17 * profile.SoilWeight * base.Diagnostics.SoilScore[i]
		score += 0.13 * profile.AccessWeight * base.Diagnostics.AccessScore[i]
		score += 0.09 * profile.ResourceWeight * base.Diagnostics.ResourceScore[i]
		score -= 0.24 * profile.HazardWeight * base.Diagnostics.HazardPenalty[i]

		score += profile.RiverBias * base.Diagnostics.RiverBonus[i]
		score += profile.CoastalBias * base.Diagnostics.CoastalBonus[i]
		score += profile.AlluvialBias * sdiag.Alluvial[i]
		score += profile.FertilityBias * sdiag.Fertility[i]
		score += profile.RockBias * sdiag.Rockiness[i]
		score += profile.ElevationBias * smoothstep01(500, 2400, elevation[i])
		score += profile.ColdTolerance * (1 - smoothstep01(0.10, 0.85, bdiag.AnnualIceFraction[i]))
		score += profile.AridityTolerance * smoothstep01(0.30, 1.00, bdiag.AridityRatio[i])

		if vegetation != nil && vegetation.Diagnostics != nil && i < len(vegetation.Types) {
			forestness := clamp01(0.60*vegetation.Diagnostics.TreeCover[i] + 0.40*vegetation.Diagnostics.MoistureAvailability[i])
			wetness := vegetation.Diagnostics.WetlandCover[i]
			score += profile.ForestBias * forestness
			score += profile.WetlandBias * wetness
		}

		suitability := clamp01(score)
		out.Suitability[i] = suitability
		out.Classes[i] = classifySettlementClassWithThresholds(suitability, profile.FavorableThreshold, profile.PrimeThreshold)
	}
	return out
}

func classifySettlementClassWithThresholds(score, favorable, prime float64) SettlementClass {
	switch {
	case score >= prime:
		return SettlementPrime
	case score >= favorable:
		return SettlementFavorable
	case score >= favorable-0.20:
		return SettlementMarginal
	default:
		return SettlementUnsuitable
	}
}

func DominantSettlementPreference(results []*SettlementPreferenceResult) []int {
	if len(results) == 0 {
		return nil
	}
	n := len(results[0].Classes)
	best := make([]int, n)
	for i := 0; i < n; i++ {
		bestScore := -1.0
		bestIdx := 0
		for p, result := range results {
			if i >= len(result.Suitability) {
				continue
			}
			if result.Suitability[i] > bestScore {
				bestScore = result.Suitability[i]
				bestIdx = p
			}
		}
		best[i] = bestIdx
	}
	return best
}
