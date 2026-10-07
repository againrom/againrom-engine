# 0153 — character-generator plan

## Approach

Keep the existing mission-entry gate and one-member party constructor, but replace the diagnostic
list inside that gate with a two-stage `Chargen` state machine and a headless RGBA compositor.
`pkg/game` resolves install bytes and produces the same panel subject, loadout and figure that a
confirmed party uses; `pkg/ui` owns stage transitions, geometry, input arbitration and pixels. No
world exists until valid Play invokes the launch closure already captured for the opening row.

## Facts verified during planning

- `App` already owns a 640-by-480 canvas and maps window coordinates through
  `Placement.WindowToFrame`; letterbox input therefore has one existing rejection point.
- `flow.armChargen` stores a fresh model, its Back destination and one `Begin` closure. Picker mission
  rows and direct `-mission` starts already use that seam, while campaign continuation bypasses it.
- `Chargen` already owns the legal cumulative-cost model. `FrontEnd.ChargenParty` is the only
  confirmed-result-to-party constructor, and `mapload.PartySpawnWithTable` is the mission mint's pure
  derived-value path.
- The live panel draws a `ui.PanelSubject`; its `UnitCombat` already gives weapon-spell damage
  precedence. `composeUnitFigure` already applies figure base, body, equipment, held-order and
  secondary-layer rules to a 160-by-240 result.
- The localized plate is `main/graphics/chrgen/leftup.bmp`. `graphics.res` holds
  `interface/chrgen/precreate/mainarea.bmp`, `buttonok.bmp`, four `heroes/{mf,mm,ff,fm}` state
  families, `precreate/mask.bmp`, two 320-by-480 `fighter|mag/column.bmp` surfaces and their indexed
  masks, and ten `fighter|mag/<skill>` state families. The detailed plate is 160 by 238; the
  pre-create background and mask are 640 by 480.
- `main/text/main.txt` carries prompt slot 125, skill hover slots 171–180, empty/reserved refusal
  slots 193–194, navigation slots 238/239/260, and the selected install's bytes. `font2` is an
  8-by-10, 224-record atlas and shares the existing language selector.
- `pkg/ui` cannot import archive, data, game or an external codec. External `x/text` imports are
  confined to registered `pkg/formats/*` leaves. Name storage is already presentation state outside
  the canonical world form, so no byte-form version changes.

## Files to touch

| Intent | Path | Responsibility |
|---|---|---|
| ADD | `pkg/formats/textinput/input.go` | Encode one Unicode rune to a Windows-1251 byte and apply the selector-1 inverse glyph mapping. |
| ADD | `pkg/formats/textinput/input_test.go` | Exhaustive accepted-byte mapping plus control, unencodable and selector cases. |
| MODIFY | `internal/archtest/dag.go` | Register the new leaf and retain the formats-only external-codec boundary. |
| MODIFY | `internal/archtest/dag_test.go` | Prove the new grant and continued denial outside the formats tier. |
| MODIFY | `pkg/ui/chargen.go` | Two-stage Draft state, direct choice/stat actions, name-byte editing, Reset/Forward/Back and preview invalidation. |
| ADD | `pkg/ui/chargen_page.go` | Source-mask hit ids, pointer latch, native shared production Card and reflowed 640-by-480 composition. |
| MODIFY | `pkg/ui/panel.go` | Make normal gameplay and generation call one full native framed character-panel component; preserve custom panel-layout substitution. |
| MODIFY | `pkg/ui/app.go` | Map input to the page, arbitrate one action, draw its RGBA frame and route pre-create Back or valid Play. |
| MODIFY | `pkg/ui/chargen_test.go` | Preserve point-buy behavior and cover stage, name, reset and pure-preview transitions. |
| MODIFY | `pkg/ui/chargen_app_test.go` | Cover keyboard/pointer dispatch, cancellation, letterbox rejection, Back chain and single launch. |
| ADD | `pkg/ui/chargen_page_test.go` | One-instance placement, source-mask hits, state selection, five-region bounds, production Card and stale-Doll clearing. |
| MODIFY | `pkg/game/chargen.go` | Supply the two-stage setup and build preview values from `ChargenParty`. |
| ADD | `pkg/game/chargenassets.go` | Load and validate source images, raw string slots, `font2`, language selector and rune encoder. |
| ADD | `pkg/game/chargenassets_test.go` | Synthetic archive tests for every address, slot, state mapping, localization and required-node failure. |
| MODIFY | `pkg/game/frontend.go` | Load the immutable generator presentation once and hand it to every fresh model. |
| MODIFY | `pkg/game/world.go` | Factor the party-member-to-panel projection so preview and live mission start call one expression. |
| MODIFY | `pkg/game/chargen_test.go` | Cover all 20 choices, exact weapons/loadouts, Card parity, preview purity and fresh two-stage gate entries. |
| MODIFY | `pkg/game/release_integration_test.go` | Carry one generated identity through launch, save/load and campaign transition. |

