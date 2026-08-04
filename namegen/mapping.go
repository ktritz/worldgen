package namegen

// Base keys for the embedded name bases. Real-world bases carry the flavor
// of a toponym tradition; fantasy bases come from FMG's curated fantasy
// lexicons (see bases_data.go for attribution).
const (
	BaseGermanic = "germanic"
	BaseNordic   = "nordic"
	BaseRoman    = "roman"
	BaseArabic   = "arabic"
	BaseCeltic   = "celtic"
	BaseSlavic   = "slavic"
	BaseGreek    = "greek"
	BaseElven    = "elven"
	BaseDwarven  = "dwarven"
	BaseGoblin   = "goblin"
	BaseOrcish   = "orcish"
	BaseDraconic = "draconic"
)

// DefaultBase is used when a base, culture, or ancestry key is unknown.
const DefaultBase = BaseGermanic

// CultureBase maps the project's fantasy culture profile names
// (config/profiles/fantasy/cultures/*.json file keys) to name bases.
// climgen can join on these keys when wiring names into review summaries.
// Keys are normalized (see normalizeKey), so display names also resolve.
var CultureBase = map[string]string{
	"desert_caravan":    BaseArabic,   // caravan cities and oasis emirates
	"forest_enclave":    BaseCeltic,   // woodland enclaves, Brythonic flavor
	"highland_hold":     BaseNordic,   // mountain clan holds, Norse flavor
	"imperial_mandate":  BaseRoman,    // centralized empire, Latin flavor
	"iron_legion":       BaseRoman,    // martial legions, Latin flavor
	"marsh_clans":       BaseSlavic,   // wetland clans, Old East Slavic flavor
	"merchant_league":   BaseGermanic, // Hanseatic-style trade league
	"river_confederacy": BaseSlavic,   // riverine confederation
	"sky_aerie":         BaseGreek,    // high aeries, Classical Greek flavor
}

// AncestryBase maps the project's fantasy ancestry profile names
// (config/profiles/fantasy/ancestries/*.json file keys) to name bases.
var AncestryBase = map[string]string{
	"aarakocra":  BaseGreek,
	"dragonborn": BaseDraconic,
	"dwarf":      BaseDwarven,
	"elf":        BaseElven,
	"gnoll":      BaseOrcish,
	"gnome":      BaseGermanic,
	"goblin":     BaseGoblin,
	"halfling":   BaseCeltic,
	"hobgoblin":  BaseOrcish,
	"human":      BaseGermanic,
	"lizardfolk": BaseDraconic,
	"orc":        BaseOrcish,
}
