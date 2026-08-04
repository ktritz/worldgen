package climgen

import "testing"

func TestSeasonalOceanEvaporationFactorTracksTemperature(t *testing.T) {
	cold := seasonalOceanEvaporationFactor(271.15, 0.45)
	temperate := seasonalOceanEvaporationFactor(288.15, 0.45)
	warm := seasonalOceanEvaporationFactor(303.15, 0.45)

	if !(cold < temperate && temperate < warm) {
		t.Fatalf(
			"expected evaporation factor to increase with ocean temperature: cold=%.3f temperate=%.3f warm=%.3f",
			cold, temperate, warm,
		)
	}
	if cold < 0.45 || warm > 1.45 {
		t.Fatalf("expected bounded evaporation factor, got cold=%.3f warm=%.3f", cold, warm)
	}
}

func TestPrecipitationSnowFractionTracksTemperature(t *testing.T) {
	if snow := precipitationSnowFraction([]float64{268.15}, 0); snow < 0.99 {
		t.Fatalf("expected fully snowy precipitation below freezing, got %.3f", snow)
	}
	if snow := precipitationSnowFraction([]float64{278.15}, 0); snow > 0.01 {
		t.Fatalf("expected mostly rain above warm threshold, got %.3f", snow)
	}

	mixed := precipitationSnowFraction([]float64{273.15}, 0)
	if mixed <= 0 || mixed >= 1 {
		t.Fatalf("expected mixed rain/snow near freezing, got %.3f", mixed)
	}
}

func TestConvectiveCondensationPotentialFavorsWarmHumidInteriors(t *testing.T) {
	coolCoast := computeConvectiveCondensationPotential(0.70, 0.95, 8.0, 0.15, 0.05)
	warmInterior := computeConvectiveCondensationPotential(0.95, 0.95, 28.0, 0.40, 0.95)

	if warmInterior <= coolCoast {
		t.Fatalf("expected warm humid interior convection to exceed cool coast: coast=%.3f interior=%.3f", coolCoast, warmInterior)
	}

	dryInterior := computeConvectiveCondensationPotential(0.20, 0.95, 28.0, 0.40, 0.95)
	if dryInterior >= warmInterior {
		t.Fatalf("expected dry interior to convect less than humid interior: dry=%.3f humid=%.3f", dryInterior, warmInterior)
	}
}

func TestLandRetainedHumidityRespondsToBlockingAndInterior(t *testing.T) {
	coastalOpen := computeLandRetainedHumidity(1.0, 0.78, 0.60, 0.05, 0.90, 0.90, 0.10, 0.10, 0.85, 1.0)
	coastalBarrier := computeLandRetainedHumidity(1.0, 0.78, 0.60, 0.95, 0.90, 0.90, 0.10, 0.10, 0.85, 1.0)
	corridor := computeLandRetainedHumidity(1.0, 0.78, 0.60, 0.10, 0.90, 0.90, 0.35, 0.55, 0.85, 1.0)

	if coastalBarrier >= coastalOpen {
		t.Fatalf("expected stronger blocking to retain less humidity: open=%.3f barrier=%.3f", coastalOpen, coastalBarrier)
	}
	if corridor <= coastalOpen {
		t.Fatalf("expected near-inland storm corridor to retain more humidity than immediate coast: coast=%.3f corridor=%.3f", coastalOpen, corridor)
	}
}

func TestLandCondensationIsLessEfficientInVeryColdAir(t *testing.T) {
	warmCapacity := 0.70
	coldCapacity := 0.20
	warm := computeLandCondensation(warmCapacity*1.15, warmCapacity, 0.70, 0.30, 0.90, 0.90, 0.15, 0.10, 0.90, 1.0, 0.07, []float64{281.15}, 0)
	cold := computeLandCondensation(coldCapacity*1.15, coldCapacity, 0.70, 0.30, 0.90, 0.90, 0.15, 0.10, 0.90, 1.0, 0.07, []float64{258.15}, 0)

	if cold >= warm {
		t.Fatalf("expected very cold maritime air to condense less than warm maritime air: cold=%.3f warm=%.3f", cold, warm)
	}
}