## Design decisions

### D-1 — One model carries two stages

`Chargen` gains an explicit `PreCreate`/`Detailed` stage. Pre-create holds stored name and combined
class/sex index. `Forward` constructs fresh detailed statistics and the configured skill index;
detailed `Back` drops those fields and returns to the same pre-create values. Pre-create Back is the
only model action that asks `App` to unwind to `flow.chargenBack`. A second flow screen was rejected:
it would duplicate the captured mission closure and make the Back preservation rule a transfer
between models instead of one state transition.

### D-2 — Results retain the existing wiring vocabulary

Internally the combined choice maps to the existing `(sex, class)` indices, and `ChargenResult`
continues to carry sex, class and skill in that order. New direct methods select a combined choice,
select a skill and adjust a named statistic; the detailed page never exposes the old generic rows.
Changing the party constructor's result shape was rejected because the existing constructor, flags
and continuation tests already agree on it and the new combined picture is presentation, not a new
character axis.

### D-3 — Stage geometry follows the source surfaces

Pre-create paints `mainarea.bmp` once at `(0,0)`. The four native choice patches are then painted
once at their matching background positions: `mf (16,273)`, `mm (416,190)`, `ff (124,166)`, and
`fm (288,130)`. Their rectangular bounds may overlap because they are source patches; their mask
hot pixels do not. Mask indices `80/140/100/120` map to those four choices. The prompt starts at
`(448,4)` and the Name field is `[448,20)–[628,46)`. Forward paints the native 100-by-56
`buttonok.bmp` at `(468,373)` over the lower `BOOK` marking while mask index 180 is hovered or
pressed. Back paints no text: mask index 160 activates the far-right ornate brooch and its native
112-by-204 `amulet.bmp` highlight at `(528,140)`. Leaving either control restores `mainarea.bmp`.
Paint order is background, four state patches, conditional keyed brooch/OK highlights, then the
source-font prompt/field. Pure-black patch pixels are skipped; every other source pixel is copied
opaquely.

Detailed keeps the plate at `[0,160)×[0,238)` and places the complete shared production Card at
`[0,300)×[207,480)`. That overlap covers only the plate's decoration below the remaining-points
value. The chosen `fighter|mag/column.bmp` is cropped without scaling from source
`[72,252)×[0,480)` into destination `[300,480)×[0,480)`. Fighter skill patch positions within the
source are `(88,93)`, `(92,126)`, `(88,182)`, `(84,225)`, `(88,250)`; mask indices in skill order
are `255/191/152/127/102`. Mage positions are `(200,150)`, `(72,165)`, `(132,98)`, `(140,228)`,
`(136,158)`; indices are `127/102/255/152/191`. Translation `(228,0)` maps both source masks and
patches into the centre; every fighter/mage hot extent lies inside the crop. The four value,
minus and plus rectangles are respectively x `82..102`, `107..127` and `132..152`, with y
`54+32i..74+32i`; remaining points occupy `[46,123)×[181,203)`. Values use the game font, while
minus/plus use the supplied 20-by-20 state bitmaps. The full framed upper-right
`[480,640)×[0,240)` contains Back, Reset and Play in three vertically centred rows, with each source
label measured and centred inside the rectangle that receives its hit. The framed lower-right
`[480,640)×[240,480)` contains the native Doll and redraws its border after the Doll. No state patch
is centred in a full-width band, no portrait or skill picture is painted twice, and only pure-black
key pixels are omitted.

### D-4 — Source states have one authored mapping

Pre-create `on/l/lon` and detailed `on/shine_off/shine_on` map to rest/hover/selected. A pointer
press latches the hit id and selected picture; release activates only the same id and always clears
the latch. Only the Draft's selected skill remains selected after release; keyboard focus alone
changes no pixels and no yellow focus outline is painted. Combining hover and selected into one
state was rejected because the install supplies distinct images and the contract exposes all three.
The statistic buttons map `nloff/loff/lon/nlon/disable` to rest, hover, pressed-under-pointer,
captured-outside and unavailable.

### D-5 — Input arbitration admits one control action

