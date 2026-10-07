# Plan — `.reg` binary registry parser (ROM1)

**Intensity:** spec-anchored / static (the `PROFILE` default for `pkg/formats/*` — the format spec is
the durable contract). **Terrain:** greenfield — `pkg/formats/reg/doc.go` and `cmd/regtool/main.go` are
story-0000 skeletons, not an implementation, so there is no shipped `.reg` behaviour to preserve.

Reading key: `FR-x` / `AC-x` / `P-x` → `spec.md`. This file fixes the package boundaries, the API
contract, every design decision and the success criteria the work is verified against. It is derivable
from the spec alone.

## Approach

Three layers, each decidable without a game install.

1. **`pkg/formats/reg` (new code in an existing skeleton package, stdlib only)** — `Parse(data []byte)
   (*Reg, error)`: header, node table, heap, an exhaustive per-node validation pass and a bounded tree
   walk, producing an exported tree of typed nodes; then the four two-level accessors over it
   (FR-1, FR-2, AC-1…AC-7, AC-9, P-1, P-2, P-3).
2. **`cmd/regtool` (skeleton replaced)** — `dump <archive.res> <entry.reg>` and `sweep <dir>`, the
   developer-run tool (FR-3, AC-8, AC-10). Its rendering is a pure function over a `*reg.Reg` writing to
   an `io.Writer`, so the whole display convention is unit-testable with no archive and no install.
3. **Fixtures and the architecture gate** — a `.reg` byte-stream builder in the existing
   `internal/synth` helper package, and the `internal/archtest` change that makes AC-11 an *enforced*
   property rather than a documented intention.

Two mechanical facts shape the decomposition. `pkg/formats/reg` performs no character conversion, so it
takes **no** text-encoding dependency, and AC-11 pins that — but the shipped import check permits
`golang.org/x/text` to every `pkg/formats/*` package, so AC-11 is not enforced today and this story must
make it so (DD13). And `cmd/regtool`'s DAG row currently permits `pkg/formats/reg` alone, while FR-3
requires the tool to read a `.reg` entry out of a `.res` archive — so the row gains `pkg/formats/res`
(DD12). Both are `internal/archtest` edits; the check is fail-closed and authoritative, so
`docs/ARCHITECTURE.md` is edited alongside it.

No `go.mod` change: `golang.org/x/text` is already required by the module for `pkg/formats/res`, and
this story neither adds nor removes a dependency, so `THIRD_PARTY_NOTICES.md` and `internal/notices` are
untouched.

## Facts verified during planning (baseline, frozen)

**The shipped tree.**

- `pkg/formats/reg` holds only `doc.go` — a package clause plus a tier comment reading "will implement
  the .reg registry format" and "(plus golang.org/x/text for CP866 string decoding)". No code, no tests.
