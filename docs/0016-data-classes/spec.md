# Spec — typed data classes from the graphics registries

## Problem and current state

`pkg/formats/reg` parses a `.reg` registry into typed nodes that report presence, but nothing above
knows what a registry *means*: three registries in `graphics.res` define every unit, structure and
static object the game can place, and a placed `.alm` record names its class by a number nothing here
resolves. This story delivers that layer — typed classes, inheritance and per-key defaults, `ID`
lookup, sprite paths, and a tool resolving every class shipped maps place.

## Source contract

| registry | class sections | `[Global]` |
|---|---|---|
| `units/units.reg` | `[Unit0]`…`[Unit33]` | `UnitCount` = 34, `FileCount` = 33 |
| `objects/objects.reg` | `[Object0]`…`[Object81]` | `ObjectCount` = 82, `FileCount` = 56 |
| `structures/structures.reg` | `[Structure0]`…`[Structure65]` | `Count` = 66 |

Names match case-insensitively over ASCII — the fold `pkg/formats/reg` applies. Class sections are
**dense**: every index in `[0, count)` exists. `[Files]` holds backslash-separated, extensionless
sprite paths at `File0`…`File<FileCount−1>`; `structures.reg` has neither `[Files]` nor `FileCount`,
each structure carrying its own path. On-disk child order is name-sorted, **not** section order —
`Unit0, Unit1, Unit10, … Unit2, …` — so ordering by index is a real sort.

Every class carries an int `ID`, and `ID` — never the section index — is how a placed record and a
`Parent` name a class. The domains differ and none may be assumed from another: units `1..80`, 34
distinct and **sparse**; objects `0..81` = the section index; structures `1..66` = index + 1. A
map's stored class number is used **directly**: a type-4 `kind` (low 16 bits) is a `structures.reg`
`ID`, a type-6 class field a `units.reg` `ID`, a nonzero type-3 `code − 1` an `objects.reg` `ID`.

## Inheritance

`Parent`, **when present**, holds the `ID` of another class in the same registry and the child
inherits from it. Presence alone decides, never the value: `Parent = 0` is a real reference to the
class whose `ID` is 0, so reading a zero `Parent` as "no parent" is wrong. Resolution is **eager, at
load, per key** — the nearest ancestor that sets a key supplies it, and a loaded class keeps no
reference to its parent.

**The guard differs between scalar and array keys — the trap of this contract.**

- A **scalar** (`int`, `str`) is inherited **iff the child's own section lacks the key**; a child
  that writes it keeps its own value, a literal `0` included, **overriding** the parent. Scalars
  **chain**: a grandchild sees the parent's already-resolved value.
- An **array** (`[]int`) is inherited **iff the child's own read yields length 0** — absent, empty
  string or zero-length array alike; length, not presence, is the whole test. Arrays do **not**
  chain: only the parent's *own* section is read, so one the parent itself inherited leaves the child
  nil.

Inheritance MUST be decided from the registry node's presence (scalars) or resolved length (arrays),
never from a loaded field's zero value.

`File` is inherited like the other scalars in `units.reg` but **not** in `objects.reg`, never taken
from an ancestor there (*Absent-everywhere defaults*). Structures never inherit: no structure
carries `Parent`, and a `Parent` on one is malformed.

Depth is unbounded — any acyclic chain resolves — but a **forward** `Parent`, whose named parent's
section index is at least the child's, is malformed: the engine would read an unpopulated slot. One
forward pass over `0..count-1`, resolving by `ID` among the classes already built, therefore
suffices.

## The empty-string sentinel

An array-valued key may be stored as a **zero-length string** — the editor's "none" marker — and it
is exactly an absent key: length 0, therefore inherited if `Parent` is present. **There is no
representation of "clear this array"**: an explicitly empty value falls through the length guard and
the parent's array is taken. With **no inheritable ancestor** a length-0 array resolves to **nil**,
full stop — narrower than the engine, which leaves its buffer untouched.

## Absent-everywhere defaults

A key no section on a class's chain sets does not resolve to zero. Each `objects.reg` int key has its
own default: **`0`** for `InMapEditor`, **`-1`** for every other one the engine loads — `ID`, `File`,
`Parent`, `Index`, `Phases`, `Width`, `Height`, `CenterX`, `CenterY`, `DeadObject`, `FireObject`. `0`
is a legal object `ID`, so the two are not interchangeable: an unset `DeadObject` or `FireObject`
reads `-1` and never names the class whose `ID` is 0, and an unset `Parent` reads `-1` where `0` is a
real edge. Nothing else defaults: an absent `DescText` is `""`, an absent array nil, and `IconID`,
which the engine does not load, `0`.

