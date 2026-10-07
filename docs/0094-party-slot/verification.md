# Verification — 0094, the party's roster slot

Toolchain: the `go.mod` pin, `go test -count=1 -trimpath`. Both lawful installs are read for the
install-backed evidence; `10.alm` is byte-identical between them (67554 bytes,
md5 `2d983ccbf249c5336ebb7ccc41fc405c`), so the map readings cover both at once and the **drives are
run twice regardless**, because a drive resolves its units out of the install and not out of the map.

## The gate

```
go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go')   # all silent
go test -count=1 -trimpath ./...                                    # no FAIL
bash scripts/check-no-game-assets.sh   -> check-no-game-assets: clean (tree scan)
bash scripts/check-doc-budget.sh       -> every 0094 artifact ok, plan <= 1.2 x spec, tasks <= 1.2 x plan
```

## AC-1, AC-2 — the slot and the group word

`pkg/mapload` `TestAPartyMemberStandsOnThePlayersRosterSlot`, over 0, 1 and 35 placements against
parties of 1, 3 and 6. Every member carries `sim.SelfSlot` and group word 0, and every placement
still carries the 0 its record names — that second loop is the discriminator against a change that
stamped the slot on everything.

**Witnessed by reverting.** With `Owner: sim.SelfSlot` deleted from the composite literal:

```
--- FAIL: TestAPartyMemberStandsOnThePlayersRosterSlot
    party_test.go:89: placements=0 size=1: member 0 owns slot 0, want 1 — slot 0 is outside
        the relation matrix in both directions
```

## AC-3 — one definition of the player's slot

`grep -rn selfSlot --include=*.go .` returns nothing; `SelfSlot` is declared once, in
`pkg/sim/engage.go`, and is read by `aiGroup.stance`, by the party literal and by the front-end
push. No literal 1 stands for the player's slot anywhere.

## AC-4, AC-5, AC-6 — each direction of the matrix on its own

`pkg/sim` `TestTheMatrixDecidesEachDirectionOnItsOwn`. Four relations over the same adjacent pair,
so geometry decides neither answer:

| relation | faction acquires the player | the player acquires the faction |
|---|---|---|
| neither direction | no | no |
| faction -> player only | **yes** | no |
| player -> faction only | no | **yes** |
| mutual | **yes** | **yes** |

The two one-way rows are AC-6 and they are the reason the table is one test: a build with either
direction wired to the other passes the first and last rows unchanged.

## AC-7 — he stands his ground

`pkg/sim` `TestThePlayerStandsHisGroundAndDoesNotChase`. At `reach` the slot-1 entity acquires; at
`reach + 1` it acquires nothing and its cell is unchanged after the decision. The adjacent case is
the discriminator — without it, an implementation that had dropped the player out of the decision
again would pass.

## AC-7a / FR-6a — what the slot costs, measured

`pkg/sim` `TestAPlayerOrderIsDroppedWhenHisUnitAcquires`. One drive, three ways:

| drive | where the unit ends |
|---|---|
| ordered **once**, hostile adjacent | still in contact, holding a victim |
| ordered **every eight ticks** | the destination side of the map |
| the same, at **slot 0** | the same cell |

The third row is the measure. "The player still gets there" means nothing against an absolute; it
means something against the cell the identical drive reached while nobody could see him.

## AC-8, AC-8a — nothing else moved

`TestStartMissionWithNoPartyBuildsThePlainWorld` (already in the tree) hashes a start with no party
against `FromALMWith` over the same map and they agree, so a world with no party is the world it was.
The whole suite went green at T2 with no test edited for it, and that is itself the finding: every
frozen digest in the tree is of a world without a party, so nothing that should not have moved did.

`TestAStartedMissionFightsWithoutBeingTold` was one of the green ones and had **stopped
discriminating** — its hostile now sees two slot-1 candidates, the placement at Chebyshev 2 and the
party at 5, and passed on cost ordering alone. It now names the party and fails if the party leaves
the matrix.

## AC-9 — mission 10's own diplomacy, read from the file

`go run ./cmd/almtool roster 10.alm`, over the entry extracted from either root's `scenario.res`:

