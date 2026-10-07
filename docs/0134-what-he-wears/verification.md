# 0134 — verification

Environment: Windows 11, Go toolchain pinned by `go.mod`. Both lawful roots read at
`gameversions/en` and `gameversions/ru`; no game data is committed.

## Gates

```
go build ./...                          clean
go vet ./...                            clean
gofmt -l $(git ls-files '*.go')         printed nothing
go test -count=1 -trimpath ./...        every package ok; pkg/data 2.352s, pkg/game 2.828s,
                                        pkg/mapload 0.610s, pkg/sim 4.512s, cmd/appearcheck ok
bash scripts/check-no-game-assets.sh    check-no-game-assets: clean (tree scan)
bash scripts/check-doc-budget.sh        exit 0; plan <= 1.2 x spec, tasks <= 1.2 x plan
bash scripts/check-hotfix-ledger.sh     check-hotfix-ledger: ok
bash scripts/check-sdd-audit.sh         one FAIL, this file's own absence, taken before it existed;
                                        the FAIL set is otherwise empty, as it is on the base commit
```

## AC-13 — the instrument, on both roots

`appearcheck -assets <root>`, exit 0 on both. The two roots' output is **byte-identical**
(`diff` reports no difference), which is the shipped data agreeing cell for cell across languages.

```
fighter / male   weapon=Iron Short Sword
  trained Blade
  slot 1 Short Sword 0x0103
  slot 7 Soft Mail 0xb70f
  slot 12 Soft Boots 0xac3c
  body=swordsman dir=heroes_l class=3 sheet=units/heroes_l/swordsman/sprites.256 (address inside the graphics container) loads=yes

fighter / female   weapon=Iron Short Sword
  slot 1 Short Sword 0x0103
  slot 7 Soft Mail 0xb72f
  body=swordsman dir=heroes_l class=3 sheet=units/heroes_l/swordsman/sprites.256 loads=yes

mage / male   weapon=Wood Staff
  slot 1 Staff 0x810d
  slot 7 Robe 0x072d
  slot 8 Cloak 0x082b
  body=mage_st dir=heroes class=24 sheet=units/heroes/mage_st/sprites.256 loads=yes

mage / female   weapon=Wood Staff
  slot 1 Staff 0x810d
  slot 7 Dress 0x072e
  slot 8 Cloak 0x082b
  body=mage_st dir=heroes class=24 sheet=units/heroes/mage_st/sprites.256 loads=yes
```

`loads=yes` is not a directory listing: it is `LoadHeroBody` run against the install through the
same code the game uses, reporting that the composed address decoded into frames.

The witness across the trained skill, both classes, same roots:

```
skill witness (class, trained skill -> weapon, body, dir, drawn class):
  fighter Blade    weapon=Iron Short Sword       body=swordsman  dir=heroes_l class=3
  fighter Axe      weapon=Uncommon Bronze Axe    body=axeman     dir=heroes_l class=7
  fighter Bludgen  weapon=Uncommon Bronze Mace   body=clubman    dir=heroes_l class=10
  fighter Pike     weapon=Bronze Pike            body=pikeman    dir=heroes_l class=12
  fighter Shooting weapon=Uncommon Wood Short Bow body=archer    dir=heroes_l class=14
  mage    Fire     weapon=Wood Staff             body=mage_st    dir=heroes   class=24
  mage    Water    weapon=Wood Staff             body=mage_st    dir=heroes   class=24
  mage    Air      weapon=Wood Staff             body=mage_st    dir=heroes   class=24
  mage    Earth    weapon=Wood Staff             body=mage_st    dir=heroes   class=24
  mage    Astral   weapon=Wood Staff             body=mage_st    dir=heroes   class=24
```

An unplanned agreement worth recording, because neither side was fitted to the other: the drawn
class this derivation produces for each archetype — 3, 14 for the two fighters at their own
archetype's trained skill, 24 for both mages — is exactly the value each of the four shipped base
rows carries in its own drawn-class column. The derivation reads a weapon's definition row against
a shipped text payload; the column is a different cell of a different file.

