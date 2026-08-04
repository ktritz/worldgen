// Package namegen provides deterministic fantasy name generation for
// polities, settlements, rivers, seas, and mountain ranges, keyed to the
// project's fantasy cultures and ancestries.
//
// The generation method (pseudo-syllable chains built from per-base seed
// lexicons, with cleanup rules) and the embedded lexicons are ported from
// Azgaar's Fantasy Map Generator, MIT License, Copyright (c) Azgaar.
// https://github.com/Azgaar/Fantasy-Map-Generator
//
// # Determinism contract
//
// Every generated name is a pure function of:
//
//	(Namer seed, name base, method, caller-supplied key)
//
// Call order never matters: each call hashes those four values into a
// private PRNG, so callers get stable names regardless of when or how often
// other names are generated. Pass a stable key per call site (for example a
// settlement's cell index: fmt.Sprintf("settlement/%d", cellID)).
// Namer.Derive offers the same splitting for callers that need raw
// sub-seeds.
package namegen

import (
	"strings"
	"sync"
)

// Namer is the root generator. It is cheap to construct and safe for
// concurrent use; all state is immutable after construction.
type Namer struct {
	seed int64
}

// New returns a Namer rooted at the given world seed.
func New(seed int64) *Namer {
	return &Namer{seed: seed}
}

// Seed returns the root seed the Namer was constructed with.
func (n *Namer) Seed() int64 { return n.seed }

// Derive deterministically splits the Namer's seed with a string key,
// returning a stable sub-seed. Derive(k) is constant for a given (seed, k)
// and independent of any other calls.
func (n *Namer) Derive(key string) int64 {
	return int64(hashSeed(n.seed, "derive", key))
}

// Base returns the BaseNamer for a base key (see BaseNames). Unknown keys
// fall back to DefaultBase so callers always get a usable generator.
func (n *Namer) Base(name string) *BaseNamer {
	b, ok := basesByKey[normalizeKey(name)]
	if !ok {
		b = basesByKey[DefaultBase]
	}
	return &BaseNamer{namer: n, base: b}
}

// ForCulture returns the BaseNamer mapped to a culture profile name (see
// CultureBase). Names are normalized, so "Highland Hold" and "highland_hold"
// are equivalent. Unknown cultures fall back to DefaultBase.
func (n *Namer) ForCulture(culture string) *BaseNamer {
	if key, ok := CultureBase[normalizeKey(culture)]; ok {
		return n.Base(key)
	}
	return n.Base(DefaultBase)
}

// ForAncestry returns the BaseNamer mapped to an ancestry profile name (see
// AncestryBase). Unknown ancestries fall back to DefaultBase.
func (n *Namer) ForAncestry(ancestry string) *BaseNamer {
	if key, ok := AncestryBase[normalizeKey(ancestry)]; ok {
		return n.Base(key)
	}
	return n.Base(DefaultBase)
}

// BaseNames lists the available base keys in stable order.
func BaseNames() []string {
	keys := make([]string, len(baseSpecs))
	for i := range baseSpecs {
		keys[i] = baseSpecs[i].Key
	}
	return keys
}

// BaseNamer generates names in the style of one name base. Obtain one via
// Namer.Base, Namer.ForCulture, or Namer.ForAncestry.
type BaseNamer struct {
	namer *Namer
	base  *nameBase
}

// BaseName returns the base key this namer draws from (e.g. "celtic").
func (b *BaseNamer) BaseName() string { return b.base.spec.Key }

// Place generates a generic toponym. key selects the name deterministically
// (see the package determinism contract).
func (b *BaseNamer) Place(key string) string {
	return b.word("place", key, b.base.spec.Min, b.base.spec.Max, b.base.spec.Dupl)
}

// Settlement generates a settlement name.
func (b *BaseNamer) Settlement(key string) string {
	return b.word("settlement", key, b.base.spec.Min, b.base.spec.Max, b.base.spec.Dupl)
}

