package climgen

import (
	"encoding/json"
	"fmt"
	"os"
)

const ProfileCatalogSchemaVersion = "profile-catalog/v1"

type ProfileSocialModule struct {
	Openness             float64 `json:"openness"`
	HierarchyPreference  float64 `json:"hierarchyPreference"`
	TraditionPreference  float64 `json:"traditionPreference"`
	ClanBias             float64 `json:"clanBias"`
	GuildBias            float64 `json:"guildBias"`
}

type ProfileSocialOverrides struct {
	Openness            *float64 `json:"openness,omitempty"`
	HierarchyPreference *float64 `json:"hierarchyPreference,omitempty"`
	TraditionPreference *float64 `json:"traditionPreference,omitempty"`
	ClanBias            *float64 `json:"clanBias,omitempty"`
	GuildBias           *float64 `json:"guildBias,omitempty"`
}

type ProfileGovernanceModule struct {
	CentralizationPreference float64 `json:"centralizationPreference"`
	LegalismPreference       float64 `json:"legalismPreference"`
	MeritPreference          float64 `json:"meritPreference"`
	RepublicBias             float64 `json:"republicBias"`
	AutocracyBias            float64 `json:"autocracyBias"`
	TheocracyBias            float64 `json:"theocracyBias"`
}

type ProfileGovernanceOverrides struct {
	CentralizationPreference *float64 `json:"centralizationPreference,omitempty"`
	LegalismPreference       *float64 `json:"legalismPreference,omitempty"`
	MeritPreference          *float64 `json:"meritPreference,omitempty"`
	RepublicBias             *float64 `json:"republicBias,omitempty"`
	AutocracyBias            *float64 `json:"autocracyBias,omitempty"`
	TheocracyBias            *float64 `json:"theocracyBias,omitempty"`
}

type ProfileEconomicModule struct {
	TradeBias           float64            `json:"tradeBias"`
	AgrarianBias        float64            `json:"agrarianBias"`
	CraftBias           float64            `json:"craftBias"`
	ExtractiveBias      float64            `json:"extractiveBias"`
	ProfessionAptitudes map[string]float64 `json:"professionAptitudes,omitempty"`
}

type ProfileEconomicOverrides struct {
	TradeBias           *float64           `json:"tradeBias,omitempty"`
	AgrarianBias        *float64           `json:"agrarianBias,omitempty"`
	CraftBias           *float64           `json:"craftBias,omitempty"`
	ExtractiveBias      *float64           `json:"extractiveBias,omitempty"`
	ProfessionAptitudes map[string]float64 `json:"professionAptitudes,omitempty"`
}

type ProfileAttitudeModule struct {
	Xenophilia float64 `json:"xenophilia"`
	Aggression float64 `json:"aggression"`
	HonorBias  float64 `json:"honorBias"`
	Curiosity  float64 `json:"curiosity"`
}

type ProfileAttitudeOverrides struct {
	Xenophilia *float64 `json:"xenophilia,omitempty"`
	Aggression *float64 `json:"aggression,omitempty"`
	HonorBias  *float64 `json:"honorBias,omitempty"`
	Curiosity  *float64 `json:"curiosity,omitempty"`
}

type SettlementPreferenceOverrides struct {
	ClimateWeight      *float64 `json:"ClimateWeight,omitempty"`
	WaterWeight        *float64 `json:"WaterWeight,omitempty"`
	TerrainWeight      *float64 `json:"TerrainWeight,omitempty"`
	SoilWeight         *float64 `json:"SoilWeight,omitempty"`
	AccessWeight       *float64 `json:"AccessWeight,omitempty"`
	ResourceWeight     *float64 `json:"ResourceWeight,omitempty"`
	HazardWeight       *float64 `json:"HazardWeight,omitempty"`
	RiverBias          *float64 `json:"RiverBias,omitempty"`
	CoastalBias        *float64 `json:"CoastalBias,omitempty"`
	AlluvialBias       *float64 `json:"AlluvialBias,omitempty"`
	FertilityBias      *float64 `json:"FertilityBias,omitempty"`
	ForestBias         *float64 `json:"ForestBias,omitempty"`
	WetlandBias        *float64 `json:"WetlandBias,omitempty"`
	RockBias           *float64 `json:"RockBias,omitempty"`
	ElevationBias      *float64 `json:"ElevationBias,omitempty"`
	ColdTolerance      *float64 `json:"ColdTolerance,omitempty"`
	AridityTolerance   *float64 `json:"AridityTolerance,omitempty"`
	FavorableThreshold *float64 `json:"FavorableThreshold,omitempty"`
	PrimeThreshold     *float64 `json:"PrimeThreshold,omitempty"`
}

