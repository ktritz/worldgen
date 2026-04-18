package main

import (
	"fmt"
	"strconv"
	"strings"

	"worldgen/climgen"
)

func parseSeeds(raw string) ([]int64, error) {
	parts := strings.Split(raw, ",")
	seeds := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", part, err)
		}
		seeds = append(seeds, value)
	}
	if len(seeds) == 0 {
		return nil, fmt.Errorf("no seeds provided")
	}
	return seeds, nil
}

func loadSettlementProfiles(enabled bool, path string) []climgen.SettlementPreferenceProfile {
	profiles := climgen.DefaultFantasySettlementProfiles()
	if !enabled {
		return profiles
	}
	if catalog, err := climgen.LoadProfileCatalog(path); err != nil {
		fmt.Printf("Using built-in profile catalog, failed to load %s: %v\n", path, err)
		return profiles
	} else {
		profiles = climgen.ExtractComposedSettlementProfiles(catalog)
		if len(profiles) == 0 {
			profiles = climgen.ExtractSettlementProfiles(catalog)
			fmt.Printf("Loaded %d ancestry settlement profiles from %s\n", len(profiles), path)
		} else {
			fmt.Printf("Loaded %d composed settlement profiles from %s\n", len(profiles), path)
		}
	}
	return profiles
}

func loadProfileCatalogWithFallback(path string) *climgen.ProfileCatalog {
	catalog, err := climgen.LoadProfileCatalog(path)
	if err != nil {
		fmt.Printf("Using built-in profile catalog, failed to load %s: %v\n", path, err)
		return climgen.DefaultFantasyProfileCatalog()
	}
	fmt.Printf("Loaded profile catalog from %s\n", path)
	return catalog
}

func extractSettlementProfilesFromCatalog(enabled bool, catalog *climgen.ProfileCatalog) []climgen.SettlementPreferenceProfile {
	profiles := climgen.DefaultFantasySettlementProfiles()
	if !enabled || catalog == nil {
		return profiles
	}
	profiles = climgen.ExtractComposedSettlementProfiles(catalog)
	if len(profiles) == 0 {
		profiles = climgen.ExtractSettlementProfiles(catalog)
	}
	return profiles
}

func loadResourceAbundanceSettings(path string) climgen.ResourceAbundanceSettings {
	settings := climgen.DefaultResourceAbundanceSettings()
	if loaded, err := climgen.LoadResourceAbundanceSettings(path); err != nil {
		fmt.Printf("Using built-in resource abundance settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded resource abundance settings from %s\n", path)
	}
	return settings
}

func loadAgricultureSettings(path string) climgen.AgricultureProductivitySettings {
	settings := climgen.DefaultAgricultureProductivitySettings()
	if loaded, err := climgen.LoadAgricultureProductivitySettings(path); err != nil {
		fmt.Printf("Using built-in agriculture productivity settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded agriculture productivity settings from %s\n", path)
	}
	return settings
}

func loadWildlifeSettings(path string) climgen.WildlifeProductivitySettings {
	settings := climgen.DefaultWildlifeProductivitySettings()
	if loaded, err := climgen.LoadWildlifeProductivitySettings(path); err != nil {
		fmt.Printf("Using built-in wildlife productivity settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded wildlife productivity settings from %s\n", path)
	}
	return settings
}

func loadCoastalResourceSettings(path string) climgen.CoastalResourceSettings {
	settings := climgen.DefaultCoastalResourceSettings()
	if loaded, err := climgen.LoadCoastalResourceSettings(path); err != nil {
		fmt.Printf("Using built-in coastal resource settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded coastal resource settings from %s\n", path)
	}
	return settings
}

func loadWaterResourceSettings(path string) climgen.WaterResourceSettings {
	settings := climgen.DefaultWaterResourceSettings()
	if loaded, err := climgen.LoadWaterResourceSettings(path); err != nil {
		fmt.Printf("Using built-in water resource settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded water resource settings from %s\n", path)
	}
	return settings
}

