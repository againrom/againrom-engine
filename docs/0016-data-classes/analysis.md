# Analysis — typed data classes from the graphics registries

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **research-first / static** — the owner's call, taken so FR-3 would not have to be scoped down to what `pkg/formats/alm` exposes today |
| Terrain — `pkg/data` | **greenfield**: the package is `doc.go` and nothing else |
| Terrain — `pkg/formats/alm`'s placed-class fields | **brownfield, and deliberately not opened here** — see *The sequencing problem* |

## The research pin

This story is pinned to research **`e61153d`** (`master`, 2026-07-27), frozen for its whole duration;
`PIPELINE-STATUS.md` tracks no per-story pin, so this line is the record. Its **revision** runs at
**`9c01af7`**, the next pin and frozen in turn — what that settles is the last section here.

The tree was on `7d06122`, which predates both claims the contract turns on: `REG-KEY-044`'s
2026-07-27 re-audit and the new `REG-KEY-045`. Written against it, this spec would have resolved
`Parent` as a **section index** — a reading that pin carried at **High** and `retracted.md` now
lists as withdrawn, with the "a placed class id must be translated" consequence drawn from it.
Bumping first was the difference between a correct loader and a confidently wrong one.

## What we did not know before that pin

1. **Is `Parent` a section index or an `ID`?** An `ID` — `units.reg` and `structures.reg` store
   their class arrays *by* `ID`, and only `objects.reg` appends in order: the one registry where the
   two readings are indistinguishable (`ID == index` on 82/82), and the one the retracted reading
   was taken from.
2. **Does an explicit `= ""` clear an inherited array?** The baseline said yes and named `Unit33`.
   It does not: the engine tests the array's *length after the read*, so the empty value falls
   through the guard and `Unit33` inherits `Unit0`'s track unchanged. There is **no representation
   of "clear this array"** in the format.
3. **Do scalars and arrays share one inheritance guard?** Nobody had asked. They do not: scalars
   fall back only when the record is **absent**, arrays whenever the read yields **length 0**. This
   is the trap of the story — a loader resolving inheritance by testing an `int` field's zero value
   gets the scalar case wrong, and the shipped registry discriminates.
4. **Is the "units inherit `File`, objects do not" asymmetry real?** Unanswerable from the data and
   deleted: every class in both registries writes its own `File`, so no shipped record could
   contradict it either way and the only cross-check was a third-party reference. **Answered at the
   revision's pin** — below.
5. **Do inherited arrays chain to a grandparent?** No — the engine re-reads the *parent's own
   `.reg` section*, so one hop. Unreachable on shipped data: `objects.reg` has exactly one depth-2
   chain and it inherits no array.

Key *meanings* stay unasked, deliberately: the loader reads every key verbatim and computes on none,
so a wrong meaning misleads a comment and cannot corrupt a value. `REG-LOC-016` carries at High that
the names are the game's own ASCII labels; every meaning the baseline attached came from a
third-party reference and stays unconfirmed.

## What we measured ourselves

`go run ./cmd/regtool dump` and `./cmd/restool list` over the owner's install, asset root on the
command line, dumps outside the tree. `provenance.md` carries the ledger; what bears on the five
questions:

- Every class in both inheriting registries writes its own `File` — 16/16 `Parent`-carrying units,
  39/39 `Parent`-carrying objects — so question 4 has no witness either way.
- Depth: units `{0:18, 1:16}`; objects `{0:43, 1:38, 2:1}`. The one depth-2 chain is
  `Object81` -> `Object78` -> `Object77`, neither of the first two carrying any array key, so
  question 5 has nothing to act on.
- No parent's section index is >= its child's, in either registry, so one forward pass over
  `0..N-1` resolving by `ID` suffices and a forward or dangling `Parent` is malformed.
- Question 3 is live on shipped data twice: `Unit21`/`Unit33` write `AttackDelay = 0` over a parent
  setting `4`, and `Object40`/`Object42` write `Parent = 0` — a real reference to `Object0`, objects'
  `ID` domain being 0-based. Both are what a zero-value test gets wrong.
- The engine's class array is `ID`-keyed and sparse (81 slots for 34 units, 47 NULL) and the node
  table is name-sorted (`Unit0, Unit1, Unit10, … Unit2, …`), so the owner's numeric section order is
  the dense enumeration and reaching it is a real sort.

## The sequencing problem we did not solve here

FR-3's `-sweep` needs all three placement kinds. Two were reachable when this was written —
`Map.Overlay` and `Object.Kind` — and the type-6 unit class id at file `+0x08` was not: `alm.Unit`
was `{X, Y}`. The spec carried that as a declared precondition rather than inventing a workaround or
narrowing FR-3, and a 0003 revision exposed `alm.Unit.ClassID` before Stage 4.

## What we looked at, and what stays unknown

`research/claims/reg.md` (`REG-KEY-044`/`045`, `REG-LOC-016`, `REG-UNITS-018`, `REG-OBJ-039`,
`REG-STR-040`, `REG-VAL-029`, `REG-REC-032`, `REG-KIND-033`/`034`; at the revision's pin
`REG-OBJ-046`/`047`), `claims/alm.md` (`ALM-CLS-035`/`036`/`038`/`042`) and `claims/retracted.md`,
read first; `pkg/formats/reg`'s accessors, which already report presence — what the scalar guard
needs; `pkg/formats/alm`'s `Map.Overlay`, `Object` and `Unit`; `pkg/vfs`, still a stub, so this story
builds on `pkg/formats/res` where it needs an archive.

Unknown and disclosed rather than guessed: every key meaning; whether the engine's absent-array path
ever reads stale residue from the two `CArray` objects it reuses across six keys (`REG-KEY-045`'s
own Unknown — we do not emulate it); and the 64 type-3 cells in two maps whose code resolves past the
registry, which `ALM-CLS-035` grades Unknown as to whether the engine reads them at all.

## What the revision's pin settles

The first pass never asked what a key set by **no** section on a class's chain resolves to, so Go's
zero became the policy. `REG-OBJ-046` reads the whole `objects.reg` loader and answers it: every
scalar read carries a per-key default, and zero is a legal object `ID`. Measured on the install
before revising, at the story's landed code, that gave 33 of the 82 object classes a `DeadObject` of
`Object0` and 54 a `FireObject` of it — against `REG-OBJ-047`'s value spaces of `-1` on 54 and
`{-2 ×21, -1 ×61}`. The arithmetic closes exactly, which is what makes the fix falsifiable.

The same claim answers question 4 for one registry: `File`'s default is a literal passed whatever
the parent holds, so **`File` does not inherit in `objects.reg`**. Still unobservable — 0 of 39
`Parent`-carrying object classes omit it — so it lands because the contract was wrong, not because
an output moves. `units.reg`'s own defaults are decoded only *after* this pin and are not
implementable here; that registry keeps the zero and the contract says so, rather than letting the
objects table read as general.
