package climgen

import "testing"

func TestBuildProtoCivilizationsClaimsSupportedTerritory(t *testing.T) {
	sites := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 0.99, Y: 0.1, Z: 0},
		{X: 0.97, Y: 0.2, Z: 0},
		{X: 0.95, Y: 0.3, Z: 0},
		{X: 0.93, Y: 0.4, Z: 0},
		{X: 0.91, Y: 0.5, Z: 0},
	}
	cells := []VoronoiCell{
		{SiteIndex: 0, NeighborSiteIndices: []int32{1}},
		{SiteIndex: 1, NeighborSiteIndices: []int32{0, 2}},
		{SiteIndex: 2, NeighborSiteIndices: []int32{1, 3}},
		{SiteIndex: 3, NeighborSiteIndices: []int32{2, 4}},
		{SiteIndex: 4, NeighborSiteIndices: []int32{3, 5}},
		{SiteIndex: 5, NeighborSiteIndices: []int32{4}},
	}
	settlements := &SettlementResult{
		Classes: []SettlementClass{
			SettlementMarginal,
			SettlementFavorable,
			SettlementFavorable,
			SettlementFavorable,
			SettlementFavorable,
			SettlementMarginal,
		},
		Diagnostics: &SettlementDiagnostics{
			AccessScore:  []float64{0.55, 0.74, 0.80, 0.78, 0.74, 0.60},
			RiverBonus:   []float64{0.10, 0.35, 0.40, 0.42, 0.30, 0.10},
			CoastalBonus: []float64{0, 0, 0, 0, 0, 0},
			Suitability:  []float64{0.45, 0.72, 0.82, 0.80, 0.70, 0.48},
		},
	}
	population := &PopulationResult{
		Classes: []PopulationClass{
			PopulationSparseFrontier,
			PopulationRural,
			PopulationDenseRural,
			PopulationDenseRural,
			PopulationRural,
			PopulationSparseFrontier,
		},
		Diagnostics: &PopulationDiagnostics{
			CarryingCapacity: []float64{0.28, 0.48, 0.61, 0.60, 0.47, 0.24},
			UrbanPotential:   []float64{0.16, 0.36, 0.55, 0.54, 0.34, 0.14},
		},
	}
	soils := &SoilResult{Diagnostics: &SoilDiagnostics{
		Relief:    []float64{110, 130, 160, 170, 140, 120},
		Rockiness: []float64{0.12, 0.12, 0.16, 0.16, 0.12, 0.12},
	}}
	biomes := &BiomeResult{Diagnostics: &BiomeDiagnostics{
		AnnualIceFraction: []float64{0, 0, 0, 0, 0, 0},
		AridityRatio:      []float64{0.35, 0.32, 0.30, 0.30, 0.34, 0.38},
		WetlandAffinity:   []float64{0.10, 0.24, 0.34, 0.36, 0.22, 0.10},
	}}
	elevation := []float64{100, 100, 100, 100, 100, 100}

	network := BuildSettlementNetwork(sites, cells, settlements, population, biomes, soils, nil, elevation, 0, DefaultSettlementNetworkSettings())
	settings := DefaultProtoCivilizationSettings()
	settings.MinRegionAnchors = 1
	settings.MinTerritoryCells = 3
	result := BuildProtoCivilizations(cells, network, settlements, population, biomes, soils, elevation, 0, settings)
	if len(result.Civilizations) == 0 {
		t.Fatalf("expected at least one proto-civilization")
	}
	if result.Civilizations[0].TerritoryCells == 0 {
		t.Fatalf("expected claimed territory for first proto-civilization")
	}
	if result.Civilizations[0].Style != ProtoCivilizationRiverine {
		t.Fatalf("expected riverine civilization style, got %s", ProtoCivilizationStyleName(result.Civilizations[0].Style))
	}
}

