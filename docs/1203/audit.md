# docs/1203/audit.md -- the divergence ledger's own audit

Part of story1203 (`docs/1203/story.md`), required by the owner's standing
request to audit the ledger's open rows. This is a REPORT: it changes no
row's `Status` or `Type`. The owner decides what to do with each finding
below; this document only names what was checked, over what population, and
what it found, per this project's own rule that a negative result says what
was searched and what was not found.

Every part reads the 507 live rows now split under `docs/divergences/` (the
same set the split in `docs/1203/story.md` produced from the 510-row single
file at `07cddcf`, minus the 3 deleted `CLOSED` rows).

## (a) OPEN UNKNOWN rows whose silence a new claim may have broken

`pipeline/check-div-claims.sh` finds the CITED half of claim staleness: a row
citing a claim id that a later retraction table row names. It cannot find the
UNCITED half: a row whose `ROM1 behaviour (claims)` cell asserts research is
silent on a question, when a published claim written after that row now
answers it. That half has no automated check; the index's own "An UNKNOWN
row's ROM1 cell goes stale silently" section names this as a hand-read
problem and points here.

**Population.** 201 rows carry `Type: UNKNOWN` in the split ledger (matches
the brief's own starting-state count). Of those, 51 carry the specific
`"No claim..."`/`"No promoted claim..."`/`"No claim states/establishes/
identifies/addresses/names"` phrasing that names an explicit silence
assertion, as opposed to an `UNKNOWN` row that partially cites claims and
narrows a question without claiming total silence -- these 51 are the rows
where a single new claim could most directly close the gap this audit looks
for.

**What was checked.** Of the 51 silence-assertion rows, this pass ran a
targeted `go run ./tools/claim -k "<regexp>"` search against the full research
claims corpus for the specific technical noun each row's own cell names
(route/wall invalidation, stacked effect clamps, weight/capacity fields,
`VirtualCaster` SAV reach, formation speed, Prismatic Spray victim selection),
covering 11 of the 51 rows. This is a bounded, not exhaustive, pass: the
remaining 40 were read but not each individually searched against the corpus.

**Found: one confirmed break.**

- **`DIV-035`** (`docs/divergences/simulation.md`) -- cell reads "No claim
  establishes what the original does to a stored route when a wall closes one
  of its cells." `MOVE-REFRESH-012` (`knowledge/claims/move.md:39`, promoted
  via `EXP-0054`) states plainly: "Nothing invalidates a stored route on
  terrain change or on target movement: only a new target cell, a completed
  step, the tick counter, and a blocked adjacent waypoint via `MOVE-WAIT-008`."
  A wall closing a cell is exactly a terrain change, and the claim says that
  alone does not invalidate a stored route (only a blocked *adjacent
  waypoint*, `MOVE-WAIT-008`, a narrower and different condition, does). This
  is a direct, on-point answer to a question the row says has none. The row's
  own `Revisit condition` -- "A claim reading the wall's effect on a queued
  path" -- is satisfied by `MOVE-REFRESH-012`'s own text.

