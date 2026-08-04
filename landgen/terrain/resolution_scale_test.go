package terrain

import "testing"

func TestMeshResolutionScaleUsesBaselinePhysicalLength(t *testing.T) {
	if scale := meshPathCostResolutionScale(10242); scale < 0.99 || scale > 1.01 {
		t.Fatalf("baseline path scale = %.3f, want about 1", scale)
	}
	if scale := meshPathCostResolutionScale(40962); scale < 0.49 || scale > 0.51 {
		t.Fatalf("refined path scale = %.3f, want about 0.5", scale)
	}
	if steps := meshResolutionAdjustedSteps(15, 40962); steps != 30 {
		t.Fatalf("refined 15-hop physical radius = %d, want 30", steps)
	}
	if iterations := meshResolutionAdjustedDiffusionIterations(3, 40962); iterations != 12 {
		t.Fatalf("refined diffusion iterations = %d, want 12", iterations)
	}
}
