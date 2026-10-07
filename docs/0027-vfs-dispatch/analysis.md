# Analysis — multi-archive resolution by identity prefix

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-anchored / static** — the profile names this tier by hand as a cross-cutting engine contract; no watcher tool exists, so it is discipline |
| Terrain — `pkg/vfs` | **greenfield**: the package is a doc comment and nothing else, with no behaviour to preserve |
| Terrain — the archive call sites in `pkg/game`, `pkg/render/menu`, `pkg/render/terrain` | **brownfield**: they resolve entries today, and every address they use changes when this rule lands |

## The premise was refuted, and our own baseline already held the refutation

The story existed to make `patch.res` shadow same-named entries in the base archives. Research
refuted it outright: an archive answers only addresses beginning with its own identity name, so the
shipped archives are disjoint namespaces and no shadowing is expressible; `patch.res` holds one
entry, a release-notes text; the one override mechanism is `update.lst`, and neither install ships
one.

The baseline's own research section recorded that this build's `patch.res` "adds rather than
overrides on this install" and that the override was "proven only synthetically … not yet observed
shadowing real data end to end" — then filed both as a disclosed corpus-coverage limitation. That is
the shape worth carrying out of this story: **a synthetic proof of a mechanism nothing real
exercises is evidence about our code, not about the game.** When the corpus cannot show a mechanism
firing, the open question is whether it exists, not whether we sampled enough. Two clauses of the
surviving contract fail exactly that test below, and are cut rather than kept for symmetry.

## What the baseline names that this repo's reader cannot express

Re-derived against `pkg/formats/res` as it stands, not taken as given:

- **There is nothing to close.** `res.Archive` has no `Close` method and holds no host handle:
  `res.Open` is `os.ReadFile` followed by `OpenBytes`, so the file is closed before the archive
  value exists. "Every archive already opened is closed, no leaked handle" and "closing closes every
  archive even if some fail, returning the failures joined" therefore describe an operation that
  cannot exist and failures that cannot occur. A close that cannot fail, asserted to join failures,
  is a contract with nothing behind it. The half that survives is atomicity: a failed open returns
  nothing usable and retains nothing.
- **`ReadFile` has exactly one error path.** It returns either the bytes or an `*fs.PathError`
  wrapping `fs.ErrNotExist`; no other failure is reachable. So "an archive that holds the path but
  fails with a non-not-found error must not fall through" has no witness through the shipped reader.
  It gains one from the loose-file tier, which reads the host filesystem and can fail on permission,
  on a directory standing where a file was expected, or on IO.
- **`Entries()` is in registry (node) order**, unsorted, and where a container holds one path twice
  the first record wins. A stable enumeration order is ours to impose, not something to pass through.
- **The whole-path fold is already in the leaf.** `res.normalize` rewrites `\` to `/`, trims the
  outer separators and folds ASCII `A`–`Z` byte by byte, and every lookup goes through it. What this
  story adds to the folded region is the identity segment; the fold itself is unchanged, and the
  reader's lower-casing — carried as ours by choice in `0001` — is now backed.

`OpenBytes` cannot know the identity of the file it was opened from, and a reader indexing one
container has no set to choose within. That impossibility is the layering argument, and
`docs/0001-res-archive/provenance.md` records it under Divergence; it is inherited here rather than
re-derived.

## The prefix is which archive the caller means, never something read off the path

`pkg/render/menu` reads its assets from **`main.res`** under `EntryPrefix = "graphics/mainmenu/"`.
Under this rule its addresses become `main\graphics\mainmenu\…` — the case research names with the
shipped literal `main\graphics\chrgen\leftup.bmp`, where `graphics` is a subdirectory of `main`,
present in our own tree. Prepending `graphics\` there, or inferring a prefix from a path that
already begins with the word, yields an address resolving to the wrong archive or to none. So every
call site has to be told which container it means:

| Call site | Container | Path today | Address after |
|---|---|---|---|
| `pkg/game/statics.go` | graphics | `objects/objects.reg` | `graphics\objects\objects.reg` |
| `pkg/game/units.go` | graphics | `units/units.reg` | `graphics\units\units.reg` |
| `pkg/game/statics.go` sheet cache | graphics | registry `File` values | `graphics\` + the value |
| `pkg/render/terrain` | graphics | `terrain.3d/*.bmp` | `graphics\terrain.3d\*.bmp` |
| `pkg/render/menu` | main | `graphics/mainmenu/*.bmp` | `main\graphics\mainmenu\*.bmp` |
| `pkg/game/maplist.go` | scenario | enumerated entry paths | enumerated addresses |

`maplist.go` enumerates one container and feeds the paths it gets straight back into a read, so
enumerate-then-read is a live path here and not a hypothetical. `pkg/game/{maplist,frontend}.go`
also already hand-roll the two-tier resolution — an archive entry, else a file under the asset root
— which is the loose-file tier written out once per caller.

## Two escapes from disjointness the re-scope does not name

`RES-IDENT-034` describes the identity compare as skipped in two cases, and each turns the archive
list back into a genuine priority rather than a tiebreak:

- an address that **begins with a separator**: the identity is not compared at all, every archive is
  searched from its own root and the first match wins. Research names this as its own open item —
  whether any runtime-constructed path can be shaped that way — and no shipped literal is;
- an archive whose derived identity is **empty** (a basename beginning with `.`): it answers every
  address, so it shadows every archive listed after it.

Both are refused by the contract rather than implemented, on the test above: nothing real exercises
either, so building them would be evidence about our code alone. A third reading is genuinely
ambiguous in the claim's wording — whether a leading segment that is a strict *prefix* of an
identity matches, since the compare is described as stopping both when the identity is consumed and
when the path reaches a separator. The contract requires equality and records the ambiguity rather
than settling it by preference.

## What we looked at

`pkg/formats/res/{res.go,res_test.go}`, `pkg/vfs/doc.go`,
`pkg/game/{archives,statics,units,maplist,frontend}.go`,
`pkg/render/{menu/menu.go,terrain/tileset.go}`, `cmd/restool/main.go`, `internal/archtest/dag.go`,
`docs/0001-res-archive/{spec,provenance}.md`, `docs/ARCHITECTURE.md`, `AGENTS.md`, both check
scripts, and in research `claims/retracted.md` first, then `claims/res.md`.
