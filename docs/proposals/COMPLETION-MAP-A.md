# Completion map A — againrom against the Project Completion Model proposal

## Basis

Implementation master: `744b2fc77ed20121025ca4c0f3f49dcdca9b7905` (`git -C implementation rev-parse HEAD`,
branch `master`, clean working tree). Research master: `d1d350db1d834ab03c923ba5e2055f3bd35ce984`
(`git -C research rev-parse HEAD`, branch `master`, clean). Submodule pin: `git -C implementation
submodule status research` prints ` d1d350db1d834ab03c923ba5e2055f3bd35ce984 research
(remotes/origin/HEAD)` with no leading character, so the pin matches the checkout exactly.

`docs/DOMAINS.md` names nine domains (confirmed, `## The nine domains`). `implementation/scenarios/`
holds 13 `*.json` scenario files plus `README.md` (confirmed, `ls`).

`PROJECT-COMPLETION-MODEL.md` and its sibling `INTEGRATION-FIDELITY-HARNESS.md` and
`AGENT-REVIEW-BRIEF.md` are not yet part of master. They exist on `origin/proposal/
integration-fidelity-harness`, an unmerged branch (commits `9be3c47`, `3457346`, `6ae2196`, dated
2026-08-21 00:29-00:30). `pipeline/check-pin-forward.sh` reports this branch's research pin, `18c89dd`,
as older than master's `d1d350d`; `git -C research merge-base --is-ancestor 18c89dd d1d350d` confirms
`18c89dd` is a plain ancestor, so the branch is simply behind, not diverged. This is the only open,
unmerged branch on `origin`; `git branch` shows no local lane branches.

## Method

Repository evidence only. Every count below names the command or file that printed it. Read-only:
no branch, worktree, or file under `implementation/`, `research/`, or `gameversions/` was created,
edited, or deleted.

Sanity of the base tree, all run against `744b2fc`: `go build ./...` clean; `go vet ./...` clean;
`go test -trimpath -count=1 ./...` — 39 packages `ok`, 0 `FAIL`, 20 packages report no test files, exit
0; `gofmt -l` over tracked and untracked `*.go` prints nothing; `bash scripts/check-no-game-assets.sh`
— clean; `bash pipeline/check-preserved-installs.sh` — ok, 162 files, both roots as recorded.

One invocation error is recorded because it changed a result. `bash pipeline/check-release-tests.sh
../gameversions/en` (relative root) reported all 34 install-gated tests FAILED. The script sets
`AGAINROM_ASSETS` to the literal string passed in and then runs `go test ./...`, which runs each
package's test binary from that package's own source directory, not the invoking shell's directory; a
relative root resolves against the wrong directory for every package but the one at repository root.
Re-run with an absolute root (`<seat>/gameversions/en`) passed: 34 of 34 install-gated
tests ran and passed, 0 skipped. The same absolute-root run against `.../gameversions/ru` also passed
34 of 34. This is an invocation defect in how this report first ran the script, not a build defect;
the corrected runs are what is cited below.

## 1. Release level

**R2 candidate — production paths exist for R1 and pieces of R2, but no instrument in the
repository asserts "every required campaign mission is naturally reachable and completable."**

