package elevation

import (
	"worldgen/icosphere"
	"worldgen/landgen/tectonics"
)

// Type aliases for external dependencies
type Vector3D = icosphere.Vector3D
type TectonicPlate = tectonics.TectonicPlate
type TectonicsData = tectonics.TectonicsData
type MidOceanRidge = tectonics.MidOceanRidge
type Hotspot = tectonics.Hotspot
type VolcanicFeature = tectonics.VolcanicFeature
type SubductionZone = tectonics.SubductionZone
type SeafloorAgeModel = tectonics.SeafloorAgeModel

// ElevationSettings holds parameters for comprehensive elevation generation
type ElevationSettings struct {
	// Basic elevation parameters
	ElevationMultiplier           float64 `json:"elevationMultiplier"`           // Base elevation scaling factor
	GlobalSeed                    int64   `json:"globalSeed"`                    // Master seed for all elevation generation
	
	// Tectonic boundary effects
	CharacteristicFalloffDistance float64 `json:"characteristicFalloffDistance"` // e.g., 0.15 (15% of radius)
	MaxBoundaryEffectDistance     float64 `json:"maxBoundaryEffectDistance"`     // e.g., 0.45 (45% of radius)
	ConvergentBoundaryStrength    float64 `json:"convergentBoundaryStrength"`    // e.g., 2000 (meters)
	DivergentBoundaryStrength     float64 `json:"divergentBoundaryStrength"`     // e.g., 800 (meters, for rifting)
	
	// Ridge system effects
	EnableRidgeTopography         bool    `json:"enableRidgeTopography"`         // Whether to model ridge elevation
	RidgeElevationAboveSeafloor   float64 `json:"ridgeElevationAboveSeafloor"`   // Ridge elevation above surrounding seafloor (2000-3000m)
	RidgeInfluenceDistance        float64 `json:"ridgeInfluenceDistance"`        // Distance ridge effects extend (km)
	
	// Hotspot and volcanic effects  
	EnableVolcanicElevation       bool    `json:"enableVolcanicElevation"`       // Whether to model volcanic features
	VolcanicElevationMultiplier   float64 `json:"volcanicElevationMultiplier"`   // Scaling for volcanic feature heights
	HotspotInfluenceRadius        float64 `json:"hotspotInfluenceRadius"`        // Hotspot thermal uplift radius (km)
	
	// Seafloor age-depth modeling
	EnableSeafloorAgeDepth        bool    `json:"enableSeafloorAgeDepth"`        // Whether to use age-depth relationships
	SeafloorModel                 SeafloorAgeModel `json:"seafloorModel"`        // Age-depth model parameters
	
	// Subduction zone effects
	EnableSubductionEffects       bool    `json:"enableSubductionEffects"`       // Whether to model trenches and arcs
	TrenchDepthMultiplier         float64 `json:"trenchDepthMultiplier"`         // Scaling for trench depths
	VolcanicArcElevation          float64 `json:"volcanicArcElevation"`          // Base elevation for volcanic arcs (m)
	
	// Erosion and weathering
	EnableErosion                 bool    `json:"enableErosion"`                 // Whether to apply erosion modeling
	ErosionRate                   float64 `json:"erosionRate"`                   // Base erosion rate (m/Myr)
	MaxErosionAge                 float64 `json:"maxErosionAge"`                 // Maximum age for erosion calculation (Myr)
	
	// Fractal noise for terrain detail
	NoiseScale                    float64 `json:"noiseScale"`                    // Scale of fractal noise
	NoiseOctaves                  int     `json:"noiseOctaves"`                  // Number of noise octaves
	NoisePersistence              float64 `json:"noisePersistence"`              // Noise persistence (amplitude decay)
	NoiseLacunarity               float64 `json:"noiseLacunarity"`               // Noise lacunarity (frequency multiplier)
	NoiseAmplitude                float64 `json:"noiseAmplitude"`                // Maximum noise amplitude (m)
	
	// Isostatic adjustment
	EnableIsostasy                bool    `json:"enableIsostasy"`                // Whether to apply isostatic adjustment
	CrustalDensity                float64 `json:"crustalDensity"`                // Crustal density (kg/m³)
	MantleDensity                 float64 `json:"mantleDensity"`                 // Mantle density (kg/m³)
}

