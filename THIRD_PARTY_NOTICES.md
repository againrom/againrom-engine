# Third-party notices

againrom is licensed under GPL-3.0-or-later (see `LICENSE`). This file inventories
the third-party open-source modules compiled into againrom binaries, together with
their licenses; the full license texts live under `LICENSES/`.

## Compiled third-party modules

againrom depends on the Go standard library plus the modules below: CP866 string
decoding for the `pkg/formats` tier, CPU font rendering, and Ebitengine (with
its transitive dependencies) for the `pkg/ui` and `cmd` tiers.
Every `go test ./...` run cross-checks this list against `go.mod`'s `require` set
by way of `internal/notices`, which fails if the two disagree.

| Module | Version | License (SPDX) |
|---|---|---|
| `golang.org/x/text` | v0.40.0 | BSD-3-Clause |
| `github.com/hajimehoshi/ebiten/v2` | v2.9.9 | Apache-2.0 |
| `github.com/hajimehoshi/bitmapfont/v4` | v4.1.0 | Apache-2.0 |
| `golang.org/x/image` | v0.31.0 | BSD-3-Clause |
| `github.com/ebitengine/gomobile` | v0.0.0-20250923094054-ea854a63cce1 | BSD-3-Clause |
| `github.com/ebitengine/hideconsole` | v1.0.0 | Apache-2.0 |
| `github.com/ebitengine/oto/v3` | v3.4.0 | Apache-2.0 |
| `github.com/ebitengine/purego` | v0.9.0 | Apache-2.0 |
| `github.com/jezek/xgb` | v1.1.1 | BSD-3-Clause |
| `github.com/pierrec/lz4/v4` | v4.1.22 | BSD-3-Clause |
| `golang.org/x/sync` | v0.22.0 | BSD-3-Clause |
| `golang.org/x/sys` | v0.42.0 | BSD-3-Clause |
| `go.starlark.net` | v0.0.0-20260930220527-d7438c5a85ac | BSD-3-Clause |

The x/image license is the retained `LICENSES/BSD-3-Clause.txt` (Copyright
2009 The Go Authors). The lz4 module's full text, including Copyright (c)
2015 Pierre Curto and its named endorsement clause, is retained separately
in `LICENSES/lz4-BSD-3-Clause.txt`. The Starlark interpreter's license (Copyright (c) 2017 The Bazel Authors) is retained in
`LICENSES/starlark-BSD-3-Clause.txt`. bitmapfont uses `LICENSES/Apache-2.0.txt`.

## Source-derived (ported) code

Some packages are a from-scratch Go port of a third-party C library's
algorithm rather than a compiled Go module: there is no `go.mod` `require`
line to cross-check, so `internal/notices` covers this table separately (see
below) instead of through the module-vs-`go.mod` diff above.

| Path | Library | Version | Author | License (SPDX) |
|---|---|---|---|---|
| `pkg/video/smacker` | libsmacker | 1.2.0 | Greg Kennedy | LGPL-2.1-or-later |

`pkg/video/smacker` decodes the Smacker (`.smk`) cutscene container and its
DPCM audio in pure Go, ported file-by-file from libsmacker 1.2.0's C source
(upstream: `libsmacker.sourceforge.net`) for auditability against the
original. The original library is LGPL-2.1-or-later; this port carries the
same license, and is compatible with distributing the whole againrom binary
under GPL-3.0-or-later. Every ported file under `pkg/video/smacker` carries a
header citing this origin; the full license text is `LICENSES/LGPL-2.1.txt`.
This package is not a research artifact: it is third-party derived code under
attribution, not evidence about ROM1's own Smacker usage.

## Regenerating this file

When a compiled dependency is added:

1. Run `go list -m all` (and `go list -deps ./...`) to see the modules actually
   built into the binaries.
2. For each third-party module, add a row above with its module path, version,
   and SPDX license identifier, and place the module's full license text under
   `LICENSES/<SPDX-ID>.txt`.
3. Keep this list exactly in sync with `go.mod` / `go.sum` — no stale or missing
   entries (spec AC-3).

When a third-party algorithm is ported by hand instead of vendored as a
module, add a row to "Source-derived (ported) code" above, place its license
text under `LICENSES/`, and add an entry to the `portedSources` registry in
`internal/notices` so `TestPortedSourceNoticesPresent` keeps this section
honest the same way `TestNoticesMatchGoMod` keeps the module table honest.