| Level | Condition (model's wording) | Verified | Not verified |
|---|---|---|---|
| R0 | "Formats and engine layers can load meaningful original data and exercise core simulation/rendering paths." | Met. `pkg/formats/*`, `pkg/vfs`, `pkg/data` all `go test` clean (13 packages). `pkg/sim` clean (deterministic tick, hash, movement — `pkg/sim` is stdlib-only per `internal/archtest`). `check-preserved-installs.sh` confirms 162 real files read against both lawful roots. `check-scenarios.sh` opens real archives on both roots, 13/13. | — |
| R1 | "At least one production player journey crosses menu/gameplay and reaches a real mission outcome. Major systems may be absent." | Each half is separately witnessed: `scenarios/0163-chargen-mission10.json` crosses menu → mission picker → character generation → a live, controllable party inside mission 10 (`assert_state {"screen":"map"}`, then `assert_member` on skills). `cmd/missionrun`'s `TestTheTenthMissionRunsEndToEndOnALawfulInstall` (one of the 34 install-gated tests, passing on both roots) drives a mission to a recorded verdict (`lost` at tick 224, an unattended escort, by mission design, not asserted as pass/fail). | No single committed scenario chains menu → chargen → mission → decided verdict in one run. `0163-chargen-mission10.json` stops at "party is in the mission," and `missionrun`'s driven verdict does not go through chargen or the menu — it opens `sim.World` directly (`scenarios/README.md`: "`cmd/missionrun` remains a separate developer tool"). Whether R1 requires one continuous witness or accepts this composed evidence is model-ambiguous (guess 3, below). |
| R2 | "Every required campaign mission is naturally reachable and completable without developer-only bypasses. Required transitions between mission, town, world map and subsequent missions work. Save/load is usable for campaign play." | `scripts/campaign-sweep.sh` drives all 28 shipped campaign maps (`scenario.res` ships 28 on both roots) without a load failure — this is reachability of the script, not of a production journey, and the model explicitly warns against reading it as completion ("Never count a loaded map as a completed mission"). Production-flow transition evidence exists for specific boundaries: mission verdict → town → next mission (`scenarios/0163-mission-to-town.json`, purse/documents/roster asserted on both sides, tavern interaction, walk-out); world map single click → mission open (`scenarios/1013-world-map-one-click.json`); game menu abort → new game → mission entry (`scenarios/1020-abort-witness.json`); mid-mission trigger hand-over (`scenarios/0159-mission40-join.json`). | No instrument reports "N of 28 missions completable" through ordinary production play. `pipeline/check-milestone.sh` (run today, both roots) reports 59 script nodes this build cannot run, spread over 14 of the 28 maps (detail in section 2). `campaign-sweep.sh`'s own header states it "issues no orders, so the drive is unattended and most missions end `undecided`" — an unattended drive is not evidence of completability. Mission 30 has only a narrow mission-stage scenario (`scenarios/0156-mission30-cure.json`, an instant/item hand-over check at tick 20); no scenario or test in this tree drives mission 30 through a chargen'd hero to a decided verdict. Save/load is usable in the sense that all persistence-domain release tests pass (section 3, domain 9), but "usable for campaign play" at R2's scope (all 28 missions) is not established. |
| R3 | "All required player-facing systems have reached at least FUNCTIONAL... no known missing feature required to experience the shipped campaign as designed." | Not evaluated as met: system completeness (section 3) finds every domain FUNCTIONAL, none PARTIAL or ABSENT by direct evidence, but R3 also requires R2, which is not established. | — |
| R4 / R5 | Fidelity-debt closure, frozen-pin audit. | Not applicable — DIVERGENCES.md carries 28 open FIDELITY-DEBT rows and 11 open (non-accepted) DEVIATION rows (section 4); R4 requires these closed or owner-waived. | — |

The example progress-report line the model gives, `Release level: R2 candidate`, is the closest true
statement: R1's pieces exist and R2's constituent mechanisms (town, world map, save/load, most of the
script surface) work, but nothing in the repository asserts campaign-wide completability, and the
model's own rule against inferring completion from a loaded map applies directly to
`campaign-sweep.sh`'s 28/28.

## 2. Campaign reachability

Instrument: `pipeline/check-milestone.sh`, run today against both preserved roots (`gameversions/en`,
`gameversions/ru`), exit 0.

- **Missions in the shipped campaign: 28**, both roots (`scenario.res` ships 28; `check-milestone.sh`
  enumerates `m10` through `m151`).
- **Script nodes (checks + instants + triggers) on the EN root: 1837** (680 checks + 759 instants +
  398 triggers, summed from the per-mission `script N checks, M instants, K triggers` lines). RU
  differs by 2 checks on one mission (`m100`: 42 on EN, 40 on RU); the rest match.
- **Script nodes this build cannot run: 59, identical on both roots**, over **14 of the 28 maps**
  (`m40, m60, m71, m81, m90, m91, m100, m101, m111, m130, m131, m140, m150, m151`), in **21 distinct
  (mission, opcode) rows**: check op 17 (7 rows), op 4 (5), op 21 (4), op 16 (3), op 9 (2). Every
  INSTANT runs; the entire gap is in CHECK opcodes. This matches the count `PIPELINE-STATUS.md`
  records as of 2026-08-17 — the census has not moved since, which is consistent (no story since has
  targeted these five opcodes) but was independently re-measured today, not taken on the document's
  word.
- **Unattended mission-10 drive**: outcome `lost` at tick 224, 4 of 36 units moved, 1 fell — printed
  and explicitly not asserted (mission 10 is an escort; an unattended drive losing is the mission's own
  design, not a defect, per the script's header and the owner's 2026-08-11 ruling cited there).
- **Missions completable through ordinary production play, end to end**: no command in this repository
  answers this for the campaign as a whole. The available evidence is per-mission and partial: mission
  20 → town → mission 30 transition is witnessed by save-load plus notice-driven progression
  (`0163-mission-to-town.json`), not by a from-scratch chargen'd win; mission 40's trigger hand-over is
  witnessed (`0159-mission40-join.json`) but not its win/lose verdict; mission 10's unattended verdict
  is witnessed and is a scripted loss, not a win. No scenario or test witnesses a mission being **won**
  through ordinary player-shaped commands from a fresh character.
- **Required town/world-map transitions**: town arrival/departure, tavern interaction, and one-click
  world-map travel are each witnessed in production flow by name (scenarios above); world-map route,
  marker, and arrival mechanics also carry 5 CLOSED divergence rows (`DIV-104`, `DIV-106`, `DIV-128`,
  `DIV-130`, `DIV-135`) recording that they were brought to researched-fidelity levels at their own
  story landings.
- **Campaign terminal state**: not established either way by any command found. No scenario, test, or
  divergence row addresses whether the campaign has a reachable final/ending state distinct from its
  individual mission verdicts.

## 3. System completeness

Classified against the model's four labels (ABSENT / PARTIAL / FUNCTIONAL / FIDELITY), using
`docs/DOMAINS.md`'s nine domains. A domain is read as FIDELITY-blocked by any **OPEN** (non-accepted,
non-closed) `FIDELITY-DEBT`, `DEVIATION`, or `CONFLICT` row in its scope; an `OPEN UNKNOWN` row does
not block FIDELITY under the model's own text ("except accepted deviations/unknowns"). Row counts are
from `docs/DIVERGENCES.md`, both its tables, read together (see section 4 for why the tables are read
together rather than by their own headers).

