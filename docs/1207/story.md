# 1207 — typed constructors for sim.Command

## Result

`sim.Command` is unchanged field for field, and nothing outside
`pkg/sim/command.go` writes one as a composite literal any more. Every command
in the tree's production code is now built by a constructor named for its kind:
`MoveTo(entity, at)`, `Attack(attacker, victim)`, `CastAt(caster, spell, at)`,
`Equip(actor, item, slot)`, `SetPlayerParameter(player, parameter, value)` and
twenty-one more.

| population | before | after |
|---|---|---|
| production literals outside `pkg/sim` | 42 | 0 |
| production literals inside `pkg/sim` | 8 | 26, all in `command.go` |
| test literals | 936 | 936, under a falling ratchet that counts 867 |

Player-visible behaviour is unchanged, and so is every byte of every command.

## Intent

Owner direction: the command architecture is right and the concrete type is a
good wire and replay form, but it had stopped being a good API. `X`, `Y` and
`Spell` mean different things per kind, so a literal can carry a cell where an
entity id belongs and compile. Keep the compact value, forbid manual literals
from outside, and create commands through typed constructors, so determinism
and compatibility survive while the compiler catches a confused `X`, `Spell` or
`Player`.

The overloading measured on the base commit, each meaning proved by its reader
in `pkg/sim`: `X` carries a cell coordinate, an entity id, a structure handle, a
damage amount, an equipment slot, a container element, a spell id, a standing
order byte and a parameter selector — NINE meanings. `Spell` carries a spell id,
an equipment slot, a container element and a scroll's carried index — four.

## As built

### The types

Four named types are added and three existing ones are reused.

| type | underlying | what it names |
|---|---|---|
| `SpellID` | `int32` | a spell row id as a command carries one |
| `ItemSlot` | `int32` | an element of an entity's own container |
| `EquipSlot` | `int32` | an equipment slot, the original's 1..12 |
| `PlayerParameter` | `int32` | opcode 0x46's selector sub-code |
| `EntityID` | existing | a unit handle |
| `StructureID` | existing | a building handle |
| `CellPoint` | existing | one map cell |

The cell parameter is `CellPoint`, the type `CellEffect` already reports cells
in, rather than a new one: a second cell type in one package would make a cast's
target and the cells that cast paints two different things for no reason a
reader gains from.

`SpellID` is `int32` and not the `uint16` a spell row is, because the field a
spell rides in depends on the kind — `Spell` for `KindCastAt`, `Y` for
`KindCast`, `X` for `KindAutocast`. Narrowing in the type would answer the
autocast arm's own question about an id too wide for a row, which is its to
answer. `int32` keeps every call site byte-identical: `uint16(int32(v))` equals
`uint16(v)` for every integer `v`.

`Group` gets no type. It is the one field that is not overloaded: every kind
that reads it reads the same correlation tag, so a name would add a conversion
and catch nothing.

`Player` keeps `uint32`. Retyping it reaches `Entity.Owner`, the relation table
and every ownership comparison in two packages, which is a larger change than
this one and is listed as debt below.

### The constructors

Twenty-six functions for twenty-five kinds; `Equip` and `EquipDisplacing` are
the one kind with two, because the second equipment slot is absent at two of its
three call sites. Each is pure field assignment: no validation, no clamping, no
defaulting, no reordering. Every conversion is the arm's own, written as the arm
reads it back — `Attack` sets `X: int32(uint32(victim))` because
`EntityID(uint32(c.X))` is what undoes it.

### The two seams that took a kind as data

Two production sites built a command from a kind held in a variable, which no
per-kind constructor can replace.

`pkg/game`'s `queueGroup` now takes a builder, `func(tag uint32) sim.Command`,
instead of a kind and two loose numbers. Its four callers — `enqueue`, `stance`,
`march` and `defend` — each name their own group constructor. The order is built
twice: the first build is a probe whose kind and aim the membership test reads,
and only the tag differs. Building is pure, so the probe disturbs nothing.
`march`'s own local kind variable is split into two branches.

`cmd/missionrun` gains `standingCommand`, which switches a parsed standing
order's three kinds onto their constructors. Its false arm reports an order the
tool cannot build rather than issuing the zero command, which is a move; that
arm is unreachable, because `parseStanding` admits exactly those three kinds.

### The guard

`internal/archtest` gains `LoadCommandLiterals`, in the idiom
`LoadComposition` already uses: a syntactic walk of the module, resolving
element types elided inside `[]sim.Command` literals, which is the form most of
the test population takes and the one a text search misses. Two rules:

- **Production is an absolute zero.** A non-test file outside
  `pkg/sim/command.go` that writes a `sim.Command` composite literal fails the
  build.
