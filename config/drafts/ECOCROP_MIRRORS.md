# EcoCrop mirrors — comparison (investigated 2026-08-02)

FAO's live UI (ecocrop.fao.org) is dead (redirects to Google IAP). All surviving copies are
third-party. Verdict up front: **yes, better mirrors exist — we should switch.**

## Comparison

| Mirror | Records | Cols | Vintage | Access | Adds over Recocrop? |
|---|---|---|---|---|---|
| **Recocrop** `inst/parameters/ecocrop.rds` (current) | 1710 | 35 | ~2019 FAO snapshot | local | baseline |
| **dismo** `ECOcrops.RData` (CRAN 1.3-16) | 1710 | 35 | data file dated 2018-07-30 | CRAN tarball | **Nothing.** Byte-identical values; only diffs are cosmetic (dismo strips the trailing ` *` from 1265 NAMEs, ASCII-folds 2 SCIENTNAMEs). Same table, same vintage. |
| **supersistence/EcoCrop-ScrapeR** `cropbasics_scrape.csv` | 2568 | **63** | scrape of FAO ~2023 | GitHub raw | **Everything below.** Full FAO datasheet scrape. |
| ↳ same repo `crop_view_data.csv` | 2567 | 8 | same | GitHub raw | **Free-text `Notes` (2049 filled, mean 822 chars) + `Sources` (2033) + Synonyms/Common_names/Authority.** |
| **PyPI `ecocrop` 0.2.0** `data/Ecocrop_updated_1312023.xlsx` | 2568 | 60 | 2023-01-13 | pip/sdist | Same scrape as above minus 5 cultivation cols, plus `Communities`/`Common_names`. Code is MIT, a Recocrop reimplementation; data is the redistributed FAO scrape. |
| **Zenodo 13319716 "ECOCROP"** (DwC-A) | 2566 taxa / 47k facts | — | 2024-08-14 | Zenodo API | Same underlying scrape (contributor field credits supersistence), reshaped to Darwin Core long-form. Lossy vs the CSV; no notes text. License "notspecified". |
| Wayback `ecocrop.fao.org/.../dataSheet?id=N` | ~1000 pages archived | — | 2008–2019 | CDX + `id_` replay | Renders the same field set as the 63-col scrape. Usable as spot-check ground truth, not as a bulk source. |
| Other Zenodo/Dataverse/Figshare hits | — | — | — | — | All are *applications* of EcoCrop (suitability maps, R scripts), not database copies. |

## Key question

- **(a) >1710 records?** Yes — 2568 (FAO's own stated species count). 1635/1710 of our rows match a
  species in the 2568 set; ~830 species are new to us.
- **(b) Fields beyond our 35?** Yes, ~28 more. Directly relevant: **`Killing.temp..during.rest` and
  `Killing.temp..early.growth`** (our single `KTMP` collapses these — it equals *during-rest* in 775
  matched rows and *early-growth* in 41, so it is not a consistent quantity), plus `Climate.Zone`,
  `photoperiod`, `Life.span`, `Plant.attributes`, `Abiotic.toler.`/`suscept.`, latitude and altitude
  envelopes (opt+abs), `Cropping.system`, `Product..system`, `Level.of.mechanization`,
  `Labour.intensity`, `use.main`/`detailed`/`part`. No explicit winter-vs-spring *column* exists.
- **(c) Free-text notes?** **Yes** — `crop_view_data.csv` `Notes`. 263 crops' notes mention
  winter/vernalization/sowing. Rye's reads: *"KILLING T Seedlings of winter rye may tolerate -18°C …
  mature in 110-130 days if spring sown and in 210-270 days if autumn sown … Autumn sown cultivars
  require exposure to low temperatures as a prerequisite to flowering."* This is the winter-hardiness
  and sowing-season signal the numeric table lacks.

## Value disagreements (different vintages, not transcription noise)

Recocrop vs the 2023 scrape, joined on binomial (n≈1632 per field): TMIN 102 disagree, TOPMN 131,
TOPMX 114, TMAX 101, RMIN 126, RMAX 120, GMIN 67, GMAX 75, PHMIN 92, PHMAX 108 — a 4–8% per-field
drift. Examples: Red clover TMAX 26 (ours) vs 40; Garden beet GMAX 90 vs 240; Celery GMIN 80 vs 40.
Headline case — **wheat `KTMP` = 0 °C in our table, but FAO 2023 gives killing-temp-during-rest
-20 °C / early-growth 0 °C; rye 0 vs -18 / -1; olive -10 during rest.** Our current catalog therefore
encodes the *early-growth* frost limit for the very crops whose winter hardiness we care about.
Recocrop and dismo agree with each other exactly, so the drift is FAO's 2019→2023 update.

## Recommendation

Treat **supersistence/EcoCrop-ScrapeR** as canonical: `cropbasics_scrape.csv` (63 cols, 2568 rows)
joined to `crop_view_data.csv` on `crop_code`/`Ecocrop_code` for `Notes`. It is the most complete,
most recent, and the only one carrying descriptive text; the PyPI xlsx and the Zenodo DwC archive are
derivatives of it. Keep `ecocrop.rds` as a cross-check for the 1635 overlapping species. **Stop
treating Recocrop/dismo as the source** — they are one dataset, one vintage, and demonstrably stale.
Caveat: the scraper repo carries no LICENSE file; the upstream data is FAO's (© FAO, EcoCrop
1993–2007 era attribution on the datasheets), so attribute FAO rather than the mirror.
