package main

import (
	"fmt"
	"strings"
	"time"
)

func reviewPhaseStart(layer string) time.Time {
	start := time.Now()
	fmt.Printf("  phase start: layer=%s started_at=%d\n", layer, start.Unix())
	return start
}

func reviewPhaseDone(layer string, cacheStatus string, start time.Time, parts ...string) {
	fields := []string{
		fmt.Sprintf("layer=%s", layer),
		fmt.Sprintf("cache=%s", cacheStatus),
		fmt.Sprintf("elapsed_ms=%d", time.Since(start).Milliseconds()),
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		fields = append(fields, part)
	}
	fmt.Printf("  phase done: %s\n", strings.Join(fields, " "))
}