`App` maps the cursor once, then dispatches with `Escape`, completed pointer release, `Enter`,
`Up`, `Down`, `Backspace`, typed input as descending priorities. Press only changes the latch.
Down/Up wrap the stage's declared focus list, and Enter activates that id. A release transition is
complete in that frame, so no release edge remains for the mission screen; the existing global
single-screen switch likewise prevents Enter replay. Independent `if` statements were rejected
because two simultaneous edges could mutate and launch in one tick.

Pre-create also records the completed choice id and time. A second completion on that same choice
within an authored 500 ms window selects it and calls the same Forward method as graphical OK,
once. Expiry, a different choice or another control clears/replaces the candidate. An untimed
"second click sometime later" rule was rejected because it is not a double-click.

### D-6 — Stored bytes cross through a narrow encoder

`pkg/formats/textinput.EncodeRune` returns one CP1251 byte or refusal, then applies the inverse of
selector 1's glyph conversion (`C0..EF→80..AF`, `F0..FF→E0..EF`). `ChargenSetup` receives a closure
over the startup selector; `pkg/ui` sees only `(byte, bool)`. Capacity and control-byte checks happen
before commit, so a refused rune neither clears untouched `Danath` nor consumes space. Importing
`x/text` into UI/game or storing UTF-8 and converting only at draw time was rejected: either breaks
the dependency wall or makes displayed and committed bytes differ.

### D-7 — Required generator assets fail at construction

`LoadChargenAssets` reads the full pre-create background and indexed mask, keyed amulet and Forward
highlight bitmaps, plate, four three-state choice families, both source columns and masks, both
five-by-three skill banks, both five-state +/- button families, raw table slots and `font2`. A missing or malformed
required node returns an address-bearing error from `NewFrontEnd`; no half-pictorial page opens.
The loader keeps byte strings unchanged and assigns the font's existing language selector. Silent
fallbacks were rejected for controls and text because they would make a lawful-root completeness
failure look like an authored UI. Figure faces and equipment layers are expressly excluded from
this eager required set: they remain lazy preview inputs so a missing base is observable on the page.

### D-8 — Preview is one projection of the confirmed party

The setup callback first calls `ChargenParty`. A factored party-panel helper calls
`PartySpawnWithTable` once and fills `PanelSubject`, including pools, combat, absorption, all skills,
protections, experience, sight, speed, weapon name and weapon-spell interval. The same helper feeds
the mission's initial character projection. Doll composition receives that member's figure id and
generated worn set through `composeUnitFigure`. Failure to read its figure-face base returns a nil
Doll and no loader error; missing primary or secondary equipment layers skip only those layers and
still return the composed base. Recomputing card arithmetic in UI or layering a
second doll loop was rejected because either could disagree with Play.

### D-9 — Card is the production component

A shared character-panel renderer owns `AuthoredPanelLayout`, the native source-font draw and the
complete frame. Normal `Viewer` presentation and the generator call that same renderer. The
generator directly copies its measured 300-by-273 result to `(0,207)` without a wrapper frame,
resampling or later message overlay. A full-image equality regression compares its Card rectangle
with normal gameplay's component output for the same subject, while a normal gameplay regression
guards the shared bounds, frame and custom-layout substitution. There is no
`ChargenCardLayout`, no draft-only panel field, no copied label, no abbreviated skill family and no
second formula. A parity regression constructs a Draft, confirms it into the first mission member,
and compares the 19 context-free production-layout statements byte-for-byte. It separately requires
the preview's unplaced `CELL -, -` and the live member's actual cell. The shared call and complete
pixels for the same subject are the oracles, so deleting or bypassing it fails rather than letting
two copied expected lists agree.

### D-10 — Preview state is replaced atomically

Forward, accepted statistic/skill edits and Reset call the preview callback once and replace the
complete `(PanelSubject, Card, Doll)` cache. Refused edits do not call it. The UI always clears the
lower-right rectangle before consuming the replacement; nil then draws `PREVIEW UNAVAILABLE`, while
an optional-layer miss is a valid non-nil image. Thus readable-base→missing-base cannot retain one
old pixel. Updating Card and Doll separately was rejected because an intermediate frame could show
two different Drafts.

### D-11 — Validation precedes party construction

Forward performs no name check. Play checks zero bytes, then ASCII-folded whole stored bytes against
`Self` and `Computer`; it chooses the loaded empty or reserved message and stays detailed on refusal.
Only a valid result reaches the captured `Begin` closure. Begin/opener errors stay on the detailed
stage and are shown without retargeting the closure. Falling back to `Danath` in `ChargenParty` is
removed for generator results, because that would turn an invalid Draft into a different player.

### D-12 — Existing mission ownership and persistence stay authoritative