// Polity generates a polity name for a size class (see PolitySizeClasses),
// applying a morphology table keyed to size class and base, e.g.
// "Kingdom of X", "X Empire", "Xian League". Unknown size classes fall back
// to the "kingdom" forms.
func (b *BaseNamer) Polity(sizeClass, key string) string {
	sizeClass = normalizeKey(sizeClass)
	r := b.rng("polity", sizeClass, key)
	core := b.wordWith(r, b.base.spec.Min, b.base.spec.Max, b.base.spec.Dupl)
	core = singleWord(core)
	tmpl := politeTemplate(b.base.spec.Key, sizeClass, r)
	return expandTemplate(tmpl, core)
}

// River generates a river name such as "Nagold River" or "River Tave".
// River, Sea, and Range use slightly shortened core names, following FMG.
func (b *BaseNamer) River(key string) string {
	r := b.rng("river", key)
	core := singleWord(b.shortWordWith(r))
	return expandTemplate(riverTemplates[r.intn(len(riverTemplates))], core)
}

// Sea generates a sea, gulf, or bay name such as "Sea of Kandia".
func (b *BaseNamer) Sea(key string) string {
	r := b.rng("sea", key)
	core := singleWord(b.shortWordWith(r))
	return expandTemplate(seaTemplates[r.intn(len(seaTemplates))], core)
}

// Range generates a mountain-range name such as "the Brei Peaks".
func (b *BaseNamer) Range(key string) string {
	r := b.rng("range", key)
	core := singleWord(b.shortWordWith(r))
	return expandTemplate(rangeTemplates[r.intn(len(rangeTemplates))], core)
}

// --- internals ---

func (b *BaseNamer) rng(parts ...string) *rng {
	all := append([]string{b.base.spec.Key}, parts...)
	return newRNG(hashSeed(b.namer.seed, all...))
}

func (b *BaseNamer) word(method, key string, min, max int, dupl string) string {
	return b.wordWith(b.rng(method, key), min, max, dupl)
}

func (b *BaseNamer) wordWith(r *rng, min, max int, dupl string) string {
	raw := generateWord(r, b.base.chain(), min, max)
	name := cleanupWord(raw, dupl)
	if len(name) < 2 {
		name = b.base.fallback(r, max)
	}
	return name
}

// shortWordWith mirrors FMG's getBaseShort: min-1 / max-2 (clamped) and no
// duplicated letters, giving crisper names for physical features.
func (b *BaseNamer) shortWordWith(r *rng) string {
	min := b.base.spec.Min - 1
	if min < 2 {
		min = 2
	}
	max := b.base.spec.Max - 2
	if max < min {
		max = min
	}
	return b.wordWith(r, min, max, "")
}

// singleWord collapses multi-word names ("Bad Hall" -> "Badhall") so they
// compose cleanly with templates like "Sea of %s".
func singleWord(name string) string {
	if !strings.Contains(name, " ") {
		return name
	}
	parts := strings.Split(name, " ")
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "")
}

// nameBase pairs a spec with its lazily-built syllable chain.
type nameBase struct {
	spec  baseSpec
	words []string

	once  sync.Once
	built syllableChain
}

func (b *nameBase) chain() syllableChain {
	b.once.Do(func() { b.built = buildChain(b.words) })
	return b.built
}

// fallback deterministically picks a lexicon word no longer than max+2,
// used when a generated name degenerates (FMG falls back the same way).
func (b *nameBase) fallback(r *rng, max int) string {
	start := r.intn(len(b.words))
	for i := 0; i < len(b.words); i++ {
		w := b.words[(start+i)%len(b.words)]
		if len(w) <= max+2 {
			return w
		}
	}
	return b.words[start]
}

var basesByKey = func() map[string]*nameBase {
	m := make(map[string]*nameBase, len(baseSpecs))
	for i := range baseSpecs {
		spec := baseSpecs[i]
		m[spec.Key] = &nameBase{spec: spec, words: strings.Split(spec.Lexicon, ",")}
	}
	return m
}()

// normalizeKey lowercases and converts spaces/hyphens to underscores so
// profile display names ("Highland Hold") and file keys ("highland_hold")
// address the same entries.
func normalizeKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}
