# A mission SAV reopens on its own battlefield, not in the city

A player who SAVEs while a mission is running and picks SAV used to get a
detached city file: current map, orders, active effects, and RNG state all
discarded, the file reopening in the town he last visited instead of the
battlefield he saved from. That was the save dialog's only SAV producer for a
mission (`playerCitySave1173`, story1173), because the source-backed
current-battlefield producer story1170 built (`ExportCurrentWorldSave`,
`pkg/game/worldsave1170.go:28`) was wired only into the quick-SAVE seam
(`SaveSeams`, `resume.go:547`), never into the player's own save dialog
(`SaveDialogSeams.Prepare`, `savedialog1173.go`). This is milestone M4 of
`pipeline/SAV-ENDGAME.md`.

`playerMissionSave1198` (`pkg/game/missionsave1198.go`) is now the dialog's
own SAV producer for `ui.SaveSAV` and `ui.SaveBoth`
(`savedialog1173.go:121`). A save taken off the map (`s.Mission == 0`) is
unchanged: `playerCitySave1173` directly. A save taken from a running mission
tries `ExportCurrentWorldSave` first — the same producer ordinary SAVE
already uses — and only falls back to `playerCitySave1173`'s detached city
when that producer refuses with a named `*worldSaveUnsupportedError`
(never-imported world, mid-turn facing, pending native cast, unbound
container slot, and the writer's other named refusals). Any other error
(malformed input, a genuine publication failure) is not a fallback trigger
and propagates directly, exactly as `ExportCurrentWorldSave`'s own doc
comment requires (checked with `errors.As`, not a type switch, so a wrapped
refusal is still recognized).

## Disclosure

The player is told which of the two he received through the existing
story1194 mechanism, not a new control: `missionBattlefieldFallbackNotice1198`
composes one sentence naming the battlefield producer's own refusal reason
and returns it as `ui.PreparedSave.Notice`, which `commitSaveDialog`
(`pkg/ui/save_dialog.go`) already appends to the dialog's acknowledgement
message. It fires whenever the fallback runs, independent of whether the
detached city itself also approximated something (DIV-1319..1322), because a
file that reopens in the wrong location is a larger surprise than any
field-level approximation story1194 disclosed. `citySaveNotice1194` always
returns `""` for a mission-path export today (`world != nil`, out of that
story's scope by its own doc comment); `missionBattlefieldFallbackNotice1198`
folds that value in rather than discarding it, so it keeps disclosing both if
that ever stops being empty. Ledger row: DIV-1334, ACCEPTED.

## The pre-town mission20 fix

`citySnapshot1173` (`savedialog1173.go:148`) validates a settled town
(`validateCampaignLocation(..., mission=0, ...)`) on every detached-city
construction, even when the actual request is for an active mission — it
builds as if constructing a TOWN save regardless of caller intent. Missions
10 and 20 have no city (owner ruling, `pipeline/SAV-ENDGAME.md`'s "two save points"
section), so this refused every SAV from those two missions outright before
this story, no matter the mission state. `ExportCurrentWorldSave` has no
town dependency, so a mission20 SAV now succeeds like any other running
mission's, through the same dialog seam. This is a fix, not a new feature:
the SAV-ENDGAME.md ruling is about the CITY save point's existence, not about
whether a mission save point can succeed while a no-city mission is running.
DIV-1230's row is amended in place (story1194's own established
correction-append convention) to record this; it stays OPEN because the
fallback itself is not closed.

## Authority

No ROM1 behaviour question is open here. The player-result requirement, the
disclosure obligation, and the fallback trigger are all owner direction
(`pipeline/SAV-ENDGAME.md` M4) or this project's own prior stories (1170's
producer, 1173's dialog, 1194's notice mechanism), not a claim about what the
original SAV writer does. No promoted claim addresses this project's own
dual-producer fallback or disclosure sentence — DIV-1334's own "ROM1
behaviour" column states this directly: no claim lookup was performed for
this story because none of its decisions turn on one. The BOTH-on-mission
refusal (below) is a UI-reachability fact about this codebase
(`pkg/ui/save_dialog_draw.go`, `save_dialog_headless.go`), also not a claim
question.

## BOTH-on-mission: left refused

