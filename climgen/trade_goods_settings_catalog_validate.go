package climgen

import (
	"fmt"
	"sort"
	"strings"
)

// Cross-good validation for the trade goods catalog.
//
// These rules are only meaningful once every good the runtime will see is
// present. Trade goods settings are loaded document-by-document and may be
// merged (see trade_goods_settings_merge.go), so an individual document can
// legitimately reference an input declared in the base catalog. Applying these
// checks per document would reject valid overlays; they therefore live apart
// from ValidateTradeGoodsSettings and run only on the assembled catalog.

// sortedTradeGoodInputs returns a good's input names in a stable order so that
// validation errors are deterministic regardless of Go map iteration order.
func sortedTradeGoodInputs(good TradeGoodSpec) []string {
	if len(good.Inputs) == 0 {
		return nil
	}
	names := make([]string, 0, len(good.Inputs))
	for name := range good.Inputs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ValidateTradeGoodsCatalog enforces the cross-good rules that only hold on a
// fully assembled catalog:
//
//	(a) every declared input names a good that exists in the catalog,
//	(b) the input graph is acyclic,
//	(c) every consumer appears after all of its inputs in the catalog array.
//
// Rule (c) is a hard error, not a convention. climgen/trade_goods_nodes.go:61
// walks settings.Goods exactly once per settlement node, writing each good's
// output into balance.Supply as it goes, and nodeGoodSupply reads
// currentSupply[input] for its inputs from that same partially built map. A
// consumer evaluated before its input therefore sees supply 0, which either
// trips tradeGoodsHasLocalInputCapability (output forced to 0) or drives
// inputAccess to the trade-access floor. The failure is silent: the run
// completes and simply reports that the good is never produced locally.
//
// Call this once, after any merging, on the catalog the runtime will actually
// use. Do not call it on a partial document.
func ValidateTradeGoodsCatalog(settings TradeGoodsSettings) error {
	return validateTradeGoodsInputGraph(settings.Goods)
}

func validateTradeGoodsInputGraph(goods []TradeGoodSpec) error {
	indexByName := make(map[string]int, len(goods))
	for i, good := range goods {
		indexByName[good.Name] = i
	}

	// (a) unknown input names.
	for _, good := range goods {
		for _, input := range sortedTradeGoodInputs(good) {
			if _, ok := indexByName[input]; !ok {
				return fmt.Errorf("trade good %q input %q is not a known trade good", good.Name, input)
			}
		}
	}

	// (b) dependency cycles. DFS with a path stack so the error names the exact
	// cycle. Checked before ordering so a genuine cycle reports as a cycle rather
	// than as an out-of-order consumer.
	const (
		unvisited = 0
		onPath    = 1
		done      = 2
	)
	state := make([]int, len(goods))
	path := make([]string, 0, len(goods))

	var visit func(idx int) error
	visit = func(idx int) error {
		state[idx] = onPath
		path = append(path, goods[idx].Name)
		for _, input := range sortedTradeGoodInputs(goods[idx]) {
			next := indexByName[input]
			switch state[next] {
			case onPath:
				start := 0
				for i, name := range path {
					if name == goods[next].Name {
						start = i
						break
					}
				}
				cycle := append(append([]string{}, path[start:]...), goods[next].Name)
				return fmt.Errorf("trade goods input cycle: %s", strings.Join(cycle, " -> "))
			case unvisited:
				if err := visit(next); err != nil {
					return err
				}
			}
		}
		path = path[:len(path)-1]
		state[idx] = done
		return nil
	}

	for i := range goods {
		if state[i] == unvisited {
			if err := visit(i); err != nil {
				return err
			}
		}
	}

	// (c) consumer declared before one of its inputs.
	for i, good := range goods {
		for _, input := range sortedTradeGoodInputs(good) {
			if j := indexByName[input]; j >= i {
				return fmt.Errorf(
					"trade good %q (index %d) is declared before its input %q (index %d); "+
						"inputs must precede consumers because node supply is built in array order",
					good.Name, i, input, j)
			}
		}
	}

	return nil
}