func TestLandCondensationShiftsSomeMaritimeRainInland(t *testing.T) {
	capacity := 0.70
	coast := computeLandCondensation(capacity*1.30, capacity, 0.05, 0.20, 0.90, 0.90, 0.05, 0.05, 0.95, 1.0, 0.07, []float64{282.15}, 0)
	inland := computeLandCondensation(capacity*1.30, capacity, 0.05, 0.20, 0.90, 0.90, 0.55, 0.20, 0.95, 1.0, 0.07, []float64{282.15}, 0)

	if inland <= coast {
		t.Fatalf("expected frontal inland cell to condense more than first-landfall coast under weak uplift: coast=%.3f inland=%.3f", coast, inland)
	}
}

func TestLandCondensationDampsUnsupportedSupersaturation(t *testing.T) {
	capacity := 0.80
	unsupported := computeLandCondensationDiagnostic(capacity*1.55, capacity, 0.02, 0.10, 0.00, 0.00, 0.45, 0.75, 0.55, 1.0, 0.07, []float64{301.15}, 0)
	supported := computeLandCondensationDiagnostic(capacity*1.55, capacity, 0.20, 0.55, 0.90, 0.85, 0.15, 0.20, 0.85, 1.0, 0.07, []float64{301.15}, 0)

	if unsupported.SupersatSupport >= supported.SupersatSupport {
		t.Fatalf("expected unsupported tropical supersaturation to have lower support: unsupported=%.3f supported=%.3f", unsupported.SupersatSupport, supported.SupersatSupport)
	}
	if unsupported.SupersatCondensation >= supported.SupersatCondensation {
		t.Fatalf("expected unsupported tropical supersaturation to condense less: unsupported=%.3f supported=%.3f", unsupported.SupersatCondensation, supported.SupersatCondensation)
	}
}

func TestLandCondensationSupportsWarmOnshoreTropicalCoast(t *testing.T) {
	capacity := 1.20
	coolOffshore := computeLandCondensationDiagnostic(capacity*1.05, capacity, 0.02, 0.20, 0.90, 0.05, 0.00, 0.00, 0.90, 1.0, 0.07, []float64{285.15}, 0)
	warmOnshore := computeLandCondensationDiagnostic(capacity*1.05, capacity, 0.02, 0.65, 0.95, 0.95, 0.00, 0.00, 0.90, 1.0, 0.07, []float64{301.15}, 0)

	if warmOnshore.TropicalCoastSupport <= coolOffshore.TropicalCoastSupport {
		t.Fatalf("expected warm onshore tropical coast support to exceed cool offshore support: cool=%.3f warm=%.3f", coolOffshore.TropicalCoastSupport, warmOnshore.TropicalCoastSupport)
	}
	warmRawPenalty := (0.22 + 0.20*0.90) * (0.95 * 0.95)
	if warmOnshore.CoastalPenalty >= warmRawPenalty {
		t.Fatalf("expected warm onshore tropical coast penalty to be reduced below raw baseline: raw=%.3f got=%.3f", warmRawPenalty, warmOnshore.CoastalPenalty)
	}
	if warmOnshore.Condensed <= coolOffshore.Condensed {
		t.Fatalf("expected warm onshore tropical coast to condense more rainfall overall: cool=%.3f warm=%.3f", coolOffshore.Condensed, warmOnshore.Condensed)
	}
}

func TestMarineDominatedFlowRetainsMoreHumidityAlongTransportCorridor(t *testing.T) {
	coast := computeLandRetainedHumidity(1.0, 0.70, 0.60, 0.05, 0.90, 0.90, 0.10, 0.10, 0.95, 1.0)
	inlandMarine := computeLandRetainedHumidity(1.0, 0.70, 0.60, 0.05, 0.90, 0.90, 0.55, 0.25, 0.95, 1.0)
	inlandMixed := computeLandRetainedHumidity(1.0, 0.70, 0.60, 0.05, 0.90, 0.90, 0.55, 0.25, 0.25, 1.0)

	if inlandMarine <= coast {
		t.Fatalf("expected marine-dominated corridor to retain more humidity than immediate coast: coast=%.3f inland=%.3f", coast, inlandMarine)
	}
	if inlandMarine <= inlandMixed {
		t.Fatalf("expected stronger marine share to retain more corridor humidity: marine=%.3f mixed=%.3f", inlandMarine, inlandMixed)
	}
}

