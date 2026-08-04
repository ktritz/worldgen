# EcoCrop source data (vendored)

Provenance for the crop climate envelopes. Vendored because FAO's own EcoCrop web UI is **dead**
(`ecocrop.fao.org` / `ecocrop.review.fao.org` now redirect to Google IAP auth), so these community
copies are the source of record.

## Canonical source: the 2023 full-datasheet scrape

| File | What it is |
|------|-----------|
| `cropbasics_scrape.csv.gz` | **2568 records × 63 columns**, FAO datasheets scraped ~2023 (`supersistence/EcoCrop-ScrapeR`). This is the canonical table. |
| `crop_view_data.csv.gz` | 2567 records × 8 columns, joined on `crop_code`/`Ecocrop_code`. Carries per-crop **free-text `Notes`** (2049 filled, mean 822 chars) plus `Sources`, synonyms, authority. |

## Superseded, kept as a cross-check

| File | What it is |
|------|-----------|
| `ecocrop.rds` | 1710 records × 35 columns, ~2019 vintage, from the R **Recocrop** package. What `crops_earthlike_v1.json` was originally built from. Retain to cross-check the 1635 overlapping species. |
| `parameters.Rd` | Recocrop's column documentation — the authoritative `KTMP` definition and its "User beware: these parameters are expert opinion" warning. |
| `Recocrop_LICENSE` | GPL (>=3), the *package* license. |
| `rds2.py` | Dependency-free Python reader for R `.rds`/`.RData` (no R toolchain needed). Reusable for re-diffing. |

The `dismo` R package ships the *same* 1710×35 table as Recocrop — identical values, only cosmetic
name differences. They are one dataset at one vintage, not independent sources.

## Why we switched — the killing-temperature split

The 2019 table has a single `KTMP` column. **FAO's actual datasheets have two**:
`Killing.temp..during.rest` and `Killing.temp..early.growth`. The single column collapsed them
inconsistently (it equals during-rest in 775 matched rows and early-growth in 41), and for exactly
the crops whose winter hardiness matters it kept the *early-growth* value:

| crop | our old `KTMP` | FAO during rest | FAO early growth |
|------|----------------|-----------------|------------------|
| wheat (*Triticum aestivum*) | 0 | **−20** | 0 |
| rye (*Secale cereale*) | 0 | **−18** | −1 |

Verified first-hand against the vendored file. This is precisely the two-field split we had
independently proposed adding by hand — it already exists upstream, so the hand-sourced
"consensus" overrides are largely unnecessary. Re-derive from `cropbasics_scrape.csv.gz` instead.

The scrape also adds ~830 species we lacked, plus Climate.Zone, photoperiod, Life.span,
latitude/altitude envelopes, cropping system, and uses. **No mirror has an explicit winter-vs-spring
type column** — but 263 crops' `Notes` discuss winter/vernalization/sowing, e.g. rye's:
*"Seedlings of winter rye may tolerate -18°C … mature in 110-130 days if spring sown and in 210-270
days if autumn sown … Autumn sown cultivars require exposure to low temperatures as a prerequisite
to flowering."*

## Known traps (still apply)

1. **`KTMP` is not winter hardiness.** `src/ecocrop.cpp`'s `movingmin_circular()` only scores months
   inside the candidate growing season, so overwintering is structurally invisible to the model.
2. **The old table's zeros are a packaging artifact — the 2023 table is clean.** FAO's datasheets
   use an explicit string sentinel `"no input"` for unmeasured values, and `Killing.temp..during.rest`
   contains **not a single literal 0 in all 2568 rows** (1410 `"no input"`, 1158 real numbers).
   Since nobody ever typed 0 in that column, the 390 zeros in `Killing.temp..early.growth` are
   genuine measurements. The 819 zeros (47.9%) in the old `ecocrop.rds` are `"no input"` **coerced
   to 0 by the Recocrop packagers** — i.e. the 2019 copy is not merely reduced, it is corrupted in
   a way that silently turns "unknown" into "dies at 0 °C".
   **Parsing rule: `"no input"` → null; a literal 0 → keep as a measurement.** This is the inverse
   of the rule the old table appeared to require.
3. The 2019 and 2023 tables **disagree on 4–8% of values per field** (TMIN 102 rows, TOPMN 131,
   TMAX 101, RMIN 126, PHMAX 108, GMAX 75) — different vintages, so don't mix them silently.

## Licensing

Short version: **the numbers are facts and we can use them freely. Attribute FAO. The only open
question is whether to redistribute these tables verbatim.**

**Using the facts is fine.** Under US law facts are not copyrightable — *Feist v. Rural Telephone*
(1991) rejected the "sweat of the brow" doctrine and held that a compilation earns only thin
protection for its original selection, coordination, and arrangement, which does not reach the
underlying data. "Wheat's killing temperature during rest is −20 °C" is a fact about wheat.
Extracting envelope values into our catalog, computing suitability from them, and shipping a
simulation whose behavior depends on them is not copying anyone's expression. **Derived output —
`crops_earthlike_v1.json` and anything the sim produces from it — is ours, unambiguously.**

**What the thin compilation right actually covers** is wholesale reproduction preserving FAO's
selection and arrangement. That is the one thing this directory does: `cropbasics_scrape.csv.gz`
and `ecocrop.rds` are verbatim copies. That is appropriate here — they are the provenance record
that makes the catalog auditable and re-derivable — but it is the piece to revisit if this repo
goes public or ships in a distributed artifact. Shipping only the ~28-crop derived catalog avoids
the question entirely.

**Two caveats that don't block us:**
- *Jurisdiction.* The EU's sui generis database right (Directive 96/9/EC) protects substantial
  investment in obtaining or verifying contents — essentially the doctrine *Feist* rejected — and
  restricts extraction of substantial parts even absent copyright. A derived catalog of ~28 crops
  is nowhere near a "substantial part" of 2568 records, but it is a different analysis from US law
  if this ever distributes into the EU.
- *Terms of use are contract, not copyright.* FAO could impose conditions by agreement that
  copyright would not. In practice FAO data is generally permissive (CC BY-type), which cuts in our
  favour — the real obligation is attribution, which the schema's `envelope_source` /
  `envelope_ref` fields already satisfy per crop.

**Attribution: credit FAO, not the mirrors.** The scraper repo has no LICENSE file and the GPL
covers the Recocrop *package*, not the data; neither mirror originated these facts.

Not legal advice — if this becomes a commercial release, have counsel confirm FAO's terms.
