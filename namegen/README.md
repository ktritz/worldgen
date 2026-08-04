# namegen

Deterministic fantasy name generation for polities, settlements, rivers, seas, and mountain ranges, keyed to the project's 9 fantasy cultures and 12 ancestries (`CultureBase` / `AncestryBase`).

Method and seed lexicons are ported from [Azgaar's Fantasy Map Generator](https://github.com/Azgaar/Fantasy-Map-Generator) (MIT License, Copyright (c) Azgaar): pseudo-syllable chains keyed by preceding letter, built from 12 embedded name bases, with FMG's per-base length/duplication tuning and cleanup rules.

**Determinism contract:** every name is a pure function of `(Namer seed, base, method, key)` — call order and call count never matter. Pass a stable key per call site (e.g. `Settlement(fmt.Sprintf("settlement/%d", cellID))`); `Namer.Derive(key)` provides raw sub-seeds the same way.

TODO (wiring into review summaries later):
- Join `CultureBase` on polity culture profiles in climgen and name polities/capitals in `cmd/review_planets` output.
- Name major rivers/seas/ranges from landgen feature IDs (use feature ID as the key).
- Align `PolitySizeClasses` with real polity tiers once climgen defines them.
