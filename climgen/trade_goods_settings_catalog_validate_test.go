package climgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fullCatalog wraps a goods list into a complete, schema-valid settings document
// so the table cases below exercise ValidateTradeGoodsCatalog the way production
// does: on an assembled catalog, never on a fragment.
func fullCatalog(goods []TradeGoodSpec) TradeGoodsSettings {
	return TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods:         goods,
	}.withDefaults()
}

func TestValidateTradeGoodsCatalog(t *testing.T) {
	tests := []struct {
		name      string
		goods     []TradeGoodSpec
		wantErr   bool
		wantParts []string
	}{
		{
			name: "valid ordered catalog",
			goods: []TradeGoodSpec{
				{Name: "grain", Category: "raw"},
				{Name: "salt", Category: "raw"},
				{Name: "preserved_food", Category: "processed", Inputs: map[string]float64{"grain": 0.5, "salt": 0.2}},
				{Name: "rations", Category: "processed", Inputs: map[string]float64{"preserved_food": 0.8}},
			},
		},
		{
			name: "raw-only catalog with no inputs",
			goods: []TradeGoodSpec{
				{Name: "grain", Category: "raw"},
				{Name: "timber", Category: "raw"},
			},
		},
		{
			name: "unknown input name",
			goods: []TradeGoodSpec{
				{Name: "grain", Category: "raw"},
				{Name: "preserved_food", Category: "processed", Inputs: map[string]float64{"sal": 0.2}},
			},
			wantErr:   true,
			wantParts: []string{"preserved_food", "sal", "not a known trade good"},
		},
		{
			name: "unknown input reported before ordering",
			goods: []TradeGoodSpec{
				{Name: "preserved_food", Category: "processed", Inputs: map[string]float64{"ghost": 0.2}},
				{Name: "grain", Category: "raw"},
			},
			wantErr:   true,
			wantParts: []string{"ghost", "not a known trade good"},
		},
		{
			name: "two-good cycle",
			goods: []TradeGoodSpec{
				{Name: "ale", Category: "processed", Inputs: map[string]float64{"malt": 0.5}},
				{Name: "malt", Category: "processed", Inputs: map[string]float64{"ale": 0.5}},
			},
			wantErr:   true,
			wantParts: []string{"cycle", "ale", "malt"},
		},
		{
			name: "three-good cycle",
			goods: []TradeGoodSpec{
				{Name: "a", Category: "processed", Inputs: map[string]float64{"c": 1}},
				{Name: "b", Category: "processed", Inputs: map[string]float64{"a": 1}},
				{Name: "c", Category: "processed", Inputs: map[string]float64{"b": 1}},
			},
			wantErr:   true,
			wantParts: []string{"cycle", "a", "b", "c"},
		},
		{
			name: "self referential good",
			goods: []TradeGoodSpec{
				{Name: "grain", Category: "raw"},
				{Name: "compost", Category: "processed", Inputs: map[string]float64{"compost": 0.5}},
			},
			wantErr:   true,
			wantParts: []string{"cycle", "compost"},
		},
		{
			name: "consumer declared before its input",
			goods: []TradeGoodSpec{
				{Name: "preserved_food", Category: "processed", Inputs: map[string]float64{"grain": 0.5}},
				{Name: "grain", Category: "raw"},
			},
			wantErr:   true,
			wantParts: []string{"preserved_food", "grain", "inputs must precede consumers"},
		},
		{
			name: "consumer before a later intermediate",
			goods: []TradeGoodSpec{
				{Name: "grain", Category: "raw"},
				{Name: "rations", Category: "processed", Inputs: map[string]float64{"preserved_food": 1}},
				{Name: "preserved_food", Category: "processed", Inputs: map[string]float64{"grain": 1}},
			},
			wantErr:   true,
			wantParts: []string{"rations", "preserved_food", "index"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTradeGoodsCatalog(fullCatalog(tc.goods))
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if err == nil {
				return
			}
			for _, part := range tc.wantParts {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("error %q missing %q", err.Error(), part)
				}
			}
		})
	}
}

// TestPartialTradeGoodsDocumentSkipsCrossGoodValidation is the regression guard
// for the merge case: a document that defines a consumer whose input lives in the
// base catalog must still load, because cross-good rules are only decidable once
// the catalog is assembled.
func TestPartialTradeGoodsDocumentSkipsCrossGoodValidation(t *testing.T) {
	overlay := TradeGoodsSettings{
		SchemaVersion: TradeGoodsSchemaVersion,
		Goods: []TradeGoodSpec{
			{Name: "cloth", Category: "processed", BaseValue: 0.62, Inputs: map[string]float64{"fiber": 0.45}},
		},
	}.withDefaults()

	if err := ValidateTradeGoodsSettings(overlay); err != nil {
		t.Fatalf("partial document must pass local validation, got %v", err)
	}
	if err := ValidateTradeGoodsCatalog(overlay); err == nil {
		t.Fatalf("catalog validation of the fragment alone should still report the dangling input")
	}

	base := []TradeGoodSpec{{Name: "fiber", Category: "raw", BaseValue: 0.20}}
	merged := fullCatalog(append(append([]TradeGoodSpec{}, base...), overlay.Goods...))
	if err := ValidateTradeGoodsCatalog(merged); err != nil {
		t.Fatalf("assembled base+overlay catalog must validate, got %v", err)
	}
}

func TestEmbeddedTradeGoodsCatalogSatisfiesInputGraph(t *testing.T) {
	settings := DefaultTradeGoodsSettings()
	if len(settings.Goods) == 0 {
		t.Skip("no default trade goods catalog available")
	}
	if err := ValidateTradeGoodsCatalog(settings); err != nil {
		t.Fatalf("default trade goods catalog violates input graph rules: %v", err)
	}
}

func TestShippedTradeGoodsConfigSatisfiesInputGraph(t *testing.T) {
	path := filepath.Join("..", "config", "trade_goods_earthlike.json")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("shipped catalog not present: %v", err)
	}
	settings, err := LoadTradeGoodsSettings(path)
	if err != nil {
		t.Fatalf("expected shipped catalog to load: %v", err)
	}
	if err := ValidateTradeGoodsCatalog(settings); err != nil {
		t.Fatalf("shipped catalog %s violates input graph rules: %v", path, err)
	}
}
