# The second game's character generator as the original shows it

## Intent

The second game's generator shows the original's look for seven owner-reported
defects on `rom2-en` and `rom2-ru`: difficulty and control hover help, the name
field's inks, the detail tip's position, the attribute digits, a fourth command
with Restore, the hover box and the statistics card. Every fix is description
data for the one generator builder (`pkg/ui/generator.go`, `chargen.go`,
`chargen_page.go`) and the one hover box painter (`pkg/ui/widgethover.go`); no
code path asks which game runs. A parity guard keeps the two descriptions from
drifting apart again. Base: public main `c735bc69`; pin k258.

## Authority

- `R2-ENGINE-384`: pre-create hover help lines (difficulty 247..249, heroes by
  mask 250, 252, 253, 251, Cancel 255, OK 254, name 256); detail skills 171+i
  and 176+i.
- `R2-ENGINE-385`: the EN hover box: font2 in (160,120,50) over (8,8,8) at
  (1,1), `#` lines kept as authored with blank ones, no wrap.
- `R2-ENGINE-386`: font4 prompt in (185,159,73) and name in (255,255,116).
- `R2-ENGINE-387`: values in font4 at (84,58+32i) in (185,159,73) over black at
  (1,1); the pool centred on (84,194) less half the first glyph's width;
  caption ink (185,159,73).
- `R2-ENGINE-388`: the shared sheet's inks and rectangle-relative anchors.
- `R2-ENGINE-389`: tip bounds, parent-relative.
- `R2-ENGINE-249`: the generic shop's buttons child (464,0,640,238) and the
  four `ShopButton` rectangles; `R2-ASSET-090` and the installed
  `graphics.res` listing: `interface/shopmenu.bmp` and `shopbutton1..4.bmp`
  exist in the second game.
- Owner direction: the detail tip at the bottom centre; four commands with
  Restore as on the first game (`DIV-2868`).
- Owner observation: the screenshots in the seat's owner-artifact directory,
  sampled on the original's 640x480 frame at 1:1 (shot 3, frame origin
  (35,39) in the image). Each sampled value is named in its divergence row.

## As built

`pkg/game/generators/rom2.json`:

1. Levels, heroes, Cancel, OK and the name carry `tooltip` lines from
   `R2-ENGINE-384`.
2. The name field's `ink` is (255,255,116) and the prompt's (185,159,73).
3. `tips.detail-rect` is (160,280,472,480) (`DIV-3015`).
4. `stats` gains `value-font` font4, `value-at` (2,4), `value-shadow` black,
   `value-ink` and `pool-ink` (185,159,73) and `pool-at` (84,186); the command
   ink is (185,159,73).
5. The command panel is the shop's composition: `interface/shopmenu.bmp` keyed
   over (464,0)-(640,238) and Accept, Restore, Reset, Back with
   `shopbutton1..4.bmp` at the shop's rectangles; `restore` takes the template
   values and the opening skill; `skill-clip` ends at x 464 (`DIV-3014`).
6. A new description object `hover` names the hover help style: font `text`
   (the generator's font2), ink (160,120,50), shadow (8,8,8), lines
   `authored`. `GeneratorHover` decodes it; `App.hoverStyle` hands it to the
   one tooltip builder while the generator shows, and `composeHoverBox` draws
   that ink over that shadow. `tooltipPicture` keeps authored lines whole. A
   nil style is the shared hover help, so every other hover surface and the
   first game draw as before (`DIV-3016`).
7. `card-offset` (12,0) and the keyed `interface/chrgen/fullstatsr.bmp` strip
   at (160,238)-(176,480) (`DIV-3017`).

The first game's description is unchanged. `DIV-2769` and `DIV-2770` narrow
to what stays Unknown; `DIV-2868` names the second game's Restore.

## Proof

- `TestSecondGeneratorDescribesTheOriginalLook` (`pkg/game`): one subtest per
  item on the description. Loss control: with the base `rom2.json` all seven
  subtests fail.
- `TestGeneratorDescriptionsShareEveryField`: every field one description sets
  and the other lacks is stated or waived with a reason; a waiver that covers no
  gap fails. `TestGeneratorParityGuardCatchesARemovedField` removes each of
  eight fields this story adds to `rom2.json` and sees the guard fail.
- `TestHoverStyle*` and `TestGeneratorHoverStyleDecodes` (`pkg/ui`): the styled
  box's ink and shadow, authored lines, and the shared box unchanged.
- `TestReleaseSecondGameGeneratorDrawsAsTheOriginal`: on the install, one
  subtest per item through one live App. Loss control: with the base
  `rom2.json` it fails.
- `TestReleaseSecondGameGeneratorRestoreReturnsTheOpeningDraft`: Restore after a
  spend, a skill change or a Reset returns each picture's opening result, the
  result Accept commits and the SAV records.
- `TestReleaseGeneratorWitnessFrames` hashes four frames per install and writes
  them for the owner. The first game's eight frames (EN and RU) hash the same
  before and after this story.

## Open debt

- Hover help outside the generator in the second game keeps the shared style;
  the RU receiver is not read (`DIV-3016`).
- The hovered caption ink (255,255,116) is not drawn; font4's first glyph
  height and width are not read, so the name origin and the pool row are
  fitted (`DIV-2769`, `DIV-2770`).
- The tip's parent and the card's caller rectangle are not claimed
  (`DIV-3015`, `DIV-3017`).
- RU does not refuse `~` in the name (`DIV-2769`).