| # | Domain | Verdict | Evidence | Named blockers to FIDELITY |
|---|---|---|---|---|
| 1 | Assets | FUNCTIONAL* | `pkg/formats/{alm,bmp,databin,itemname,pal,reg,res,sav,spr16,spr256,textinput}`, `pkg/vfs`, `pkg/data` — 13 packages, `go test` clean. `check-preserved-installs.sh` — 162 files verified against both roots. No divergence-ledger row is filed against this domain by name (`grep` over `docs/DIVERGENCES.md` for asset/archive/format-flavoured subsystem text: no match). | None found — but see guess 1: this domain has no player path of its own to classify against. |
| 2 | Sim Core | FUNCTIONAL | `pkg/sim` clean; digest reproducible across roots per `scenarios/README.md`. | `DIV-027` (random generator, DEVIATION/OPEN), `DIV-053`/`DIV-054`/`DIV-055` (stacked-effect clamp cost, movement occupancy anchor-only, area-effect/cell-slot collision — all FIDELITY-DEBT/OPEN). |
| 3 | Combat & Magic | FUNCTIONAL | Story closures `1001`-`1004` landed; `DIV-001`/`DIV-002` (point/area effects) CLOSED. Spellcasting, effects, and projectiles run in the scenario suite and the milestone drive. | `DIV-021` (area movement cost), `DIV-037` (Control Spirit template), `DIV-056` (weapon-borne release), `DIV-076` (projectile-driver mark writer), `DIV-077` (Meteor Storm presentation), `DIV-079` (retained overlay/projectile draw order) — all FIDELITY-DEBT/OPEN; `DIV-082` (burst timing, DEVIATION/OPEN); `DIV-103` (Teleport target visibility, HOTFIX/OPEN). |
| 4 | AI & Orders | FUNCTIONAL | Story `0167-escort-tick` landed (Follow/defend closing behaviour). `campaign-sweep.sh` runs live entity behaviour across all 28 maps without failure. | `DIV-007` (sim/escort residues, FIDELITY-DEBT/OPEN) — `PIPELINE-STATUS.md` names the escort defender's heal fork and idle turn as "named not built," which this report did not independently re-verify against current master. |
| 5 | Party, Items & Heroes | FUNCTIONAL | Story `1005-interactive-doll`, `0162-wear-rule` landed. `scenarios/1005-doll-and-shop.json` and `1005-doll-carry-over-worn.json` pass both roots — equip, unequip, ground-drop, real hero and a companion with a starting-weapon fallback. | `DIV-111` (starting weapon drawn but not equipped, DEVIATION/OPEN), `DIV-112` (WeaponMaterialized decode, FIDELITY-DEBT/OPEN). |
| 6 | Campaign & Scripts | FUNCTIONAL | `check-milestone.sh`: all INSTANTs run; `0163-mission-to-town.json`, `1013-world-map-one-click.json`, `1020-abort-witness.json`, `1019-session-end` closure (campaign/session aspect PASS) each witness a real cross-boundary production journey. | 59 unsupported CHECK nodes (section 2) is the domain's largest named debt. `DIV-011` (dialogue/announcement cadence, FIDELITY-DEBT/OPEN), `DIV-138` (world-map marker-selection cache is session state not persisted, FIDELITY-DEBT/OPEN). |
| 7 | Town & Economy | FUNCTIONAL | Stories `0157-shop`, `1011-shop-tip`, `1015-school-column`, `1016-town-square`, `1017-button-frames`, `1018-tips` landed. | The largest open-row concentration in the ledger: shop (`DIV-020` merchant sprite undrawn, FIDELITY-DEBT/OPEN, plus ~9 UNKNOWN rows), tavern (~4 UNKNOWN rows), school (`DIV-142`, `DIV-144`, `DIV-147`, `DIV-165`, all FIDELITY-DEBT/OPEN), town square (`DIV-149`, `DIV-150`, DEVIATION/OPEN; `DIV-153`, FIDELITY-DEBT/OPEN). |
| 8 | Client | FUNCTIONAL | Stories `1006-shared-screen-shell`, `1014-menu-accelerators`, `1017-button-frames` landed. | `DIV-099` (in-game menu omissions: Game/Sound Options and Quest Objectives visible-but-disabled, Diplomacy and confirmation panels entirely absent, FIDELITY-DEBT/OPEN) is the strongest single candidate anywhere in the ledger for PARTIAL rather than FUNCTIONAL — see guess 2. Also `DIV-043`, `DIV-140`/`DIV-141`, `DIV-167`, `DIV-168`/`DIV-169`/`DIV-170`. |
| 9 | Persistence | FUNCTIONAL | Story `1008-save-safety` landed; `DIV-097`/`DIV-098` (save publication, load transaction) CLOSED. All persistence-touching release tests are inside the 34/34 passing on both roots (section "Method"). `1019-session-end` closure: persistence/save-load aspect PASS. | `DIV-026` (original saves restore only a supported subtree — no script state, ground loot, sacks, corpses, trigger latches — FIDELITY-DEBT/OPEN), `DIV-041` (ghost template, DEVIATION/OPEN), `DIV-095` (`.ags` compatibility), `DIV-096` (current-install dependency), `DIV-100` (unreadable-save visibility) — all FIDELITY-DEBT/OPEN. Writing a save the *original* game can load has no live divergence row asserting it as in-scope debt or as accepted out-of-scope; see guess 8. |