// ElevationData holds all generated elevation data
type ElevationData struct {
	// Primary elevation data
	SiteElevations                []float64               `json:"siteElevations"`                // Elevation at each icosphere site (m)
	CellElevations                map[int32]float64       `json:"cellElevations"`                // Keyed by site ID for compatibility
	
	// Component elevation contributions
	BaseElevations                []float64               `json:"baseElevations"`                // Base plate-type elevations
	TectonicElevations           []float64               `json:"tectonicElevations"`           // Boundary effect contributions
	VolcanicElevations           []float64               `json:"volcanicElevations"`           // Volcanic feature contributions
	SeafloorElevations           []float64               `json:"seafloorElevations"`           // Age-depth contributions
	RidgeElevations              []float64               `json:"ridgeElevations"`              // Ridge topography contributions
	ErosionElevations            []float64               `json:"erosionElevations"`            // Erosion modifications
	NoiseElevations              []float64               `json:"noiseElevations"`              // Fractal noise contributions
	
	// Validation and analysis data
	ElevationMetrics             ElevationValidationMetrics `json:"elevationMetrics"`          // Validation metrics
	HypsometricCurve             []HypsometricPoint      `json:"hypsometricCurve"`             // Elevation-area distribution
}

// ElevationParameters holds computed parameters for elevation generation
type ElevationParameters struct {
	// Basic parameters
	BaseAmplitude                float64 // Base elevation amplitude
	ElevationSeed                int64   // Seed for elevation generation
	PlanetRadius                 float64 // Planet radius for distance calculations
	
	// Tectonic boundary parameters
	CharacteristicFalloffDistAbs float64 // Absolute characteristic falloff distance
	MaxBoundaryEffectDistAbs     float64 // Absolute maximum boundary effect distance  
	ConvergentStrength           float64 // Strength of convergent boundary effects
	DivergentStrength            float64 // Strength of divergent boundary effects
	
	// Ridge system parameters
	RidgeElevation               float64 // Ridge elevation above seafloor
	RidgeInfluenceDistAbs        float64 // Absolute ridge influence distance
	
	// Volcanic parameters
	VolcanicMultiplier           float64 // Volcanic elevation scaling
	HotspotInfluenceRadiusAbs    float64 // Absolute hotspot influence radius
	
	// Seafloor parameters
	SeafloorModel                SeafloorAgeModel // Age-depth model
	
	// Subduction parameters
	TrenchDepthMultiplier        float64 // Trench depth scaling
	ArcElevation                 float64 // Volcanic arc base elevation
	
	// Erosion parameters
	ErosionRate                  float64 // Erosion rate (m/Myr)
	MaxErosionAge                float64 // Maximum erosion age
	
	// Noise parameters
	NoiseScale                   float64 // Noise scale
	NoiseOctaves                 int     // Number of octaves
	NoisePersistence             float64 // Persistence
	NoiseLacunarity              float64 // Lacunarity
	NoiseAmplitude               float64 // Noise amplitude
	
	// Isostatic parameters
	CrustalDensity               float64 // Crustal density
	MantleDensity                float64 // Mantle density
}

// ElevationValidationMetrics contains statistics for validating elevation distribution
type ElevationValidationMetrics struct {
	// Basic elevation statistics
	MinElevation                 float64 // Minimum elevation (m)
	MaxElevation                 float64 // Maximum elevation (m)
	MeanElevation                float64 // Mean elevation (m)
	MedianElevation              float64 // Median elevation (m)
	ElevationRange               float64 // Total elevation range (m)
	
	// Land-sea distribution
	LandArea                     float64 // Percentage of surface above sea level
	OceanArea                    float64 // Percentage of surface below sea level
	MeanLandElevation            float64 // Mean elevation of land areas (m)
	MeanOceanDepth               float64 // Mean depth of ocean areas (m)
	
	// Hypsometric analysis
	ContinentalFreeboard         float64 // Mean continental elevation above sea level
	OceanicDepthMean             float64 // Mean oceanic depth below sea level
	HypsometricIntegral          float64 // Hypsometric integral (landscape maturity)
	
	// Tectonic feature validation
	MountainPeakCount            int     // Number of peaks above threshold
	TrenchCount                  int     // Number of deep ocean trenches
	VolcanicIslandCount          int     // Number of volcanic islands
	RidgeElevationMean           float64 // Mean ridge elevation
	
	// Earth comparison scores (0-1)
	ElevationDistributionScore   float64 // How well elevation matches Earth distribution
	HypsometricScore             float64 // How well hypsometric curve matches Earth
	LandSeaRatioScore            float64 // How well land-sea ratio matches Earth (~29% land)
	TopographicRoughnessScore    float64 // How well topographic roughness matches Earth
	OverallElevationScore        float64 // Overall elevation system Earth-likeness
}

// HypsometricPoint is defined in elevation_validation.go

// ElevationComponent represents different sources of elevation
type ElevationComponent string

const (
	BaseElevation       ElevationComponent = "base"        // Base plate-type elevation
	TectonicElevation   ElevationComponent = "tectonic"     // Boundary effects
	VolcanicElevation   ElevationComponent = "volcanic"     // Volcanic features
	SeafloorElevation   ElevationComponent = "seafloor"     // Age-depth relationships
	RidgeElevation      ElevationComponent = "ridge"        // Ridge topography
	ErosionElevation    ElevationComponent = "erosion"      // Erosion effects
	NoiseElevation      ElevationComponent = "noise"        // Fractal detail
	IsostaticElevation  ElevationComponent = "isostatic"    // Isostatic adjustment
)

