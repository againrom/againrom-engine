# Story 1022 — the character generator's canonical composition

## Result

The detailed character-generator page composes on the same four column slots and one 320-wide centre
that `1021` gives the town rooms, drawn from the shipped `chrgen` art at its own dimensions. The
centre column is no longer cropped or displaced. The left column carries two shipped panes instead of
one oversized authored card. The tip panel opens on the detailed page. The authored message box is
removed.

The owner supplied a screenshot of the original's detailed page on 2026-08-21 and named it the
canonical composition. He asked for four things: the canonical layout; reuse of `1021`'s pane result
so the same parts build other screens; removal of "some black rectangle with text"; and tips on this
page.

A screenshot of the original establishes owner intent. It is not evidence of ROM1 behaviour and does
not reach research as fact (B1). Where this contract states a decoded value it cites the claim.

## The composition is determined by the shipped art

Every slot below is a shipped entry whose dimensions were measured from the BMP header on the EN
root. Reproduce:

```
go run ./cmd/restool cat <root>/graphics.res <path> | head -c 54
```

| slot | rect | source | measured |
|---|---|---|---|
| upper-left body | `(0,0)-(160,238)` | `main.res graphics/chrgen/leftup.bmp` | 160x238 |
| upper-left seam | `(160,0)-(176,238)` | `interface/chrgen/rollstatsr.bmp` | 16x238 |
| lower-left body | `(0,238)-(160,480)` | `interface/chrgen/fullstatsl.bmp` | 160x242 |
| lower-left seam | `(160,238)-(176,480)` | `interface/chrgen/fullstatsr.bmp` | 16x242 |
| centre | `(160,0)-(480,480)` | `interface/chrgen/{fighter,mag}/column.bmp` | 320x480 |
| upper-right seam | `(464,0)-(480,238)` | `interface/inn/ruover.bmp` | 16x238 |
| upper-right body | `(480,0)-(640,238)` | `interface/chrgen/buttonsarea.bmp` | 160x238 |
| lower-right seam | `(464,238)-(480,480)` | `interface/HumanBackL.bmp` (`TOWN-312`) | 16x242 |
| lower-right body | `(480,238)-(640,480)` | `interface/humanbackr.bmp` | 160x242 |

The four rects at `x:[160,176)` and `x:[464,480)` are `TOWN-234`'s four border-column blits, at its
own literal coordinates `(160,0,16,238)`, `(160,238,16,242)`, `(464,0,16,238)`, `(464,238,16,242)`,
issued through `vt+0x38` and therefore keyed, over a background the same child blits opaque across
`(160,0)-(480,480)` through `vt+0x18`. Grade High for the vtable-slot identity and the four blits'
exact literal coordinates. The centre bitmap is 320x480 and the centre child's rect is 320 wide: the
two agree, and the strips overpaint the centre's own outer 16 columns.

**`TOWN-312` (arrived at the pin bump) names the archive entry behind each of those four blits, so
none of them is a rendering score any more.** `R0133`, called with `this` set to the `+0x7c`
child rather than the top-level screen, writes all four before any class branch: `+0x68` from
`chrgen\RollStatsR.bmp`, `+0x6c` from `chrgen\FullStatsR.bmp`, `+0x70` from `inn\RUOver.bmp`, and
`+0x74` by direct copy of the global `L11832`, whose one writer loads
`interface\HumanBackL.bmp`. Grade High for the three fresh loads' exact paths and for `+0x74`'s
mechanism. **Unknown for whether `L11832` is populated before the final stage runs** — the
writer's own call chain was traced two hops, not to completion, and that writer loads roughly 30
bitmaps in a routine whose neighbouring filenames are not chargen-specific. That Unknown is the one
part of the four this story handles defensively: if the global is empty at that moment the original
draws nothing in the lower-right seam, and this build always draws.