`*` Assets is marked FUNCTIONAL for consistency of presentation; see guess 1 for why this domain does
not fit the model's player-path-based definitions cleanly.

**No domain classifies as FIDELITY.** Every domain carries at least one OPEN `FIDELITY-DEBT` or OPEN
non-accepted `DEVIATION`/`HOTFIX` row under the mapping above. **No domain classifies as ABSENT.**
**No domain was found to clearly cross into PARTIAL** by this report's reading, though domain 8
(Client, `DIV-099`) and domain 6 (Campaign & Scripts, the 59-node script gap) are the two closest
candidates and are flagged as guesses rather than resolved.

Progress-report line: **Systems: 0 FIDELITY / 9 FUNCTIONAL / 0 PARTIAL / 0 ABSENT** (this report's
classification; see guess 2 for where a stricter reading could move one or two of the nine to PARTIAL).

## 4. Fidelity reconciliation

`docs/DIVERGENCES.md` holds two live tables, "Divergences" (row types `CONFLICT`, `DEVIATION`,
`HOTFIX`, `FIDELITY-DEBT`) and "Authored where research is silent" (row type `UNKNOWN`), plus
`docs/DIVERGENCES-CLOSED.md` for rows whose status became `CLOSED`. Counted directly from the tables
(`awk` over the pipe-delimited columns, cross-checked by `check-div-claims.sh`'s own header count of
"121 live row(s)"):

