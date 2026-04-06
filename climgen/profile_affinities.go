package climgen

import "strings"

type ProfileTraitMap map[string]float64

type ProfileAffinityRule struct {
	TargetType string  `json:"targetType"`
	Target     string  `json:"target"`
	Weight     float64 `json:"weight"`
}

type ProfileAffinityContext struct {
	ProfileName  string
	AncestryName string
	CultureName  string
	Tags         []string
	Traits       map[string]float64
}

func cloneTraitMap(src map[string]float64) map[string]float64 {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]float64, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneAffinityRules(src []ProfileAffinityRule) []ProfileAffinityRule {
	if len(src) == 0 {
		return nil
	}
	out := make([]ProfileAffinityRule, len(src))
	copy(out, src)
	return out
}

func mergeProfileTags(base []string, extra []string) []string {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	for _, raw := range append(append([]string(nil), base...), extra...) {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out
}

func mergeTraitMaps(base map[string]float64, extra map[string]float64) map[string]float64 {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	out := cloneTraitMap(base)
	if out == nil {
		out = make(map[string]float64, len(extra))
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func mergeAffinityRules(base []ProfileAffinityRule, extra []ProfileAffinityRule) []ProfileAffinityRule {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	out := cloneAffinityRules(base)
	out = append(out, extra...)
	return out
}

func hasProfileTag(tags []string, target string) bool {
	for _, tag := range tags {
		if tag == target {
			return true
		}
	}
	return false
}

func BuildResolvedProfileAffinityContext(profile *ResolvedProfile) ProfileAffinityContext {
	if profile == nil {
		return ProfileAffinityContext{}
	}
	return ProfileAffinityContext{
		ProfileName:  profile.Name,
		AncestryName: profile.AncestryName,
		CultureName:  profile.CultureName,
		Tags:         append([]string(nil), profile.Tags...),
		Traits:       cloneTraitMap(profile.Traits),
	}
}

func ScoreProfileAffinity(subject *ResolvedProfile, target ProfileAffinityContext) float64 {
	if subject == nil {
		return 0
	}
	score := 0.0
	for _, rule := range subject.Affinities {
		switch rule.TargetType {
		case "profile":
			if target.ProfileName == rule.Target {
				score += rule.Weight
			}
		case "ancestry":
			if target.AncestryName == rule.Target {
				score += rule.Weight
			}
		case "culture":
			if target.CultureName == rule.Target {
				score += rule.Weight
			}
		case "tag":
			if hasProfileTag(target.Tags, rule.Target) {
				score += rule.Weight
			}
		case "trait":
			score += rule.Weight * clamp01(target.Traits[rule.Target])
		}
	}
	return score
}