func TestEligibleProtoCivilizationRegionRequiresRegionalAnchorCluster(t *testing.T) {
	settings := DefaultProtoCivilizationSettings()
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, Kind: SettlementNodeTown, Score: 0.62, River: true},
			{ID: 1, Kind: SettlementNodeVillage, Score: 0.58, River: true},
			{ID: 2, Kind: SettlementNodeVillage, Score: 0.58, River: true},
			{ID: 3, Kind: SettlementNodeVillage, Score: 0.58, River: true},
			{ID: 4, Kind: SettlementNodeVillage, Score: 0.58, River: true},
			{ID: 5, Kind: SettlementNodeVillage, Score: 0.58, River: true},
			{ID: 6, Kind: SettlementNodeVillage, Score: 0.58, River: true},
		},
	}

	isolatedTown := SettlementRegion{NodeIndices: []int{0}, CenterNode: 0, MeanScore: 0.62, River: true}
	if eligibleProtoCivilizationRegion(isolatedTown, network, settings) {
		t.Fatalf("expected isolated town peak to remain an outpost rather than a proto-civilization")
	}
	townWithVillageSatellites := SettlementRegion{NodeIndices: []int{0, 1, 2}, CenterNode: 0, MeanScore: 0.59, River: true}
	if eligibleProtoCivilizationRegion(townWithVillageSatellites, network, settings) {
		t.Fatalf("expected one town plus village satellites to remain below regional-anchor threshold")
	}
	twoTownsWithVillageSatellite := SettlementRegion{NodeIndices: []int{0, 1, 7}, CenterNode: 0, MeanScore: 0.59, River: true}
	network.Nodes = append(network.Nodes, SettlementNode{ID: 7, Kind: SettlementNodeTown, Score: 0.60, River: true})
	if !eligibleProtoCivilizationRegion(twoTownsWithVillageSatellite, network, settings) {
		t.Fatalf("expected two towns plus a village satellite to qualify")
	}
	villageCluster := SettlementRegion{NodeIndices: []int{1, 2, 3}, CenterNode: 1, MeanScore: 0.58, River: true}
	if eligibleProtoCivilizationRegion(villageCluster, network, settings) {
		t.Fatalf("expected small village cluster below town rank to remain an outpost")
	}
	network.Nodes = append(network.Nodes, SettlementNode{ID: 8, Kind: SettlementNodeHamlet, Score: 0.58, River: true})
	villagesPaddedByLocal := SettlementRegion{NodeIndices: []int{1, 2, 3, 4, 5, 8}, CenterNode: 1, MeanScore: 0.58, River: true}
	if eligibleProtoCivilizationRegion(villagesPaddedByLocal, network, settings) {
		t.Fatalf("expected district-centered broad cluster padded by local anchors to remain an outpost")
	}
	largeVillageCluster := SettlementRegion{NodeIndices: []int{1, 2, 3, 4, 5, 6}, CenterNode: 1, MeanScore: 0.58, River: true}
	if !eligibleProtoCivilizationRegion(largeVillageCluster, network, settings) {
		t.Fatalf("expected broad village cluster to qualify as a proto-civilization")
	}
}

func TestEligibleProtoCivilizationRegionRequiresPhysicalRegionalSupport(t *testing.T) {
	const width, height = 64, 64
	cellCount := 40962
	cells := hexLatticeMesh(width, height, cellCount)
	nodeCells := []int{
		hexLatticeIndex(width, height, 10, 10),
		hexLatticeIndex(width, height, 30, 10),
		hexLatticeIndex(width, height, 10, 30),
	}
	nodeCoords := [][2]int{{10, 10}, {30, 10}, {10, 30}}
	population := &PopulationResult{Diagnostics: &PopulationDiagnostics{
		CarryingCapacity: make([]float64, cellCount),
		UrbanPotential:   make([]float64, cellCount),
	}}
	settings := DefaultProtoCivilizationSettings()
	networkSettings := DefaultSettlementNetworkSettings()
	for _, idx := range nodeCells {
		population.Diagnostics.CarryingCapacity[idx] = networkSettings.TownThreshold + 0.03
		population.Diagnostics.UrbanPotential[idx] = networkSettings.TownThreshold + 0.03
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: nodeCells[0], Kind: SettlementNodeTown, Score: 0.62, River: true},
			{ID: 1, CellIndex: nodeCells[1], Kind: SettlementNodeTown, Score: 0.60, River: true},
			{ID: 2, CellIndex: nodeCells[2], Kind: SettlementNodeTown, Score: 0.60, River: true},
		},
	}
	region := SettlementRegion{NodeIndices: []int{0, 1, 2}, CenterNode: 0, MeanScore: 0.60, River: true}

	ok, reason := EligibleProtoCivilizationRegionWithPhysicalSupport(region, network, cells, population, settings, networkSettings)
	if ok || reason != "regional-support" {
		t.Fatalf("expected weak refined regional anchors to fail physical support, ok=%v reason=%s", ok, reason)
	}

	for _, coord := range nodeCoords {
		for _, supportCell := range hexLatticeNeighborhood(width, height, coord[0], coord[1], 1, nil) {
			population.Diagnostics.CarryingCapacity[supportCell] = networkSettings.TownThreshold + 0.03
			population.Diagnostics.UrbanPotential[supportCell] = networkSettings.TownThreshold + 0.03
		}
	}
	ok, reason = EligibleProtoCivilizationRegionWithPhysicalSupport(region, network, cells, population, settings, networkSettings)
	if !ok || reason != "anchor-kind" {
		t.Fatalf("expected a physically supported regional anchor cluster to qualify, ok=%v reason=%s", ok, reason)
	}
}