**Checked and not broken** (adjacent claims exist but do not answer the
row's specific silence):

- **`DIV-940`** (`VirtualCaster` SAV reach) -- `UNIT-T9CTOR-110` and
  `UNIT-T9LIFE-111` (both promoted, `EXP-0354`) decode `VirtualCaster`'s
  in-memory construction and deletion lifecycle in detail, but
  `UNIT-T9LIFE-111` says outright its own "serializer and Position-rebind
  slot do not establish native activation or SAV reach" -- the row's specific
  question (does any `SpellEffect`/`PointEffect`/`AreaEffect`/`SpellTransport`/
  `Effect` field reference it) is still unanswered.
- **`DIV-052`** (stacked effect clamps) -- `MAGIC-SPRAY-134` (active) decodes
  a different clamp, Prismatic Spray's victim-count cap, not the combination
  of several simultaneous effects of one kind the row asks about.
- **`DIV-081`** (path picture frame selection / victim array) --
  `MAGIC-SPRAY-134` again resolves a related but distinct question (the
  victim-list count cap); it does not name the thirteen jump-table frame
  values or the weapon-borne release arm's record tag the row asks about.

**Not individually searched against the corpus in this pass** (read, not
each keyword-checked): `DIV-233`, `DIV-432`, `DIV-433`, `DIV-435`,
`DIV-1339`, `DIV-1316`, `DIV-487`, `DIV-485`, `DIV-190`, `DIV-156`, `DIV-159`,
`DIV-162`, `DIV-133`, `DIV-028`, `DIV-030`, `DIV-031`, `DIV-050`, `DIV-083`,
`DIV-084`, `DIV-086`, `DIV-087`, `DIV-089`, `DIV-092`, `DIV-117`, `DIV-118`,
`DIV-814`, `DIV-166` (already amended in-row at the 2026-08-19 pin bump, not a
fresh finding here), `DIV-175`, `DIV-178`, `DIV-179`, `DIV-212`, `DIV-222`,
`DIV-224`, `DIV-225`, `DIV-226`, `DIV-241`, `DIV-242`, `DIV-283`, `DIV-311`,
`DIV-426`, `DIV-427`, `DIV-848`, `DIV-871`, `DIV-1258`. The owner or a future
audit pass should run the same keyword-search method against these; none of
them is claimed closed or narrowed by this pass.

## (b) Rows cited by no `.go` file and no other document

**Population.** All 507 live rows.

**Method.** `grep -rohE 'DIV-[0-9]{3,4}'` over every `*.go` file gives 281
distinct cited ids; the same pattern over every other tracked `*.md` document
(ledger files themselves excluded) gives 666 distinct cited ids. A row's id in
neither set is cited nowhere outside the ledger itself -- not necessarily
wrong or stale, since a divergence need not be cited anywhere to be true, but
a citation is what would let a future reader find the code or doc the row is
actually about.

**Found: 73 orphan rows**, 62 `OPEN` and 11 `ACCEPTED`; by `Type`: 42
`UNKNOWN`, 15 `DEVIATION`, 16 `FIDELITY-DEBT`. A concentration is visible: 25
of the 73 are the town-interior ambience/idle-process family (`tavern
interior`, `shop interior`, `town ambience`, `school training` --
`DIV-843`..`DIV-876`), all `UNKNOWN`, all about private random sources,
lifecycle and degraded-art fallbacks for town-screen idle animation -- a
research area that was clearly swept in one pass (consecutive ids) and never
cross-referenced from code or another document afterward.

Full list (id | Type | Status | Subsystem):

```
DIV-803 | DEVIATION | OPEN | original SAV / current Player purses
DIV-787 | UNKNOWN | OPEN | saved Groups / supported dispatch and local chronology
DIV-788 | DEVIATION | OPEN | saved Groups / newly authored command state
DIV-790 | FIDELITY-DEBT | OPEN | saved Groups / bounded native movement and engagement
DIV-791 | DEVIATION | OPEN | saved Groups / native compatibility
DIV-576 | FIDELITY-DEBT | OPEN | AI / sustained explicit Retreat and fallback cadence
DIV-510 | DEVIATION | OPEN | cutscene audio, focus and cadence
DIV-486 | DEVIATION | ACCEPTED | town shop / ordinary staffs and Magic Beard are excluded from generate
DIV-451 | UNKNOWN | OPEN | inventory / reading a one-spell book
DIV-009 | DEVIATION | ACCEPTED | sim / property setter
DIV-012 | DEVIATION | ACCEPTED | dialogue / event00
DIV-040 | DEVIATION | ACCEPTED | magic / Teleport autocast
DIV-115 | DEVIATION | ACCEPTED | character generation / shared shell composition
DIV-830 | FIDELITY-DEBT | OPEN | book targeting / original per-cell actor-versus-ground predicate
DIV-583 | UNKNOWN | OPEN | inventory / scroll target and cancellation integration
DIV-584 | DEVIATION | ACCEPTED | town shop / consumable single versus double click
DIV-485 | UNKNOWN | OPEN | tavern / talk-only selected doll and inspection offset
DIV-010 | UNKNOWN | OPEN | dialogue / event text keys
DIV-023 | UNKNOWN | ACCEPTED | client / spell affordances
DIV-034 | UNKNOWN | OPEN | simulation / area paint for unshipped distributions
DIV-038 | UNKNOWN | ACCEPTED | magic / book-cast training source
DIV-114 | UNKNOWN | OPEN | tavern / alternate flat price mode
DIV-116 | UNKNOWN | OPEN | character generation / rejected Accept destination
DIV-118 | UNKNOWN | OPEN | shop / learned-spell Book toggle
DIV-826 | UNKNOWN | ACCEPTED | quick spells / campaign lifetime
DIV-828 | UNKNOWN | ACCEPTED | quick spells / installed and custom presentation
DIV-829 | UNKNOWN | OPEN | quick spells / accepted population and hired-class bounds
DIV-835 | UNKNOWN | OPEN | town exterior / presentation random generator
DIV-836 | UNKNOWN | OPEN | town exterior / square reentry defaults
DIV-837 | UNKNOWN | OPEN | town exterior / gate availability projection
DIV-843 | UNKNOWN | OPEN | tavern interior / private delay random generator
DIV-844 | UNKNOWN | OPEN | tavern interior / entry, reentry and cross-visit state
DIV-845 | UNKNOWN | OPEN | tavern interior / indirect interaction aliases
DIV-846 | UNKNOWN | OPEN | tavern interior / picture cleanup and destruction
DIV-847 | UNKNOWN | OPEN | tavern interior / retained sound and audible fidelity
DIV-848 | UNKNOWN | OPEN | tavern interior / missing art-family degradation
DIV-851 | UNKNOWN | OPEN | shop interior / private idle random source
DIV-852 | UNKNOWN | OPEN | shop interior / entry, focus, dialogue and reset lifecycle
DIV-854 | UNKNOWN | OPEN | shop interior / shared merchant index and reaction mapping
DIV-855 | UNKNOWN | OPEN | shop interior / immutable art cache and cleanup
DIV-856 | UNKNOWN | OPEN | shop interior / incomplete family degradation
DIV-859 | UNKNOWN | OPEN | town ambience / private bird random stream
DIV-860 | UNKNOWN | OPEN | town ambience / bird process latch, retained delay and re-entry
DIV-861 | UNKNOWN | OPEN | town ambience / bird and overlay origins and visibility
DIV-862 | UNKNOWN | OPEN | town ambience / optional art fallback
DIV-863 | UNKNOWN | OPEN | town ambience / bird and star one-shot capability
DIV-864 | UNKNOWN | OPEN | town ambience / statue terminal counter process lifetime
DIV-865 | UNKNOWN | OPEN | town ambience / stationary selector16 and lifecycle policy
DIV-866 | UNKNOWN | OPEN | town ambience / repeating crowd lifecycle
DIV-869 | UNKNOWN | OPEN | school training / private random source
DIV-870 | UNKNOWN | OPEN | school training / process-static projection
DIV-871 | UNKNOWN | OPEN | school training / incomplete-family degradation
DIV-874 | UNKNOWN | OPEN | school training / Rotate request and audible result
DIV-875 | UNKNOWN | OPEN | school training / immutable art ownership
DIV-876 | UNKNOWN | OPEN | school training / degraded column fallback
DIV-882 | FIDELITY-DEBT | OPEN | native city SAV / LastMission
DIV-883 | FIDELITY-DEBT | OPEN | native city SAV / MissionTime
DIV-1143 | FIDELITY-DEBT | OPEN | rendering / original client registration and visibility inputs
DIV-1183 | DEVIATION | OPEN | current SAV / superseded native movement bridge
DIV-1188 | DEVIATION | OPEN | current SAV / deterministic qualified key remint
DIV-1190 | FIDELITY-DEBT | OPEN | current SAV / actor action deadline
DIV-1191 | FIDELITY-DEBT | OPEN | current SAV / native cast action lifecycle
DIV-1199 | DEVIATION | OPEN | generated mission20 / fresh spatial residuals
DIV-1200 | DEVIATION | OPEN | generated mission20 / Players, Diaries and session defaults
DIV-1201 | FIDELITY-DEBT | OPEN | current physical SAV / target, phase and attribution
DIV-1268 | FIDELITY-DEBT | OPEN | score history / observations and old saves
DIV-1276 | FIDELITY-DEBT | OPEN | pre-create / sparkle decoration
DIV-1277 | DEVIATION | ACCEPTED | character generator / selected presets
DIV-1278 | FIDELITY-DEBT | OPEN | human voice / client class and sex
DIV-1289 | FIDELITY-DEBT | OPEN | Game Options / panel and transactions
DIV-1298 | FIDELITY-DEBT | OPEN | graphics options / population and defaults
DIV-1299 | FIDELITY-DEBT | OPEN | Smoothing / backpack boundary blend
DIV-1307 | FIDELITY-DEBT | OPEN | item inspection book titles
```

## (c) OPEN/ACCEPTED rows that look closable because the engine has changed

**Population.** All 507 live rows with `Status` `OPEN` or `ACCEPTED`: 437
`OPEN` and 70 `ACCEPTED`.

**Method.** A lexical pass over each such row's `Implemented behaviour`,
`Reason` and `Revisit condition` cells for language that names an
incompleteness directly (`not (yet) implement`, `TODO`, `unimplemented`,
`not yet`, `still missing`, `not currently`, `not supported`) -- the
vocabulary a row itself would use if the gap it names might already be
closed. This is a narrow, precision-favouring pattern: it is expected to
under-count (a row can be stale without using any of these words) and was
chosen over a broader pattern specifically so every hit is worth a real
check, not a guess.

**Found: 14 candidates.** `DIV-1248`, `DIV-746`, `DIV-666`, `DIV-237`,
`DIV-255`, `DIV-1259`, `DIV-1260`, `DIV-1272`, `DIV-033`, `DIV-168`,
`DIV-944`, `DIV-969`, `DIV-1359`, `DIV-1360`.

**Checked against current source, 3 of 14:**

- **`DIV-1259`** (cutscene / Smacker audio codec coverage) -- row states
  `pkg/video/smacker` refuses a Bink-compressed audio track. Current source
  (`pkg/video/smacker/audio.go:12-27`, `container.go:34,198`) still defines
  `errBinkAudio` and still marks `compress == 2` "Bink (unsupported, matches
  upstream)". Row is accurate; not closable.
