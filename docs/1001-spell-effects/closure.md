# 1001 — Spell effects: closure

Research pin: `744214fe1f4b461cbd2d6766390b850ab15779ab`. The story consumes
`MAGIC-RING-048` from that pin. Canonical simulation format version: 53.

## Shipped result

All 28 installed spells now reach one ordinary application path. Point effects alter canonical
actor state. Area effects call the same path over decoded cells and cadence. Wall of Earth changes
and restores ordinary ground and spirit passability. Book casts have one actor-owned wind-up,
release and recovery timeline. Range, current perception, useful-target rules, facing, mana,
training and effect emission are decided at that simulation boundary.

The client routes the actual spellbook right button to autocast and shows the selected row with a
moving dashed border. It consumes canonical cast, attached-effect and area state for feedback. It
draws health and mage mana above the unit square, projects current spell characteristics, starts one
cast swing per release, and draws positive Heal feedback from the installed healing sheet.

Original-save construction restores the supported companion, skill and equipment state used by
Reniesta and Brian. Mission 40 binds Reniesta's stable identity to loss. The character-doll
compositor reads live weapon and shield slots.

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | All 28 spell rows supply their mode, effect, duration, distribution, radius, range and school. The EN and RU sweep reports 18 point and 10 area rows with identical counts. Installed picture 20 resolves through `projectiles.reg` to the healing sheet. |
| Runtime state | PASS | Pending casts, recovery, attached effects, area phase/cells, actor facing, skills, experience, protections, detector radius and the three Control Spirit source stats are canonical. Client-only Heal particles carry no simulation state. |
| Simulation | PASS | Synthetic tests cover every point family, three area modes, exact ring programs, layer conflicts, friendly fire, Fire Ball division, visibility, range, facing, training and useful Heal. The mission-91 witness applies one lasting effect, repeated area harm and Wall of Earth through live mission state. |
| Player input | PASS | `TestRightClickOnASpellCellSetsAndClearsAutocastThroughTheSink` reaches the right-button route and simulation command sink. Wrong button, empty and unknown cells do not toggle. Manual commands pass the same admission predicate. |
| AI | PASS | Automatic and idle-Heal selection filter spell condition, current perception and range before stable priority. Hidden or invalid nearer candidates do not suppress a visible valid candidate. Busy actors cannot admit a second action. |
| UI / HUD | PASS | Tests exercise the moving dashed border, one swing, current popup projection, fog-gated effect marks, actual bar coordinates and actual healing-sheet sprite placement. The owner reports no defect in the implemented playable and visual paths. |
| Triggers / scripts | PASS | Script casts 21 and 24 enter the ordinary application owner. Script-created casts remain separate from actor-directed perception admission. Mission-40 loss uses the ordinary death and outcome path. |
| Inventory / equipment | PASS | Weapon spells share ordinary application without book training. Restored staff combat values and live weapon/shield doll layers use the same equipment projection as fresh construction. |
| Persistence / save-load | PASS | Version 53 round-trips mid-wind-up, mid-recovery, facing, skills and experience, attached effects and area phase byte-identically. Derived values are recomputed rather than duplicated. Client particles are intentionally not serialized. |
| Campaign / session | PASS | Real mission 91 supplies terrain, table and world wiring. The original mission-40 and mission-41 saves exercise party restoration, loss, healing, equipment and presentation. |
| Shipped content | PASS | Both roots contain 28 maps and the same 28-row spell vocabulary. The sweep records every script-reached cast id and effect mode. Both roots produce identical spell, save, doll and healing-art measurements. |
| Interactions | PASS | Damage resistance, health/mana clamps, physical Bless/Curse rolls, invisibility targeting, movement domains, action ownership, death, skill accounting, full-sheet recomputation, equipment, popup projection, fog and projectile lifetime are exercised at their owning boundaries. |

No known in-scope GAP remains.

## Synthetic evidence

The tests use constructed spell rows, actors, save-member records, registries and sprite sheets. No
installed asset or original-save byte is present in a test or git history.

- `pkg/sim/spelleffect1001_test.go` covers lasting attach, refresh, annihilation, expiry and
  reversal; Bless and Curse; Invisibility and detection; protection and Drain Life; cloud cadence;
  multi-target training; Teleport; Fire Ball division; layer conflicts; Wall of Earth; persistence;
  useful Heal; Control Spirit; and the six decoded staged-area programs.
