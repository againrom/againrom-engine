# 0143 — plan

## Shape — and which requirement lands where

| File | What lands there |
|---|---|
| `pkg/game/save.go` | FR-1's envelope, FR-2's refusal set, FR-4's campaign half, `Snapshot` itself (FR-9) |
| `pkg/game/savestore.go` | FR-8: the directory beside the binary, list, read, write |
| `pkg/game/resume.go` | FR-3's two shapes, FR-5's residue, and FR-10's three seams |
| `pkg/game/world.go` | `residue()` / `applyResidue()` on `mapWorld` (FR-5) |
| `pkg/game/frontend.go` | the live driver is remembered so a save can reach it (FR-3) |
| `pkg/ui/save.go` | FR-6's mini-menu, FR-7's load window, FR-10's seam types |
| `pkg/ui/flow.go` | two appended `Screen` values and the fields behind them (FR-6, FR-7) |
| `pkg/ui/app.go` | four arms: step and draw for each of the two screens, and FR-7's `L` |
| `cmd/againrom/main.go` | FR-8's `-saves`, and the wiring |
| `cmd/savecheck/main.go` | the developer tool that drives FR-3's loop against a real install |

## D-1 — the payload is `encoding/gob`, inside our own envelope

The campaign half carries `[]mapload.PartyMember`, which reaches `data.Profile`, `data.Hero`,
`data.Weapon` and `mapload.Carry` — five nested structs, every field exported and plain. A
hand-written field-by-field codec for them is about 150 lines that buy nothing the envelope's own
version byte does not already buy, and every field added to `PartyMember` later would need a
matching codec edit that nothing would fail on.

The envelope is ours and strict — magic, version, length, CRC — so the payload never has to defend
itself against a foreign file; it only has to round-trip our own value. `gob` does that and names
its own fields, so an added field is a decode failure at worst and never a silent misread.

The **world half is not in the gob graph as a `sim.World`**: it is `MarshalBinary`'s `[]byte`,
carried opaque. That keeps `pkg/sim`'s version discipline exactly where it already is — the sim
refuses its own stale versions with its own message, and this envelope neither restates nor
overrides that.

## D-2 — restore replays the opener and then substitutes the world

`MissionOpenerWith(n, party)` already builds the whole map screen: it decodes the map, calls
`StartMissionFrom`, builds the viewer, hands over the font, the pointer, the sack sheet, the audio
and the start view, and calls `openMission`. Restoring re-runs exactly that and then, between
`StartMissionFrom` and `openMission`, replaces `ms.World` with the unmarshalled one.

The alternative — a second construction path that builds a map screen from a saved world directly —
was rejected. It would be a second copy of a fourteen-statement handoff whose own comments already
name two "sharp call sites" where forgetting a line leaves every automated test green and the game
silently broken.

Entity ids line up because `StartMissionFrom` assigns them deterministically from the map and the
party, and the party rides in the snapshot.

**The map is re-decoded from the install and never carried.** A save is a few kilobytes and the map
is the install's; carrying one would put game data in a file we write.

## D-3 — the residue is three maps, a counter and a byte plane

`residue()` copies `commanded`, `swing`, `phase`, `groupTag` and `fog.explored` out of the driver;
`applyResidue()` writes them back after `openMission` has built everything else. The fog plane is
refused unless its dimensions match the plane the reopened map built — a mismatch means the install
changed under the save, and overlaying it would paint exploration onto the wrong cells.

`applyResidue` runs **after** the constructor's own tick-0 push, so the restored exploration is what
the first frame draws.

## D-4 — `f.live`, and why saving needs it

Nothing above the map screen holds the driver today: `MissionOpenerWith` builds it inside a closure
and returns seven function values. Saving needs the world and the mission number, so the closure
records both on the front end as it opens, and `loadMap` records a driver with **no** mission number
— which is what makes FR-3's refusal of a picker map a property of one field rather than a rule
someone keeps.

It is a plain field and not a stack: exactly one map screen is open at a time, and every door that
opens one passes through one of those two functions.

## D-5 — two appended `Screen` values

`ScreenGameMenu` and `ScreenLoad` are APPENDED after `ScreenTown`, for the reason `ScreenChargen`
and `ScreenTown` were: a `Screen` is compared and switched on by value throughout `pkg/ui`, and a
shifted constant would silently retarget every one of those sites.

Both remember the screen they were armed from — `menuBack`, `loadBack` — rather than returning to a
constant. The mini-menu is opened from two screens and the load window from two screens, and a
constant would send the player somewhere he was not.

## D-6 — the mini-menu reuses `*Picker`

Its four rows are a `*Picker`, which is what the town screen already does with its own list and for
its reason: the scroll window, the selection marker, the hit test and the rune clip are one
implementation, so a drawn row is a clickable row by construction. Four rows never scroll, and that
costs nothing.

Its draw is `drawPicker`'s own constants for the same reason.

## D-7 — Escape on the map opens the menu instead of leaving

This is a contract change to a key the automated suite presses in twelve files. The alternative —
a second key for the mini-menu, leaving Escape as it was — was rejected: the four entries include
`EXIT`, so a menu that is not what Escape reaches is a menu with a redundant row and a player who
still leaves his mission by reflex.

Every test that pressed Escape to leave a map now presses Escape and chooses `EXIT`. The change is
mechanical and each site keeps asserting exactly what it asserted.

The town's Escape keeps its one-room-at-a-time unwind and gains the menu only where it used to
leave — so `Back()` still gets first refusal on the press.

## D-8 — the label is in the header

Listing reads a fixed-size prefix per file. Decoding a payload per row would make the load window's
cost the size of the saves rather than their number, and would make one corrupt file the reason the
list cannot be shown at all.

## D-9 — the store's directory

`os.Executable()`, then its directory, then `saves/`. `-saves` overrides outright. `os.Executable`
failing is reported, not defaulted: a default reached by failure is how a file lands somewhere
nobody looks.

The store is created lazily, on the first write, so a run that never saves creates no directory.

Read refuses a name that is not `filepath.Base` of itself — the list produces bare names, and a name
that is not one did not come from the list.

## D-10 — what tests witness, and how

Tests are synthetic and write to `t.TempDir()` (golden rule 2). The round trips are built over a
hand-made `sim.World` and a hand-made `Town`, not over an install.

The refusal set is witnessed one file per sentence: short, foreign magic, stale version, truncated
payload, flipped payload byte.

The per-field ruling is witnessed by a **census test** over `mapWorld`'s own fields, by reflection:
it fails when a field is added and the table has not been re-read. That is what keeps AC-5 a
property of the code rather than of somebody's memory.

## Risks

**R-1 — the version byte is this story's own namespace.** `saveVersion` is not `formatVersion`; the
sim's number rides inside the payload and moves on the sim's own schedule. Two numbers, two
meanings, and the test that names the envelope's one is named for its function and not for its
value, which is how the sim's own version test went stale three times.

**R-2 — a save taken mid-notice.** The mini-menu opens over a running mission whose driver may hold
an open notice. The notice is not restored (FR-5); the world's own script state is, so whatever
raised it is where it was. Accepted.

**R-3 — `gob` and a future `PartyMember` field.** An added field decodes as its zero value against
an older file only if the version byte was not bumped. Bump it; the refusal is the rule.
