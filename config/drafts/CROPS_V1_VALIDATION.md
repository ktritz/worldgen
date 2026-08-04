# crops_earthlike_v1 — KTMP validation

Method: pulled `inst/parameters/ecocrop.rds` from `cropmodels/Recocrop@master`, parsed the R
serialization directly, and diffed all 1710 source records against the JSON. FAO's own EcoCrop web UI
is **unreachable** (`ecocrop.review.fao.org` is behind Google IAP auth). Web-search budget was
exhausted mid-task; rows marked *(consensus)* have no retrieved URL — treat as lower confidence.

## (a) Verdict: KTMP is NOT winter hardiness. Your hypothesis is confirmed.

1. **Definition.** `man/parameters.Rd`: *"ktmp: The minimum and lower optimum threshold for extreme
   minimum temperature (killing temperature)... the temperature below which the plant dies."* Same file
   opens with **"User beware: These parameters are expert opinion and not necessarily optimal."**
   (https://github.com/cropmodels/Recocrop/blob/master/man/parameters.Rd)
2. **Scope is the growing season only — proven in code.** `src/ecocrop.cpp` `movingmin_circular()` takes
   a moving minimum of extreme-min-temp over a window equal to the crop's `duration`, and the model
   slides that window to find the best planting month. Months *outside* the candidate season are never
   scored. An overwintering crop's dormant period is therefore structurally invisible.
3. **The schema cannot express a winter type.** The 35 columns are NAME/FAMNAME/SCIENTNAME/CODE, GMIN,
   GMAX, KTMP, TMIN..TMAX, RMIN..RMAX, LIG/LIGR, PP/PPMIN/PPMAX, TEXT, DEP, DRA, PH, SAL, FER, LIMITS.
   There is **no vernalization field, no sowing-season field, no dormancy field**. Winter wheat needs
   30–60 d at 0–5 °C to head (https://en.wikipedia.org/wiki/Winter_wheat) — EcoCrop has nowhere to put that.
4. **No winter-type records exist.** I grepped all 1710. There is no "Rye, winter" or "Wheat, winter".
   Every *Triticum* record is KTMP 0 or −3 (aestivum 0, durum 0, spelt 0, emmer 0, club −3); every
   *Avena* is −10/−15 (sativa −15, fatua −15, byzantina −10, sterilis −10); *Hordeum vulgare* −4;
   *Secale cereale* 0 and *S. montanum* 0. So the rye/oat inversion is **systematic at genus level, not
   a transcription typo** — and it is not rescued by picking a different record.
5. **Values are coarse expert bins.** 819 of 1710 records (**47.9 %**) are exactly `0`; next commonest
   are −5 (216), −2 (136), −10 (94). Only 1 record is NA. That distribution is a defaulting artifact:
   `0` largely means "frost-sensitive / not assessed", not a measured 0 °C.
6. **Tell-tale:** apple KTMP −2. Dormant apple wood survives roughly −35 °C; −2 is its *flower/active
   tissue* number. Same for grape 0 and olive 0. Read as active-growth frost sensitivity most values
   are defensible; read as winter hardiness they are nonsense. Oats −15 is wrong under **either** reading.

**Recommendation: store two fields.** Keep `seasonality.frost_kills_at_c` = KTMP (actively growing
tissue) and add `seasonality.winter_hardy_to_c` (cold-hardened dormant plant; null where there is no
overwintering phase). Every anomaly you flagged is the collapse of these two. Only `winter_hardy_to_c`
should gate placement of autumn-sown cereals and perennials.

## (b) Recommended overrides

| crop | KTMP now | `frost_kills_at_c` (growing) | `winter_hardy_to_c` (hardened) | basis |
|---|---|---|---|---|
| rye | 0 | −4 | **−25 to −30** (w/ snow cover) | "hardiest of cereals", "more cold-tolerant than wheat"; "survives snow cover that would kill winter wheat" — [SARE](https://www.sare.org/publications/managing-cover-crops-profitably/nonlegume-cover-crops/cereal-rye/), [WP:Rye](https://en.wikipedia.org/wiki/Rye) *(magnitude: consensus)* |
| wheat | 0 | −4 | **−15 to −20** (winter type; spring type: null) | vernalization 0–5 °C/30–60 d [WP](https://en.wikipedia.org/wiki/Winter_wheat); LT50 *(consensus)* |
| barley | −4 | −4 (keep) | **−12** (winter type; spring: null) | least hardy of the true winter cereals *(consensus)* |
| oats | **−15 → wrong** | −5 | **−8 to −10** (winter oat only) | oats are the least winter-hardy small grain; EcoCrop value indefensible either way *(consensus)* |
| olive | 0 | 0 (keep) | **−10 tree injury; ~−12 wood kill** | "below −10 °C may injure even a mature tree" [WP:Olive](https://en.wikipedia.org/wiki/Olive); USDA **8a** = −12.2…−9.4 °C, some cvs 7a [NCSU](https://plants.ces.ncsu.edu/plants/olea-europaea/) |
| fig | −12 | −2 | **−15 to −18 top-kill**, roots resprout below | USDA **7a** = −17.8…−15.0 °C [NCSU](https://plants.ces.ncsu.edu/plants/ficus-carica/) |
| grape (vinifera) | 0 | −1 to −2 (shoots) | **−18 to −22** (dormant buds/canes) | *(consensus — MSU/WSU pages 404'd)* |
| date palm | −4 | −4 (keep) | **−6** (leaf burn; mature palm survives brief −9) | "short periods of frost as low as −5 °C" [WP](https://en.wikipedia.org/wiki/Date_palm) |
| potato | −1 | −1 (keep — correct) | **null** (no overwintering phase) | foliage killed at −1 to −2 *(consensus)* |

**Fig vs olive:** the *direction* is right — fig really is hardier — but by **~1 USDA zone (≈5 °C)**,
not 12 °C. Fig −15/−18 vs olive −10/−12 keeps both inside the Mediterranean band.

## (c) Parse spot-check: clean

All **20** EcoCrop-sourced crops diffed field-by-field against the source table: **13/13 numeric fields
exact for 20/20 crops, zero mismatches** (TMIN/TOPMN/TOPMX/TMAX, RMIN/ROPMN/ROPMX/RMAX, GMIN/GMAX,
PHMIN/PHMAX, KTMP). Wheat, rice, barley, olive and date palm specifically verified. The extraction is
faithful; the defect is entirely upstream in EcoCrop's semantics. (Only cosmetic note: `envelope_ref`
strips FAO's trailing `*` from `Common fig *` and `Common indigo *`.)

## (d) Other things these findings cast doubt on

- **`madder` KTMP −12 is invented**, not sourced (`envelope_source: estimated`) yet sits on the same
  scale as the EcoCrop values. Under the two-field split it should be `winter_hardy_to_c: −20` (hardy
  rootstock) and `frost_kills_at_c: −2` (top growth) — currently one number does neither job.
- **`highland_tuber`**: KTMP −1 is *correct* for potato; the problem is a different one — the record is
  a lowland envelope (TOPMN 15/TOPMX 25). *Potato, Bitter* (**Solanum × juzepczukii**, TMIN 3 / TOPMN 6
  / TOPMX 14 / TMAX 18, KTMP −5, GMIN 150/GMAX 195) exists and is confirmed in the table. It is the
  actual chuño cultivar and the only record whose temperature band is genuinely alpine. Recommend
  switching, or carrying both as separate crops.
- **`sow_seasons: [autumn, spring]`** on wheat is doing work no envelope field supports. Once
  `winter_hardy_to_c` exists, autumn sowing must be *gated* on it — otherwise autumn-sown wheat is
  placed by a 0 °C kill line and will be wiped out everywhere it historically thrived.
- **The 47.9 % zeros are a landmine at scale.** Ten of the 21 crops carry KTMP 0. If any call site
  treats 0 as "dies at first frost", half the catalogue shares one arbitrary threshold. Consider
  storing KTMP 0 as `null`/"unassessed" rather than a number.
- **Non-KTMP fields inherit the same "expert opinion" warning.** `LIMITS='U'` is set on wheat and date
  palm (unset on barley/rice/olive) and is not carried into the JSON — worth checking what it flags.
  Separately, 1265/1710 records carry a `*` name flag; the 445 unflagged (wheat, barley, oats, rye,
  rice, olive, grape, date palm, potato) look like the better-curated core set. `fig` and `indigo` are
  flagged — mild extra reason to distrust fig's −12.