- `pkg/sim/autocast_test.go`, `heal_test.go` and `spell_visibility_test.go` cover long-run automatic
  rate, range boundaries, owner/allied/neutral Heal priority, all-healthy refusal, preservation of
  orders, current perception, release-time revalidation and post-release projectile commitment.
- `pkg/sim/carry_test.go`, `combat_test.go`, `weaponspell_test.go` and cast-form tests cover the one
  action timeline, competing producers, one release, recovery, no stale queue, weapon replacement
  and persistence during the action.
- `pkg/sim/skill_test.go` covers the spell row's own Sphere, non-crossing and crossing awards,
  one-level cap, exact experience accounting, immediate projection and no award on refusal.
- `pkg/game/originalparty_test.go` carries a non-primary companion with six skill/experience slots,
  worn and carried items through filtering, construction, save/load and projection. It also covers
  staff spell reconstruction and duplicate/reference controls.
- `pkg/game/mission_test.go` covers Reniesta loss and the non-critical control. Doll tests run the
  actual figure compositor with weapon and shield independently.
- `pkg/game/spellbolt_test.go`, `pkg/ui/spellbolt_test.go`, `pkg/ui/autocast_test.go`,
  `pkg/ui/healthbar_test.go` and popup tests cover one swing, independent projectile flight, the
  right-button route, moving dashes, final HUD coordinates, current actor projection and actual
  decoded-sheet placement for Heal feedback.

## Installed spell and mission witness

The command below was run once per root with `AGAINROM_ASSETS` set. It opens no window.

```
go run ./cmd/spelleffectcheck -mission 91
```

Both roots report the three summary lines below. The tool prints all 28 spell rows first and then
these three; each is one long line here wrapped for width, and nothing within them is elided.

```
spell table: rows=28 point=18 area=10 distributions=[{1 18} {3 5} {4 2} {5 3}]
  effect-kinds=[{0 16} {1 1} {2 3} {3 2} {4 2} {5 1} {6 1} {7 1} {8 1}]
  effect-modes=[{0 16} {1 11} {2 1}]
campaign sweep: maps=28 cast-at-cell=[{3 3} {8 3} {17 1} {19 23} {21 5}]
  cast-at-unit=[{5 4} {10 4} {16 4} {20 3} {22 4} {23 1}]
  cell-effect-age=[{3 1} {19 22}]
mission 91 controlled wiring: bounds=80x80 patch=(30,28) point-speed=10->15 attached=1
  area-hp friend=1000/1000/990 enemy=1000/1000/996 outside-wall=1000/1000
  wall-route ground=true/false/true ghost=true/false/true air=true/true/true
```

`outside-wall=1000/1000` is the discriminating half of the wall witness: a unit standing beside the
wall's cells, not on them, takes nothing. Without it the wall figures are consistent with a wall
that harms everything nearby.

The controlled part is synthetic state inside a real mission-91 world. The mission loader, installed
spell table, grid, route owner and effect owner are real. The chosen actors and commands are
controlled so the result does not depend on a shipped trigger becoming ready.

The 28-map sweep derives its population from `game.ArchiveMaps` and the Assets table. It does not
contain a hard-coded shipped row list. It reaches both actor-targeted script casts and cell-targeted
script casts through the same compiler used by the game.

## Original-save witnesses

The files were read in place and were not changed.

| Save | Size | SHA-256 | Decoded label and mission |
|---|---:|---|---|
| `gameversions/saves/2026-08-15/game0019.sav` | 38,615 | `7F4D7B736C3011DF1C8E41B6B16CCC087F9CED07309F6FC9F503BE4CA72529B8` | `40 brian`, mission 40 |
| `gameversions/saves/2026-08-15/game0020.sav` | 33,248 | `99168DCA43E8F3053C864994B6B65AB8D2126264066D0CDC4D8B1C41FA4E4DD8` | `we have brian !!`, mission 41 |

The owner invocation was used for both roots. `-saves` named a temporary directory outside the
install and `-orig` named the preserved owner-save directory. Representative commands:

```
$env:AGAINROM_ASSETS='<seat>\gameversions\en'
go run ./cmd/savecheck -saves <temporary-directory> -orig <seat>\gameversions\saves\2026-08-15 -name game0019.sav -rebuild 40 -kill-member npc:22 load
go run ./cmd/savecheck -saves <temporary-directory> -orig <seat>\gameversions\saves\2026-08-15 -name game0019.sav -rebuild 40 -heal-witness npc:22 load
go run ./cmd/savecheck -saves <temporary-directory> -orig <seat>\gameversions\saves\2026-08-15 -name game0020.sav load
```

