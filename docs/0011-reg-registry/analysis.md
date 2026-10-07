# Analysis — `.reg` binary registry parser (ROM1)

## Intensity & terrain (declared for the whole work item)

- **Intensity: `spec-anchored / static`.** Per `SDD/PROFILE.md`, `pkg/formats/*` is spec-anchored: the
  format spec is the durable contract and doubles as our reverse-engineering documentation. It outlives
  this ticket — `pkg/data` (0016), the map-object stories and the VFS story all read through it, and the
  spec is the artifact a third party would implement a parser from. No watcher tool exists in this repo,
  so synchronization is **static** — discipline, not tooling — and is labelled that way.
- **Terrain: greenfield.** `pkg/formats/reg/doc.go` and `cmd/regtool/main.go` are package skeletons
  reserved by story 0000's architecture DAG, not an implementation: `doc.go` is a package clause with a
  tier comment, `main.go` prints `regtool (skeleton)`. There is no shipped `.reg` behavior to preserve
  and no characterization burden. The DAG entry `cmd/regtool → pkg/formats/reg` already exists in
  `internal/archtest/dag.go` and `docs/ARCHITECTURE.md`, and `internal/archtest` is fail-closed on any
  unlisted package — so the tool name is fixed by the DAG, not by this spec's preference.

## Source & confidence

- **Submodule pin:** `research/` at `8b14881` (`againrom-research`, module `rom1research`), frozen for
  the rest of this story's duration (S-5). Not bumped further, not fetched forward. The pin was moved
  once, from `8a406d6`, to take in the experiment that answers this story's two open research items;
  every fact below is read at `8b14881`.
- **Research artifacts consulted:** `research/formats/reg/format.md` and `research/claims/reg.md`
  — `REG-LOC-016`/`REG-LOC-038` for location, `REG-VAL-024`…`REG-VAL-030` for the value, pool and tree
  encoding, and `REG-FMT-031`, `REG-REC-032`, `REG-KIND-033`, `REG-KIND-034`, `REG-DBL-035`,
  `REG-TEXT-036`, `REG-TEXT-037` for the framing, record layout, kind bitfield and text handling
  transcribed from the game binary's own registry class; `research/claims/registry.md` for the
  standing-corrections index; `research/formats/res/format.md` and `research/claims/res.md`
  (`RES-NODE-016`, `RES-HDR-017`, `RES-NODE-019`) for the shared `&YA1` container and its two flavors.
  Claims are cited, never the experiments behind them: a claim carries its own amendment and
  retraction state — several of the `REG-*` rows above are marked AMENDED or withdrawn-in-part — and
  an experiment folder cannot tell a later reader it has been superseded.
- **Own-data grounding:** the spec's layout was additionally re-derived directly from the owner's lawful
  install by a throwaway probe run outside the repository, over **all 44 `.reg` entries in all 12
  archives** (4621 nodes). No extracted bytes entered the repo or any commit; the probe and its output
  live in a scratch directory outside the working tree. This was necessary, not decorative — see the
  reconciliation below.

## Claim inventory