// TerrainType categorizes different types of terrain
type TerrainType string

const (
	AbyssalPlain        TerrainType = "abyssal_plain"      // Deep ocean floor
	ContinentalShelf    TerrainType = "continental_shelf"  // Shallow coastal waters
	ContinentalSlope    TerrainType = "continental_slope"  // Slope to deep ocean
	MidOceanRidgeType   TerrainType = "mid_ocean_ridge"    // Ridge systems
	OceanTrench         TerrainType = "ocean_trench"       // Deep trenches
	VolcanicIsland      TerrainType = "volcanic_island"    // Volcanic islands
	ContinentalLowland  TerrainType = "continental_lowland" // Low continental areas
	ContinentalHighland TerrainType = "continental_highland" // High continental areas
	MountainRange       TerrainType = "mountain_range"     // Major mountain chains
	VolcanicArc         TerrainType = "volcanic_arc"       // Subduction zone arcs
)

// --- Default Configurations ---

// DefaultElevationSettings returns Earth-like elevation generation parameters
func DefaultElevationSettings() ElevationSettings {
	return ElevationSettings{
		// Basic parameters
		ElevationMultiplier:           1.0,
		GlobalSeed:                    42,
		
		// Tectonic boundary effects
		CharacteristicFalloffDistance: 0.15,  // 15% of radius
		MaxBoundaryEffectDistance:     0.45,  // 45% of radius
		ConvergentBoundaryStrength:    2000.0, // 2000m mountain building
		DivergentBoundaryStrength:     800.0,  // 800m rift subsidence
		
		// Ridge system
		EnableRidgeTopography:         true,
		RidgeElevationAboveSeafloor:   2500.0, // 2500m above seafloor
		RidgeInfluenceDistance:        100.0,  // 100km influence
		
		// Volcanic features
		EnableVolcanicElevation:       true,
		VolcanicElevationMultiplier:   1.0,
		HotspotInfluenceRadius:        200.0,  // 200km hotspot influence
		
		// Seafloor age-depth
		EnableSeafloorAgeDepth:        true,
		SeafloorModel:                 tectonics.DefaultSeafloorAgeModel(),
		
		// Subduction zones
		EnableSubductionEffects:       true,
		TrenchDepthMultiplier:         1.0,
		VolcanicArcElevation:          1500.0, // 1500m arc elevation
		
		// Erosion
		EnableErosion:                 true,
		ErosionRate:                   10.0,   // 10m/Myr erosion rate
		MaxErosionAge:                 200.0,  // 200 Myr max erosion
		
		// Fractal noise
		NoiseScale:                    0.01,   // Noise scale
		NoiseOctaves:                  6,      // 6 octaves
		NoisePersistence:              0.5,    // 50% persistence
		NoiseLacunarity:               2.0,    // 2x frequency multiplier
		NoiseAmplitude:                500.0,  // 500m noise amplitude
		
		// Isostasy
		EnableIsostasy:                false,  // Disabled by default (complex)
		CrustalDensity:                2700.0, // 2700 kg/m³
		MantleDensity:                 3300.0, // 3300 kg/m³
	}
}

// ArchipelagoElevationSettings returns settings optimized for archipelago worlds
func ArchipelagoElevationSettings() ElevationSettings {
	settings := DefaultElevationSettings()
	
	// Enhance volcanic features for island chains
	settings.VolcanicElevationMultiplier = 1.5
	settings.HotspotInfluenceRadius = 300.0
	
	// Reduce continental mountain building
	settings.ConvergentBoundaryStrength = 1000.0
	
	// Increase ridge prominence for underwater mountain chains
	settings.RidgeElevationAboveSeafloor = 3000.0
	settings.RidgeInfluenceDistance = 150.0
	
	// Reduce erosion to keep volcanic islands prominent
	settings.ErosionRate = 5.0
	
	return settings
}

// ContinentalElevationSettings returns settings optimized for continental worlds
func ContinentalElevationSettings() ElevationSettings {
	settings := DefaultElevationSettings()
	
	// Enhance continental mountain building
	settings.ConvergentBoundaryStrength = 3000.0
	settings.MaxBoundaryEffectDistance = 0.6
	
	// Reduce volcanic prominence
	settings.VolcanicElevationMultiplier = 0.8
	
	// Increase erosion for mature continental landscapes
	settings.ErosionRate = 15.0
	
	// Increase terrain roughness
	settings.NoiseAmplitude = 800.0
	settings.NoiseOctaves = 8
	
	return settings
}