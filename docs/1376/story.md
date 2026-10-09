# One actor constructor

## Intent

Every actor the engine creates is built by one constructor, `sim.NewActor`.
Before this story six origins each wrote their own `sim.Entity` composite
literal, so a field added in one was absent from actors made by another. The
census below tabulates what each origin set on main and from which source. The
refactor keeps the hashed World state byte-identical; the two defects the
census found are disclosed, not fixed.

Base: `5d22d8cc` (game 0.102.0). Owner direction: one builder per kind, and the
architecture audit of `c70591f1`, section 8, accepted before the ROM2
character generator line, which adds a seventh origin (a hero born in a town).

## Authority

| Part | Claims |
|---|---|
| Control Spirit raise: row `Ghost`, seven corpse stores, the row supplies the rest | MAGIC-249 (High / Medium), MAGIC-SING-019 (c) |
| no difficulty on the raised actor | MAGIC-250 (Medium) |
| dying time fetched from the actor's own Units or Humans row | DAT-SCHEMA-007 (High) |
| dying countdown is a floor on the dwell | HERO-DWELL-065 (High) |
| siege hires are Units-table creatures | MERC-LEVEL-005 |
| authored current health applied verbatim | UNIT-PLACE-034 |

## Census

Measured on main `5d22d8cc` by reading each origin's literal. Origins:
**M** map placement (`fromalm.go`; the cheat spawn goes through it, then
through the keyed basis), **P** party member (`start.go`), **S** siege hire
(`siegehire.go`, used by `start.go` and the SAV writer), **G** Control Spirit
raise (`sim/spell.go`), **F** scenario fixture (`game/scenario.go`), **L** SAV
load seed (`sourcebinding.go`). "block" is the resolved Units or Humans row at
the mission difficulty (`blockFor`); "fold" is the party member's derived
sheet (`PartySpawnWithTable`); "tmpl" is the world's ghost template; "corpse"
is the raised body; "spec" is the authored unit; "row" is the saved binding's
definition row; "-" is not set (zero).

| Field | M | P | S | G | F | L |
|---|---|---|---|---|---|---|
| ID, X, Y | record | start cell | start cell (caller) | corpse cell | spec | - |
| Owner | record | player slot | player slot | caster | spec | - |
| Group | record | - | - | caster | spec | - |
| MapUnitID | record | saved record | saved record (caller) | - | - | - |
| Facing, DesiredFacing | - | - | - | corpse | - | - |
| Class, TypeID | block | member class, hero or hire type | block | tmpl | - | binding |
| Humanoid | block | true | block | tmpl | - | binding class |
| Domain | block | ground | block | tmpl | - | - |
| HP | block or record override | restored or fold pool | block | corpse max/2 | spec | - |
| MaxHP | block | restored or fold pool | block | corpse max/2 | spec | - |
| Mana, MaxMana | block | restored or fold pool | block | - | spec | - |
| Health and mana periods | block | restored or constructor default | block | - | - | - |
| Health and mana regeneration | block | fold | block | - | - | - |
| Speed, Capacity | block | fold | block | tmpl speed; - | - | - |
| RotationSpeed | block | fold | block | tmpl | spec | row |
| ScanRange | block | fold sight | block | tmpl | spec | - |
| SeeInvisible | block | - | block | - | - | row |
| Reach, TokenSize | block | fold reach; 1 | block | tmpl, 0 read as 1 | - | - |
| DyingTime | block | - | block | tmpl | - | row |
| Withdraw, Wimpy | block | - | - | tmpl | - | row |
| ToHit, Defence | block | fold | block | corpse | - | - |
| Absorption, damage pair, attack charge and relax, AlwaysHits | block | fold | block | tmpl | damage base, AlwaysHits: spec | AlwaysHits: row |
| SecondBase, SecondSpread | - | fold | - | - | - | - |
| SecondaryDamage | block | fold | block | - | - | - |
| Protection, Resistance | block | fold | block | tmpl | - | - |
| Weapon spell, level, source | block, rule A | fold, rule B | block, rule B | - | - | - |
| KnownSpells, Book | block | member plus taught | block | - | known: spec | - |
| CreatureSpells | block | - | - | - | - | - |
| AutoSpell | - | - | - | - | spec | - |
| XPValue | block | - | block | tmpl | - | row |
| Reaction, Spirit | block | fold | block | corpse | - | - |
| Mind | block | fold | block | corpse | spec | - |
| XPSlot | block | fold | block | tmpl | - | - |
| GainsXP | block | person band | block | - | - | person band |
| Gold chance, treasure pair | block | - | block | - | - | row |
| Skill, SkillXP | block | fold, carried experience | block | - | - | - |
| SuppressCorpseLoot | block | member | block | - | - | row name |
| NativeBasis | block | initial modifier or carried | block, or carried | tmpl | - | - |
| NativeClass | block | fighter flag or carried | -, or carried | - | - | - |
| NativeTraining | roster person | member levels | - | - | - | - |

Rule A: the row's spell; the worn weapon marks it as the item's only when the
item casts that same spell at that same power. Rule B: a worn weapon that
casts replaces the row's spell and power and marks it as the item's.

### Differences and their reading

