# Provenance — 0050-text

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1, the sidecar is one `u32` per record, in record order | `SPR16A-FONT-018` | High |
| FR-1, the sidecar's length must equal the atlas's record count | `SPR16A-FONT-018` (896 B = 224 and 256 B = 64, on all five atlases) | High |
| FR-2, a glyph record is a fixed cell and the metrics live outside the sprite | `SPR16A-FONT-018` | High |
| FR-2, a glyph pixel is a 4-bit value and 0 is a written pixel | `SPR16A-FONT-013` | High |
| FR-3, byte `c` selects record `c - 32` | `SPR16A-FONT-015`, `SPR16A-FONT-018` (`SUB AL,0x20`, plus the render of records 32..62) | High |
| FR-3, record 0 is the space and is empty in all five atlases | `SPR16A-FONT-015`, `SPR16A-FONT-020` | High |
| FR-3, records 96..223 are a hybrid of CP437 accents and the Russian alphabet, none of CP866/CP1251/KOI8-R | `SPR16A-FONT-020` | Medium |
| FR-3, no shipped `.res` name or `.reg` string byte is `>= 0x80` | `RES-TEXT-022`, `REG-TEXT-037` | High (this install, this sweep) |
| FR-4, the pen advances `dat[g] + spacing`, and record 0 additionally adds `height(0)/2` | `SPR16A-FONT-018` | High |
| FR-4, spacing is 2, from the construction site | `SPR16A-FONT-018` | High |
| FR-6, the value indexes a 16-entry ramp the caller supplies, `base * k / 15` | `SPR16A-FONT-013` | High |
| FR-6, the write is opaque — no destination read, no blend | `SPR16A-FONT-013` | High |
| FR-7, the two nodes are `<base>.16` and `<base>.dat` under `<base>/` | `SPR16A-FONT-018`; the node names confirmed against the install's own listing (analysis.md) | High |
| FR-7, `font1`/`font2`/`font3` are the byte-control atlases, 224/224/64 records, cells 16x15 / 8x10 / 8x6 | `SPR16A-FONT-015` | High |
| FR-7, `font1` is byte-identical between the English and Russian releases; `font2` is the one node the Russian build replaces, with 35 records blanked | `SPR16A-FONT-022` | High |
| FR-7, the atlas stops at the count trailer; the appended sections are unreachable | `SPR16A-FONT-014`, `SPR16A-FONT-021` | High (unreachability) / Medium (why they exist) |

## Ours by choice

| Decision | Why it is ours |
|---|---|
| **`font1` is the default** (FR-7). Three arguments, in order: only a `.16` atlas makes the pixel value an intensity into a colour the caller chooses, so `font4`/`font5` cannot satisfy FR-6 without a colour rule nobody has; `font3` holds 64 records of which most are empty, so it cannot spell a word; and of the two remaining, `font1` is byte-identical across both shipped releases while `font2` is the single node the Russian build replaces. It is a **default and not a lock** — the loader takes the base name. | No source says which font a panel should use. `SPR16A-FONT-022` supplies the release-identity fact; the ranking of the three arguments is engineering. |
| **A byte with no record renders as record 0** (FR-3) — the space's advance, nothing painted. | The engine subtracts `0x20` and indexes with no bound (`SPR16A-RDR-017`: the reader validates nothing), so there is no original behaviour to copy — only one to refuse. Substituting the space keeps the string's visible length, so a missing glyph shows as a gap instead of silently closing up. |
| **Measurement reports a box that contains everything drawing paints** (FR-5), never less than the pen advance, and **the pen is also reported on its own**. | The engine measures nothing; `DrawText` only advances a pen. The box is what a panel needs for a frame, the pen is what it needs to place the next run, and both come off one placement — which is what makes "measurement and drawing cannot disagree" testable rather than a convention. |
| **An atlas holding no record is refused at load** (FR-7). | The engine's own reader validates nothing at all (`SPR16A-RDR-017`), so there is no original behaviour to copy. A zero-record atlas is well-formed to the container and would yield a font with no record 0 for the fallback to reach. |
| **The census measures the level histogram, record 0's blankness and how many records paint past their own advance** (FR-8). | `SPR16A-FONT-013` establishes all three for the corpus it read. Re-measuring rather than inheriting is ours: the level-0 clause in particular is unobservable in shipped data, so a consumer that gets it wrong finds out from a picture, not from a test. |
| **The caller's alpha is written through unscaled; only R, G and B take the level** (FR-6). | The ramp is `base * k / 15` packed to a 16-bit framebuffer with no alpha channel at all, so alpha has no original meaning. Writing it through keeps a premultiplied colour valid. |
| **The caps**: a sidecar entry over the container's own dimension cap is an error (FR-1). | `SPR16A-BOUND-016` measures what shipped and says the format bounds nothing below its field types; the cap is a decision, taken to match the cap `0021` already applies to a frame dimension. |
| Package placement: the font model in the drawing tier as plain data, the two decodes in the formats tier, the join in `pkg/game` (FR-2, FR-7). | The dependency DAG. Nothing in the research speaks to it. |

## Open

| Left undecided | Why |
|---|---|
| What the *absolute* colour of the game's text is. | Which of the thirteen ramps a given string gets is traced to the blit argument and not through the UI code that chooses it, and the per-display-mode framebuffer packing is a runtime residual (`formats/spr16a/format.md`, Open). The colour is therefore an argument, and this story asserts nothing about which value the game would pass. |
| What the eleven glyphs in `font1`'s first appended section are for. | `SPR16A-FONT-021` reads them as an older draw of `х..я` and records an open owner question. They are past the count trailer and the decoder never reaches them. |
| How the Russian release displays its CP866 strings. | `SPR16A-TXT-023`, open, and needs runtime observation. Nothing here depends on it: a byte is carried to a record and never transcoded. |
| The `^` escape in the engine's own `DrawText`. | Located, not decoded (`formats/spr16a/format.md`, Open). This story treats every byte as a character, which is the only reading available. |
| Whether the drawn text *looks* like the game's. | Only the owner can judge it. |

## Removed

| Dropped | Why |
|---|---|
| A character-set mapping from CP866 or CP1251 to the atlas arrangement. | The arrangement is none of them (`SPR16A-FONT-020`) and no shipped string this tree reads carries a byte `>= 0x80` (`RES-TEXT-022`, `REG-TEXT-037`). A conversion table would be an invention with nothing to test it against. |
| Reading `font4`/`font5` as fonts. | They decode through the `.16a` path, whose pixels resolve through the file's own palette — there is no caller-chosen colour to give them. `font5` is loaded by nothing in either executable (`SPR16A-FONT-018`). |
| Any use of the appended sections. | Unreachable by the engine (`SPR16A-FONT-014`) and carrying an open question (`SPR16A-FONT-021`). |