func TestMarineToLandMixingGrowsInland(t *testing.T) {
	coast := marineToLandMixFraction(0.90, 0.90, 0.05, 0.10)
	corridor := marineToLandMixFraction(0.90, 0.90, 0.45, 0.30)
	interior := marineToLandMixFraction(0.90, 0.90, 0.85, 0.70)

	if corridor <= coast {
		t.Fatalf("expected inland corridor to mix more marine moisture into land reservoir than immediate coast: coast=%.3f corridor=%.3f", coast, corridor)
	}
	if interior < corridor {
		t.Fatalf("expected deeper continental carry to sustain at least as much mixing as corridor: corridor=%.3f interior=%.3f", corridor, interior)
	}
}

func TestPrecipitationPerStepFractionScalesWithMeshResolution(t *testing.T) {
	base := precipitationPerStepFraction(0.20, 10242)
	fine := precipitationPerStepFraction(0.20, 163842)
	combinedFine := 1.0
	for i := 0; i < 4; i++ {
		combinedFine *= 1.0 - fine
	}
	combinedFine = 1.0 - combinedFine

	if fine >= base {
		t.Fatalf("expected finer mesh per-step fraction to be smaller: base=%.3f fine=%.3f", base, fine)
	}
	if combinedFine < base*0.98 || combinedFine > base*1.02 {
		t.Fatalf("expected four fine steps to approximate one base step: base=%.3f combinedFine=%.3f", base, combinedFine)
	}
}

func TestNeighborOceanFractionUsesPhysicalCoastalBand(t *testing.T) {
	elevation := []float64{100, 100, 100, -100, 100}
	adj := &FlatAdjacency{
		Neighbors: []int{1, 0, 2, 1, 3, 2, 4, 3},
		Offsets:   []int{0, 1, 3, 5, 7, 8},
	}

	nearCoast := computeNeighborOceanFraction(2, elevation, 0, adj)
	inland := computeNeighborOceanFraction(0, elevation, 0, adj)

	if nearCoast <= 0 {
		t.Fatalf("expected land cell within physical coastal band to see ocean support")
	}
	if inland >= nearCoast {
		t.Fatalf("expected coastal support to decay inland: inland=%.3f nearCoast=%.3f", inland, nearCoast)
	}
}

func TestMarineCorridorBlendWeightFavorsInlandStormPath(t *testing.T) {
	coast := marineCorridorBlendWeight(0.05)
	corridor := marineCorridorBlendWeight(0.45)
	deep := marineCorridorBlendWeight(0.90)

	if corridor <= coast {
		t.Fatalf("expected inland corridor blend to exceed immediate coastal blend: coast=%.3f corridor=%.3f", coast, corridor)
	}
	if deep >= corridor {
		t.Fatalf("expected deepest interior blend to stay below corridor peak: corridor=%.3f deep=%.3f", corridor, deep)
	}
}

func TestComputeUpwindLandStepCountsTracksDistanceToOcean(t *testing.T) {
	parent := []int{0, 0, 1, 2}
	strength := []float64{1.0, 0.9, 0.9, 0.9}
	elevation := []float64{-100, 100, 100, 100}

	steps := computeUpwindLandStepCounts(parent, strength, elevation, 0, 10)
	if steps[1] != 0 || steps[2] != 1 || steps[3] != 2 {
		t.Fatalf("expected coast/inland step counts 0/1/2, got %v", steps)
	}
}

func TestComputeOceanAtmosphericMoistureAccumulatesOverOcean(t *testing.T) {
	vertices := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 0, Y: 1, Z: 0},
	}
	adj := &FlatAdjacency{
		Neighbors: []int{1, 0},
		Offsets:   []int{0, 1, 2},
	}
	wind := []Vector3D{{X: 0, Y: 1, Z: 0}, {X: 0, Y: 1, Z: 0}}
	elevation := []float64{-100, -100}
	source := []float64{0.6, 0.6}
	capacity := []float64{1.0, 1.0}

	moisture, _ := computeOceanAtmosphericMoisture(vertices, elevation, 0, adj, wind, source, capacity, 0.07, 8)
	if moisture[1] <= source[1] {
		t.Fatalf("expected downwind ocean cell to accumulate more moisture than local source alone, got source=%.3f moisture=%.3f", source[1], moisture[1])
	}
}

