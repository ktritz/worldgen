package climgen

import "testing"

func TestBuildPolityProfilesAssignsProfilesAndAttitudes(t *testing.T) {
	catalog := &ProfileCatalog{
		SchemaVersion: ProfileCatalogSchemaVersion,
		Compositions: []ProfileCompositionSpec{
			{Name: "Elf + Merchant League", Ancestry: "Elf", Culture: "Merchant League"},
			{Name: "Dwarf + Highland Hold", Ancestry: "Dwarf", Culture: "Highland Hold"},
		},
		Ancestries: []AncestryProfile{
			{Name: "Elf", Tags: []string{"forest"}, Traits: ProfileTraitMap{"order": 0.4}, Affinities: []ProfileAffinityRule{{TargetType: "tag", Target: "river", Weight: 0.3}}},
			{Name: "Dwarf", Tags: []string{"mountain"}, Traits: ProfileTraitMap{"order": 0.7}, Affinities: []ProfileAffinityRule{{TargetType: "tag", Target: "mountain", Weight: 0.4}}},
		},
		Cultures: []CultureProfile{
			{Name: "Merchant League", Tags: []string{"mercantile", "coastal"}, Traits: ProfileTraitMap{"mercantile": 0.9}},
			{Name: "Highland Hold", Tags: []string{"mountain", "fortress"}, Traits: ProfileTraitMap{"warlike": 0.6}},
		},
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, Score: 0.66, River: true},
			{ID: 1, CellIndex: 1, Kind: SettlementNodeTown, Score: 0.68, Coastal: true},
		},
	}
	trade := &TradeNetworkResult{
		Corridors: []TradeCorridor{{FromNode: 0, ToNode: 1, Flow: 0.5}},
		Diagnostics: &TradeNetworkDiagnostics{
			NodeCentrality: []float64{0.10, 0.22},
		},
	}
	polities := &PolitySphereResult{
		Spheres: []PolitySphere{
			{ID: 0, CapitalNode: 0, Style: ProtoCivilizationRiverine, River: true},
			{ID: 1, CapitalNode: 1, Style: ProtoCivilizationMaritime, Coastal: true},
		},
		Diagnostics: &PolitySphereDiagnostics{
			PolityByCell: []int{0, 1},
		},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0}},
	}

	result := BuildPolityProfiles(cells, polities, network, trade, nil, nil, nil, nil, catalog)
	if len(result.Assignments) != 2 {
		t.Fatalf("expected 2 profile assignments, got %d", len(result.Assignments))
	}
	if result.Assignments[0].Profile.Name == "" || result.Assignments[1].Profile.Name == "" {
		t.Fatalf("expected named resolved profiles")
	}
	if len(result.Attitudes) != 2 {
		t.Fatalf("expected directed attitudes for both pairs, got %d", len(result.Attitudes))
	}
}

func TestScaledAffinityScoreCompressesAutoAlliance(t *testing.T) {
	if got := scaledAffinityScore(1.71); got >= 0.55 {
		t.Fatalf("expected compressed affinity below allied threshold, got %.3f", got)
	}
}

func TestPolityCompetitionPenaltyRewardsSharedTradeLessThanSharedRivalNiche(t *testing.T) {
	from := PolityProfileAssignment{
		Profile:         ResolvedProfile{AncestryName: "Human", CultureName: "River Confederacy"},
		ContextTags:     []string{"polity", "mercantile", "agrarian"},
		EnvironmentTags: []string{"surface", "river", "floodplain", "alluvial"},
	}
	to := PolityProfileAssignment{
		Profile:         ResolvedProfile{AncestryName: "Human", CultureName: "River Confederacy"},
		ContextTags:     []string{"polity", "mercantile", "agrarian"},
		EnvironmentTags: []string{"surface", "river", "floodplain", "alluvial"},
	}
	withCompetition := polityCompetitionPenalty(from, to, 0.24, 0.0)
	withTradeBuffer := polityCompetitionPenalty(from, to, 0.24, 0.24)
	if withCompetition <= 0 {
		t.Fatalf("expected bordering same-niche polities to incur competition penalty")
	}
	if !(withTradeBuffer < withCompetition) {
		t.Fatalf("expected trade to soften competition, got no-trade=%.3f trade=%.3f", withCompetition, withTradeBuffer)
	}
}

