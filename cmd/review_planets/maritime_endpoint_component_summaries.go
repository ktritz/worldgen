package main

import (
	"fmt"
	"strconv"

	"worldgen/climgen"
)

func printEndpointComponentDetails(label string, diagnostics climgen.CoastalTradeEndpointDiagnostics, limit int) {
	if len(diagnostics.PortComponentsDetail) == 0 || limit <= 0 {
		return
	}
	for i, component := range diagnostics.PortComponentsDetail {
		if i >= limit {
			break
		}
		fmt.Printf(
			"      %sComponent[%d]: endpoints=%d ports=%d nodes=%s civs=%s\n",
			label,
			i,
			component.Endpoints,
			component.Ports,
			formatIntList(component.PortNodes, 8),
			formatIntList(component.Civilizations, 8),
		)
	}
}

func formatIntList(values []int, limit int) string {
	if len(values) == 0 {
		return "-"
	}
	if limit <= 0 || limit > len(values) {
		limit = len(values)
	}
	out := ""
	for i := 0; i < limit; i++ {
		if i > 0 {
			out += ","
		}
		out += strconv.Itoa(values[i])
	}
	if limit < len(values) {
		out += ",..."
	}
	return out
}