| Claim | Statement | Confidence | Bearing on the spec |
|---|---|---|---|
| `RES-MAGIC-001` | Magic `26 59 41 31` (`"&YA1"`), LE u32 `0x31415926` | High | The spec's signature check |
| `RES-SCOPE-015` | `&YA1` labels **two** structurally distinct formats: the tail-registry archive (`u32@0x10` = registry byte offset) and the inline record store (`u32@0x10` = record count). Every *nested* `&YA1` — 48 of them, including all 44 `.reg` entries — is the inline flavor | High | Why a `.reg` blob must not be handed to the `.res` reader; the spec's `nodeCount @0x10` |
| `RES-NODE-016` | The 24-byte `&YA1` header **is** the tree's root directory node: header and node share their first 16 bytes `[+0 reserved][+4 off][+8 size][+0xc type]`, and `@0x10`/`@0x14` occupy the node's name slot. From `rom.exe` | High | The spec's "the header is the root's record; the root has no on-disk node and no name" |
| `RES-HDR-017` | Header `@0x04` = the root node's `off` = index of the first top-level node; it is **dereferenced**, not a count or checksum | High | Why the spec says `rootFirst` MUST be dereferenced and never assumed zero (it is 0 in all 44 shipped registries — a value law, not a guarantee) |
| `RES-HDR-018` | Header `@0x0C` = the root node's type/flags: bit 0 = directory, **bit 4 = children-sorted**, which the reader branches on to pick binary vs linear search. `17` in every inline store (48/48) | High | The spec's `rootFlags` row, and R-4's evidence |
| `RES-NODE-019` | Node `@0x00` is the record's reserved word; no lookup, descent or sort instruction dereferences it | High | The spec's `reserved` node row |
| `REG-LOC-016` | The five class registries live as `.reg` file nodes inside `graphics.res`; their key names are literal ASCII — the game's own labels | High | Background only; the spec assigns no key meanings |
| `REG-LOC-038` | The install holds **44** registries in **12** containers, not the 5 in `graphics.res` | High | AC-8's corpus figures; confirms our own census independently |
| `REG-FMT-031` | File framing from the binary's loader: `0x18` header, `R × 32` records at `0x18`, a `u32` pool length, then the pool. Exact structural closure on 44/44, **0** invariant violations | High | The spec's stream layout and its `heapOrigin` arithmetic — **confirmed, not changed** |
| `REG-REC-032` | Record layout: `value @+0x04`, `size @+0x08`, `kind @+0x0C`, `name char[16] @+0x10`, stride 32; `+0x00` explicitly zeroed by the writer and read by no accessor | High | The spec's node table — **confirmed, not changed**; closes R-3 |
| `REG-KIND-033` | The kind word is a **bitfield**: type = `kind & 0x0E`, bit 0 = subkey, bit 4 = children sorted, bit 28 = name truncated at 15 characters | High | The spec's kind section, rewritten as a bitfield rather than an enum |
| `REG-KIND-034` | Full kind enumeration and each one's storage; the string converter carries one further case (kind 8/9) with no producer in the binary and no instance in the corpus | High for kinds 0/1/2/4/6/10 and the flag bits; **Unknown** for kind 8/9 | The spec's kinds table; kind 10 documented but not implemented; kind 8/9 stays an unrecognised-kind rejection |
| `REG-DBL-035` | Kind 4 is a little-endian IEEE-754 double in `value`+`size`, no pool access; 122 instances, `0.0` ×61 and `1.0` ×61 | High | The spec's kind-4 paragraph; closes R-2's residual |
| `REG-TEXT-036` | The game applies **no** byte-to-character conversion to registry text on either path; the name lookup folds ASCII `A`–`Z` only | High | The spec's text handling, FR-1's dependency, and R-4 |
| `REG-TEXT-037` | Corpus fact with its parameters: **0** bytes `>= 0x80` in 4621 name fields and 873 string values across all 44 registries | High (this install, this sweep) | Confirms our own census; AC-6 stays synthetic regardless |
| `REG-FMT-017`, `REG-VAL-024`…`029` | The previously promoted `.reg` record layout, pool arithmetic and tree walk | Amended in place; the published framing is **withdrawn** | Was this story's blocking conflict — now resolved in the spec's favour, see below |
| `REG-VAL-030` | `units.reg` `Width`/`Height` figures and the sprite-bounds comparison built on them | **Figures withdrawn** — `128×128`, Dragon `160×160` on re-measurement | Nothing in the spec ever rested on them; our own `CenterX = 64 = Width/2` observation agrees with `128` |

## Statements removed for non-own-data provenance

**The rule applied.** A statement whose provenance is a source outside our own work is **removed, not
annotated**. Marked-versus-unmarked confidence is not the distinction that matters; origin is. A fact of
our own that we have not yet verified may stay, labelled honestly, because it is a claim we are entitled
to make and later test. A statement sourced from outside our own work may not stay in any form — a
labelled hypothesis is still a hypothesis we are carrying, and content steers the next reader's search
whatever label rides on it.

**Four statements were removed from the baseline under this rule.** Their content is deliberately **not
reproduced here** — reproducing it would defeat the removal. What each said, and where it sat, is
recorded in this story's hand-back to the owner, which is the channel for that detail. This table is the
audit trail: the fact of removal, its location, and whether anything load-bearing went with it.

