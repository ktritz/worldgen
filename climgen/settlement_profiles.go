package climgen

import (
	"encoding/json"
	"fmt"
	"os"

	worldgen "worldgen"
)

type SettlementPreferenceProfile struct {
	Name               string
	ClimateWeight      float64
	WaterWeight        float64
	TerrainWeight      float64
	SoilWeight         float64
	AccessWeight       float64
	ResourceWeight     float64
	HazardWeight       float64
	RiverBias          float64
	CoastalBias        float64
	AlluvialBias       float64
	FertilityBias      float64
	ForestBias         float64
	WetlandBias        float64
	RockBias           float64
	ElevationBias      float64
	ColdTolerance      float64
	AridityTolerance   float64
	FavorableThreshold float64
	PrimeThreshold     float64
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
	profiles, err := loadSettlementPreferenceProfilesData(worldgen.EmbeddedSettlementProfiles())
	if err != nil {
		return nil
	}
	return profiles
}

func LoadSettlementPreferenceProfiles(path string) ([]SettlementPreferenceProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return loadSettlementPreferenceProfilesData(data)
}

func loadSettlementPreferenceProfilesData(data []byte) ([]SettlementPreferenceProfile, error) {
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
