# Third-party license texts

This directory holds the full license text of each third-party module compiled
into an againrom binary — shared texts are named by their SPDX identifier
(for example `BSD-3-Clause.txt`). `../THIRD_PARTY_NOTICES.md` inventories those
modules and points here.

Module-specific copyright or clause variants keep a separate file.
`lz4-BSD-3-Clause.txt` preserves the lz4 module's exact full license.
`starlark-BSD-3-Clause.txt` preserves the go.starlark.net module's license (Copyright
(c) 2017 The Bazel Authors).

It also holds the license text for third-party **source-derived** code ported
into this tree by hand rather than pulled in as a Go module (no `go.mod`
`require` line, so `internal/notices`' module-cross-check does not see it).
`LGPL-2.1.txt` is one such entry: `pkg/video/smacker` is a from-scratch Go port
of libsmacker 1.2.0 (Greg Kennedy), LGPL-2.1-or-later in its original C form (its `smacker.h` header grants version 2.1 or any later version); see
`../THIRD_PARTY_NOTICES.md`'s "Source-derived (ported) code" section.

The project's own license (GPL-3.0-or-later) lives in the top-level `LICENSE`
file, not here.

When a compiled module dependency is added, place its license text here as part
of the regeneration procedure documented in `../THIRD_PARTY_NOTICES.md`.
