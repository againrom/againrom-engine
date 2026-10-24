# Story 1509: item materials and new weapon rows as mod data

## Intent

A mod's `data/items.toml` allows or forbids a material for one tier of an
armour, shield or weapon row, and adds a weapon row copied from a shipped row
it names. The items kind merges through `mod.Merge` like the other merged kinds.
Owner direction: BACKLOG T9.

## Authority

| Fact | Claim |
|---|---|
| Five `u16` material masks at `row+0x1c`, one per Shapes row, one bit per material | DAT-MATMASK-020 |
| The mask predicts the picture set exactly: 416 legal items | ITEM-PICT-050 |
| Group-C row layout; the second array is a CStringArray, empty on every weapon row | DAT-OBJ-002, DAT-STRARRAY-022 |
| Weapons count 28, rows 1..27, 17 parameters | DAT-WEAPCELLS-023 |
| Readers of parameters 0, 1, 4, 10, 16 are Unknown | ITEM-219 |
| Parameter 15 sutableFor | ITEM-220 |
| Parameter 11 is the range byte, -1 reads 1, not scaled | ITEM-221 |
| Row identity, packed code, LOAD rebinds the archived row | ITEM-222 |
| Row identity reaches pictures, hero class, shop, SAV | ITEM-223 |
| The shop walk and random selection test the mask; direct construction and LOAD do not | ITEM-224 |
| A missing inventory picture takes the error path | ITEM-225 |
| Slot map: 2 Price, 3 weight, 6/7 physicalMin/Max, 8 toHit, 9 defence | ITEM-WEAPCOL-021, ITEM-SCALE-017, ITEM-DMGCOL-018 |
| Price reads Weapons parameter 2 | SHOP-103 |

## Settable weapon parameters

| Key | Parameter | Claim |
|---|---|---|
| `price` | 2 | SHOP-103, ITEM-WEAPCOL-021 |
| `weight` | 3 | ITEM-SCALE-017 |
| `damage-min`, `damage-max` | 6, 7 | ITEM-DMGCOL-018, ITEM-WEAPCOL-021 |
| `to-hit` | 8 | ITEM-SCALE-017 |
| `defence` | 9 | ITEM-SCALE-017 |
| `range` | 11 | ITEM-221 |

Every other parameter keeps the base row's value: 0, 1, 4, 10 and 16 have no
known reader (ITEM-219); 5, 12, 13, 14 and 15 choose the combat arm, the
cadence, the two-handed rule and who may use the weapon, which stay those of
the base row.

## As-built

### Sites

| Site before (engine main `f16bf050`) | What it decided | After |
|---|---|---|
| `pkg/mod/kinds.go:96-107` | items: `LayerKept`, owner T9; two changes both apply, the later last, unnamed | `pkg/mod/kinds.go:96`: `LayerMerged`; `Data.SetOrder` (`:304`) gives items the load order |
| `pkg/mod/items.go:177-300` | `[[item]]`, `[[change]]` | the same, plus `[[material]]` and `[[weapon]]` (`pkg/mod/itemrows.go:97`, `:157`; settable columns `WeaponParams` `:50`); `ItemData` gains `Materials`, `Weapons`, `Mods`, `Order` |
| `pkg/mod/kindread.go:15-30` | append rows and changes | also materials and weapon rows, and the mod's id in load order |
| `pkg/game/moditems.go:56-213` `SetModItems` | the one item builder, its own merge | `pkg/game/moditemlayer.go:419`: the same one builder, now over `mod.Merge` (layer 0 = the shipped armour, shield and weapon rows with masks, `itemLayer0`); added items as before; shipped rows with a changed field or bit become overlay rows; added weapon rows take free rows |
| `pkg/data/shopstock.go:121` `itemMask` | the one first-game mask reader | unchanged; `ItemMasks` (`:135`) and `ItemMaskRaw` (`:144`) wrap it |
| `pkg/data/shopstock.go:184` `ShopPool` | the one first-game candidate generator | unchanged; it reads the overlay's merged masks through `EntryRaw` |
| `pkg/game/shop.go:380-408` | pools over `t.Armors/Shields/Weapons` | unchanged |
| `pkg/data/itemcode.go:196-213` | picture lookup by code | unchanged; a mod's pair picture is served at the code's address |
| `pkg/data/equip.go:115-320` `BodyList` | body = shipped entry `row-1` | `WithAddedRow`, `RowEntry` (`:185`, `:208`): an added row answers its base row's entry |
| `pkg/game/modbodies.go:131` | a weapon-bodies record for a row with no shipped entry is refused | accepts an added row; `setBodyList` shared with items |
| `pkg/game/bodylist.go:33-95` | body choice text `weapon`, `body`, `bare` | adds `row <row> <base>` |
| `pkg/game/modmark.go:426`, `:473` | SAV stand-in codec of mod items | unchanged; an added weapon row's items are `mapload.ModItem`s with the base row's same pair as stand-in |
| `pkg/shopstock`, `pkg/shopitem`, `pkg/secondshop` | stock generator of both games, shop item, second-game shop | unchanged |