- **Tests are a falling ratchet**, committed in
  `internal/archtest/commandliteral_baseline.go` at 867. A rise fails; a fall
  requires rewriting the baseline in the same commit. Regenerate with
  `go run ./internal/archtest/cmd/commandliteral`.

The walk is syntactic and resolves an elided element type one level deep. The
review's type-resolved count is 936 test literals on base and head; the other
69, mostly `[][]sim.Command` tables, are invisible to the ratchet. It also
misses an elided map key, a named slice type, a type alias and a dot import.
The production zero holds under full type information.

`pkg/sim/command_test.go` is exempt from the test ratchet for the opposite
reason to every other test file: its literals are the specification, not debt.

### Changed beyond the brief

`AttackUnit` and `AttackStructure`, the two values of the `AttackTargetKind`
field, are renamed `AttackTargetUnit` and `AttackTargetStructure`. The old name
collided with the constructor for `KindAttackStructure`, and the new names say
what the constants are values of. It is a Go identifier rename over 49 sites in
twenty files; `encoding/gob` matches persisted struct FIELDS by name and these
are constants, so no save format moves.

## Proof

**The constructors equal the literals they replaced.**
`pkg/sim/command_test.go` holds each constructor against the command written out
field by field for the same arguments, sweeps every narrowing conversion over
fourteen boundary values from `-2147483648` to `2147483647`, round-trips six
handle values including `4294967295` through `X`, and fails if a kind constant
has no constructor.

**The corpus is byte-identical on base and head.** `cmd/missionrun` was built
from base `798e62cf385f9a0ce5f55e9a11959f4955864f0a` and from this branch, and
104 runs compared:

- 56 mission traces — 28 missions on the EN root and the same 28 on RU, each
  `-trace -ticks 400`. All identical, but they stop at tick 64 and issue no
  command, so they reach no constructor and are not evidence for this change.
- 48 driven runs over the paths this story rewrote: `-order` for the four group
  verbs, `-waypoint`, `-attack`, `-census`, `-withdrawal`, `-mage`, `-wear` and
  `-take`, over six missions. All identical.

**The script census did not move.** Mission 10 and mission 20 report zero
`UNSUPPORTED` script nodes on base and zero on head. Their script populations
are `16 checks, 27 instants, 12 triggers` and `14 checks, 15 instants, 11
triggers`, equal to the `en m10` and `en m20` rows of
`pipeline/milestone-baseline.txt`. This story was not meant to move that census
and did not.

**The guard discriminates.** A `sim.Command{Kind: sim.KindMoveTo}` placed in a
temporary `pkg/game` file fails `TestLiveCommandLiteralsMatchTheirBaseline`
naming its file and line, and a two-element `[]sim.Command{{...}, {...}}` fails
it twice, which is the elided form a text search misses. The tree is clean again
after each probe.

**Gates.** `gofmt` clean; `go build ./...`; `go test -trimpath -count=1 ./...`
green over the whole module; `scripts/check-no-game-assets.sh` clean;
`pipeline/check-preserved-installs.sh` ok over 554 files; the EN and RU release
tests through one `pipeline/check-release-tests.sh` invocation, 320 of 320 gated
tests on each root; `pipeline/check-scenarios.sh` 53 of 53 on each root.

`scripts/check-milestone2-acceptance.sh` was not run. It is RED on main for a
reason that predates this branch, and nothing here touches an original save.

`internal/storyguard`'s tree-wide comment-byte baseline rises 12000 bytes over ten
files. Six are new; `pkg/game/world.go` and `pkg/game/consumables.go` are
comments on functions this commit rewrote, `cmd/missionrun/main.go` carries
`standingCommand`'s doc, and the tenth is `internal/storyguard/baseline.go`'s
own note, which names all ten.

## Open debt

- **936 test literals remain, 867 by the guard's count.** They are ratcheted, not migrated. Migrating them
  is mechanical and large, and it would bury this change's real diff.
- **`Player` is still a bare `uint32`.** A `PlayerID` type would catch a roster
  slot passed where an entity id belongs, but it reaches `Entity.Owner`, the
  relation table and every ownership test; it is its own story.
- **`GroupStance`'s order is a bare `int32`.** The two values a caller may pass
  are `OrderGuard` and `OrderStandGround`, which are already exported `int32`
  constants aliasing the engagement pass's own bytes. Typing them would retype
  those aliases and the pass behind them.
- **`queueGroup`'s probe build is a small oddity.** It exists because the
  membership test reads the command's kind and aim before the tag is known. A
  version that split the tag decision out of the append would not need it, at
  the cost of a second method on a seam with four callers.
