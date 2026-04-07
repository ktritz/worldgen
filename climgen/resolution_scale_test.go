package climgen

import "testing"

func TestMeshPathCostResolutionScale(t *testing.T) {
	if scale := meshPathCostResolutionScale(10242); scale < 0.99 || scale > 1.01 {
		t.Fatalf("expected level-5-ish scale near 1.0, got %.3f", scale)
	}
	if scale := meshPathCostResolutionScale(40962); scale < 0.49 || scale > 0.51 {
		t.Fatalf("expected level-6-ish scale near 0.5, got %.3f", scale)
	}
}
