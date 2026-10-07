# Completion map — repository evidence classification

Classification of `againrom` against `docs/proposals/PROJECT-COMPLETION-MODEL.md` (PROPOSAL,
NON-NORMATIVE), read from the copy at
`scratchpad/proposal/PROJECT-COMPLETION-MODEL.md`. Every number below names the command or file
that produced it. This document is itself a record of one classification pass; it is not evidence
of ROM1 behaviour and does not change any gate, ledger, or milestone.

## Repository state at classification time

- Implementation master: `744b2fc` (`744b2fc77ed20121025ca4c0f3f49dcdca9b7905`, 2026-08-21
  11:02:43 +0200). `git status -sb`: clean, `master...origin/master`, no ahead/behind.
- Research master: `d1d350d` (`d1d350db1d834ab03c923ba5e2055f3bd35ce984`, 2026-08-21 10:57:10
  +0200). `git status -sb`: clean, `master...origin/master`.
- Submodule pin: `git -C implementation submodule status research` prints
  ` d1d350db1d834ab03c923ba5e2055f3bd35ce984 research (remotes/origin/HEAD)` — no leading
  character, so the checkout matches the recorded pin.
- One branch is open and unmerged: `origin/proposal/integration-fidelity-harness`. It carries the
  three proposal documents (`INTEGRATION-FIDELITY-HARNESS.md`, `PROJECT-COMPLETION-MODEL.md`,
  `AGENT-REVIEW-BRIEF.md`) plus the landed chargen-seam hotfix already on master under a different
  commit. `bash pipeline/check-pin-forward.sh` reports `FAIL`: this branch pins research `18c89dd`,
  not a descendant of master's `d1d350d`. It is not landed and not normative.
- `implementation/docs/DOMAINS.md` names nine domains (confirmed by direct read). `scenarios/`
  holds 13 `*.json` files and `README.md` (confirmed by `ls scenarios/`).

Instruments run for this report, all read-only against the checked-out master and the two
preserved installs under `gameversions/en` and `gameversions/ru`:

| Command | Result |
|---|---|
| `go build ./...` (implementation) | exit 0 |
| `go vet ./...` (implementation) | exit 0 |
| `go test -trimpath -count=1 ./...` (implementation) | exit 0; 39 packages `ok`, 0 `FAIL`, 20 `[no test files]` |
| `gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')` | no output |
| `bash scripts/check-no-game-assets.sh` | `clean (tree scan)` |
| `bash pipeline/check-milestone.sh` | `ok`, matches the committed baseline on both roots |
| `AGAINROM_ASSETS=<en> bash scripts/campaign-sweep.sh` | ran; see campaign reachability below |
| `bash pipeline/check-scenarios.sh <en>` and `<ru>` | `ok (13 of 13)` on each root |
| `bash pipeline/check-release-tests.sh <en>` | `ok (34 of 34 install-gated tests ran and passed, 0 skipped)` |
| `bash pipeline/check-preserved-installs.sh` | `ok — 162 file(s), both roots as recorded` |
| `bash pipeline/check-pin-forward.sh` | `FAIL` — see above |
| `bash pipeline/check-div-claims.sh` | `selected 121 live row(s) of 121 (0 closed), citing 175 distinct claim id(s)`; 34 rows cite a claim carrying a retraction row |
| `bash scripts/check-claim-citations.sh` (implementation) | `ok (1178 distinct citations resolve against 1390 claims and 206 experiments under 783 prefixes)` |
| `bash scripts/check-claim-ids.sh` (research) | `ok (1390 ids, all distinct, 31 ledgers, 29 read back through tools/claim)` |
| `bash scripts/check-retraction-status.sh` (research) | `ok (229 overturned ids, every one marked)` |

`PIPELINE-STATUS.md` was read as a claim to check, not as evidence. Every figure below that also
appears there was re-measured independently; one disagreement was found and is reported in
Section 2.

## 1. Release level

**R1, confirmed. R2, not confirmed as a whole-campaign property; individual conditions of the
model's "campaign playable" definition are met for a small number of named missions and open for
the rest.**

