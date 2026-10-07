# Story 1302: shop and school presentation on k140

## Intent

Adopt the claims published for the shop and school presentation remainder
(known defect B10) and close or narrow the rows DIV-1550, DIV-1405, DIV-143,
DIV-142, DIV-1263 and DIV-1404 (docs/1286/story.md holds the questions).

## Authority

Pin k140. Claims `SHOP-106`..`SHOP-109`, `TOWN-499`..`TOWN-502`; `TOWN-022` is
amended by `TOWN-502`. `MISSION-MSGLINE-056` (ramp and shadow) and `TOWN-469`
(grouping) are the earlier claims the new ones lean on. Owner direction in
DIV-1263 (installed voices must play) is treated differently for the shop and the
school. The shop's selection speech is owner-requested, so it stays and outranks
`SHOP-109`'s draw until the owner rules (DIV-2115). The school's selection speech
was authored policy, and `TOWN-502` replaces it with speech on a paid train.

## As-built behaviour

- `SHOP-106`, adopted. The shop grid's quantity and the purse in the money cell
  draw in font2, left-aligned at the cell's left plus 10 and bottom less 15, in
  the gold ramp over a flat 8 8 8 shadow. `drawShopCell` takes the font the
  price already uses (`ShopScreenView.PriceFont`). DIV-1550 and DIV-1405 are
  CLOSED.
- `SHOP-108`, adopted for call operands. The mission pack bar draws the purse and
  stack counts above 1 in font2 with the same ramp and flat shadow. The claim
  does not name the screen or the text origin, so DIV-2116 records both.
- `SHOP-107`, no change. The shop's pressed-and-hovered offset (caption and
  number one pixel lower, button picture re-blit) already equals the claim; the
  hover ink stays the owner's request because the RGB of the two colour objects
  is Unknown. A pin test was added. DIV-1404 narrowed.
- `TOWN-501`, no change. The school's pressed arm (on picture, both texts one
  pixel lower) already equals the claim and is witnessed by
  `TestReleaseSchoolCaptionsMatchInstalledWordsAndPixels`. DIV-1404 narrowed.
- `TOWN-500`, adopted for the player-visible part. At rest the school draws no
  state picture (dword -1) and the idle painter draws the shine picture of a
  cycling slot every 500 ms or more, the shine-on picture for the selected slot,
  over the slot's state picture and only while no skill is hovered. The slot
  index steps in the stored order of the original's arrays (the `shineCycle` fields of
  `schoolTrainingStatic`) and maps to this build's shared slot order.
  The mage branch, the other timers and the destination rectangle are not
  established (DIV-2113). DIV-143 narrowed.
- `TOWN-499`, no change. The 83 ms strict single-step gate already equals the
  claim. The clock lifecycle difference (global, first-paint offset, no restart
  at a retarget) is stated in DIV-142; paint frequency is Unknown.
- `TOWN-502` and `TOWN-022`, adopted. Selecting a skill requests no speech. A
  paid training step requests `speech/training/npc33s%dl%d.wav` (fighter) or
  `npc34s%dl%d.wav` (mage) with the slot and tier plus one (tier 0 up to level
  20, 1 up to 50, else 2), once per latch, then posts
  `town/school/teach.wav`. The latch table has 31 dwords indexed
  `class + 6*slot + 2*tier` (class 0 or 2), set on every school entry and on a
  new game. The latch lifetime and the unlatched teaching sound are not
  established (DIV-2114).
- `SHOP-109`, divergence. The shop keeps speaking every selected item (owner
  request); the claim's 30 percent draw and single file per request are DIV-2115
  (CONFLICT, owner decision). The tavern requests only slot sounds; whether
  tavern dialogue is voiced is Unknown and unchanged.

## Proof

Each test below fails on the parent revision or names a field the parent lacks.

- `TestQuickSpellMarkInkIsTheOriginalGoldNotThePackReadoutInk` (pkg/ui): the mission
  spell bar's quick-spell digits keep ink (200,174,84), pinned by a literal that does
  not follow `shopPriceInk`. A review found them recoloured with the pack readout;
  the mark now has its own `quickSpellMarkInk`.
- `TestShopQuantityAndPurseDrawInTheGridFontLeftAligned` (pkg/ui): every glyph
  of an item cell's quantity and a money cell's purse belongs to the grid font,
  starts at (left + 10, bottom - 15) with its shadow one pixel right and down.
  Parent: the glyphs belong to the mission font.
- `TestReleaseShopGridQuantityAndPurseAreRampedWithAShadow` (EN and RU): the
  App's shop frames on four shelves and a synthetic population of counts (2 to
  9,999,999) in three grids and two money cells equal an oracle that decodes
  font2 from the install and draws the ramp and shadow. Loss controls: a pixel
  left, up and right, no shadow, the shop's former ink, a black shadow, font1
  glyphs and a right-aligned figure each differ from the screen.
- `TestPackReadoutCountsDrawInTheGoldRampOverAFlatShadow` (pkg/ui) and
  `TestReleaseMissionPackCountsDrawInTheGoldRampOverAFlatShadow` (EN and RU):
  the production App's pack layer with a stack of 12 differs from the layer with
  a stack of 1 only by ramp pixels and 8 8 8 shadow pixels, the shadow lying one
  pixel right and down of the face. Loss controls: the previous flat ink with a
  black shadow, a black shadow and a grey ramp each fail the same classification.
- `TestSchoolIdleShineDrawsTheCyclingSlot` (pkg/ui) and
  `TestSchoolShineStepsEveryHalfSecondModuloFive`,
  `TestSchoolViewCarriesTheIdleSlotInTheSharedOrder` (pkg/game).
- `TestReleaseSchoolIdleShineDrawsTheStoredSlotsShinePicture` (EN and RU): for
  both classes and all five stored slots the production school frame equals the
  frame without the idle shine with the installed shine bitmap, read at the
  claim's path and keyed on pure black, centred in the slot's rectangle. Loss
  controls: the neighbouring stored slot's picture, the shine-on picture and the
  on picture each differ from the screen.
- `TestSchoolTeacherSpeaksAfterAPaidStepOncePerLatch`,
  `TestSchoolLatchesShareDwordsAcrossClasses` (pkg/game) and
  `TestReleaseTownSpeechResponses` (EN and RU): selection is silent; 30 paid
  steps (two classes, five slots, three tiers) each request the installed
  speech file of their tier and post the installed teaching sound, with the
  latch silencing a second step in a tier and a new entry setting it again.
  Loss controls: an unpaid step posts nothing; level 21 and level 51 take the
  second and third file where the former 34-point bands took the first and
  second.
- `TestShopPressedAndHoveredCaptionsDrawOnePixelLower` pins `SHOP-107`'s offset.

## Open debt

Falsifiable questions, with no expected answer:

- Which rectangle does the school's idle painter draw into, what do its other
  timers and the mage branch do, and what are the counter's initial values
  (DIV-2113)?
- Who sets the teacher speech latches between visits, is the skill byte read
  before or after the step, are the band edges inclusive, and is the teaching
  sound posted when the latch is clear (DIV-2114)?
- Which player screen shows the pack readout object and where does it draw its
  text (DIV-2116)?
- What are the displayed RGB values of the two caption colour objects (DIV-1404)?
- Owner decision: does the shop follow the 30 percent draw and one file per
  request (DIV-2115)?
