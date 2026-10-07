# Provenance — the player's own units can fight

Every research fact below is cited by **claim id** at the pin recorded in the submodule (research
`20921e2`). No experiment folder is cited: a claim carries its own amendment and retraction state, an
experiment cannot tell you it has been superseded.

## What the contract rests on

| id | confidence | what this story takes from it |
|---|---|---|
| `HERO-DERIVE-034` | High | The whole fold. `spread = ftol(1.1^Body/20)`; `base` copied from it; `toHit = ftol((1.1^Body + 1.1^Reaction)/5)`; then, gated on the active skill being positive, `toHit += 3*skill` and `base += skill/5`; then the modifier fold. The Body term enters the minimum once and the maximum twice, the skill term the base alone. `__ftol` truncates toward zero. |
| `HERO-STATDMG-036` | High | Only Body reaches damage; only Body and Reaction reach to-hit. This is what licenses a fold with no Mind or Spirit term — a positive read census inside the sole writer, not an absence. |
| `HERO-FOLD-035` | High | The equipment fold is eight plain `ADD`s and one assignment: **no multiply anywhere from an item to a damage number**. It also fixes the complete writer set of the damage pair, which is what makes the one-routine census decisive. |
| `HERO-BARE-037` | High | An unarmed hero is the same routine with the active skill cleared, so `base = spread = ftol(1.1^Body/20)` and the roll is `[d, 2d]`; zero below Body 32. There is **no hidden bare-hands weapon** — the shipped `BareHands` row is named by no instruction. This is what makes a nil weapon a modelled state rather than a hole. |
| `HERO-START-039` | High for the dispatch and the ten literals / **Medium** for "a shipped campaign takes the skill-10 arm" | The five ordinary fighter literals by skill slot, and the five of the high arm. The story takes the **ordinary** arm and carries the Medium as DD-6. |
| `HERO-SKILL-009` | High | Six skill slots; slot order `General, Blade, Axe, Bludgen, Pike, Shooting`; chargen zeroes 1..5 and writes exactly one, at 10 on the ordinary arm. |
| `HERO-BUY-003` | High | The chargen start: all four stats **25**. |
| `HERO-CAP-015` | High | Step 0 of the recompute: `stat = min(stat, 50 + (int8)modifier)`, the 50 an immediate, not class-conditional. Every hero this tree builds has a zero modifier, so the cap is a flat 50. |
| `HERO-COMBAT-011` | active (partly retracted) | `defence = reaction / 3`, truncating. Its "degenerate range" clause is the retracted half and is not used. |
| `HERO-EQUIP-017` | High | `Weapon::Equip`'s melee arm: `+= w+0x60` into the damage base modifier, `+= w+0x61` into the spread, `+= w+0x52` into the to-hit modifier, `+= w+0x6a` into the defence modifier, and the active skill **assigned** the weapon's own kind. The unequip inverse restores the bare cadence `8 / 4`. |
| `HERO-ARMOUR-018` | High for the field identification | Absorption is the block's second word and its source is **armour**, through a modifier no weapon routine writes. A hero handed only a weapon therefore has absorption 0. |
| `HERO-FOLD-033` | High | The `+0xbe` block fold, of which this story uses the defence word. Its damage-kind half is out of scope. |
| `ITEM-LADDER-019` | High | `f64[j]` sits at `record + 0x20 + 8j` of a `0x68`-byte Shapes or Materials record, fixed by an immediate rather than by a fit. Our nine-double record begins at `+0x20`, so `f64[j]` is bytes `8j .. 8j+8` of it. |
| `ITEM-DMGFACT-020` | High | `@.damage` is `f64[4]`; a factor is the **product** of the shape's and the material's; `Common` is 0.2000 and `Iron` 1.0000; the shipped `Iron Short Sword` resolves to `(5, 3)` with `+0x52 = 5`, `+0x6a = 0`, `+0x50 = 1`, identical on both roots. **This story re-derives those figures from the install rather than adopting them** — see `analysis.md`. |
| `ITEM-DMGCOL-018` | High for the encoding / refuted for its worked values | The item carries `(base, spread)` and **not** `(min, max)`: the fill's second byte subtracts the first, re-reading it as an already-rounded unsigned byte. Only the encoding half is used; the refuted worked values are not. |
| `ITEM-WEAPCOL-021` | High | The `Weapons` slot map, anchored four times over: slot 5 `@.attackType`, 6/7 the physical pair, 8 `@.toHit`, 9 `#.deIrnce`, 0xb `@.range`, 0xc/0xd `@.charge`/`@.relax`. And that `Weapon::Equip` has three arms on slot 5, the melee one being `< 0xa`. |
| `ITEM-SCALE-017` | High for the fill and the name parse / refuted for the old ladder | The fill writes `w+0x60` from column 6 and `w+0x61` from column 7 both through the `@.damage` double, `w+0x52` from column 8 through `f64[5]`, `w+0x6a` from column 9 through `f64[6]`, and `w+0x50` from column 0xb verbatim or 1 when it is -1. The name parse is a leading shape word, then a leading material word, then the rest as the Weapons row; the ten chargen literals resolve 10/10 on both roots. |
| `ITEM-CLASS-001`, `ITEM-DEF-002` | High | A weapon's definition is a row of the `Weapons` collection, chosen by the C++ class rather than by a kind field. |
| `HERO-AUTOHIT-031` | High for the skip / Medium for the writer | Read and **not** applied: the bit's second effect (skipping absorption) is `pkg/sim`'s resolver, and a hero never carries the bit. Recorded as an open divergence rather than silently satisfied. |
| `HERO-SHEET-038` | High | The character sheet prints `[base, base + spread]`. Used only to make the owner's four bands comparable with this tree's two numbers; nothing is implemented from it. |