- `cmd/regtool/main.go` is a skeleton whose `main` prints `regtool (skeleton)`; it has no test file.
- `internal/archtest`'s allow-map lists `"pkg/formats/reg": {}` and `"cmd/regtool":
  {"pkg/formats/reg"}`. It is **fail-closed** on unregistered packages, and `archtest.Load` skips
  everything under `internal/`, so a helper there needs no allow-map row.
- `externalAllowed` grants `golang.org/x/text` (and its subpackages) to **every** package whose path
  begins `pkg/formats/`. So nothing in the shipped check would reject `pkg/formats/reg` importing it,
  and nothing enforces the documented stdlib-only status of `pkg/formats/spr256` either. **Nothing
  checks `docs/ARCHITECTURE.md` against the allow-map**; the doc lags or leads by review alone.
- `internal/synth` is a stdlib-only, unpoliced test-helper package that already builds `.res` archives
  (`synth.Archive`), Windows BMPs and `.alm` maps. Its package doc states its own rule: it builds
  **inputs only**, never an expected output. It is imported from `_test.go` files only.
- `pkg/formats/res` exposes `Open(path) (*Archive, error)`, `(*Archive).Entries() []Entry` with
  `Entry{Path string; Offset, Size int64}` in node order, and `(*Archive).ReadFile(name) ([]byte,
  error)` returning a copy, case-insensitive, `fs.ErrNotExist`-wrapped on a miss. `Entry.Path` is stored
  already normalised — `\`→`/`, trimmed, **lower-cased**.
- `pkg/formats/res`'s own walk uses `depth > maxDepth` with the roots entered at depth 0, so "depth" in
  the sibling reader counts on-disk node levels below the root. Its error strings are prefixed `res: `
  and name the offending node index.
- `go.mod` already carries `require golang.org/x/text v0.40.0` (for `pkg/formats/res`) and
  `github.com/hajimehoshi/ebiten/v2`. `internal/notices` compares `go.mod`'s require set against
  `THIRD_PARTY_NOTICES.md`; this story changes neither.
- `.gitignore` globs `*.reg` and `/builds/`; `scripts/check-no-game-assets.sh` scans **git-tracked**
  paths against an extension list that includes `reg`. A `.reg` fixture file therefore could not be
  committed even by accident — which is moot, because every fixture here is built in test code.
- `builds/README.md` carries a footnote saying `cmd/regtool` "gets its own row when a registry story
  lands"; the status table has no `0011` row yet.

**Standard-library behaviour, measured in this environment (Go 1.26.1).**

- `strings.EqualFold` is **not** an ASCII fold and **must not** be used for FR-2's matching rule. Over
  bytes that are not valid UTF-8 it decodes each to `utf8.RuneError`, so distinct high bytes compare
  **equal**: `EqualFold("x\x80", "x\x81")` and `EqualFold("\xC0", "\xFF")` both return `true`. It also
  folds beyond ASCII (`"K"`/U+212A Kelvin sign, `"s"`/U+017F long s). FR-2 requires that every byte
  outside `A`–`Z`/`a`–`z` compares as itself, which that function violates on both counts.
- `strconv.FormatFloat(v, 'g', -1, 64)` renders `0.0` as `"0"` and `1.0` as `"1"` — no decimal point —
  and renders the non-finite values as `"NaN"`, `"+Inf"`, `"-Inf"`. AC-8 asks a human to read a
  cutscene's `startfade`/`endfade` pair as exactly `0.0` and `1.0`, so the shortest-round-trip form
  alone does not serve the criterion it has to serve (DD13).

## Files to touch

| Path | Intent | Why |
|---|---|---|
| `internal/archtest/dag.go` | MODIFY | Deny external imports to `pkg/formats/reg` (DD13); add `pkg/formats/res` to `cmd/regtool`'s allowed set (DD12). |
| `internal/archtest/dag_test.go` | MODIFY | Two table rows: `pkg/formats/reg` importing `golang.org/x/text` is a violation; `cmd/regtool` importing `pkg/formats/res` and `pkg/formats/reg` is not. |
| `docs/ARCHITECTURE.md` | MODIFY | The `cmd/regtool` DAG row gains `pkg/formats/res`; the `pkg/formats/reg` row records the stdlib-only restriction the check now enforces. |
| `internal/synth/reg.go` | ADD | `RegRawNode`, `RegRaw`, `RegNode`, `Reg`, `RegHeaderSize`, `RegNodeSize`, `RegNodeOffset` — the `.reg` byte-stream builders (DD16). Inputs only. |
| `internal/synth/reg_test.go` | ADD | The builders' layout asserted at named offsets, independently of any parser. |
| `pkg/formats/reg/reg.go` | ADD | `ValueType` and its constants, `Node`, `Reg`, `Parse`, the validation pass and the bounded walk (DD1–DD10). |
| `pkg/formats/reg/lookup.go` | ADD | `GetString`, `GetInt`, `GetIntArray`, `GetFloat` and the ASCII fold (DD11). |
| `pkg/formats/reg/doc.go` | MODIFY | "will implement" → "implements"; **the tier sentence loses its `golang.org/x/text` clause** (DD17). |
| `pkg/formats/reg/reg_test.go` | ADD | AC-1…AC-6, AC-9, P-1, P-2, P-3 and the hand-laid hex witness; `FuzzParse`. |
| `pkg/formats/reg/lookup_test.go` | ADD | AC-7 and FR-2's byte-exact matching rule. |
| `cmd/regtool/main.go` | MODIFY | Replace the skeleton with `dump` and `sweep` over a pure renderer (DD14, DD15). |
| `cmd/regtool/main_test.go` | ADD | AC-10 and FR-3's four rendering properties (the first coverage gap). |

**Not touched:** every other package; `go.mod`, `go.sum`, `THIRD_PARTY_NOTICES.md`, `LICENSES/`;
`AGENTS.md`; `scripts/`; `.gitignore`. No existing test file is edited except
`internal/archtest/dag_test.go`, which gains rows and loses none.

## Design decisions

### The parsed value

- **DD1 — `Reg` holds one synthetic root `*Node`; the header *is* that node's record.** The spec's
  format definition says the root directory has no on-disk node and no name, and that the header's first
  16 bytes have a node's shape. The API mirrors that literally:

  ```go
  type Reg struct {
      Root      *Node // the root directory: Name == "", Dir == true, never nil
      NodeCount int   // the header's nodeCount == the node table's length
  }
  ```

  `Root.Kind` carries the header's `rootFlags` **verbatim and unvalidated** — the spec exempts it from
  every rejection ground — and `Root.Dir` is `true` **by construction**, not derived from bit 0, because
  the root is the root whatever that word holds. Both facts are stated in the field comments, since a
  reader who assumed `Root.Kind` had been checked would be wrong. `Root.Children` is the dereferenced
  `rootFirst`/`rootCount` range and is non-nil-but-empty for a childless registry.
  **`Root.Type` is set to `0` and is never derived from `rootFlags`.** Deriving it would surface
  `rootFlags & 0x0E` on a word nothing validated — a header holding `0x1A` would report a root of type
  `TypeFloatArray`, which is meaningless and reads as a decoded fact. `Type` is documented as meaningless
  whenever `Dir`, and the root is the one node where that is guaranteed rather than merely conventional.
  *Rejected:* `Reg.Children []*Node` with no root node — every tree walk would then need two entry
  shapes, one for the root and one for a directory, and `dump`'s recursion would carry the special case
  at every level. *Rejected:* validating `rootFlags` as a node kind — the spec forbids it by name, and
  a registry whose header word we do not understand must still parse.

- **DD2 — Names and string values are Go `string`s holding the stream's bytes verbatim; nothing is
  decoded anywhere.** A Go `string` is an immutable byte sequence, and `string(b)` performs no
  transformation, so "the bytes come back unchanged, byte for byte" (AC-6) holds exactly. FR-2 names the
  accessor `GetString`, which fixes the return type. **No `unicode/utf8`, no `golang.org/x/text`, no
  `strings.ToLower`, no `%s`-with-conversion appears anywhere in this package.**
  *Rejected:* `[]byte` for names and values — it either aliases the parser's memory (letting a caller
  mutate the tree) or costs a copy on every read, and it contradicts FR-2's accessor name; the
  immutability of `string` gives byte fidelity and safety at once.
  *Recorded consequence:* these strings are **not** guaranteed valid UTF-8. Anything that displays one
  must apply its own convention and say so — which is exactly what FR-3 makes `dump` do (DD14).

- **DD3 — One `Node` struct with a type discriminator; values are decoded eagerly at parse.**

  ```go
  type ValueType uint32

  const (
      TypeString     ValueType = 0
      TypeInt        ValueType = 2
      TypeFloat      ValueType = 4
      TypeIntArray   ValueType = 6
      TypeFloatArray ValueType = 10 // decoded, deliberately not implemented — DD10
  )

  type Node struct {
      Name     string    // raw bytes (DD2)
      Kind     uint32    // the raw kind word, verbatim
      Dir      bool      // Kind & 0x01
      Type     ValueType // Kind & 0x0E; meaningless when Dir
      Children []*Node   // Dir only, in node-table order; nil for a value node
      Str      string    // TypeString
      Int      int32     // TypeInt
      Float    float64   // TypeFloat
      Ints     []int32   // TypeIntArray
  }

  func (n *Node) Sorted() bool         { return n.Kind&0x10 != 0 }
  func (n *Node) NameTruncated() bool  { return n.Kind&0x10000000 != 0 }
  ```

  Every value is decoded during `Parse` and stored; nothing is re-read from the byte slice afterwards,
  and `Parse` does **not** retain the input slice. So a `*Reg` cannot be invalidated by a caller
  mutating `data`, and P-2 ("every heap read stays inside the heap window") is a property of one bounded
  region of code rather than of every later accessor call.
  The raw `Kind` is retained beside the derived fields because the spec documents three flag bits and
  assigns meaning to two; a tool that wants bit 28 must not have to re-parse.
  **`Children` is non-nil for every directory, including one with zero children, and nil for every value
  node**, so `Dir` and `Children != nil` always agree and a walker needs one test rather than two.
  `Ints` is likewise empty-but-non-nil for a zero-length type-6 array. Neither is a rejection: the
  spec's Validation list is exhaustive and contains no minimum count, and `nodeCount == 0` is a valid
  registry (AC-9's neighbour case).
  *Rejected:* a `Value any` or a per-type interface — every consumer would type-switch, `dump` included,
  for no gain over an explicit discriminator; *rejected:* lazy decoding from a retained slice — it moves
  the heap-bounds guarantee out of `Parse` and makes P-1's "no partial tree on error" harder to hold.

- **DD4 — `GetIntArray` returns a fresh copy; every other accessor returns a value.** Slices are the
  only aliasing surface in the API, so the one accessor that returns one copies. `Node.Ints` is still
  exported and is **not** copied on field access — a caller walking the tree directly is reading the
  parser's memory and the field comment says so. This matches the house rule the sibling reader already
  applies to `Entries()` and `ReadFile`.

### Parsing and validation

- **DD5 — Two passes: exhaustive table validation, then a bounded tree walk. An orphan is exempt from
  neither.** `Parse` runs, in order:

  1. **Framing.** `len ≥ 0x18`; `signature == 0x31415926`; read `rootFirst`, `rootCount`, `rootFlags`,
     `nodeCount`; require `len ≥ 0x18 + 0x20·nodeCount` (the whole node table) and then
     `len ≥ heapOrigin + 4` (the `heapSize` word) and `len ≥ heapOrigin + 4 + heapSize` (the heap).
     Trailing bytes past the heap end are **tolerated**, as the spec directs.
     **The root's own range is a directory range and is validated here by the same rule the table pass
     applies to every other directory: `rootFirst + rootCount ≤ nodeCount`, computed in `uint64` so the
     addition cannot wrap (DD7).** It is easy to miss because the root has no node record, and missing it
     is a panic rather than an error — the walk would slice the node table out of range, which P-1
     forbids. `rootFlags` remains exempt from every check (DD1).
  2. **Table pass** — every one of the `nodeCount` records, in index order, **orphans included**: reject
     a node whose bit 0 is set over non-zero type bits; for a directory, reject `first + count >
     nodeCount`; for a value, reject an unsupported type (10) or an unrecognised one (8 and every
     remaining value); for type 6 reject `size % 4 != 0`; for types 0 and 6 reject a heap reference
     outside the heap. Decode each value into its `Node` field.
  3. **Tree walk** — from the root's range, depth-first in node-table order, building `Children`,
     enforcing the single-parent rule and the depth cap (DD6).

  **The orphan ambiguity, resolved.** The spec says an unreferenced node "is ignored rather than
  rejected". That is a statement about *reachability*, not a validation exemption: **an orphan's own
  record is validated exactly like any other record, and "ignored" means it contributes no node to the
  tree and is not itself an error.** So a registry containing a well-formed orphan parses and its orphan
  is absent from the tree; a registry containing an orphan of type 8 is rejected. This is the reading
  that makes the rejection rules total — a malformed record that nothing points at is still a malformed
  record, and the alternative would let a stream carry arbitrary undecodable content past the parser.
  *Rejected:* validating only along the walk — an unreached malformed node would then slip through,
  which is precisely the hole P-1's "for any malformed input, an error and no partial tree" closes.

  On any rejection `Parse` returns `(nil, err)` — never a partially built tree, never a panic (P-1). A
  fuzz target over `Parse` runs its seed corpus in the ordinary `go test` run as the negative-invariant
  witness over inputs no hand-written fixture reaches.

- **DD6 — The single-parent set kills cycles; the depth cap is the stack backstop; the boundary is 32.**
  The walk carries a `seen []bool` over node indices. Entering an already-seen index is the rejection
  the spec's "a node referenced by more than one directory range is rejected" names, and it is what
  bounds total work to O(nodeCount): a cycle, a self-referencing range and an acyclic shared subtree are
  all the same rejection, and none of them can loop. The depth cap is therefore **not** the cycle guard;
  it is the stack bound for a *legal* deep chain of distinct nodes.
  **The arithmetic, pinned because it is off-by-one bait.** The root is depth **0**; a child of the root
  is depth **1**; a node at depth ≤ 32 is accepted and a node that would sit at depth **33** is
  rejected. So a registry nesting 32 levels of on-disk nodes below the root parses, and one nesting 33
  does not. This matches the convention the sibling `.res` reader already uses for its own guard.
  **Precedence, so a test knows which error to expect.** On a self-referencing or cyclic range the
  single-parent rejection fires first — the walk re-enters an index it has already marked, at a depth
  far below 32 — so the depth-cap error is reachable only from a chain of 33 *distinct* nodes. Both
  terminate; neither hangs (AC-4).
  **The rule's scope is the tree, not the table, and the two readings differ on a constructible
  input.** The spec states the rule flatly ("a node referenced by more than one directory range is
  rejected"), which read table-wide would also reject a stream where an **unreachable** directory's
  range names a child that a reachable directory also names. This design rejects **only** a node
  entered twice by the walk, so that stream parses. The reason is the spec's neighbouring rule: an
  unreferenced node is *ignored*, and a node that is ignored cannot simultaneously be a directory whose
  range binds the parser. Reading the rule table-wide would make an ignored node's range decisive, which
  is the one thing "ignored" rules out. The walk-scoped reading is therefore the one under which the two
  defensive rules are consistent, and it is what SC-8 tests.
  *Rejected:* a pre-pass counting references across every range in the table — it makes the two spec
  rules contradict each other, and it would also change which error a cycle produces.

- **DD7 — Bounds arithmetic is done in `uint64`, and the formulas are written here.** Every quantity
  read from the stream is a `u32`; every comparison that could wrap is widened first. `int` is 32 bits
  on a 32-bit build, so `int` arithmetic is not sufficient and is not used for any of these.

  - `heapOrigin = 0x18 + 0x20·nodeCount`, `heapStart = heapOrigin + 4`,
    `heapEnd = heapStart + heapSize` — all `uint64`, all compared against `uint64(len(data))`.
  - A directory range is in bounds iff `uint64(first) + uint64(count) ≤ uint64(nodeCount)`. Written that
    way the addition cannot overflow, which is the AC-4 case a same-width check would pass.
  - A type-0 or type-6 heap reference is in bounds iff `uint64(data) + uint64(size) ≤ uint64(heapSize)`
    — offsets are relative to `heapStart`, per the spec.
  - A type-4 value is `math.Float64frombits(uint64(size)<<32 | uint64(data))` — the eight bytes at node
    offset `0x04` read as one little-endian binary64, with `data` the low word and `size` the high word.
    No heap access.
  - A type-6 element `k` is `int32(le32(heap[data+4k : data+4k+4]))`.

- **DD8 — Text extraction: two cutting rules, neither of them a rejection.**
  *A name* is the 16 bytes at node offset `0x10` up to the first NUL; if no NUL is present the whole 16
  bytes are taken. Never more than 16 — the parser slices the node's own record, so it cannot overrun
  into the next node whatever the bytes say.
  *A string value* is `heap[data : data+size]` cut at the first NUL. If no NUL is present within `size`
  the whole slice is taken; `size == 0` yields the empty string. **Neither is a rejection**: the spec's
  Validation section lists the rejection grounds exhaustively and "a string value that is not
  NUL-terminated" is not among them, so a parser that rejected one would be adding a rejection the spec
  does not have. The format's own writer stores `strlen(s)+1` bytes, so a stream reaching either branch
  is one that writer could not have produced — which is exactly when a reader should be lenient about
  content and strict about bounds.

- **DD9 — Error vocabulary; *unsupported* and *unrecognised* are different words on purpose.** Every
  error is `fmt.Errorf` with the package prefix `reg: `, matching the sibling reader's house style. The
  two the spec requires to name the type and the node index are fixed here verbatim, because AC-5 is
  checkable only against a stated wording:

  - type 10 → `reg: node %d: unsupported value type %d`
  - type 8 and every other unrecognised value → `reg: node %d: unrecognised value type %d`

  **The type number printed is the masked type, `kind & 0x0E`, never the raw kind word** — so a node
  whose kind is `0x1A` reports type `10`, not `26`. The mask is what the spec compares against and what
  the reader must report for the message to name the thing that was rejected.
  The distinction is contract, not decoration: type 10 is **decoded and deliberately unimplemented**
  (see below), type 8 is **undecoded**. Collapsing them into one message would tell a future reader we
  do not know what a type-10 node holds, which is false.
  **Two further messages are contract because a criterion reads them**, and the rest are not:
  - the double-reference rejection **names the offending node index** (SC-8 asserts it);
  - the root's own out-of-range header range names **no** node index, because no node is at fault —
    `reg: root child range [%d, %d) exceeds node count %d`.

  Every other message is free to follow the sibling reader's house wording; no criterion reads one, and
  fixing text nothing checks would be ceremony.
  No sentinel error values and no error types are introduced: nothing in the spec requires
  `errors.Is`/`errors.As` discrimination, and a caller that needs to distinguish these two cases is not
  a caller this story has.
  **Order within a single node, so two agents produce the same message for a doubly-defective record.**
  Bit 0 decides first, as the spec directs. For a **directory**: the type-bits conflict (bit 0 set over
  non-zero type bits), then the child-range check. For a **value**: the type branch first — an
  unsupported (10) or unrecognised (8 and every remaining value) type is rejected before any
  type-specific check, since a `size % 4` rule is meaningless on a type that is not 6 — then, for type
  6, `size % 4`, then the heap-bounds check for types 0 and 6. A node is reported for the first of these
  that fires.

- **DD10 — Type 10 (float64 array) is documented, not implemented, and the reversal recipe is written
  down.** The type is decoded — heap offset in `data`, byte length in `size`, `size/8` little-endian
  doubles — and it occurs **zero** times in any shipped registry. So this story defines the constant
  `TypeFloatArray`, rejects a type-10 node as *unsupported* naming type and index, and ships **no
  decoder, no `GetFloatArray` accessor and no acceptance criterion** for it.
  **The reason, in the spec's own terms: the gap is scope, not ignorance.** Shipping an untested decoder
  for a type no shipped file exercises buys nothing and claims coverage we would not have; claiming the
  type is undecoded would overstate our ignorance, which the evidence-honesty rule forbids as squarely
  as overstating our knowledge. The rejection wording is what keeps both statements true at once.
  **Reversal recipe, should a registry carrying one ever appear:** decode `size/8` doubles from
  `heap[data : data+size]` under the same bounds rule as type 6 with `size % 8 == 0`; add
  `Floats []float64` to `Node`; add `GetFloatArray(section, key string) ([]float64, bool)`; add one
  acceptance criterion of AC-1's shape covering it; move the type from the rejection branch to the
  decode branch. Nothing else in this design changes.

### Lookup

- **DD11 — The accessors, their fold, and every way they miss.**

  ```go
  func (r *Reg) GetString(section, key string) (string, bool)
  func (r *Reg) GetInt(section, key string) (int32, bool)
  func (r *Reg) GetIntArray(section, key string) ([]int32, bool)
  func (r *Reg) GetFloat(section, key string) (float64, bool)
  ```

  Each resolves `section` among `Root.Children` and then `key` among that node's children.
  **`GetInt` returns `int32`, not `int`** — the format's type is a signed 32-bit integer and widening it
  in the API invites a consumer to assume a range the format does not have. *Rejected:* `int` for
  ergonomics — `pkg/data` is the next consumer and it should convert deliberately.
  **The fold is hand-written over bytes:** two names match iff they have equal length and, at every
  position, equal bytes after mapping `A`–`Z` (0x41–0x5A) to `a`–`z`; **every other byte compares as
  itself**. `strings.EqualFold` is **forbidden here** and the reason is measured, not stylistic: over
  bytes that are not valid UTF-8 it maps each to `utf8.RuneError` before comparing, so `"\xC0"` and
  `"\xFF"` compare **equal** — it would silently merge two distinct registry keys. It also folds
  non-ASCII pairs (Kelvin sign, long s) that FR-2 requires to compare as themselves. `strings.ToLower`
  is forbidden for the same reason.
  **Whole names are compared**, not their first 15 characters. FR-2 records that the original engine
  compares 15 and that no registry its own writer produced can distinguish the two rules; this is our
  API's convenience and the spec says so.
  **The first match in node-table order wins** at each level, for both the section and the key. The
  format permits duplicate sibling names and the spec assigns no meaning to one; a deterministic rule is
  required so two runs agree, and "first in table order" is the only one derivable from the stream
  alone. *Rejected:* rejecting a duplicate at parse — it adds a rejection ground the spec's exhaustive
  Validation list does not have.
  **Every miss is `(zero value, false)` and never a panic:** a missing section; a section name that
  resolves to a **value** node rather than a directory; a missing key; a key whose type is not the
  accessor's; and a key that is a directory when a value was asked for.

### The architecture edits

- **DD12 — `cmd/regtool` gains `pkg/formats/res` in the DAG, and that is forced by FR-3.** FR-3 defines
  `dump <archive.res> <entry.reg>` and `sweep <dir>`, both of which read `.reg` bytes **out of `.res`
  archives**; the analysis records that no loose `.reg` file exists on disk outside an archive. The
  tool's current row permits `pkg/formats/reg` only, so the check — which is authoritative and
  fail-closed — would reject the tool the spec requires. The row becomes
  `"cmd/regtool": {"pkg/formats/reg", "pkg/formats/res"}`, and `docs/ARCHITECTURE.md`'s matching row is
  edited in the same change.
  This is a `cmd`-tier edge between a tool and a format package, which is the shape the tier already
  has: `cmd/restool`, `cmd/sprtool` and `cmd/terraintool` each list two or more format/library packages.
  **It does not touch AC-11**, which constrains the *library* package `pkg/formats/reg`, not the command
  that drives it.
  *Rejected:* `dump` taking a bare `.reg` file path — the spec's FR-3 signature says otherwise, and no
  such file exists in an install, so AC-8 would be unrunnable. *Rejected:* routing through `pkg/vfs` —
  heavier, adds an archive-layering concern this story's Out-of-scope excludes, and still needs a new
  DAG edge.

- **DD13 — AC-11 is made enforceable by denying `pkg/formats/reg` any external import.**
  `externalAllowed` today grants `golang.org/x/text` to every `pkg/formats/*` package by prefix, so
  AC-11 — "the repository's architecture import check … it imports the standard library only" — is a
  claim the shipped check cannot currently make. The change is one narrow deny consulted before the
  tier-wide grant:

  ```go
  // Format packages that convert no text take no text-encoding dependency, and
  // the check holds them to it (0011 spec AC-11).
  var noExternalFormats = map[string]bool{"pkg/formats/reg": true}
  ```

  and a `dag_test.go` table row proving `pkg/formats/reg` importing `golang.org/x/text` is a violation,
  beside the existing row proving `pkg/formats/res` importing it is not. The live-tree test then proves
  the shipped package complies.
  **What the check does and does not reach, stated rather than assumed.** `archtest.Load` collects
  **production** imports only — it drops every `_test.go` file's imports for every package except
  `pkg/sim`. So this deny polices `reg.go`, `lookup.go` and `doc.go` and **not** the package's test
  files. That is the right scope for AC-11, which speaks of "the shipped `pkg/formats/reg` package", and
  it is written down so nobody later mistakes the guard for wider than it is. Nothing in this story
  wants `golang.org/x/text` in a test either, and no criterion claims the check would catch it.
  **Deliberately narrow, and the residual is recorded rather than fixed.** `pkg/formats/spr256` is
  documented in `docs/ARCHITECTURE.md` as stdlib-only for the same reason (`.256` carries no strings)
  and stays **unenforced** by this change. Adding it would alter the enforcement surface of another
  story's package as a side effect of this one — the silent scope growth S-6 forbids. It is reported
  here as a known gap, closable by one map entry whenever its owner wants it.
  *Rejected:* inverting the grant into a per-package allow-list — same effect, but it changes the rule
  for `res`, `alm` and `spr256` at once and turns a targeted assertion into a repo-wide policy change.
  *Rejected:* a test inside `pkg/formats/reg` asserting its own imports — AC-11 names "the repository's
  architecture import check", and a package's self-assertion is not that check.

### The tool

- **DD14 — `dump` is a pure renderer, and its display convention is fixed here in full.** The rendering
  is `func render(w io.Writer, r *reg.Reg) error` in `package main`, taking a `*reg.Reg` and writing the
  tree; the archive-opening half of `dump` is the only part that needs a file. That is what makes AC-10
  and FR-3's four stated rendering properties unit-testable with no install (and is how the first
  coverage gap is closed — see SC-9).

  **The byte convention (FR-3, AC-10).** Every byte of a name and of a string value renders as itself if
  it lies in `0x20`–`0x7E`, and as `\xNN` otherwise, with `NN` **two lower-case hex digits, always two**.
  Output is therefore pure ASCII, hence valid UTF-8 by construction — which is what AC-10 asserts.
  *Recorded consequence, since it is a property of the spec's rule and not ours to fix:* the encoding is
  **not injective** — `0x5C` (`\`) and `0x22` (`"`) are printable and print as themselves, so a name
  containing a literal `\x41` renders identically to one containing `A`, and a quoted string containing
  `"` looks unbalanced. Escaping the backslash would contradict the spec's "bytes `0x20`–`0x7E` print as
  themselves". `dump` is a developer's reading aid with no machine consumer, so the ambiguity costs
  nothing; it is written down so a later reader knows it was seen, not missed.

  **The line grammar.** Indentation is **two spaces per level**, with the root's children at level 0.
  - A directory renders as `<indent><escaped name>:` and its children follow, indented one level deeper,
    in node-table order.
  - A value renders as `<indent><escaped name> = <value>`.
  - `TypeString` → `"` + the escaped bytes + `"`.
  - `TypeInt` → the signed decimal.
  - `TypeFloat` → `strconv.FormatFloat(v, 'g', -1, 64)`, **and if the result contains none of `.`, `e`,
    `E`, `N` or `I`, `.0` is appended.** So `0` renders `0.0`, `1` renders `1.0`, `1.5` stays `1.5`,
    `1e+100` stays `1e+100`, and `NaN`/`+Inf`/`-Inf` are left alone. The reason is AC-8: a human reading
    a cutscene registry must see `startfade`/`endfade` as `0.0` and `1.0`, and it is the decimal point
    that distinguishes a double from an `int32` in a dump with no type column.
  - `TypeIntArray` → `[` + space-separated elements + `]` when the array holds **8 or fewer**; when it
    holds **more than 8**, the first eight then `...` then `(N total)`, i.e.
    `[0 1 2 3 4 5 6 7 ... (33 total)]`. An empty array renders `[]`. Eight is the last unabbreviated
    length and nine the first abbreviated one — the boundary FR-3's "abbreviated past 8 elements" names.

  Nothing else goes to stdout. `dump` writes one summary line to **stderr** (`regtool: <N> nodes`),
  which is where the sibling `restool` puts its own count, so the renderer's output stays exactly the
  tree a test asserts on.

- **DD15 — `sweep` walks recursively, matches case-insensitively, and survives a bad archive.**
  `sweep <dir>` walks `<dir>` **recursively** (`filepath.WalkDir`), taking every regular file whose name
  ends in `.res` **case-folded** — `VIDEO4.RES` and `graphics.res` are both in the shipped install, and
  AC-8's pass condition is a count of **44**, so a scan that could silently miss an archive in a
  subdirectory could report a wrong total that looks right. For each archive it iterates `Entries()` and
  parses every entry whose path ends `.reg` case-folded. Nested archives are not descended into: a
  `.reg` entry's bytes go to `reg.Parse` and never back to `pkg/formats/res`.
  Per entry it prints `<archive path>  <entry path>  <N> nodes`. An archive that fails to **open** is
  reported on stderr and the walk continues — one unreadable file must not hide the other 43 registries.
  A `.reg` entry that fails to **parse** is reported on stderr with its error and counted as failed.
  The run ends with the spec's line on stdout, `<N> regs: <N> parsed, <N> failed`, and exits **non-zero
  if any registry failed**, so AC-8's pass condition is checkable by exit status as well as by eye.
  *Rejected:* a non-recursive scan — see above; *rejected:* aborting on the first bad archive — it makes
  a partial census indistinguishable from a complete one.

### Fixtures and the two unearned notes

- **DD16 — One dumb layout writer in `internal/synth`, one tree convenience on top of it, and one
  hand-laid stream that trusts neither.** Two test packages need `.reg` byte streams — `pkg/formats/reg`
  and `cmd/regtool` — which is exactly the case `internal/synth` exists for (it is unpoliced by
  `archtest`, imported only from tests, and its own doc rule is *inputs only, never an expected
  output*). Its new file adds:

  ```go
  const (
      RegHeaderSize = 0x18
      RegNodeSize   = 0x20
  )

  // RegNodeOffset returns the byte offset of node i's 32-byte record.
  func RegNodeOffset(i int) int

  // RegRawNode is one node record written verbatim: Name is laid into the
  // 16-byte name field (NUL-padded, cut at 16), Data/Size/Kind are written as
  // given. Nothing is validated.
  type RegRawNode struct {
      Name       []byte
      Data, Size uint32
      Kind       uint32
  }

  // RegRaw lays out header | node table | heapSize | heap with no validation of
  // any kind: the escape hatch for fixtures a well-formed tree cannot express.
  func RegRaw(rootFirst, rootCount, rootFlags, nodeCount uint32, nodes []RegRawNode, heap []byte) []byte

  // RegNode is a node in a well-formed tree. Only the field the Kind's type bits
  // select is read; a record word the type ignores is written as 0.
  type RegNode struct {
      Name     string
      Kind     uint32
      Children []RegNode
      Str      string
      Int      int32
      Float    float64
      Ints     []int32
  }

  // Reg assembles a well-formed registry from a tree, assigning node indices
  // breadth-first (so each directory's children are a contiguous range) and
  // packing heap items in node-index order with no gap. rootFlags is written
  // verbatim.
  func Reg(rootFlags uint32, children []RegNode) []byte
  ```

  `Reg` is implemented **on top of** `RegRaw`, so there is exactly one place that knows the layout.
  Every malformed fixture — an orphan, a doubly-referenced node, a cycle, a range past `nodeCount`, an
  overflowing `first + count`, a type-8 or type-10 node, a `size % 4 != 0`, a heap overrun, a 16-byte
  name with no NUL — is written with `RegRaw`, or by patching a well-formed stream at
  `RegNodeOffset(i) + 0x04 / +0x08 / +0x0C`, which are the spec's own offsets rather than the builder's
  internals.
  **`nodeCount` is a parameter of `RegRaw`, separate from `len(nodes)`**, so a stream whose header
  over-declares the table (AC-3) needs no truncation trick.
  **The anti-tautology witness, and why it is not gold-plating.** A builder and a parser written against
  the same misread offset agree perfectly and pass every test — which is not hypothetical here: a
  registry framing that satisfied every structural invariant while sitting one word off survived review
  for a whole story interval, and only an instrument outside the model caught it. So one fixture in
  `reg_test.go` is a **hand-laid hex byte string**, written directly from the spec's layout tables and
  built by nothing, asserted to parse to a tree stated by hand. `internal/synth/reg_test.go`
  independently asserts the builders' output **at named offsets** — signature at `0x00`, `nodeCount` at
  `0x10`, node `i`'s `kind` at `RegNodeOffset(i)+0x0C`, the `heapSize` word at `0x18 + 0x20·nodeCount` —
  so the builder is pinned before any parser exists to agree with it.

- **DD17 — `pkg/formats/reg/doc.go`'s tier sentence is corrected here; the two repo-level tier tables
  are not, and the distinction is real.** The package comment currently reads "may import only the Go
  standard library (plus golang.org/x/text for CP866 string decoding)". That clause is a claim **about
  this package**, it is false — the parser converts nothing (FR-1, AC-11) — and the file is
  implementation surface this story rewrites anyway, so it is fixed in the change that first touches it.
  `AGENTS.md`'s tier table and `docs/ARCHITECTURE.md`'s `pkg/formats/reg` DAG row say
  `stdlib + golang.org/x/text` for the formats tier as a **group**. Read as *permission* statements
  those stay accurate — `externalAllowed` does grant the tier that import, and `pkg/formats/res` and
  `pkg/formats/alm` genuinely use it. Read as an *anticipation* that `reg` will need CP866 they are
  unearned, and that anticipation is what this story retires.
  **`AGENTS.md` is out of scope and is left alone: known and pending.** `docs/ARCHITECTURE.md` is edited
  by this story only in its `cmd/regtool` DAG row (DD12) and in recording the restriction DD13 now
  enforces on `pkg/formats/reg`; its tier-table prose is otherwise untouched. Anyone reconciling the
  three should note they are three statements of the same shape and should move together.

- **DD18 — The test shape, pinned only where two agents would otherwise diverge on something that
  matters.** Style is left free; these five are not.
  **(a) Test package clauses.** `pkg/formats/reg`'s tests are `package reg_test`, matching every sibling
  format package (`res_test`, `alm_test`, `spr256_test`); they exercise the exported API only, which is
  all this story has. `cmd/regtool`'s test is `package main`, because the renderer it tests is
  unexported — the house shape for a `cmd` test.
  **(b) The hand-laid hex witness's coverage is a requirement, not a flourish.** Its whole purpose is to
  be an instrument outside the model, so *what it covers* is the entire question. It MUST contain, in
  one stream: at least one **heap-backed** value (a misplaced `heapOrigin` is otherwise invisible to
  it); at least two nodes at **different table indices** (a wrong node stride is otherwise invisible);
  a **directory** with a child range (a wrong `first`/`count` slot is otherwise invisible); and a
  **type-4** node (the word order R-2 resolved). Its expected tree MUST be written by reading
  `spec.md`'s layout tables — **never** produced by running the parser and pasting the result, which
  would make the witness tautological in exactly the way it exists to prevent.
  **(c) The SC-5 fixtures place their defective nodes at indices whose decimal cannot alias the type
  number** they assert — index `1` is a substring of `10` and index `8` equals type `8` — so an
  assertion that reads the type and the index out of the message text cannot pass on the wrong
  substring. The type-10 node and the type-8 node sit at indices ≥ 2 and distinct from their own type
  values, and their kind words carry **no flag bits**, so the masked-versus-raw question (DD9) cannot
  hide there either.
  **(d) `FuzzParse`'s body asserts one thing: `(nil, err)` or `(*Reg, nil)`, never both nil and never
  both non-nil.** It does **not** walk the returned tree — a walk over a tree a broken parser produced
  could not terminate, and a fuzz target that hangs reports nothing at all. The cycle guarantee is
  SC-4's and SC-8's job, where the input is known. Seeds: a well-formed multi-type registry, the
  hand-laid stream, the empty slice, a header-only stream, a truncated table, a type-8 stream and a
  cyclic one.
  **(e) The AC-9 case is two fixtures, not one**: `nodeCount == 0` (an empty registry, whose root child
  set is empty-but-non-nil and which never enters the table loop) and a **heapless but populated**
  registry carrying only int32 and float64 nodes. They exercise different code, and neither is a
  rejection.

## Success criteria

Each is a named automated test or a named developer-run observation, against the requirement it serves.
`unit` fixtures are byte streams built in test code; **no test reads a game install**.

1. **SC-1 — the whole tree (FR-1, AC-1, AC-9).** *Automated.* `TestParseTree` in `pkg/formats/reg`: a
   synthetic registry using all five implemented node kinds — string, directory, int32, float64, int32
   array — with nested directories and one directory carrying `kind == 17`, parses to the exact tree:
   every name, every value, every array element, and the nesting, compared against hand-stated
   expectations. The `kind == 17` node is a **directory**, not a rejection, and its `Sorted()` reports
   true. Separately, AC-9 is asserted as **two** fixtures (DD18e) — an empty registry (`nodeCount == 0`,
   root child set empty-but-non-nil) and a heapless-but-populated one carrying only int32 and float64
   nodes — and a **hand-laid hex byte stream** built by no builder parses to a hand-stated tree, with
   the coverage DD18b requires of it: a heap-backed value, two nodes at different table indices, a
   directory with a child range, and a type-4 node (DD16).
2. **SC-2 — the double (AC-2).** *Automated.* `TestParseFloat`: a type-4 node whose `data`/`size` carry
   a known IEEE-754 bit pattern **with a non-zero mantissa low word** yields exactly the corresponding
   `float64`, compared bit-for-bit via `math.Float64bits`. The shipped corpus witnesses only the high
   half, so this criterion is **synthetic and must not be described as corpus-backed**; the `0.0`/`1.0`
   pair is asserted too, as the values the developer-run criterion will meet.
3. **SC-3 — framing rejections (AC-3, P-1).** *Automated.* `TestParseRejectsFraming`: a truncated
   stream, a bad signature, and a node table shorter than the header's `nodeCount` each yield
   `(nil, err)` with no panic. A stream with **trailing bytes past the heap end** parses (the spec
   tolerates them), which is the boundary on the other side of the same length check.
4. **SC-4 — range, cycle and depth rejections (AC-4, P-1).** *Automated.* `TestParseRejectsRanges`: a
   directory range past `nodeCount`; **the header's own `rootFirst`/`rootCount` range past `nodeCount`**;
   a self-referencing range; a cyclic pair; and a `first + count` that overflows 32 bits each yield an
   error with no hang and **no panic**. The depth boundary is asserted at both sides: a chain nesting
   **32** levels below the root parses, and one nesting **33** is rejected (DD6).
5. **SC-5 — value-type rejections (AC-5, P-2).** *Automated.* `TestParseRejectsValues`: type 6 with
   `size % 4 != 0`; a heap overrun for type 0 and separately for type 6; a type-10 node; and a type-8
   node each yield an error. The type-10 message contains **`unsupported`**, the type-8 message contains
   **`unrecognised`**, and each names its type and its node index (DD9) — asserted by reading the
   substrings out of the message, so a degraded message fails. The two defective nodes sit at the
   indices DD18c requires, so a substring assertion cannot pass on the wrong number.
6. **SC-6 — byte fidelity (AC-6, FR-1).** *Automated.* `TestParseBytesVerbatim`: a name and a string
   value carrying bytes in `0x80`–`0xFF` come back **byte for byte unchanged**, with no character
   mapping applied; a name filling all 16 bytes with **no NUL** is kept whole and does not run into the
   next node's record. The fixture bytes and the expected bytes are written as **hex byte slices**,
   never as literal non-ASCII text.
7. **SC-7 — the accessors (FR-2, AC-7).** *Automated.* `TestAccessors` in `pkg/formats/reg`: over one
   parsed tree, each of `GetString`, `GetInt`, `GetIntArray`, `GetFloat` returns the value and `true` on
   a hit whose section and key differ from the stored names **only by ASCII case**; and returns
   `(zero, false)` for a wrong-type key, a missing key, a missing section, and **a value node used as a
   section** — that last without panicking. The fold's stated boundary is asserted too: two names
   differing only in a byte `≥ 0x80` do **not** match (which `strings.EqualFold` would wrongly accept —
   DD11), and `GetIntArray`'s result is a copy, so mutating it leaves a second call unchanged.
8. **SC-8 — reachability, orphans and shared nodes (P-3, and the spec's two defensive rules).**
   *Automated.* `TestReachability` in `pkg/formats/reg` — **this criterion closes a coverage gap: P-3
   and both defensive rules had no criterion.** Three arms:
   *(a)* **P-3, completeness** — a registry with no orphans parses, and a walk of the tree yields
   exactly `NodeCount` nodes, each reached **once** (the test counts nodes and collects them into a set
   whose size it compares to `NodeCount`).
   *(b)* **the orphan is ignored** — the same registry with one extra, **well-formed** node that no
   directory range references parses cleanly, the orphan is absent from the tree, and the walk yields
   `NodeCount − 1` nodes. A second arm shows the exemption is about reachability only: the same
   registry whose orphan is a **type-8** node is **rejected** (DD5).
   *(c)* **the doubly-referenced node is rejected** — a registry in which two *distinct, non-nested*
   directories both list the same child index is rejected with an error naming that index, and no hang.
   This is the acyclic case, distinct from SC-4's cycle.
9. **SC-9 — the dump rendering (FR-3, AC-10).** *Automated.* `TestDumpRender` in `cmd/regtool` — **this
   criterion closes a coverage gap: FR-3's four stated rendering properties had no criterion, only the
   byte convention did.** Over one synthetic registry parsed by `reg.Parse` and rendered by `render`:
   *(a)* **indentation** — a nested directory's children are indented two spaces deeper than their
   parent's line, asserted at two levels;
   *(b)* **quoting** — a string value appears wrapped in `"`, an int32 does not;
   *(c)* **abbreviation** — an array of **8** elements renders in full and one of **9** renders as its
   first eight, then `...`, then `(9 total)`; an empty array renders `[]`;
   *(d)* **doubles** — `0.0` renders `0.0` and `1.0` renders `1.0` (not `0` and `1`), `1.5` renders
   `1.5`, and a non-finite value renders `NaN`/`+Inf` with no `.0` appended (DD14);
   *(e)* **AC-10** — a name and a string value carrying bytes outside `0x20`–`0x7E` render with the
   printable bytes verbatim and every other byte as `\xNN`, no code page applied, and the **whole
   rendered output is valid UTF-8** (`utf8.ValidString`) and contains no byte above `0x7E`.
10. **SC-10 — the parser takes no text-encoding dependency (AC-11, FR-1).** *Automated.* The
    `internal/archtest` table test gains a row asserting `pkg/formats/reg` importing
    `golang.org/x/text` is a violation naming that edge, beside the existing row that permits it to
    `pkg/formats/res`; and a row asserting `cmd/regtool` importing both `pkg/formats/reg` and
    `pkg/formats/res` is clean. `TestLiveTreeClean` then proves the shipped `pkg/formats/reg` imports
    the standard library only and no other module package. The second half — `docs/ARCHITECTURE.md`
    agreeing with the allow-map — is a **reviewed edit, not a test**, because nothing checks the doc.
11. **SC-11 — the install (AC-8).** *Developer-run, needs a lawful install.* `regtool sweep <root>` over
    the install root reports **44** `.reg` entries, **44 parsed, 0 failed**, and exits zero; the
    per-archive distribution matches the spec's figures (33 cutscene registries across
    `VIDEO4.RES`/`VIDEO8.RES`, 5 in `graphics.res`, 3 in `scenario.res`, 2 in `world.res`, 1 in
    `sfx.res`). Then the spot-checks a wrong field mapping could not pass, read off `regtool dump`:
    `units.reg` `[Global] UnitCount == 34` and `FileCount == 33`; `[Unit0] DescText` reads as coherent
    text; `[Unit0] AttackPhases` equals the element count of `[Unit0] AttackAnimFrame` and `MovePhases`
    the count of `MoveAnimFrame`; and a cutscene registry's `startfade`/`endfade` pair reads exactly
    `0.0` and `1.0`. **The evidence is the observed output — counts, values and the summary line — and
    no game byte, extracted entry or archive is committed.** If the install is unavailable the criterion
    is recorded as an explicit pending limitation naming what stays unobserved; it is never marked
    passed on the strength of a synthetic test, because no synthetic test can corroborate a corpus
    figure.
12. **SC-12 — project hygiene.** *Project-mechanics gate (AGENTS.md), recorded as such rather than as a
    spec criterion.* With **no game install visible to the test suite**: `go build ./...`,
    `go vet ./...`, `go test -count=1 ./...` (including `internal/archtest` and `internal/notices`),
    `gofmt -l $(git ls-files '*.go')` empty, and `scripts/check-no-game-assets.sh` in both tree and
    `--history` mode, all clean. `go.mod`, `go.sum` and `THIRD_PARTY_NOTICES.md` are unchanged, so the
    notices equality test is unaffected.

## Risks (product)

- **R-1 (plan) — a fixture builder and a parser can agree on a wrong offset, and every test still passes.**
  This is not a hypothetical risk in this format: a registry framing that satisfied every structural
  invariant while sitting one word off the truth survived review for a whole story interval, tiled the
  stream exactly, and computed the same pool start. A synthetic suite cannot by itself distinguish a
  correct model from a self-consistent wrong one. *Mitigations:* the layout builder is landed and
  pinned at named offsets **before** any parser exists (DD16); one fixture is hand-laid hex that the
  builder never touches; and SC-11's spot-checks are chosen so that a shifted mapping fails them —
  `AttackPhases` equalling the element count of a different key, and an exact `0.0`/`1.0` pair, are
  agreements a wrong framing does not produce. **Residual:** until SC-11 is run, correctness rests on
  synthetic evidence plus the spec's own derivation.
- **R-2 (plan) — every defensive rule this story implements is unexercised by real data, permanently.** The
  shipped corpus contains zero orphans, zero doubly-referenced nodes, zero nodes past two levels of
  nesting, zero bytes `≥ 0x80` in any name or string, and zero type-8 and type-10 nodes. So the depth
  cap, the single-parent rule, the orphan tolerance, the byte
  fidelity and both unrecognised/unsupported rejections are carried by
  synthetic criteria **alone**, and SC-11 cannot corroborate any of them. The **empty heap is the one
  exception and is not a risk at all**: 35 of the 44 shipped registries carry `heapSize == 0` with no
  type-0 or type-6 node — AC-9's exact shape — so that path is the best-corroborated rule in the
  story, not an uncorroborated one. *Mitigation:* each remaining item has a named
  criterion (SC-3…SC-8) and the spec already labels them as this project's own defensive engineering
  rather than claimed properties of the game's writer. This is recorded, not mitigated away.
- **R-3 (plan) — a type-10 node in a future registry fails the whole parse rather than degrading.** A modded or
  later-title registry carrying a float64 array is rejected outright, not skipped. That is the
  deliberate choice (DD10) — silently skipping a node would hand a caller an incomplete tree that looks
  complete — but it means the failure mode is total. *Mitigation:* the error names the type and the node
  index so the diagnosis is immediate, and DD10 carries the reversal recipe.
- **R-4 (plan) — `dump`'s output format is a plan-level contract, not a spec-level one.** FR-3 names four
  rendering properties and fixes none of their exact forms, so a later change could alter the byte-exact
  output without contradicting the spec. *Mitigation:* SC-9 pins the exact forms as tests, so a change
  is a deliberate test edit rather than a silent drift. **Accepted exposure:** `dump` is a developer's
  reading aid with no machine consumer, and the spec's own escape rule is not injective (DD14), so
  nothing should ever parse this output.
- **R-5 (plan) — the tool now depends on two format packages.** `cmd/regtool` importing `pkg/formats/res`
  (DD12) means a change to the archive reader can break the registry tool. *Mitigation:* the tool uses
  only `res.Open`, `Entries` and `ReadFile` — the same surface `cmd/restool` already depends on and the
  narrowest the job admits — and the library package `pkg/formats/reg` gains no dependency at all, which
  is the boundary AC-11 actually protects.
