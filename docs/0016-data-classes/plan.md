# Plan — typed data classes from the graphics registries

## Baseline

**Nobody had taken FR-3's census.** 0003 carries `R-6`: nothing counted how many of the 8094 shipped
type-6 records are overridden. DD-6 is that classification's design, SC-9 binds to it, and SC-10
records whatever the first count is — `0` for both included is the answer, not the absence of one.

**The revision's baseline is a measurement.** `pkg/data` loads all three registries and `classdump`
prints them; at the landed code an absent scalar takes Go's zero, so `classdump` reads
`DeadObject = 0` on 33 of the 82 object classes and `FireObject = 0` on 54. DD-9 is the only new
decision.

## Design decisions

### DD-1 — presence is read off the node, not from `Get*`

`GetInt`/`GetString`/`GetIntArray` return one `false` for three situations: section missing, key
missing, key present at the wrong type. The contract needs the third split from the first two — an
absent key **inherits**, a wrong-kind key is a hard error. So `pkg/data` resolves the section node and
then the key node itself over `Reg.Root`, under its own copy of the ASCII-only fold (`reg`'s is
unexported and `.reg` is frozen), reading `Node.Dir`/`Type`/`Int`/`Str`/`Ints`. Rejected: reading the
accessors' `false` as absent — it inherits over a malformed key, turning every wrong-kind fixture in
AC-5 into a silent pass. `Node.Ints` is the parser's own memory and one parent's array reaches every
child, so an array is **copied when taken**.

### DD-2 — two tables per class, `own` and `eff`; the guard is which one the parent is read from

Each registry has one **key table** of `{name, kind, ptr}` rows — `kind` int / str / array, `ptr` a
closure yielding that key's field address — the single place the inventory is written, so no key is
validated but never stored, or stored but never kind-checked. Stage 1 records, per section, `own[i]` =
the `*reg.Node` for row `i` in **that section alone**, or nil. Stage 2 fills `eff[i]`:

- **scalar**: `own[i] != nil` → the class's own value, `0` and `""` included; else the parent's
  `eff[i]`. Reading the parent's *resolved* table is what makes scalars chain, and the child's
  **presence**, never any value, decides.
- **array**: take `own[i]` (a nil node, or a `TypeString` node with empty `Str`, both length 0); length
  > 0 stands, else the parent's **`own[i]`** copied, or nil when that is 0 too.

The public struct is filled from `eff` only after every class is resolved, so at resolution time no
loaded field exists to test: a zero-value implementation of this design is unwritable. What it was
silent about is what `eff[i]` **nil everywhere** means — DD-9 answers that, and until it did the fill
fell through to the Go zero.

Rejected: option types on the public fields — FR-1 wants typed inventory fields, and they push presence
onto every consumer forever to serve one loader. Rejected: resolving arrays from the parent's built
struct — it holds what the parent *inherited*, so arrays would chain and AC-3 fail. Rejected: one
generic class type — the registries differ in inventory, `File`'s kind, `Parent`'s legality and sprite
prefix, so its parameters would carry nearly everything.

### DD-3 — sections are addressed by constructed name, so numeric order is the iteration order

The loader reads `[Global]`'s count, then loops `i` over `0..count-1` looking up `prefix + itoa(i)`;
it never walks `Root.Children`. `All()` is therefore numeric section order because
that is build order; a missing dense section is a lookup miss, an error at the index where it happens;
a stray extra section is ignored. Rejected: sorting `Root.Children` on a numeric suffix parsed from
each name — it needs a name parser and a policy for names that do not fit, and a *missing* `Unit17`
degrades from an error into "absent from the list". One descriptor beside each key table carries what
differs: count key, the two prefixes, whether `[Files]` exists, whether `Parent` is legal.

A collection holds one `[]*Class` in build order plus `map[int32]*Class` into it, so P-3 holds because
there is one list, not by a check. `All()` copies the pointer slice; `ByID` takes `int32`, the type
`reg` stores, so callers convert where the sign stays visible.

### DD-4 — `Parent` resolves through an ID→index map; cycles cannot arise