- 121 live rows total: 44 in "Divergences," 77 in "Authored where research is silent."
- 13 rows in the "Authored where research is silent" section carry a type other than `UNKNOWN`
  (6 `DEVIATION`, 7 `FIDELITY-DEBT`) despite that section's own header text ("Type `UNKNOWN`"). This
  is a filing inconsistency in the document, not a miscount by this report — the type column is the
  row's own stated type and is what this report counted by.
- 22 rows are `CLOSED`, in `docs/DIVERGENCES-CLOSED.md`.

By type and status, both tables combined:

| Type | OPEN | ACCEPTED | CLOSED (separate file) |
|---|---|---|---|
| `FIDELITY-DEBT` | 28 | 0 | 16 |
| `DEVIATION` | 11 | 11 | 1 |
| `CONFLICT` | 0 | 4 | 1 |
| `HOTFIX` | 3 | 0 | 1 |
| `UNKNOWN` | 57 | 7 | 3 |
| **Total** | **99** | **22** | **22** |

Mapped to the model's five reconciliation categories:

- **Open known fidelity debt** (correct ROM1 behaviour known, not yet built): the 28 `FIDELITY-DEBT`
  rows at status `OPEN`. Largest concentrations: Town & Economy (school/shop/town-square, ~7 rows),
  Persistence (5 rows), Combat & Magic (6 rows), Sim Core (3 rows).
- **Accepted deliberate deviations**: the 11 `DEVIATION` rows at status `ACCEPTED` plus the 4
  `CONFLICT` rows (all `ACCEPTED`) — 15 total owner-ruled, no-revisit-planned differences from
  researched ROM1 behaviour. The 11 `DEVIATION` rows still at `OPEN` are deliberate differences the
  owner has not yet ruled permanent; the model's rule 4 ("never call an accepted deviation a defect")
  applies to the 15 `ACCEPTED` rows, not to these 11.
- **Authored behaviour where research is silent**: the 64 `UNKNOWN` rows (57 `OPEN` + 7 `ACCEPTED`).
  Per the model, none of these is automatically a defect; they are a research backlog, each with its
  own `Revisit condition` naming the claim that would retire it.
- **Closed divergences**: 22 rows in `DIVERGENCES-CLOSED.md` — 16 `FIDELITY-DEBT`, 3 `UNKNOWN`, 1 each
  `DEVIATION`, `CONFLICT`, `HOTFIX`. Each carries what was once divergent and what settled it; two
  (`DIV-145`, `DIV-146`) were closed by a later research experiment (`EXP-0195`) that partially
  retracted the claim the row had originally leaned on, a documented case of research overturning an
  earlier reading rather than the implementation changing.