## What is OURS by choice, and where the seam is

- **Which skill the party's hero trained.** `PartySkillSlot = 1`, the Blade slot, so the weapon is
  the `Iron Short Sword`. Chargen is a screen this tree does not have, so the slot cannot be read
  from anywhere; 1 is chosen because it is the slot the owner's own measurement was taken on, which
  makes his four bands a live check rather than a coincidence. Every other slot resolves too.
- **How a weapon's name is split.** The claim states that a name is *a leading shape word, then a
  leading material word, then the rest*, and does not state how the search matches. `Uncommon Magic
  Wood Short Bow` proves the material may be more than one word. Ours is **longest matching prefix on
  a word boundary, table order breaking ties**. It is stated as ours in `spec.md` DD-3, and its whole
  falsification is that the five ordinary literals resolve against the shipped file, which
  `verification.md` measures.
- **`math.Pow` for `1.1^n`.** The image calls the CRT `pow` on the double `0x3FF199999999999A`. Two
  `pow` implementations may differ by an ULP, which would matter at a truncation boundary — so the
  margin over the whole legal stat range is asserted by a test rather than assumed. It is 8.98e-3 at
  worst on the damage term and 2.05e-4 on the to-hit term.
- **Refusing a ranged weapon at the decode boundary.** The `0xb`/`0xc` equip arms feed a second
  damage component this tree does not model. Rather than fold half of it, the resolver refuses such a
  row by name — the same rule `NewUnitDef` already applies to the damage-routing arms it does not
  model, and for the same reason: loudly wrong beats quietly wrong.
- **A carried failure, not a fatal one.** A definition table that does not yield the starting weapon
  leaves the front end with a nil weapon and the reason beside it, exactly as `FontErr` is kept. The
  hero is then *bare*, which is a real state of the original and not an error state of ours — and the
  headless check line says so, so it cannot pass unnoticed.

## What this supersedes

`pkg/mapload/start.go`'s `PartyMember` comment — *"nothing here knows what a hero is: a party that
fought at numbers this tree invented would be a fabrication"* — was correct when written and is
retired by this story, which does not invent the numbers but derives them. The replacement states
what a hero now IS and what he still is not.

`pkg/formats/databin`'s *"Its slots are undecoded, so the bytes are kept and not read as numbers"* is
retired for the nine-double record alone: `ITEM-LADDER-019` decodes the ladder, and the bytes are
still kept.

## What is NOT claimed

- That a hero's **health** is 100. It is not; `HERO-HP-005` derives roughly 60 at the chargen start.
  `SpawnHP` stands, disclosed, out of scope.
- That a hero's **speed** is 10, or his **reach** one cell. Both are derived elsewhere and untouched.
- That the campaign takes the ordinary chargen arm. That clause is research's **Medium** and is
  carried as such; the owner's sheet agrees with it and is testimony, which is a question and not a
  verdict.
- That the Shapes and Materials tables are named `Common`/`Iron` at any particular index. Nothing
  here matches a shape or material by an expected name: the resolver reads the name the *weapon
  literal* carries and takes index 0 when the name carries no shape word, which is what the item
  constructor's own default does.