func loadPopulationSupportSettings(path string) climgen.PopulationSupportSettings {
	settings := climgen.DefaultPopulationSupportSettings()
	if loaded, err := climgen.LoadPopulationSupportSettings(path); err != nil {
		fmt.Printf("Using built-in population support settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded population support settings from %s\n", path)
	}
	return settings
}

func loadLandRouteSettings(path string) climgen.LandRouteSettings {
	settings := climgen.DefaultLandRouteSettings()
	if loaded, err := climgen.LoadLandRouteSettings(path); err != nil {
		fmt.Printf("Using built-in land route settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded land route settings from %s\n", path)
	}
	return settings
}

func loadRiverRouteSettings(path string) climgen.RiverRouteSettings {
	settings := climgen.DefaultRiverRouteSettings()
	if loaded, err := climgen.LoadRiverRouteSettings(path); err != nil {
		fmt.Printf("Using built-in river route settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded river route settings from %s\n", path)
	}
	return settings
}

func loadMaritimeRouteSettings(path string) climgen.MaritimeRouteSettings {
	settings := climgen.DefaultMaritimeRouteSettings()
	if loaded, err := climgen.LoadMaritimeRouteSettings(path); err != nil {
		fmt.Printf("Using built-in maritime vessel settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded maritime vessel settings from %s\n", path)
	}
	return settings
}

func loadMaritimePortSettings(path string) climgen.MaritimePortSettings {
	settings := climgen.DefaultMaritimePortSettings()
	if loaded, err := climgen.LoadMaritimePortSettings(path); err != nil {
		fmt.Printf("Using built-in maritime port settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded maritime port settings from %s\n", path)
	}
	return settings
}

func loadCoastalTradeSettings(path string) climgen.CoastalTradeSettings {
	settings := climgen.DefaultCoastalTradeSettings()
	if loaded, err := climgen.LoadCoastalTradeSettings(path); err != nil {
		fmt.Printf("Using built-in coastal trade settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded coastal trade settings from %s\n", path)
	}
	return settings
}

func loadOceanTradeSettings(path string) climgen.OceanTradeSettings {
	settings := climgen.DefaultOceanTradeSettings()
	if loaded, err := climgen.LoadOceanTradeSettings(path); err != nil {
		fmt.Printf("Using built-in ocean trade settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded ocean trade settings from %s\n", path)
	}
	return settings
}

func loadTradeGoodsSettings(path string) climgen.TradeGoodsSettings {
	settings := climgen.DefaultTradeGoodsSettings()
	if loaded, err := climgen.LoadTradeGoodsSettings(path); err != nil {
		fmt.Printf("Using built-in trade goods settings, failed to load %s: %v\n", path, err)
	} else {
		settings = loaded
		fmt.Printf("Loaded trade goods settings from %s\n", path)
	}
	return settings
}

func selectMaritimeComparisonVessels(raw string, settings climgen.MaritimeRouteSettings) []string {
	addUnique := func(out *[]string, seen map[string]struct{}, name string) {
		if name == "" {
			return
		}
		if _, ok := settings.VesselByName(name); !ok {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		*out = append(*out, name)
	}

	out := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	if strings.TrimSpace(raw) == "" {
		for _, name := range []string{settings.DefaultVessel, "lateen-dhow", "knarr", "caravel"} {
			addUnique(&out, seen, name)
		}
	} else {
		for _, name := range strings.Split(raw, ",") {
			addUnique(&out, seen, strings.TrimSpace(name))
		}
	}
	if len(out) == 0 {
		if settings.DefaultVessel != "" {
			addUnique(&out, seen, settings.DefaultVessel)
		} else if len(settings.Vessels) > 0 {
			addUnique(&out, seen, settings.Vessels[0].Name)
		}
	}
	return out
}

func maritimeSettingsForVessel(settings climgen.MaritimeRouteSettings, vesselName string) climgen.MaritimeRouteSettings {
	override := settings
	override.DefaultVessel = vesselName
	return override
}

func maritimeOutputSuffix(defaultVessel, vesselName string) string {
	if vesselName == "" || vesselName == defaultVessel {
		return ""
	}
	replacer := strings.NewReplacer(" ", "_", "/", "_")
	return "_" + replacer.Replace(vesselName)
}