| # | Where it sat in the baseline | Kind of statement | Left a hole? |
|---|---|---|---|
| 1 | Kinds table + the `[R-2]` block | an attribution of the kind-4 layout to outside reference material | No — see below |
| 2 | Kinds table parenthetical + Out-of-scope | two enumerated value-kind constants carrying claimed meanings | No |
| 3 | FR-2 | an assertion about how the original engine's own registry lookup behaves | **Yes** — see below |
| 4 | Provenance paragraph | two named third-party projects cited as consulted sources | No |

A fifth deletion was not a provenance failure but a dangling reference: the baseline cited a
provenance-audit document, and a tier within it, that do not exist in this repository. The paragraph
holding it is gone regardless, since B2 puts provenance in this file rather than in the spec.

### What the removals cost the story

- **Removal 3 left a real hole, and it is now a research question.** The baseline's FR-2 justified its
  lookup contract by asserting how the engine's own lookup behaves. With that assertion removed, the
  spec has no basis on which to claim any engine-faithful matching rule, so FR-2 now states its matching
  as **our own API convenience** and the engine's actual behaviour is raised as **R-4** rather than
  assumed. The hole is not patched with a substitute of our own: we do not say what the engine does, we
  say what our API does and that the two are not claimed to coincide.
- **Removals 2 and 4 left no hole.** Neither of the two enumerated kinds occurs anywhere in the shipped
  registries — a fact established from the owner's install, independent of the removed statement — so
  nothing in the spec needed them. The spec now says only that an unrecognised kind is a rejection, and
  names no meaning for any.
- **Removal 1 left no hole, because the underlying fact turned out to be ours.** What was removed is the
  *attribution*; the kind-4 layout itself is now carried on our own observation of the owner's install
  (below), not on the outside source that once suggested it. Recorded plainly because it is a judgment
  call: had the install contained no kind-4 node, nothing about kind 4 would appear in the spec at all.

### Two baseline assertions that were ours, unverified, and false

Not provenance failures — the baseline was entitled to make them — but both are refuted by the owner's
install, and both are corrected rather than removed:

- **"No node of kind 4 occurs anywhere in the 44-registry ROM1 corpus."** There are **122** of them.
- **"kind 4 … the exact bit layout is unproven for ROM1."** It is proven, from own data.

## Decoded-vs-open reconciliation

### The record framing (spec R-3) — resolved from the binary; our framing is the right one

**The answer.** A node's value is the `u32` at `0x1C + 32i` — eight bytes ahead of its `kind` word and
four ahead of its `size`, not the word after its name. In absolute file terms record `i` occupies `[0x18 + 32i, 0x18 + 32(i+1))` as
`[+0x00 reserved][+0x04 value][+0x08 size][+0x0C kind][+0x10 name×16]`, stride 32, the record array
begins at file offset `0x18`, and the pool is length-prefixed: `poolStart = 0x18 + R·32 + 4`. The same
word serves an integer key and a string key's pool offset, which also refutes the possibility that
different kinds read from different displacements.

This is transcribed from the game's own registry class, not fitted, and it is attested **four
independent ways** — the loader (six 4-byte header reads, then the record array read verbatim), the
lookup (`records + node[+0x04] × 32`, iterating `node[+0x08]`, comparing `node[+0x10]`), the value
accessors (`node+0x04` for an int, `poolBase + node+0x04` with length `node+0x08` for a string), and the
**node-insert writer**, which places a new child at `records[parent[+0x04] + parent[+0x08]]`, increments
the parent's count, writes the name at `node+0x10` clamped to 15 characters — so the name field is 16
bytes and cannot be 20 — and stores `0` into `node+0x00`, making that word a real field rather than name
padding.

**The spec's layout table is confirmed unchanged.** Every offset it published is the offset the binary
uses. What changes in the spec is not the layout but the surrounding truth: R-3 is closed and retired,
the kind word is documented as a bitfield rather than an enum, and the name field's limit of 15
significant characters is now a decoded fact rather than a defensive guess.