Stage 1 builds `byID: ID → section index` over every section, catching a duplicate `ID` there. Stage 2
resolves a **present** `Parent` against it: no entry → *unresolvable*; entry at index `>= i` →
*forward*. A cycle needs at least one edge to an index not below the child's, so **every cycle
contains a forward edge and is rejected as one**: no cycle detector exists, and AC-5's cyclic fixture
is met by the forward error, naming its section and `Parent`. `byID` is a Go map, so
`Parent = 0` is an ordinary hit and the sparse domains need no sentinel; nothing compares a
`Parent` value to zero.

### DD-5 — the sprite base is resolved at load into an unexported field

`SpritePath()`/`OverlayPath()` return `base + ".256"` and `base + "b.256"` from an unexported `base`
computed once at load: prefix + the `[Files]` entry at the resolved `File` index, or the structure's own
`File`, backslashes turned to `/`, case preserved. `File`'s bound and the `[Files]` entry's presence and
non-emptiness are checked there, the only place they can be. **Every exported field is an inventory key
and nothing else**, so P-2 stays literally true of the API.
Rejected: `Classes.SpritePath(id)` — it leaves a class handed out by `ByID` unable to answer for a
value it fully determines. A class resolving no `File` node has empty `base` and returns `""` from
both: *Validation* lists no missing `File`, so an error here would reject data the contract takes.

### DD-6 — four independent buckets; a diverted type-6 record is counted, never failed

`Flags & 1 != 0` and `DefID != 0 && DefID != 0xcdcdcdcd` are evaluated **independently**, giving
`direct` (neither), `npc`, `def` and `both`. Only `direct` records are resolved against `units.reg`;
only a `direct` record that misses is a failure. The independence is the decision: `alm.Unit` asserts
no precedence between the two paths, so a priority chain would invent an ordering research has not
published and would hide every record where the two coincide. `units.reg` is not the table either path
reads, so excluding all three buckets holds whichever wins. `ClassID` converts `int16 → int32`, so a
negative key misses `ByID` at its signed value instead of folding to ~65000 and missing for the wrong
reason.

Counts print as fixed greppable tokens, so evidence quotes rather than narrates: `type3 nonzero=
resolved= past-registry=`, `type4 refs= resolved= unresolved=`, `type6 records= direct= resolved=
unresolved= npc= def= both=`, plus how many diverted records' `ClassID` *would* have resolved anyway —
whether a divert is load-bearing. One line per source, then a totals block; each
failing reference gets a line naming source, kind, index and value. Sources are not de-duplicated —
that needs a content comparison and would hide a divergence.

### DD-7 — the `cmd/classdump` DAG rows land in the commit that creates the package

`internal/archtest` is fail-closed, so the instant `cmd/classdump/main.go` exists without a row in its
`allow` map, `go test ./...` is red. The row cannot be a task of its own, and `docs/ARCHITECTURE.md`'s
tier and dependency rows move in the same commit because that file's preamble binds it to the
allow-map. **The first task creating `cmd/classdump/` owns all three files**, with the entry
`cmd/classdump → pkg/data, pkg/formats/reg, pkg/formats/res, pkg/formats/alm` — `pkg/vfs` being a stub,
the tool reads archives through `pkg/formats/res`. `pkg/data`'s row is not edited: the `pkg/vfs` it
permits, which FR-1 forbids this loader, stays — the allow-map is a ceiling and SC-11 holds the import
set.

### DD-8 — the two readings *Validation* leaves open, and the error shape

Every other case lands structurally: the dense-section miss in DD-3, the `Parent` family in DD-4,
`File` and `[Files]` in DD-5, wrong-kind as one comparison against the key table's `kind` — excepting
the `TypeString` node with empty `Str` the contract lets stand for an array. Lengths are checked on the
**resolved** values, so an asymmetrically inherited pair is caught as the mismatch it is.

- `ShootOffset` **nil is legal** — some unit classes omit the key with no ancestor to take it from;
  only a resolved value of another length is malformed.
- `Sound`'s "always length 5" is a **measurement, not a rule** — *Validation* does not name it, so no
  length check is written: a rejection the contract never states fails data it accepts.
- Errors are `fmt.Errorf` with a fixed `<section>: <key>: <what>` prefix, the key omitted where the
  fault has none; no exported error type, nothing asking a caller to branch on a reason. Atomicity
  needs no mechanism: every failure path is `return nil, err`.

### DD-9 — a default table beside the key table; the guards read one more bool

