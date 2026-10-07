# 0131 — the campaign advances: tasks

Reading key: `FR`/`AC`/`P` → `spec.md`; `DD`/`R`/`SC` → `plan.md`. In order:
T2 widens a type and must leave the tree green with nothing yet producing the
new value, and T3 is the first thing that produces it.

## T1 — the successor a section declares

**Kind:** implementation. **Boundary:** the mission set's own read; nothing that
consumes a campaign is touched.

**Files:** `pkg/game/campaign.go` MODIFY · `pkg/game/campaign_test.go` MODIFY.

**Covers:** FR-1, FR-2 · AC-1, AC-2, AC-3, P-4 · DD-1, DD-2 · SC-1, SC-2.

**Fences:** do not change `NextMission`, `Offer`, `TownBegins` or what they
answer. Do not touch `frontend.go`. Do not add a reader of the new value.

`campaign_test.go`'s existing `AutoGetMission` cross-check reaches its key by
hand and its comment says interpreting the name is not this tree's to do. Re-aim
it at the new read and replace that reason: `REG-SCN-063` supersedes it, and the
test's remaining worth is AC-3's — that the file's answer and ours are two
facts, agreeing here.

**Done when:** a parsed registry answers, per section, the successor it declares;
the four rows of FR-1's table are each witnessed by a case built from synthetic
bytes; `go test ./pkg/game/` is green.

## T2 — a notice that can open a map

**Kind:** implementation. **Boundary:** the seam's shape and the front-end's
handling of it. **Nothing produces the new destination when this lands** — every
implementation hands back nothing for the new value, so the running game is
unchanged and every existing test still asserts what it asserted.

**Files:** `pkg/ui/notice.go`, `pkg/ui/flow.go`, `pkg/ui/app.go` MODIFY ·
`pkg/ui/advance_test.go` ADD · `pkg/ui/notice_test.go`, `pkg/ui/halt_test.go`
MODIFY · in `pkg/game`, the mechanical half alone wherever the seam is named:
`world.go`, `frontend.go`, `world_test.go`, `frontend_test.go`,
`continuity_test.go` MODIFY.

**Covers:** FR-3, FR-8, FR-9, FR-10 · AC-9, AC-13, P-2, P-3, P-5, P-6 · DD-4,
DD-5, DD-6, DD-7 · R-2 · SC-3, SC-7, SC-8, SC-11.

**Fences:** decide nothing about campaigns, successors or carrying — no mission
number is read, compared or composed in `pkg/ui`. Do not change what an existing
destination does. Do not add a member to `ui.MapLoader`'s tuple.

**Done when:** the new destination drives the front-end onto the map screen the
opener returns, through the entry the picker's own choice uses, with that viewer
laid out before the frame ends; a failing opener and a missing one land on the
map list with the messages DD-5 assigns; the tests reach all three dismissing
inputs; `go test ./...` is green.

## T3 — the win that opens the next mission

**Kind:** implementation. **Boundary:** what a win decides. The seam it speaks
through already exists and no tier below `pkg/game` is touched.

**Files:** `pkg/game/frontend.go` MODIFY · `pkg/game/continuity_test.go` MODIFY.

**Covers:** FR-3, FR-4, FR-5, FR-6, FR-7 · AC-4, AC-5, AC-6, AC-7, AC-8, AC-12,
P-1 · DD-3 · R-1, R-3 · SC-4, SC-5, SC-6.

**Fences:** do not touch this file above `FinishMission` — another change is in
flight around the front end's construction. Do not add carrying code: the party
is already captured and already read at the door. Do not change the four
sentences a win produces today, only add the two the contract names.

**Done when:** a won mission that declares an openable successor answers with an
opener onto it and the party that finished; each of the six sentences is
produced by a case that reaches it; every ending carries the party; `go test
./pkg/game/` is green.

## T4 — saying it without a window

**Kind:** implementation. **Boundary:** a report. It decides nothing and changes
no behaviour the game has when it is not asked for one.

**Files:** `pkg/game/frontend.go` MODIFY · `cmd/againrom/main.go` MODIFY.

**Covers:** FR-11 · DD-8 · AC-10, AC-11.

**Fences:** stay below `FinishMission` in `frontend.go`. In `cmd/againrom` touch
only the arm that already reports a mission headlessly. Do not fabricate an
outcome, and do not add a flag — the mode that already asks about a mission is
where this belongs.

**Done when:** the check mode, asked about a mission, prints the advance line in
whichever of its two shapes applies, and prints it from a run that really
started both missions.

## Traceability

| Spec | Plan criterion | Task |
|---|---|---|
| FR-1, FR-2, AC-1, AC-2, AC-3, P-4 | SC-1, SC-2 | T1 |
| FR-3, FR-9, FR-10, AC-9, AC-13, P-3, P-5, P-6 | SC-3, SC-7, SC-8, SC-11 | T2 |
| FR-8, P-2 | SC-6 | T2, T3 |
| FR-4, FR-5, FR-6, FR-7, AC-4, AC-5, AC-6, AC-7, AC-8, AC-12, P-1 | SC-4, SC-5, SC-6 | T3 |
| FR-11, AC-10, AC-11 | SC-9 | T4 |