`File` also **does not inherit** here: a class whose own section omits it takes `-1` whatever an
ancestor holds, and a `File` at its default names no sprite, both path methods answering `""`;
*Validation*'s bound is about a `File` a section states. `units.reg` and `structures.reg` have no
default table here — a key absent on the whole chain resolves to its type's zero, and what the engine
defaults it to is not established.

## Key inventory

Every key is `int` unless marked `str` or `[]int`; the sets below are complete. Apart from `ID`,
`Parent`, `File`, `DescText` and the structure geometry keys in *Validation*, a key's **meaning is
unconfirmed**: the labels are the game's own spellings, what a value does is not. A comment may say
what a key is *called* and MUST NOT assert what it does.

**Units** (34 classes, 37 keys): `ID`, `File`, `DescText` str, `Sound` []int, `InfoPicture` str,
`AttackAnimTime`/`AttackAnimFrame` []int, `AttackDelay`, `InMapEditor`, `Dying`, `AttackPhases`,
`Palette`, `DyingPhases`, `MoveAnimTime`/`MoveAnimFrame` []int, `Index`, `MovePhases`,
`MoveBeginPhases`, `Width`, `Height`, `CenterX`, `CenterY`, `SelectionX1`/`X2`/`Y1`/`Y2`, `Parent`,
`BonePhases`, `ShootOffset` []int, `Flip`, `Projectile`, `ShootDelay`, `TileSize`, `IdlePhases`,
`IdleAnimTime`/`IdleAnimFrame` []int, `Z`. Only the first four are on every class.

**Objects** (82 classes, 16 keys): `ID`, `File`, `DescText` str, `InMapEditor`, `Index`, `Phases`,
`Width`, `Height`, `CenterX`, `CenterY`, `Parent`, `DeadObject`, `IconID`,
`AnimationTime`/`AnimationFrame` []int, `FireObject`. Only `ID` and `File` are on every class; seven
carry **no `DescText` and no `Parent`**, a missing `DescText` being no error and a class's identity
its `ID`.

**Structures** (66 classes, 23 keys). On all 66: `ID`, `DescText` str, `File` str (a direct sprite
path), `TileWidth`, `TileHeight`, `FullHeight`, `SelectionX1`/`X2`/`Y1`/`Y2`, `ShadowY`, `Phases`,
`Picture` str, `AnimMask` str, `AnimTime`/`AnimFrame` []int. Partial: `Indestructible`, `IconID`,
`Usable`, `Flat`, `LightRadius`, `LightPulse`, `VariableSize`.

## Sprite paths

The stored path is backslash-separated and extensionless; every backslash becomes `/` and case is
preserved (archive lookup folds it). The sprite entry is `units/` or `objects/` + `Files[File]` +
`.256`, or `structures/` + the structure's own `File` + `.256`; the overlay entry is the same with `b`
before `.256`. Construction opens nothing and checks nothing: **an absent sprite file is valid
shipped data, not a failure**.

## Validation

A malformed registry yields a **nil** collection and an error naming the section and key (P-1).
Malformed is any of: a missing or non-int `[Global]` count; a missing dense section; a duplicate
`ID`; a `Parent` unresolvable, cyclic, forward or on a structure; a `File` index outside
`[0, FileCount)`; a missing or empty `[Files]` entry at a referenced index; a known key of the wrong
kind, a non-empty string for an array included; a `ShootOffset` of length other than 16; a non-empty
`AnimMask` whose length is not `TileWidth × FullHeight`; a paired animation key without its partner
or of a different length. Unknown keys are ignored.

## API requirements

- **FR-1** `pkg/data` MUST expose `LoadUnitClasses`, `LoadStructureClasses` and `LoadObjectClasses`,
  each taking a parsed `*reg.Reg` and returning a typed collection or an error, performing no IO and
  importing only `pkg/formats/reg` and the standard library. Each class MUST carry every key of its
  registry's inventory as a field, loaded **verbatim**, with inheritance resolved per *Inheritance* —
  decided from the registry node, never from a loaded field's zero value — and a key absent on the
  whole chain per *Absent-everywhere defaults*.