The corresponding RU commands differ only in `AGAINROM_ASSETS`.

### Reniesta

Both roots restore stable member `npc:22` as entity 48 with health 23/23, mana 139/139, class 24,
spellbook `0x41042`, worn codes
`[33037 0 0 0 0 0 63278 63531 0 0 0 0]`, and no carried codes. The staff resolves to
Wood Staff with live spell damage 5–10. The simulation and hand tooltip both report 5–10; the
reported 0–0 projection is no longer present.

The full-health window runs 64 ticks with zero casts. The witness then uses live movement and death
commands to establish a controlled non-combat scene, wounds one eligible member, and advances the
world. Exactly one seven-tick cast restores 8 health, applies the installed mana cost while ordinary
regeneration remains active, awards 3 experience to school 2, emits one feedback burst with seven
sprites, and stops at full health. Observed mana is 139 before and 136 after the interval. A second
64-tick window has zero casts and zero feedback. The release travels from `(76,109)` to `(77,108)`.
Facing is written at admission; this same-cell witness correctly retains facing 32.

Killing `npc:22` through the live driver sets outcome 2. Killing an appended controlled
non-critical member through the same path leaves outcome 0. The fresh mission-40 path and the
original-save rebuild use the same stable binding.

### Brian

The mission-41 save decodes Brian at human record offset `0x001111`, stable human id 79, unit row 6
and cell `(43,68)`. It carries body 41, Reaction 39, Mind 25, Spirit 21, health 157/157, skill levels
`[0 26 3 0 0 1]`, per-skill experience `[0 10144 331 0 0 100]`, and total 10,575. The displayed
total is the sum of the six decoded slots; 10,575 is not hard-coded.

The supported participant subtree contains seven worn codes and zero backpack codes. Its live worn
array is `[8486 0 0 0 0 9770 10032 10291 10550 10810 0 11326]`. Brian enters the live party as
`player:brian`; his equipped Steel Two Handed Sword reports damage 13–26. The implementation
restores every item present in the supported subtree and does not invent the additional items the
owner expected. The unsupported save axes are listed in `DIV-026` in player-visible terms.

### Mission-41 mercenary dolls

The measurements below come from `game0020.sav`, which this document decodes as mission 41. An
earlier revision of this section headed them mission 40; the digests were always the mission-41
save's own.

The owner-visible actors are stable live entity ids 36, 37 and 38. Each has worn array
`[20803 45603 0 21569 21826 22088 22352 0 47413 47672 0 48188]`, weapon `0x5143` and shield
`0xb223`. Both roots produce the same compositor measurements for all three actors:

```
full        8989bdf402094f5ff024fb16add3a0c09f1cdb3a0ad72d6ec658cd2ad0cc5c38
bare        74e1e3b3a2d9a595e2f83454db670c6e0b8c9e34bc83a0ff02e2957bcc276fdb
weapon-only 00ec523854289e4b861e40c3982f17c3d146cc65c144c54c6a77428ded1ecaa4
shield-only b4d29a6d7525a2695ac5ef55accc3b600a95eaf67c04d63198391d6b99221425
```

All four compositions draw. The full composition differs from the bare, weapon-only and shield-only
controls. The diagnostic calls the actual compositor; it does not infer layers from item names.

## Healing-sheet witness

The Assets diagnostic resolves picture 20 identically on both roots:

```
path graphics/projectiles/healing/sprites.16a
payload sha256 13cf1049161a32f9fda73fb87b81262e4dc22e0b7ea1732aefe01eec2369d501
frames walked 8; accessible phases 7; art 12x12; registry centre 8,8
```

The target-local consumer uses only the seven accessible phases. The ordinary travelling consumer
continues to suppress picture 20. Synthetic rendering proves that seven independently phased
instances use the decoded sheet, rise, clip, obey fog and expire. No procedural fallback is present.

## Owner observation

**Superseded, and kept as the record of what was believed at the round-1 push.** At that time the
owner had tested the playable result and reported no visible or playable defect, and by his ruling
that was a PASS witness for the implemented right-click and moving border, HUD bars, Heal animation,
one-swing pacing, facing, popup refresh, sequential actions, visibility boundary, post-release travel
into darkness, Brian presentation and mercenary dolls.

