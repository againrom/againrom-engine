# 0134 — plan

## Approach

The appearance law already exists whole in `pkg/data` and is called with literals. This story
supplies its inputs and gives it one composed entry point, then feeds that entry point from the two
places a party is built and from one new place on the frame-paced path. The resolution that already
turns a shipped person row into a worn set is reused for a generated character, with his own
generated weapon substituted into the row's weapon cell, so he starts wearing his archetype's
clothing and is drawn as the body it composes. The art bundle is rekeyed by directory and name and
filled on demand.

## Facts verified during planning

- `pkg/data/appearance.go` holds the shield suffix, the two substitutions, the sixteen material
  blocks and the directory three-way; `HeroBodyName` has no caller outside its own test. The base
  body name is derived in `pkg/data/equip.go` off equipment slot 1's field D against the body list.
- `PartyBodyDir` in `pkg/game/hero.go` calls the three-way with three literals.
- `pkg/game/frontend.go` loads exactly one body at construction; `LoadHeroBody` writes one entry
  into a map keyed by the body name alone, and every failure is a silent skip.
- `mapload.PartyMember` carries no equipment, and `StartMission` composes a member's whole starting
  `sim.Stock` from his weapon's code alone.
- `mapload`'s `wearRow` reads all ten cells by position, resolves each against the three piece
  collections, applies the two-handed displacement and answers a worn array, a carried slice and
  the weapon pointer; `startingLoadout` applies the two gates past it.
- `pkg/game/world.go` builds `mw.art` once at mission open from each member's `Body`, and
  `entityDraws` is its only reader; `refreshEquipment` rebuilds the inventory figure on the paced
  path off `currentEquipment`, which reads the world's twelve slots.
- `data.ResolveWeapon` strips a trailing `{...}` attachment before resolving.
- `pkg/render/terrain` is a leaf of the import graph and cannot import `pkg/data`. `pkg/game`'s own
  `entrySource` and `terrain.EntrySource` are the same one-method interface.
- The base row search answers a parsed row and a boolean, and no index.

## Design decisions

**D-1 — one composed derivation, in `pkg/data`.** A single function takes the body list, an
equipment set and the class and dying flags and answers the body name, the directory and the drawn
class. *Rejected:* composing the four existing calls at each of the three call sites — the preload,
the party assembly and the refresh — which is exactly how FR-9 would be broken.

**D-2 — the law keeps taking facts, and a new function turns an equipment set into them.** The
directory three-way is not given a slot number; a separate function reads the directory slot off an
equipment set and answers the material index and whether that slot is occupied. *Rejected:* passing
the equipment array into the three-way, reversing 0085's decision that the law consumes facts
rather than slots, for no gain — the slot number is stated once either way.

**D-3 — the bundle key is composed by one exported function.** The directory and the name are
joined by a function in `pkg/data`; the loader writes under it and the driver reads under it.
*Rejected:* a second map keyed by directory. The render tier holds the map and cannot import
`pkg/data` at all, so a second map means a second field on a type in a tier that must not learn
what a hero body is. THE WRITER AND THE READER MOVE IN ONE CHANGE: a landing that rekeys the write
without rekeying the read misses every lookup silently, and FR-8's own no-failure contract is what
would swallow it.

**D-4 — the starting worn set is the base row's own cells with the weapon cell substituted.** The
generated weapon's own literal is written into a copy of the row's ten cells at the weapon cell —
an empty string when he is bare — and the whole ten go through the existing row resolution.
*Rejected:* resolving the nine other cells and then writing the weapon's code into slot 1 by hand.
That would put the shield's two-handed displacement out of the generated weapon's reach and would
be a second place in this tree that decides which slot a weapon goes to.

**D-5 — `mapload.PartyMember` gains four fields.** The worn array and the carried slice, which the
mint composes into the member's `sim.Stock` in place of today's weapon-only composition; the body
directory, because the body name alone no longer addresses a sheet; and the class flag.
*Rejected:* reading the class flag off the member's existing profile flag. That flag's meaning is
this project's own contested reading of which archetype the pool graph names, and binding the
picture to it would make one authored choice decide two unrelated things.

**D-6 — the refresh is its own method on the paced path, with its own tracker.** It compares the
subject's current equipment against the equipment the last derivation ran on and returns before
composing anything when they agree. *Rejected:* folding it into the figure refresh, which already
holds a tracker over the same question. That is a real duplication and it is taken deliberately:
the two write to two different viewer surfaces, and one method owning both is how a later change to
one silently moves the other. Its whole cost is a third copy of twelve sixteen-bit words per paced
call, which allocates nothing. *Rejected:* running it inside the tick beside the combat recompute,
because the drawn body reaches no hashed value and does not belong in the tick's fixed order.

**D-7 — the load is on demand and what is reached is kept.** The loader returns early when the
bundle already holds the key, and a derivation whose art fails is not retried per frame because the
tracker advances first. *Rejected:* preloading every reachable body at construction.

**D-8 — the two party builders are collapsed onto one inputs value.** Both hand the assembler one
struct rather than a positional list. *Rejected:* extending the positional signature to nine
arguments of which three are booleans — the shape in which a caller silently transposes two.

**D-9 — `MissionParty` takes the definition table rather than the humans collection alone.** It
needs the three piece collections to resolve the base row's clothing, and the table already carries
both. *Rejected:* reading the collections off package state.

