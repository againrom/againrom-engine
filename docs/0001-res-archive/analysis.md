# Analysis — `.res` resource archive container (ROM1)

## Source & confidence

All facts below are game-derived by the research team, not by this repo's own
reverse-engineering. Provenance chain:

- **Submodule pin:** `research/` at `3dcf8f7` (`againrom-research`, module `rom1research`).
- **Experiments:** `research/experiments/EXP-0001-res-container/` (the container), promoted to
  `research/formats/res/format.md`; `EXP-0014-res-opaque/` (the opaque words' value laws +
  the two-flavor `&YA1` scope, on a widened 60-blob corpus).
- **Corpus:** all **11 shipped `*.res` archives + `KIDS.LM`** (a 12th, empty archive).
- **Strength:** **0 invariant violations across the corpus; falsification passed** (empty /
  min / corrupt-synthetic / same-header-different-body probes). This is the promoted,
  evidence-backed layer (research "level 3"), not a hypothesis.

## Claim inventory (what the spec is built on)

| Claim | Statement | Confidence | Used by spec |
|---|---|---|---|
| RES-MAGIC-001 | Magic `26 59 41 31` (`"&YA1"`), LE u32 `0x31415926` | High | signature check (FR-3) |
| RES-HDR-002 | Header is 24 bytes; payload data begins at `0x18` | High | data region `[0x18, regOffset)` |
| RES-HDR-003 | `u32@0x10` = registry offset; registry `[regOffset, EOF)` length ≡ 0 (mod 32) | High | geometry (FR-1, FR-3) |
| RES-HDR-004 | `u32@0x14` = node count = `regLen/32` | High | `nodeCount` cross-check (FR-3) |
| RES-HDR-005 | `u32@0x08` = root count (top-level / unowned nodes) | High | root-count cross-check |
| RES-HDR-006 | `u32@0x04` and `u32@0x0C` **semantics unknown**; value laws pinned → RES-HDR-012/013 | **Unknown (meaning)** | opaque (R-1) |
| RES-HDR-012 | `@0x04 ∈ {0, non-root-node count}`; filled only by graphics/main/sfx; not recomputed, not a checksum, no structural predictor for populate-vs-zero | High (law) / Unknown (why) | opaque (R-1) |
| RES-HDR-013 | `@0x0C` per-file constant ∈ {1, 17}; bit 4 (`0x10`) is the discriminator; a format-variant / writer-generation flag | High (domain) / Unknown (meaning) | opaque (R-1) |
| RES-NODE-007 | Node = 32 bytes `{u32, u32 off, u32 size, u32 type, char[16] name}`; name `0xCD`-padded | High | node parse, CP866 name (FR-1) |
| RES-NODE-008 | `type` 0=file (byte range in `[24, regOff)`), 1=dir (first-child index + count) | High | tree walk, range validation (FR-1, FR-3) |
| RES-TREE-009 | Nodes form a tree; corpus-wide all reachable, no cycles; file ranges tile `[24, regOff)` exactly | High | bounded total walk, P-1/P-2 |
| RES-SCOPE-010 | `.LM` = same container (empty archive valid); `Allods/*.RES` = identical container | High | empty-archive case (AC-9) |
| RES-NODE-011 / -014 | Node `u32@0x00` reserved/always-0 across all 4592 tail-registry nodes (hash/id refuted); semantics unknown | High (always-0) / Unknown (meaning) | opaque (R-1) |
| RES-SCOPE-015 | `&YA1` is **two** formats: tail-registry archive (`@0x10` = byte offset; this spec) vs inline REG-style store (`@0x10` = record count); every *nested* `&YA1` is the inline flavor | High | scope note (spec), reader boundary |

## Decoded-vs-open reconciliation

One correction fell out of writing the spec against this research, worth recording because a
naive reading of the header would get it wrong:

- **Roots are found by *exclusion*, not by a header-pointed range.** The root set is exactly
  the nodes never referenced as a child of any type-1 node (RES-TREE-009); `rootCount`
  (`0x08`, RES-HDR-005) only *cross-checks* their count. Header `0x04` is **not** a root
  pointer — it is opaque (RES-HDR-006, its observed `nNodes−roots` value is a coincidence of
  arithmetic, not a decoded field). The spec's root-by-exclusion walk (FR-1) and its opaque
  treatment of `0x04` both come from this.
- **`0x14` is decoded, not opaque.** `u32@0x14` = `nodeCount = (EOF − regOffset)/32`
  (RES-HDR-004) and the reader *uses* it — as a cross-check that must agree with the measured
  registry geometry, never as a trusted bound on its own (a disagreement is an FR-3 rejection).
- **The three opaque words got value laws, not semantics (EXP-0014).** A widened 60-blob survey
  pinned `@0x04 ∈ {0, non-root-node count}`, `@0x0C ∈ {1,17}` (bit 4 = variant flag), and node
  `@0x00` = reserved/always-0 (a per-node hash/id was refuted) — but *why* each is set stays open.
  The reader's read-and-ignore choice is unchanged; this only tightens the confidence behind "safe
  to ignore."
- **`&YA1` is two formats (RES-SCOPE-015).** This reader is the tail-registry flavor (`@0x10` = byte
  offset), which appears *only* as a standalone top-level `.res`/`.LM`; nested `.reg`/save blobs are
  the inline flavor (`@0x10` = record count). So the reader must never be applied to a nested `&YA1`
  blob — a scope boundary now stated in the spec.

## Open items (flagged to the research team — R-1)

Three words carry no meaning the reader needs; EXP-0014 pinned their **value laws** but their
**semantics stay open**, so this is **non-blocking** and they are read raw and ignored:

- header `0x04` — RES-HDR-012, `∈ {0, non-root-node count}` (filled only by graphics/main/sfx; no
  structural predictor for why)
- header `0x0C` — RES-HDR-013, per-file constant `∈ {1, 17}` (bit 4 = format/writer-variant flag)
- node `0x00` — RES-NODE-014, reserved/always-0 across all 4592 tail-registry nodes (hash/id refuted)

What stays a **research-team** item: *why* `0x04` is populated for only three archives, and *what* the
`0x0C` variant flag / the reserved words mean — RE of `rom.exe`'s archive-open/-write routine. Related
scope fact from the same experiment: nested `.reg`/save `&YA1` blobs are the *inline* flavor
(RES-SCOPE-015), a separate format investigation (story 0011), not this container.

When the research team closes any of these, the flow is: repull the submodule
(`git submodule update --remote research`), derive the newly-decoded field here, then update
`spec.md` to state it as established and resolve the matching *Research needed* item.