type AncestryProfile struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description,omitempty"`
	Tags        []string                     `json:"tags,omitempty"`
	Settlement  *SettlementPreferenceProfile `json:"settlement,omitempty"`
	Social      *ProfileSocialModule         `json:"social,omitempty"`
	Governance  *ProfileGovernanceModule     `json:"governance,omitempty"`
	Economy     *ProfileEconomicModule       `json:"economy,omitempty"`
	Attitudes   *ProfileAttitudeModule       `json:"attitudes,omitempty"`
}

type CultureProfile struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	Tags        []string                 `json:"tags,omitempty"`
	Settlement  *SettlementPreferenceOverrides `json:"settlement,omitempty"`
	Social      *ProfileSocialOverrides        `json:"social,omitempty"`
	Governance  *ProfileGovernanceOverrides    `json:"governance,omitempty"`
	Economy     *ProfileEconomicOverrides      `json:"economy,omitempty"`
	Attitudes   *ProfileAttitudeOverrides      `json:"attitudes,omitempty"`
}

type ProfileCatalog struct {
	SchemaVersion string           `json:"schemaVersion"`
	Ancestries    []AncestryProfile      `json:"ancestries,omitempty"`
	Cultures      []CultureProfile       `json:"cultures,omitempty"`
	Compositions  []ProfileCompositionSpec `json:"compositions,omitempty"`
}

type ProfileCompositionSpec struct {
	Name     string `json:"name"`
	Ancestry string `json:"ancestry"`
	Culture  string `json:"culture,omitempty"`
}

type ResolvedProfile struct {
	AncestryName string
	CultureName  string
	Settlement   *SettlementPreferenceProfile
	Social       *ProfileSocialModule
	Governance   *ProfileGovernanceModule
	Economy      *ProfileEconomicModule
	Attitudes    *ProfileAttitudeModule
}