**A field-offset collision to read carefully.** `TOWN-312`'s `+0x70` is a border-source field ON THE
`+0x7c` CHILD. `TOWN-313`'s `+0x70` is the fourth child OF THE TOP-LEVEL SCREEN. Same offset, two
different objects, and the two rows are adjacent in the ledger.

`SHOP-FIGURE-041` gives the shared lower-right panel as `ctor(id 7, 0, 238, 160, 480)`, offset by
`640 - width` when a screen borrows it, which is `(480,238)-(640,480)`, 160x242 — the same slot and
the same object every room reuses. Grade High.

**A quarter of this screen still has no established painter, and this story does not close that.**
`TOWN-313` identifies the final stage's fourth top-level child as the `+0x70` object,
`R0833`, rect `(0,0)-(160,238)`, which refutes `TOWN-283`'s own prediction that the unfetched
fourth child would paint the remainder. With that child's rect known, 230,400 of 307,200 pixels are
covered and **76,800 (25%) are not**, in two disjoint regions: `x:[0,160)` `y:[242,480)` (38,080 px)
and `x:[480,640)` `y:[238,480)` (38,720 px). Grade High for the child's identity, rect and content
and for the arithmetic; Unknown for what paints either region.

Both regions are slots the table above fills from shipped art — the lower-left body and the
lower-right body — so **this story draws them on authored grounds, not decoded ones**, and each
owes its own typed row. Do not read the table as decoded throughout: the four seams are decoded
(`TOWN-234` for the rects, `TOWN-312` for the entries) and the lower-right panel is decoded
(`SHOP-FIGURE-041`); the two bodies inside the uncovered regions are not.

The loader already agrees with the table. `pkg/game/chargenassets.go` validates `p.Columns[class]` at
exactly 320x480, validates every skill patch against `image.Rect(0,0,320,480)`, and validates
`p.Plate` at 160x238. The shipped sources were never the problem.

## Why the page does not match today

**Re-measured on master `87cf8b4`, 2026-08-21.** This list was first written against `25cd7a1`.
Story `1021` has landed since and closed three of its nine items, and the pin bump added a tenth the
list did not have. Items 6, 7 and 8 are struck through with what closed them rather than deleted,
because a lane reading this contract needs to know they were considered.

1. **The centre column is cropped and displaced.** `chargenColumnSourceCrop = (72,0)-(252,480)` and
   `chargenColumnDestination = (300,0)-(480,480)` (`pkg/ui/chargen_page.go:122-123`). 72 source
   columns are discarded on the left and 68 on the right, 140 of 320; the surviving 180 are drawn
   68 px right of where the source's own origin belongs.
2. **The character card is 300 wide.** `chargenCardBox = (0,207)-(300,480)` (`:124`), because
   `RenderCharacterPanel(AuthoredPanelLayout(), …)` composes at `sidebarWidth = 300`
   (`pkg/ui/panel.go:654`). The card therefore covers `x:[160,300)` of the centre and starts 31 rows
   above the canonical `y=238`. This constant is the cause of item 1: the centre was cropped to fit
   the card, not the other way round.
3. **The black rectangle with text.** `chargenDetailMessageBox = (160,0)-(300,207)` (`:140`), drawn
   at `:576` with `drawChargenFrame`. Its own comment calls it "the otherwise unused space between
   the plate and the cropped source column". In the canonical composition that space is the centre
   column. This is the rectangle the owner asked to remove.
4. **The plate is drawn into a 240-row box.** `copyNative(dst, p.Plate, image.Point{}, image.Rect(0,
   0, 160, 240))` (`:672`) for a source the loader validates at 160x238.
5. **The lower-left body is never drawn.** `interface/chrgen/fullstatsl.bmp` has no reference in any
   non-test Go file. The card is drawn on the page's flat fill.
6. ~~**The upper-left seam is never drawn.**~~ **Closed by `1021`.** `chargenassets.go` loads
   `interface/chrgen/rollstatsr.bmp` as `PlateSeam`, keyed at load, and `chargen_page.go` draws it
   at `chargenPlateSeamRegion = (160,0)-(176,238)`. `TOWN-312` has since named that same entry as
   the `+0x68` source, so `1021`'s rendering score is now decoded.