func TestEligibleProtoCivilizationRegionAllowsCompactHighRankSupport(t *testing.T) {
	cellCount := 40962
	cells := make([]VoronoiCell, cellCount)
	for i := range cells {
		cells[i].SiteIndex = int32(i)
	}
	population := &PopulationResult{Diagnostics: &PopulationDiagnostics{
		CarryingCapacity: make([]float64, cellCount),
		UrbanPotential:   make([]float64, cellCount),
	}}
	settings := DefaultProtoCivilizationSettings()
	networkSettings := DefaultSettlementNetworkSettings()
	nodes := []SettlementNode{
		{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, Score: 0.56, River: true},
		{ID: 1, CellIndex: 10, Kind: SettlementNodeVillage, Score: 0.54, River: true},
		{ID: 2, CellIndex: 20, Kind: SettlementNodeVillage, Score: 0.53, River: true},
		{ID: 3, CellIndex: 30, Kind: SettlementNodeVillage, Score: 0.52, River: true},
	}
	for _, node := range nodes {
		cells[node.CellIndex].NeighborSiteIndices = []int32{int32(node.CellIndex + 1), int32(node.CellIndex + 2)}
		cells[node.CellIndex+1].NeighborSiteIndices = []int32{int32(node.CellIndex)}
		cells[node.CellIndex+2].NeighborSiteIndices = []int32{int32(node.CellIndex)}
		switch node.Kind {
		case SettlementNodeTown:
			population.Diagnostics.CarryingCapacity[node.CellIndex] = networkSettings.TownThreshold + 0.03
			population.Diagnostics.UrbanPotential[node.CellIndex] = networkSettings.TownThreshold + 0.03
			population.Diagnostics.CarryingCapacity[node.CellIndex+1] = networkSettings.TownThreshold + 0.03
			population.Diagnostics.UrbanPotential[node.CellIndex+1] = networkSettings.TownThreshold + 0.03
		case SettlementNodeVillage:
			population.Diagnostics.CarryingCapacity[node.CellIndex] = networkSettings.VillageThreshold + 0.03
			population.Diagnostics.UrbanPotential[node.CellIndex] = networkSettings.VillageThreshold
			population.Diagnostics.CarryingCapacity[node.CellIndex+1] = networkSettings.VillageThreshold + 0.03
			population.Diagnostics.CarryingCapacity[node.CellIndex+2] = networkSettings.VillageThreshold + 0.03
		}
	}
	network := &SettlementNetworkResult{Nodes: nodes}
	region := SettlementRegion{NodeIndices: []int{0, 1, 2, 3}, CenterNode: 0, MeanScore: 0.54, River: true}

	ok, reason := EligibleProtoCivilizationRegionWithPhysicalSupport(region, network, cells, population, settings, networkSettings)
	if !ok || reason != "anchor-kind" {
		t.Fatalf("expected compact high-rank cluster with supported regional anchor to qualify, ok=%v reason=%s", ok, reason)
	}
}

