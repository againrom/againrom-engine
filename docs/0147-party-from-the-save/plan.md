# 0147 — plan

Six packages, one contract. The walk is new; everything above it is a substitution at one call site
and one new pointer field.

## Shape

```
pkg/formats/sav   program.go  the serialization programmes and the recursive walk
                  party.go    Character and Piece: the walk's answer, typed
pkg/mapload       saved.go    Saved: the cell and the pools a file states
                  start.go    two arms that honour it
pkg/game          originalparty.go  Character -> PartyMember, and the withdrawal
                  originalsave.go   the two resume doors and the two sentences
cmd/savtool       party.go    the evidence tool for the walk
cmd/savecheck     main.go     the evidence tool for the restore, read off the world
cmd/missionrun    main.go     the resume drive
```

## Decisions

**DD-1 — the walk is a second programme table beside `class.go`'s, not an extension of it.**
`class.go`'s `Member` expresses a straight run of fixed-width fields and a `CString`, which is every
construct its record-oriented `decode`/`encode`/`set` needs, and its `encode` is what proves a
record was UNDERSTOOD rather than merely stepped. A counted list, an object reference, a presence
flag and an embedded object cannot be a `Member` at all. So `program.go` carries the compound
constructs as its own step ops and REUSES `class.go`'s `Member` slices wherever a class's fields are
a straight run — `tokenMembers`, `playerMembers`, `buildingMembers`, `effectMembers`. No field list
exists twice in the package.

Rejected: adding the compound kinds to `Kind`. `Class.decode` and `Class.encode` would then have to
refuse them at run time, and `Class.Fixed()`'s answer for `Building` — the one class this package
chains — would depend on a table that had grown constructs it cannot measure.

**DD-2 — the walk returns what it read alongside its error.** A walk that desynchronised at the
fourth actor read three, and those three plus the offset the fourth failed at are the only evidence
about where the programme is wrong. `Chain` already works this way and for the same reason.

**DD-3 — a group record is read into the enclosing `Player`'s record.** Groups are a direct inline
call with no class record (`SAV-OBJ-016`), so a group takes no index in the shared counter, and this
tree consumes nothing about a group. The actor lists of every group therefore accumulate under one
site name on the `Player`, in file order. A `Group` type would be a type nobody reads.

Consequence, and it is the reason the counts accumulate rather than assign: a site that appears
twice — one actor list per group — must sum. A walker that assigned would report the last group's
count as the whole.

**DD-4 — `Saved` is a new pointer field on `PartyMember` and not a field of `Carry`.** A `Carry` is
what THIS tree's simulation wrote at a mission boundary; a `Saved` is what ANOTHER program wrote
inside a mission. They arrive on different paths, either can be nil, and a member can hold both. The
pointer is for `Carry`'s own reason: the zero value has to mean "this member came from no save", and
a character standing at cell (0,0) with no mana is a different state from a member the drop cell
places.

**DD-5 — the pool override is all six values or none.** `restoredPools` returns the health pair, the
mana pair and the two periods together. Taking health from the file and its maximum from the fold
would produce a health bar that is a ratio of two numbers written by two different programmes: a
full bar reading as a third full. The two periods move with them although every record measured
carries the base constructor's own 100 and 50 — which is exactly what `data.UnitDefaults()` answers,
so this changes no world today. They move because the file states them.

**DD-6 — a piece is routed by the slot its own code names.** `data.EquipSlotFor` reads field B, and
field B is both the slot and the class at once. The file writes fourteen equipment references at
fourteen sites; a site-to-slot table here would be a second statement of what the code already
carries, kept true by hand. Measured over the owner's fourteen saves: 200 worn pieces, every one
with field B in 1..12; 71 `Weapon` records all at B=1; 18 `Shield` records all at B=2, which is
`pkg/data`'s own authored `shieldItemClass`; 25 `Item` records all at B=14, which is
`data.ItemClassCarried`.

**DD-7 — a claimed map placement is withdrawn, not left in.** The alternative was to restore only
the characters the map never placed, which would leave a five-character save opening with one party
member and four uncommandable duplicates of the other four. Withdrawing filters `alm.Map.Units` in
the one window where the map's units are still records — the same window `applyOriginalPositions`
writes in. The divergence is spec.md L-2.

**DD-8 — the party is resolved when the opener is BUILT, not inside the closure.** `NextParty`'s own
reasoning: an opener built before something changed and run after it must not silently change which
party it opens. It also has to be, because the party is an argument to `missionOpener` and the
prepare closure runs after that argument is bound.

**DD-9 — `ResumeOriginalSave` takes a `data.BodyList`.** The restored character's drawn body comes off
his worn set through `data.HeroAppearance`, which needs the shipped body list. The developer resume
had no way to reach one, and `cmd/missionrun` already holds `defs.Bodies` at the call site.

**DD-10 — the profile comes from the file's own Humans row.** The head's `+0x0c` is a row index into
the collection the class picks (`ITEM-DEF-002`), and `data.NewHumanDef` builds a definition from a
row. So the three inputs the derived-stat graph reads that are not a statistic, a skill or an item
come off the same row the original streamed the character from, rather than off this tree's own
default archetype. A row that does not resolve gives the zero profile, which is the state every
party member in this tree carried before 0119-chargen.

**DD-11 — the two evidence tools print the same values from two ends.** `savtool party` prints what
the file says; `savecheck load` prints what the world holds. Neither reads the other. A restored
health of 107/131 agreeing across the two is evidence that could have disagreed; a single tool
printing its own parse is not.

## Risks

**R-1 — the walk is new code against a published programme, and a wrong field width desynchronises
everything after it.** The extent test is the answer: `SAV-TOPLVL-052` states the two words that
follow the first record, so a walk that ends anywhere else read some field wrong. It is asserted
synthetically and measured on every file of the install.

**R-2 — the party substitution changes every resumed world's hash.** That is the point of the story;
no existing world moves, because a member with no `Saved` takes exactly the arm he took before and
the fresh-start path passes no `Saved` at all. AC-3 is the pin.

**R-3 — a landed test asserted the old contract.** `TestTheReportNamesWhatIsNotCarried` required the
report to name "health" and "inventories" as NOT carried. Both are now carried, so the test moves
with the contract rather than the contract being written around it, and the test's own comment says
which two words moved and why.

## Which decision carries which requirement

| Requirement | Where it lands | Decisions |
|---|---|---|
| FR-1 the party comes from the save | `game.RestoreParty`, called by `FrontEnd.RestoreOriginal` and `ResumeOriginalSave` | DD-1, DD-2, DD-3, DD-8, DD-9 |
| FR-2 the saved cell, and the withdrawal | `mapload.Saved.Cell` honoured in `StartMission`'s placement loop; `game.WithdrawRestored` | DD-4, DD-7 |
| FR-3 the decoded state | `sav.Character` and `sav.Piece`; `game.restoredMember`, `restoredLoadout`, `restoredDefinition`; `mapload.restoredPools` | DD-1, DD-5, DD-6, DD-10 |
| FR-4 the two statements | `game.OriginalSaveNote`; `RestoredParty.String` inside `OriginalSaveResume.String` | DD-11 |
| FR-5 the four hotfix rules | `docs/hotfix/LEDGER.md`'s four `Owes` cells; the rules themselves are already in the code the hotfixes landed | — |

## Success criteria

**SC-1** The owner loads one of his own saves in `builds/current/` and his characters arrive with
their positions, statistics, health and items.

**SC-2** The script-gap census (`pipeline/check-milestone.sh`) does not move: this story touches no
script arm.