He has since reported fifteen presentation defects and one simulation defect, on 2026-08-15, listed
in `pipeline/OWNER-RULINGS.md`. Two of them were in this contract and were fixed in the fix round;
the rest are a later story, and Stone Curse's immobilisation is a later story and a research
question. **The paragraph above is therefore no longer a PASS witness for the visual paths it
names.** Owner observation is not ROM1 research evidence in either direction, and it does not replace
the automated gates or the headless measurements above.

## Addendum reconciliation

| Requirement | Before | Evidence after | Adjacent control |
|---|---|---|---|
| Reniesta staff | Owner saw 0–0 after original-save load | Both roots report one live 5–10 value to simulation and tooltip | Fresh construction uses the same equipment projection |
| Autocast gesture | Activation and state were unclear | Actual right-button route toggles the simulation command; dash phase advances | Left button, empty and unknown cells do not toggle; manual cast coexists |
| Mission-40 loss | Reniesta death did not lose | Stable `npc:22` death produces outcome 2 | Controlled non-critical death leaves outcome 0 |
| One swing and cast duration | Fire Arrow appeared to swing and fire repeatedly | One admitted cast has one event, one swing start, one release and recovery | Long projectile flight does not extend recovery or restart animation |
| Useful Heal | Reniesta healed healthy members repeatedly | Healthy 64-tick windows have zero casts; one wound produces one useful Heal | Full, dead, hostile, hidden and out-of-range targets spend nothing |
| Effect feedback | Effects were not visible | Live attached/area state drives marks; one positive Heal drives one installed-sheet burst | Expiry and full-health refusal remove or emit no feedback |
| Unit bars | Mana was absent and health crossed the unit | Health and blue mage mana are above the final unit square | Non-mages, invalid maxima, clipping, relief and moving footprints are covered |
| Range and idle targets | Automatic selection did not enforce useful in-range targets | All candidates are filtered; Heal uses owner, ally, neutral tiers | Invalid nearer, exact boundary and active-order cases are covered |
| Training and recomputation | Successful use did not update the school and sheet | Ordinary book apply awards the row's Sphere and recomputes in the same state | Refusals and weapon casts do not train; area count is explicit |
| Popup refresh | Skill growth was visible but popup values were stale | Popup obtains current canonical characteristics on every paint | Flat mana cost stays flat; Mind and member switches are covered |
| Sequential actor actions | Staff, Fire Arrow and Heal appeared adjacent or overlapping | One busy predicate owns attack, book cast, weapon spell and automatic producers | Rejected competitors spend, train and emit nothing and are not queued |
| Facing | Caster did not reliably face the target | Admission writes the established octant once before animation | Self casts and refusals retain facing; movement does not restart it |
| Brian restoration | `we have brian !!` loaded without Brian | Stable id 79 restores the decoded sheet, XP and seven worn items | Duplicate and non-persistent member controls prevent injection |
| Mercenary dolls | Three dolls lacked sword and shield | Entities 36–38 each compose both live held layers | Weapon-only, shield-only, bare and full outputs differ |
| Visibility | Actors cast into black or current fog | Actor-local current perception is required at admission and release | A committed projectile continues after later visibility loss; no new hidden-target cast starts |

## Research reconciliation

- `MAGIC-EFFMODE-009`, `MAGIC-CEIL-013`, `MAGIC-ARM-014`, `MAGIC-EFFECT-015`,
  `MAGIC-ATTACH-016`, `MAGIC-TARGET-017`, `MAGIC-TRAIN-018` and `MAGIC-SING-019` define the
  ordinary point, attachment, training and singular paths. Control Spirit carries Reaction, Mind
  and Spirit as source state and applies the published three assignments.
- `MAGIC-AREATICK-036` through `MAGIC-LIGHTDARK-044`, `MAGIC-WALLBLOCK-045`,
  `MAGIC-FIREDIV-047` and `MAGIC-RING-048` define area visit order, layers, cadence, passability,
  Fire Ball division and staged coordinates. Ring stages run at ticks 0, 3 and 6 rather than the
  earlier provisional two-tick wording.
- `MAGIC-AREACOST-046` is established and is not implemented. `DIV-021` records the resulting
  movement-cost fidelity debt. It is not used as a substitute for Wall of Earth passability.
