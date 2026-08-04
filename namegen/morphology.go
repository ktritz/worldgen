package namegen

import "strings"

// Polity morphology: templates keyed by size class, with per-base flavor
// overrides. In templates, %s is the generated core name and %a is its
// adjectival form ("Belz" -> "Belzian").

// politySizeClasses in stable order (see PolitySizeClasses).
var politySizeClasses = []string{
	"empire",
	"kingdom",
	"principality",
	"duchy",
	"city_state",
	"league",
	"confederacy",
	"tribe",
}

// PolitySizeClasses lists the size classes the Polity morphology table
// covers, in stable order. Unknown size classes fall back to "kingdom".
func PolitySizeClasses() []string {
	out := make([]string, len(politySizeClasses))
	copy(out, politySizeClasses)
	return out
}

var polityTemplates = map[string][]string{
	"empire":       {"Empire of %s", "%s Empire", "%a Dominion"},
	"kingdom":      {"Kingdom of %s", "Realm of %s", "%s"},
	"principality": {"Principality of %s", "%a Principality"},
	"duchy":        {"Duchy of %s", "Grand Duchy of %s"},
	"city_state":   {"Free City of %s", "Republic of %s", "%s"},
	"league":       {"%s League", "League of %s", "%a League"},
	"confederacy":  {"%s Confederacy", "Confederation of %s", "%a Compact"},
	"tribe":        {"%s Tribes", "%a Clans"},
}

// polityOverrides adds culture flavor for specific bases; they replace the
// generic template set for that (base, size class) pair.
var polityOverrides = map[string]map[string][]string{
	BaseArabic: {
		"empire":  {"%s Caliphate", "Caliphate of %s"},
		"kingdom": {"Sultanate of %s", "Emirate of %s"},
	},
	BaseRoman: {
		"empire":     {"%s Imperium", "Empire of %s"},
		"city_state": {"Republic of %s", "%s"},
	},
	BaseOrcish: {
		"tribe":   {"%s Horde", "%s Warband"},
		"kingdom": {"%s Dominion", "%s Horde"},
	},
	BaseGoblin: {
		"tribe": {"%s Warrens", "%s Horde"},
	},
}

func politeTemplate(baseKey, sizeClass string, r *rng) string {
	if over, ok := polityOverrides[baseKey]; ok {
		if ts, ok := over[sizeClass]; ok {
			return ts[r.intn(len(ts))]
		}
	}
	ts, ok := polityTemplates[sizeClass]
	if !ok {
		ts = polityTemplates["kingdom"]
	}
	return ts[r.intn(len(ts))]
}

// Physical-feature templates. Kept deliberately simple.
var riverTemplates = []string{"%s River", "River %s", "%s River", "the %s"}
var seaTemplates = []string{"Sea of %s", "%s Sea", "Gulf of %s", "Bay of %s"}
var rangeTemplates = []string{"%s Mountains", "%s Range", "the %s Peaks", "%s Highlands"}

// expandTemplate substitutes %s with the core name and %a with its
// adjectival form.
func expandTemplate(tmpl, name string) string {
	out := strings.ReplaceAll(tmpl, "%a", adjectival(name))
	return strings.ReplaceAll(out, "%s", name)
}

// adjectival derives a demonym-style adjective from a name:
// ends in "a" -> +"n" (Elia -> Elian); ends in another vowel -> drop it and
// add "ian" (Breisgau -> Breisgian is avoided by only dropping one letter:
// Karlsruhe -> Karlsruhian); otherwise +"ian" (Belz -> Belzian).
func adjectival(name string) string {
	if name == "" {
		return name
	}
	last := name[len(name)-1]
	switch {
	case last == 'a' || last == 'A':
		return name + "n"
	case isVowel(lowerByte(last)):
		return name[:len(name)-1] + "ian"
	default:
		return name + "ian"
	}
}

func lowerByte(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c - 'A' + 'a'
	}
	return c
}