**The A/B is not close.** Re-parsing all 44 shipped registries under strict invariants — subkey blocks
in range and covering the record array exactly once, strings NUL-terminated inside the pool, array sizes
multiples of 4 or 8, pool items tiling with no gap or overlap, exact structural closure — gives **0
violations** under the corrected framing and **82** under the previously promoted one, which breaks on
every one of the 44 including all five class registries its own claims were built on.

**All five of our corroborating regularities were re-measured under both framings and are now
explained, not merely consistent.** The `heapSize` word is the loader's pool-length read; the
`UnitCount`/`FileCount`/`AttackPhases`/`MovePhases` and selection-box anomalies are the off-by-one key
disappearing; the exact `0.0`/`1.0` doubles are the reader's 8-byte load from `node+0x04`. Under the old
framing the same kind-4 records read `1.0000000000002873` — "about 1.0", which is the shift showing
through and the reason a corpus fit alone could not have closed this. One qualification the research
adds and we did not make: `CenterX = 64` is the midpoint of `SelectionX1/X2`, but `CenterY = 78` is
**not** the midpoint of `SelectionY1/Y2` (that would be 69). The X coincidence is real; the Y one does
not hold, and neither was generalised beyond `Unit0`.

The three published carve-outs we identified as symptoms all dissolve: the "snapshot slot" scalar, the
`Files`-indexed-by-`ID` rule, and the "placeholder `128`" phase count. Each existed only to excuse an
anomaly the shift created. One published figure is **withdrawn** rather than amended: `units.reg`
`Width`/`Height` re-measure at `128×128` (Dragon `160×160`), not `128×64`, and the sprite-bounds
comparison built on the old figures is now an open question rather than a finding. Nothing in this
story's spec ever carried those numbers; our own `CenterX = 64 = Width/2` observation is consistent
with `128`.

#### This was an internal contradiction, not a routine amendment

The framing this story had to fight was **published by our own research lane**, and that same lane's
**`RES` ledger already held the correct record** — `RES-NODE-016` gives the node as
`[+0 reserved][+4 off][+8 size][+0xc type]`, `RES-HDR-017` quotes the very lookup instruction that
dereferences `[node+4]`, and `RES-NODE-019` records that `[node+0]` is never dereferenced. Those have
stood since the `.res` container was opened. A `.reg` record is exactly that node plus a 16-byte name — one class, one lookup —
so the two ledgers contradicted each other for the whole interval, and **the fix was already in the
repository, unpropagated.**

That is worth naming as a failure mode in its own right, distinct from "an old claim turned out wrong":

- **Nothing was missing.** No new instrument was needed to see it. The correcting evidence was sitting
  in a sibling ledger of the same repository, at High confidence, cited by name.
- **The two ledgers were never read side by side.** `RES` was consumed by the archive story and `REG` by
  this one; each was internally consistent, and the contradiction only exists in the join. This story's
  claim inventory is the first place the two records met, and the conflict surfaced the moment they did.
- **A wrong model that satisfies the invariants survives review indefinitely.** Both framings tile the
  stream exactly and both compute the same pool start — `0x20 + R·32 − 4` and `0x18 + R·32 + 4` are the
  same number. Nothing structural complained. This is the same shape as the `.alm` framing that was
  split 8 bytes early, and it is now the second instance.

The practical consequence for us is the one the readiness gate exists to prevent. Our own re-derivation
put the spec on the correct framing; had the gate instead deferred to the promoted spec on seniority —
"the research lane published it, so it wins" — this story would have been re-specified onto a parser
that returns every key's neighbour's value, with a corpus that parses cleanly and a test suite that goes
green. Corpus evidence raised the question; only the disassembly could answer it; and the right move at
the gate was to hold the story rather than pick a side.

#### The evidence that raised it

Our own re-derivation over all 44 shipped registries, recorded here as it stood when it was the only
discriminator available. Five independent lines, none of them tiling arithmetic:

1. **The `heapSize` word.** Under this spec the u32 at `0x18 + 32·nodeCount` equals the sum of `size`
   over every kind-0/kind-6 node in **44/44** registries, and the heap then ends exactly at
   end-of-stream in **44/44**. Under the research framing that same u32 is trailing NUL padding of the
   last record's 20-byte name, with no account of why it holds `5967`, `4161`, `3060`, … exactly.
