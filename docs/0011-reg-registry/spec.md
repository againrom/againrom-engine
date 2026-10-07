# Spec — `.reg` binary registry parser (ROM1)

Static game data — unit, structure, object, projectile and material class definitions, scenario and
AI parameters, and the cutscene fade scripts — lives in binary `.reg` registries stored as entries
inside `.res` archives. Nothing in this repository decodes them yet. They gate the typed game classes
(`pkg/data`) and, through those, real sprites on the map. Greenfield.

This story delivers the pure parser and a developer dump tool. It assigns **no meaning to any
individual key**: `DescText`, `AttackPhases`, `Files` and the rest are opaque names to this layer.

## Format definition

All integers are little-endian. A `.reg` byte stream is:

```
header (0x18) | node table (nodeCount × 0x20) | heapSize (u32) | heap (heapSize bytes)
```

### Header (0x18 bytes)

| off | type | field | contract |
|---|---|---|---|
| 0x00 | u32 | signature | MUST be `0x31415926` — the same signature the `.res` archive container uses |
| 0x04 | u32 | rootFirst | node-table index of the root's first child. MUST be dereferenced, never assumed zero: it is a pointer, not a constant |
| 0x08 | u32 | rootCount | number of root children |
| 0x0C | u32 | rootFlags | the root's own `kind` word, read as the bitfield below — bit 0 marks a directory, bit 4 marks its child list as name-sorted. `17` in every shipped registry. A parser MUST NOT reject on its value |
| 0x10 | u32 | nodeCount | total nodes in the table, at all depths |
| 0x14 | u32 | reserved | read and ignored |

The header's first 16 bytes have the same shape as a node's first 16 (reserved / data / size /
kind), and `0x10`–`0x17` occupy the slot where a node keeps its name. The root directory
therefore has **no on-disk node of its own and no name**; the header is its record.

### Node (0x20 bytes each)

| off | type | field | contract |
|---|---|---|---|
| 0x00 | u32 | reserved | a real field, not name padding — the format's own writer zeroes it explicitly, and it is 0 in every shipped node. Read and ignored |
| 0x04 | u32 | data | per type, below |
| 0x08 | u32 | size | per type, below |
| 0x0C | u32 | kind | a bitfield — value type plus flags, below |
| 0x10 | char[16] | name | at most 15 significant characters, cut at the first NUL |

A name carries **at most 15 significant characters** plus a NUL: the format's own writer clamps a
longer name to 15 and records the truncation in the kind word's bit 28. A parser MUST nevertheless
tolerate a name that fills all 16 bytes with no NUL, taking it whole rather than overrunning into the
next node — defensive engineering against a stream that writer could not have produced.

### Kind

The `kind` word is a **bitfield**, not an enumeration:

| field | meaning |
|---|---|
| `kind & 0x0E` | the value **type** — the table below |
| bit 0 (`0x01`) | the node is a **directory**; its type bits are 0 |
| bit 4 (`0x10`) | this node's child list is **name-sorted** |
| bit 28 (`0x10000000`) | the name was longer than 15 characters and was **truncated** |

A parser MUST mask the type out before comparing it, and MUST NOT reject a node merely because a flag
bit is set: a directory whose children are sorted carries `kind == 17`, exactly as the root does. In
every shipped registry only the root sets bit 4 and nothing sets bit 28, but neither is a guarantee.
Bit 0 decides directory-versus-value first; a node that sets it carries type bits of 0 in every shipped
registry, and this spec assigns no meaning to bit 0 combined with any other type, which is therefore a
rejection **in a node's `kind`**. The header's `rootFlags` is not a node kind and is never a rejection
ground, as its row above says.

| type | bit 0 | meaning | `data` | `size` |
|---|---|---|---|---|
| 0 | clear | string | byte offset into the heap | byte length including the trailing NUL; the value is cut at the first NUL |
| 0 | set | directory | node index of the first child | number of children — a contiguous node-table range |
| 2 | — | int32 | the value itself, signed | ignored |
| 4 | — | float64 | the low 32 bits | the high 32 bits |
| 6 | — | int32 array | byte offset into the heap | byte length; MUST be a multiple of 4; elements are little-endian int32 |
| 10 | — | float64 array | byte offset into the heap | byte length, a multiple of 8; elements are little-endian float64 |

A **type-4** value is the little-endian IEEE-754 binary64 formed by the eight bytes at node offset
`0x04` — that is, the `data` and `size` words read as one double. It needs no heap access.

