package namegen

// Pseudo-syllable chain generation, ported from Azgaar's Fantasy Map
// Generator (src/generators/names-generator.ts), MIT License,
// Copyright (c) Azgaar. https://github.com/Azgaar/Fantasy-Map-Generator
//
// Method: each seed word is split into short pseudo-syllables; syllables are
// bucketed by the letter that precedes them (empty/zero key = word start,
// empty syllable = end-of-word marker). Generation walks the chain, picking a
// random syllable keyed by the last letter emitted, within per-base min/max
// length bounds, then applies cleanup rules (duplicate letters, "ae" -> "e",
// triple letters, capitalization after space/hyphen).

import "strings"

const vowels = "aeiouy"

func isVowel(c byte) bool {
	return strings.IndexByte(vowels, c) >= 0
}

// syllableChain maps a preceding letter (0 for word start) to the syllables
// that may follow it. An empty syllable marks a possible word end.
type syllableChain map[byte][]string

// buildChain splits every lexicon word into pseudo-syllables keyed by the
// preceding letter. Port of FMG's calculateChain; the lexicons embedded here
// are all printable ASCII, so the English-style diphthong rules always apply.
func buildChain(words []string) syllableChain {
	chain := make(syllableChain)
	for _, raw := range words {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		for i := -1; i < len(name); {
			var prev byte // letter before the syllable; 0 at word start
			if i >= 0 {
				prev = name[i]
			}
			syll := ""
			hasVowel := false
			for c := i + 1; c < len(name) && len(syll) < 5; c++ {
				that := name[c]
				var next byte
				if c+1 < len(name) {
					next = name[c+1]
				}
				syll += string(that)
				if syll == " " || syll == "-" {
					break // syllable is just a separator
				}
				if next == 0 || next == ' ' || next == '-' {
					break // word or part boundary
				}
				if isVowel(that) {
					hasVowel = true
				}
				// keep some diphthongs together
				if that == 'y' && next == 'e' {
					continue // "ye"
				}
				if that == 'o' && next == 'o' {
					continue // "oo"
				}
				if that == 'e' && next == 'e' {
					continue // "ee"
				}
				if that == 'a' && next == 'e' {
					continue // "ae"
				}
				if that == 'c' && next == 'h' {
					continue // "ch"
				}
				// Note: FMG has a comparison here (isVowel(that) === next as
				// boolean) that is always false in JS; intentionally omitted.
				if hasVowel && c+2 < len(name) && isVowel(name[c+2]) {
					break // syllable has a vowel and another vowel is near
				}
			}
			chain[prev] = append(chain[prev], syll)
			if len(syll) > 0 {
				i += len(syll)
			} else {
				i++
			}
		}
	}
	return chain
}

// generateWord walks the chain to produce one raw lowercase word within
// [min, max] length bounds. Port of FMG's getBase main loop.
func generateWord(r *rng, chain syllableChain, min, max int) string {
	starts := chain[0]
	if len(starts) == 0 {
		return ""
	}
	v := starts
	cur := v[r.intn(len(v))]
	w := ""
	for i := 0; i < 20; i++ {
		if cur == "" {
			// end-of-word marker reached
			if len(w) < min {
				w = "" // too short: restart
				v = starts
			} else {
				break
			}
		} else {
			if len(w)+len(cur) > max {
				// word would get too long
				if len(w) < min {
					w += cur
				}
				break
			}
			nv, ok := chain[cur[len(cur)-1]]
			if !ok || len(nv) == 0 {
				nv = starts
			}
			v = nv
		}
		w += cur
		cur = v[r.intn(len(v))]
	}
	return w
}

// cleanupWord applies FMG's post-processing: strip junk trailing characters,
// collapse disallowed duplicate letters, capitalize word starts and letters
// after space/hyphen, rewrite "ae" to "e", drop triple letters, and join
// multi-word names when any part is a single letter. As an extra guarantee
// beyond FMG, any remaining run of three identical letters is collapsed.
// dupl lists the letters allowed to appear doubled.
func cleanupWord(w, dupl string) string {
	for len(w) > 0 {
		last := w[len(w)-1]
		if last == '\'' || last == ' ' || last == '-' {
			w = w[:len(w)-1]
		} else {
			break
		}
	}

	var out []byte
	for i := 0; i < len(w); i++ {
		c := w[i]
		if i+1 < len(w) && c == w[i+1] && strings.IndexByte(dupl, c) < 0 {
			continue // duplication not allowed for this letter
		}
		if len(out) == 0 {
			out = append(out, upperByte(c))
			continue
		}
		last := out[len(out)-1]
		if last == '-' && c == ' ' {
			continue // no space after hyphen
		}
		if last == ' ' || last == '-' {
			out = append(out, upperByte(c)) // capitalize after separator
			continue
		}
		if c == 'a' && i+1 < len(w) && w[i+1] == 'e' {
			continue // "ae" -> "e"
		}
		if i+2 < len(w) && c == w[i+1] && c == w[i+2] {
			continue // no three same letters in a row
		}
		out = append(out, c)
	}
	name := string(out)

	// join the name if any space-separated part is a single letter
	parts := strings.Split(name, " ")
	if len(parts) > 1 {
		short := false
		for _, p := range parts {
			if len(p) < 2 {
				short = true
				break
			}
		}
		if short {
			for j := 1; j < len(parts); j++ {
				parts[j] = strings.ToLower(parts[j])
			}
			name = strings.Join(parts, "")
		}
	}

	return collapseTriples(name)
}

// collapseTriples reduces any run of three or more identical letters
// (case-insensitively, so "Sss" is caught) to two.
func collapseTriples(s string) string {
	var out []byte
	for i := 0; i < len(s); i++ {
		n := len(out)
		if n >= 2 && lowerByte(out[n-1]) == lowerByte(s[i]) && lowerByte(out[n-2]) == lowerByte(s[i]) {
			continue
		}
		out = append(out, s[i])
	}
	return string(out)
}

func upperByte(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - 'a' + 'A'
	}
	return c
}
