package climgen

import (
	"math"
)

// =============================================================================
// TEMPERATURE GENERATION - SOLAR INSOLATION AND ALBEDO
// =============================================================================
// This file contains functions for computing solar insolation and surface albedo.
// These are the primary energy inputs to the energy balance model.

// --- Insolation Calculation ---

// ComputeInsolation calculates latitude-dependent solar radiation for each vertex.
//
// Uses the annual-average insolation distribution:
//
//	Q(lat) = (S/4) * L * s(lat)
//
// where:
//   - S = solar constant (1361 W/m²)
//   - L = solar luminosity multiplier
//   - s(lat) = latitudinal distribution function
//
// The distribution s(lat) uses a Legendre polynomial approximation:
//
//	s(lat) = 1 - 0.48 * P2(sin(lat))
//	P2(x) = 0.5 * (3x² - 1)
//
// This gives more radiation at the equator and less at the poles,
// matching Earth's annual-average pattern.
func ComputeInsolation(vertices []Vector3D, settings SolarSettings) []float64 {
	numVertices := len(vertices)
	insolation := make([]float64, numVertices)

	baseQ := (SolarConstant / 4.0) * settings.SolarLuminosity

	for i, v := range vertices {
		lat := getLatitude(v)
		sinLat := math.Sin(lat)

		// Legendre polynomial P2(sin(lat)) = 0.5 * (3*sin²(lat) - 1)
		p2SinLat := 0.5 * (3.0*sinLat*sinLat - 1.0)

		// Latitudinal distribution: more at equator, less at poles
		sLat := 1.0 - 0.48*p2SinLat

		insolation[i] = baseQ * sLat
	}

	return insolation
}

// ComputeSeasonalInsolation calculates insolation with seasonal variation.
//
// The solar declination angle varies throughout the year:
//
//	delta = -axialTilt * cos(2*pi*seasonPhase)
//
// where seasonPhase = 0 at northern winter solstice, 0.5 at summer solstice.
//
// Insolation at each latitude is then:
//
//	Q(lat) = (S/4) * L * max(0, cos(lat)*cos(delta) + sin(lat)*sin(delta))
//
// This is a simplified model that captures the basic seasonal pattern.
func ComputeSeasonalInsolation(vertices []Vector3D, settings SolarSettings) []float64 {
	numVertices := len(vertices)
	insolation := make([]float64, numVertices)

	// If no axial tilt, fall back to annual average
	if settings.AxialTilt < 0.01 {
		return ComputeInsolation(vertices, settings)
	}

	tiltRad := settings.AxialTilt * math.Pi / 180.0

	// Solar declination: varies from -tilt to +tilt over the year
	// seasonPhase = 0   -> NH winter solstice  (declination = -tilt)
	// seasonPhase = 0.25 -> NH spring equinox  (declination = 0)
	// seasonPhase = 0.5 -> NH summer solstice  (declination = +tilt)
	// seasonPhase = 0.75 -> NH autumn equinox  (declination = 0)
	declination := -tiltRad * math.Cos(2.0*math.Pi*settings.SeasonPhase)

	baseQ := SolarConstant * settings.SolarLuminosity

	sinDec := math.Sin(declination)
	cosDec := math.Cos(declination)

	for i, v := range vertices {
		lat := getLatitude(v)
		sinLat := math.Sin(lat)
		cosLat := math.Cos(lat)

		// Daily-integrated insolation (simplified)
		// This approximates the integral of solar flux over a day
		// accounting for day length variation with latitude and season

		// Hour angle at sunrise/sunset
		// cos(H) = -tan(lat)*tan(dec)
		tanLat := sinLat / (cosLat + 1e-12)
		tanDec := sinDec / (cosDec + 1e-12)
		cosH := -tanLat * tanDec

		var dayFraction float64
		if cosH >= 1.0 {
			// Polar night: no sunlight
			dayFraction = 0.0
		} else if cosH <= -1.0 {
			// Midnight sun: 24 hours of sunlight
			dayFraction = 1.0
		} else {
			// Normal day/night cycle
			H := math.Acos(cosH) // Hour angle at sunset
			dayFraction = H / math.Pi
		}

		// Insolation proportional to day length and solar angle
		// Factor of 1/pi comes from averaging over the hemisphere
		q := baseQ / math.Pi * (dayFraction*sinLat*sinDec + math.Sin(dayFraction*math.Pi)*cosLat*cosDec)

		// Ensure non-negative
		if q < 0 {
			q = 0
		}

		insolation[i] = q
	}

	return insolation
}

