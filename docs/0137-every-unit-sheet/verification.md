# Verification — 0137 every unit states its own character sheet

Toolchain `go 1.26.1`. Developer runs are against the two lawful installs at
`gameversions/en` and `gameversions/ru`; no test reads either.

## SC-1 — the gate

```
go build ./...                       clean
go vet ./...                         clean
gofmt -l $(git ls-files '*.go')      prints nothing
go test -count=1 -trimpath ./...     EXIT=0, every package ok
bash scripts/check-no-game-assets.sh EXIT=0   check-no-game-assets: clean (tree scan)
bash scripts/check-doc-budget.sh     EXIT=0
bash scripts/check-hotfix-ledger.sh  EXIT=0   19 commit(s) since 9729459 examined
bash scripts/check-sdd-audit.sh      EXIT=0, FAIL set empty
```

`gofmt` named `cmd/classdump/sheet_test.go` once, after T4; it is fixed in its own commit. The
note is worth keeping because of *how* it survived: `gofmt -l` exits 0 whether or not it names a
file, so a gate written as one `&&` chain runs green straight over it.

The audit's note/warning count is not comparable from a lane — a worktree has no `builds/` — so
only the FAIL set is read here.

## The unit evidence

| Test | Package | What it pins |
|---|---|---|
| `TestCreatureSheetFamiliesAndSkillPositions` | `pkg/mapload` | **AC-2, AC-3, R-1.** Ten distinct cells in one synthetic units row: the elemental columns reach `Elemental`, the weapon-kind columns reach `WeaponKind`, and it is the **weapon-kind** block — not the elemental one — that reaches skill positions 1..5, with position 0 left unstated |
| `TestPersonSheetStatsSkillsFamiliesExperience` | `pkg/mapload` | **AC-4, SC-3.** Capped statistics, the row's own six raw levels, the derived families and the graph's experience. It writes one level outside the clamp range, so it fails if the graph's restored `Skill` is ever read instead of the row's |
| `TestPersonSheetIgnoresEquipment` | `pkg/mapload` | **AC-6a, FR-5a, SC-3.** A bare row and an otherwise identical row naming weapon, shield and armour produce equal sheets |
| `TestPlacedSheetsSkipsAnUnresolvedPlacement`, `TestPlacedSheetsWithNoMapOrTable` | `pkg/mapload` | **AC-1, AC-9.** No entry, nil map, nil table, table with neither collection: absence, not a zero character, and no panic |
| `TestPanelSkillRowWidthByBand` | `pkg/ui` | **AC-7, AC-8.** Six numbers for a person and for an unstated band, five for a creature — pinned with six *distinct* values, so it measures that position 0 is what was dropped. An unknown character still states nothing on any band |
| `TestPanelSkillRowNotSuppressedForAllZerosOnEitherBand` | `pkg/ui` | 0125's rule does not regress: an all-zero skill row is still drawn, six wide for a person and five for a creature |
| `TestDriverStatesACharacterForEachPlacedBandAndForTheParty` | `pkg/game` | **AC-1, AC-2, AC-4.** A placed creature and a placed person both reach `mw.chars` through the driver, with the values their rows state |
| `TestMissionCharactersMergesPlacementsThenParty` | `pkg/game` | **AC-5, SC-4.** With an id collision forced, the party's character wins and equals `partyCharacters` exactly |
| `TestAStepDoesNotMoveACreaturesSkillPositions` | `pkg/game` | **AC-6.** A stepped world, one blow landed by each of a creature and a person: the creature's skill positions stay bit-for-bit its columns while its experience still rises; the person's credited slot becomes the level of his entity's own experience |
| `TestDataBinVerbPrintsTheWholeCharacterSheet` | `cmd/classdump` | **AC-11, P-3, P-4.** A creature block, a person block and an unresolved one, over a synthetic map; the entity-carried numbers are cross-checked against `mapload.FromALMWith` rather than restated |
| `TestCampaignVerbPrintsTheSameSheetBlockAsDataBin` | `cmd/classdump` | **P-1.** Both verbs over one map and table: byte-identical blocks, placement for placement |