A default is a per-key fact, so it goes where the per-key facts are: rows of `{key, value, noInherit}`
beside `objects.reg`'s key table, resolved once per load into two arrays over row indices. Stage 2's
scalar arm gains `&& !noInherit[k]` — the whole of `File`'s non-inheritance — and stage 3's
`eff[k] == nil` arm writes `defaults[k]` where it fell through to zero. `units.reg` and
`structures.reg` pass no rows, so they are unchanged by construction, not by a promise. Rows are per
key, not "`-1` unless listed", because that is the shape of the evidence: one immediate per read site.
A rule naming a key its table lacks is silent, as the length rules are; a test pins every row to a
`kindInt` row of that table.

**The default does not reach the sprite base**: `base` comes from the resolved `File` *node*, so a class
resolving none keeps an empty base and two empty paths (DD-5) instead of looking up `Files[-1]`.
Rejected: defaulting the node instead of the field — every `File`-less object class would become a load
error, rejecting data the contract takes. *Validation*'s bound stays a rule about a `File` a section
states.

## Success criteria

- **SC-1** AC-1's two children resolve `0` and `4`. Inheriting on a zero loaded field returns `4`
  twice.
- **SC-2** AC-2, all four children. Treating `""` as "clear" yields nil for the second.
- **SC-3** AC-3: the scalar chains, the array does not. Resolving arrays from the parent's *built*
  struct returns the grandparent's array.
- **SC-4** AC-4, both halves. Any "`0` means no parent" reading fails the first.
- **SC-5** AC-5 with one fixture per case named in *Validation*, counting the `Parent` family as four
  and both `[Files]` faults as two; none panics (P-1). Two more pin DD-8 from the other side — a class
  with no `ShootOffset` and one with a 4-element `Sound` both load clean.
- **SC-6** AC-6 over sections in name-sorted node order (`Unit0`, `Unit1`, `Unit10`, `Unit2`, …), with
  `ByID` missing `0` where no class holds it and one past the maximum (P-3). Enumerating
  `Root.Children` fails the order.
- **SC-7** AC-7 for a unit, an object and a structure, compared byte for byte over synthetic
  `*reg.Reg` bytes; nothing is opened.
- **SC-8** AC-8's three that are values and not errors, plus a class with **every** inventory key at a
  distinct sentinel against a written-out expected struct, so a dropped key shows as a zero and a field
  with no key as a surplus (P-2).
- **SC-9** The classifier is a pure function of `ClassID`/`Flags`/`DefID`, asserted over all four
  buckets, over `DefID` at both sentinels, and over a `ClassID` written `0x8001` that must read
  `-32767` and be reported unresolved at that value. A synthetic sweep whose only non-`direct` records
  are diverted exits **zero** with them counted; one unresolvable `direct` record exits non-zero
  (FR-3).
- **SC-10** The owner's install, manually (AC-9): `classdump` prints 34 / 82 / 66 classes; `-sweep`
  resolves every type-4 and every `direct` type-6 reference, exits zero, and prints the type-3 residual
  as a counted figure, not a failure. The census is recorded verbatim — `R-6`'s first count,
  `npc=0 def=0 both=0` included.
- **SC-11** `go list -deps ./pkg/data` names no `againrom/` package but `pkg/formats/reg` (FR-1); the
  loaders take a `*reg.Reg`, so no test there opens a file (P-5).
- **SC-12** AC-10, and a written `DeadObject = 0` still resolving to `0` (P-6). Falling through to the
  Go zero reads `0`, `0`, `0`; a unit class in the same shape still reads `0` — the scope limit, not a
  defect.
- **SC-13** AC-11. Nothing on shipped data can show it, so the fixture is the only witness there is.
- **SC-14** The owner's install, manually: `DeadObject` = `-1` on exactly **54** of the 82 object
  classes, `FireObject` = `-2` on **21** and `-1` on **61**, nothing else — value spaces measured
  independently of this loader, so the run reproduces them or the fix is wrong.

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-1, DD-2, DD-8, DD-9 | SC-1, SC-2, SC-3, SC-4, SC-5, SC-8, SC-11, SC-12, SC-13, SC-14 |
| FR-2 | DD-3, DD-5 | SC-6, SC-7 |
| FR-3 | DD-6, DD-7 | SC-9, SC-10 |