7. ~~**The lower-right seam is the wrong family.**~~ **Closed by `1021`.** `chargenassets.go` loads
   `interface/humanbackl.bmp` as `DollPane.Seam` and it draws through `drawTownPane` at
   `chargenLowerSeamRegion`. `TOWN-312`'s `+0x74` is a direct copy of the global its one writer
   loads from that same entry, so the family choice is decoded rather than scored.
8. ~~**The lower-right body is a flat fill.**~~ **Closed by `1021`.** `chargenassets.go` loads
   `interface/humanbackr.bmp` as `DollPane.Body` and `chargen_page.go:832` draws the pane. The
   near-black `image.Uniform` fill survives at `:834` as the load-failure fallback only.
9. **The tip panel cannot open on this page.** `Chargen.TipPanel` returns an empty view when
   `c.stage != PreCreateStage` (`pkg/ui/chargen.go:768`, re-read at `87cf8b4`). `TOWN-187` decodes a second call site,
   `R1986`, which reads `main\text\tips\chrgen2.txt` and replaces the existing popup's text
   in place on a once-only latch, never constructing a second popup. That text is not loaded
   anywhere in this tree.

10. **The lower-left seam is never drawn**, added by the pin bump and carried as `DIV-189`.
    `TOWN-312` names `chrgen\FullStatsR.bmp` as the `+0x6c` source and `TOWN-234` gives that
    field's own blit as `(160,238,16,242)`. Nothing draws at `(160,238)-(176,480)` in this tree:
    the character card, `chargenCardBox = (0,207)-(300,480)`, covers that rectangle and is the last
    thing composed over it. The doll pane composes after the card (`chargen_page.go:832`) but at
    `TownCharacterRegion = (480,238)-(640,480)`, which does not reach the left column.
    **This is item 2 again**, the same authored width producing both, which is why `1021` recorded
    it rather than adding a blit nothing would show. `DIV-189` closes when the strip draws at that
    rect from the named entry and a witness observes it through the production composer on both
    roots.

Item 2 is the root: one authored width forced the crop, the displacement, the message box and item
10. **Seven of the ten remain**: 1, 2, 3, 4, 5, 9 and 10. Items 5 and 10 are the two halves of the
left column's lower row and both close on the same re-lay.

## Behaviours

**B1 — the centre column draws whole at its decoded rect.** `chargenColumnDestination` becomes
`(160,0)-(480,480)`, the source crop is removed, and `chargenColumnOffset` becomes `(160,0)`. The
skill-icon origins in `detailedSkillOrigin` are already expressed in 320x480 source space and shift
with the offset; the mask lookup already subtracts the offset and follows. This alone resolves the
mage page's fire plaque reaching the right seam, because `detailedSkillOrigin[1][0]` moves from
screen `x=428` to `x=360`.

**B2 — the four column slots draw as shipped panes, through `1021`'s pane type.** Bodies opaque,
seams keyed, at the nine rects in the table above. The generator must call the same pane helper the
town rooms call, not a private copy: that is the reuse the owner asked for.

**B3 — the character card is composed for a 160x242 box and drawn on `fullstatsl`.**
`chargenCardBox` becomes `(0,238)-(160,480)`. The arrangement stays the owner's 2026-08-10
arrangement, which is what his screenshot shows. Required content, in his order:

```
                <name>
BODY     nn   HEALTH  nnn/nnn
AGILITY  nn   MANA    nnn/nnn
MIND     nn
SPIRIT   nn
DMG    nn-nn  ABSORB   nn
ATTACK   nn   DEFENSE  nn
SKILLS        RESISTANCE
<five skill rows>   <five element rows>
WEIGHT  nn.n
XP      nnnn
              SIGHT  nn.n
              SPEED  nn
```