- `SPR16A-CAST-028`, `SPR16A-PROJ-024`, `SPR16A-PROJ-026`, `SPR16A-ALPHA-025`,
  `ANIM-PHASECLOCK-028`, `ANIM-CAST-027` and `MAGIC-CASTSPAWN-033` define the installed Heal
  sheet and the ordinary no-blit arm. Only target-local arrangement, rise and lifetime are
  owner-authored.
- `AI-RAND-058` and `AI-RANGE-102` establish ROM1's random generator. This story retains the
  simulation's existing deterministic generator; `DIV-027` records the deviation.
- No promoted claim was refuted. Owner-authored cross-producer ordering, facing/visibility snapshot,
  UI affordances, Heal composition and mission-40 loss remain separate from ROM1 claims in
  `DIV-022` through `DIV-025`.

`DIV-001` and `DIV-002` are CLOSED. `DIV-021` and `DIV-026` through `DIV-027` remain OPEN.
`DIV-022` through `DIV-025` are ACCEPTED owner-authored boundaries.

## Persistence and customisation

Version 53 adds canonical pending casts, attached effects, area records and the actor fields required
by their consumers. The entity record is 260 bytes. Reaction is at offset 252 and Spirit at offset
256; Mind remains in its existing canonical slot. The version-free binary tests pin offsets,
truncation, malformed forms, round trips and digests. Versions 51 and 52 remain returned and unused.

The installed spell table is the seam for row count, magnitude, mode, duration, distribution,
radius, range, area lifetime and school. Fixed engine limits are six area layers, six records per
anchor, byte spell/stage/direction coordinates, word duration, eight orientations, fixed ring
program capacity and the canonical serialized widths. Lifting these engine widths changes engine
representation. It does not require changing a shipped file. Editing installed table data changes
the install and is not performed by this story.

## Fix round 2

The matrix above was written at the round-1 push, `5cee2da`. It recorded twelve PASS verdicts and
no GAP. The mandatory adversarial review ran at that sha under four lenses — point effects, area
effects, client and persistence, actions and regressions — and **all four returned INCOMPLETE**,
each with a reproducing counterexample. The matrix was therefore wrong when it was written, and the
verdicts below replace it. The consolidated findings are `round2-review.md`; the three fix lanes'
own measurements are `round2-orders.md`, `round2-area.md` and `round2-point.md`.

One finding was withdrawn rather than fixed: an idle mage's unbidden Heal on a wounded ally, and
the school experience it earns, are the owner's intended default (2026-08-15). One handed premise
was refuted by the lane it was handed to: `areaLife`'s `rule.ID == 21` branch was called
unreachable by the seat and is reachable through `SpellCharacteristicsFor`, where it gave Meteor
Storm an invented lifetime of 12 against the decoded 94.

The round changed 45 files, 6160 insertions against 353 deletions. `docs/DIVERGENCES.md` grew from
27 rows to 43; the new rows are `DIV-028` through `DIV-043`.