func DefaultFantasyProfileCatalog() *ProfileCatalog {
	profiles := DefaultFantasySettlementProfiles()
	catalog := &ProfileCatalog{
		SchemaVersion: ProfileCatalogSchemaVersion,
		Ancestries:    make([]AncestryProfile, 0, len(profiles)),
	}
	for _, profile := range profiles {
		ancestry := AncestryProfile{
			Name:       profile.Name,
			Settlement: cloneSettlementPreferenceProfile(profile),
		}
		switch profile.Name {
		case "Human":
			ancestry.Tags = []string{"adaptable", "coastal", "agrarian"}
			ancestry.Social = &ProfileSocialModule{Openness: 0.55, HierarchyPreference: 0.45, TraditionPreference: 0.45, ClanBias: 0.35, GuildBias: 0.60}
			ancestry.Governance = &ProfileGovernanceModule{CentralizationPreference: 0.50, LegalismPreference: 0.55, MeritPreference: 0.50, RepublicBias: 0.45, AutocracyBias: 0.35, TheocracyBias: 0.20}
			ancestry.Economy = &ProfileEconomicModule{TradeBias: 0.60, AgrarianBias: 0.65, CraftBias: 0.45, ExtractiveBias: 0.35, ProfessionAptitudes: map[string]float64{"farmer": 0.70, "merchant": 0.65, "administrator": 0.55}}
			ancestry.Attitudes = &ProfileAttitudeModule{Xenophilia: 0.50, Aggression: 0.45, HonorBias: 0.40, Curiosity: 0.55}
		case "Elf":
			ancestry.Tags = []string{"forest", "long-lived", "low-density"}
			ancestry.Social = &ProfileSocialModule{Openness: 0.45, HierarchyPreference: 0.35, TraditionPreference: 0.75, ClanBias: 0.40, GuildBias: 0.25}
			ancestry.Governance = &ProfileGovernanceModule{CentralizationPreference: 0.30, LegalismPreference: 0.35, MeritPreference: 0.40, RepublicBias: 0.25, AutocracyBias: 0.20, TheocracyBias: 0.35}
			ancestry.Economy = &ProfileEconomicModule{TradeBias: 0.35, AgrarianBias: 0.30, CraftBias: 0.55, ExtractiveBias: 0.10, ProfessionAptitudes: map[string]float64{"warden": 0.75, "artisan": 0.60, "scholar": 0.65}}
			ancestry.Attitudes = &ProfileAttitudeModule{Xenophilia: 0.35, Aggression: 0.20, HonorBias: 0.50, Curiosity: 0.60}
		case "Dwarf":
			ancestry.Tags = []string{"mountain", "craft", "subterranean"}
			ancestry.Social = &ProfileSocialModule{Openness: 0.30, HierarchyPreference: 0.65, TraditionPreference: 0.80, ClanBias: 0.70, GuildBias: 0.65}
			ancestry.Governance = &ProfileGovernanceModule{CentralizationPreference: 0.60, LegalismPreference: 0.70, MeritPreference: 0.55, RepublicBias: 0.20, AutocracyBias: 0.45, TheocracyBias: 0.25}
			ancestry.Economy = &ProfileEconomicModule{TradeBias: 0.40, AgrarianBias: 0.15, CraftBias: 0.80, ExtractiveBias: 0.85, ProfessionAptitudes: map[string]float64{"miner": 0.90, "smith": 0.85, "engineer": 0.75}}
			ancestry.Attitudes = &ProfileAttitudeModule{Xenophilia: 0.20, Aggression: 0.40, HonorBias: 0.70, Curiosity: 0.35}
		case "Halfling":
			ancestry.Tags = []string{"river-valley", "agrarian", "smallhold"}
			ancestry.Social = &ProfileSocialModule{Openness: 0.60, HierarchyPreference: 0.25, TraditionPreference: 0.55, ClanBias: 0.55, GuildBias: 0.35}
			ancestry.Governance = &ProfileGovernanceModule{CentralizationPreference: 0.20, LegalismPreference: 0.40, MeritPreference: 0.40, RepublicBias: 0.50, AutocracyBias: 0.10, TheocracyBias: 0.15}
			ancestry.Economy = &ProfileEconomicModule{TradeBias: 0.45, AgrarianBias: 0.85, CraftBias: 0.35, ExtractiveBias: 0.05, ProfessionAptitudes: map[string]float64{"farmer": 0.90, "brewer": 0.70, "merchant": 0.45}}
			ancestry.Attitudes = &ProfileAttitudeModule{Xenophilia: 0.55, Aggression: 0.15, HonorBias: 0.30, Curiosity: 0.40}
		}
		catalog.Ancestries = append(catalog.Ancestries, ancestry)
	}
	return catalog
}

func cloneSettlementPreferenceProfile(profile SettlementPreferenceProfile) *SettlementPreferenceProfile {
	copied := profile
	return &copied
}

func LoadProfileCatalog(path string) (*ProfileCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var catalog ProfileCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("decode profile catalog: %w", err)
	}
	if err := ValidateProfileCatalog(&catalog); err != nil {
		return nil, err
	}
	return &catalog, nil
}

func ValidateProfileCatalog(catalog *ProfileCatalog) error {
	if catalog == nil {
		return fmt.Errorf("nil profile catalog")
	}
	if catalog.SchemaVersion == "" {
		return fmt.Errorf("profile catalog schemaVersion is required")
	}
	if catalog.SchemaVersion != ProfileCatalogSchemaVersion {
		return fmt.Errorf("unsupported profile catalog schemaVersion %q", catalog.SchemaVersion)
	}
	seenAncestry := make(map[string]struct{}, len(catalog.Ancestries))
	for _, ancestry := range catalog.Ancestries {
		if ancestry.Name == "" {
			return fmt.Errorf("ancestry name cannot be empty")
		}
		if _, ok := seenAncestry[ancestry.Name]; ok {
			return fmt.Errorf("duplicate ancestry profile %q", ancestry.Name)
		}
		seenAncestry[ancestry.Name] = struct{}{}
		if ancestry.Settlement != nil {
			if err := validateSettlementPreferenceProfiles([]SettlementPreferenceProfile{*ancestry.Settlement}); err != nil {
				return fmt.Errorf("ancestry %q: %w", ancestry.Name, err)
			}
		}
	}
	seenCulture := make(map[string]struct{}, len(catalog.Cultures))
	for _, culture := range catalog.Cultures {
		if culture.Name == "" {
			return fmt.Errorf("culture name cannot be empty")
		}
		if _, ok := seenCulture[culture.Name]; ok {
			return fmt.Errorf("duplicate culture profile %q", culture.Name)
		}
		seenCulture[culture.Name] = struct{}{}
	}
	seenComposition := make(map[string]struct{}, len(catalog.Compositions))
	for _, composition := range catalog.Compositions {
		if composition.Name == "" {
			return fmt.Errorf("composition name cannot be empty")
		}
		if _, ok := seenComposition[composition.Name]; ok {
			return fmt.Errorf("duplicate composition profile %q", composition.Name)
		}
		seenComposition[composition.Name] = struct{}{}
		if composition.Ancestry == "" {
			return fmt.Errorf("composition %q missing ancestry", composition.Name)
		}
		if _, ok := seenAncestry[composition.Ancestry]; !ok {
			return fmt.Errorf("composition %q references unknown ancestry %q", composition.Name, composition.Ancestry)
		}
		if composition.Culture != "" {
			if _, ok := seenCulture[composition.Culture]; !ok {
				return fmt.Errorf("composition %q references unknown culture %q", composition.Name, composition.Culture)
			}
		}
	}
	return nil
}

