# The second game's death item roll

## Intent

In the second game a unit whose Units row reaches the roll leaves gold and
an item in a ground sack at its cell when it dies. Dragons nearly always
drop a staff, book, scroll or potion; ogres, orcs, the undead and the
necromancers sometimes drop enchanted equipment or a magic item. The first
game keeps its drops, its World hash and its SAV bytes.

## Authority

- `R2-ENGINE-940` (High/Medium): the death's gate (typeID 64 or more,
  +0x1a0 clear), the order of its draws and the sack.
- `R2-ENGINE-941` (High/Medium/Unknown): the death's item list is the shop
  draw of one category {parameters 42, 43, 100 draws, bound 1, mask 44}.
- `R2-ENGINE-265` (High) names EN draw `L2.00813` as the town fill's
  callee; `R2-ENGINE-941` names that same function as the death draw.
  `R2-ENGINE-800` confirms the fill's call and distinguishes a second,
  differently behaving routine. All three corrected exits therefore apply
  to the town fill and the death draw.
- `R2-ENGINE-942` (High/Medium): the 66 Units rows that reach the roll, and
  their chances and masks.
- `R2-ENGINE-943` (High/Medium): a template whose bag is discarded still
  rolls, and its gold and item still drop.
- `HERO-KILL-027`: the first game's gold roll, unchanged.

## As-built behaviour

The Units reader `data.NewUnitDef` reads columns 41..44 as `ItemChance`,
`ItemMin`, `ItemMax` and `ItemMask`; a cell of −1 reads as 0. A table with a
death item draw gives each Units actor those columns as its
`sim.DeathItemRoll`; the first game's table has none, so its actors hold the
zero roll.

`World.deathRoll` is the one death roll; `deathGold` is gone. It draws, on
the World's one generator, in this order:

1. rand(100) against the gold chance; when the chance is greater, the
   amount, minimum plus rand(span).
2. Under the second game's profile only: rand(100) against the item chance.
   A chance of 0 or below never passes.