### Re-derived twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | Unchanged from round 1, with two corrections: Wall of Fire reads the `Distribution system` column rather than a hard-coded table, and Stone Curse takes its magnitude from the installed row rather than a literal. Control Spirit resolves the `Units` row named `Ghost` and takes its class key, domain, speed, sight, reach, token size, dying time, experience value, protections and combat block. |
| Runtime state | PASS | Two classes of unloadable save are closed. A wall landing discards stored routes that now cross a cell its own mover cannot cross, so the byte form no longer refuses a state a tick produced. Occupant slots are keyed by movement domain. |
| Simulation | PASS | An effect may not take `Entity.Speed` below 1, so a slowed unit is slower and not unrated. Effect removal restores exactly what was applied. A refused application takes no mana, on the unit form and now on the cell form. Fire Sacrifice's payment of the caster's health and mana happens after the refusals: measured under the revert, a refused cast left its caster at 1 health and 0 mana. |
| Player input | PASS | A movement order is never refused for busyness and never suppresses unbidden casting. A group order addresses every living member. An attack order on a weapon-spell carrier is admitted as for a plain fighter; actor-local perception gates the release alone. A click on a second enemy retargets. |
| AI | PASS with a disclosed authorship gap | Automatic selection never picks Teleport, and a stored autocast naming it is cleared. The map-owned mage's spell decision is built without cited authority and without a `spec.md` sentence; it is recorded as `DIV-029`, type UNKNOWN, and is measurable on shipped data — mission 90 drives to 150 alive / 4 fallen with the arm and 153 / 1 without it. |
| UI / HUD | PASS for what this story owns; the presentation defects are a later story | Area-effect sprites are handed positions in `ShotScale` units and now draw on their own cells; before the fix a cell at (76,109) drew at world pixel (9,13) instead of (2432,3488). Character-sheet protection rows read the live actor rather than a mission-start derivation. A unit's bars are witnessed against the northern cell's glyphs at all five scales, with the one mana-bar/selection-rim crossing recorded as `DIV-043`. The owner's eight further presentation reports of 2026-08-15 are a separate story, listed in `pipeline/OWNER-RULINGS.md`. |
| Triggers / scripts | PASS | A script-cast point row lands state through `ordinaryEffect(-1, i, rule, power)`. How many of the 20 shipped instant-24 nodes land state is not measured, and the code comment says so rather than claiming coverage. |
| Inventory / equipment | PASS | Unchanged from round 1. |
| Persistence / save-load | PASS with one disclosed exception | Version 53's authority paragraph now sits above the constant in `pkg/sim/binary.go`. `sim.GhostTemplate` is install-derived input on the world and is deliberately absent from the byte form, so two worlds equal in bytes may differ in which template they raise from; recorded as `DIV-041`. |
| Campaign / session | PASS | The mission-91 witness is unchanged, `outside-wall=1000/1000` on both roots. The mercenary-doll digests come from `game0020.sav`, mission 41. |
| Shipped content | PASS | `pipeline/check-milestone.sh` reports both roots at 28 maps with the recorded per-mission script counts, and the escort drive lost at tick 224 with 4 of 36 units moved and 1 fallen — by design for an unattended escort. The census did not move: no shipped script casts an area row at a cell already holding six records. |
| Interactions | PASS | Bless no longer writes its magnitude into astral protection. Invisibility is broken by approach rather than by being hit. Prismatic Spray applies to the living only. A cast no longer removes the caster's own movement order through automatic self-targeting. |

No known in-scope GAP remains. Two aspects carry a disclosed row rather than a clean PASS, and both
rows are typed and open.

## Gates and milestone

The complete Go chain was run on clean commit `bf4ec33` with
`GOCACHE=<seat>\.gocache-impl1001`:

```
go build ./...                              PASS
go vet ./...                                PASS
gofmt -l <tracked and untracked Go files>   PASS, no output
go test -trimpath -count=1 ./...            PASS
scripts/check-no-game-assets.sh             PASS, clean tree scan
```

The asset guard used Git's installed Bash; the WindowsApps `bash.exe` stub is not executable on this
host. **`PATH` must keep `git` reachable.** The script shells out to git, so a `PATH` of
`/usr/bin:/bin` alone makes it exit 127 having scanned nothing, which is a gate that cannot run
looking like one that passed. The invocation that produced this result prepended those two
directories to the existing `PATH` rather than replacing it.

`cmd/missionrun` was built from this worktree. With `AGAINROM_ASSETS` naming the EN install,
mission 10 and mission 20 both exit zero and print zero `UNSUPPORTED` lines at tick 1. The baseline
already carries zero unsupported nodes for these two missions after the earlier script stories, so
this story leaves the script-gap census unchanged at 0 and 0. Its player-visible result is the
spell effect, action, HUD and save behaviour measured above rather than a script-opcode fall.

The deletion set against `b51b439f09160968aa2b0dc115b820f63301ed3c` is empty. The commit range
contains no `Co-Authored-By` trailer. `git submodule status` reports research at
`744214fe1f4b461cbd2d6766390b850ab15779ab` without a `-` or `+` state marker.

### Fix round 2 gate

The same chain was re-run by the seat on the merged fix round, on a clean tree at commit `7a4e41f`:

```
go build ./...                              PASS
go vet ./...                                PASS
gofmt -l <tracked and untracked Go files>   PASS, no output
go test -trimpath -count=1 ./...            PASS, 39 packages with tests, 1610 cases in pkg/sim
scripts/check-no-game-assets.sh             PASS, clean (tree scan)
pipeline/check-preserved-installs.sh        PASS, 162 files, both roots as recorded
pipeline/check-milestone.sh                 PASS, both roots, census unchanged
```

The deletion set against master is empty. No commit in the range carries a `Co-Authored-By` or
`Claude-Session` trailer. `git ls-tree HEAD research` records the gitlink at
`744214fe1f4b461cbd2d6766390b850ab15779ab`, the pin frozen at the story's start.