- **`DIV-1260`** (cutscene / Smacker frame-interval sanity bound) -- row
  states `pkg/video/smacker`'s `Decoder.Interval()` carries no sanity bound
  the retired `pkg/video/native.go` used to have. Current source
  (`decoder.go:88`) still defines `Interval()` with no such bound. Row is
  accurate; not closable.
- **`DIV-1272`** (Prismatic Spray / secondary selection) -- row states the
  original's group-sight population and score/turn ranking for secondary
  targets are not implemented; the current build still uses "the native
  hostile neighbourhood and entity order" instead. A grep of
  `pkg/sim/spelldelivery.go` and the other files referencing `Prismatic`
  found no group-sight-population or score/rank logic matching the row's own
  description of what ROM1 does. Row still reads as accurate at this grep
  depth; not confirmed closable, though this check did not read every
  Prismatic-Spray-referencing file's full body.

**Not checked, 11 of 14:** `DIV-1248`, `DIV-746`, `DIV-666`, `DIV-237`,
`DIV-255`, `DIV-033`, `DIV-168`, `DIV-944`, `DIV-969`, `DIV-1359`, `DIV-1360`.
`DIV-1359` and `DIV-1360` are worth the owner's own attention first among
these: both are large, recent, `DEVIATION`-typed rows (retyped from `OWNER-DIRECTION` by this split) explicitly built
on story1202's own corrected mechanism cell and cross-reference `DIV-1361`,
so an engine change closing or narrowing either is plausible and this audit
did not verify it either way.

## Summary

| Part | Population | Checked | Found |
|---|---|---|---|
| (a) uncited-claim silence breaks | 201 UNKNOWN rows, 51 silence-assertion rows | 11 of 51 | 1 confirmed break (`DIV-035`), 3 adjacent-not-broken |
| (b) cited by neither `.go` nor another doc | 507 live rows | all 507 (set operation, exact) | 73 orphan rows |
| (c) OPEN/ACCEPTED rows possibly closable | 507 live rows, lexical filter | 3 of 14 candidates | 0 confirmed closable |

No row's `Status` or `Type` changed as a result of this audit. `DIV-035` is
the one finding with a concrete, citable claim id ready for the owner or a
future story to act on.

## Silence rows re-searched at k128

Part (a) searched 11 of 51 silence-asserting rows and found one break
(`DIV-035`). This section re-derives the population at the k128 pin and
searches every row.

**Population.** 171 rows have `Status` OPEN and an ROM1 cell that contains
"research is silent" or "no claim", "no promoted claim", "no published claim"
or "no active claim" (case-insensitive), read from `docs/divergences/*.md` at
the pre-edit commit: 102 `UNKNOWN`, 40 `DEVIATION`, 28 `FIDELITY-DEBT`, and one
row (`DIV-1795`) whose Type cell does not parse under that split. The phrase
set is wider than part (a)'s 51 rows, which were `UNKNOWN` rows with a
fixed "No claim states/establishes/identifies/addresses/names" opening; it
adds rows that cite claims and then state a narrow absence.

**Instruments.**

1. A ranked search over the 2604 non-retracted claim cards in
   `knowledge/claims/*.md` (`retracted.md` and `registry.md` excluded). The
   query per row is the content words and backticked identifiers of the
   sentence holding the silence statement and the sentence after it, the
   Subsystem cell and the Revisit cell. Score is the sum of inverse document
   frequency over shared terms present in under 4 percent of cards, divided by
   a card-length factor. Claims the row already cites are excluded. The five
   top-scoring headlines were read for each row, and the full card for every
   one whose headline could answer the row's question.
2. The headlines of all 136 active claims whose evidence cell names an
   experiment from 0424 to 0445 (the claims newest to the rows), read against
   every row.

The ranked search is a bounded instrument. A row marked "still silent" means
none of the headlines read and none of the full cards read answers or narrows
its question; it does not exclude a claim that neither instrument ranked.

**Outcomes.** Answered 9 rows, narrowed 11, still silent 151.

- Answered, engine behaviour differs from the claim, row kept OPEN with the
  mismatch stated in its Reason cell: `DIV-035` (`MOVE-REFRESH-012`),
  `DIV-1457` (`VIDEO-067` to `VIDEO-070`, `ANIM-119`, `ANIM-120`), `DIV-1725`
  (`MAGIC-239`, `MAGIC-240`), `DIV-1726` (`MAGIC-237`), `DIV-1727` (`AI-376`),
  `DIV-1728` (`MAGIC-238`), `DIV-1729` (`AI-377`).
- Answered, engine behaviour matches the claim, row kept OPEN and marked
  closable in its Revisit cell: `DIV-1491` (`MAGIC-240`), `DIV-1735`
  (`MOVE-087`). Moving them to `DIVERGENCES-CLOSED.md` is a seat decision.
- Narrowed: `DIV-052`, `DIV-088`, `DIV-131`, `DIV-178`, `DIV-311`,
  `DIV-1387`, `DIV-1409`, `DIV-1516`, `DIV-1545`, `DIV-1651`, `DIV-1741`. The
  Revisit cell of each states what remains. `DIV-178` and `DIV-311` also carry
  a mismatch between the claim and the Implemented cell.

No row's `Type` changed, so no row moved between headings. No engine code
changed.

**Search record.** One line per row: the area file, the five headlines read
from the ranked search (claims already cited by the row excluded), the
outcome, and the claim that answers or narrows it. For a row still silent
after a full-card read, the claim read and why it does not answer follows the
dash.