- **FR-2** Each collection MUST expose `ByID(id) (*Class, bool)` over its sparse `ID` domain, `All()`
  in **numeric section-index order** of length the `[Global]` count, and `SpritePath()`/`OverlayPath()`
  per *Sprite paths*.
- **FR-3** `cmd/classdump` MUST print the three collections for a given `graphics.res`, and with
  `-sweep <dir>` MUST resolve every placed class reference in every `.alm` under `<dir>` — loose
  files and archive entries alike — reporting per placement kind how many resolved and how many did
  not. It MUST exit non-zero when a type-4 or type-6 reference fails to resolve. A nonzero type-3
  code resolving past the registry MUST be counted and printed but MUST NOT fail the run: such cells
  ship and whether the engine reads them is undecided. An absent sprite file MUST not affect the
  outcome. A record whose `DefID` is nonzero and `!= 0xcdcdcdcd`, or whose `Flags` bit 0 is set, the
  engine resolves through another table: it MUST be counted and reported separately and MUST NOT
  count as a failure.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a parent setting `AttackDelay = 4`; one child writing `0`, one omitting the key | loaded | the writer resolves to **0**, the omitter to **4**; a zero-value test fails |
| AC-2 | unit | a parent with a 7-element array; children that omit the key, store the empty string, store a zero-length array, store 3 elements | loaded | the first three take the parent's 7, the fourth keeps its 3; nothing clears an array |
| AC-3 | unit | a grandparent setting an array and a scalar, neither set by parent or child | loaded | the child's scalar is the grandparent's (chaining), its array is **nil** (one hop) |
| AC-4 | unit | `Parent = 0` naming the class whose `ID` is 0 | loaded | the child inherits from it; an absent `Parent` inherits nothing |
| AC-5 | unit | one fixture per malformed case named in *Validation* | each loaded | each yields a nil collection and an error naming the offending section and key; no panic, no partial |
| AC-6 | unit | sparse `ID`s, sections in name-sorted node order | loaded | `ByID` hits every `ID` and misses every non-`ID`; `All()` is numeric section order of length `[Global]` |
| AC-7 | unit | classes of all three kinds, one path with several backslashes | `SpritePath`/`OverlayPath` | prefix, forward slashes, preserved case, `.256` and the `b` before it are exact; no archive opened |
| AC-8 | unit | a class with no `DescText`, one whose sprite entry is absent, one with an unknown key, every inventory key at its own kind | loaded | the first two load as values not errors; the unknown key is ignored; every key round-trips **verbatim** |
| AC-9 | manual | the owner's install | `classdump`, `classdump -sweep` | 34 / 82 / 66 classes print; every type-3 and type-4 reference resolves but for the counted out-of-range residual; every type-6 one resolves or diverts |
| AC-10 | unit | an object parent setting `DeadObject`, `Width`, `InMapEditor`; a child omitting all three; a class on no chain | loaded | the child takes the parent's three; the loner reads `-1`, `-1`, `0`; a written `DeadObject = 0` still resolves to `0` |
| AC-11 | unit | an object and a unit class, each with a `Parent` whose section sets `File` | loaded | the object's `File` is `-1` with both paths empty; the unit inherits its parent's `File` and path |

## Derived properties

- **P-1** (atomicity) A load returns a complete collection and a nil error, or a nil collection and
  an error naming a section and key. Never a partial collection or a panic.
- **P-2** (verbatim) Every field equals the registry value at that key, the value the resolving
  ancestor's key held, or the key's default (P-6). No arithmetic is performed on a registry value.
- **P-3** (enumeration) `All()` is numeric section order of length `[Global]`, and a class is
  reachable by `ByID` iff it appears in `All()`.
- **P-4** (guards) For a scalar present in a class's own section the resolved value is that class's
  own, `0` included; for an array of own length 0 it is the parent's own section's value, or nil when
  there is none.
- **P-5** (purity) `SpritePath` and `OverlayPath` are pure string functions; loading does no IO.
- **P-6** (defaults) A field whose key no section on the chain sets equals that key's
  *Absent-everywhere defaults* value, not the type's zero — and `objects.reg`'s `File` whenever its
  own section omits it.

## Out of scope

Key semantics beyond the names; combat stats; `.pal`, sprite and frame decoding; the projectile and
material registries; sounds; `patch.res` layering; any change to `.reg`/`.alm` decoding.