Every entry creates a new model but closes over its opening mission action exactly as today. Play
uses that closure once; Back never calls it. Save/load and campaign carry continue through the
existing `PartyMember.Name`, hero, worn set and carried-party paths, and campaign difficulty is never
written by either stage. A new generator-owned mission id or persistence record was rejected because
both would create a second owner for state already carried by the front end.

## Requirement coverage

| Requirement | Design decisions |
|---|---|
| FR-1 | D-1, D-3, D-4, D-5, D-7 and D-9 |
| FR-2 | D-1 and D-10 |
| FR-3 | D-2, D-8, D-10 and D-12 |
| FR-4 | D-8, D-9 and D-10 |
| FR-5 | D-1, D-5, D-6, D-7, D-11 and D-12 |

## Risks

- **R-1 — source patch bounds overlap or drift from their background.** Tests give every source
  surface and patch unique pixels, require one paint at each declared offset, and assert the indexed
  masks select disjoint controls; lawful-root verification records the real bounds and masks.
- **R-2 — Russian input is converted twice.** Encoder tests assert stored bytes while font tests
  assert the existing forward selector mapping lands on the intended glyph record.
- **R-3 — preview mutates state through a convenient mission path.** Preview accepts a value result,
  calls only party construction, spawn derivation and composition, and is tested against copied
  front-end campaign/purse/doc/save/random state.
- **R-4 — the generator silently grows a second character sheet.** A synthetic subject gives every
  production field a distinct value; generator and confirmed first-frame statements must be
  byte-identical except for the explicitly contextual `CELL` value, and the test fails when the
  shared presenter call is removed.
- **R-5 — stale art survives a failed composition.** The page clears both preview rectangles before
  every draw and tests readable→missing-base replacement at pixel level.
- **R-6 — Back or Play reuses the entry gesture.** Dispatch tests combine every transition with a
  press/release or Enter edge and assert exactly one stage change or launch call.

## Success criteria

1. `TestChargenPreCreateOwnsIdentityAndStoredName` and
   `TestChargenResetPreservesSkillAndReplacesPreview` prove stage field ownership and idempotence.
2. `TestChargenAC2StepCostIsTheCostDifference`, `TestChargenAC3ReachableStatesAreLegal` and
   `TestDetailedPointBuyRefusalsKeepTheWholeProjection` cover point-buy costs, bounds and refusal.
3. `TestChargenNamePreservesCaseAndStopsAtTenBytes` and `TestEncodeRune` prove replacement mode,
   both selectors, ten-byte capacity and refusal non-mutation.
4. `TestChargenSourcePlacementsMasksAndFrames`, `TestDetailedControlBoundsAndDollReplacement`,
   `TestDetailedSkillUsesSelectedHoverAndPressedSourceStates` and
   `TestDetailedMessageDoesNotAlterNativeCard` cover source geometry and framed previews.
5. `TestPreCreateFlow`, the three pre-create double-click/keyboard tests,
   `TestDetailedSkillClicksMoveTheOneSelectedSourceState` and
   `TestDetailedPlayUsesSourceRefusalsAndLaunchesOnce` cover input arbitration.
6. `TestChargenPreviewCardCoversEveryGeneratedIdentity`,
   `TestChargenPreviewUsesTheConfirmedPartyProjection` and the lawful release continuity test cover
   all 20 choices and compare the production statement, weapon/loadout and first mission projection.
7. `TestChargenPreviewLeavesPersistentFrontEndStateAlone` and
   `TestDetailedPointBuyRefusalsKeepTheWholeProjection` cover transient preview state.
8. `TestPreCreateFlow`, `TestChargenPreCreateOwnsIdentityAndStoredName`,
   `TestChargenResetPreservesSkillAndReplacesPreview` and
   `TestDetailedPlayUsesSourceRefusalsAndLaunchesOnce` cover Back, Forward, re-entry and one launch.
9. `TestReleaseGeneratedCharacterLaunchSaveLoadAndCampaignContinuity` covers exact identity,
   equipment and difficulty through launch, save/load and campaign transition.
10. `TestLoadChargenAssets` resolves every required EN/RU-shaped synthetic address and source slot;
    the environment-gated lawful-root drive repeats that census for both installs.
11. The runnable build visibly exposes both stages; mission 10/20 unsupported-node counts remain at
    the recorded baseline because this slice changes front-end presentation, not scripts.
12. `TestPreCreateControlBoundsAndNativePixels` proves the source background, four pictures, Name,
    Back and Forward have native pixels, exact hit bounds, cancellation and declared focus order.
13. `TestPreCreateFlow` proves fresh entry, stored identity, Forward without validation and
    pre-create Back without launch while leaving a buildable detailed-stage shell.