R0 conditions (formats and engine layers load meaningful original data, core simulation/rendering
paths exercised): met. `bash scripts/campaign-sweep.sh` loads and script-compiles all 28 campaign
maps on the EN root; `bash pipeline/check-scenarios.sh` passes 13 of 13 install-backed and
synthetic scenarios on both roots.

R1 condition (at least one production player journey crosses menu/gameplay and reaches a real
mission outcome): met. `scenarios/0163-chargen-mission10.json` generates a character through the
production chargen screen and enters mission 10; `scenarios/0155-mission10-escort.json` drives
that mission through production simulation commands to a recorded outcome;
`scenarios/0163-mission-to-town.json` drives a mission to its verdict, crosses into town, and
asserts roster/purse/documents on both sides.

R2's own definition, checked against the model's seven-point "candidate definition of campaign
playable":

| # | Condition | Status |
|---|---|---|
| 1 | Fresh session starts through production menu/chargen | Met for at least one entry point (`0163-chargen-mission10.json`). Not exercised for every mission's own chargen arm. |
| 2 | Every mandatory mission enterable through campaign's production routing | Not established for all 28. All 28 load when addressed directly by mission number (`check-milestone.sh`, `campaign-sweep.sh`); only a handful are shown entered through the world-map/town click path itself (`1013-world-map-one-click.json` demonstrates the click-to-entry mechanism generically, for one mission). |
| 3 | Each mandatory mission reaches its verdict through ordinary simulation/commands | Demonstrated for a small number of missions (10, 20 partially, 30, 40) via scenario or story closure. Not demonstrated for the other ~24. |
| 4 | Verdicts cross into notices/town/world-map/next-mission selection | Demonstrated generically by `0163-mission-to-town.json` and by the 1007/1010/1013 world-map stories. Exercised for specific missions, not the full set. |
| 5 | Party/inventory/purse/documents/skills survive the transitions | Demonstrated for the missions the scenarios cover (`0163-mission-to-town.json`, `0159-mission40-join.json`, `1005-doll-carry-over-worn.json`). |
| 6 | Save/load interrupts and resumes a campaign run without developer repair | This build's own save/load is built and transaction-safe (`1008-save-safety`, `DIV-097`/`DIV-098` closed). No scenario among the 13 saves this build's own format mid-mission and reloads to resume play; `0152-save666.json` loads an *original* ROM1 save, which is a different capability (`DIV-026`, open). |
| 7 | No mandatory transition needs a debug command/map selector/hand-edited save | The production paths exercised by the 13 scenarios need none. `cmd/missionrun` and `campaign-sweep.sh`'s direct `-mission N` addressing are developer instruments, not part of any scenario's asserted path. |

R3 (feature complete): not met. Section 3 finds no domain reaching a state where "no known missing
feature required to experience the shipped campaign" holds; named examples include the AI escort
order's undecoded heal fork and idle turn (`0167-escort-tick`, "named not built"), the world-map
route's unhandled off-node/disconnected fallback (`DIV-129`, OPEN), and 21 distinct script
operations this build cannot run over 14 of 28 campaign maps (`check-milestone.sh`).

R4 and R5: not applicable. R4 requires R3; R3 is not met.

## 2. Campaign reachability

Numbers below are freshly measured this session; the instrument that printed each is named.

**Maps that load and script-compile**: 28 of 28, both roots (`pipeline/check-milestone.sh`,
`scripts/campaign-sweep.sh`). This is load, not completion (model rule 2).

