# 0136 — armour is worn, and it counts: verification

Go 1.26.1 (`go.mod`'s own pin), windows/amd64. Both lawful roots read for every corpus and drive
figure below; every one is identical on the two unless the line says otherwise. Nothing here was
run from a dirty tree. Every figure was re-measured **after** master was merged in mid-story: the
merge brought `0134`, which makes a generated character start wearing his own base row's cells, and
that changes what this story's fold has to fold.

## The gate

```
go build ./...                                   clean
go vet ./...                                     clean
test -z "$(gofmt -l $(git ls-files '*.go'))"     clean
go test -count=1 -trimpath ./...                 every package ok, none failed
bash scripts/check-no-game-assets.sh             check-no-game-assets: clean (tree scan)
bash scripts/check-hotfix-ledger.sh              check-hotfix-ledger: ok
bash scripts/check-doc-budget.sh                 0136: plan <= 1.2 x spec, tasks <= 1.2 x plan, every file under
bash scripts/check-sdd-audit.sh                  FAIL set empty for 0136; 5/5 tasks landed, one trailer each
git diff --diff-filter=D --name-only <base>..HEAD   empty
```

`gofmt` is spelled as a `test -z` and not as `gofmt -l … && echo`, and the difference is not
cosmetic: `gofmt -l` exits 0 whether or not it names a file, so the second form reports clean while
printing the offending name. Two test files landed unformatted behind exactly that reading, and
were corrected in their own commit.

## The deliverable — fell a clubman, take the boots, wear them

`missionrun -mission 10 -attack p0:u19 -wear p0:24:56 -ticks 4000`, byte-identical on both roots:

```
attack 1  p0 -> u19 : FELLED it after 156 ticks, victim at 0 hp, attacker facing N
  before, worn: 1=0801007(slot 1) "Common Wood Club" 12=1012028(slot 12) "Soft Boots" defence 1 absorption 0
  before, carried: none
  ground (24,56): 0801007(slot 1) "Common Wood Club" 1012028(slot 12) "Soft Boots" defence 1 absorption 0
wear 1  p0 take (24,56)
  before: defence 8 absorption 0
  0801007(slot 1) "Common Wood Club" : slot 1
  1012028(slot 12) "Soft Boots" defence 1 absorption 0 : slot 12
  after: defence 13 absorption 0
```

**Defence 8 → 13** (AC-12, SC-5), and the +5 needs taking apart, because three things moved at
once. Three further runs against the same install do that:

| Run | `after` |
|---|---|
| as above | 13 |
| the same, with `FoldWear`'s result discarded in `Rearm` | **8** |
| `-wear p0:0:0`, a cell holding no sack, so only the re-derivation runs | **14** |

So: the whole of the movement is the fold, and swapping the weapon contributes nothing to defence
at all. The party hero already wears Soft Mail in slot 7 and Soft Boots in slot 12 — `0134`'s doing
— and until this story none of it counted; folding what he already wears is worth **+6**. Taking
the clubman's boots then costs him **one point**, because his own boots are the `Uncommon` shape
(factor 0.300001, scaling to defence 2) and the clubman's are `Common` (0.250001, scaling to 1).
That is the right answer and not a defect: the piece on the floor is worse than the piece he has
on, and the arithmetic says so per piece.

## The milestone did not move

`missionrun -mission 10 -census -waypoint u21:56:21:3 -waypoint p0:66:16:3`, before this story, at
its head, and again after the merge — both roots, `outcome lost at tick 224` every time, with the
same four movers and the same census (SC-6). Nothing this story changes runs until an equip or a
re-derivation is issued, and that drive issues neither.

## Corpus, both roots, from a scratch probe deleted before any commit

30 named `Armors` rows, every one 17 params wide. `Slot` ∈ {4,5,6,7,8,9,10,12}. The defence column
is nonzero on **all 30**; the absorption column is zero on **25** of them — the five that are not
are the plate pieces and the cuirass. No column, on either root, is negative. Over every
row × shape × material the shipped tables allow, scaled defence spans `[0.1395, 101.9059]` and
scaled absorption `[0.0000, 4.7159]`, so DD-11's 16-bit conversion is unreachable on shipped data
in both directions.

The clubman's boots: `Armors` row 28, material index 10, shape index 0, defence column 21,
absorption column 0; shape factor 0.250001 and material factor 0.279083 at ladder index 6.
`21 × 0.250001 × 0.279083 = 1.46519`; `+ 0.5 = 1.96519`; truncated, **1**. Absorption
`0 × 1 × 1 = 0`. That is the arithmetic the drive above prints as `defence 1 absorption 0`.

**DD-1's premise, measured rather than argued.** Recomposing an armour's name from its code and
re-entering the name resolver — `WeaponFromCode`'s own trick — fails on a shipped piece:

```
recomposed "Common Leather Soft Boots" -> err=data: armour "...": no Armors entry named "Soft Soft Boots"
recomposed "Common Leather Shoes"      -> err=data: armour "...": no Armors entry named "Soft Shoes"
recomposed "Common Leather Helm"       -> {Row:7 Slot:6} err=<nil>
```

The implied-shape word goes back on a residue that already carries it. `ArmorFromCode` therefore
resolves by index, and the class of piece the owner asked about is exactly the class that would
have failed.

**The doll's premise, measured the same way.** `cmd/appearcheck` against both roots prints a
fighter starting in slots 1, 7 and 12 and a mage in slots 1, 7 and 8 — so occupied slots that the
window drew nothing for existed before this story ever opened the equip gate.

## Acceptance criteria

| # | Evidence |
|---|---|
| AC-1 | `TestResolveArmorRoundsDefenceUpAndAbsorptionDown` — equal columns and equal factors, defence one above absorption |
| AC-2 | `TestResolveArmorAnIntegerProductRoundsDefenceAndAbsorptionEqual` |
| AC-3 | `TestArmorFromCodeAgreesWithResolveArmorFieldForField` — `Name` apart, which FR-4 states |
| AC-4 | `TestArmorFromCodeRefusesEveryFR5Case` — each of the six refusals separately |
| AC-5 | `TestFoldWearSumsArmourAloneAndZeroesProtectionAndResistance` |
| AC-6 | `TestRearmSumsTwoWornArmourPiecesWithEachSumOnItsOwnStatistic` |
| AC-7 | `TestEnqueueEquipAppendsAnArmourCommandAtTheRowsOwnSlotNotThePackIndex`; and the drive above, where the code sat at pack index 1 and landed in slot 12 |
| AC-8 | `TestFoldWearContributionDoesNotDependOnWhichSlotCarriesTheCode` |
| AC-9 | `TestEquippingAMetalPieceIsAcceptedForAMagesOwnCharacter` |
| AC-10 | `TestEquippingAnArmourRaisesDefenceAndAbsorptionWhileLeavingTheWeaponsNumbersUntouched` — the eight weapon-derived fields asserted byte-equal across the equip |
| AC-11 | `TestRefreshEquipmentMovesTheTrackerWhenAPieceLandsOutsideSlotOne`, `TestRefreshEquipmentRecomposesNothingWhenTheEquipmentHasNotChanged` |
| AC-11a | `TestComposeInventorySubjectPaintsEveryOccupiedSlotInAscendingOrder` — three occupied slots, the paint ORDER proved by a pixel all three layers touch, the nine empty slots asserted iconless; and `TestRefreshEquipmentDrawsEveryOccupiedSlotsIconIncludingTheTwelfth` |
| AC-11b | `TestBuildInventorySubjectComposesFromTheMembersWornSet` — worn-set only, weapon fallback on an empty worn set, and worn winning where both are present |
| AC-12 | the drive above, both roots |

## Success criteria — the six that were falsified rather than asserted

A criterion is only worth its row if something fails when the line it names is removed. Six were
tested that way, each restored afterwards and the gate re-run clean:

| Removed | Result |
|---|---|
| the `+ 0.5` from the defence term (`pkg/data/wear.go`) — SC-1 | `TestResolveArmorRoundsDefenceUpAndAbsorptionDown` failed, `Defence = 4, want 5` |
| the defence/absorption pairing, exchanged in `FoldWear` — SC-7 | three tests failed, `TestFoldWearDefenceAndAbsorptionSumsLandOnTheirOwnField` among them |
| `FoldWear`'s result, discarded in `Rearm` — SC-4 | three `pkg/game` tests failed, and the real drive's `after` fell back to 8 |
| `refreshEquipment`'s guard, narrowed to slot 1 — SC-4 | `TestRefreshEquipmentMovesTheTrackerWhenAPieceLandsOutsideSlotOne` failed |
| the composition's loop, narrowed to slot 1 — SC-10 | `TestComposeInventorySubjectPaintsEveryOccupiedSlotInAscendingOrder` and `TestRefreshEquipmentDrawsEveryOccupiedSlotsIconIncludingTheTwelfth` failed |
| the worn-set read at the open — SC-11 | covered by `TestBuildInventorySubjectComposesFromTheMembersWornSet`, whose three subtests separate the worn set, the fallback and the precedence |

SC-2, SC-3, SC-8 and SC-9 are covered by the named tests in the AC table — SC-2 by
`TestArmorFromCodeRefusesEveryFR5Case`, SC-3 by the zero-array assertions in
`TestFoldWearSumsArmourAloneAndZeroesProtectionAndResistance`, SC-8 by
`TestEquippingAMetalPieceIsAcceptedForAMagesOwnCharacter` together with the fact that neither
`EquipTarget` nor `FoldWear` takes an argument a wearer test could be written against, and SC-9 by
`TestArmorFromCodeAgreesWithResolveArmorFieldForField` and
`TestFoldWearContributionDoesNotDependOnWhichSlotCarriesTheCode`. SC-5 and SC-6 are the drives
above.

## Limitations, stated as limitations

- **Nobody has looked at the rendered doll.** What is measured is which addresses the composition
  read, which slots carry an icon, and — through one pixel three layers overlap on — the order they
  were painted in. Pixels are the owner's instrument, not this lane's.
- **The mint and the first tick disagree by the fold.** `mapload.PartySpawn` derives a party
  member's block with no armour, and the first tick re-derives it with his armour: measured as
  defence 8 at the mint and 14 one `Rearm` later, on the same character. The game closes the gap on
  tick 1 and does it deterministically; a headless caller that never ticks reads the unfolded
  number. Not fixed here — `PartySpawn` takes no definition table, and folding there moves every
  party member's minted, hashed defence at load.
- **A placed person's starting armour is still not folded.** The clubman felled above wears the
  boots himself and fights without their defence. This is the divergence the spec discloses, and it
  is the largest thing this story leaves standing.
- **Shields are still refused** by the gate, exactly as before this story.
- **The tool's take has no walk and no distance test.** It issues the transfer primitive directly,
  which is faithful to the routine that has none, but it is not a player picking something up.
- The corpus figures above come from a scratch probe run against both installs and deleted before
  any commit; they are reproducible from the tool's own output only for the boots.