3. On a pass, the bound draw, which takes its own rand() values from the
   same generator (the shop draw's order, below).
4. When the list holds n > 0 entries, entry rand(n − 1) as one unit.

The gate admits typeID 0x41 and above in the first game and 64 and above
in the second (`deathRollAt64`); a reward mark closes it in the second
game, as before. The three callers are the plain and discarding death in
`step.go`, the source-equipment body drop and the saved-body ground drop.
Each pours the gold and the item through `pourSack`; a discarded bag still
drops the roll. `hasTerminalLoot` counts an item roll, so a body whose only
reward is its item waits for a cell.

The seam is `sim.DeathItemDraw`, a function of the roll and a rand()
source answering the list. `mapload.Table.DeathItems` carries it,
`bindTableInputs` binds it beside the rules, and `pkg/game` sets it for the
second game in `gameFiles.deathItems` (`secondDeathItems`), the shop draw
over the table's shop collections. A table without them draws no list.

The shop draw is `secondshop.Tables.DeathItems`: the shop's category
builder `Tables.category`, which the shop fill's `Profile` also uses, for
the one death category, without the mods' stocked items, each entry turned
into its bag unit by `Tables.Instance`. Its draws, in order: the magic
admission's chooser draw per spell row when the mask holds bit 26; the
potions' quantities; per draw a candidate pick, its book redraws, rand(100)
under bits 28 and 29, the enchant's own draws, and a quantity draw for a
kept stackable unit. Under bits 28 and 29 a unit is enchanted only on
rand(100) above 50; a kept unit that stacks draws its quantity; magic
admission keeps books and scrolls by price. The shared `shopstock.Generate`
checks for candidates before the lead. A magic category with no candidate
leaves an empty list and takes no potion draws (DIV-4621 closed).
`shopstock.roll` ends after more than ten failures per requested draw
(1001 failures for a death), or immediately when a candidate is still a
book after 1000 redraws. These are the High exits of `R2-ENGINE-941`, shared
with the town fill. No caller-specific exit field or second loop is needed.

The roll crosses the World's byte form in the `DIT1` wrapper (version 134),
written only when an actor holds a roll, and the SAV's departed-body JSON
only when it is set (`omitzero`); a first-game World and SAV keep their
bytes. A living actor keeps its roll across a mission SAVE and LOAD. The
draw is carried across a World decode and across the original-living
staging decode.

### Kinds and builders

| kind | builder |
| --- | --- |
| the Units table reader | `data.NewUnitDef` (columns 41..44 added) |
| the death roll, both games | `World.deathRoll`, the second game by `gameProfile` data |
| the shop draw | `secondshop.Tables.category`, used by `Profile` and `DeathItems` |
| the ground sack | `World.pourSack`, unchanged |
| the World byte form | the `formWrappers` list (`DIT1` added) |
| the game's table inputs | `readUnder` over `gameFiles`, `Table.bindTableInputs` |

## Proof

- `pkg/game/deathrollrecord_test.go`, `TestFirstGameDeathRollKeepsTheRecord`,
  committed alone at the base: four rounds of first-game deaths (an orc and
  an ogre whose rows hold item columns, a type-64 goblin and a troll), their
  sacks and gold, the World hash and the SAV digest. Equal after the change.
- `pkg/sim/deathroll_test.go`: the draw order against a replica of the
  generator in both random modes, a pinned seeded sequence (an order change,
  the item chance drawn before the gold, fails both), no item for chance 0
  or −1, type 63, a reward mark or the first game's profile, type 64 under
  the second game, the kill's sack, the discarded bag, terminal loot and the
  byte form.
- `pkg/secondshop/deathitems_test.go`: the death list equals the shop fill
  of the one category on the same draws, an empty candidate set gives an
  empty list for magic masks too, and bit 28 leaves some units plain.
  `TestDeathItemsInstalledOgreFailureBound` loads the installed Ogre price
  windows 300..1500 and 2100..10050 at seed 1: both count 1001 failures;
  the lists hold 5 and 99 entries, and post-death random states are
  3265786310 and 1827980306. It requires `AGAINROM_ASSETS`.
- `pkg/game/deathdraw_test.go`,
  `TestSecondDeathDrawEndsAtTheClaimBoundary`: the review's book-only Dragon
  takes 1036 generator draws and drops potion 0x0e49, random state
  1745243601; the no-candidate Dragon takes 29 and leaves only 6381 gold,
  random state 1423890337. A synthetic failed-clone Ogre takes 1001 draws,
  random state 2662887517. All three assertions fail on the returned code.
- `pkg/game/seconddeathdrop_test.go`: a second-game mission whose dragon
  dies drops gold and a book, scroll or potion; a mission SAVE after the
  drop LOADs with the same sack and item, and the hero's pick-up takes it; a
  living dragon keeps its roll across SAVE and LOAD and its death there
  drops the item.
- `pkg/game/seconddeathdrop_release_test.go`,
  `TestReleaseSecondDeathDropUnitsRows`, on the EN and RU second-game
  roots: the 66 rows of `R2-ENGINE-942` with their chances and masks,
  through the engine's loaders, each row's candidate count, a candidate for
  every row whose mask holds the magic bit, and the installed draw of every
  Dragon and Ogre row on 20 seeds.

### Corrected town shelves

The correction reran the review's town walk on EN and RU separately through
`OpenArchives` and `LoadDefinitionsFor`: the initial records and ten ascending
departure/stage pairs (10 -> 20 through 100 -> 110), class 4 added after
departure 80, towns 1..3, seeds 1, 2 and 71. Each root supplies 99 fills and
396 category lists. Against returned candidate `408e22607427de8e87fa85520da3489130a480d1`,
four lists change: town 2, seed 2, categories 2 and 3 in the initial records
and after departure 10 / stage 20. The final random state changes for those
two fills. The other 392 lists and 97 final states stay equal. Each of the
11 distinct magic categories retains candidates; this walk reaches no
empty-magic exit. These are bounded inputs, not every possible campaign or
imported assortment.

Both roots' complete-stock/random-state digest changes from
`c6ed56c8d219e25e40065d7f797bc7a497f6aa5d95bf14c113fa8d8693a9a320` to
`2f23e19453d23cd44e26f1e6e11c876b7e26335087099d3b5652ecd876a9406e`.
The stock digest helper over initial towns 1 and 2 and seeds 1..100 gives
`e9a2db22218792f3fcb4e94fffc9167736dbe8bafdb92d41ca7736f8d2c99861`
on both roots. The golden in `TestReleaseSecondShopStockMatchesMain` is
updated to that value with its claim; the release test itself was not run
in the correction. The synthetic Sleep fixture now admits scroll rows 6
and 35 before the potion lead, so its restock and trade witnesses keep a
nonempty shelf. Its Sleep, shop-state and random-stream goldens cite
`R2-ENGINE-941`; their first-game records stay unchanged.

## Open debt

- DIV-4619: rand(0) of a one-entry list, the order of a list holding books
  or magic items, and which clones stack. The two loop exits are corrected.
- DIV-4620: the range test of a plain unit under bit 28.
- DIV-4622: the Medium Units binding and the unread sack merge.