`TestPartyCharactersPairsTheStartsOwnSlices` (`pkg/game`, SC-4) was updated in T3 — three `want`
entries gained `Band: ui.CharacterBandPerson` and nothing else moved. That edit **is** AC-5's
evidence: the test asserts exact equality on the whole value, so had any number changed it would
have failed on that number instead.

`TestAStepDoesNotMoveACreaturesSkillPositions` was checked for hollowness by reverting the band
guard in `entityDraws`; it failed with `creature Skills = [0 0 0 0 0 0]`, which is the defect it
exists to catch.

## SC-6, SC-7, AC-10, AC-11 — both roots, campaign missions 10 and 20

`classdump -campaign <root> 10.alm` and `20.alm`, over each install. Every placement of both maps
resolved to an entry — 35 on map 10 (19 creature, 16 person) and 56 on map 20 (41 creature, 15
person) — so no block on either map exercises the no-entry arm; that arm is covered by the unit
tests above and named here as a gap in the developer evidence rather than left implied.

**The two roots agree.** The sheet sections are byte-identical: 249 lines on map 10 and 396 on map
20, `diff` clean both times. (The surrounding campaign census differs by four lines because the
two installs carry different loose maps at the root, which is not this story's output.)

`NPC14_1` — the row the owner is looking at, three placements on each map. Identical on both maps
and both roots:

```
  sheet 29  band person   entry "NPC14_1"
    Body 35  Agility 29  Mind 24  Spirit 17
    Health 120/120  Mana 0/0
    Dmg 10-16  Absorb 0  Attack 82  Defense 9
    Skill Blade 6 Axe 7 Bludgen 24 Pike 3 Shooting 1
    Elemental Fire 8 Water 8 Air 8 Earth 8 Astral 8
    Weight -  XP -  Sight 6  Speed 17
```

The five combat numbers match what this build already derived before this story — 120 health,
attack 82, damage 10-16, defence 9, absorb 0 — so the sheet added the eleven values that were
missing and moved none that were there. `Elemental` is 8 across, and `Spirit` is 17: the
derivation for a person is Spirit halved, so the row is its own cross-check.

A creature block, from map 20 and from `Horror.alm`, which is where the weapon-kind columns are
not all zero:

```
  sheet 0  band creature entry "Squirrel"
    Body 10  Agility 60  Mind 3  Spirit 6
    Health 20/20  Mana 0/0
    Dmg 3-6  Absorb 0  Attack 40  Defense 0
    Skill Blade 0 Axe 0 Bludgen 0 Pike 0 Shooting 0
    Elemental Fire 30 Water 0 Air 0 Earth 0 Astral 0
    Weight -  XP 6  Sight 8  Speed 24

  sheet 1  band creature entry "Dragon.4"
    Skill Blade 0 Axe 0 Bludgen 30 Pike 0 Shooting 20
    Elemental Fire 95 Water 80 Air 91 Earth 60 Astral 88
```

`Axe 0` on every creature seen is the corpus's own figure and not a bug in the copy; the two rows
above also show the elemental and weapon-kind blocks holding different numbers, which is the
corpus-side answer to R-1.

## SC-2, SC-5, AC-7 — what the owner sees

`paneldump -mission N`, which composes the real panel against the install. Witnessed by reverting
the one line that feeds it, so the two columns are the same binary either side of one call:

```
BEFORE (mission 10)                     AFTER (mission 10)
placed id=0  box=227x139  rows=7        placed id=0  box=290x229  rows=12
  | Human ClubMan                         | Human ClubMan
  | HP 10/10                              | HP 10/10
  | CELL 24, 54  SPEED 16                 | CELL 24, 54  SPEED 16
  | WORN Club, Soft Boots                 | BODY 5  REACT 20
  | DMG 2-3  HIT 3                        | MIND 15  SPIRIT 15
  | DEF 6  ABS 0                          | SKILL 0 0 0 0 0 0  XP 0
  | SWING 7/4                             | WORN Club, Soft Boots
                                          | DMG 2-3  HIT 3
                                          | DEF 6  ABS 0
                                          | SWING 7/4
                                          | PROT 7 7 7 7 7
                                          | RES 0 0 0 0 0

BEFORE (mission 20)                     AFTER (mission 20)
placed id=0  box=227x121  rows=6        placed id=0  box=267x211  rows=11
  | Squirrel                              | Squirrel
  | HP 20/20                              | HP 20/20
  | CELL 50, 46  SPEED 24                 | CELL 50, 46  SPEED 24
  | DMG 3-6  HIT 40                       | BODY 10  REACT 60
  | DEF 0  ABS 0                          | MIND 3  SPIRIT 6
  | SWING 8/4                             | SKILL 0 0 0 0 0  XP 0
                                          | DMG 3-6  HIT 40
                                          | DEF 0  ABS 0
                                          | SWING 8/4
                                          | PROT 30 0 0 0 0
                                          | RES 0 0 0 0 0
```

Identical on both roots. The creature's `SKILL` row states **five** numbers and the person's
**six** — AC-7 on screen rather than in a formatter test — and the creature's `PROT 30 0 0 0 0`
is its own elemental columns, which no party member's row could produce.

## SC-8 — the mission-10 drive

Unmoved, both roots, with the milestone's own argv
(`-mission 10 -census -waypoint u21:56:21:3 -waypoint p0:66:16:3`):

```
before and after:  outcome lost at tick 224
                   census: 4 of 36 unit(s) moved, 1 fell, over 224 tick(s)
```

## P-1, P-2, P-3, P-4

- **P-1** — `PlacedSheets` reads its two arguments and nothing else: no clock, no generator, no
  global, no file. Measured rather than asserted by `TestCampaignVerbPrintsTheSameSheetBlockAsDataBin`,
  which builds the sheets twice by two call paths over one map and requires the printed blocks to
  be byte-identical.
- **P-2** — `git diff --stat <base>..HEAD -- pkg/sim/` is **empty**: no simulation file changed, no
  entity field was added, the byte form and its version are untouched and every digest test passes
  unchanged.
- **P-3** — the tool reads health, mana, the damage pair, absorption, to-hit, defence, sight, rate
  and the experience value off the built world's entities and takes only the statistics, the skill
  positions and the elemental family from `PlacedSheets`. `TestDataBinVerbPrintsTheWholeCharacterSheet`
  compares the entity-sourced columns against `mapload.FromALMWith`'s own output.
- **P-4** — `Weight` prints `-` on every row of both maps and both roots, and a person's `XP`
  prints `-`; a creature's `XP` prints its column. **One disclosed edge:** an *unresolved*
  placement's `XP` prints a number rather than a dash. It is not an invented zero — it is the
  entity's own `XPValue` field as the world carries it, which is the base constructor's value —
  but a reader could take it for a column. No placement of either campaign map takes that arm.

## AC-12 — the campaign verb

The verb still reports all 38 maps on both roots, and its table now names the armour and shield
collections a later story added and never wired here. No existing report's numbers moved: the
whole `cmd/classdump` suite passes unchanged, and on this arm nothing could move — a person's
combat block and health maximum read no armour and no shield in this tree, so the only newly
resolved values are the worn and carried records.

## Reconciliations, and what was not done

Two commits carry no trailer because neither is a task:

1. **Comments this story falsified.** Four sentences in `pkg/game/world.go` (twice), `pkg/ui/overlay.go`
   and `pkg/ui/panel.go` (twice) said a character is known for the party and for nobody else. One
   further edit removed a reference to a task brief from `CharacterBand`'s doc (S-4).
2. **`cmd/paneldump`.** It composed its subject from `PartyCharacters` and would have gone on
   measuring the panel the game stopped drawing. `pkg/game` now exports `MissionCharacters` beside
   `PartyCharacters` and the tool reads it; `PartyCharacters` stays, because with every unit known
   it is what still tells the two subjects apart.

Not done, and out of scope by the spec: a placed unit's `WEAPON` row is still absent — its name is
never resolved for it, and the row states nothing rather than an empty string. Where the original
draws each value is not decoded and is not claimed; the one authored decision — that a creature's
five weapon-kind columns belong in the sheet's weapon-skill positions — is the owner's ruling and
is recorded as authored in `provenance.md`, against `UNIT-PANEL-011`, which is active and says the
layout is on nothing.
