# Analysis — what a font is, and what this tree already has

**Intensity: spec-anchored / static.** Two of the three things this story fixes are contracts that
outlive it — a byte-level layout for the font's second node, and a drawing API the unit-information
panel will be written against. No watcher tool exists, so the sync is discipline.

**Terrain:** greenfield. `pkg/render/text` and `cmd/texttool` are new; `pkg/formats/spr16` gains a
function beside its two decoders and changes neither, so the shipped parser behaviour this story
inherits is not modified and its own tests are not edited.

## What we did not know

The tree can decode a `.16` glyph atlas — story `0021` shipped that — and cannot draw a word. The
gap is not the pixels. It is everything around them: which of the shipped atlases to use, how wide a
character is, what a byte means, and what colour a 4-bit value is.

## What the research says, and what it cost to find out

Five font atlases ship (`SPR16A-FONT-015`) in two container forms, and `rom.exe` loads four; the
`.16a` pair (`font4`, `font5`) are ordinary palette-bearing sheets, and `font5` is named by nothing
in either executable. The three `.16` atlases are the byte-control form `0021` decodes.

The finding that decides the shape of this story is `SPR16A-FONT-018`: **a font is two nodes.** Each
font object loads `<base>.16` *and* `<base>.dat`, and the sidecar — one `u32` per record — is the
advance. The cell is never the advance: `dat[g]` is strictly less than the record width on all 960
glyphs of all five atlases. A consumer that spaces glyphs by the cell draws at the wrong pitch, and
that is a picture that looks plausible until it is compared with the game.

The second is `SPR16A-FONT-013`, at High because the blitter itself was read: the 4-bit value is
**not** a palette index and not a level into the `.16a` LUT. It indexes a 16-entry ramp the *caller*
supplies, built as `base * k / 15`. So a `.16` glyph is a coverage mask over a colour the caller
chooses, and the write is opaque — no destination read, no blend. Two consequences a decoder must
carry: value 0 is a **written** pixel (the low nibble has no zero test), and no shipped glyph
exercises it, so getting it wrong is unobservable in data.

The arrangement (`SPR16A-FONT-020`) is `char = record + 32`, attested twice — by the owner-review
render of records 32..62 and by `DrawText`'s own `SUB AL,0x20`. Past ASCII it is a hybrid of CP437's
accented Latin and the whole Russian alphabet, and it is **none of CP866/CP1251/KOI8-R**. Which
matters less than it looks here: `RES-TEXT-022` and `REG-TEXT-037` sweep every shipped archive name
and every shipped registry string and find **0 bytes >= 0x80** in 45 865 name bytes and in 873
string values. Every caption we can source from the shipped data is ASCII, so the high half is
carried, not interpreted.

Three findings are load-bearing as *warnings* rather than as inputs. `SPR16A-FONT-014` and
`SPR16A-FONT-021`: `font1`/`font2` continue past their count trailer into sections that are older
in-place build layers, unreachable by the engine, and one of them holds eleven glyphs with an open
owner question. A decoder stops at the count — which `0021`'s container walk already does.
`SPR16A-FONT-019`: the byte blitter aliases its unused quadrant to *skip*, the opposite of the
sibling `.16a` blitter — also already in `0021`.

`SPR16A-TXT-023` is the one measured mismatch: the Russian release's own data strings are CP866 and
its fonts are arranged otherwise, and how that release shows Russian text is open. Nothing here
depends on the answer.

## What the install says

`graphics.res`, listed through `cmd/restool`, carries the ten nodes the claims predict and the two
sizes that pin the sidecar's shape:

```
   32932  font1/font1.16        896  font1/font1.dat
   13892  font2/font2.16        896  font2/font2.dat
    1073  font3/font3.16        256  font3/font3.dat
   41748  font4/font4.16a       896  font4/font4.dat
   76674  font5/font5.16a       896  font5/font5.dat
```

896 = 224 x 4 and 256 = 64 x 4: one `u32` per record, on every atlas, with no header and no trailer
to account for.

## What this tree already has

`pkg/formats/spr16` decodes both grammars and walks the shared container, taking the count from the
trailer and touching nothing past the last counted record. Its `PixelG` keeps the raw 4-bit value
and a separate `Painted` flag, so the "0 is a written pixel" rule is already representable rather
than something this story has to retrofit.

`pkg/render/terrain` and `pkg/render/menu` both import nothing inside the module: art reaches them
as plain data a loader in `pkg/game` fills. That is the seam a font travels on, and it is why this
story adds no edge to the dependency graph.

## What we deliberately did not look at

Which of the thirteen ramps a given UI string gets, and the per-display-mode framebuffer packing,
are both open in the research and both about the *absolute* colour of drawn text. This story takes
the colour as an argument, so neither is on its path.