func TestComputeCoastalMarineRetentionTargetsHotLandfallSources(t *testing.T) {
	mild := computeCoastalMarineRetention(0.60, 0.30, 0.40)
	hotLandfall := computeCoastalMarineRetention(0.95, 0.90, 0.95)

	if hotLandfall >= mild {
		t.Fatalf("expected hot landfall source retention to be lower than mild coastal source: mild=%.3f hot=%.3f", mild, hotLandfall)
	}
	if hotLandfall < 0.76 || mild > 1.0 {
		t.Fatalf("expected bounded retention values, got mild=%.3f hot=%.3f", mild, hotLandfall)
	}
}

func TestComputeWeightedUpwindDonorsUsesMultipleAlignedNeighbors(t *testing.T) {
	vertices := []Vector3D{
		{X: 0, Y: 0, Z: 1},
		{X: -0.7, Y: 0.0, Z: 0.7},
		{X: -0.6, Y: 0.3, Z: 0.7},
		{X: 0.6, Y: 0.0, Z: 0.7},
	}
	adj := &FlatAdjacency{
		Neighbors: []int{1, 2, 3},
		Offsets:   []int{0, 3, 3, 3, 3},
	}
	wind := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
	}

	donors, weights := computeWeightedUpwindDonors(0, vertices, adj, wind, 0.08)
	if len(donors) != 2 {
		t.Fatalf("expected two aligned upwind donors, got donors=%v weights=%v", donors, weights)
	}
	if donors[0] != 1 || donors[1] != 2 {
		t.Fatalf("expected westward neighbors as donors, got %v", donors)
	}
	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	if sum < 0.999 || sum > 1.001 {
		t.Fatalf("expected normalized donor weights, got %.3f", sum)
	}
	if weights[0] <= weights[1] {
		t.Fatalf("expected more aligned donor to carry more weight, got %v", weights)
	}
}

func TestMarineLandDiffusionFavorsInlandCorridorsOverImmediateCoast(t *testing.T) {
	vertices := []Vector3D{
		{X: 0, Y: 0, Z: 1},
		{X: 0.8, Y: 0.0, Z: 0.6},
		{X: 1.6, Y: 0.0, Z: 0.2},
	}
	adj := &FlatAdjacency{
		Neighbors: []int{1, 0, 2, 1},
		Offsets:   []int{0, 1, 3, 4},
	}
	wind := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
	}
	marine := []float64{10, 1, 0}
	elevation := []float64{100, 100, 100}
	oceanFetch := []float64{0.9, 0.9, 0.8}
	coastalOnshore := []float64{0.9, 0.9, 0.9}
	landTravel := []float64{0.05, 0.45, 0.85}
	landInterior := []float64{0.05, 0.35, 0.85}

	applyMarineLandDiffusion(marine, vertices, elevation, 0, adj, wind, oceanFetch, coastalOnshore, landTravel, landInterior)

	if marine[1] <= 1.0 {
		t.Fatalf("expected corridor cell to gain moisture from diffusion, got %.3f", marine[1])
	}
	if marine[0] >= 10.0 {
		t.Fatalf("expected coastal cell to diffuse some moisture inland, got %.3f", marine[0])
	}
	if marine[2] <= 0.0 {
		t.Fatalf("expected deeper inland cell to receive some diffused moisture, got %.3f", marine[2])
	}
}

func TestMarineDiffusionNeighborWeightFavorsAlongWindNeighbors(t *testing.T) {
	vertices := []Vector3D{
		{X: 0, Y: 0, Z: 1},
		{X: 0.8, Y: 0.0, Z: 0.6},
		{X: 0.0, Y: 0.8, Z: 0.6},
	}
	wind := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
	}

	along := marineDiffusionNeighborWeight(0, 1, vertices, wind)
	cross := marineDiffusionNeighborWeight(0, 2, vertices, wind)
	if along <= cross {
		t.Fatalf("expected along-wind diffusion weight to exceed cross-wind weight: along=%.3f cross=%.3f", along, cross)
	}
}

func TestMarineDiffusionRegimeFactorFavorsWesterlyMidlatitudes(t *testing.T) {
	vertices := []Vector3D{
		seasonalLatLonVertex(50, 0),
		seasonalLatLonVertex(25, 0),
		seasonalLatLonVertex(8, 0),
	}
	wind := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
	}

	midlat := marineDiffusionRegimeFactor(0, vertices, wind)
	subtropical := marineDiffusionRegimeFactor(1, vertices, wind)
	tropical := marineDiffusionRegimeFactor(2, vertices, wind)

	if midlat <= subtropical {
		t.Fatalf("expected westerly midlatitude regime to exceed subtropical regime: midlat=%.3f subtropical=%.3f", midlat, subtropical)
	}
	if midlat <= tropical {
		t.Fatalf("expected westerly midlatitude regime to exceed tropical regime: midlat=%.3f tropical=%.3f", midlat, tropical)
	}
}

