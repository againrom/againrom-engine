# Spec — the party stands in the roster, at the slot the engine indexes it by

**Intensity:** spec-anchored / static. **Terrain:** brownfield — the party placement, the engagement
layer and the map-reading developer tool all ship and all change here.

## Terms

- **A start** is the load path that places the player's party on a map when a mission opens. Every
  entry point that does so is one, and this contract binds all of them.
- **A party member** is an entity a start created for the player. Every other entity in a world was
  placed by the map.
- **Owner slot** is an entity's 1-based position in the map's own type-5 roster. **0 names no roster
  entry.**
- **Hostile** is **bit 0** of the relation cell `[from][to]`. A cell the matrix does not hold reads
  as 0 and is therefore not hostile. The diagonal is forced to 2, whose bit 0 is clear.
- **Sight**, **notice circle** and **reach** are three distances this build already has: what a group
  can see, the frozen circle a guarding group clips its candidates to, and how far a blow carries.
  This story introduces none of them and changes none of them.
- **Both roots** are the two lawful installs, English and Russian. A claim made on one is not made.

## Why

Every engagement decision this build takes is indexed by owner slot, and slot 0 is outside the
relation matrix in both directions.

The units a start places for the player carried slot 0. So the player was not merely a poor target,
he was **outside the diplomacy relation entirely** — no map group could acquire him and he could
acquire nobody, on every map, by construction. The only fights a world could hold were between units
the map itself placed.

The player's slot is not a free choice. The campaign party is the roster's first record, the one
shipped maps name `Self`, and its 1-based slot is **1** — which is already the slot this build's
stance rule names. Nothing about that rule moves; what moves is that the party is now inside it.

## Scope

**In scope.** The roster slot a start's party members carry, everything that follows from carrying
it, and a witness that reads a map's roster and diplomacy without ticking a world.

**Out of scope.** Patrol, roam, withdraw and the six unimplemented group orders; any new arm for a
group under the stance slot 1 already selects; **what group word the original's party units carry**,
which is undecoded; what decides mission 10's outcome; seating a client at any slot but the
campaign's.

## The contract

### The slot

**FR-1** A unit a start places for the player carries **owner slot 1** — the map's type-5 roster
record 0, the entry shipped maps name `Self`. No other field of a party entity changes.

**FR-2** The slot is a **constant of the simulation package**, published there and read by the
placement. It is the same symbol the stance rule already compares a group's owner against, and there
is no second definition of the player's slot anywhere in the tree. It is not derived from a roster
name at load time: a map whose record 0 is not named `Self` is still the player's.

**FR-3** A party member's **group word is 0**, which is the value the field already held. All party
members of one start therefore form exactly one group. Nothing here says what a second start's
members would do, and no such start exists.

### What follows

**FR-4** A party member is a candidate for a map group exactly when the map's own matrix has bit 0
of `[that group's slot][1]`, and a map-placed unit is a candidate for the party exactly when the
matrix has bit 0 of `[1][that unit's slot]`. **Neither direction implies the other.**

**FR-5** No entity's owner slot other than a party member's changes, and no other field of any
entity changes. A world with no party is unaffected field for field.

**FR-6** The party's group takes the stance its owner slot already selects: it strikes what stands
within reach and takes **no step** toward anything further. A party member is given no order against
a unit slot 1 is not hostile to, including one that is striking it — there is no retaliation arm.

**FR-6a** A party member that acquires a target has its standing walk **dropped**, which is what
every entity in this build does on acquiring one. So a single move order issued while a hostile
stands within reach is overridden on the next decision, and a re-issued order moves the unit
normally — it reaches the same cell it would have reached at slot 0. This is a disclosed
consequence of the slot, not a rule authored here: the original branches inside this very arm on
whether the unit belongs to a human participant, and that branch is undecoded.

**FR-10** The front end is told which roster slot the local participant holds, and it is told 1.
Without this a party member's damage numeral would flip its drift direction, because that comparison
reads true today only by both values being zero. The numeral's **colour** is keyed on the owner slot
alone and does move: the player's units take slot 1's colour rather than the unowned colour.

### The byte form

**FR-7** The canonical byte form's **version and field set are unchanged**: the owner slot is
already a field of an encoded entity. A world carrying a party encodes and decodes with the slot
preserved. The digest of that byte form differs from the digest the same world had while the party
stood at slot 0 — a value change, not a form change.