// --- Albedo Calculation ---

// ComputeAlbedo returns the surface albedo for each vertex.
//
// Albedo depends on surface type:
//   - Ice (temperature < 273.15 K): 0.45 (highly reflective)
//   - Land (elevation >= threshold): 0.20
//   - Water (elevation < threshold): 0.08 (absorbs most light)
//
// If ice-albedo feedback is disabled, ice albedo is not applied
// (useful for testing without feedback effects).
func ComputeAlbedo(
	temperature []float64,
	elevation []float64,
	seaLevelThreshold float64,
	iceAlbedoFeedback bool,
) []float64 {
	numVertices := len(temperature)
	albedo := make([]float64, numVertices)

	for i := range temperature {
		// Check for ice first (if feedback enabled)
		if iceAlbedoFeedback && temperature[i] < FreezingPoint {
			albedo[i] = AlbedoIce
			continue
		}

		// Land vs water
		if elevation[i] >= seaLevelThreshold {
			albedo[i] = AlbedoLand
		} else {
			albedo[i] = AlbedoWater
		}
	}

	return albedo
}

// ComputeAlbedoSmooth computes albedo with smooth transitions for ice/snow.
//
// Land and ocean have different freezing behavior:
//   - Land: Snow/ice forms at 0°C (273.15K) with moderate transition
//   - Ocean: Sea ice forms at -2°C (271.15K) with wider transition due to
//     thermal mass and salt content. Sea ice is also harder to form.
//
// The ice fraction varies smoothly using a smoothstep function.
func ComputeAlbedoSmooth(
	temperature []float64,
	elevation []float64,
	seaLevelThreshold float64,
	transitionWidth float64, // K, typically 2-5 for land
) []float64 {
	numVertices := len(temperature)
	albedo := make([]float64, numVertices)

	// Ocean freezes at lower temp (-2°C) and has wider transition
	const seawaterFreezing = 271.15               // -2°C for seawater
	oceanTransitionWidth := transitionWidth * 3.0 // Much wider transition for ocean

	for i := range temperature {
		t := temperature[i]
		isOcean := elevation[i] < seaLevelThreshold

		var baseAlbedo float64
		var freezePoint float64
		var width float64

		if isOcean {
			baseAlbedo = AlbedoWater
			freezePoint = seawaterFreezing
			width = oceanTransitionWidth
		} else {
			baseAlbedo = AlbedoLand
			freezePoint = FreezingPoint
			width = transitionWidth
		}

		// Smooth transition to ice albedo
		tLow := freezePoint - width
		tHigh := freezePoint + width

		var iceFrac float64
		if t <= tLow {
			iceFrac = 1.0
		} else if t >= tHigh {
			iceFrac = 0.0
		} else {
			// Smoothstep interpolation
			x := (t - tLow) / (tHigh - tLow)
			iceFrac = 1.0 - x*x*(3.0-2.0*x)
		}

		// Sea ice has slightly lower albedo than land snow/ice
		iceAlbedo := AlbedoIce
		if isOcean {
			iceAlbedo = 0.35 // Sea ice is darker than fresh snow
		}

		albedo[i] = iceFrac*iceAlbedo + (1.0-iceFrac)*baseAlbedo
	}

	return albedo
}

// --- Absorbed Solar Radiation ---

// ComputeAbsorbedSolar returns the absorbed solar radiation (ASR) for each vertex.
//
//	ASR = Q * (1 - albedo)
//
// where Q is the incoming solar radiation and albedo is the reflectivity.
func ComputeAbsorbedSolar(insolation []float64, albedo []float64) []float64 {
	numVertices := len(insolation)
	absorbed := make([]float64, numVertices)

	for i := range insolation {
		absorbed[i] = insolation[i] * (1.0 - albedo[i])
	}

	return absorbed
}

// --- Net Radiation Budget ---

// ComputeNetRadiation returns the net radiative flux (ASR - OLR) at each vertex.
// Positive values indicate net energy gain (warming), negative indicates cooling.
func ComputeNetRadiation(absorbed []float64, olr []float64) []float64 {
	numVertices := len(absorbed)
	net := make([]float64, numVertices)

	for i := range absorbed {
		net[i] = absorbed[i] - olr[i]
	}

	return net
}