func TestComputeTropicalMarineSourceFeedsInteriorSummerTropics(t *testing.T) {
	vertices := []Vector3D{
		seasonalLatLonVertex(12, -30),
		seasonalLatLonVertex(12, -20),
		seasonalLatLonVertex(12, -10),
		seasonalLatLonVertex(12, 0),
	}
	elevation := []float64{-100, 100, 100, 100}
	adj := &FlatAdjacency{
		Neighbors: []int{1, 0, 2, 1, 3, 2},
		Offsets:   []int{0, 1, 3, 5, 6},
	}
	wind := []Vector3D{
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
	}
	marine := []float64{0.9, 0.7, 0.4, 0.2}
	oceanFetch := []float64{0, 1.0, 0.8, 0.5}
	coastalOnshore := []float64{0, 0.9, 0.8, 0.6}
	landTravel := []float64{0, 0.0, 0.4, 0.8}
	landInterior := []float64{0, 0.1, 0.4, 0.8}
	tropicalScale := []float64{0, 1.0, 1.6, 1.8}

	near := computeTropicalMarineSource(2, marine, vertices, elevation, 0.0, adj, wind, oceanFetch, coastalOnshore, landTravel, landInterior, tropicalScale)
	deep := computeTropicalMarineSource(3, marine, vertices, elevation, 0.0, adj, wind, oceanFetch, coastalOnshore, landTravel, landInterior, tropicalScale)

	if near <= 0 {
		t.Fatalf("expected tropical source to feed first inland tropical cell, got %.3f", near)
	}
	if deep <= 0 {
		t.Fatalf("expected tropical source to reach deeper inland tropical cell, got %.3f", deep)
	}
}

func TestDeriveEffectiveMaritimeAccessDoesNotInventCoastalExposure(t *testing.T) {
	diag := deriveEffectiveMaritimeAccessDiagnostic(
		0.0,
		0.0,
		0.02,
		0.0,
		0.50,
		0.99,
		0.12,
	)

	if diag.EffectiveFetch > 0.12 {
		t.Fatalf("expected inland maritime fetch to stay limited without geometric support, got %.3f", diag.EffectiveFetch)
	}
	if diag.EffectiveOnshore > 0.05 {
		t.Fatalf("expected inland maritime onshore to stay limited without coastal geometry, got %.3f", diag.EffectiveOnshore)
	}
}

func TestDeriveEffectiveMaritimeAccessPreservesSupportedInlandMarinePath(t *testing.T) {
	diag := deriveEffectiveMaritimeAccessDiagnostic(
		0.0,
		0.0,
		0.35,
		0.0,
		0.10,
		0.25,
		0.36,
	)

	if diag.EffectiveFetch < 0.30 {
		t.Fatalf("expected inland maritime access to remain material with strong footprint support, got %.3f", diag.EffectiveFetch)
	}
	if diag.EffectiveOnshore <= 0.05 {
		t.Fatalf("expected some directional maritime exposure with supported inland path, got %.3f", diag.EffectiveOnshore)
	}
}

func TestMarineLandfallEntryScaleTargetsWarmImmediateCoasts(t *testing.T) {
	warmCoast := marineLandfallEntryScale(0.9, 0.0, 0.0, 28.0)
	mountainCoast := marineLandfallEntryScale(0.9, 1.0, 0.0, 28.0)
	inlandCorridor := marineLandfallEntryScale(0.9, 0.0, 0.5, 28.0)

	if warmCoast >= 1.0 {
		t.Fatalf("expected warm immediate coast to damp entry humidity, got %.3f", warmCoast)
	}
	if mountainCoast <= warmCoast {
		t.Fatalf("expected strong uplift to preserve more entry humidity than lowland coast: mountain=%.3f coast=%.3f", mountainCoast, warmCoast)
	}
	if inlandCorridor <= warmCoast {
		t.Fatalf("expected inland corridor to preserve more entry humidity than immediate coast: corridor=%.3f coast=%.3f", inlandCorridor, warmCoast)
	}
}