`SaveDialogSeams.Prepare` already refused `ui.SaveBoth` on the map
(`request.OnMap && request.Format == ui.SaveBoth`) before this story. That
refusal stays. A mission SAV now usually writes the same battlefield AGS
does, so the two formats would usually agree — but BOTH has no rendered
control on the map screen at all today, so lifting the refusal here alone
could not be reached by any player; offering it would need a new button,
map-specific wording and layout, which is a separate UI change, not this
seam's wiring. Comment left at the refusal site pointing here
(`savedialog1173.go:90`).

## Touched surfaces

- `pkg/game/missionsave1198.go` (new) — `playerMissionSave1198`,
  `missionBattlefieldFallbackNotice1198`.
- `pkg/game/savedialog1173.go` — SAV/BOTH payload branch now calls
  `playerMissionSave1198` instead of `playerCitySave1173` directly; comment
  added above the BOTH-on-map refusal.
- `pkg/game/citysavenotice1194.go` — doc comment only, noting who reaches
  this function's always-empty mission-path result now and why.
- `pkg/game/savedialog1173_correction_test.go` —
  `saveDialogPreTownBattlefield1198` (renamed and rewritten from
  `saveDialogPreTownBattlefield1173`'s prior refusal assertion) asserts the
  new success path end to end: dialog SAV succeeds, no disclosure (nothing
  was approximated), the written file decodes with `doc.World != nil`, cold
  LOAD reopens the mission with the same active effect, and the next-action
  proof below.
- `pkg/game/savedialog1173_release_test.go` — `TestReleaseSaveDialog1173MissionCity`
  renamed subtest `pre_town_refuses` to `pre_town_battlefield`;
  `saveDialogMission1173` (shared by the `native`, `imported_city_loot` and
  `direct_original_mission` subtests) now asserts the DIV-1334 disclosure text
  on the explicit mission SAV it already took, since all four of that
  helper's call sites reach a state `ExportCurrentWorldSave` refuses.
- `docs/DIVERGENCES.md` — DIV-1230 amended in place (correction paragraph);
  DIV-1334 added.

## Proof

The dialog-path "next action" proof
(`saveDialogPreTownBattlefield1198`, following `cmd/saveconvert/world1170_test.go`'s
established shape, reproduced here for the seam a player actually uses, not
only for `ExportCurrentWorldSave` directly): mission20 opened from
`2026-08-02/game0007.sav`, saved through the dialog as SAV, the written file
cold-loaded into an independent `FrontEnd`/session, both sessions advanced
one tick, and compared by `SourceBinding.RuntimeID`-matched entity HP and
position (`world1170Entity`/`world1170Position`, not raw World bytes —
`sav.RemintDocumentKeys` deliberately reassigns internal keys across
sessions) plus `ScriptRegisters()` equality. Both matched. A trailing
explicit AGS checkpoint, taken after that next-action step, matches the then-
current live World exactly.

"Pin it" (the fallback path must still produce byte-identical output to
before this story): `TestReleaseSaveDialog1173MissionCity/direct_original_mission`
run with `savedialog1173.go`'s SAV branch temporarily reverted to call
`playerCitySave1173` directly (this story's pre-existing behavior) produced
SHA256 `eee39635cbed814818b3a635df67c408cd539fbedfa3c1c7f47ae67ed39eb20a`;
re-run with `playerMissionSave1198` wired in (the fallback path, since that
scenario's state is one `ExportCurrentWorldSave` refuses) produced the
identical hash. `git stash` was not used (forbidden while lanes are open);
the revert and restore were direct file edits.

Gates, this worktree (branch `wt-story-1198-mission-savepoint`, base `cc88655`):
- `gofmt -l .` — clean.
- `go test -trimpath -count=1 ./...` — all packages pass, no assets needed.
- `scripts/check-no-game-assets.sh` — clean.
- `pipeline/check-div-claims.sh` — exit 0.
- One `pipeline/check-release-tests.sh` invocation over EN and RU together:
  `check-release-tests: 9 package(s), 312 gated test(s), 2 root(s)`;
  `root .../gameversions/en: ok (312 of 312 ran, 0 lacked a subject)`;
  `root .../gameversions/ru: ok (312 of 312 ran, 0 lacked a subject)`;
  exit 0. RU comparisons use `f.Words.SaveAcknowledgement` (the loaded
  install's own text), never `ui.AuthoredWords()`'s hardcoded EN default.

Milestone census (`pipeline/milestone-baseline.txt`, the canonical gate
table's "simulation timing, pathing, or scripts" row — not required here,
since this story touches no script/simulation execution code, but run anyway
as this lane's own "run the game" check): `go build -o /tmp/mr1198.exe
./cmd/missionrun`, then `-mission 10 -trace -ticks 1` and `-mission 20
-trace -ticks 1`, grep-counted for `UNSUPPORTED` lines, on both roots.

| root | mission | UNSUPPORTED (measured) | baseline |
|---|---|---|---|
| en | 10 | 0 | 0 (no "cannot run" row for en/m10) |
| en | 20 | 0 | 0 (no "cannot run" row for en/m20) |
| ru | 10 | 0 | 0 (no "cannot run" row for ru/m10) |
| ru | 20 | 0 | 0 (no "cannot run" row for ru/m20) |

Unchanged on both roots, both missions. This story does not touch script
compilation or execution, so no change was expected; the baseline file
(`pipeline/milestone-baseline.txt`) records `script N checks, M instants, K
triggers` for `en/m10`, `en/m20`, `ru/m10`, `ru/m20` with no accompanying
"cannot run" line for any of the four, confirming zero unsupported nodes
before this story and after it alike. This is a real claim, checked against
the actual baseline file content, not assumed from the diff's shape.

## M4's owner-run half (not performed by this lane)

This lane cannot drive the original game. The owner-run half of M4's proof —
does the shipped EN/RU game itself accept and correctly resume a
dialog-written mission SAV — is unperformed. Request, in the vocabulary of
`pipeline/SAV-OWNER-RUNS.md`'s acceptance levels:

1. Start any campaign mission already reachable in the shipped game (no
   slot-range or candidate-kit selection needed — unlike the city-SAV ladder
   in SAV-OWNER-RUNS.md, this is one save point, not a corpus walk). Take one
   action (move, or let a turn pass) so the battlefield differs from the
   mission's opening state.
2. Open Against's own SAVE dialog, choose SAV, confirm.
3. Read the acknowledgement message. Level A: it says only "Your character is
   saved" (or the RU equivalent) with no DIV-1334 sentence — the battlefield
   producer accepted this state outright. Level B: it names a reason and
   says the file reopens in the city instead — the fallback ran; note the
   reason text, since a reason clustering on one specific state, not spread
   across many, would point at the next producer to build for M6.
4. Load that exact `.sav` file in the ORIGINAL, unmodified game (not
   Againrom). Level C: it opens on the same battlefield, with the party
   where it was left, and one further order can be given and re-saved.
   Level D (fallback case only): it opens in the city with current gold and
   surviving party, as the acknowledgement said it would.
5. Report which level was reached and, for level B, the exact reason text
   shown. No save name, no install byte, no screenshot enters any tracked
   file — describe what was seen in chat only.

## Correction pass

The sole adversarial pass returned this story on one finding, and this is its
one correction pass.

**The finding.** `playerMissionSave1198` called `ExportCurrentWorldSave` before
anything reached `playerCitySave1173`'s `0x20..0x7e` label rule, and that
producer assigns the label straight into the document and has no rule of its
own. A player who typed a non-ASCII save name inside a running mission
therefore got a file whose slot name neither this build nor the original can
display, where the same name used to be refused — and the two branches of one
dialog disagreed with each other: same name, same running mission, battlefield
branch wrote the broken label and the fallback branch still refused. This
dialog is the first caller ever to hand that producer a player-typed label, so
the gap was not reachable before this story.

**The fix.** The rule is factored into one function, `savLabelRefusal1173`
(`pkg/game/savedialog1173.go`), called from `playerCitySave1173` and from the
top of `playerMissionSave1198` before any producer runs. It is deliberately one
function and not a second copy of the loop: a separately allocated story owns
replacing this refusal with a real encoding into the install's own single-byte
code page, and it must have exactly one place to replace. Writing UTF-8 into
the original's label region is not that encoding, so the refusal stays for now.

`TestSaveLabel1198RefusalIsOnePlace` covers the rule itself in both directions,
including that it does not over-refuse a writable name.
`TestReleaseSaveLabel1198MissionBranchRefusesBeforeProducing` is the
discriminating half: it runs on a mission whose battlefield producer succeeds,
so the fallback that carries the rule is never reached, and it first proves
with an ASCII name that the branch does produce a file. Removing the call at
the top of `playerMissionSave1198` makes that subtest produce 44,194 bytes
instead of a refusal, measured.

**What registering the new release test cost.** Two gates rejected it before
it ran, and both rejections were correct. `check-release-tests.sh` refused with
"unregistered gated test" because
`internal/gatedtests/testdata/population.txt` did not list it; the line was
added. `TestScanMatchesTheCheckedInPopulationList` then failed the other way:
the manifest listed a test the source scan could not see. `gatedtests.Scan` is
a lexical scan and its package comment says it does not resolve function
values, so `t.Run("name", helperName)` hides a gated test from the scan while
the asset-free runtime census still finds it. The subtests are handed to
`t.Run` as literal closures containing a bare-name call instead, and the scan,
the census and the population now agree. A release test written in the other
form is invisible to one of the three instruments that are supposed to
corroborate each other.

**Notes folded in.** DIV-1230 said the detached city's bytes were pinned by
direct comparison and not merely logged; they were logged, and the comparison
is made now. The shared release helper puts the snapshot the dialog saved from,
and the same label, through the untouched `playerCitySave1173` and requires the
dialog's own file to equal the result byte for byte. It is evaluated per
scenario and per install root rather than frozen as a digest: a first attempt
did freeze one, and the release gate failed it on both roots, because the
helper's four call sites build four different cities and the `native` one is
built from the install's own data and is not the same file on EN and RU. The
comparison excludes a change to this fallback's routing or to the inputs it is
handed; it does not exclude a change inside `playerCitySave1173`, which is
story1173's producer and has its own town-pair witness. It discriminates at one
byte: flipping a single byte of the fallback's own output inside
`playerMissionSave1198` fails all three gated scenarios and names the offset,
measured. The comparison is made at the point the file is read, before the
helper's own continuation ticks advance the live world; taken after them it
reports six differing bytes that belong to those ticks and not to the save. The DIV-1334 disclosure
was asserted only by its ledger id, which survived inverting the sentence; its
own words are asserted now. The `s.Mission == 0` short-circuit was unpinned,
and a town save carrying the battlefield-fallback notice it did not earn is now
a failure. DIV-1334's ROM1 column asserted a universal negative no search
supports; it carries the affirmative form from `SAV-SHAPE-023` and
`SAV-CITY-030` instead. And "it now SAVs like any other running mission" read
wider than it is: a native campaign's mission opened from the town carries
`Application1170.LocalOnly1186 = true` and still falls back, which is the
trigger that actually fires in two of the three gated fallback scenarios.

Two notes are deferred rather than done here: broadening the fallback trigger
to every error is caught by nothing, which is scope rather than a defect; and
the DIV-1334 sentence is hardcoded English on the RU root, which is consistent
with story1194's own notices and is a question about every notice, not this one.

## Open debt

- DIV-1230 stays OPEN: the fallback is real and still runs for every state
  `ExportCurrentWorldSave` refuses (never-imported worlds, a current native
  turn, and the writer's other named refusals); each producer M6 adds
  shrinks how often the dialog still reaches the detached city.
- DIV-1334 is ACCEPTED: the disclosure mechanism itself has nothing further
  to build; it retires only if DIV-1230 itself closes.
- BOTH-on-mission stays refused on the map screen; lifting it is a separate
  UI-surface change (new button, layout), not this seam's wiring.
- The owner-run half of M4's proof (above) is unperformed. No claim from the
  original game's own acceptance of a dialog-written mission SAV should be
  made until that request is run.
- The label refusal is retained debt, not a result. `savLabelRefusal1173` is
  narrower than the original's own behaviour, and the separately allocated
  code-page story is what closes it. Until then a player who wants his own
  name on a save takes AGS.
- Broadening the fallback trigger from `*worldSaveUnsupportedError` to any
  error is caught by no test. Nothing does that today; it is an unpinned
  boundary, carried here rather than fixed in a correction pass.
- DIV-1335 through DIV-1338 (reserved for this story) are returned unused —
  only DIV-1334 was needed; DIV-1230 was amended in place rather than given
  a new row, following story1194's own established convention.
