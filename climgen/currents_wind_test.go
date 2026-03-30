package climgen

import "testing"

func TestGenerateCurrentsStreamfunctionFromWindProducesOceanFlow(t *testing.T) {
	vertices, _, elevation, adj := buildWindTestWorld(4)
	for i, v := range vertices {
		// Add a simple meridional barrier so the Sverdrup-style solver has
		// eastern boundaries to integrate away from.
		if v.X > 0.82 && absFloat(v.Z) < 0.35 {
			elevation[i] = 500
		}
	}

	windSettings := DefaultWindSettings()
	windSettings.Seed = 42
	windSettings.Orographic.DeflectionStrength = 0
	windResult, err := GenerateWindField(vertices, elevation, 0.0, adj, windSettings)
	if err != nil {
		t.Fatalf("GenerateWindField failed: %v", err)
	}

	currentSettings := DefaultCurrentSettings()
	componentAssignments, components := FindOceanComponents(elevation, 0.0, adj)
	currents := GenerateCurrentsStreamfunctionFromWind(
		vertices,
		elevation,
		0.0,
		adj,
		windResult.MarineWind,
		componentAssignments,
		components,
		currentSettings,
	)

	nonZero := 0
	for i, current := range currents {
		if elevation[i] >= 0 {
			continue
		}
		if Length(current) > 1e-6 {
			nonZero++
		}
		if radial := Dot(vertices[i], current); radial > 1e-6 || radial < -1e-6 {
			t.Fatalf("current at %d has radial component %.6g", i, radial)
		}
	}
	if nonZero == 0 {
		t.Fatalf("expected wind-forced currents to produce non-zero ocean flow")
	}
}

func TestGenerateCurrentsStayTangentToCoastlines(t *testing.T) {
	vertices, _, elevation, adj := buildWindTestWorld(4)
	for i, v := range vertices {
		if v.X > 0.82 && absFloat(v.Z) < 0.35 {
			elevation[i] = 500
		}
	}

	windSettings := DefaultWindSettings()
	windSettings.Seed = 7
	windSettings.Orographic.DeflectionStrength = 0
	windResult, err := GenerateWindField(vertices, elevation, 0.0, adj, windSettings)
	if err != nil {
		t.Fatalf("GenerateWindField failed: %v", err)
	}

	componentAssignments, components := FindOceanComponents(elevation, 0.0, adj)
	currents := GenerateCurrentsStreamfunctionFromWind(
		vertices,
		elevation,
		0.0,
		adj,
		windResult.MarineWind,
		componentAssignments,
		components,
		DefaultCurrentSettings(),
	)
	landDirs := CalculateCoastlineLandDirs(vertices, elevation, 0.0, adj)

	checked := 0
	for i, current := range currents {
		if elevation[i] >= 0 {
			continue
		}
		landDir := landDirs[i]
		landLen := Length(landDir)
		currLen := Length(current)
		if landLen < 1e-6 || currLen < 1e-6 {
			continue
		}

		alignment := absFloat(Dot(current, landDir) / (currLen * landLen))
		if alignment > 0.12 {
			t.Fatalf("current at %d is not coast-parallel enough: alignment=%.3f", i, alignment)
		}
		checked++
	}

	if checked == 0 {
		t.Fatalf("expected to validate at least one near-coast current")
	}
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
