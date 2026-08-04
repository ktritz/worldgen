package namegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeterminismAndKeyIndependence(t *testing.T) {
	a := New(42)
	b := New(42)

	// Same seed + key -> same name, independent of call order.
	aRiver := a.ForCulture("river_confederacy").River("river/7")
	aSettle := a.ForCulture("river_confederacy").Settlement("cell/123")

	bSettle := b.ForCulture("river_confederacy").Settlement("cell/123")
	bRiver := b.ForCulture("river_confederacy").River("river/7")

	if aRiver != bRiver {
		t.Errorf("river name not deterministic: %q vs %q", aRiver, bRiver)
	}
	if aSettle != bSettle {
		t.Errorf("settlement name not deterministic: %q vs %q", aSettle, bSettle)
	}

	// Different seeds should (almost always) change output somewhere.
	c := New(43)
	diff := false
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("cell/%d", i)
		if a.Base(BaseCeltic).Settlement(key) != c.Base(BaseCeltic).Settlement(key) {
			diff = true
			break
		}
	}
	if !diff {
		t.Error("different seeds produced identical outputs for 10 keys")
	}

	// Different keys -> a healthy spread of distinct names.
	seen := map[string]bool{}
	bn := a.Base(BaseGermanic)
	for i := 0; i < 60; i++ {
		seen[bn.Settlement(fmt.Sprintf("cell/%d", i))] = true
	}
	if len(seen) < 30 {
		t.Errorf("expected at least 30 distinct names from 60 keys, got %d", len(seen))
	}
}

func TestDeriveStable(t *testing.T) {
	n := New(7)
	if n.Derive("x") != n.Derive("x") {
		t.Error("Derive is not stable")
	}
	if n.Derive("x") == n.Derive("y") {
		t.Error("Derive collided for different keys")
	}
	if New(7).Derive("x") != n.Derive("x") {
		t.Error("Derive differs across Namers with the same seed")
	}
}

func TestAllCulturesAndAncestriesResolve(t *testing.T) {
	n := New(1)
	for culture, base := range CultureBase {
		bn := n.ForCulture(culture)
		if bn.BaseName() != base {
			t.Errorf("culture %q resolved to base %q, want %q", culture, bn.BaseName(), base)
		}
	}
	for ancestry, base := range AncestryBase {
		bn := n.ForAncestry(ancestry)
		if bn.BaseName() != base {
			t.Errorf("ancestry %q resolved to base %q, want %q", ancestry, bn.BaseName(), base)
		}
	}
	// Display-name normalization.
	if n.ForCulture("Highland Hold").BaseName() != BaseNordic {
		t.Error("display-name culture lookup failed")
	}
	// Unknown keys fall back to the default base.
	if n.ForCulture("no_such_culture").BaseName() != DefaultBase {
		t.Error("unknown culture did not fall back to DefaultBase")
	}
	if n.Base("no_such_base").BaseName() != DefaultBase {
		t.Error("unknown base did not fall back to DefaultBase")
	}
}

// If the fantasy profile configs are present in the repo, every profile file
// must have a mapping entry. Skipped when run outside the repo layout.
func TestMappingCoversProfileFiles(t *testing.T) {
	cases := []struct {
		dir string
		m   map[string]string
	}{
		{"../config/profiles/fantasy/cultures", CultureBase},
		{"../config/profiles/fantasy/ancestries", AncestryBase},
	}
	for _, c := range cases {
		entries, err := os.ReadDir(c.dir)
		if err != nil {
			t.Skipf("profile dir %s not available: %v", c.dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
				continue
			}
			key := strings.TrimSuffix(e.Name(), ".json")
			if _, ok := c.m[key]; !ok {
				t.Errorf("profile %s/%s has no name-base mapping", c.dir, e.Name())
			}
		}
	}
}

func TestGeneratedBatchSanity(t *testing.T) {
	n := New(1234)
	for _, baseKey := range BaseNames() {
		spec := basesByKey[baseKey].spec
		bn := n.Base(baseKey)
		for i := 0; i < 200; i++ {
			name := bn.Settlement(fmt.Sprintf("batch/%d", i))
			if len(name) < 2 {
				t.Fatalf("[%s] name too short: %q", baseKey, name)
			}
			if len(name) > spec.Max+4 {
				t.Errorf("[%s] name too long (%d > %d): %q", baseKey, len(name), spec.Max+4, name)
			}
			first, last := name[0], name[len(name)-1]
			if !(first >= 'A' && first <= 'Z') {
				t.Errorf("[%s] name not title-cased: %q", baseKey, name)
			}
			if !isLetter(last) {
				t.Errorf("[%s] trailing junk: %q", baseKey, name)
			}
			for j := 0; j < len(name); j++ {
				if !isLetter(name[j]) && name[j] != ' ' && name[j] != '-' && name[j] != '\'' {
					t.Errorf("[%s] bad character %q in %q", baseKey, name[j], name)
				}
			}
			if hasTriple(name) {
				t.Errorf("[%s] triple letter in %q", baseKey, name)
			}
		}
	}
}

func TestPolityMorphologyCoverage(t *testing.T) {
	n := New(99)
	classes := PolitySizeClasses()
	if len(classes) == 0 {
		t.Fatal("no polity size classes")
	}
	for _, baseKey := range BaseNames() {
		bn := n.Base(baseKey)
		for _, sc := range classes {
			for i := 0; i < 20; i++ {
				name := bn.Polity(sc, fmt.Sprintf("polity/%d", i))
				if name == "" || strings.Contains(name, "%") {
					t.Fatalf("[%s/%s] bad polity name %q", baseKey, sc, name)
				}
				if hasTriple(name) {
					t.Errorf("[%s/%s] triple letter in %q", baseKey, sc, name)
				}
			}
		}
	}
	// Every generic template class and every override class must be exercised
	// by the table lookup without falling through to a format verb.
	for base, over := range polityOverrides {
		for sc := range over {
			name := n.Base(base).Polity(sc, "override-check")
			if strings.Contains(name, "%") {
				t.Errorf("override [%s/%s] left format verb: %q", base, sc, name)
			}
		}
	}
	// Unknown size class falls back to kingdom forms.
	if name := n.Base(BaseRoman).Polity("galactic_hegemony", "k"); name == "" || strings.Contains(name, "%") {
		t.Errorf("unknown size class fallback produced %q", name)
	}
}

func TestFeatureNames(t *testing.T) {
	n := New(5)
	bn := n.ForCulture("forest_enclave")
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("f/%d", i)
		for _, name := range []string{bn.River(key), bn.Sea(key), bn.Range(key)} {
			if name == "" || strings.Contains(name, "%") {
				t.Fatalf("bad feature name %q", name)
			}
			if hasTriple(name) {
				t.Errorf("triple letter in feature name %q", name)
			}
		}
	}
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func hasTriple(s string) bool {
	ls := strings.ToLower(s)
	for i := 0; i+2 < len(ls); i++ {
		if isLetter(ls[i]) && ls[i] == ls[i+1] && ls[i] == ls[i+2] {
			return true
		}
	}
	return false
}