func TestEligibleProtoCivilizationRegionUsesAreaSupportForSparseCoarseAnchors(t *testing.T) {
	cellCount := 40962
	cells := make([]VoronoiCell, cellCount)
	for i := range cells {
		cells[i].SiteIndex = int32(i)
	}
	population := &PopulationResult{Diagnostics: &PopulationDiagnostics{
		CarryingCapacity: make([]float64, cellCount),
		UrbanPotential:   make([]float64, cellCount),
	}}
	settings := DefaultProtoCivilizationSettings()
	networkSettings := DefaultSettlementNetworkSettings()
	nodes := []SettlementNode{
		{ID: 0, CellIndex: 0, Kind: SettlementNodeTown, Score: 0.56, River: true},
		{ID: 1, CellIndex: 100, Kind: SettlementNodeVillage, Score: 0.51, River: true},
		{ID: 2, CellIndex: 200, Kind: SettlementNodeHamlet, Score: 0.49, River: true},
	}
	townSupport := make([]int32, 0, 40)
	for supportCell := 1; supportCell <= 40; supportCell++ {
		townSupport = append(townSupport, int32(supportCell))
		cells[supportCell].NeighborSiteIndices = []int32{0}
		population.Diagnostics.CarryingCapacity[supportCell] = networkSettings.TownThreshold + 0.03
		population.Diagnostics.UrbanPotential[supportCell] = networkSettings.TownThreshold + 0.03
	}
	cells[0].NeighborSiteIndices = townSupport
	population.Diagnostics.CarryingCapacity[0] = networkSettings.TownThreshold + 0.03
	population.Diagnostics.UrbanPotential[0] = networkSettings.TownThreshold + 0.03

	villageSupport := make([]int32, 0, 4)
	for supportCell := 101; supportCell <= 104; supportCell++ {
		villageSupport = append(villageSupport, int32(supportCell))
		cells[supportCell].NeighborSiteIndices = []int32{100}
		population.Diagnostics.CarryingCapacity[supportCell] = networkSettings.VillageThreshold + 0.03
	}
	cells[100].NeighborSiteIndices = villageSupport
	population.Diagnostics.CarryingCapacity[100] = networkSettings.VillageThreshold + 0.03

	network := &SettlementNetworkResult{Nodes: nodes}
	region := SettlementRegion{NodeIndices: []int{0, 1, 2}, CenterNode: 0, MeanScore: 0.52, River: true}

	ok, reason := EligibleProtoCivilizationRegionWithPhysicalSupport(region, network, cells, population, settings, networkSettings)
	if !ok || reason != "area-support" {
		t.Fatalf("expected area-supported sparse coarse anchors to qualify, ok=%v reason=%s", ok, reason)
	}
}

func TestEligibleBroadProtoCivilizationRegionUsesPhysicalStrength(t *testing.T) {
	const width, height = 64, 64
	cellCount := 40962
	cells := hexLatticeMesh(width, height, cellCount)
	population := &PopulationResult{Diagnostics: &PopulationDiagnostics{
		CarryingCapacity: make([]float64, cellCount),
		UrbanPotential:   make([]float64, cellCount),
	}}
	protoSettings := DefaultProtoCivilizationSettings()
	networkSettings := DefaultSettlementNetworkSettings()
	nodes := make([]SettlementNode, 0, 6)
	nodeCoords := make([][2]int, 0, 6)
	for i := 0; i < 6; i++ {
		coord := [2]int{5 + (i%3)*20, 5 + (i/3)*20}
		nodeCoords = append(nodeCoords, coord)
		nodeCell := hexLatticeIndex(width, height, coord[0], coord[1])
		population.Diagnostics.CarryingCapacity[nodeCell] = networkSettings.VillageThreshold + 0.03
		population.Diagnostics.UrbanPotential[nodeCell] = networkSettings.VillageThreshold
		nodes = append(nodes, SettlementNode{ID: i, CellIndex: nodeCell, Kind: SettlementNodeVillage, Score: 0.58, River: true})
	}
	network := &SettlementNetworkResult{Nodes: nodes}
	region := SettlementRegion{
		NodeIndices: []int{0, 1, 2, 3, 4, 5},
		CenterNode:  0,
		MeanScore:   0.58,
		River:       true,
	}

	ok, reason := EligibleProtoCivilizationRegionWithPhysicalSupport(region, network, cells, population, protoSettings, networkSettings)
	if ok || reason != "broad-strength" {
		t.Fatalf("expected raw broad cluster without equivalent physical strength to fail, ok=%v reason=%s", ok, reason)
	}

	for _, coord := range nodeCoords {
		for _, supportCell := range hexLatticeNeighborhood(width, height, coord[0], coord[1], 1, nil) {
			population.Diagnostics.CarryingCapacity[supportCell] = networkSettings.VillageThreshold + 0.03
			population.Diagnostics.UrbanPotential[supportCell] = networkSettings.VillageThreshold
		}
	}
	ok, reason = EligibleProtoCivilizationRegionWithPhysicalSupport(region, network, cells, population, protoSettings, networkSettings)
	if !ok || reason != "broad-cluster" {
		t.Fatalf("expected physically supported broad cluster to qualify, ok=%v reason=%s", ok, reason)
	}
}