## AC-1, AC-2, AC-3, AC-4, AC-5, AC-14, AC-15, P-6

Unit tests over synthetic collections in `pkg/mapload` and `pkg/game`. Reverting the line that
hands the base row's cells to the worn-set resolution fails `TestAC3TheHandedWeaponDisplacesTheBase
RowsOwnWeaponCell` and `TestAC14NoEquipIsRefusedForAClass`. Removing the mage arm from the weapon
name lookup fails `TestAC4AMageHoldsTheMagesWeaponForEverySkillChoice` and
`TestTheMageArmNamesOneLiteralWhateverSlot`.

## AC-6, AC-7, AC-8, AC-9, AC-16, P-1, P-2

Unit tests in `pkg/data`: AC-6 the shield suffix, AC-7 the mage substitution, AC-8 the material's
own directory and the refusal past the sixteen, AC-9 two characters differing only in sex deriving
one name, one directory and one class, AC-16 the mage and unarmoured arms, P-1 totality over the
empty set and the empty list, P-2 the class always being the composed name's own. Reverting the composed derivation's directory arm to the three literals
it replaced fails `TestHeroAppearanceComposesTheDirectory`.

## AC-17, and the loader

Reverting the bundle key to the bare body name fails five named tests across three packages —
`TestTwoDirectoriesShippingOneNameBothResolveIntoOneBundle`,
`TestLoadHeroBodyDoesNotReloadAKeyItAlreadyHolds`,
`TestAHeroBodyDrawsTheComposedSheetOnTheRecordsGeometry`, `cmd/againrom`'s `TestUnitBundle`, and
two in `cmd/appearcheck`. The writer and the reader are coupled by that fact rather than by care.

## AC-10, AC-11, AC-12, AC-18, P-3, P-4, P-5

P-4 is witnessed by construction and is stated as such: the refresh replaces the bundle entry the
draw path substitutes and writes nothing else, and no path in this tree draws anything over a
world sprite — the whole sheet is what moves. Unit tests in `pkg/game` carry the rest. Commenting
out the appearance refresh's call from the frame-paced path
fails `TestPacedMovesTheDrawnClassWhenEquipmentComposesADifferentBody`, which drives a real equip
command through `sim.Step` and then one paced call, and observes the substituted class move.

## The install still opens

```
againrom -check -assets <en>
againrom: 66 map rows, 8 of 8 buttons have a mask region; hero Body 43, Reaction 26, Mind 15,
Spirit 15, Blade 10, Iron Short Sword 10-16, to-hit 49, defence 8

missionrun -assets <en> -mission 10 -ticks 200
mission 10  scenario/10.alm  80x80  36 entities
outcome undecided at tick 64
```

## What is NOT witnessed

**Nobody has seen the mage drawn.** No test in this tree observes pixels. What is witnessed is the
chain up to the substituted class record: the body name, the directory, the composed sheet address,
that the sheet decodes into frames out of both installs, and that the substitution reaches the map
the draw path reads. The last hop — that map to the screen — is 0085's wiring and is unchanged by
this story, so it is inherited rather than re-proved. The owner is the instrument for that step.

Equally unwitnessed: the appearance of a character who equips a piece of **armour** in a mission,
because nothing in this tree equips armour from the pack yet; the refresh derives from all twelve
slots, but only slot 1 can currently move. And the dying substitution stays unreached.

## Divergences, disclosed

- The shipped mage literal carries a spell attachment. This tree has no weapon that carries a
  spell, so the staff is handed over without it: the mage's blow is a staff's, and no spell is
  attached to it.
- A generated fighter's own base row also names a weapon, and the trained skill's weapon takes its
  place in the hand. Which of the two the original leaves in the hand is not established.
- A piece whose name carries no material word resolves to the first material block. No shipped
  generated character reaches the arm that reads it, because only the two mages fill the directory
  slot and a mage takes the mage arm regardless.