| Difference | Reading |
|---|---|
| P, S, L: no Group | Deliberate. A party is one group; a seed takes its group from the saved record |
| P: no SeeInvisible, Withdraw, Wimpy, XPValue, treasure, CreatureSpells | Deliberate. The person arm of a block sets none of them either; the death-gold roll is gated above the person band |
| **P: no DyingTime** | **Defect, finding D1**, below |
| M, S: no SecondBase pair | Equivalent. A definition row never yields one; only a saved Human attack block does. NewActor now copies the block's zero |
| S: no Withdraw, Wimpy, NativeClass, CreatureSpells | Omission. All four are zero on both shipped siege rows (Catapult and Ballista, EN and RU). The siege hire now takes the whole block definition, so a mod row's values reach the hire |
| S rule B, M rule A | Equivalent on shipped data: both shipped siege rows give the same spell, level and source under either rule, and 0 of 8094 EN and 0 of 3991 RU map placements carry a worn weapon whose cast differs from the entity's spell. The siege hire now follows rule A, the block's rule; the party keeps rule B, since a member's worn weapon can be any item |
| **G: no regeneration, mana, see-invisible, capacity, skills, book, creature spells, treasure, secondary damage** | **Defect, finding D2**, below |
| F: only the authored fields | Deliberate. A scenario states what it measures |
| L: only row policy fields | Deliberate. The SAV loader overlays the saved record's fields |

### Finding D1: a fresh party member has no dying time

Player effect: a party member who falls in a mission started from the town
takes a dwell of 0 (no dying countdown), while the same member after a SAV
and load takes his Humans row's dying time. Measured with the release install
on both roots: the chargen hero row and the hired rows read hold 12 or 8; the
minted member holds 0. The fresh and the loaded World differ in hashed state.
DAT-SCHEMA-007 reads the dying time from the actor's own row. Disclosed as
DIV-2692 (FIDELITY-DEBT, OPEN). The fix changes the hashed state of every
fresh mission with a party and is left to a separate result.

### Finding D2: a raised ghost lacks most of its row

Player effect: a Control Spirit ghost never regenerates health, cannot see
invisible actors and has carrying capacity 0. On both roots the `Ghost` Units
row holds health period 200, see-invisible 2, capacity 300 and general skill
40; the raised actor carries 0 for each. MAGIC-249 says the row supplies every
stat the corpse does not (Medium). The ghost template is persisted in the
current-world policy of a save, so carrying more row columns changes save
content as well as hashed state. DIV-037's implemented behaviour now names the
columns not taken.

## As built

- `sim.ActorDefinition` holds every field a source states about a new actor;
  `sim.ActorPlacement` holds the id, cell, facing, owner, group and map unit
  id. `sim.NewActor` (`pkg/sim/actorconstructor.go`) is the one function that
  writes an actor `Entity` literal; it copies every field and sets the desired
  facing equal to the facing.
- Origin-specific values arrive as data. Each origin resolves its source into
  a definition: `spawnBlock.actorDefinition` for a resolved row (map
  placement, siege hire, and through the map path the cheat spawn); the party
  loop for a generated member; `raisedGhost` for the template with its corpse
  stores; `HeadlessWorldSpec.Build` for an authored unit; `SourceActorSeed` for
  a saved binding's row policy.
- The brief's four constructor inputs map as follows. The definition row and
  the difficulty enter through the origin's resolver (`blockFor` applies the
  difficulty). The placement or party entry is `ActorPlacement` plus the
  member's own data. The key and runtime ID enter at
  `mapload.ConstructActorBasis`, the one keyed stage, which the SAV writer,
  the cheat spawn and the siege hire call after `NewActor`; it needs the
  install tables, which `pkg/sim` does not see, and a mission-start actor
  receives its key when a save writes it.
- `Entity.standAtPost` is the guard-at-own-cell state the world constructor
  applies to every initial actor; the raise applies it to the actor it
  appends.
- `carriedNativeHistory` writes a carried member's native basis and class
  into the definition before construction.
- `internal/archtest` ratchets `sim.Entity` composite literals in non-test
  files: the seven origin files hold none, and every other file's count may
  only fall (`actorliteral_baseline.go`,
  `go run ./internal/archtest/cmd/actorliteral`). An empty `Entity{}` is a
  lookup's zero value and is not counted. The remaining sites are probe
  entities handed to a rule function or a one-actor scratch world, witness
  tool fixtures, and the snapshot decoder.

## Proof

- `TestNewActorCarriesEveryDefinitionAndPlacementField` fills every input
  field with a distinct nonzero value and fails if `NewActor` drops one,
  renames one or sets a field no input names.
- Per-origin witnesses compare every definition and placement field against
  the census source: `TestMapPlacementActorCarriesItsCensusFields`,
  `TestSiegeHireActorCarriesItsCensusFields`,
  `TestPartyActorCarriesItsCensusFields`,
  `TestSavedUnitSeedCarriesItsCensusFields` (`pkg/mapload`),
  `TestRaisedGhostCarriesItsCensusFields` (`pkg/sim`, whole Entity), and
  `TestScenarioFixtureActorCarriesItsCensusFields` (`pkg/game`). Each states
  enough nonzero sources that a dropped field fails it.
- Byte identity, by a differential dump against base on both roots: for
  every shipped map at difficulties 1 to 3, the ALM world hash, the started
  world hash with a six-member party (fighter and mage heroes, both siege
  types, two hired types) and a digest of every entity; both siege hire SAV
  bases; 1536 `SourceActorSeed` bindings; the cheat spawn of five names.
  EN and RU dumps are identical to base. The headless scenario hashes are
  unchanged on EN and RU.

## Open debt

- D1 (DIV-2692) and D2 (DIV-037).
- The party definition is still built in the party loop rather than by a
  resolver shared with a seventh origin; the ROM2 town-born hero will take one
  of the existing resolvers or add its own, and call `NewActor`.
- The `PartyMember` template, items, sacks and structures each still have
  several builders (audit addendum); out of scope.
