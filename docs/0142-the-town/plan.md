# 0142 — plan

## Shape

Three tiers, and the seam between the last two is the constraint that shapes everything else.

| tier | file | holds |
|---|---|---|
| the data | `pkg/game/campaign.go` | the chapters the registry declares, per section |
| the model | `pkg/game/town.go` | the between-missions state and the rules over it |
| the wiring | `pkg/game/townscreen.go` | rooms, rows and the authored words |
| the screen | `pkg/ui/town.go`, `flow.go`, `app.go` | drawing rows and reading presses |

## Decisions

**DD-1 — the chapter is a query, not a field.** `Town.Chapter()` recomputes the lowest main mission
the campaign offers that is not yet won, every time it is asked. A stored field would need a writer
on every win, on every arrival and on every side-mission finish, and the one that got it wrong
would be invisible. FR-2's "recomputed, never assigned" is this and only this.

**DD-2 — `Campaign` grows a per-section record and keeps `Offered` untouched.** `Chapters map[int]
Chapter` carries the four arrays, the two shop prices and the payment of each `[Mission<n>]`
section — the four arrays FR-7 and FR-8 read, and the payment FR-9 pays.
`Offered` — the merged, deduplicated set the campaign already computed — is not derived
from it and does not move, so `NextMission`, `TownBegins` and `Offer.Town` answer exactly what they
answered before this story. The new field is read by the town and by nothing else.

**DD-3 — the town screen is ONE `Screen` value with rooms inside it, not five.** FR-5's four doors
and FR-8's conversation are five rooms; `ScreenTown` is
appended to the enum (P-3); which room is open lives in the model behind it. Five screen values
would put five step arms and five draw arms in `app.go` for what is one list drawn five ways, and
every one of them would have to re-derive the same escape rule. The cost is that `pkg/ui` cannot
tell the rooms apart — which is a feature: it cannot invent a rule about one.

**DD-4 — `pkg/ui` is handed an interface, not data.** `ui.TownScreen` is `Header() string`,
`Rows() []TownRow`, `Footer() []string`, `Choose(row int) TownAction`, `Back() bool`. Strings,
integers, booleans and a `MapOpener` — the same shape three doors already hand this package — so
P-2 holds by there being no type here that could name a simulation value. A struct of rows pushed
across instead would need a push on every mutation, and the mutations are exactly the two calls the
interface already has.

**DD-5 — the list is rebuilt on `Choose` and `Back` and at no other time.** Those two are the only
entry points that can mutate the model, so they are the only moments the rows can change. That is
what lets the flow hold a plain `*Picker` over the current room's rows — the map list's own model,
reused rather than copied, so scrolling, the selection marker, the hit test and the clip are one
implementation (FR-11). The selection resets to the first row on every rebuild; keeping a place
across a room change would be keeping a place in a different list.

**DD-6 — the town is entered through a fourth `NoticeDest`.** That is where FR-1's boundary lands.
`NoticeToTown` is appended (P-3) and
`advanceNotice` gains an arm that tears the map screen down through the same `leaveMap` the other
three use and then shows the town. The alternative — reusing `NoticeToMapList` and having the
wiring tier decide — would put the destination in two places.

**DD-7 — `continuity` decides it, and `FinishMission` records it.** `FinishMission` keeps its
signature and its two returns; it gains two statements, marking the mission won in the town and
opening the town when the offer it computed is a town offer. `continuity` then reads the town's own
state: a win with no successor and an open town answers `NoticeToTown`, and — new — a **loss** with
an open town answers `NoticeToTown` instead of `NoticeToMenu`. Before the town is open both arms
are byte-for-byte the behaviour they had, which is what FR-3's second sentence buys.

**DD-8 — the buildings append to one set.** `Town.take` is one method over
`offerRef{chapter, building, index}`; the tavern, the shop and the school differ only in which
array `take` reads and whether the entry is consumed. FR-6's "exactly one list" is that
`Town.Available()` has one backing map and three callers of one writer, and FR-8a's gates are its
only reader — so the tavern of FR-8 and the shelves of FR-7 append to the same place.

**DD-9 — the shop and the school offer the head and nothing else.** Their rows list every un-taken
element so the player can see what is there, and only the first is choosable — `REG-SCN-064`'s
"take element 0 on entry and remove it" with the taking made a press instead of an entry, because a
room that consumed an offer just for being entered cannot be looked at.

**DD-10 — the conversation is state on the wiring tier, not on the model.** `townScreen`'s own `npc`, `offer` and
`said` hold who is being spoken to and how far in; `Town` holds only what accepting DID. So the
model is decidable without a conversation and the conversation cannot leave the model half-written:
the only mutation is the accept row, which calls `take` once.

**DD-11 — the authored words live in `pkg/game`.** Every line the town says is composed at the
wiring tier. `pkg/ui` receives strings it does not compose and cannot compose, which is the same
rule the map list's own rows already follow.

**DD-12 — FR-10 and FR-4 meet in one struct.** What survives a mission is the `Town` and
`FrontEnd.Carried`, and what a save would write is exactly those. `Town.snapshot` does not exist;
`town.go`'s header names the four maps and the two integers that are the whole of the state, and
says that writing them is a different story. FR-10 is discharged by there being nothing to find.

## Risks

**R-1 — `InnNPC` and `InnMission` of unequal length.** The corpus has them equal in 22/22 sections;
a modded scenario need not. Pairing walks the shorter of the two, so a mismatch loses entries
rather than panicking.

**R-2 — `AdvanceLine` calls `FinishMission` on a mission that was merely started.** It already
writes the carry that way and this story adds the town write beside it. The headless report
therefore also marks the mission won in the town — consistent with what that method already does,
and it builds its own front end per run.

## Order

1. `Campaign.Chapters` and the payment key.
2. `Town` — state, chapter, take, available, gold.
3. `ui.TownScreen`, `ScreenTown`, `NoticeToTown`, the flow arm, the step and draw arms.
4. `townScreen` — the rooms and the words; `FrontEnd.Town`; `FinishMission` and `continuity`.
5. Tests, the build, `verification.md`.