- **Research claims provisional or retracted, still affecting live expectations**:
  `pipeline/check-div-claims.sh`, run today, reports **34 of the 121 live rows** cite a claim that
  carries a row in `research/claims/retracted.md` (which itself lists 229 ids in its own first
  column). The script's own header is explicit that a claim is usually retracted in part and most
  citing rows are unaffected; it flags candidates for a by-hand read, not confirmed defects. This
  report did not read all 34 rows' cited claims against their retraction text — that is a full
  worked instrument run outside this report's scope, and is itself a candidate forced guess (guess 5).

`check-div-claims.sh` cannot see the complementary failure mode — a row whose "research is silent"
cell has since been answered by a brand-new claim that cites no id, so nothing for the script to
match. That class is read by hand, area by area, and this report did not perform that hand read at
scale; it is reflected only in the 5 `DIV-1xx` rows already known-closed for exactly this reason
(`DIV-104`, `DIV-106`, `DIV-105`, `DIV-084`, `DIV-088`, all `CLOSED` per `DIVERGENCES-CLOSED.md`).

Progress-report line: **Fidelity debt: 28 OPEN FIDELITY-DEBT / 15 ACCEPTED deviations (11 DEVIATION +
4 CONFLICT) / 64 authored UNKNOWN rows (57 open, 7 accepted) / 34 rows citing a claim with a
retraction on record.**

## 5. Journey integrity

**The curated-journey portfolio does not exist on master.** It is proposed in
`docs/proposals/INTEGRATION-FIDELITY-HARNESS.md` on the unmerged `origin/proposal/
integration-fidelity-harness` branch (section "Basis," above), which defines six journeys (J1 new
game → first mission, J2 mission success → town → next mission, J3 combat → loot → equip → later
combat, J4 save/load across non-trivial state, J5 dialogue/trigger progression, J6 multi-mission
continuity) with three evidence levels (M mechanism, I ingestion, J journey). The proposal is
explicitly non-normative and not adopted; nothing in the current repository reports `passed /
applicable` against these six.

**Nearest existing evidence, and whether it stands in:**

- The 13 headless scenarios (`implementation/scenarios/`) are the raw material the proposal's own
  text says the curated portfolio should be built from ("a curated suite of existing headless
  scenarios plus a small number of new cross-system scenarios"), not a substitute for it. All 13 pass
  on both roots today (`check-scenarios.sh`, re-run twice for confirmation).
- Individually, several scenarios already approximate one proposed journey without being labelled or
  curated as such: `0163-chargen-mission10.json` approximates J1's front half (menu → chargen → live
  party in mission; does not reach a verdict). `0163-mission-to-town.json` approximates J2 (verdict →
  town → next mission, with roster/purse/document checkpoints on both sides). `1005-doll-and-shop.json`
  approximates J3's inventory/equipment half (does not show a later combat outcome changed by the
  equip). `1020-abort-witness.json` approximates part of J4 (abort/new-session boundary; not
  save/load). `0159-mission40-join.json` approximates J5's trigger half (no dialogue, no downstream
  consequence check). No scenario approximates J6 (repeated multi-mission continuity).