```
roster (type5): 5 record(s), 35 placed unit(s)
slot  name                              units  relation words (raw u16, k=0..15)
   1  "Self"                               0  [2 0 1 1 1 0 0 0 0 0 0 0 0 0 0 0]
   2  "Villagers"                         14  [0 2 1 1 1 0 0 0 0 0 0 0 0 0 0 0]
   3  "Rogues"                             2  [1 1 2 1 1 0 0 0 0 0 0 0 0 0 0 0]
   4  "Beasts"                            16  [1 1 1 2 1 0 0 0 0 0 0 0 0 0 0 0]
   5  "Nocturnal"                          3  [1 1 1 1 2 0 0 0 0 0 0 0 0 0 0 0]

  from   1   2   3   4   5   <- to
     1   2   0   1   1   1
     2   0   2   1   1   1
     3   1   1   2   1   1
     4   1   1   1   2   1
     5   1   1   1   1   2
```

`Rogues -> Villagers` is **1** and `Villagers -> Rogues` is **1**. Bit 0 set in both directions.

The two clubmen are placements 0 and 1: class id **10**, which `units.reg` names
`"Human ClubMan"`, owner **3 `Rogues`**. The witch is placement 2, the NPC arm, `scenario/npc.reg`
`npc51`, `Flags = "Human,Female,Mage,Face"`, owner **2 `Villagers`**, script unit id 21 — the unit
the milestone escorts. **So the map itself authors that fight, and nothing is owed to it.**

## AC-10, AC-11 — the byte form

`pkg/mapload` `TestAStartedWorldCarriesThePartysSlotThroughTheByteForm`: a world built by a start
round-trips byte-identically, hashes the same after the decode, and every party entity comes back on
`sim.SelfSlot` — asserted on the decoded entity, never on the version byte.

AC-11 was **already witnessed before this story**, by `pkg/sim`'s
`TestTwoWorldsDifferingOnlyInAnOwnerHashDifferently`, whose last clause reads *"a world of unowned
units hashes like one owned by roster slot 1"*. Nothing was added for it; re-asserting it would have
added no coverage.

**No `formatVersion` was taken.** The field set does not change — the owner slot has been encoded at
the entity's `+87` since long before this story — and the round trip above is what says so.

## AC-12, AC-13 — the readout

`cmd/almtool` `TestRosterVerbPrintsTheSlotsAndTheEffectiveMatrix` over a synthetic document built
byte by byte, whose roster spells its own diagonal as 1, carries a `0x0101` whose narrowing is
visible against its own raw column, and whose owner census lands in the wrong row if taken per record
index. `TestRosterVerbOnADocumentWithNoRoster` prints the header alone and returns nil;
`TestRosterVerbRejectsAnUndecodableStream` returns an error with nothing on standard output.

**Witnessed by reverting.** With the forced diagonal removed from `effectiveCell`, all three matrix
rows fail.

## AC-14 — the front end

`pkg/game` `TestTheFrontEndIsToldWhichSlotThePlayerHolds`. A fresh viewer holds 0; after the world is
installed it holds the same slot the started party carries. Asserted as an **equality between the
two**, because that is the property the numeral drift turns on.

**Witnessed by reverting.** With the push put back to `0`:

```
--- FAIL: TestTheFrontEndIsToldWhichSlotThePlayerHolds
    localowner_test.go:43: the front end was told the local participant holds slot 0 while the
        party stands on slot 1 — the numeral drift of the player's own units turns on these
        two agreeing
```

## SC-1 — the player is fought, on a lawful install, on both roots

`missionrun -mission 10 -ticks 400 -waypoint p0:25:55:1 -attack p0:u19`, which orders the hero to a
cell beside the two `Rogues` clubmen and then has him strike the nearer one. **Identical on EN and
RU**, and the second line is what attributes the first:

```
BEFORE (the party at slot 0)
  waypoint 1  p0 -> (25,55) r1 : reached (24,56), Chebyshev 1, after 171 ticks
  attack 1  p0 -> u19 : FELLED it after 32 ticks, victim at -8 hp, attacker facing N

AFTER (the party at slot 1)
  waypoint 1  p0 -> (25,55) r1 : STOPPED SHORT of (24,57), Chebyshev 2, after 400 ticks
  attack 1  p0 -> u19 : FELLED it after 1 ticks, victim at -17 hp, attacker facing SE
```

Before, the hero walks up unmolested and then needs **32 ticks** to reach `u19` — the clubman never
left its post. After, the hero never arrives, and the blow lands in **1 tick** because `u19` is
already standing next to him. **The clubman came to the player.** The stop is FR-6a in the same
line: the walk is dropped at every decision while a hostile is in reach.

## SC-2 — the milestone, before and after, both roots

`missionrun -trace -mission 10 -waypoint u21:56:21:3 -waypoint p0:66:16:3`. **Identical before and
after, and identical between the roots:**