Truncation is acceptable and the main information must fit (owner, 2026-08-21, restating his ruling
of the same day on the town character panel). Choose the font by measuring; do not assume one. Every
row that cannot fit whole is named in `closure.md` with what was dropped.

**B4 — the authored message box is removed.** `chargenDetailMessageBox`, its `drawChargenFrame` call
and the wrapping loop that follows it all go. Any hover or refusal copy it carried either moves into
the tip panel's text area or is dropped; state which in `closure.md`.

**B5 — the tip panel opens on the detailed page**, with the `SHOW TIPS NEXT TIME` toggle and the
`CLOSE` button the owner's screenshot shows, carrying `main/text/tips/chrgen2.txt`. `TOWN-187` gives
the popup as one object of 312x200 whose construction immediates are `left=0, top=0x118, right=0x138,
bottom=0x1e0`, and the second call site replaces its text rather than moving it. **The rect is
decoded and already shipped** (`TOWN-314`, hotfix `23bb8e6`); see below.

**B6 — the town's statistics mode uses the same 160x242 card in the character pane.**
`townStatisticsCardRegion = chargenCardBox` (`pkg/ui/townshell.go:41`) currently ties the town's
stats card to the generator's. Moving `chargenCardBox` moves it, and the town's own character pane is
`TownCharacterRegion = (480,238)-(640,480)`, the same 160x242 slot. Draw the card there, replacing
the doll, which is what the `DOLL`/`STATS` toggle means. This is the second half of the reuse.

## The tip popup's rect, answered

**`TOWN-314` (High) answers this, and hotfix `23bb8e6` has already shipped the answer. The lane
inherits the rect; it does not build it again.**

`TOWN-187` reported the popup's construction immediates as `(0,280)-(312,480)` and flagged its own
result as anomalous, "the lower-left quadrant, unlike every other room's own tip rect". `TOWN-314`
reads the parent attach at the same call site, an instruction already present in `TOWN-187`'s own
cited evidence file but not read there as a parent attach: `R1870` constructs the popup at
`L11836` with `ECX=[ESI+0x7c]` and calls `R0385`, which sets `*(child+0x30)=this`. The
popup's paint routine resolves its absolute rect through `R1224`, the ancestor accumulator
`TOWN-232` establishes. The `+0x7c` child's own chain is empty at construction, so the walk applies
one step: `(0,0x118,0x138,0x1e0)` + `(160,0)` = **`(160,280)-(472,480)`**, lower-centre, straddling
the screen's own midpoint at 320.

**The general rule is what to carry out of this: a published construction literal is
parent-relative, and construction never leaves the ancestor chain empty**, so a literal LTRB is never
itself the absolute rect. `TOWN-314` enumerates all 7 `R1261` call sites, exhaustive with 0
orphans, and every one attaches a non-null parent immediately. Grade High for the mechanism and for
the chargen site's own resolution; Medium for the rule across those 7 call sites.

**This was a shipped defect, not only a documentation question.** Hotfix `23bb8e6` corrected
`ChargenTipRect` to `(160,280)-(472,480)` and `TavernTipRect` to `(160,0)-(472,322)`, both of which
had been shipped by reading a published literal as absolute. Both carried the correct WIDTH, so both
panels drew at the right size in the wrong place, and every test that mentions them reads the
constant symbolically and moved with it. At `(0,280)` the generator's panel covered
`preChoiceOrigin[0] = (16,273)`, the first hero portrait the player must click, at 97.9% of that
control's own box, and reverting the origin reddened no test in the module.

**The witness exists and constrains this story.** `pkg/ui/tiprectorigin_test.go`'s
`TestEveryTipRectSitsAtItsResearchedOrigin` pins the origin and width of all five tip rects against
each claim's own literal written on the right-hand side, mutation-proved 5 of 5 at +1 in X.
`ChargenTipRect` is already correct: changing it is out of scope, and any change to it must fail that
test. Height is deliberately not pinned, because height is `DIV-162`'s disclosed deviation.