- `scripts/campaign-sweep.sh` and story closures' own "integration witness in a real campaign mission"
  sections (pipeline v2's own closure requirement) are the other candidate stand-in. `1019-session-end/
  closure.md` is a concrete, current example: a twelve-aspect matrix plus a named integration witness
  (`pkg/game/sessionend_test.go`, reusing `continuityFront`) reproducing the reported symptom before
  the fix and passing after it — this is real cross-system evidence, but it is per-story and not
  curated as a standing, re-run-every-landing suite the way the proposal describes.

None of this evidence is a substitute for the proposal's stated purpose — a small, long-lived,
cross-story set of witnesses specifically aimed at composition risk between otherwise-correct
subsystems. The existing evidence is real but scattered across 179 story directories and 13
uncurated scenario files; nothing currently re-runs a fixed, named journey list at every landing.

Progress-report line: **Journeys: portfolio not adopted; 0/6 named journeys formally passed, partial
coverage exists informally for J1, J2, J3, J4 and J5 across separate uncurated artifacts, none for J6.**

## 6. Forced guesses

Every item below is a place the model's own wording did not resolve to one reading from repository
evidence alone.

**1. Domain classification has no defined answer for a domain with no player path.**
Model wording: `"ABSENT — player path does not exist"` / `"FUNCTIONAL — required player path works
end to end"`. Domains 1 (Assets), 2 (Sim Core), and 9 (Persistence) are described in `docs/DOMAINS.md`
as substrate — "Nothing above it. Knows no game rule" (Assets); a determinism wall with "no IO" (Sim
Core) — with no player-observable outcome of their own, only in composition with other domains.
Evidence sought: a model clause distinguishing infrastructural from player-facing domains. Not found —
the model's four-way classification is written entirely in player-path terms. This report classified
Assets as FUNCTIONAL by analogy (ingestion works, tests pass), but a different, equally defensible
reading is that these three domains are outside the model's classification scheme entirely and should
carry no letter. What would answer it: an explicit model clause for infrastructure domains, or an
owner ruling that substrate domains inherit the classification of what they compose into.

**2. Whether DIV-099 (Client) and the 59-node script gap (Campaign & Scripts) cross from FUNCTIONAL
into PARTIAL.**
Model wording: `"PARTIAL — useful path exists but a known required behaviour is missing"` versus
`"FUNCTIONAL — required player path works end to end, fidelity gaps may remain."` `DIV-099` states
Diplomacy and both confirmation panels are entirely absent from the in-game menu, not merely
imperfect; the 59 unsupported script nodes leave 14 of 28 missions with CHECK opcodes that can never
evaluate true. Evidence sought: a model rule for what counts as "required" versus "a fidelity gap."
Not found — the model does not define "required behaviour" against the shipped campaign's own design,
and this report cannot determine from the divergence row or the census alone whether any specific
missing menu row or unresolved CHECK opcode blocks a mission's own designed win path, as opposed to
a side branch. What would answer it: per-row (or per-opcode) tracing of which missions' win/lose logic
actually depends on the missing behaviour — a instrumented run naming, for each of the 21 (mission,
opcode) rows, whether that opcode gates the mission's own recorded win condition.

**3. Whether R1 requires one continuous witness or accepts composed evidence.**
Model wording: `"At least one production player journey crosses menu/gameplay and reaches a real
mission outcome."` The word "journey" and "crosses" suggest one continuous run. The repository has
menu→chargen→mission entry in one scenario and mission→verdict in a separate tool invocation that
bypasses chargen, but no single committed artifact chains both. Evidence sought: a further scenario or
test combining `create_character` with a driven verdict in one file. Not found under `scenarios/`
(checked all 13 file bodies). What would answer it: either such a scenario, or an explicit model
statement that composed evidence from separate committed witnesses satisfies "one production player
journey."

**4. Whether "missions completable" (campaign reachability) means winnable, or merely "reaches a
recorded verdict of any kind."**
Model wording: `"missions completable"` and the stronger definition's clause 3, `"reach its
success/failure verdict through ordinary simulation and player commands."` Mission 10's only
production-flow verdict on record is a scripted loss from an unattended drive, explicitly not treated
as a pass/fail signal by the mission's own design. Evidence sought: a scenario or test in which a
mission is driven to a **won** verdict through ordinary player-shaped commands (not an unattended
drive, not a developer tool bypassing chargen). Not found for any of the 28 missions. What would
answer it: such a scenario for at least one mission, or an owner/model clarification that a recorded
loss-by-design counts toward "completable" the same way a win would.

**5. Whether a citing-a-retracted-claim hit (`check-div-claims.sh`'s 34) should be read into the
fidelity-debt count, or left as an unread flag.**
Model wording: `"research claims that are provisional or retracted and still affect live
expectations."` The script's own header states most of the 34 hits are not actually affected (a claim
is usually retracted only in part) and that judgement "stays with the reader." This report did not
read all 34 cited claims against their retraction text (`go run ./tools/claim <ID>` for each), which
would take this report well outside a single-session classification pass. Evidence sought: a
per-row determination of whether the retracted clause is the one each row leans on. Not performed.
What would answer it: the 34-row hand read the script's own header calls for, most likely as its own
short-lived pass rather than folded into this report.

**6. Whether the two proposal documents count as evidence for their own axis.**
Model wording (journey integrity): `"Use the curated integration-fidelity journeys if that proposal is
adopted."` The proposal is not adopted (unmerged, non-normative by its own header). This report
therefore did not use the J1-J6 table as adopted evidence, only as a naming scheme to describe how
close existing scenarios come. A stricter reading would exclude even that comparison, since the
proposal defines terms the current repository has no obligation to be measured against. What would
answer it: an owner ruling on the proposal (accept, reject, or return with a required revision), which
`docs/proposals/AGENT-REVIEW-BRIEF.md` states is still pending.

**7. Whether PIPELINE-STATUS.md's historical sections should be read as current claims.**
The document is a reverse-chronological journal: a "Now — 2026-08-21" section, then "## Before
that — 2026-08-17," then further "Before that" sections going back to 2026-08-09 and earlier, and a
final "## Open" table whose entries cite experiment ids from 2026-08-09 to 2026-08-14 with no visible
update since. Several numbers in the older sections (a 46/47 divergence-table split, a nine-item
FIDELITY GAPS / FUNCTIONAL / NOT STARTED subsystem table dated 2026-08-09) are stale relative to the
live ledger and current `docs/DOMAINS.md` framework, but the document does not mark them as
superseded — only their position below "Now" implies it. This report used only live files
(`docs/DIVERGENCES.md`, `git log`, running scripts) for every number in sections 1-5 and treated
`PIPELINE-STATUS.md` purely as a cross-check, per the task's own instruction, but a reader without that
instruction could easily cite the "## Open" table's items (e.g. "a save the original game can load...
not in a lane") as current status without checking whether a later story addressed them. This report
found no later story titled around writing an original-compatible save and treated `DIV-026` (which
covers reading, not writing, original saves) as the nearest live evidence, but could not confirm
one way or the other whether the specific claim in the old "Open" table is still true. What would
answer it: a `docs/<NNNN>/` story or a live `DIV-` row that names writing an original-loadable save,
confirming either that it remains unbuilt or that it landed.

**8. Whether "a save the original game can load" is in scope for R4/R5 fidelity or accepted
out-of-scope.**
Related to guess 7. `DIV-097` (closed) states outright: "The `.ags` writer is authored by againrom;
ROM1 behaviour is not asserted." No live divergence row asserts that producing an original-loadable
save file is either required debt or an accepted deviation; it simply has no row. Evidence sought: a
row of either disposition. Not found. What would answer it: an owner ruling placing this in or out of
the declared fidelity target scope, recorded as a `DIV-` row either way — its absence is itself an
example of the model's "declared target scope" needing to be explicit before R4/R5 can be evaluated
at all, since an unrecorded silent omission and an accepted one are indistinguishable from the ledger
alone.

**9. Whether the divergence ledger's own section-header/type mismatch (section 4, 13 rows) affects
any of this report's counts.**
The "Authored where research is silent" table's header states its type is `UNKNOWN`; 13 of its 77
rows carry `DEVIATION` or `FIDELITY-DEBT`. This report counted by each row's own `Type` column,
not by which of the two tables the row physically sits in, on the reasoning that a row's type field is
the more specific, more directly load-bearing fact and the section header is descriptive prose about
the section's *intended* contents. A different, equally defensible reading would flag these 13 rows as
themselves a filing defect requiring correction before they can be trusted at all. Evidence sought: a
document rule stating which is authoritative when a row's type and its section disagree. Not found.
What would answer it: a landing that either moves the 13 rows to the correctly-typed section or
amends the section header's own claim.