**Type 10 is decoded but deliberately not implemented by this story** (see *Out of scope*): no shipped
registry contains one. A parser built to this spec rejects a type-10 node as *unsupported*, naming the
type and node index — it does not guess, and it does not silently skip.

**Type 8 is undecoded**, and every remaining type value is one the format's own reader does not handle
either. A parser MUST reject the stream, naming the offending type and node index, rather than assign a
meaning to it.

### Text

Names and string values are **byte strings**. The format defines no character encoding, and the original
engine applies none: it reads the node table and the heap verbatim and hands a string value's bytes on
unchanged. Its only byte-level transform anywhere on the path is the `A`–`Z` fold its name lookup uses
when comparing.

A parser built to this spec therefore **decodes nothing**. It returns names and string values as the
bytes the stream holds, in order, unmodified. Mapping those bytes to characters is a **presentation
choice**, belonging to whatever displays them, and the component that makes it MUST state it (FR-3)
rather than leave it implied. CP866 and CP1251 are both defensible renderings; neither is the format's,
and naming either one "the registry's encoding" would attribute to the format a property it has not got.

No shipped registry exercises the question: every name and every string value in every shipped registry
is pure ASCII, the observed byte range being `0x20`–`0x7A`.

### Heap

`heapOrigin = 0x18 + 0x20 × nodeCount`. A u32 `heapSize` sits at `heapOrigin`; the heap is
`[heapOrigin + 4, heapOrigin + 4 + heapSize)`. `heapSize == 0` (a registry holding no string or array
value, hence no heap bytes) is valid and MUST parse.

A parser MUST require every type-0 and type-6 reference `[data, data + size)` to lie inside the heap.
It SHOULD NOT require the stream to end exactly at the heap end — trailing bytes are tolerated, as
the `.res` reader tolerates its own unread header words — even though every shipped registry does end
exactly there.

### Validation

Every rejection is atomic: nil result, a wrapped error, no panic, no partial tree.

- signature; minimum lengths for the header, the full node table, and the `heapSize` word;
- directory ranges: `first + count ≤ nodeCount`, evaluated so the addition cannot overflow;
- type-6 `size % 4 == 0`; heap bounds for types 0 and 6; an unsupported type (10) or an unrecognised
  one (8, and any remaining value), each naming the type and the node index;
- a node kind that sets bit 0 over non-zero type bits — a directory that claims a value type.

Otherwise a set flag bit is never grounds for rejection: validation reads `kind & 0x0E` for the type and
ignores every other bit. The header's `rootFlags` is exempt from all of this and is never validated.

Three further rules are **this project's own defensive engineering, not claimed properties of the
game's own writer**, and no shipped registry exercises any of them:

- recursion depth is capped at 32 — a cycle guard against the same flat-table hazard `.res` has;
- a node referenced by more than one directory range is rejected;
- a node referenced by no range (an orphan) is ignored rather than rejected.

## Functional requirements

- **FR-1 (parse)** — `pkg/formats/reg` exposes `Parse(data []byte) (*Reg, error)`, building a tree of
  typed nodes from a byte slice. No IO, no global state, no engine knowledge. The package sits at the
  lowest tier: **standard library only**. It performs no character conversion (see *Text*), so it takes
  no text-encoding dependency.
- **FR-2 (lookup)** — typed two-level accessors `GetString`, `GetInt`, `GetIntArray`, `GetFloat`
  taking `("Section", "Key")` and returning `(value, bool)`; a missing section, a missing key, or a
  key of the wrong type returns `false`. Matching is case-insensitive over the ASCII range only —
  `A`–`Z` fold with `a`–`z`, every other byte compares as itself. **This matching rule is our own API
  convenience.** It coincides with the original engine's own lookup, which folds the same range, but the
  spec claims no fidelity beyond the coincidence and one difference is known: the engine compares only a
  name's first 15 characters, while these accessors compare whole names. No registry the format's own
  writer produced can distinguish the two, since that writer clamps names to 15 characters. The full
  parsed tree stays exported so tools can walk depths and types the two-level accessors do not reach.