func TestPolityTerritorialRivalryPenalizesCrowdedSameNicheBorders(t *testing.T) {
	from := PolityProfileAssignment{
		Profile: ResolvedProfile{
			AncestryName: "Human",
			CultureName:  "River Confederacy",
			Attitudes:    &ProfileAttitudeModule{Aggression: 0.34, HonorBias: 0.48, Xenophilia: 0.30},
			Traits:       map[string]float64{"warlike": 0.20},
		},
		ContextTags:     []string{"polity", "mercantile", "agrarian", "urban"},
		EnvironmentTags: []string{"surface", "river", "lowland", "floodplain", "alluvial"},
	}
	to := PolityProfileAssignment{
		Profile: ResolvedProfile{
			AncestryName: "Human",
			CultureName:  "River Confederacy",
			Attitudes:    &ProfileAttitudeModule{Aggression: 0.34, HonorBias: 0.48, Xenophilia: 0.30},
			Traits:       map[string]float64{"warlike": 0.20},
		},
		ContextTags:     []string{"polity", "mercantile", "agrarian", "urban"},
		EnvironmentTags: []string{"surface", "river", "lowland", "floodplain", "alluvial"},
	}
	fromSphere := PolitySphere{ID: 1, TerritoryCells: 86, MeanSupport: 0.22}
	toSphere := PolitySphere{ID: 2, TerritoryCells: 92, MeanSupport: 0.24}

	rivalry := polityTerritorialRivalryPenalty(fromSphere, toSphere, from, to, 0.36, 0, nil)
	buffered := polityTerritorialRivalryPenalty(fromSphere, toSphere, from, to, 0.36, 0.32, nil)
	noBorder := polityTerritorialRivalryPenalty(fromSphere, toSphere, from, to, 0, 0, nil)
	if rivalry < 0.40 {
		t.Fatalf("expected crowded same-niche border to create strong rivalry, got %.3f", rivalry)
	}
	if !(buffered < rivalry) {
		t.Fatalf("expected trade to soften rivalry, got no-trade=%.3f trade=%.3f", rivalry, buffered)
	}
	if noBorder != 0 {
		t.Fatalf("expected no rivalry without border pressure, got %.3f", noBorder)
	}
}

func TestClassifyPolityAttitudeLeavesModerateNegativesWary(t *testing.T) {
	if stance := classifyPolityAttitude(-0.20); stance != PolityAttitudeWary {
		t.Fatalf("expected moderate negative score to be wary, got %s", PolityAttitudeStanceName(stance))
	}
	if stance := classifyPolityAttitude(-0.60); stance != PolityAttitudeHostile {
		t.Fatalf("expected strong negative score to be hostile, got %s", PolityAttitudeStanceName(stance))
	}
}

func TestClassifyDetailedPolityAttitudeAllowsSuzeraintyAlliance(t *testing.T) {
	if stance := classifyDetailedPolityAttitude(0.12, 0.44, true); stance != PolityAttitudeAllied {
		t.Fatalf("expected strong suzerainty pair with modest positive score to classify allied, got %s", PolityAttitudeStanceName(stance))
	}
	if stance := classifyDetailedPolityAttitude(0.34, 0.42, true); stance != PolityAttitudeAllied {
		t.Fatalf("expected suzerainty pair with strong alliance bonus to classify allied, got %s", PolityAttitudeStanceName(stance))
	}
	if stance := classifyDetailedPolityAttitude(0.34, 0.10, true); stance == PolityAttitudeAllied {
		t.Fatalf("expected weak suzerainty bonus not to force alliance")
	}
}

