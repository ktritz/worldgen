package terrain

// Hotspot types and configuration constants
// Hotspots are fixed magma sources in the mantle. As plates move over them,
// they create chains of volcanic islands (Hawaii, Galápagos) or caldera tracks (Yellowstone).

// Hotspot represents a fixed volcanic source and its island chain
type Hotspot struct {
	Position  Vector3D // Fixed position on unit sphere
	PlateID   int      // Plate the hotspot is currently under
	IsOceanic bool     // Whether the plate is oceanic
}

// HotspotIsland represents a single island in a hotspot chain
type HotspotIsland struct {
	CellIndex int      // Index of the cell containing this island
	Position  Vector3D // Physical island position on the unit sphere
	Age       float64  // Normalized age: 0 = newest (active), 1 = oldest
	Strength  float64  // Relative eruption strength for this position in the chain
}

// HotspotChain represents a complete island chain from a hotspot
type HotspotChain struct {
	ID        int
	Hotspot   Hotspot
	Islands   []HotspotIsland
	IsOceanic bool // Whether this is an oceanic (island) or continental (caldera) chain
}

// HotspotCellInfo tracks which cells were modified by hotspots and their minimum elevation
type HotspotCellInfo struct {
	IsOceanic    bool    // Oceanic island or continental caldera
	MinElevation float64 // Minimum elevation to preserve after erosion
}

// Hotspot configuration constants
const (
	// Number of hotspots per planet (Earth has ~50 active hotspots, ~10-20 notable ones)
	HotspotsPerPlanet = 15

	// Island spacing in radians (~150km = 0.024 radians on Earth)
	IslandSpacingRadians = 0.024

	// Minimum physical radius for a hotspot feature to become an emergent land
	// component on the global mesh. Smaller volcanic events remain seamounts or
	// submerged shoals so fine meshes do not materialize one-cell islands that a
	// coarser mesh cannot physically resolve.
	MinEmergentHotspotIslandRadius = IslandSpacingRadians * 0.45

	// Base maximum chain length in radians (~4000km = 0.63 radians)
	// Actual length varies with plate velocity (faster = longer chain)
	// Longer chains show more curvature
	BaseChainLengthRadians = 0.63

	// Minimum chain length even for slow plates (~1000km)
	MinChainLengthRadians = 0.16

	// Minimum chain length to be visible (lowered to allow shorter chains)
	MinChainLength = 2

	// Elevation boost for hotspot islands (normalized units, applied before hypsometry)
	// Ocean floor varies from -0.7 to -1.0, so we need ~1.5 boost for islands to rise clearly
	// Peak islands should reach ~+0.5 (maps to good land elevation)
	HotspotElevationBoost = 1.5

	// Continental hotspot parameters (resolution-independent, in radians)
	// Yellowstone caldera is ~70km x 45km, so radius ~35-40km = 0.006 radians
	// We use slightly larger to account for the full volcanic field
	ContinentalCalderaRadius = 0.012 // ~75km radius, ~150km diameter

	// Continental hotspot elevations (in meters)
	ContinentalPeakElevation = 2800.0 // Yellowstone Plateau is ~2400m
	ContinentalSubsidence    = 350.0  // Meters below neighbor average for old track

	// Continental lifecycle timing
	ContinentalPeakAge       = 0.12 // Peak activity (caldera phase)
	ContinentalSubsidenceAge = 0.35 // Age at which subsidence begins

	// Underwater continental hotspot parameters
	// Continental plates with underwater areas (shelves, flooded margins) create
	// smaller volcanic islands than true oceanic hotspots due to thicker crust
	// Reference: Kerguelen Islands (~1850m), Heard Island (~2745m)
	UnderwaterContinentalPeakElevation = 1800.0 // Max peak for underwater continental hotspots
	UnderwaterContinentalMaxSpread     = 1      // Max spread rings (smaller than oceanic)
)