One witness was re-verified here rather than taken on report. Removing the refusal hoist at the head
of `landAreaFacing` and re-running `pkg/sim` fails
`TestAFireSacrificeAtAFullAnchorDoesNotSpendItsCaster` with "a refused Fire Sacrifice left the caster
at 1 health and 0 mana, want 90 and 60". The line is witnessed and the witness discriminates.

### Cut list read against the owner's own words

The seat reads a story's cut list against the request that opened it, at the landing, because a gate
cannot tell that a cut removes the subject of the request.

This contract cuts one thing: area movement-cost multiplication from `MAGIC-AREACOST-046`, which
remains `DIV-021`. The owner's request was spells that have their effects. A second mutable terrain
plane is not required to produce any spell effect or Wall of Earth passability, so the cut does not
remove the subject.

The contract does not claim spell art. It claims that lasting point marks and area activity are
driven by the live effect records and disappear with them, and that is built. The owner reported
fifteen presentation defects on 2026-08-15 — bolt shape, per-cell area animation, Shield's shell,
protection head anchors, Light and Darkness lighting, Wall of Fire's burn frames, Fire Ball's tail,
Fire Sacrifice's spread, the acid cone's origin, the floating Wall of Earth sprite, Meteor Storm's
falling rocks, Stone Curse's stone mask. Two of them were in this contract and were fixed: the area
sprite `ShotScale` unit fault and the character sheet's dead protection rows.

One more of them was NOT presentation, and this correction belongs at the landing rather than in the
later story's scope. The acid cone's origin is this contract's own subject: a staged area anchors at
the aimed cell, so the cone's CELLS start where the player clicked, and the damage follows the cells.
It is a simulation defect, it is recorded as this landing's open gap in the round-3 section above,
and it is in a lane. Whether Fire Sacrifice's own report is mechanics or appearance turns on whether
its installed row is self-targeted, which the same lane is settling. The remaining twelve are a later
story, scoped from `EXP-0175` through `EXP-0179`. **This story delivers the mechanics of the
spell effects and not their appearance**, which is stated here so the reader is not left to infer it
from the contract's silence.

### Research arriving after the pin

`MAGIC-AREARADIUS-054` was promoted to research master after this story's pin was frozen and bears
on one thing built here. It establishes that `MAGIC-POWER-004`'s power-scaled area radius does not
reach the `AreaEffect`: the radius is the shipped `Radius, Length/2` column, read once at the single
construction site. The claim is not in this story's pin, which is why it is recorded here rather
than cited in `spec.md`. This build reads `rule.Radius` directly and does not scale it with power, so the
implementation agrees with the later claim. No change follows; the note is recorded so a later reader
does not re-derive it.

## Fix round 3

The second adversarial review ran at `70a7b74` under three lenses -- the fix round's own claims, the
untouched adjacent subsystems, and the documents against the code -- and all three reviewers returned
INCOMPLETE. Six counterexamples stood, one of which wrote a save the loader refuses, and two of which
were regressions introduced by round 1's own fixes. Four lanes closed five of the six. Each was
verified at this seat by reverting the fix with its witness in place, not by reading the report.

**A move order no longer writes a save that cannot be loaded.** `KindMoveTo` and the group
destination writer rewrote an entity's target and left its stored route standing. The byte form
requires a non-empty route to end at the entity's target, and the game's save is that byte form
verbatim. An ordinary mover never showed it because its own walk discards a mismatched route and
re-searches the same tick; it needs an actor the walk loop skips, which a pending book cast produces
and an owed transit crossing already produced on master. Both writers now drop the route the write
invalidates. Under isolated revert of each writer the other's witness stays green, so both are
independently load-bearing. The 28-map campaign sweep is byte-identical including its hash column
with the fix present and reverted.

**An area effect reaches every actor covering a cell.** Round 2 capped the per-cell read at one
ground-layer actor and one flyer, which is what the decoded cell record holds. That cap is safe in
the original only because its own footprint registration is all-or-nothing, so it never lets two
actors overlap. This build's occupancy plane is anchor-only and its Teleport landing tests terrain
alone, so two ground actors can share a cell here, and the cap made the second one immune to every
area effect. It is reachable on 147 large actors across the 28 campaign maps through ordinary
movement, and with no large unit at all through Teleport. One round-2 assertion was flipped in the
fix: it asserted that a second ground actor takes zero damage, which is the defect written down as
correct.