func TestClassifyDetailedPolityAttitudeAllowsStrongKinTradeAlliance(t *testing.T) {
	if stance := classifyDetailedPolityAttitude(0.53, 0.20, false); stance != PolityAttitudeAllied {
		t.Fatalf("expected strong non-suzerain alliance bonus to classify allied, got %s", PolityAttitudeStanceName(stance))
	}
	if stance := classifyDetailedPolityAttitude(0.49, 0.20, false); stance == PolityAttitudeAllied {
		t.Fatalf("expected sub-threshold trade-linked kin pair not to classify allied")
	}
}

func TestPolityAllianceBonusSupportsSuzerainty(t *testing.T) {
	relations := []PolitySphereRelation{{Kind: PolityRelationSuzerain, Overlord: 1, Subject: 2, Strength: 1.0}}
	from := PolityProfileAssignment{Profile: ResolvedProfile{AncestryName: "Human", CultureName: "River Confederacy"}}
	to := PolityProfileAssignment{Profile: ResolvedProfile{AncestryName: "Human", CultureName: "River Confederacy"}}
	fromSphere := PolitySphere{ID: 1, Style: ProtoCivilizationRiverine, River: true}
	toSphere := PolitySphere{ID: 2, Style: ProtoCivilizationRiverine, River: true}

	overlordBonus := polityAllianceBonus(1, 2, fromSphere, toSphere, from, to, 0.34, 0.12, 0.04, 0.01, relations)
	subjectBonus := polityAllianceBonus(2, 1, toSphere, fromSphere, to, from, 0.34, 0.12, 0.04, 0.01, relations)
	if !(overlordBonus > subjectBonus) {
		t.Fatalf("expected overlord->subject alliance bonus to exceed reverse direction, got %.3f <= %.3f", overlordBonus, subjectBonus)
	}
	if overlordBonus < 0.35 {
		t.Fatalf("expected suzerainty to provide a meaningful alliance bonus, got %.3f", overlordBonus)
	}
}

func TestPolityAllianceBonusRewardsStrongTradeLinkedKin(t *testing.T) {
	from := PolityProfileAssignment{Profile: ResolvedProfile{AncestryName: "Human", CultureName: "Merchant League"}}
	to := PolityProfileAssignment{Profile: ResolvedProfile{AncestryName: "Human", CultureName: "Merchant League"}}
	fromSphere := PolitySphere{ID: 1, Style: ProtoCivilizationMaritime, Coastal: true}
	toSphere := PolitySphere{ID: 2, Style: ProtoCivilizationMaritime, Coastal: true}

	strong := polityAllianceBonus(1, 2, fromSphere, toSphere, from, to, 0.44, 0.32, 0.03, 0.02, nil)
	weak := polityAllianceBonus(1, 2, fromSphere, toSphere, from, to, 0.28, 0.02, 0.03, 0.02, nil)
	contested := polityAllianceBonus(1, 2, fromSphere, toSphere, from, to, 0.44, 0.32, 0.22, 0.20, nil)
	if strong <= 0 {
		t.Fatalf("expected strong trade-linked kin polities to receive alliance bonus")
	}
	if weak != 0 {
		t.Fatalf("expected weak affinity/trade kin polities to receive no alliance bonus, got %.3f", weak)
	}
	if !(contested < strong) {
		t.Fatalf("expected contested borders to suppress alliance bonus, got contested=%.3f strong=%.3f", contested, strong)
	}
}

func TestPolityAllianceBonusRewardsComplementaryLowTensionRoles(t *testing.T) {
	from := PolityProfileAssignment{Profile: ResolvedProfile{AncestryName: "Human", CultureName: "Merchant League"}}
	to := PolityProfileAssignment{Profile: ResolvedProfile{AncestryName: "Human", CultureName: "River Confederacy"}}
	maritime := PolitySphere{ID: 1, Style: ProtoCivilizationMaritime, Coastal: true}
	river := PolitySphere{ID: 2, Style: ProtoCivilizationRiverine, River: true}

	bonus := polityAllianceBonus(1, 2, maritime, river, from, to, 0.38, 0.22, 0.04, 0.03, nil)
	contested := polityAllianceBonus(1, 2, maritime, river, from, to, 0.38, 0.22, 0.22, 0.20, nil)
	if bonus <= 0 {
		t.Fatalf("expected complementary river-maritime roles to receive alliance pressure")
	}
	if contested != 0 {
		t.Fatalf("expected high-tension complementary roles not to form alliance bonus, got %.3f", contested)
	}
}

