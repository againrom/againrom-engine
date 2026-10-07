# Spec — multi-archive resolution by identity prefix

## Problem and current behaviour

`pkg/vfs` is a package comment and nothing else — no type, no function, no test. Every consumer that
needs bytes out of a game container holds one archive value and resolves against it by hand.

The archive reader indexes exactly one container. It normalises a lookup path by rewriting `\` to
`/`, trimming the outer separators and folding ASCII `A`–`Z` to `a`–`z`; a path it does not hold
yields no bytes and an error wrapping `fs.ErrNotExist`, and other reads yield a fresh copy of the
entry's bytes. Its entry listing is in registry order, unsorted. It has **no close operation** — the
whole file is read into memory and the host file closed before the archive value exists. And it has
no notion of *which* container a path belongs to: nothing in it names the file it was opened from.

The front-end opens three named containers and every caller then knows, out of band, which holds
what: the object and unit registries, the object sheets and the terrain strips from graphics, the
menu art from main, and the campaign maps from scenario under the entry paths an enumeration of that
container returned. **No address carries the container it belongs to**, so no single string
identifies an entry across the set, and a caller wanting a file the install ships loose beside the
containers resolves it through a second code path of its own.

## Functional requirements

- **FR-1** Opening MUST take an ordered list of archive host paths and an ordered list of
  directories, either or both of which MAY be empty, and MUST yield a read-only filesystem over
  them. Each archive's **identity** MUST be derived from its host path alone: the text after the
  last `\` or `/`, truncated before the first `.`, folded, and cut to its first 15 bytes. An archive
  whose derived identity is empty MUST be refused.
- **FR-2** If any listed archive cannot be opened, or its identity is empty, the open MUST fail,
  yielding no filesystem a caller can read from and retaining no listed archive's bytes. A listed
  directory MUST NOT be validated at open time.
- **FR-3** An **address** is an identity segment, a separator, and a non-empty remainder. `\` and
  `/` MUST be interchangeable at every position, and the whole address — identity segment
  included — MUST be folded before anything is compared, ASCII `A`–`Z` to `a`–`z` and every other
  byte as itself. An address whose leading segment is empty, i.e. one beginning with a separator, is
  not addressable and MUST be answered as absent.
- **FR-4** An archive MUST answer an address only if its identity **equals** that address's leading
  segment. The leading segment is then consumed and the remainder resolved inside that archive. An
  address naming no registered identity MUST NOT be looked up inside any archive.
- **FR-5** Archives sharing one identity MUST be consulted in list order, the **first** holding the
  remainder answering. This is the only circumstance in which list order changes any outcome.
- **FR-6** An address that no archive and no directory holds MUST yield no bytes and an error
  satisfying `errors.Is(err, fs.ErrNotExist)` that names the address.
- **FR-7** A source that holds an address but fails to produce its bytes with an error that is
  **not** "not found" MUST surface that error, and resolution MUST NOT continue to any further
  archive or to the directory tier: no copy elsewhere may mask a source that failed.
- **FR-8** When no archive answers, the folded address MUST be resolved as a **relative path** under
  each listed directory in order, the first readable file answering. This tier MUST be consulted for
  every address, including one whose leading segment equals no identity or that has no separator at all.
- **FR-9** Enumeration MUST list every unique address the archives serve exactly **once**, in
  ascending order of the folded address, each entry naming the archive serving it and the size of
  its bytes. Reading an enumerated address MUST return the bytes of exactly the archive the
  enumeration named. Enumeration MUST NOT list the directory tier.
- **FR-10** For any address the filesystem MUST report which source serves it — the archive, named
  by its identity and its position in the list, or the directory, named by its position — or a clear
  absent answer, without returning any bytes. Where a read succeeds, that report and that read MUST
  name the same source.
- **FR-11** Every read MUST yield bytes the caller owns: mutating them MUST NOT change what a later
  read of the same address returns. No operation may write to an archive, to an entry of one, or to a
  file under a listed directory.
- **FR-12** Every resolution of a container entry in this repo's library packages MUST become an
  address carrying that container's identity segment and MUST reach the entry it reaches today. No
  decoded or rendered result may change, and no library caller may be left resolving an entry by a
  path with no identity segment.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | no archives and no directories | opened, then any address read | the open succeeds and the read yields no bytes and `fs.ErrNotExist` |
| **AC-2** | unit | archives from host paths `Graphics.RES`, `sub.d/graphics.res`, and one whose basename stem is 20 bytes | their identities inspected | the first two are `graphics`, the third its folded stem's first 15 bytes |
| **AC-3** | unit | one archive from `graphics.res` holding `terrain.3d/dirt.bmp` | read at `graphics\terrain.3d\dirt.bmp`, at `GRAPHICS/TERRAIN.3D/DIRT.BMP`, and at `terrain.3d/dirt.bmp` | the first two return that entry's bytes and the third yields `fs.ErrNotExist` |
| **AC-4** | unit | archives from `graphics.res` and `main.res`, each holding `x/y.bmp` with different bytes | both addresses read, then the list reversed and read again | each address returns its own archive's bytes, unchanged by the order |
| **AC-5** | unit | two archives with equal host basenames, both holding `x.bin` with different bytes | read, then the list reversed and read again | the earlier archive answers each time, so the bytes swap with the order |
| **AC-6** | unit | one archive from `graphics.res` holding `x.bin` at its root | `\graphics\x.bin` and `/x.bin` read | both yield no bytes and `fs.ErrNotExist` |
| **AC-7** | unit | one archive from `graphics.res` holding `x.bin` at its root, and no directories | `nosuch\x.bin` read | no bytes and `fs.ErrNotExist` |
| **AC-8** | unit | a list of three archives whose second holds malformed bytes | opened | the open fails and yields no filesystem to read from |
| **AC-9** | unit | a host path whose basename is `.res` | opened | the open fails |
| **AC-10** | unit | two listed directories, the earlier holding the address as a permission-denied file, the later a readable copy | that address read | the permission error surfaces and the later source's bytes are not returned |
| **AC-11** | unit | two archives with distinct identities, one internal path present in both | enumerated, then every listed address read | each address appears once, the listing ascends by folded address, and every read returns the bytes of the archive the listing named |
| **AC-12** | unit | two archives with distinct identities and one directory | an address from each archive, one served by the directory, and one absent address located | each report names the source a read of it uses, and the absent address reports absent |
| **AC-13** | unit | one archive from `graphics.res` holding `x.bin`, and two directories, the first holding `graphics/x.bin` with other bytes, the second `graphics/y.bin` | both addresses read | `x.bin` returns the **archive's** bytes and `y.bin` the second directory's file |
| **AC-14** | unit | any address that reads successfully, and a listed directory | the returned bytes mutated and the address read again, the directory's contents compared across every read and enumeration | the second read is unaffected, and no file under the directory was created, removed or changed |
| **AC-15** | unit | every address a library caller resolves, each with a synthetic archive holding that entry under the matching identity | resolved through the filesystem | every address carries an identity segment, and each yields the bytes a direct read of that container's own path yields |

**Error cases:** AC-6 no identity segment · AC-7 an unregistered identity · AC-8 and AC-9 a failed
open · AC-10 a source failure that must not be masked.

## Derived properties

- **P-1 (invariant)** — For any address that reads successfully, the source report names where those
  bytes came from; and if that is an archive, the address occurs exactly once in the enumeration,
  carrying that archive.
- **P-2 (negative-invariant)** — For any address whose read fails, no bytes are returned and nothing
  in any archive or under any listed directory changes.
- **P-3 (idempotence)** — For any filesystem, two reads of one address return equal bytes and two
  enumerations return equal listings.
- **P-4 (completeness)** — Every address resolves in exactly one way: through one archive whose
  identity it names and which holds the remainder, else through one listed directory holding it, else
  absent. At most one archive ever answers a given address.
- **P-5 (invariant)** — For any two listed archives with **distinct** identities, every address
  resolves identically however those two are ordered in the list.
- **P-6 (negative-invariant)** — For any open that fails, no filesystem a caller can read from is
  returned and no listed archive's bytes are retained.

## I/O examples

An address is the container's identity, a separator, and the path inside it. Identity comes from the
host filename, so `…/graphics.res` answers `graphics\…` and nothing else. Note the third line:
`graphics` there is a directory **inside** `main.res`, not the graphics container.

```
graphics\terrain.3d\dirt.bmp      -> graphics.res, entry terrain.3d/dirt.bmp
graphics/units/units.reg          -> graphics.res, entry units/units.reg
main\graphics\mainmenu\menu_.bmp  -> main.res,     entry graphics/mainmenu/menu_.bmp
world\data\data.bin               -> no such identity -> a listed directory, else absent
\graphics\terrain.3d\dirt.bmp     -> not addressable
```

## Constraints

| # | Constraint | Alternatives and trade-off |
|---|---|---|
| **C-1** | This tier depends on the archive reader's public surface and the standard library, changing neither the reader nor any container format. | **(A) teach the reader its own identity** — it cannot have one: an archive built from bytes has no host path to derive it from, and a one-container reader has no set to choose within. **(B, chosen) resolve above it.** |
| **C-2** | There is no close operation and no resource to release: an archive holds no host handle once open. | **(A) close-all returning joined errors** — a no-op asserted to report failures, i.e. more than this tier can do. **(B, chosen) an atomic open (FR-2)**, which is the part that is real. |
| **C-3** | The directory tier looks up the **folded** address as a relative host path. | A case-insensitive walk instead costs a directory read per segment and a rule for which of two differing files wins; not taken. |
| **C-4** | The archive set is exactly what the caller lists, in the order given. | No discovery — no directory scan, no registry value, no defaulted extension, no list file. Each makes resolution depend on the host environment rather than the argument. |

## Out of scope

- **Masking.** The engine's one override mechanism is driven by a file no shipped install carries. It
  is not built, and that is an absence in the data rather than a gap here.
- **Registering an archive partway through a run**, and reopening one.
- **Writing.** No entry is added, replaced or removed and no container is created.
- **Nested containers.** A container stored as an entry of another stays one opaque blob.
- **The per-format developer tools.** They address one container directly and keep taking a bare
  entry path; FR-12 binds the library packages.

**Disclosed limitations**, accepted and owned:

- an address beginning with a separator, and an archive whose identity would be empty, are both
  refused rather than resolved;
- on a case-sensitive host a loose file whose on-disk name is capitalised is not found (C-3).

## Verification mapping

Every criterion is a unit test over synthetic containers built in test code and over directories
created in the test's own temporary tree: no game install is read and no container bytes committed.
AC-1 … AC-15 are therefore all CI-automatable. P-1 … P-6 are universals and are **sampled, not
proved**; nothing here claims otherwise. A figure taken against a real install is evidence about that
install and no part of a criterion.

## Gate check

FR-1 → AC-1, AC-2, AC-9 · FR-2 → AC-8, AC-9, P-6 · FR-3 → AC-3, AC-6 · FR-4 → AC-3, AC-4, AC-7, P-4 ·
FR-5 → AC-5, P-5 · FR-6 → AC-1, AC-7, P-2 · FR-7 → AC-10, P-2 · FR-8 → AC-13, P-4 ·
FR-9 → AC-11, P-1, P-3 · FR-10 → AC-12, P-1 · FR-11 → AC-14, P-3 · FR-12 → AC-15.
