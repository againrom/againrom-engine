# Provenance — `.res` archive reader

## Backing

| `spec.md` anchor | Claim | Confidence |
|---|---|---|
| Signature `&YA1` | `RES-MAGIC-001` | High |
| Header — registry offset, opaque words, `nodeCount` | `RES-HDR-003`, `RES-HDR-004`, `RES-HDR-005`, `RES-HDR-006`, `RES-HDR-012`, `RES-HDR-013` | High for the field semantics; see *What the second release cost* for the two clauses that were not |
| FR-6 / FR-3 — the registry is `nodeCount` records at `regOffset`, the tail is ignored, the magic is the only assert the format makes | `RES-ACCEPT-031`, `RES-OPEN-026`, `RES-OPEN-027` | High (instruction-level; the acceptance predicate and the enumeration that the module cannot observe EOF) |
| The archive that discriminates it — a valid 504-node registry plus 23 stale bytes | `RES-GEOM-028` | Medium (corpus measurement, exact; the overwrite-layer reading is the Medium part and the reader does not depend on it) |
| FR-2 — the whole-path fold exists in the original and is required | `RES-CASE-036` | High |
| Node record — `[+0 reserved][+4 off][+8 size][+0xC type]`, name at `+0x10` | `RES-NODE-008`, `RES-NODE-011`, `RES-NODE-014`, `RES-HDR-017` | High |
| Name field — 16 bytes, cut at the first NUL, `0xCD` padding beyond it | `RES-NODE-007` | High |
| Roots & tree — roots are nodes never referenced as a child | `RES-TREE-009` | High |
| Tail-registry vs inline `&YA1` flavor | `RES-SCOPE-015` | High |
| Empty archive (`nodeCount == 0`) | `RES-SCOPE-010` | High |
| FR-2 — lookup folds ASCII `A`–`Z` and nothing else | `RES-LOOKUP-023` | High |

`RES-LOOKUP-023` is the one row where fidelity and our own convenience **coincide**, and the
coincidence is worth stating precisely: the original's in-tree child lookup is `_strnicmp` with an
ASCII-only fold, which is the rule FR-2 states. `RES-CASE-036` now extends that to the whole path.

## What the second release cost

Two clauses this contract was built on were **corpus facts about one packer, worded as format law**, and
both carry a `retracted.md` row: `RES-HDR-003`'s registry region `≡ 0 (mod 32)` and `RES-HDR-004`'s
`nodeCount = regLen/32`. Both were graded **High** while believed, and the grade was not careless — the
tail open really does consume both words, just without those identities. The field semantics of both
rows were never wrong and stand.

What replaced them is a different kind of evidence: `RES-OPEN-026` reads the acceptance predicate
instruction by instruction (the magic is the only value check; no instruction computes `EOF − regOffset`
or tests 32-byte alignment) and `RES-OPEN-027` enumerates that the archive module **cannot observe EOF at
all**. `RES-ACCEPT-031` grades the consumer contract that follows: assert the magic and the two header
words (High), and know that every further check is corpus hardening (Medium) that is stricter than the
original. The spec adopts it that way — FR-3's list past the signature is declared as ours.

The cost was concrete. Our reader asserted the mod-32 identity and rejected a file the original opens;
the Russian release's `MAIN.RES` failed with `res: registry length 16151 not a multiple of 32` until this
revision. The lesson belongs to the playbook, not here: a corpus of one packer's output cannot tell a
format rule from a packing habit, and a spec table's basis column is where the difference has to be
recorded.

`RES-PATH-025`'s lower-casing clause moved the other way — from *ours by choice* to *backed*. It claimed
no case-normalization pass exists anywhere, at High, on an import-table absence that was true and bounded
only four symbol names; the fold is an inlined loop over the CRT `tolower` no search for those names
could reach (`RES-CASE-036`, `retracted.md`). Our shipped lower-casing was right for a reason we did not
have, and its ASCII bound — `A`–`Z` and nothing else — is exactly what the original's `tolower` does.

## Ours by choice

None of these is asserted by any claim. Each is engineering we own, changeable by a later story
without contradicting research.

| What the spec fixes | What the evidence actually says |
|---|---|
| Names are decoded **CP866** | `RES-TEXT-021`: `rom.exe` applies **no** byte-to-character conversion to entry names — the node table including every name field is one verbatim bulk read, and no code-page API has any caller in the archive code range. Re-derived from `.res`'s own reader, not carried over from the `.reg` result. So CP866 is **our display convention**, not the format's encoding. |
| Lookup rewrites `\` → `/` | `RES-PATH-025` (the half that stands): both separators are **native**; the splitters test for each symmetrically at every position and neither rewrites one into the other. The rewrite buys the equivalence the original gets from testing both, and nothing else. |
| A registry the file does not hold is **rejected** | `RES-OPEN-026`: the original takes a short registry read silently — the byte count is discarded and the allocation is never zeroed — so it walks uninitialised memory. We reject; there is no third option a leaf reader can offer under P-2. |
| Comparison covers **whole names** | `RES-LOOKUP-023`: the original compares **15 characters**. No archive the format's own writer produced can distinguish the two, since the name field is 16 bytes with a NUL. |

The corpus cannot test any of it. `RES-TEXT-022`: across all 12 standalone containers and 4 592
nodes, **0 of 45 865 name bytes are `≥ 0x80`** — 55 distinct values in `0x27`–`0x7A`. Every candidate
code page that agrees on ASCII is indistinguishable here, and no shipped name exercises a fold beyond
`A`–`Z`.

## Open — assigned no meaning by the spec

| What | Status |
|---|---|
| Header `0x04`, and the three opaque header words | Value laws pinned on a widened 60-blob corpus of one packer's output — and `0x04`'s is refuted as format law by the second release, which stores an in-table index (`RES-HDR-029`); `0x0C`'s `{1, 17}` set survives while the "17 on graphics/main" association does not (`RES-HDR-030`). **Semantics remain open.** Read raw and ignored either way. |
| Node flag bits `0x20000000` / `0x40000000` | Undecoded. Not consulted. |

## Divergence recorded, not resolved

`RES-IDENT-024`: the **leading** path segment is matched against a resolving object's own stored
name by a **direct, case-sensitive byte compare** — a narrower rule than the in-tree fold, and one
this reader does not implement, because it indexes a single archive and has no notion of *which*
archive a path belongs to. That decision belongs to the **VFS story**, where several archives are
layered and a leading segment first has to pick one. Recorded here so the VFS story inherits it
rather than rediscovering it.

`RES-IDENT-034` now resolves that row's open half, and it is **assigned to the VFS story (0027), not
adopted here**. The rule is that an archive answers only to paths beginning with its own identity — its
filename cut at the first `.`, clamped to 15 characters — so the shipped archives are disjoint
namespaces and the manager's ascending scan is a dispatch, not a priority (`RES-ORDER-033`). Adopting it
in this story would mean the reader deriving an identity from the *filesystem path it was opened from*
and matching it before its own tree: a fact `OpenBytes` cannot know at all, and a rule about which of
several archives answers a name, which a single-archive reader has no set to choose from. It stays in the
tier that owns the set. `RES-COLL-038` is the same call for the same reason — a census over both installs
finding 0 identity collisions and 0 cross-archive path collisions is a statement about a *set* of
archives.