func ExtractSettlementProfiles(catalog *ProfileCatalog) []SettlementPreferenceProfile {
	if catalog == nil {
		return nil
	}
	out := make([]SettlementPreferenceProfile, 0, len(catalog.Ancestries))
	for _, ancestry := range catalog.Ancestries {
		if ancestry.Settlement != nil {
			out = append(out, *ancestry.Settlement)
		}
	}
	return out
}

func ExtractComposedSettlementProfiles(catalog *ProfileCatalog) []SettlementPreferenceProfile {
	if catalog == nil {
		return nil
	}
	out := make([]SettlementPreferenceProfile, 0, len(catalog.Compositions))
	for _, composition := range catalog.Compositions {
		resolved := ResolveProfileComposition(catalog, composition)
		if resolved == nil || resolved.Settlement == nil {
			continue
		}
		profile := *resolved.Settlement
		profile.Name = composition.Name
		out = append(out, profile)
	}
	return out
}

func ResolveProfileComposition(catalog *ProfileCatalog, spec ProfileCompositionSpec) *ResolvedProfile {
	if catalog == nil {
		return nil
	}
	var ancestry *AncestryProfile
	for i := range catalog.Ancestries {
		if catalog.Ancestries[i].Name == spec.Ancestry {
			ancestry = &catalog.Ancestries[i]
			break
		}
	}
	if ancestry == nil {
		return nil
	}
	var culture *CultureProfile
	if spec.Culture != "" {
		for i := range catalog.Cultures {
			if catalog.Cultures[i].Name == spec.Culture {
				culture = &catalog.Cultures[i]
				break
			}
		}
	}
	resolved := ComposeResolvedProfile(*ancestry, culture)
	if resolved != nil && spec.Name != "" && resolved.Settlement != nil {
		resolved.Settlement.Name = spec.Name
	}
	return resolved
}

func ComposeResolvedProfile(ancestry AncestryProfile, culture *CultureProfile) *ResolvedProfile {
	resolved := &ResolvedProfile{
		AncestryName: ancestry.Name,
	}
	if culture != nil {
		resolved.CultureName = culture.Name
	}
	if ancestry.Settlement != nil {
		resolved.Settlement = cloneSettlementPreferenceProfile(*ancestry.Settlement)
	}
	if ancestry.Social != nil {
		v := *ancestry.Social
		resolved.Social = &v
	}
	if ancestry.Governance != nil {
		v := *ancestry.Governance
		resolved.Governance = &v
	}
	if ancestry.Economy != nil {
		v := *ancestry.Economy
		v.ProfessionAptitudes = copyStringFloatMap(v.ProfessionAptitudes)
		resolved.Economy = &v
	}
	if ancestry.Attitudes != nil {
		v := *ancestry.Attitudes
		resolved.Attitudes = &v
	}
	if culture == nil {
		return resolved
	}
	applySettlementOverrides(&resolved.Settlement, culture.Settlement)
	applySocialOverrides(&resolved.Social, culture.Social)
	applyGovernanceOverrides(&resolved.Governance, culture.Governance)
	applyEconomicOverrides(&resolved.Economy, culture.Economy)
	applyAttitudeOverrides(&resolved.Attitudes, culture.Attitudes)
	return resolved
}