**A clamped effect delta is accounted rather than discarded.** Round 2's floor was exact for one
effect in isolation, but the clamp is decided against the target's current value, which already
carries every other active effect. Three effects at seven each on a base of ten ended at fifteen; a
fresh derived recompute carried no floor at all and fed a negative speed to the movement readers,
which the group reader narrows to a wrapped byte. Whatever a clamp cannot deliver now moves onto a
still-active sibling of the same kind. Health is excluded, because its ceiling is reachable by
ordinary healing unrelated to any effect; that exclusion had no witness and the lane built one.

**A weapon-borne spell reaches the ordinary apply.** The release applied damage alone behind a
damaging gate, so a mage carrying a non-damaging weapon spell had his attack replaced by a release
that then refused itself -- zero damage on mission 130 where the same fighter unarmed does 371. A
weapon-borne row now routes through the same application a book cast reaches. The item tooltip
carries the same Prismatic Spray exclusion the release has. The fighter's rider arm, which had no
implementation at all, is built: three shipped Boulder Throwers now throw their Fire Ball.

**The sixth counterexample was open at the merge and was closed the same day.** The acid cone's
origin is this story's own code: a staged area anchored at the aimed cell and read the caster's cell
only for a direction byte, so the cone's cells, and therefore its damage, started wherever the player
clicked. Its lane was still running when the owner directed this story into master, and landed
afterwards. Acid Stream now anchors at the caster's own cell and the aimed cell supplies orientation
alone; Meteor Storm is unchanged and carries its own negative-control test. Witnessed at eight click
offsets covering both orientation families through the real cast path: with the fix reverted all
eight redden, north reporting the effect's own cell at (20,18) against the caster's (20,20).

Fire Sacrifice was settled from its installed row rather than assumed. Its maximum range is 0, the
same column and value Shield ships on, which forces a computed range of 0, so every book-cast
admission refuses an aim off the caster's own cell. Its anchor was already correct, and the owner's
separate report about it is about drawing.

**No third adversarial review ran.** Pipeline v2 makes it a mandatory gate and this landing did not
meet it. The owner directed the merge as author, which is his to do; recording it as anything other
than a skipped gate would misstate what happened.

**One resolution here is reasoned rather than witnessed.** Master's bare-handed unequip hotfix added
an argument that this branch's own new call sites did not pass, so the text merge produced a tree
that did not build. The derived-skill recompute now passes a subject-scoped value, which is exact
because the only producer of an unequip acts on the inventory subject alone; the two original-save
restores pass whether the restored slot is occupied, which preserves that path's existing behaviour
exactly. No test covers a level gain landing on a hero the player has just stripped, and that
witness is owed.

### Round-3 twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | Unchanged from round 2. |
| Runtime state | PASS | The last known class of unloadable save is closed at both writers of a move order, each independently witnessed by isolated revert. |
| Simulation | PASS | Acid Stream anchors at the caster's own cell, closed after the merge by the lane that was running at it. An effect clamp is accounted rather than discarded, an area effect reaches every actor on a cell, and a weapon-borne row applies through the ordinary path. One residual is disclosed at `DIV-064`: the six-slot admission pre-check still tests the aimed cell rather than the anchor, unreachable on shipped content. |
| Player input | PASS | Unchanged from round 2. |
| AI | PASS with the disclosed authorship gap | Unchanged from round 2; `DIV-029` still stands. |
| UI / HUD | PASS for what this story owns | The item tooltip no longer advertises damage the release refuses. The owner's presentation reports remain a later story. |
| Triggers / scripts | PASS | Unchanged from round 2. |
| Inventory / equipment | PASS | A weapon's own spell now reaches its carrier: the mage's replacement arm applies non-damaging rows, and the fighter's rider arm exists for the first time. |
| Persistence / save-load | PASS with the round-2 exception | `DIV-041` unchanged. The stale-route class is closed. |
| Campaign / session | PASS | Unchanged from round 2. |
| Shipped content | PASS | The milestone census is where it was recorded on both roots. Three shipped Boulder Throwers changed behaviour, measured at 130 and 69 extra damage over 240 ticks on missions 90, 111 and 140, with the attacker's own health unchanged. |
| Interactions | PASS | The two round-1 regressions are closed and each is witnessed by isolated revert. |

No in-scope GAP stands. One did at the merge itself -- the acid cone's anchor -- and the story
landed on the owner's own direction with that gap open and its lane running; the lane landed the fix
the same day. The skipped third adversarial review is not repaired by that and stays recorded above
as what it was.