**Script-node support**: `pipeline/check-milestone.sh` reports 59 unsupported script-node
instances (all `check` operations; zero `instant` operations, confirming
`PIPELINE-STATUS.md`'s "every INSTANT runs" line), 21 distinct (mission, opcode) rows, over 14 of
the 28 campaign maps: missions 40, 60, 71, 81, 90, 91, 100, 101, 111, 130, 131, 140, 150, 151. The
other 14 (10, 20, 30, 31, 41, 50, 51, 61, 70, 80, 110, 120, 121, 141) carry zero unsupported
operations. All three figures (59, 21, 14 of 28) match `PIPELINE-STATUS.md`'s current-dated
figures exactly, independently reproduced.

**Disagreement found.** `PIPELINE-STATUS.md`'s "Open" list states, for mission 20: "Instant 12 and
check 15 remain unsupported... Nobody has reached a corner alive." The first clause is refuted by
this session's own `check-milestone.sh` run: mission 20 carries zero unsupported script operations
on either root. That row in `PIPELINE-STATUS.md` is not dated to the current landing and appears to
predate the operations that later closed those two ops. The second clause ("nobody has reached a
corner alive," i.e., no completion witness) could not be checked either way from this repository:
no scenario or closure asserts mission 20 completed, but its absence does not confirm the claim
either, since a completion witness could exist and simply not have been searched for successfully.

**Unattended campaign drive**: `scripts/campaign-sweep.sh` over the EN root, 2000 ticks, no orders
issued (its own design): all 28 missions load; 0 of 28 reach a decided outcome (`0 decided` in the
summary row); total unsupported 59 (matches the census above); total *reached* unsupported (arms
the drive actually walked into) 7375. Zero decided outcomes under a zero-order drive is the tool's
documented design, not itself evidence of a defect — an unattended escort or combat mission is not
expected to resolve without orders.

**Missions with a player-command verdict witness**: `scenarios/0155-mission10-escort.json` and
`scenarios/0155-mission20-sweep.json` drive missions 10 and 20 through production orders to a
recorded outcome (mission 10's unattended-vs-hand-played distinction is itself recorded:
`pipeline/check-milestone.sh`'s own header states mission 10 is "passable by hand" but an
unattended drive loses by the mission's own escort design). `0156-mission30-cure.json` exercises
mission 30's instants 12/13. `0163-mission-to-town.json` drives one mission to verdict and crosses
to town. `0159-mission40-join.json` drives mission 40's own trigger to the knight hand-over. No
instrument in the repository demonstrates a player-command verdict for the remaining ~24 missions,
and none demonstrates the full 28-mission campaign traversed in sequence.

**Required town/world-map transitions**: the route graph (`TOWN-116`/`117`/`119`/`120`), one-click
travel-and-arrival (`TOWN-121`), and marker painting are implemented and closed (`DIV-104`,
`DIV-135`, `DIV-130` in `docs/DIVERGENCES-CLOSED.md`). Two edge cases remain open:
off-node/disconnected-endpoint fallback (`DIV-129`, UNKNOWN/OPEN) and the search's tie-break rule
for a non-unique shortest chain (`DIV-131`, UNKNOWN/OPEN, further complicated by an unresolved
101-vs-110 instrument disagreement recorded in the same row).

**Campaign terminal state**: no instrument or scenario in the repository addresses reaching the
end of the 28-mission campaign, in either direction (first mission to last, or any defined
"campaign complete" condition). This is an absence of evidence, not a claim that it fails.

## 3. System completeness

Domains are `implementation/docs/DOMAINS.md`'s own nine. None is classified `FIDELITY`: every
domain below has at least one `OPEN` `FIDELITY-DEBT` row in `docs/DIVERGENCES.md` naming a place
where correct ROM1 behaviour is known and not yet implemented, which the model's `FIDELITY-DEBT`
type definition ("correct behaviour known, implemented otherwise for now") makes a direct
disqualifier under any reading of the `FIDELITY` tier. None is classified `ABSENT`: every domain
has a shipped, exercised player path. Two domains carry a named "not built" behaviour that borders
`PARTIAL`; both are kept at `FUNCTIONAL` here with the reasoning stated, and both are listed again
in Section 6 because the model gives no test to settle the boundary.

| # | Domain | Class | Evidence for | Evidence against `FIDELITY` |
|---|---|---|---|---|
| 1 | Assets | FUNCTIONAL | All 28 campaign maps load on both roots (`check-milestone.sh`); 13/13 install-backed and synthetic scenarios pass on both roots; wide decode coverage landed (`EXP-0188` `PathMap.bmp`, town/school/tavern/shop composition, `Data.bin` classes, text tables). | `DIV-166`: 8 of 8 sixteen-pixel `interface/` bitmaps still have no located reference site or reader for at least one edge; several RU-specific label sources undecoded (`DIV-165`). |
| 2 | Sim Core | FUNCTIONAL | `internal/archtest`'s lexical determinism scan (`determinism.go`, `dag.go`) gates `pkg/sim`; `pkg/sim` package tests pass (`go test`, 5.3s); campaign-sweep drives all 28 maps deterministically (fixed-tick, no clock/rand/float). | `DIV-054` (movement occupancy is anchor-only, OPEN), `DIV-055` (area effect application on a cell-slot collision, OPEN), `DIV-053` (stacked effect clamp recompute cost, OPEN), plus a large open `UNKNOWN` population (order-during-cast, AI spell arm, route invalidation, effect speed floor, and others). |
| 3 | Combat & Magic | FUNCTIONAL | `1001-spell-effects`/`1002`/`1003`/`1004` landed; `DIV-001`/`DIV-002` (point and area effect contracts) CLOSED; nine owner spell-presentation corrections landed as a hotfix line. | `DIV-037` (Control Spirit template), `DIV-056` (weapon-borne release), `DIV-076` (projectile-driver mark writer), `DIV-077` (Meteor Storm presentation), `DIV-079` (retained overlay/projectile draw order) all OPEN `FIDELITY-DEBT`; roughly two dozen further OPEN `UNKNOWN` rows under "magic /". |
| 4 | AI & Orders | FUNCTIONAL | Acquisition, chase, group orders, guard, escort and follow are landed and exercised in the campaign sweep (`0086-ai-engages` through `0167-escort-tick`); mission 10's waypoint drive shows engagement and death under AI control. | `0167-escort-tick`'s own landing note names "the defender's heal fork, the idle turn" as built-in-name-only ("named not built"); `DIV-029` (AI spell arm, UNKNOWN/OPEN). Whether the heal fork is a *required* behaviour or a fidelity refinement is exactly the ambiguity in Section 6, item 3. |
| 5 | Party, Items & Heroes | FUNCTIONAL | `1005-interactive-doll` closure: per-pixel doll hit test, hover, press, drag, equip/unequip/sell/drop, thirteen adversarial passes; inventory/sack/corpse-loot/training/stat-recompute stories landed (0110–0138 range, 0162). | `DIV-111` (starting weapon drawn but not equipped, OPEN `DEVIATION`), `DIV-112` (`WeaponMaterialized` decoded from a pre-existing save, OPEN `FIDELITY-DEBT`); the inventory drag-gesture parity across screens has a cluster of OPEN `UNKNOWN` rows (`DIV-084`…`DIV-092` range). |
| 6 | Campaign & Scripts | FUNCTIONAL | Mission start/trigger/verdict/town-return/world-map-routing mechanism demonstrated end to end for specific missions (10, 30, 40, and one generic mission-to-town crossing); trigger opcodes and mission flow stories landed (0063, 0093, 0121, 0122, 0164–0169). | 21 distinct unsupported script operations over 14 of 28 shipped maps (`check-milestone.sh`), which permanently disables the triggers holding them on those maps; whether "required behaviour" is scored per-mechanism (met) or per-shipped-map (not met on 14 of 28) is undecided by the model — see Section 6, item 4. |
| 7 | Town & Economy | FUNCTIONAL | Shop stock/pricing/buy/sell/wheel/wear-rule (`0157-shop`, three rounds); tavern hiring, school training including the decoded hidden `General` skill (`EXP-0187`), mercenaries; a real shipped weapon is traded end to end in `1005-doll-and-shop.json`. | `DIV-165` (school/tavern button labels hardcoded English; shop's own labels are now install-sourced), OPEN `FIDELITY-DEBT`. Most other open rows tagged `shop /`, `school /`, `tavern /` in the ledger are screen-composition (Client-domain) rather than economic-rule gaps; see Section 6, item 2. |
| 8 | Client | FUNCTIONAL | Every shipped screen is reachable and interactive end to end: menu, chargen, map/HUD, town shell (shared shop/tavern/school shell, `1006`), in-game menu (`0158`), dialogue (`0160`), world map (`1007`/`1010`/`1013`). | The largest and most currently active open population in the whole ledger: `DIV-166`/`167`/`168`/`169`/`170` (the 176-pixel room-plaque strip, still `OPEN` as of this landing, amended three times across three pin bumps and one hotfix without closing), `DIV-140`/`DIV-141` (menu accelerator physical-key mapping and a RU collision). |
| 9 | Persistence | FUNCTIONAL | This build's own save/load is transactional and safety-reviewed (`1008-save-safety`; `DIV-097`/`DIV-098` CLOSED); `0143`/`0147`–`0152` restore party/skills/XP from this build's format and from an original `.sav`'s straight-run fields. | `DIV-026` (original-save compatibility beyond `Token`/`Building`/`Effect`/`Player`'s straight run, OPEN `FIDELITY-DEBT`); `DIV-137`/`DIV-138` (world-map position/marker-selection cache is session state, not a persisted field — the one `GAP` row found in any closure's own aspect matrix, `docs/1013-world-map-arrival/closure.md`). |

Summary: **0 FIDELITY / 9 FUNCTIONAL / 0 PARTIAL / 0 ABSENT**, with two FUNCTIONAL calls (AI & Orders,
Campaign & Scripts) resting on a PARTIAL/FUNCTIONAL boundary the model does not define (Section 6).

## 4. Fidelity reconciliation

`docs/DIVERGENCES.md` holds two live tables (`Divergences`, `Authored where research is silent`);
`docs/DIVERGENCES-CLOSED.md` holds settled rows. Counts below are from a full column extraction of
both tables (44 + 77 = 121 live rows, 22 closed rows, 143 total), cross-checked against the file's
own row totals.

**Open known fidelity debt** (`FIDELITY-DEBT`, status `OPEN`): 28 rows (21 in `Divergences`, 7 in
`Authored where research is silent` — a row lands in the second table when it was *written* against
research silence and later reclassified without yet being built to match). Three further `OPEN`
`HOTFIX` rows (`DIV-044` ground pick, `DIV-068` structure shadow, `DIV-103` Teleport target
visibility) name a temporary implementation whose behavioural divergence, if any, is recorded but
not yet reconciled; the model's five-bucket list has no dedicated slot for `HOTFIX` (Section 6,
item 9).

**Accepted deliberate deviations**: 15 rows — `DEVIATION`/`ACCEPTED` (11: 10 in `Divergences`, 1 in
`Authored where research is silent`) plus `CONFLICT`/`ACCEPTED` (4: `DIV-072` Prismatic Spray
targeting, `DIV-075` Shield/Bless mark geometry, `DIV-093` Wall of Fire overlap, `DIV-121` school
skill order). A `CONFLICT` row is where an owner directive contradicts a promoted claim; all four
live `CONFLICT` rows are `ACCEPTED`, meaning the owner has ruled and no `CONFLICT` row is
outstanding.

A further 11 rows are typed `DEVIATION` but status `OPEN` (6 in `Divergences`, 5 in `Authored where
research is silent`): a deliberate difference from ROM1 that has not yet been owner-ruled
`ACCEPTED` or brought to `CLOSED`. The model's bucket list assumes a deviation is either accepted or
absent; these 11 sit outside both.

**Authored behaviour where research is silent** (`UNKNOWN` type): 64 rows, all in the `Authored
where research is silent` table (zero `UNKNOWN` rows remain in `Divergences`, consistent with the
2026-08-17 audit's stated intent of moving every research-silence row out of that table). 7 are
`ACCEPTED` (owner has ruled to keep the authored answer regardless of what research later says);
57 are `OPEN` (research question genuinely live). Per the model's own rule, none of these 64 is
counted as known ROM1 fidelity.

**Closed divergences**: 22 rows in `docs/DIVERGENCES-CLOSED.md`, by original type: 16
`FIDELITY-DEBT` (research published the ROM1 behaviour and the build was brought into line, e.g.
`DIV-104` world-map route, `DIV-070` NPC death suppression, `DIV-097`/`DIV-098` save/load
transactions), 3 `UNKNOWN` (research answered a prior silence, e.g. `DIV-004` shop character block),
1 `CONFLICT` (`DIV-145`, superseded by a later decode that resolved the conflict), 1 `DEVIATION`
(`DIV-017`, the shared character-panel widget), 1 `HOTFIX` (`DIV-094`, Darkness brightness, closed
when the five-percent instruction was withdrawn in favour of the promoted claim).

**Research claims that are provisional or retracted and still affect live expectations**:
`pipeline/check-div-claims.sh` reports 34 of 121 live divergence rows citing at least one claim id
that carries a row in `research/claims/retracted.md` (229 overturned ids total, of 1390 claims,
per `check-retraction-status.sh`). The script's own documented caveat applies unchanged: a claim is
usually retracted in part, and most citing rows are unaffected; this is a screening count, not a
count of confirmed-wrong rows. No instrument in the repository counts "provisional" (non-High,
not-yet-promoted) claims cited by a live divergence row; that would need a per-cited-id confidence
lookup across the 175 distinct ids the live table cites, which was not attempted here (Section 6,
item 8).

## 5. Journey integrity

**No curated integration-fidelity journey portfolio exists in the current repository.** The model's
own text is conditional: "use the curated integration-fidelity journeys *if that proposal is
adopted*." The sibling proposal that defines them (`docs/proposals/INTEGRATION-FIDELITY-HARNESS.md`,
journeys J1–J6) lives only on `origin/proposal/integration-fidelity-harness`, which
`pipeline/check-pin-forward.sh` reports as `FAIL` (its research pin `18c89dd` is behind master's
`d1d350d`) — the branch is unmerged and non-normative. No manifest, no `passed/applicable` count,
and no journey identifier exists anywhere on master.

**Nearest existing evidence**: `scenarios/*.json` (13 files, `pipeline/check-scenarios.sh`, `ok
(13 of 13)` on both `en` and `ru`) and `scripts/campaign-sweep.sh`. Read against the unmerged
proposal's own J1–J6 definitions purely as an informal correspondence (this mapping is this
report's own construction, not a repository artifact):

| Journey (unmerged proposal) | Nearest scenario | Coverage |
|---|---|---|
| J1 — new game to first playable mission | `0163-chargen-mission10.json` | Menu → chargen → mission entry, for mission 10 only. |
| J2 — mission success to town to next mission | `0163-mission-to-town.json` | One mission driven to verdict, crossed to town, roster/purse/documents asserted both sides, generic (not mission-specific) production path. |
| J3 — combat to loot to equip to later combat | Partial: `1005-doll-and-shop.json`, `1005-doll-carry-over-worn.json`, `0159-mission40-join.json` | Equip/unequip/sell/drop and worn-item carry-over across a mission boundary are witnessed; no scenario asserts that an equipped item changes a *later* combat outcome. |
| J4 — save/load across non-trivial state | Not covered | `0152-save666.json` loads an *original* ROM1 save; `1020-abort-witness.json` asserts a fresh new-game state after abort. No scenario saves this build's own format mid-gameplay and reloads to resume. |
| J5 — dialogue/trigger driven progression | Partial: `0159-mission40-join.json` | A map trigger's consequence (a hand-over) is asserted. No scenario exercises a dialogue panel gating and being responded to. |
| J6 — multi-mission continuity (N, N+1, repeated) | Not covered | `0163-mission-to-town.json` covers one hop; no scenario chains three or more missions. |

This is a coverage sketch, not a `passed/applicable` verdict: the proposal's own manifest fields
(`boundaries`, `observations`, `owner_intent`) do not exist for any of the 13 scenarios, so "which
boundary does this scenario really cross" is this report's inference from `scenarios/README.md`'s
prose, not a measured property.

`pipeline/check-release-tests.sh` (34 of 34 install-gated tests passing on the EN root) is broader
corroboration that production code paths touching a lawful install do not silently skip, but it is
a unit/integration test count, not a journey count, and the model's own rule 5 (never claim
`FIDELITY` from unit tests alone) applies to reading it that way.

## 6. Where the model forced a guess

Ten places. Each states the model's wording, what was looked for, why the repository does not
answer it, and what would answer it.

**1. What counts as "reconciled with current research" for the `FIDELITY` domain tier.**
Wording: *"`FIDELITY` — required behaviour is reconciled with current research except accepted
deviations/unknowns."* Looked for: a worked example distinguishing an `OPEN` `UNKNOWN` row (research
silent, not yet owner-accepted) from an `ACCEPTED` one for this purpose. Not found: the phrase
"accepted deviations/unknowns" could mean "deviations and unknowns, when accepted" (57 `OPEN`
`UNKNOWN` rows would then block `FIDELITY`) or "deviations, and unknowns generally" (an `OPEN`
`UNKNOWN` would not block `FIDELITY` by itself). The model's own R4/R5 text pulls the other way for
that later tier ("known research silence is reported as uncertainty rather than silently counted as
fidelity"), which is not repeated in the domain-tier definition. This report avoided the question
by finding an `OPEN` `FIDELITY-DEBT` row in every domain regardless (Section 3), so no domain's
classification here depends on the answer, but a future report with fewer open `FIDELITY-DEBT` rows
would need it resolved. A worked example in the model, or an owner ruling on the phrase, would
answer it.

**2. Mapping `docs/DIVERGENCES.md`'s free-text `Subsystem` column onto `DOMAINS.md`'s nine domains.**
Wording: *"Use the project's domain model rather than story directories."* Looked for: an index or
column in the ledger naming a `DOMAINS.md` domain per row. Not found: the ledger's `Subsystem`
column uses ad hoc labels ("shop /", "school /", "town square /", "magic /") that do not declare a
domain, and several straddle the boundary `DOMAINS.md` itself draws between Town & Economy ("the
rules never depend on a screen") and Client ("screens live in Client") — e.g. `DIV-142` "school /
training column rotation" is pixel-composition (Client) despite its "school" label. Section 3's
per-domain evidence lists were built by manual content judgment on a sample of rows, not a
mechanical join over all 121; a second reviewer doing the same join by hand could place some rows
differently. A `domain` column added to `DIVERGENCES.md`, or a documented mapping table, would
answer it.

**3. "Known required behaviour is missing" (`PARTIAL`) versus "fidelity gaps may remain"
(`FUNCTIONAL`).** Wording: both phrases are the model's own, in the `PARTIAL` and `FUNCTIONAL`
definitions respectively, with no test distinguishing them. Looked for: a rule for when a named
"not built" item is required rather than a refinement. Not found, concretely on AI & Orders:
`0167-escort-tick`'s own landing note names the defender's heal fork and the idle turn as "named
not built," while escort/guard/chase/group orders already work end to end. This report classified
AI & Orders `FUNCTIONAL`; an equally defensible reading calls the heal fork "required" (an escort
mission's defend order is incomplete without it) and classifies it `PARTIAL`. An owner ruling on
which named-but-unbuilt AI behaviours are "required" for the shipped campaign, or a `required: true`
flag in the divergence ledger, would answer it.

**4. Whether system completeness is scored per-mechanism or per-shipped-content-instance.**
Wording: *"required player path works end to end."* Looked for: whether "the path" means the mission
flow mechanism (start → trigger → verdict → town/world-map → next mission), demonstrated for a
handful of missions, or every one of the 28 shipped maps individually. Not found: `check-milestone.sh`
measures 14 of 28 maps carrying at least one script operation this build cannot run, which
permanently disables whatever trigger depends on it, on those maps specifically. Under a
per-mechanism reading, Campaign & Scripts is `FUNCTIONAL` (this report's choice); under a
per-map reading, it is `PARTIAL` (14 of 28 maps have a known missing behaviour). Since axis 1
(campaign reachability) already tracks "missions completable" as a separate count, one reading of
the model is that axis 2's domain tier is deliberately mechanism-level and axis 1 carries the
per-map detail — but the model does not say so explicitly.

**5. Axis 1's own "missions completable" has no single-command answer.** Wording: *"Track
separately: missions reachable; missions completable."* Looked for: an instrument that counts
missions completable through ordinary player-equivalent commands. Not found as a single number:
`check-milestone.sh` counts script-node support (a necessary-but-not-sufficient condition);
`campaign-sweep.sh`'s unattended drive produces 0 of 28 decided outcomes by design (it issues no
orders, so it cannot demonstrate completion either way); the only completion-adjacent evidence is a
handful of scenario files and closure documents (Section 2), and turning that into an exact count
would require auditing every scenario and closure by hand for whether it actually asserts a mission
*verdict* versus merely a state crossing. This report gives named examples (10, 20, 30, 40) rather
than a count, per the model's own rule 7 (prefer named blockers over a percentage), but a reader
wanting "N of 28" cannot get it from this repository today. A `missions completable` counter
alongside `check-milestone.sh`'s script-node counter would answer it.

**6. Whether an open world-map edge case (`DIV-129`, `DIV-131`) counts as blocking a "required"
transition.** Wording: *"required town/world-map transitions working."* Looked for: whether the
off-node/disconnected-endpoint fallback and the tie-break rule for a non-unique shortest route are
part of "required" or a fidelity refinement on an already-working mechanism. Not found: both are
edge cases of an otherwise-closed mechanism (`DIV-104`, `DIV-135` CLOSED), and whether any shipped
mission's own `MapPoint` graph actually exercises the disconnected or tied case is not established
in this repository (`DIV-131`'s own text records a 101-vs-110 disagreement between two instruments
over which `MapPoint` pairs differ, itself unresolved). A per-mission audit of whether the
disconnected/tied case is reachable on shipped content would answer it.

**7. No adopted definition exists for axis 4 at all.** Wording: *"Use the curated
integration-fidelity journeys if that proposal is adopted."* Looked for: the proposal's landing
status. Found: it is not landed (`check-pin-forward.sh` `FAIL` on its branch). The model gives no
instruction for what to report when the referenced sub-proposal is unadopted; this report
substituted an informal scenario-to-journey correspondence (Section 5) that is this report's own
construction, not a repository fact. Landing the harness proposal (or the model stating what
"nearest existing evidence" should look like) would answer it.

**8. No instrument counts "provisional" claims on the citing side.** Wording: *"research claims that
are provisional or retracted and still affect live expectations."* Looked for: a script or index
giving each cited claim id's confidence tier. Found only the retraction half
(`check-div-claims.sh`, 34 of 121 rows). A claim's confidence (High/Medium/provisional) is read
per-id via `go run ./tools/claim <ID>`; doing this for all 175 distinct ids cited by live rows was
not attempted in this pass. A script cross-referencing `DIVERGENCES.md` citations against each
claim's confidence field would answer it.

**9. `HOTFIX`-typed rows have no slot in the model's five-bucket list.** Wording: axis 3 asks for
"open known fidelity debt; accepted deliberate deviations; authored behaviour where research is
silent; closed divergences; provisional/retracted claims." `DIVERGENCES.md`'s own `HOTFIX` type
("temporary implementation; the behavioural divergence, if any, lives here") is none of the five
cleanly: it is not "correct behaviour known, implemented otherwise" (that is `FIDELITY-DEBT`'s own
wording) so much as "implementation known to be provisional regardless of what research says." This
report folded the 3 `OPEN` `HOTFIX` rows into the open-debt count and named them separately. A sixth
bucket, or an instruction to fold `HOTFIX` into `FIDELITY-DEBT`, would answer it.

**10. `DEVIATION`/`OPEN` rows (11) are neither accepted nor absent.** Wording: the model's bucket
list assumes a deviation is "accepted" or not mentioned. `DIVERGENCES.md` carries 11 rows typed
`DEVIATION` with status `OPEN` — a deliberate difference from ROM1 that has not yet been owner-ruled
either way. This report reported them as a separate line rather than counting them with the 15
`ACCEPTED` deviations. Whether an `OPEN` `DEVIATION` should read as "provisionally accepted" or
"undecided debt" for axis-3 reporting purposes is not settled by the model.
