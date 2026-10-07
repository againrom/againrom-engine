# 0148 — plan

Four commits, one per functional requirement, in the order the state flows: the party's order is
decided first because FR-2's entity ids are indexed by party position.

## Step 1 — order the restored party participant-first (FR-1)

`pkg/game/originalparty.go`. `RestoreParty` walks `sf.Party()`; insert an ordering step between the
walk and the mint.

- `leadFirst(chars []sav.Character) ([]sav.Character, int)` returns the characters with the one
  whose `RuntimeID` is 1 moved to index 0 and every other character in file order, and the file
  position that character came from. It returns the input unchanged and `-1` when no character
  carries the id or more than one does.
- `RestoredParty` gains `LeadFrom int`, which is that position. Its zero value is a real answer
  (the leader was already first), so `RestoreParty` sets it explicitly on every path, including the
  fallback.
- `pkg/formats/sav/party.go`'s `Party` doc comment asserts "The first is the hero: the walk reads
  the groups in file order and each group's actor list in file order". The second clause is what the
  function does; the first is refuted by the corpus. Rewrite the comment to state the order the walk
  produces and that it is not participant-first, and name the measurement.

Tests, in `pkg/game`: a hand-built save whose second character carries runtime id 1 restores a party
led by that character; a save with no id-1 character keeps file order; a save with two keeps file
order.

## Step 2 — rebind a withdrawn placement to the party member (FR-2)

`pkg/mapload/saved.go`, `pkg/mapload/script.go`, `pkg/game/originalparty.go`,
`pkg/game/mission.go`, `cmd/almtool/script.go`, `cmd/missionrun/main.go`.

- `mapload.Saved` gains `MapUnitID uint16` — the map record's own identifier word the character
  claimed, zero for a character who claimed none. `restoredSaved` sets it from `c.MapUnitID`.
- `mapload.ScriptUnits(m *alm.Map, party []PartyMember) map[uint16]sim.EntityID`. After the walk
  over `m.Units`, it writes one entry per member `i` whose `Saved` names a non-zero unit:
  `out[Saved.MapUnitID] = PartyEntity(m, i)`. The party's entry overwrites a surviving record's
  (DD-3).
- Callers: `pkg/game/mission.go` passes `party`; `cmd/missionrun/main.go` passes `ms.Party`;
  `cmd/almtool/script.go` passes `nil`, which is the map alone and the answer it has always given.
- `WithdrawRestored`'s doc states a divergence that no longer holds. Rewrite it: the record is still
  withdrawn, and the script's binding follows the person.

`PartyEntity(m, i)` is `len(m.Units) + i` and the withdrawal has already shrunk `m.Units` when the
compile runs — the sequence in `pkg/game/originalsave.go` is restore, withdraw, then
`StartMissionFrom`. That ordering is what makes the ids agree, and the test below pins it rather
than the comment.

Tests, in `pkg/mapload`: `ScriptUnits` over a map plus a party whose member 1 carries a `Saved` with
a unit id resolves that id to `PartyEntity(m, 1)`; a nil party gives the map's own table exactly; a
member whose id collides with a surviving record wins. In `pkg/game`: a synthetic map whose script
holds a proximity check on a unit id a restored character claims does not fire its trigger, and the
same map with the record present and the character absent does (AC-1).

## Step 3 — a `Saved` does not cross a mission boundary (FR-3)

`pkg/mapload/carry.go`. `CarryParty` clears `out[i].Saved` for every member, before the liveness
loop, so a member whose entity did not survive is cleared too. Document it on the function: a
`Saved` describes the mission the save was taken in and the carry is the boundary.

Test, in `pkg/mapload`: a party whose members carry a `Saved` is carried out of a world and holds
none; started into a second map, its members stand at that map's drop cell and not at the saved one
(AC-4).

## Step 4 — the report says what happened (FR-4)

`pkg/game/originalsave.go`. `RestoredParty.String` gains the leader's file position and the rebound
count. `RestoredParty` gains `Rebound int`, written where the placements are collected: it is the
number of members whose `Saved` names a unit, which is what `ScriptUnits` will bind.

Test: the report over a restored party names both.

## Risks

**R-1 — The gob envelope.** `mapload.Saved` is reachable from `game.Snapshot`, which is
`encoding/gob`. Gob addresses fields by name, so an existing save decodes the new field as zero and
a new save adds it. No envelope version changes and `pkg/sim`'s `formatVersion` is not reached at
all — `Saved` never enters `pkg/sim`.

**R-2 — A fresh start must not move.** FR-2 adds table entries only for members carrying a `Saved`,
and a fresh party carries none, so the compiled script for a fresh start is unchanged. Checked by
the script-gap census over both roots before and after (AC-5).

**R-3 — The ordering changes which entity the hero band resolves to.** `StartMissionFrom` binds the
hero-band reference to `PartyEntity(m, 0)`. Reordering the party therefore changes that binding for
a resumed mission — from the witch to the participant's own hero, on the corpus's mission-10 saves.
That is the intended consequence of FR-1 and not a side effect to guard against, and it is recorded
in `verification.md` rather than left to be noticed.