### The witness

**FR-8** The developer tool that already reads one `.alm` and prints what it holds gains a **mode**
printing, per roster record: its **1-based slot**, its name, how many placed units name that slot as
owner, its sixteen raw relation words, and the **effective** matrix row — word `k` narrowed to its
low byte at column `k+1`, then the diagonal forced to 2. It ticks no world and reads no registry.
The effective row is **re-derived from the file by that published rule**, not read out of the
loader: it is a reading of the map, and it witnesses the map rather than the loader.

**FR-9** That mode writes no asset and embeds no install path.

## Acceptance criteria

| | Given | When | Then |
|---|---|---|---|
| **AC-1** | any start that places party members | the start returns | every party entity's owner slot is 1 |
| **AC-2** | the same | the same | every party entity's group word is 0 |
| **AC-3** | the tree | it is searched for a definition of the player's slot | exactly one exists, exported by the simulation package, and the placement writes that symbol rather than a literal |
| **AC-4** | a map whose roster makes slot `k` hostile to slot 1, a party member in sight of a group at slot `k` and inside its notice circle | a decision is taken | that group's members are ordered onto the party member |
| **AC-5** | a map whose roster makes slot 1 hostile to slot `k`, a unit of slot `k` in sight and within reach | a decision is taken | the party member is ordered onto it |
| **AC-6** | a map whose roster makes slot `k` hostile to slot 1 but **not** the reverse, the two standing adjacent | a decision is taken | the slot-`k` group is ordered onto the party member and the party member is given **no** order |
| **AC-7** | a party member and a unit slot 1 **is** hostile to, further away than reach | a decision is taken | the party member is given no order and does not move |
| **AC-7a** | a party member given a single move order past a hostile within reach, and the same member given that order repeatedly | both are stepped | the single order is dropped and the member stays; the repeated order carries it to the cell a slot-0 member reaches |
| **AC-8** | a world holding no entity at slot 1 | it is loaded and stepped | it behaves exactly as it did before this story, entity field for entity field |
| **AC-8a** | a started mission whose map places a slot-1 unit nearer a hostile group than the party | a decision is taken | the group takes the nearer one, and the party member is the other candidate rather than absent |
| **AC-9** | mission 10 of a lawful install | its roster and matrix are read by the mode of FR-8 | `Rogues` -> `Villagers` carries bit 0, and so does `Villagers` -> `Rogues`; the two clubmen are owned by `Rogues` and the female-mage NPC by `Villagers` |
| **AC-10** | a world carrying a party | it is encoded and decoded | the round trip is byte-identical, the version byte is the one that already shipped, and every party entity's owner slot survives — asserted on the entity, not on the version byte |
| **AC-11** | two worlds differing in nothing but one entity's owner slot, including the pair 0 and 1 | both byte forms are hashed | the digests differ, while a pair differing in nothing at all is equal |
| **AC-14** | a viewer driven by a started mission | the world is installed | it is told the local participant holds slot 1, so a party member's damage numeral keeps the drift direction it had |
| **AC-12** | any `.alm` of a lawful install | the mode runs on it | it prints one line per roster record carrying slot, name, owner count, the sixteen raw words and the effective row, and exits 0 — a document with no roster record prints the header, no line, and exits 0 |

## Error cases

**AC-13** On a file that is not a decodable `.alm`, the mode fails through the shared error path the
tool's other modes already use — a message on standard error and a nonzero exit — and prints no
partial table.

## Success conditions

**SC-1** On mission 10, a party member ordered to a cell beside the `Rogues` clubmen is
**intercepted**: the drive stops short and spends its whole tick budget, and a commanded blow on the
nearer clubman lands at once because that clubman has closed on the party — where the same drive on
a tree without this change arrives, and the same blow has to walk. Measured on **both roots**: a
drive reads the install's own registries and definition table, not the map file, so byte-identical
maps do not make one root's run the other's.

**SC-2** The mission-10 outcome and its tick are recorded on **both roots**, before and after. No
test expectation, threshold or predicate is edited to alter that outcome — falsifiable from the
story's own diff.

**SC-3** No shipped map hands a group over to a player by naming group word 0, so fixing the party's
group word at 0 collides with nothing that ships. Measured over both roots.