2. **Internal consistency of `units.reg`.** Under this spec `[Unit0] AttackPhases = 7` and both
   `AttackAnimTime` and `AttackAnimFrame` hold exactly **7** elements; `MovePhases = 8` and both move
   tracks hold exactly **8**. Under the research framing `AttackPhases` reads `128` and `MovePhases`
   reads `2`, against the same 7- and 8-element arrays.
3. **Declared counts.** `[Global] UnitCount = 34` with exactly 34 `Unit*` sections, and
   `FileCount = 33` with exactly 33 children under `Files`. The research framing yields `33` and **`0`**
   for the same two keys.
4. **Geometry.** `CenterX = 64 = Width/2`, and `SelectionX1 = 48`, `SelectionX2 = 80` — symmetric about
   the centre. The research framing yields `SelectionX1 = 80 > SelectionX2 = 48`, an inverted box.
5. **Kind 4.** Under this spec the `data`/`size` pair is eight adjacent bytes forming an exact `0.0` or
   `1.0`, in cutscene sections that read `Fading1{startfade 0 → endfade 1}` then
   `Fading2{startfade 1 → endfade 0}` under a `Common{nFadings = 2}`. Under the research framing a
   kind-4 node's "byte length" field reads `0x3FF00000`.

The three carve-outs we named as symptoms — the "snapshot slot" scalar (`REG-VAL-026`), the
`Files`-indexed-by-`ID` rule (`REG-VAL-029`) and the "placeholder `128`" phase count (`REG-VAL-030`) —
all dissolve under the resolved framing, as anticipated: each was a symptom of reading one slot over.
`File` runs `0,1,2,…` and *is* the index into the `Files` table, while `ID` is a sparse class id.

**The verdict came from the instrument, as it had to.** Our evidence was corpus evidence and could only
raise the question; the closing instrument was static analysis of the game's own registry class, and it
went further than a reader alone could — finding the *writer* as well, which is what fixes the name
field at 16 bytes and `+0x00` as a real field rather than padding. Corpus statistics cannot distinguish
a field that is always zero from padding that is always zero; a 15-character clamp in the writer can.

**Cross-story exposure — corrected.** `docs/0001-res-archive/spec.md` stated, in its "Two `&YA1`
flavors" note, that in the inline flavor "records start at `0x20`". That was the withdrawn framing; the
correct origin is `0x18`, and that sentence has been fixed. Its own registry-node table
(`0x04` value, `0x08` size, `0x0C` type, `0x10` name×16) needed no change at all — it is the
`RES-NODE-016` record that the `.reg` ledger had been contradicting, sitting one page above the wrong sentence.
Nothing in the shipped `.res` reader depends on either; the flavors note is a descriptive aside about a
format that reader deliberately refuses to parse.

### R-1 — the code page: resolved, and the answer is that there is no code page

**The answer is "no conversion at all."** The game performs **no** byte-to-character conversion on
registry text, on either path. The loader bulk-reads the record array and the pool verbatim; the string
accessor is a `strncpy` out of the pool; and the INI→REG compiler — the one place a code page would have
to be applied, since that is where plain text becomes stored bytes — writes text into the pool as
`strlen(s)+1` raw bytes with no transform. A reachability sweep of the whole image puts every
code-page API (`MultiByteToWideChar`, `WideCharToMultiByte`, `LCMapStringA`, `GetACP`, `GetOEMCP`)
inside the statically-linked MFC/CRT range and none of them inside the registry class; the two
Ansi↔Oem thunks that do exist have **zero callers**; and `setlocale` / `_setmbcp` / `IsDBCSLeadByte`
appear nowhere at all. The only byte transform anywhere on the path is the *name lookup*'s
case-insensitive compare, whose fast path folds `A`–`Z` only and leaves bytes `≥ 0x80` untouched.

**Confidence: High**, and the reason a negative earns it here is that it was established the only way a
negative can be — both paths disassembled (reader *and* compiler), plus a whole-image reachability sweep
of every code-page entry point, rather than an absence of evidence in one function.