func TestBuildPolityEnvironmentContextDistinguishesFloodplainFromMarsh(t *testing.T) {
	river := buildPolityEnvironmentContext(
		PolitySphere{Style: ProtoCivilizationRiverine, River: true},
		&SettlementNetworkResult{},
		nil,
		PolityEcologyMetrics{
			MeanAridity:    1.25,
			MeanWetland:    0.18,
			FloodplainFrac: 0.24,
		},
	)
	if !hasProfileTag(river.Tags, "floodplain") || !hasProfileTag(river.Tags, "alluvial") || !hasProfileTag(river.Tags, "agrarian") {
		t.Fatalf("expected floodplain river polity tags, got %v", river.Tags)
	}
	if hasProfileTag(river.Tags, "wetland") || hasProfileTag(river.Tags, "marsh") || hasProfileTag(river.Tags, "delta") {
		t.Fatalf("did not expect marsh/delta tags for ordinary floodplain polity, got %v", river.Tags)
	}

	marsh := buildPolityEnvironmentContext(
		PolitySphere{Style: ProtoCivilizationMaritime, Coastal: true},
		&SettlementNetworkResult{},
		nil,
		PolityEcologyMetrics{
			MeanWetland:      0.40,
			WetlandBiomeFrac: 0.26,
			DeltaFrac:        0.12,
			LakeFrac:         0.10,
		},
	)
	if !hasProfileTag(marsh.Tags, "wetland") || !hasProfileTag(marsh.Tags, "marsh") || !hasProfileTag(marsh.Tags, "delta") {
		t.Fatalf("expected marsh polity tags, got %v", marsh.Tags)
	}
}

func TestAncestryPrevalenceAdjustmentDampsOnSpecialistMismatch(t *testing.T) {
	human := AncestryProfile{Name: "Human", BaselinePrevalence: 1.0, Tags: []string{"adaptable"}}
	lizardfolk := AncestryProfile{Name: "Lizardfolk", BaselinePrevalence: 0.14, Tags: []string{"wetland", "delta", "marsh"}}
	marshEnv := PolityEnvironmentContext{Tags: []string{"surface", "wetland", "marsh", "delta"}}
	generalEnv := PolityEnvironmentContext{Tags: []string{"surface", "river", "floodplain", "alluvial"}}

	humanGeneral := ancestryPrevalenceAdjustment(human, generalEnv)
	humanMarsh := ancestryPrevalenceAdjustment(human, marshEnv)
	lizardGeneral := ancestryPrevalenceAdjustment(lizardfolk, generalEnv)
	lizardMarsh := ancestryPrevalenceAdjustment(lizardfolk, marshEnv)

	if !(humanMarsh < humanGeneral) {
		t.Fatalf("expected human prevalence bonus to damp in marsh niche, got general=%.3f marsh=%.3f", humanGeneral, humanMarsh)
	}
	if lizardMarsh != lizardGeneral {
		t.Fatalf("expected specialist marsh ancestry prevalence term to remain unchanged in marsh niche, got general=%.3f marsh=%.3f", lizardGeneral, lizardMarsh)
	}
}

func TestExtractResolvedProfilesIncludesCompositionsAndAncestries(t *testing.T) {
	catalog := &ProfileCatalog{
		SchemaVersion: ProfileCatalogSchemaVersion,
		Compositions: []ProfileCompositionSpec{
			{Name: "Human + Merchant League", Ancestry: "Human", Culture: "Merchant League"},
		},
		Ancestries: []AncestryProfile{
			{Name: "Human"},
			{Name: "Elf"},
		},
		Cultures: []CultureProfile{
			{Name: "Merchant League"},
		},
	}
	profiles := ExtractResolvedProfiles(catalog)
	if len(profiles) != 3 {
		t.Fatalf("expected compositions plus ancestries, got %d profiles", len(profiles))
	}
}