**D-11 — the mage's weapon is the shipped mage literal `Wood Staff`, and the class axis is a
boolean parameter on the existing resolution.** The name lookup that today takes a trained slot
takes a class flag before it and answers that one literal for a mage whatever the slot; the weapon
resolution it feeds grows the same leading flag rather than acquiring a sibling function, so there
stays exactly one resolution of a generated character's weapon. *Rejected:* a second table indexed
by slot for the mage — the shipped source names one mage literal, and five copies of it would
assert a variation nothing states. *Rejected:* carrying the spell attachment the shipped literal
also names; no weapon in this tree carries a spell, and the attachment is dropped and disclosed.

**D-12 — the refresh writes the drawn class into the map the push already reads.** The one map the
draw path consults for a party member's substituted art is written again, at the member's own
entity id, with the bundle entry the newly derived key resolves to; a key the bundle cannot supply
leaves the existing entry standing. Nothing new crosses the seam and no second lookup is added.
*Rejected:* a second map read by the push, which would give the draw path two places to look and a
precedence rule to get wrong.

**D-13 — the base row search yields the row it found, not only what it parsed.** It answers the
collection index it resolved beside the parsed row, so the equipment cells come off the entry the
search landed on. *Rejected:* a second search by the archetype's own name, which after the search's
own fallback can name a different row from the one that resolved.

**D-10 — the front-end preload stays, over the party's opening body alone, derived through D-1.**
*Rejected:* deleting it and letting the refresh fill the bundle, which would draw the first frame
of every mission through the class record's own art.

## Files to touch

| Path | Intent |
|---|---|
| `pkg/data/appearance.go`, `appearance_test.go` | MODIFY — D-1, D-2, D-3 |
| `pkg/data/chargenbase.go`, `chargenbase_test.go` | MODIFY — D-13 |
| `pkg/mapload/spawn.go` | MODIFY — D-4 |
| `pkg/mapload/start.go` | MODIFY — D-5, and the mint's `sim.Stock` composition |
| `pkg/mapload/startequip_test.go`, `party_test.go`, `start_test.go` | MODIFY |
| `pkg/game/hero.go` | MODIFY — D-8, D-9, D-11, and the derived directory |
| `pkg/game/chargen.go` | MODIFY — D-8 |
| `pkg/game/heroart.go` | MODIFY — D-3, D-7 |
| `pkg/game/frontend.go` | MODIFY — D-10, the preload call alone |
| `pkg/game/world.go` | MODIFY — D-3, D-6, D-12 |
| `pkg/game/hero_test.go`, `chargen_test.go`, `heroappear_test.go`, `heroart_test.go`, `frontend_test.go`, `heropicture_test.go` | MODIFY |
| `cmd/missionrun/main.go`, `cmd/paneldump/main.go` | MODIFY — D-9 |
| `cmd/appearcheck/main.go`, `main_test.go` | ADD |
| `internal/archtest/dag.go` | MODIFY — the new command's row in the fail-closed allow-map |
| the tests of every package whose signatures move | MODIFY — `pkg/mapload/carry_test.go`, `pkg/game/continuity_test.go`, `cmd/missionrun/commandgroup_test.go` |
| `cmd/againrom/main_test.go` | MODIFY — the installed-body helper |

## Risks

- **R-1** A generated character's worn set now reaches hashed world state, so a piece landing in
  the wrong slot moves a digest with nothing to point at. *Mitigation:* one resolution serves a
  party member and a placed person, and the tool prints every archetype's slots on both roots.
- **R-2** An archive read on the frame-paced path could stall a frame. *Mitigation:* the read is
  attempted only when the derived pair actually moved, and never repeated for a pair that failed.
- **R-3** Rekeying the bundle could silently lose every lookup, and a lost lookup is invisible — it
  draws the picture this tree had before the bundle existed. *Mitigation:* D-3's own coupling, plus
  a test that loads two bodies of one name under two directories and reads both back.
- **R-4** A mage's staff carries a spell attachment this tree cannot express, so his blow comes off
  a staff without it. *Mitigation:* disclosed, not approximated; the staff itself is the shipped
  literal and resolves to its own row.
- **R-5** Substituting the weapon cell means a fighter starts holding his trained skill's weapon
  rather than his base row's. *Mitigation:* that is FR-2's own requirement.

## Success criteria

1. Each archetype starts with his base row's clothing in the slots those pieces name, an
   unresolvable or empty cell costing nothing — named tests in `pkg/game` (FR-1; AC-1, AC-2, AC-15).
2. A mage holds the mage's weapon for every skill choice; a fighter holds exactly the weapon he
   held before this story — named tests in `pkg/game` (FR-2; AC-4, AC-5).
3. Slot 1 holds the handed weapon and not the row's, and it is the weapon the member's numbers are
   folded from — named test in `pkg/game` (FR-3; AC-3, P-6).
4. The composed derivation answers a name, a directory and a class for every input, carries the
   suffix, substitutes the mage body, takes the mage and unarmoured arms, and refuses a material
   outside the sixteen — named tests in `pkg/data` (FR-4, FR-5; AC-6..AC-9, AC-16, P-1, P-2).
5. Two bodies of one name under two directories both live in one bundle — named test in `pkg/game`
   (FR-7; AC-17).
6. An equip composing a different body moves the drawn class by the next frame; unchanged equipment
   loads nothing and moves nothing; an unresolvable body leaves the drawn class standing; the open
   and a refresh agree over one equipment set — named tests in `pkg/game` (FR-6, FR-8, FR-9;
   AC-10..AC-12, AC-18, P-3, P-4, P-5).
7. An equip is never refused for the character's class — named test in `pkg/game` (FR-10; AC-14).
8. The tool prints every archetype's worn set, body, directory, sheet address and drawn class for
   both lawful roots, writes no file and exits zero — developer-run (FR-11; AC-13).