func TestProtoClaimAnchorStrengthUsesPhysicalSupport(t *testing.T) {
	const width, height = 64, 64
	cellCount := 40962
	cells := hexLatticeMesh(width, height, cellCount)
	nodeCoords := [][2]int{{10, 10}, {30, 10}, {10, 30}}
	population := &PopulationResult{Diagnostics: &PopulationDiagnostics{
		CarryingCapacity: make([]float64, cellCount),
		UrbanPotential:   make([]float64, cellCount),
	}}
	settings := DefaultSettlementNetworkSettings()
	nodeCells := make([]int, 0, len(nodeCoords))
	for _, coord := range nodeCoords {
		nodeCell := hexLatticeIndex(width, height, coord[0], coord[1])
		nodeCells = append(nodeCells, nodeCell)
		// Only the anchor cell and a single neighbour are habitable: a refined
		// spike, not a physically supported regional catchment.
		population.Diagnostics.CarryingCapacity[nodeCell] = settings.TownThreshold + 0.03
		population.Diagnostics.UrbanPotential[nodeCell] = settings.TownThreshold + 0.03
		neighbor := int(cells[nodeCell].NeighborSiteIndices[0])
		population.Diagnostics.CarryingCapacity[neighbor] = settings.TownThreshold + 0.03
		population.Diagnostics.UrbanPotential[neighbor] = settings.TownThreshold + 0.03
	}
	network := &SettlementNetworkResult{
		Nodes: []SettlementNode{
			{ID: 0, CellIndex: nodeCells[0], Kind: SettlementNodeTown},
			{ID: 1, CellIndex: nodeCells[1], Kind: SettlementNodeTown},
			{ID: 2, CellIndex: nodeCells[2], Kind: SettlementNodeTown},
		},
	}
	region := SettlementRegion{NodeIndices: []int{0, 1, 2}, CenterNode: 0}

	raw := ProtoCivilizationRegionAnchorStrength(region, network)
	physical := ProtoCivilizationRegionPhysicalAnchorStrength(region, network, cells, population, settings)
	claim := protoRegionClaimAnchorStrength(region, network, cells, population, settings)

	if raw != 3 {
		t.Fatalf("expected raw strength to count all refined anchors, got %.2f", raw)
	}
	if !(physical > 0 && physical < raw) {
		t.Fatalf("expected physical strength to discount weak refined anchors, raw=%.2f physical=%.2f", raw, physical)
	}
	if claim != physical {
		t.Fatalf("expected claim strength to use physical strength, got claim=%.2f physical=%.2f", claim, physical)
	}
}

func TestFinalizeProtoCivilizationsUsesEquivalentTerritoryArea(t *testing.T) {
	cellCount := 40962
	diagnostics := &ProtoCivilizationDiagnostics{
		CivilizationByCell: make([]int, cellCount),
		ClaimCost:          make([]float64, cellCount),
	}
	for i := range diagnostics.CivilizationByCell {
		diagnostics.CivilizationByCell[i] = -1
	}
	for i := 0; i < 20; i++ {
		diagnostics.CivilizationByCell[i] = 0
	}
	population := &PopulationResult{Diagnostics: &PopulationDiagnostics{CarryingCapacity: make([]float64, cellCount)}}
	settings := DefaultProtoCivilizationSettings()
	settings.MinTerritoryCells = 12

	filtered := finalizeProtoCivilizations([]ProtoCivilization{{ID: 0}}, diagnostics, population, settings)
	if len(filtered) != 0 {
		t.Fatalf("expected high-resolution twenty-cell fragment to be filtered by equivalent area, got %d civs", len(filtered))
	}
}