- **FR-3 (dump tool)** — `cmd/regtool` gains `dump <archive.res> <entry.reg>`, printing the tree as
  indented `name = value` (strings quoted, arrays abbreviated past 8 elements, doubles printed as
  numbers), and `sweep <dir>`, which walks every `*.res` in a directory, parses every `*.reg` entry,
  prints per-entry node counts, and ends with `N regs: N parsed, 0 failed`. Because names and string
  values are bytes (see *Text*), `dump` **states and applies one display convention**: bytes
  `0x20`–`0x7E` print as themselves, every other byte prints as `\xNN`. It is the only component that
  maps registry bytes to characters, and it maps them to no code page. Developer-run against a lawful
  install; never part of the test suite.

## Acceptance criteria

Every unit-level fixture is a byte stream built in test code. No test reads a game install.

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a synthetic registry using all five implemented node kinds (string, directory, int32, float64, int32 array), with nested directories, one of them carrying the sorted-children flag (`kind == 17`) | parsed | the exact tree: names, values, array elements, nesting; the flagged node is a directory, not a rejection |
| AC-2 | unit | a type-4 node whose `data`/`size` carry a known IEEE-754 bit pattern, including a non-zero mantissa low word | parsed | the corresponding `float64`, exactly |
| AC-3 | unit | truncated stream; bad signature; node table shorter than `nodeCount` | parsed | nil + error each, no panic |
| AC-4 | unit | a directory range past `nodeCount`; a cyclic and a self-referencing range; a `first + count` that overflows | parsed | bounds error / depth-guard termination; never a hang |
| AC-5 | unit | type 6 with `size % 4 != 0`; a heap overrun for type 0 and for type 6; a type-10 node; a type-8 node | parsed | an error each; the type-10 and type-8 errors name the type and the node index |
| AC-6 | unit | a name and a string value carrying bytes in `0x80`–`0xFF`; separately a name filling all 16 bytes with no NUL | parsed | the bytes come back **unchanged**, byte for byte, with no character mapping applied; the 16-byte name is kept whole. Fixture and expected bytes are written as hex, never as literal non-ASCII text |
| AC-7 | unit | a parsed tree | each `Get*` | case-insensitive hit; wrong-type miss; missing-key miss; missing-section miss; a non-directory used as a section misses rather than panics |
| AC-8 | manual | a lawful install | `regtool sweep` over the install root | all **44** `.reg` entries parse with 0 failures — 33 cutscene registries across `VIDEO4.RES`/`VIDEO8.RES`, 5 in `graphics.res`, 3 in `scenario.res`, 2 in `world.res`, 1 in `sfx.res`. Spot-checks that a wrong field mapping could not pass: `units.reg` `[Global] UnitCount == 34` and `FileCount == 33`; `[Unit0] DescText` reads as coherent text; `[Unit0] AttackPhases` equals the element count of `[Unit0] AttackAnimFrame`, and `MovePhases` the count of `MoveAnimFrame`; a cutscene registry's `startfade`/`endfade` pair reads exactly `0.0` and `1.0` |
| AC-9 | unit | a registry with `heapSize == 0` and no type-0/type-6 nodes | parsed | parses cleanly; no error |
| AC-10 | unit | a parsed tree whose name and string value carry bytes outside `0x20`–`0x7E` | rendered by `dump` | printable ASCII appears verbatim and every other byte as `\xNN`; no code page is applied and the output is valid UTF-8 |
| AC-11 | unit | the shipped `pkg/formats/reg` package | the repository's architecture import check | it imports the standard library only — no text-encoding package, no other module package |

## Derived properties

- **P-1** (negative-invariant) `Parse` never panics and never returns a partial tree on error.
- **P-2** (invariant) Every heap read stays inside `[heapOrigin + 4, heapOrigin + 4 + heapSize)`.
- **P-3** (completeness) Walking the tree of a registry with no orphans yields exactly `nodeCount`
  reachable nodes, each reached once.

## Out of scope

- Writing or editing `.reg` streams — read-only this story.
- The meaning of any individual key or section (`Files`, `UnitCount`, `AttackPhases`, …) and the typed
  game classes built from them — `pkg/data`, a later story.
- **Value type 10 (float64 array).** It is decoded — heap offset in `data`, byte length in `size`,
  `size/8` little-endian doubles — but it occurs in no shipped registry, so this story implements no
  decoder and no accessor for it and a type-10 node is rejected as unsupported. Documented and
  deliberately unimplemented; the gap is scope, not ignorance.
- **Value type 8 and every remaining type value.** Undecoded. An unrecognised type is a rejection,
  never a guess.
- Archive layering and `patch.res` override order — the VFS story.