**This is a stronger answer than either candidate we dispatched, and it changes the question.** We asked
"CP866 or CP1251?" and the format's answer is "neither — the format stores bytes." There is nothing to
transcribe, because there is nothing the original does. That has three consequences the spec must carry
in its own terms:

1. **Any character mapping we apply is a presentation choice of ours, not a property of the format.**
   Quietly decoding as CP866 and moving on would have been a fabricated fidelity claim: the parser would
   silently attribute to the game a mapping the game does not perform. CP866 and CP1251 are both
   defensible *renderings*; neither is the game's.
2. **The parser therefore performs no decode.** Names and string values are byte strings, exposed
   unchanged, and the display convention belongs to whatever displays them — which for this story is the
   dump tool, and it must state its convention rather than assume one.
3. **`pkg/formats/reg` needs no text-encoding dependency.** The package's own tier note and the
   repository's tier table both anticipate `golang.org/x/text` for CP866 in this package; that
   anticipation is now unearned *for this package*. The dependency itself stays correct and stays in the
   module for `pkg/formats/res` (CP866 entry names) and `pkg/formats/alm` (CP1251 descriptions), both of
   which genuinely convert. Only the `reg` half of the tier note is affected, and that note lives
   outside this story's files.

**The corpus premise held, and is now independently confirmed.** The research's own sweep — every
regular file under the install root, nested payloads classified by root kind rather than by extension —
finds **0** bytes `≥ 0x80` in 4621 name fields and 873 string values, byte range `0x20…0x7A`. That
matches our own census exactly. So no shipped registry datum exercises a code page at all, which is both
why the corpus could never have answered this and why the question was worth little effort. AC-6's
high-byte fixtures stay **synthetic** on those grounds, and now test byte fidelity rather than a decode.

### R-4 — the lookup's case sensitivity: not dispatched, but settled anyway

R-4 was raised by this story and deliberately **not** sent out: non-blocking, nothing observable changes
under either answer, and every shipped key is uniquely named under both rules. It is settled regardless,
as a by-product of transcribing the lookup for R-3 — so it is recorded here as resolved rather than left
open on a technicality.

**The engine's registry lookup is case-insensitive.** It compares a key against a child's name field
with a case-insensitive compare over **15** characters, whose fold covers `A`–`Z` only and leaves bytes
`≥ 0x80` alone; when the parent's kind carries the sorted-children bit, it binary-searches the child
block with that same comparator, otherwise it scans linearly. **Confidence: High** — same instrument,
same listing, same four-way attestation as the framing.

Two things follow, one of them a correction to our own reasoning:

- **Our FR-2 matching rule coincides with the engine's** on the point it was worried about. It remains
  written as our own API convenience — that is what it is, and it does not become an engine-fidelity
  claim by turning out to agree — but the spec no longer needs to hold the disagreement open.
- **The inference that raised R-4 was wrong in its conclusion.** We argued from the root's sorted
  child list that an ordered search implies a byte-wise, hence case-sensitive, comparison. The ordering
  is real and the search is genuinely binary; the comparator is simply case-insensitive anyway. Sorted
  order constrains the *search*, not the *comparator*, and we read one into the other.

One residual difference, which cannot arise in a game-written registry: the engine compares only the
first **15** characters, so two names agreeing there are indistinguishable to it, while our accessors
compare whole names. The game's own writer clamps a name to 15 significant characters plus a NUL and
flags the truncation in the kind word, and no shipped name is longer than 15 — so the two rules cannot
diverge on any registry the game itself produced.

### The kind word — a bitfield, and one kind we now know but do not implement

The kind word is **not a plain enum**: the value type is `kind & 0x0E`, bit 0 marks a subkey, bit 4
marks the child list sorted, and bit 28 marks a name that was longer than 15 characters and got
truncated. The root's `17` is therefore `subkey | sorted`, not a magic constant — which is why every
shipped registry carries `17` in the header's kind slot. Our spec already treated the header's word this
way; what changes is that a *node's* kind must be read the same way rather than matched against a list
of whole values.