### Two mods on one row

A field is one number of a shipped row (`price`, `weight`, `defence`,
`absorption`) or one mask bit (`<tier> <material>`) of a shipped or added
row. An added item or weapon row is a new key whose record is one field; a
later mod's `[[weapon]]` of that key replaces the whole row.
Two mods on one field or one key need `load-after`; the later wins and
`againrom -check` names the override. Without `load-after` the launch is
refused. Before this story two changes of one field both applied, the
later's value last, with no message; such a pair of mods is now refused
unless one loads after the other (`DIV-4007`).

### Weapon rows

An added row takes the lowest free weapon row (28 to 31 on a shipped
install). Its parameters are the base row's with the settable columns
replaced; they scale by tier and material like a shipped row's. Its masks
are the base row's, changed by `[[material]]` tables naming its key. Every
pair is named by the mod's `name`; its icon is the mod's `sprite`, else the
base row's icon of the same pair; its figure layers are the base row's of the
same pair. The body list answers the base row's entry for it, so the hero is
drawn and keyed as with the base row; a weapon-bodies record may name the
added row by its key to choose another drawn body.

### Persistence

A pair a mod allows on a shipped row keeps its own 16-bit code in the SAV
(ITEM-224: LOAD tests no mask); the mod mark names the mod set, so a LOAD
without the mod refuses. An item of an added weapon row is a mod item: SAVE
writes the base row's same pair into `F40` and the base row into `T0C`, and
lists the object with its true code and row in the `AgainromMods` leaf; LOAD
under the same mod set writes them back. The default for every old save: a
leaf that lists no such object (every save written before this story, and
every unmodded save, which has no leaf) holds shipped items only.

### Divergences

`DIV-4005` (a material a mod allows), `DIV-4006` (a weapon row a mod adds),
`DIV-4007` (two mods on one item row). `DIV-1861` now points at `DIV-4006`
for weapons. `DIV-4008` is returned unused.

## Proof

- Base census: `TestReleaseShopStockCensusWithoutModsIsShipped`
  (`pkg/game/shopcensus_release_test.go`) digests the candidate pools at five
  ceilings and 64 town-arrival stocks. On engine main `f16bf050` with no mod it
  is `ad00c6b7...e927c6` on both EN and RU, and so it stays on this branch.
- Base unchanged: with no mod, a probe run at main and on this branch wrote
  the War Hammer sessions (World hash, world bytes, mission SAV), a town SAV
  and a digest of every icon and figure layer of every (row, tier, material)
  code (1151 files) on EN and RU, and the War Hammer World on ROM2 EN. All 20
  outputs are byte-equal.
- Fixture mods (`pkg/modrt/testdata/mods`): `bone-pike` allows the common
  bone pike with a synthetic 80x80 picture; `no-iron-pike` forbids the common
  iron pike; `war-glaive` adds a weapon row from the Pike with damage 40-80,
  range 3 and price 900.
- Release tests (`pkg/game/moditemrows_release_test.go`, EN and RU): the bone
  pike joins the candidates, rolls in the town shop, shows the mod's picture
  and name, is bought, worn, saved and loaded; the iron pike never rolls over
  256 seeds at two ceilings; the glaive has the stated numbers and the Pike's
  others, pairs, pictures, body and class key, strikes a person three cells
  away without stepping (the Pike hero steps), and survives a mission SAVE,
  a cold LOAD and the next strike, and a town purchase, SAVE, LOAD and the
  next action; a LOAD without the mod refuses naming it; an unmarked town SAV
  written by main loads without mods and, with `-mods-accept-unmarked`, under
  each fixture; refusals on the install: an allowed pair with no picture, an
  unknown base row, a parameter a mod may not set, two mods adding one key,
  two mods on one mask bit without `load-after`; with `load-after` the later
  wins; a weapon-bodies record names the added row.
- With mission chains and guest parties (`docs/1508/story.md`)
  (`pkg/game/moditemguest_release_test.go`, EN and RU): the guest-chain
  sellsword armed with `Iron war_glaive` holds the added row at range 3 in
  the second link, and a SAVE there marks it with its true row; a cold LOAD
  under both mods restores it, and a LOAD under the chain alone refuses,
  naming `war-glaive`. Items and characters take one load order: a mod
  changing the Pike's price, a definition row's face and a catalog character,
  and a mod loaded after it changing the same three, give the later mod's
  values in either listing order, and the report names both overrides; two
  mods on the Pike's price without `load-after` are refused, while a second
  face edit alone applies.
- Unit tests: `pkg/mod/itemrows_test.go`, `pkg/game/moditemlayer_test.go`.
- Loss control: with the mask merge broken, 5 unit and 3 release tests fail;
  with the added row's settable parameters not written, 1 unit and 2 release
  tests fail.

## Open debt

- A mod cannot give figure layers for a pair it allows on a shipped row; the
  doll paints none where the install has none (`DIV-4005`).
- Parameters 0, 1, 4, 10 and 16 stay the base row's until a reader is found
  (ITEM-219); 5, 12 to 15 are not offered.
- The second game's shop and items take no mod data.