| Row | Area file | Ranked headlines read | Outcome | Claim |
|---|---|---|---|---|
| DIV-1654 | audio | ANIM-PHASE-003, ANIM-SND-022, ANIM-118, TOWN-413, TRIG-MSGLIMIT-050 | still silent | -; ANIM-118 gives the swing hook class, not the cast hook |
| DIV-506 | audio | TOWN-382, SAV-908, VIDEO-MUSIC-012, SHOP-RNG-008, ANIM-098 | still silent | - |
| DIV-1497 | campaign | HERO-DYINGTICK-145, SAV-1094, SAV-1092, SAV-1086, REG-SCN-059 | still silent | - |
| DIV-432 | campaign | TRIG-OFFMAP-041, AI-CURSOR-208, TRIG-CHECK-053, AI-ORDER-294, AI-ROUTE-045 | still silent | - |
| DIV-311 | campaign | TOWN-373, TOWN-469, DIALOGUE-057, TOWN-345, AI-CURSOR-236 | narrowed | TOWN-373 |
| DIV-1495 | character-generation | VIDEO-SFX-059, VIDEO-OPTIONS-057, TEXT-073, TEXT-CHARGEN-029, DLG-KEYS-040 | still silent | - |
| DIV-190 | character-generation | DLG-DRESS-024, AI-CENSUS-047, TOWN-265, HERO-FIGURE-061, TEXT-091 | still silent | - |
| DIV-156 | character-generation | VIDEO-SFX-059, VIDEO-OPTIONS-057, DLG-KEYS-040, VIDEO-SFX-058, TOWN-283 | still silent | - |
| DIV-1389 | character-generation | HERO-CHARGEN-083, SAV-649, TOWN-373, TRIG-BIND-010, MOVE-STEP-010 | still silent | - |
| DIV-1409 | character-generation | TEXT-CHARGEN-029, TEXT-UI-038, VIDEO-SFX-059, VIDEO-OPTIONS-057, ANIM-WALLFIREFRAME-033 | narrowed | TEXT-082 |
| DIV-1496 | character-generation | ANIM-AMBIENT-016, TOWN-182, R2-SESSION-002, PARTY-GROUP-009, TEXT-075 | still silent | - |
| DIV-1507 | character-generation | VIDEO-OPTIONS-057, ITEM-USE-112, DLG-KEYS-040, SESS-INPUT-037, VIDEO-SFX-059 | still silent | - |
| DIV-1508 | character-generation | VIDEO-OPTIONS-057, TEXT-075, TOWN-331, DLG-MSGNUM-025, SHOP-MISSION-018 | still silent | - |
| DIV-247 | client-cursor-and-selection | AI-MINIMAP-062, SPR16A-CURSOR-067, AI-CURSOR-196, AI-CURSOR-207, AI-CURSOR-208 | still silent | - |
| DIV-335 | client-cursor-and-selection | AI-CLICK-050, DIALOGUE-044, AI-PANEL-123, SESS-INPUT-037, TEXT-068 | still silent | - |
| DIV-1534 | client-cursor-and-selection | SESS-COMPOSE-058, SPR16A-CURSOR-061, VIDEO-SFX-059, MISSION-MSGLINE-056, TEXT-077 | still silent | - |
| DIV-304 | client-documents-panel | MISSION-VICTORY-031, MENU-053, TOWN-254, TOWN-207, SAV-1101 | still silent | - |
| DIV-303 | client-documents-panel | TEXT-077, TEXT-079, TEXT-076, MISSION-MSGLINE-056, SPR16A-FONT-015 | still silent | - |
| DIV-1387 | client-documents-panel | DLG-LINE-038, TEXT-088, SAV-1127, DIALOGUE-062, TEXT-SAVELABEL-061 | narrowed | DLG-LINE-038 |
| DIV-233 | client-input-and-keys | TOWN-265, MISSION-VIP-018, SPR256-DOLL-045, SAV-TERRKEY-056, TERR-EDGE-024 | still silent | - |
| DIV-253 | client-mission-screen | AI-CURSOR-052, AI-CURSOR-242, AI-CURSOR-207, AI-CURSOR-204, TERR-TILE-044 | still silent | - |
| DIV-316 | client-mission-screen | TOWN-093, TOWN-353, AI-CURSOR-220, TOWN-345, TOWN-281 | still silent | - |
| DIV-200 | client-mission-screen | MENU-052, DLG-PANEL-035, TOWN-222, TOWN-260, TOWN-282 | still silent | - |
| DIV-1455 | client-mission-screen | AI-CURSOR-220, SESS-COMPOSE-058, TOWN-091, AI-CURSOR-218, MAGIC-ACTOR-066 | still silent | - |
| DIV-1500 | client-mission-screen | MENU-GEOM-006, MENU-ASSET-001, TERR-LIGHT-123, TEXT-HOVERPAINT-053, AI-INPUT-127 | still silent | - |
| DIV-1795 | client-mission-screen | DIALOGUE-070, TEXT-079, DIALOGUE-057, TEXT-HOVERPAINT-053, DIALOGUE-062 | still silent | - |
| DIV-1761 | client-mission-screen | TEXT-STRTAB-023, REG-KEY-044, TEXT-091, SPR256-TRLR-021, TEXT-092 | still silent | - |
| DIV-178 | client-presentation-other | TOWN-480, SHOP-PICKER-043, TOWN-282, TOWN-122, TOWN-428 | narrowed | TOWN-467 |
| DIV-212 | client-presentation-other | SHOP-FIGURE-041, INV-SCOPE-005, ALM-SEC-002, SHOP-CLS-001, SHOP-SCREEN-030 | still silent | - |
| DIV-1459 | client-presentation-other | ITEM-CASTSTATE-056, TOWN-467, SAV-1127, AI-377, TOWN-124 | still silent | - |
| DIV-1460 | client-presentation-other | TERR-SPR-041, TERR-LIGHT-113, TERR-SPR-047, TERR-SPR-038, HERO-BARE-037 | still silent | - |
| DIV-1481 | client-presentation-other | HERO-DKIDX-162, PAL-BAND-016, ANIM-105, REG-UNITS-050, SAV-1129 | still silent | - |
| DIV-166 | client-room-presentation | TOWN-333, TOWN-312, TOWN-313, TOWN-258, TOWN-214 | still silent | - |
| DIV-168 | client-room-presentation | TERR-GEOM-036, SHOP-GEN-005, TOWN-252, TOWN-333, TOWN-096 | still silent | - |
| DIV-169 | client-room-presentation | SPR16A-FONT-018, TEXT-078, TEXT-085, TEXT-066, SPR16A-FONT-020 | still silent | - |
| DIV-175 | client-room-presentation | SHOP-LIMIT-049, TOWN-095, TOWN-087, TERR-SPR-144, TOWN-281 | still silent | - |
| DIV-177 | client-room-presentation | TOWN-333, TOWN-467, MAGIC-AUTOCAST-020, DLG-PANEL-035, TERR-SEM-004 | still silent | - |
| DIV-179 | client-room-presentation | SPR16A-FONT-018, TEXT-078, TEXT-066, SPR16A-FONT-020, SPR16A-FONT-015 | still silent | - |
| DIV-183 | client-room-presentation | TOWN-282, TOWN-245, TOWN-312, TOWN-260, TOWN-214 | still silent | - |
| DIV-1519 | combat-and-ai-attack-and-hold-orders | AI-STRIKE-055, AI-353, AI-352, AI-SCRIPTATTACK-120, AI-FOLLOWDEATH-119 | still silent | -; AI-352, AI-353, AI-370, AI-374 do not read state 3 once ord+0x0c is gone |
| DIV-1523 | combat-and-ai-attack-and-hold-orders | AI-GUARD-012, SESS-VIEW-030, TERR-SPR-047, R2-ASSET-029, AI-PATROL-018 | still silent | - |
| DIV-1517 | combat-and-ai-attack-and-hold-orders | AI-GUARD-021, ANIM-AMBIENT-016, AI-FACE-066, AI-SWARM-022, AI-353 | still silent | - |
| DIV-1543 | combat-and-ai-attack-and-hold-orders | MENU-055, AI-QUICKSAVE-281, ANIM-STATE-002, AI-364, MAGIC-AIBIT-082 | still silent | - |
| DIV-2018 | combat-and-ai-attack-and-hold-orders | AI-352, HERO-CADENCE-114, AI-353, AI-REACH-072, AI-365 | still silent | -; AI-374 already cited; the executor rows stay unread |
| DIV-1544 | combat-and-ai-attack-and-hold-orders | AI-357, SAV-GRPIDENT-562, SAV-GRPSAVENEXT-572, MAGIC-SPRAY-136, MAGIC-SPRAY-134 | still silent | -; AI-357, MAGIC-240 do not state the state at cast end |
| DIV-1651 | combat-and-ai | AI-340, AI-329, SAV-1132, AI-327, AI-STANCE-098 | narrowed | MOVE-087 |
| DIV-1256 | cutscene | VIDEO-036, VIDEO-SFX-021, TOWN-413, VIDEO-035, VIDEO-046 | still silent | - |
| DIV-1259 | cutscene | VIDEO-036, VIDEO-SFX-021, TOWN-434, R2-ENGINE-001, VIDEO-034 | still silent | - |
| DIV-1260 | cutscene | VIDEO-036, HERO-CADENCE-023, TERR-EDGE-026, VIDEO-032, VIDEO-SFX-021 | still silent | - |
| DIV-1261 | cutscene | VIDEO-036, VIDEO-SFX-021, R2-ENGINE-001, TOWN-493, VIDEO-034 | still silent | - |
| DIV-1258 | cutscene | VIDEO-036, VIDEO-SFX-021, VIDEO-049, VIDEO-033, SAV-WRITER-284 | still silent | - |
| DIV-1485 | dialogue | DIALOGUE-057, DIALOGUE-050, AI-LOS-089, DLG-PANEL-035, MAGIC-AUTOCAST-020 | still silent | - |
| DIV-1521 | dialogue | DLG-SPEAKER-041, TAVERN-023, TAVERN-BUTTON-020, AI-LOS-089, DIALOGUE-050 | still silent | - |
| DIV-1522 | dialogue | DLG-SPEAKER-041, DIALOGUE-051, DIALOGUE-050, TOWN-484, TOWN-475 | still silent | - |
| DIV-1547 | dialogue | DLG-SYNTH-042, DIALOGUE-057, DLG-DRESS-024, DIALOGUE-054, PAL-FIGURE-014 | still silent | - |
| DIV-1736 | healing | AI-ORDER-010, AI-REACH-072, SAV-1129, MAGIC-AUTOCAST-020, HERO-FINISH-066 | still silent | - |
| DIV-1757 | inventory | ITEM-PRICETAG-144, TEXT-UI-047, REG-VAL-029, TEXT-HOVERCHAR-050, AI-CURSOR-189 | still silent | -; ITEM-152, ITEM-154, TEXT-UI-038 give no range caption |
| DIV-1749 | inventory | AI-335, ANIM-107, MOVE-073, AI-CURSOR-126, SESS-INPUT-037 | still silent | - |
| DIV-1541 | inventory | PAL-BAND-016, SPR256-DOLL-045, SPR16A-PROJ-026, TOWN-490, HERO-FIGURE-144 | still silent | - |
| DIV-451 | inventory | AI-QUICKSAVE-281, MAGIC-BOOK-002, TOWN-246, MAGIC-MIND-010, ALM-EFFREC-071 | still silent | - |
| DIV-111 | inventory | ITEM-SUIT-035, AI-SPRAY-267, ITEM-PICT-052, AI-RETREAT-275, ITEM-AUTHDROP-087 | still silent | - |
| DIV-112 | inventory | SAV-TOKENPTR-075, TOWN-483, R2-ENGINE-003, HERO-DEFEAT-136, AI-329 | still silent | - |
| DIV-433 | inventory | MERC-PRICE-004, MERC-DEATH-006, MERC-CMD-007, SAV-1086, SAV-608 | still silent | -; ITEM-WEAR-057 is the wear rule, not window eligibility |
| DIV-084 | inventory | AI-CURSOR-176, AI-PANEL-123, TOWN-479, TEXT-HOVERROOM-051, TOWN-359 | still silent | - |
| DIV-086 | inventory | TEXT-HOVERROOM-051, SESS-INPUT-037, TOWN-353, TOWN-261, TEXT-UI-047 | still silent | - |
| DIV-088 | inventory | AI-PANEL-123, AI-CURSOR-126, SESS-INPUT-037, VIDEO-070, AI-RETREAT-270 | narrowed | SHOP-096 |
| DIV-057 | magic-spell-geometry | MAGIC-REACH-180, TEXT-FIT2-013, MAGIC-ITEMTRAIN-116, ITEM-CASTSTATE-056, TEXT-CHARGEN-028 | still silent | - |
| DIV-081 | magic-spell-geometry | MAGIC-AUTOCAST-020, MAGIC-ITEM-007, ITEM-CASTSTATE-056, MAGIC-ITEMTRAIN-116, HERO-ITEMSKILL-096 | still silent | - |
| DIV-083 | magic-spell-geometry | MAGIC-REACH-179, MAGIC-REACH-180, AI-COST-071, MAGIC-239, REG-UNITS-050 | still silent | - |
| DIV-1687 | magic-spellbooks-and-quick-spells | FAME-BOUNDARY-016, TEXT-LANG-002, MISSION-CURE-026, AI-SPELLITEM-290, PARTY-M20-032 | still silent | - |
| DIV-1725 | magic-spellbooks-and-quick-spells | SAV-BOUNDARY-367, AI-377, ITEM-CMD-007, MAGIC-ITEMTRAIN-116, SAV-931 | answered | MAGIC-239, MAGIC-240 |
| DIV-1726 | magic-spellbooks-and-quick-spells | MAGIC-237, MAGIC-CEIL-013, SAV-BOUNDARY-367, MAGIC-SPELL-001, SAV-ORIGLOAD-332 | answered | MAGIC-237 |
| DIV-1727 | magic-spellbooks-and-quick-spells | AI-376, AI-377, SAV-1128, MAGIC-CADENCE-127, AI-352 | answered | AI-376 |
| DIV-1728 | magic-spellbooks-and-quick-spells | SAV-BOUNDARY-367, SAV-ORIGLOAD-332, MAGIC-238, AI-FACE-067, ANIM-PROJ-025 | answered | MAGIC-238 |
| DIV-103 | magic | MAGIC-237, AI-QUICKSAVE-281, MAGIC-ITEMTRAIN-116, MAGIC-CLOUDEND-163, MAGIC-SING-019 | still silent | - |
| DIV-1311 | magic | SHOP-CONSUME-073, ITEM-CMD-007, MAGIC-AUTOCAST-020, SAV-HUMFIRST-465, HERO-BARE-037 | still silent | - |
| DIV-1524 | magic | ITEM-OWNED-028, AI-LOS-087, TOWN-495, AI-REACH-072, AI-353 | still silent | - |
| DIV-1655 | magic | MAGIC-ITEMTRAIN-116, AI-SWARM-022, FAME-STRING-010, SAV-HUMFIRST-465, ITEM-PICK-016 | still silent | - |
| DIV-1656 | magic | MAGIC-ITEMTRAIN-116, FAME-STRING-010, SAV-REGENFAULT-530, MAGIC-AUTOCAST-020, HERO-CADENCE-112 | still silent | - |
| DIV-487 | mission-screen | TOWN-010, MERC-DEATH-006, MERC-CMD-007, SAV-1086, SAV-608 | still silent | -; ITEM-WEAR-057 is the wear rule, not window eligibility |
| DIV-1904 | mods | UNIT-PICT-038, ALM-CLS-038, SAV-HUMNEW-505, MERC-PRICE-004, HERO-JOIN-121 | still silent | - |
| DIV-1913 | mods | HERO-JOIN-121, HERO-JOIN-120, SAV-891, MOVE-088, AI-CURSOR-190 | still silent | -; HERO-JOIN-120 states producers, not a save between arrival and grant |
| DIV-1319 | persistence-city-sav-and-imports | SAV-WORLDFRONT-432, SAV-HUMRUNTIME-476, SAV-598, DLG-ZEROARM-029, SAV-TOKENPTR-075 | still silent | - |
| DIV-1320 | persistence-city-sav-and-imports | SAV-1037, SAV-1059, VIDEO-033, SAV-649, VIDEO-SFX-053 | still silent | - |
| DIV-1321 | persistence-city-sav-and-imports | UNIT-CTOR-004, SAV-HUMRUNTIME-476, ITEM-EFFKEY-147, HERO-MOD-016, HERO-EFFECT-019 | still silent | - |
| DIV-1491 | persistence-current-sav-actions | AI-350, AI-357, AI-376, AI-356, AI-364 | answered | MAGIC-240 |
| DIV-1501 | persistence-current-sav-actions | HERO-DEFEAT-136, TOWN-445, SAV-972, VIDEO-MUSIC-064, ANIM-098 | still silent | - |
| DIV-1741 | persistence-current-sav-actions | SAV-1128, HERO-DWELL-065, VIDEO-SFX-014, SAV-ORIGLOAD-332, SAV-1127 | narrowed | HERO-FINISH-066, AI-374 |
| DIV-1679 | persistence-current-sav-player-save | TEXT-TILDE2-017, MAGIC-216, MAGIC-MAPLAYER-040, MAGIC-197, SAV-POSTLOAD-223 | still silent | - |
| DIV-1680 | persistence-current-sav-player-save | ANIM-098, SAV-SAVELABEL-1017, TEXT-TILDE-009, TEXT-075, SAV-PTRMAP-035 | still silent | - |
| DIV-1681 | persistence-current-sav-player-save | REG-VAL-026, ALM-LIM-067, TERR-SHDW-131, SAV-WRITERAUDIT-380, TERR-VER-005 | still silent | - |
| DIV-1682 | persistence-current-sav-player-save | SAV-1093, MAGIC-CLOUDEND-163, MAGIC-092, TERR-PASS-073, SAV-WRITERAUDIT-380 | still silent | - |
| DIV-1339 | persistence-current-sav-player-save | TEXT-COLL-025, TEXT-ALIAS-011, TEXT-FIT2-013, AI-FOLLOWAUTH-117, TEXT-IN-005 | still silent | - |
| DIV-1359 | persistence-current-sav-player-save | REG-SCN-067, SAV-628, SAV-CAMPAIGN-083, SAV-CAMPAIGN-076, SAV-926 | still silent | -; SAV-926 states command 37 stock construction, not the tavern-open fault |
| DIV-1414 | persistence-current-sav | SAV-893, SAV-PLAYERPOP-831, REG-KIND-033, HERO-DYINGTICK-145, SAV-615 | still silent | - |
| DIV-1433 | persistence-current-sav | VIDEO-067, SPR16A-CURSOR-046, VIDEO-MUSIC-066, VIDEO-070, AI-CMD-054 | still silent | - |
| DIV-1436 | persistence-current-sav | TERR-STRUCT-069, MOVE-086, TERR-CELLREC-146, SAV-SACKENTRY-590, TERR-STRUCT-076 | still silent | -; MOVE-086 states the restore side, not the cost byte written without planes |
| DIV-1729 | persistence-entities | UNIT-SPELL-007, AI-377, REG-MAT-042, MOVE-GROUP-037, VIDEO-SFX-014 | answered | AI-377 |
| DIV-886 | persistence-native-city-sav | MERC-SHELF-002, SAV-609, REG-SCN-067, SPR256-RLE-019, SAV-926 | still silent | - |
| DIV-891 | persistence-native-city-sav | ITEM-DOC-069, SAV-WRITERAUDIT-380, SAV-ORIGMISSION-400, TOWN-372, SAV-1028 | still silent | - |
| DIV-902 | persistence-native-city-sav | SAV-615, SAV-626, SAV-609, SAV-608, SAV-614 | still silent | - |
| DIV-905 | persistence-native-city-sav | SAV-614, FAME-STRING-010, SAV-1101, SHOP-TRAY-025, SHOP-EFFPRICE-066 | still silent | - |
| DIV-907 | persistence-native-city-sav | SAV-629, SAV-602, SAV-614, MERC-CMD-007, SAV-1086 | still silent | - |
| DIV-1411 | persistence-native-city-sav | SAV-1140, SAV-1139, SAV-1094, DLG-ZEROARM-029, SHOP-TRAY-025 | still silent | -; SAV-1135..1137 state the siege Unit spawn, not the group |
| DIV-1412 | persistence-native-city-sav | ALM-GRID-032, SHOP-GEN-005, SPR16A-TXT-023, ANIM-103, SHOP-NPC-012 | still silent | - |
| DIV-1683 | persistence-native-city-sav | SAV-CAMPAIGN-084, SAV-CAMPAIGN-085, AI-THREAT-044, SAV-936, DLG-ZEROARM-029 | still silent | - |
| DIV-1684 | persistence-native-city-sav | SAV-1134, SAV-1139, SAV-616, TAVERN-BUTTON-020, SAV-1135 | still silent | -; SAV-1135..1137 state the siege Unit spawn, not the group |
| DIV-1685 | persistence-native-city-sav | MERC-DEATH-006, SHOP-TRAY-025, DLG-ZEROARM-029, SAV-775, SAV-CROSSNEXT-585 | still silent | - |
| DIV-1691 | persistence-native-city-sav | SHOP-CONSUME-074, UNIT-STRUCTUSE-091, SHOP-GEN-005, ITEM-USE-113, MAGIC-ATTACH-016 | still silent | - |
| DIV-940 | persistence-original-saves-corpus | SAV-TAGSCAN-158, SAV-PRODDIRECT-207, SAV-EFFECTGRAPH-366, SAV-TOKENLOAD-093, SAV-1037 | still silent | -; MAGIC-235 states a temporary caster, not SAV reach |
| DIV-954 | persistence-original-saves-corpus | MOVE-DOM-026, MAGIC-WALLBLOCK-045, SAV-SACKCALLER-593, SAV-791, MOVE-088 | still silent | - |
| DIV-1815 | persistence-original-saves-format | SAV-DEATH-051, MOVE-TICK-015, SAV-TOKENLOAD-096, ANIM-DEATH-007, VIDEO-MUSIC-012 | still silent | -; SAV-DEADLOAD-126..129 state stored scalars and the terminal class, not the writer |
| DIV-1329 | persistence-original-saves-format | TOWN-417, RES-MAGIC-001, TEXT-UI-034, TERR-ANIM-006, TERR-SHDW-132 | still silent | - |
| DIV-1419 | persistence-original-saves-format | SAV-1091, ITEM-STARCOMP-100, ITEM-CODE-029, SHOP-ROUND-017, ITEM-ARMFILL-032 | still silent | - |
| DIV-096 | persistence | AI-QUICKSAVE-281, ANIM-071, TOWN-124, AI-RETREAT-271, AI-364 | still silent | - |
| DIV-100 | persistence | AI-SPRAY-267, SAV-SUFF-303, TOWN-124, SAV-PLAYERPOP-831, TOWN-407 | still silent | - |
| DIV-987 | persistence | R2-ASSET-022, R2-ASSET-024, MENU-051, AI-STRIKE-055, MISSION-M30-024 | still silent | - |
| DIV-1385 | rendering | TEXT-077, TEXT-078, TEXT-079, TEXT-065, TEXT-071 | still silent | - |
| DIV-1386 | rendering | TOWN-091, TERR-SPR-067, AI-CURSOR-236, SAV-1135, ALM-TERR-015 | still silent | - |
| DIV-1553 | shop | TOWN-479, ITEM-STARSURF-097, AI-CURSOR-176, SHOP-099, SAV-CITYSALE-513 | still silent | -; SHOP-096..SHOP-101 already cited in the row |
| DIV-1550 | shop | TEXT-078, DLG-BUTTON-039, TEXT-079, TOWN-392, SHOP-SCREEN-037 | still silent | - |
| DIV-133 | shop | ALM-SACK-066, AI-CENSUS-047, DLG-LINE-038, TOWN-258, TEXT-090 | still silent | - |
| DIV-087 | shop | HERO-SKILLBUY-076, AI-CMD-032, TEXT-HOVERROOM-051, ITEM-WEAR-057, SHOP-TRAY-026 | still silent | - |
| DIV-089 | shop | TEXT-HOVERROOM-051, AI-CURSOR-126, SAV-SUFF-302, SAV-ORIGLOAD-332, ITEM-STARSURF-097 | still silent | - |
| DIV-092 | shop | R2-ASSET-027, ITEM-PRICETAG-144, SHOP-100, R2-ASSET-029, ANIM-101 | still silent | - |
| DIV-118 | shop | MAGIC-BOOK-002, MAGIC-MIND-010, AI-QUICKOWNER-280, AI-QUICKINVOKE-279, HERO-MP-006 | still silent | - |
| DIV-319 | shop | SHOP-GEN-005, SHOP-OBJ-002, SHOP-SELL-010, TERR-TILE-044, ITEM-CMD-007 | still silent | - |
| DIV-1405 | shop | SHOP-PRICE-011, ITEM-DMGFACT-020, SHOP-MONEY-048, SHOP-TRAY-026, ITEM-STARCOMP-100 | still silent | - |
| DIV-1463 | shop | SHOP-TRAY-025, ITEM-PRICETAG-144, SHOP-099, SAV-CITYMOVE-512, SESS-VIEW-031 | still silent | - |
| DIV-1549 | shop | TEXT-078, SHOP-SCREEN-037, TRIG-DIST-014, TEXT-TILDE2-017, DLG-LINE-038 | still silent | - |
| DIV-028 | simulation-cast-and-pursuit | AI-357, SAV-1092, MAGIC-AUTOCAST-020, AI-350, AI-RETREAT-270 | still silent | -; AI-357, AI-350 do not state a new order arm over action 0xd |
| DIV-029 | simulation-cast-and-pursuit | MAGIC-ACTGATE-079, UNIT-GATE-014, AI-GUARD-007, AI-TICK-008, MISSION-VIP-018 | still silent | - |
| DIV-030 | simulation-cast-and-pursuit | UNIT-M10CAST-056, MAGIC-CADENCE-126, HERO-CADENCE-023, SAV-HUMNEWSAVE-507, AI-356 | still silent | - |
| DIV-031 | simulation-cast-and-pursuit | SAV-CITYDERIVE-515, UNIT-CTOR-004, SHOP-LIMIT-049, AI-377, AI-RETREAT-270 | still silent | - |
| DIV-050 | simulation-cast-and-pursuit | HERO-CROSSHOLD-146, SAV-615, ANIM-107, MAGIC-219, HERO-DYINGTICK-145 | still silent | - |
| DIV-241 | simulation-cast-and-pursuit | SAV-CHILDBOUNDARY-496, SAV-INITGUARD-488, PARTY-GROUP-009, SAV-ID-015, AI-362 | still silent | - |
| DIV-242 | simulation-cast-and-pursuit | SAV-PACK-007, TERR-SPR-047, TRIG-CAST-033, SAV-FRAME-021, DLG-NPCTAG-019 | still silent | - |
| DIV-1563 | simulation | AI-352, SAV-1132, PAL-PROJ-011, TOWN-373, AI-376 | still silent | - |
| DIV-1577 | simulation | AI-357, AI-355, VIDEO-MUSIC-066, SPR16A-CURSOR-046, SPR16A-CURSOR-067 | still silent | - |
| DIV-035 | simulation | ANIM-071, MAGIC-CASTCLOCK-171, FAME-021, MAGIC-CAST-003, SHOP-ANIMATION-081 | answered | MOVE-REFRESH-012 |
| DIV-052 | simulation | SHOP-103, MAGIC-REACH-178, AI-CURSOR-208, MISSION-VIP-018, SAV-WRITER-284 | narrowed | ITEM-EFFSYM-148 |
| DIV-1735 | simulation | MAGIC-REACH-180, AI-361, TEXT-078, TERR-DRAWSTAMP-170, SHOP-MERCHANT-046 | answered | MOVE-087 |
| DIV-1529 | tavern | TOWN-483, AI-337, TAVERN-023, TAVERN-CLICK-019, TOWN-468 | still silent | - |
| DIV-485 | tavern | ITEM-DOC-069, PAL-FIGURE-014, TEXT-FONT2-016, SAV-791, TRIG-DIST-014 | still silent | - |
| DIV-117 | tavern | SAV-PROJLOAD-429, ITEM-USE-113, TOWN-012, TOWN-010, MERC-TYPE-001 | still silent | - |
| DIV-426 | tavern | TOWN-122, TEXT-HOVERROOM-051, AI-CURSOR-176, AI-CURSOR-052, AI-KEYMOD-059 | still silent | - |
| DIV-427 | tavern | SAV-1107, MERC-PRICE-004, ALM-GRID-014, TOWN-012, AI-327 | still silent | - |
| DIV-1530 | tavern | TAVERN-CLICK-019, MOVE-ALT-020, REG-VAL-026, TOWN-483, SAV-DEATH-051 | still silent | - |
| DIV-1763 | tavern | SAV-SAVEDIR-334, SAV-MAP-005, SPR256-CORPUS-006, AI-CURSOR-194, VIDEO-050 | still silent | - |
| DIV-159 | town | TOWN-261, TOWN-280, TOWN-165, TOWN-183, MISSION-LATCH-014 | still silent | - |
| DIV-1404 | town | MISSION-MSGLINE-056, TOWN-254, DLG-BUTTON-039, DIALOGUE-065, ITEM-154 | still silent | - |
| DIV-1525 | town | MOVE-ALT-021, SAV-629, MOVE-ALT-018, MERC-POOL-011, SAV-928 | still silent | - |
| DIV-1527 | town | VIDEO-SFX-059, DLG-KEYS-040, TOWN-480, VIDEO-OPTIONS-057, DIALOGUE-044 | still silent | - |
| DIV-1647 | town | TOWN-344, VIDEO-050, TOWN-481, SHOP-LIFE-014, SHOP-GEN-005 | still silent | - |
| DIV-1649 | town | REG-UNITS-049, ANIM-ARM-018, TAVERN-BUTTON-020, ANIM-REGISTER-083, TERR-FOG-081 | still silent | - |
| DIV-1650 | town | ANIM-ARM-018, UNIT-DIFF-002, MISSION-VICTORY-033, SHOP-BUY-009, ANIM-REGISTER-083 | still silent | -; ANIM-REGISTER-083 maps registration selectors, not the tavern slot files |
| DIV-435 | ui-and-settings | UNIT-FIGURE-032, SAV-1104, DLG-LANG-010, TOWN-469, MERC-PRICE-004 | still silent | - |
| DIV-1504 | ui-and-settings | AI-CURSOR-174, TEXT-LANG-002, MISSION-VICTORY-029, AI-357, MOVE-EVENT-061 | still silent | - |
| DIV-162 | ui-and-settings | DLG-RECT-037, TEXT-UI-047, DLG-INNVOICE-030, TOWN-185, TOWN-356 | still silent | - |
| DIV-222 | ui-and-settings | SHOP-104, SAV-635, SAV-793, SAV-1126, MAGIC-STAFF-022 | still silent | - |
| DIV-225 | ui-and-settings | ITEM-ARMFILL-032, SAV-CITYMOVE-512, ALM-127, ITEM-SPELLMOVE-132, TERR-MOVE-057 | still silent | - |
| DIV-1457 | ui-and-settings | VIDEO-MUSIC-066, ANIM-120, SESS-CLOCK-021, SHOP-ANIMATION-084, VIDEO-067 | answered | VIDEO-067..070, ANIM-119, ANIM-120 |
| DIV-1503 | ui-and-settings | SPR256-CURSOR-046, TEXT-CHARGEN-029, DLG-PANEL-035, VIDEO-051, VIDEO-036 | still silent | - |
| DIV-129 | world-map | ALM-SEC-002, MOVE-083, TOWN-141, TOWN-144, SHOP-EFFRETRY-068 | still silent | - |
| DIV-131 | world-map | TOWN-145, HERO-TARGET-024, MAGIC-BOLTSHAPE-070, DLG-LINE-038, MOVE-GATE-035 | narrowed | TOWN-142, TOWN-144 |
| DIV-105 | world-map | MENU-053, SHOP-FIGURE-042, DIALOGUE-044, TOWN-096, AI-SPRAY-267 | still silent | - |
| DIV-1479 | world-map | TOWN-124, TOWN-038, TOWN-120, DIALOGUE-044, SAV-1098 | still silent | -; TOWN-486 reads the counter helper, not a scroll click during travel |
| DIV-1515 | world-map | MENU-053, TOWN-121, TOWN-038, TOWN-124, TOWN-096 | still silent | - |
| DIV-1516 | world-map | TERR-TILE-044, SESS-VIEW-030, TOWN-038, TERR-FOG-089, DLG-LIFE-005 | narrowed | TOWN-486 |
| DIV-1545 | world-map | TOWN-486, MISSION-MSGLINE-057, TOWN-444, SHOP-SCREEN-034, TOWN-414 | narrowed | TOWN-486 |
| DIV-1762 | world-map | MENU-052, TOWN-045, ITEM-STARCOMP-100, MENU-053, AI-356 | still silent | - |