**What B5 still owes is the panel's opening, not its rect.** The panel cannot open on the detailed
page at all today (item 9), and that is the behaviour to build.

## Domains

Client (`pkg/ui`, view parts of `pkg/game`) and Assets (`pkg/game/chargenassets.go`). Two domains.

## Ceiling

**Three passes**, one behaviour class across two domains, no reach into hashed simulation state. This
contract names six behaviours, which is above the five-behaviour split test, and it does not split
for one reason: B1 through B4 are one re-layout. Moving the centre without moving the card leaves the
card over the column; removing the message box without moving the centre leaves a hole. B5 and B6 are
separable and would each be a story of one behaviour with its own doc stack and landing, which is the
cost the atomicity rule exists to refuse.

**`1022` opens when `1021` lands.** It is not parallel with `1021`'s round 3: both edit
`pkg/ui/chargen_page.go` and `pkg/ui/townshell.go`, and B2 consumes `1021`'s pane type, which does
not exist on master yet.

## What is authored and what is not

Decoded, cited: the four border-column rects and their keyed blit mode (`TOWN-234`); the shared
lower-right panel's rect and transfer (`SHOP-FIGURE-041`); the tip popup's size, its two call sites,
its once-only latch and its two text files (`TOWN-187`); the class-conditional first text
(`TOWN-187`); the archive entry behind each of the four border strips (`TOWN-312`); the tip popup's
absolute rect `(160,280)-(472,480)` (`TOWN-314`), already shipped by hotfix `23bb8e6`.

Authored, each owing a typed row in `docs/DIVERGENCES.md`: which archive entry fills each of the nine
slots, where no claim names the reader; the 160x242 card's field arrangement and font; the
disposition of the message box's former copy; the town statistics card's move into the character
pane.

`UNIT-PANEL-011` states the original's panel layout cannot be recovered from the executable: the
value set and its arithmetic are on evidence and the layout is on nothing. The card's arrangement is
therefore the owner's and stays the owner's.

`DIV-190`..`DIV-198` are reserved to this story in `PIPELINE-STATUS.md`. **`DIV-189` is NOT free**:
story `1021` spent it on the undrawn lower-left seam (item 10), which this story closes rather than
opens. Allocate with `pipeline/next-div-id.sh` and read the `found at:` lines it prints. Ids not spent are returned
at the landing and recorded as returned.

## Out of scope

- **The map screen's own column.** `sidebarWidth = 300` (`pkg/ui/panel.go:654`) has five readers
  across `hud.go`, `inventory.go`, `minimap.go` and `panel.go`, and the owner ruled on 2026-08-21
  that it goes to the decoded 160 (*«давай все приводить к оригиналу»*; `SESS-VIEW-028` gives the
  map viewport as the screen minus a 160-pixel right strip, High). That move is **story `1023`**,
  which re-fits the minimap, the control panel, the worn set, the doll and the unit panel into the
  column's decoded slots. **This story does not change the constant**, and the reason is
  sequencing rather than scope: `1023` adopts the 160-wide card B3 builds, so the card must exist
  first. B3 is therefore not a second layout beside `AuthoredPanelLayout()` — it is the one card,
  built here and reused there, which is the reuse the owner asked for.
- The pre-create stage's own composition. Only its tip rect is touched, and only if research answers
  the question above before the landing.
- Splitting `shopmenu.bmp`, carried over from `1021` as a ledger row.
- `interface/chrgen/centerarea.bmp`, measured at 320x480 and unreferenced in this tree. Record it in
  `closure.md` as an unspent shipped source; do not draw it.

## Gates

Both repository chains by glob. `pipeline/check-release-tests.sh` and `pipeline/check-scenarios.sh`
on both roots, by name — this story changes what the screen draws, the repository chain cannot reach
those tests, and a skip and a pass both print `ok`. Report the count each script prints, not its
verdict. Deletion set empty or explained.
