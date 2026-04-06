package main

import (
	"fmt"
	"sort"

	"worldgen/climgen"
)

func printPolityProfileSummary(result *climgen.PolityProfileResult) {
	if result == nil {
		return
	}
	if len(result.Assignments) == 0 {
		fmt.Println("    polityProfiles: assignments=0 attitudes=0")
		return
	}
	fmt.Printf("    polityProfiles: assignments=%d attitudes=%d\n", len(result.Assignments), len(result.Attitudes))
	for _, line := range topPolityProfileCounts(result, 6) {
		fmt.Printf("      profileCount[%s]=%d\n", line.name, line.count)
	}
	for _, line := range topPolityAncestryCounts(result, 6) {
		fmt.Printf("      ancestryCount[%s]=%d\n", line.name, line.count)
	}
	for _, line := range topPolityStanceCounts(result) {
		fmt.Printf("      stanceCount[%s]=%d\n", line.name, line.count)
	}
	for _, assignment := range topPolityAssignments(result, 5) {
		fmt.Printf(
			"      profile[%d]: %s ctx=%v env=%v score=%.2f\n",
			assignment.PolityID,
			assignment.Profile.Name,
			assignment.ContextTags,
			assignment.EnvironmentTags,
			assignment.Score,
		)
	}
	for _, attitude := range topPolityAttitudes(result, 6) {
		fmt.Printf(
			"      attitude[%d->%d]: %s score=%.2f cultural=%.2f strategic=%.2f trade=%.2f alliance=%.2f border=%.2f competition=%.2f\n",
			attitude.From,
			attitude.To,
			climgen.PolityAttitudeStanceName(attitude.Stance),
			attitude.Score,
			attitude.Cultural,
			attitude.StrategicTension,
			attitude.TradeBonus,
			attitude.AllianceBonus,
			attitude.BorderPenalty,
			attitude.Competition,
		)
	}
}

type polityProfileCount struct {
	name  string
	count int
}

func topPolityAssignments(result *climgen.PolityProfileResult, limit int) []climgen.PolityProfileAssignment {
	if result == nil || len(result.Assignments) == 0 {
		return nil
	}
	sorted := append([]climgen.PolityProfileAssignment(nil), result.Assignments...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Score > sorted[j].Score })
	if len(sorted) < limit {
		limit = len(sorted)
	}
	return sorted[:limit]
}

func topPolityAttitudes(result *climgen.PolityProfileResult, limit int) []climgen.PolityAttitude {
	if result == nil || len(result.Attitudes) == 0 {
		return nil
	}
	sorted := append([]climgen.PolityAttitude(nil), result.Attitudes...)
	sort.Slice(sorted, func(i, j int) bool {
		return abs(sorted[i].Score) > abs(sorted[j].Score)
	})
	if len(sorted) < limit {
		limit = len(sorted)
	}
	return sorted[:limit]
}

func topPolityProfileCounts(result *climgen.PolityProfileResult, limit int) []polityProfileCount {
	if result == nil || len(result.Assignments) == 0 {
		return nil
	}
	counts := make(map[string]int, len(result.Assignments))
	for _, assignment := range result.Assignments {
		counts[assignment.Profile.Name]++
	}
	out := make([]polityProfileCount, 0, len(counts))
	for name, count := range counts {
		out = append(out, polityProfileCount{name: name, count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].count != out[j].count {
			return out[i].count > out[j].count
		}
		return out[i].name < out[j].name
	})
	if len(out) < limit {
		limit = len(out)
	}
	return out[:limit]
}

func topPolityAncestryCounts(result *climgen.PolityProfileResult, limit int) []polityProfileCount {
	if result == nil || len(result.Assignments) == 0 {
		return nil
	}
	counts := make(map[string]int, len(result.Assignments))
	for _, assignment := range result.Assignments {
		name := assignment.Profile.AncestryName
		if name == "" {
			name = assignment.Profile.Name
		}
		counts[name]++
	}
	out := make([]polityProfileCount, 0, len(counts))
	for name, count := range counts {
		out = append(out, polityProfileCount{name: name, count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].count != out[j].count {
			return out[i].count > out[j].count
		}
		return out[i].name < out[j].name
	})
	if len(out) < limit {
		limit = len(out)
	}
	return out[:limit]
}

func topPolityStanceCounts(result *climgen.PolityProfileResult) []polityProfileCount {
	if result == nil || len(result.Attitudes) == 0 {
		return nil
	}
	counts := map[string]int{
		climgen.PolityAttitudeStanceName(climgen.PolityAttitudeHostile):  0,
		climgen.PolityAttitudeStanceName(climgen.PolityAttitudeWary):     0,
		climgen.PolityAttitudeStanceName(climgen.PolityAttitudeNeutral):  0,
		climgen.PolityAttitudeStanceName(climgen.PolityAttitudeFriendly): 0,
		climgen.PolityAttitudeStanceName(climgen.PolityAttitudeAllied):   0,
	}
	for _, attitude := range result.Attitudes {
		counts[climgen.PolityAttitudeStanceName(attitude.Stance)]++
	}
	out := make([]polityProfileCount, 0, len(counts))
	for _, name := range []string{
		climgen.PolityAttitudeStanceName(climgen.PolityAttitudeHostile),
		climgen.PolityAttitudeStanceName(climgen.PolityAttitudeWary),
		climgen.PolityAttitudeStanceName(climgen.PolityAttitudeNeutral),
		climgen.PolityAttitudeStanceName(climgen.PolityAttitudeFriendly),
		climgen.PolityAttitudeStanceName(climgen.PolityAttitudeAllied),
	} {
		out = append(out, polityProfileCount{name: name, count: counts[name]})
	}
	return out
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