func applySettlementOverrides(dst **SettlementPreferenceProfile, src *SettlementPreferenceOverrides) {
	if src == nil {
		return
	}
	if *dst == nil {
		*dst = &SettlementPreferenceProfile{}
	}
	set := *dst
	applyFloat(&set.ClimateWeight, src.ClimateWeight)
	applyFloat(&set.WaterWeight, src.WaterWeight)
	applyFloat(&set.TerrainWeight, src.TerrainWeight)
	applyFloat(&set.SoilWeight, src.SoilWeight)
	applyFloat(&set.AccessWeight, src.AccessWeight)
	applyFloat(&set.ResourceWeight, src.ResourceWeight)
	applyFloat(&set.HazardWeight, src.HazardWeight)
	applyFloat(&set.RiverBias, src.RiverBias)
	applyFloat(&set.CoastalBias, src.CoastalBias)
	applyFloat(&set.AlluvialBias, src.AlluvialBias)
	applyFloat(&set.FertilityBias, src.FertilityBias)
	applyFloat(&set.ForestBias, src.ForestBias)
	applyFloat(&set.WetlandBias, src.WetlandBias)
	applyFloat(&set.RockBias, src.RockBias)
	applyFloat(&set.ElevationBias, src.ElevationBias)
	applyFloat(&set.ColdTolerance, src.ColdTolerance)
	applyFloat(&set.AridityTolerance, src.AridityTolerance)
	applyFloat(&set.FavorableThreshold, src.FavorableThreshold)
	applyFloat(&set.PrimeThreshold, src.PrimeThreshold)
}

func applySocialOverrides(dst **ProfileSocialModule, src *ProfileSocialOverrides) {
	if src == nil {
		return
	}
	if *dst == nil {
		*dst = &ProfileSocialModule{}
	}
	mod := *dst
	applyFloat(&mod.Openness, src.Openness)
	applyFloat(&mod.HierarchyPreference, src.HierarchyPreference)
	applyFloat(&mod.TraditionPreference, src.TraditionPreference)
	applyFloat(&mod.ClanBias, src.ClanBias)
	applyFloat(&mod.GuildBias, src.GuildBias)
}

func applyGovernanceOverrides(dst **ProfileGovernanceModule, src *ProfileGovernanceOverrides) {
	if src == nil {
		return
	}
	if *dst == nil {
		*dst = &ProfileGovernanceModule{}
	}
	mod := *dst
	applyFloat(&mod.CentralizationPreference, src.CentralizationPreference)
	applyFloat(&mod.LegalismPreference, src.LegalismPreference)
	applyFloat(&mod.MeritPreference, src.MeritPreference)
	applyFloat(&mod.RepublicBias, src.RepublicBias)
	applyFloat(&mod.AutocracyBias, src.AutocracyBias)
	applyFloat(&mod.TheocracyBias, src.TheocracyBias)
}

func applyEconomicOverrides(dst **ProfileEconomicModule, src *ProfileEconomicOverrides) {
	if src == nil {
		return
	}
	if *dst == nil {
		*dst = &ProfileEconomicModule{}
	}
	mod := *dst
	applyFloat(&mod.TradeBias, src.TradeBias)
	applyFloat(&mod.AgrarianBias, src.AgrarianBias)
	applyFloat(&mod.CraftBias, src.CraftBias)
	applyFloat(&mod.ExtractiveBias, src.ExtractiveBias)
	if len(src.ProfessionAptitudes) > 0 {
		if mod.ProfessionAptitudes == nil {
			mod.ProfessionAptitudes = make(map[string]float64, len(src.ProfessionAptitudes))
		}
		for k, v := range src.ProfessionAptitudes {
			mod.ProfessionAptitudes[k] = v
		}
	}
}

func applyAttitudeOverrides(dst **ProfileAttitudeModule, src *ProfileAttitudeOverrides) {
	if src == nil {
		return
	}
	if *dst == nil {
		*dst = &ProfileAttitudeModule{}
	}
	mod := *dst
	applyFloat(&mod.Xenophilia, src.Xenophilia)
	applyFloat(&mod.Aggression, src.Aggression)
	applyFloat(&mod.HonorBias, src.HonorBias)
	applyFloat(&mod.Curiosity, src.Curiosity)
}

func copyStringFloatMap(src map[string]float64) map[string]float64 {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]float64, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func applyFloat(dst *float64, src *float64) {
	if src != nil {
		*dst = *src
	}
}