The enumeration is High for kinds 0/1/2/4/6/**10** and for the flag bits, each attested by an explicit
type test *and* an explicit writer. Kind 10 is a double array — pool offset in `value`, byte length in
`size`, `size/8` elements — and it is the one genuinely new type this fold-in brings. **It occurs zero
times in the install.** The spec documents it (it is decoded, and the spec doubles as our format
documentation) but this story does not implement it: no accessor, no acceptance criterion, and a
kind-10 node is rejected as unsupported. That is a scope choice, recorded as one, and it is honest in
both directions — we do not claim ignorance of a type we have decoded, and we do not ship an untested
decoder for a type no shipped file exercises.

**Kinds 8 and 9 stay Unknown, and no requirement rests on them.** The string converter carries one
further case for them, but nothing in the binary produces such a node and nothing in the corpus is one,
so the research deliberately claims no semantics. The spec's existing stance already covers them: an
unrecognised kind is a rejection that names the kind and the node index, never a guess. Two other words
also stay Unknown and are likewise load-bearing on nothing — the header's last word, and the record's
`+0x00`, now known to be a real field the writer explicitly zeroes rather than name padding, read by no
accessor.

### R-2 — resolved by observation, identifier retired

The baseline asserted that no kind-4 node occurs anywhere in the corpus. **122 of them occur**, across
31 of the 33 cutscene registries in `VIDEO4.RES` and `VIDEO8.RES`, under exactly two key names,
`startfade` and `endfade`. Decoded as a little-endian binary64 from the `data`/`size` pair they take
exactly two values, `0.0` (61 nodes) and `1.0` (61 nodes), and they sit in a structure that reads as a
cutscene fade script: a `Common` section carrying `nFadings = 2`, then `Fading1` and `Fading2` sections
each carrying `startframe` / `endframe` / `startfade` / `endfade`, with the first fading in (`0 → 1`)
and the second fading out (`1 → 0`). A wrong field mapping does not produce that.

**One limit of that evidence, stated rather than glossed.** Both shipped values — `0.0` and `1.0` —
have a zero low word, so all 122 nodes carry `data = 0` and vary only in `size` (`0x00000000` /
`0x3FF00000`). The corpus therefore proves that the *high* half of the double lives in `size`, and
leaves the placement of the low half resting on the fact that the eight bytes at node `0x04` are
contiguous and are the only eight available — the reading under which both values come out exact. No
shipped node exercises a non-zero mantissa low word. This does not weaken the "kind 4 is a float64"
conclusion, which the two exact values and the fade semantics carry on their own; it bounds what the
shipped data can witness about byte order within the pair, and `AC-2` consequently tests the full
64-bit pattern synthetically rather than claiming corpus support for it.

The kind-4 layout the spec carries therefore stands on **our own observation of the owner's install** —
122 nodes, two key names, two exact values, in a structure whose own declared count agrees with it. It
is not inherited, and it is not annotated: it is a decoded fact, and the spec states it as one. Nothing
of the removed attribution's content survives with it. Its residual — instruction-level confirmation —
is the same question as R-3, since a kind-4 value *is* the `data`+`size` pair whose position the two
framings dispute. `R-2` is retired rather than reused (stable-ID discipline).

**That residual is now closed, and the caveat above should be read for what it always was.** The
disassembly settles the byte order directly: the reader loads *eight bytes* from `node + 0x04` as one
`double` in a single x87 instruction, and the compiler writes a `double` back into those same eight
bytes — so the low half's placement is the machine's little-endian qword layout, not an inference from
"the only eight available". Every other candidate 8-byte window was evaluated over all 122 records and
yields denormal garbage. The corpus caveat stays true **as a statement about the corpus** and must
survive: 122/122 shipped nodes still carry `data = 0` and still witness only the high half, and `AC-2`
must still be described as exercising the full 64-bit pattern **synthetically**, never as corpus-backed.
What changed is that the question the caveat bounded is no longer open — it was closed by the
instrument, exactly where the caveat said it would have to be.

**Recorded as a judgment call, for the owner to ratify.** The instruction under which this polish was
written said to delete the kind-4 layout outright and have the spec say only that kind 4 is undecoded.
That instruction was issued on the premise that no kind-4 node exists in the install; the install
refutes the premise. Deleting the layout would have required `spec.md` to assert that kind 4 is
undecoded while we hold 122 nodes that decode coherently — overstating our ignorance, which the
evidence-honesty rule forbids as squarely as overstating our knowledge. The layout is kept on own-data
grounds. If the owner still wants it out, the change is the kinds-table row, the paragraph beneath it,
AC-2, and the `GetFloat` accessor in FR-2.

**RATIFIED by the orchestrator, 2026-07-25.** The judgment stands and the executor was right to refuse
the instruction. The owner's rule turns on **provenance, not on confidence**: a statement sourced from
outside our own work stays in no form, while a fact of ours may stay, honestly labelled. Once the
install's own bytes carry the layout, its provenance *is* ours — the earlier attribution described how
the sentence first reached the baseline, not what now supports it. Refusing the instruction was the
correct move rather than an exception to it: the instruction was derived from a premise the executor
then disproved, and executing it anyway would have written a false statement into `spec.md`.

Two things this leaves on the record. The honesty bound the executor added is the substantive part and
must survive any later edit: all 122 shipped nodes carry `data = 0` and vary only in `size`, because
both values happen to have a zero low word — so the corpus witnesses the double's **high** half in
`size`, and the low half's placement rests on the eight bytes at node `0x04` being contiguous and the
only eight available. "Kind 4 is a float64" is carried by the two exact values and the fade semantics;
AC-2 exercises the full 64-bit pattern **synthetically**, and must not be described as corpus-backed.
And the general lesson, which research's own `AGENTS.md` already carries: an
instruction inherits the truth of its premise, so a lane agent that disproves the premise is obliged to
stop and say so rather than comply. The owner may reverse this ratification; the revert recipe above
stays accurate.

### Corpus census (own data, for the record)

44 `.reg` entries across 12 archives — 33 cutscene registries (18 in `VIDEO4.RES`, 15 in `VIDEO8.RES`),
5 in `graphics.res` (`units`, `material`, `objects`, `structures`, `projectiles`), 3 in `scenario.res`
(`globalmap`, `npc`, `scenario`), 2 in `world.res` (`data/ai`, `data/map`), 1 in `sfx.res`. No loose
`.reg` file exists on disk outside an archive. 4621 nodes total: kind 0 = 873, kind 1 = 512, kind 2 =
2792, kind 4 = 122, kind 6 = 322 — no other kind value appears. Header `0x0C` is `17` in 44/44 and
`0x14` is `0` in 44/44; `rootFirst` is `0` in 44/44. Every registry nests **exactly two** levels; there
are **zero** orphan nodes and **zero** nodes referenced by more than one directory across the whole
corpus, so the spec's orphan-tolerance and single-parent rules are unexercised defensive choices, as it
says. No kind-6 node has a `size` that is not a multiple of 4. The root's child list is byte-ascending
by name in 44/44 — matching the sorted-children flag the header carries — while nested directories
carry no such flag and are unordered in 431 of 512 cases; that asymmetry is what raised R-4.

Read as a bitfield, the same census says a little more. All 512 subkey nodes carry kind exactly `1`, so
**no nested subkey sets the sorted bit** and the binary search the engine can take is exercised only at
the root. **No node sets bit 28**, consistent with every shipped name being 15 characters or fewer.
Kind 10 (double array) occurs **zero** times, as do kinds 8 and 9. The research's independent sweep
agrees with this census throughout — same 44 registries, same 4621 records, same per-kind counts, same
zero high bytes — which is worth stating because the two counts were produced by different probes from
different lanes against the same install.

One numeric discrepancy inside the pinned research, checked rather than assumed: the experiment's own
container table lists `VIDEO4.RES` as holding **17** registries (making 43, against the 44 the same page
states), and the standing-corrections index refers to "the install's **32** video registries". The same
research's claim ledger and promoted spec both say **18** and **15**, i.e. **33** video registries and
44 in total, with 31 of the 33 carrying kind 4 — and that is what our own probe counted. So the two
outliers are clerical slips in prose, not a conflict of substance, and nothing this story carries
depends on them; `AC-8`'s per-archive figures already read 33 / 18 / 15.
