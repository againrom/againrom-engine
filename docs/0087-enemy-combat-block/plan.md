# Plan — one more arm on the resolution, and no second copy of the arithmetic

The two collections are not two spellings of one thing. One carries its combat block and derives
nothing; the other carries no combat block at all and derives everything. So the shape is a **second
definition type beside the first**, a **second block builder beside the first**, and one place in
this package where the two meet — the composite literal that fills an entity.

## Design decisions

### DD-1 — `data.Collection` widens by one method, rather than a second interface beside it

FR-3 needs an entry's trailing strings, and the interface this tier reads a collection through
exposes only the name and the parameters. Two candidates:

- a second, narrower interface the humans arm type-asserts its way to. Rejected: the assertion folds
  "this table carries no equipment" onto "this table's type does not admit any", and FR-3 has to tell
  those apart;
- widen `Collection` with `EntryStrings(i int) []string`. **Taken.** A `Data.bin` entry IS a name, a
  parameter array and a run of trailing strings.

Cost is exact: the format tier gains a one-line accessor and the three test fakes gain a one-line
method each. It is a compile error at every site, which is the failure mode to want.

### DD-2 — `HumanDef` lives beside `UnitDef` and shares its cursor, not its slot map

`pkg/data/humandef.go`, next to `unitdef.go`. It reuses `slotCursor` verbatim — the one place the
empty cell is tested and the only thing that advances — so the two loaders cannot come to disagree
about what an empty cell does, which is the single law both collections share.

It does NOT share the slot map or the field set. `NewHumanDef` is its own ascending `switch` over
slots 0..22, written out slot by slot in the shape `NewUnitDef` has, so a diff between the two files
is a diff between two decoded maps rather than between two abstractions.

Rejected: a table-driven loader over a shared slot map. It would make the two maps look like data
when their differences are structural — one routes a pair through a selector, the other copies a
skill array — and the selector arm has no counterpart to be a row of.

### DD-3 — the empty-cell fallback is the base actor's, and it is stated as ours

`UnitDefaults()` is what the other arm falls back to, and it is the base actor constructor's value.
Nothing read for this story says the humans object's constructor agrees with it and nothing says it
differs. `NewHumanDef` therefore starts from the same defaults value, via `UnitDefaults()` rather
than a second literal, so there is one statement of those numbers in the tree.

The reuse is a FIELD-BY-FIELD COPY and not a conversion — the two structs differ, which DD-2 is
about — so the protection it buys is weaker than "one statement of those numbers" and is stated
honestly: it is one SOURCE, read at one site, so the two arms cannot drift apart silently; it is not
a compile-time guarantee, and a field added to the base value with no line here would be missed. The
test that catches that is the all-empty row's, whose expected value is written out by hand rather
than read back off the value the loader itself reads.

Two of the base value's fields have no counterpart on this arm at all — the two regeneration periods
— and the type simply does not carry them, so the copy cannot pass them on.

Rejected: a separate `humanCtorDefaults` literal. It would be the same numbers with no source behind
the second copy, and the empty-cell law they stand under IS shared even where the values are not.

Rejected: refusing an empty cell. Shipped rows leave `TokenSize` and `MovementType` empty routinely
and some leave every statistic empty; refusing them would refuse the collection.

### DD-4 — the derivation is a method on `HumanDef`, in `pkg/data`

`func (d HumanDef) Combat(w *Weapon) Combat`, beside `Hero.Derive` and **defined in terms of it**:
it builds the `Hero` this definition's statistics and skills are, derives through the one routine
that owns that arithmetic, then applies the one thing a template has and a character does not.

Deriving in `pkg/mapload` was rejected: it would put the combat graph in the tier that wires rather
than the tier that owns it, and the constants it needs — the bare cadence pair and the empty-cell
value — both live here.

`func (d HumanDef) Hero() Hero` is exported beside it: "what character is this row" is a question the
front end will ask for a panel and it should have one answer.

The two post-derive steps, and each is FR-4's or FR-2's:

- **the cadence.** `Derive` writes the hero's bare pair when there is no weapon, which is not right
  for a row shipping its own two columns; this method writes the template's pair and then lets the
  weapon assign over each half under DD-10's one predicate.
- **nothing else.** Absorption, the always-hits mark and the two damage halves come out of `Derive`
  exactly as they are.

### DD-5 — `Table` grows the three item collections

`mapload.Table` gains `Shapes`, `Materials` (as `data.ScaleTable`) and `Weapons` (as
`data.Collection`), for the reason the other three collections are there: what a placement is worth
is a property of the installed files. A nil field is "no collection", and one accessor beside the
existing three keeps a nil `*Table` answering rather than panicking.

Rejected: reading them in `pkg/mapload` — this tier opens nothing. Rejected: passing a resolved
weapon per placement from `pkg/game` — the front end would be resolving equipment for a map it has
not loaded, and the loader would have two callers that could disagree.

### DD-6 — one block value, three arms, one literal

`FromALMWith`'s loop keeps its structural completeness argument and extends it. A private
`spawnBlock` carries what a resolution is worth — the health, the domain, the rate and a
`data.Combat` — and one function `blockFor(u, t, diff)` produces it down three arms:

- **units**: today's `definitionFor`, then `UnitDef.Combat()` — a new accessor returning the eight
  it already carries, so both arms hand back one type;
- **humans**: the entry the resolution reached, `NewHumanDef`, the weapon search, `HumanDef.Combat`;
- **neither**: today's behaviour, `UnitDefaults()` and `SpawnHP`, unchanged.

The entity literal then names every RESOLUTION-DERIVED field once off that one value. Six fields in
it are not resolution-derived and MUST NOT be routed through the block: the id, the two coordinates,
the class key, the group and the owner all come off the placement record, and the owner in
particular is carried at that one site precisely so that "an entity carries an owner exactly when a
map placed it" is true by there being one assignment. Routing them through a resolution would break
that.

This keeps FR-7's completeness claim structural — a resolution field left unfilled is one a reader
can see is absent — and it is why the block is a value rather than a helper that writes into an
entity.

The difficulty is applied on the units arm alone, inside its own arm, exactly where it is today. FR-6
is then true by there being no call rather than by a test of the setting.

`domainFor` reads the movement code off the creature definition and its comment makes the
load-bearing claim that one condition decides the domain. The person arm needs the same mapping off a
different type, so the switch is lifted into `domainForCode(int32)` and both definitions reach it —
a twin switch is exactly what that comment forbids.

### DD-7 — the weapon search takes the first cell that resolves, and swallows the rest

`firstWeapon(names []string, t *Table) *Weapon` walks the cells in order and returns the first that
`data.ResolveWeapon` accepts. A cell that fails is skipped without a word: FR-3 says a name that is
not a weapon row is armour or a shield, and the resolver's refusal is exactly how "not a weapon row"
is detected. The refusal text is therefore not an error to report — it is the search's own predicate.

The cost is named in BOTH directions. A genuinely malformed weapon name is indistinguishable from a
helmet and the person comes out bare; and a trailing cell that is not equipment at all but happens to
equal a weapon row's name arms him, with no way left to notice. The alternative — reporting the first
failure — would refuse every armed row in the collection, because every armed row's second cell is
armour.

An empty cell is skipped before the resolver sees it, so a row's trailing empties cost nothing.

### DD-8 — `pkg/game` wires the three collections in the one walk it already does

`LoadDefinitions` already reads `Shapes`, `Materials` and `Weapons` to arm the party's hero, out of
one walk of one file. The same three go onto the table it returns, so an install cannot answer the
hero and not the placements.

### DD-9 — `pkg/sim` is not touched at all

FR-8 is met by there being no diff under `pkg/sim` — stronger than a test on the version literal,
which only says the number is still the one this story was written beside.

## Risks

- **R-1 — the digest of every world holding a map-placed person moves.** That is FR-8's own point,
  and it is bounded: the pinned pre-story digest in `pkg/mapload/fromalm_test.go` is taken over the
  single-argument entry point, which resolves nothing, so it must stay green untouched. If it moves,
  something outside this story's arm moved. **The pin is the detector, and it needs no new test.**
- **R-2 — the widened interface is a compile error at four sites.** Wanted; listed so a reviewer
  sees the three test-only ones are not scope creep.
- **R-3 — a person comes out weaker than he is today.** Sixteen of mission 1's thirty-five
  placements carry a health of 100 that nothing decoded supports, and several rows' columns are far
  below it. A brigand getting harder to kill was never the claim; his numbers becoming his own is.
- **R-4 — the weapon search runs once per equipment cell per placement, at load.** Linear over three
  collections; a map places tens of people and the walk is once per world, not once per tick.
- **R-5 — four existing tests encode the behaviour this story reverses, and three of them are in a
  file no task would otherwise open.** A humans-arm placement is currently one of the three shapes a
  test calls "resolves to nothing", and two more assert that such placements keep the provisional
  health. They are the story's own evidence that the defect was real, and they are REPAIRED rather
  than deleted — each keeps its original question and asks it of the population that still resolves
  to nothing. The repair must not neutralise the two numbers the derivation now produces before
  comparing, which is how such a repair goes quietly wrong.
- **R-6 — a typed nil in an interface field is not a nil interface.** The three item fields are
  interfaces, and the format tier hands back a typed nil pointer for a collection id out of range, on
  which the presence test would pass and the first call would panic. The one producer passes named
  in-range ids, so it is not reachable in this tree; it is the reason the scenario-NPC field beside
  them is a concrete type, and it is recorded rather than guarded because a guard would have to call
  a method on the value it is guarding against.

## Success criteria

- **SC-1 → FR-1, FR-2, FR-4, FR-5** A humans-arm placement carries its row's health, rate and
  cadence and a derived blow band, off the one routine the hero already uses.
- **SC-2 → FR-3** A row's own weapon reaches its placement, chosen by resolving rather than by
  position; a table missing any item collection arms nobody without failing.
- **SC-3 → FR-6, FR-7** The other two arms and all three difficulties are where they were, and the
  table-less entry point is unmoved to the byte.
- **SC-4 → FR-8** `git diff` names no file under `pkg/sim`, and the digest that DOES move is
  witnessed by lifting this story's fields back out rather than by a number pasted from a run.
- **SC-5 → AC-14** Both lawful roots place two brigands who carry their own entry's health and a
  band whose top is above zero.