```
tick 262  check 12 vip(u21) COUNTED A LOSS: its unit is not alive (-1 hp)
tick 271  REPORT won=0 lost=1 -> lost
waypoint 1  u21 -> (56,21) r3 : was STOPPED BY THE WORLD DECIDING, short of (43,46), Chebyshev 25, after 272 ticks
outcome lost at tick 272
```

**The milestone does not come back, and it was not made to.** No test expectation, threshold or
predicate was edited — the diff carries no change to `cmd/missionrun` or to
`TestTheTenthMissionIsDrivenToAWin`.

Why the slot does not reach it: the escort walks `u21` from (36,51) toward (56,21) and is acquired
at tick 103 by entity **13** — owner 4 `Beasts`, class 73 `Bee`, standing at (47,45) — and falls at
250. The party never moves in that drive; it stands at the drop cell (17,66), more than thirty cells
from that `Bee`, so putting it in the matrix changes nothing about what that group can see. The
hypothesis that `u21` dies *because the escort is invisible* is **not supported**: the escort is
nowhere near the killer.

The clubmen kill nobody in that drive. With nothing ordered at all, they acquire nothing in 400
ticks and the witch stands at full health — before the change and after it.

## SC-3 — the group word collides with nothing shipped

A sweep of every `.alm` in both installs, loose and inside every `.res`, compiling each map's script
and reading every `GiveGroup` instant: **EN 38 maps / 14 nodes / 0 naming group 0**, **RU 34 maps
(33 with a compiled script) / 14 nodes / 0 naming group 0**. That arm writes an owner to every entity
whose group word matches, and the party's is 0 — so it is the one shipped instant that could re-own
the party, and nothing aims it there.

## What the player now does, and does not do

- He **is attacked.** A map faction hostile to slot 1 acquires him on sight, inside its circle.
- He **strikes back at what is already beside him** and takes no step toward anything further:
  slot 1 selects Stand Ground, whose scorer refuses every candidate past reach.
- He **does not chase, does not retreat on his own, and has no retaliation arm** against a slot he
  is not hostile to.
- And **a single move order is dropped** on the next decision while a hostile stands in reach. A
  re-issued order moves him normally, to the same cell he reached outside the matrix. This is the
  one cost of the story and it is a disclosed divergence, not a repair declined: the original
  branches inside this arm on whether the unit belongs to a human participant (`AI-STAND-076`,
  `R0015`) and that branch is undecoded. Authoring one here would invent an answer in the
  single field the whole engagement layer indexes by.

## Limitations

- The frozen notice radius stays diverged (`0086` DD-3, `AI-RADFREEZE-075`): our circle is recomputed
  every tick, so a group led away by the player carries its circle with it and can reach a target the
  original would never have noticed. Measured on mission 10: a clubman pulled from (32,62) to (31,53)
  by chasing the player then acquires the witch. **This story neither causes nor fixes it** — it is
  what makes the divergence reachable from the player's side for the first time.
- `AI-STAND-076`'s name is Medium and its human-participant branch is unread. FR-6 and FR-6a rest on
  the arm this build has, applied to every slot.

## The build

`builds/0094-party-slot/` holds `againrom.exe`, `almtool.exe`, `missionrun.exe` and `restool.exe`,
built `-trimpath` from this branch, with a README.

**Every command in that README was run before it was written down, in PowerShell, against both
installs.** One of them was wrong and is corrected there: the obvious way to get one map out of an
archive is `restool cat ... > $env:TEMP\10.alm`, and PowerShell's `>` prepends a UTF-8 byte-order
mark, so `almtool` is handed `EF BB BF 4D ...` and refuses it with `bad magic 0x4dbfbbef`. The README
uses `restool extract`, which writes bytes as bytes. `againrom.exe -check` resolves both roots (38
map rows on EN, 34 on RU).

## A fifth comment, found at Verify

`pkg/sim/world.go`'s own doc for `Entity.Owner` — the field this story is about — read *"An entity
built from no placement — the hero, and everything this tree spawns — carries that zero and is owned
by nobody"*, beside *"NOTHING IN THIS BUILD READS IT"*. The second half had been stale since 0086
put the field under every acquisition; together they made the sentence that looked like a note about
a spare field into the statement that no map could ever fight the player. Corrected here, with the
two obligations it names settled where they now stand: which units the player may command is
answered on the campaign path, and the far-search budget's owner term is **not** — it asks whether a
human participant owns the mover, and knowing where the campaign seats its own player does not
answer that. That arm goes in unguarded and is **unchanged**; only its stated premise moved, and
`pkg/sim/step.go` says so. Comments only, no executable line.
