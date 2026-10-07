# 0153 — character generator

**Intensity:** spec-anchored / static. **Terrain:** brownfield in the generator, front-end and
production character-sheet presenter; greenfield for transient preview values. **Threshold:** High for statistics,
the selected skill and the starting weapon. Name and presentation stay outside the world digest.

## Problem and current behavior

The current generator is a keyboard-only diagnostic list. It exposes sex, class and skill as three
text rows, draws no source art, accepts no pointer input, and shows derived values as debug text. It
does not show the in-game character card or dressed doll. Name input counts UTF-8 bytes and Play
does not apply the source name rules.

Replace it with a two-stage generator inside one redesigned 640-by-480 shell. Pre-create chooses
identity; the detailed editor shows the character that Play will create.

## Terms

- **Draft:** transient stored name, one combined class-and-sex choice, one skill position and four
  statistics. Pre-create owns its first two fields; the detailed stage owns the latter two.
- **Pre-create:** the Name, four class-and-sex pictures, Back and Forward stage.
- **Detailed stage:** the statistic, Card, skill, Doll and Back/Reset/Play stage.
- **Spread:** Body, Reaction, Mind and Spirit, in that order.
- **Skill position:** one of five shared positions. Fighter names are Blade, Axe, Bludgeon, Pike and
  Shooting; mage names are Fire, Water, Air, Earth and Astral.
- **Card:** the production in-game character sheet, presented inside the generator's lower-left
  destination without a second formatter, row set or value model.
- **Doll:** the class-and-sex base figure plus the selected hero's starting equipment layers.
- **Play:** the source-labelled Accept control that commits the draft and enters the selected
  mission.
- **Stored name:** zero through ten bytes in the selected install's character-input representation;
  Play refuses zero.

## Functional requirements

**FR-1 — Two stages share one pictorial shell.**

Both stages MUST draw in the existing 640-by-480 virtual frame. Pre-create shows the editable Name,
four source-backed class-and-sex choices, Back and Forward together. The detailed stage shows
statistic controls upper-left, Card lower-left, five class-dependent skill controls in the centre,
Back/Reset/Play upper-right, and the native 160-by-240 Doll lower-right. Its five functions MUST be
visible together. The Card MUST be the complete native production character panel, including its
frame and bounds. The centre source surface may lose unused side margins, but MUST retain every
skill picture and hot pixel at native scale and MUST NOT overlap the Card.

Pre-create MUST paint its full source background once, then paint each selected/hover/rest choice
patch once at that choice's source position. Detailed MUST paint the selected class's 320-by-480
source column once in the centre, then paint each skill-state patch once at its matching source
position. Source patches MUST be drawn at native size and MUST NOT be stretched into authored
bands. Rectangular source backgrounds are part of the pictures. Pure black in a control patch is the
owner-confirmed key and MUST leave the background beneath it unchanged; every non-black rectangle
pixel remains opaque. The source masks define disjoint hot pixels even where patch bounds overlap.
Rest, hover and selected states MUST differ where the install supplies distinct pixels. Pointer
targets MUST equal their source-mask hot pixels for pictorial controls and the drawn rectangles for
text/statistic controls. A pressed control uses its selected state until release; cancelled release
restores the state under the pointer. Exactly one skill uses the source selected state, and it is
the Draft's selected index; pointer hover and keyboard focus MUST NOT pin another skill in that
state. No authored yellow focus outline is painted. Letterbox pixels MUST activate nothing.

The four statistic values and remaining-points value MUST use the install's game font and be
centred in the source panel's decoded value rectangles. Minus and plus MUST use the supplied native
20-by-20 button-state pictures rather than host-font signs. Pre-create's brooch and graphical OK
MUST visibly use their supplied keyed highlight patches while hovered. A second completed click on
the same class-and-sex picture within the authored 500 ms double-click interval MUST select it and
perform the same single Forward transition as graphical OK; a later or different second click MUST
not transition.

A pointer press and release inside one control MUST activate it once; release elsewhere cancels it.
Pre-create keyboard focus MUST cycle through name, the four class-and-sex choices in row order,
Back and Forward. Detailed focus MUST cycle through five skills, each statistic's minus then plus in
Spread order, and Back, Reset, Play. Down moves one place forward and Up one place backward, both
with wrap. Enter activates the focused control, Escape activates that stage's Back, and typed
characters and Backspace act only while Name has focus. One frame performs at most one action.

**FR-2 — Statistic buttons implement the source point buy.**

The opening spread MUST be `25/25/25/25` with 100 points remaining. Each row MUST show minus,
current value and plus. One accepted activation changes only that statistic by one. Values remain
in the inclusive range 15 through 45. The cumulative cost of value `n` is

`T(n) = trunc(0.349 × 1.15^(n-1) + 0.5)`.

The remaining counter MUST equal `140 - ΣT(stat)`. Plus MUST be refused when it would cross 45 or
make the remainder negative; minus MUST be refused at 15. Refusal leaves the complete draft and both
previews unchanged.

**FR-3 — Class, sex and skill select the starting hero.**

Exactly one pre-create choice MUST be selected from male fighter, male mage, female fighter and
female mage. The opening choice is male fighter. Changing it changes no detailed state because none
exists until Forward.
The choice selects the existing character-definition table's declared generation base for that
class-and-sex pair; its face, profile and equipment rows feed both preview and confirmed player. If
no declared base resolves, face 1, the empty profile and empty equipment are the common fallback.
The chosen ordinary weapon occupies its own declared equipment slot through the existing generated
loadout rule. It replaces the base-row item in that slot or fills an empty slot; it is never added as
a second weapon or displaced into carried inventory.

Forward creates the detailed stage from the current name and class-and-sex pair. Exactly one skill
position MUST then be selected. The existing front-end skill setting supplies its opening position,
clamped to 1 through 5; without that setting it is position 1. The selected class-specific slot MUST
start at 10 and the other four visible slots at zero. The ordinary starting weapon is:

| Position | Fighter | Mage |
|---:|---|---|
| 1 | Blade; Iron Short Sword | Fire; Fire Arrow staff |
| 2 | Axe; Uncommon Bronze Axe | Water; Fire Arrow staff |
| 3 | Bludgeon; Uncommon Bronze Mace | Air; Fire Arrow staff |
| 4 | Pike; Bronze Pike | Earth; Fire Arrow staff |
| 5 | Shooting; Uncommon Wood Short Bow | Astral; Fire Arrow staff |

Changing skill MUST rebuild Card and Doll immediately. Mage positions share the staff but train
different slots. Preview and Play MUST never grant a second level-10 visible slot.

**FR-4 — Card and Doll are live previews of the confirmed player.**

For the same Draft and definitions, the Card MUST call the exact production character-sheet
presenter with a transient subject projected through the same party-member adapter as the
confirmed player. Its labels, class-sensitive skill mapping, derived formulas, weapon-or-spell
damage selection, defence and absorption, conditional rows and line ordering MUST have one source
of truth. Normal gameplay and the generator MUST call one shared character-panel component with
the same model, statement, source font and conversion, native bounds and frame. The generator may
change only the component's destination. It MUST NOT
define a compact row set, copy field labels or formulas, or introduce a draft-only sheet field. A
spell-bearing staff therefore shows the invoked spell's damage range because the production sheet
does. The complete generator Card image MUST be pixel-identical to the normal gameplay component
for the same subject. The preview subject is not placed on a mission map, so its production
`CELL` row MUST read `CELL -, -`. After Play, the confirmed first player's production `CELL` row
MUST show its actual map coordinates. Every other `PanelStatement` line MUST equal the preview
line byte-for-byte. Destination position is excluded from the image comparison.

The Doll MUST be the same 160-by-240 dressed image that the in-game doll composer produces for the
confirmed player: selected class-and-sex figure, selected base-row face, base-row starting equipment
with the chosen weapon, and every readable primary and secondary layer in the in-game order. A
missing optional layer MUST suppress only that layer. A missing base MUST clear the preceding Doll
and show `PREVIEW UNAVAILABLE`. Here “missing base” means that the selected or fallback figure-face
bitmap cannot be read; it is distinct from an unresolved definition row's declared fallback above.

Every accepted statistic or skill change and Reset MUST rebuild both detailed previews before the
next frame. Forward builds their first images from the current pre-create choice. Previewing MUST
NOT create or mutate a world, party, clock, purse, document bundle, campaign difficulty, random
stream, save or persistent inventory.

**FR-5 — Name, Forward, Reset, Back and Play use source data and distinct transitions.**

The selected install MUST supply the statistic plate, pictorial controls, hover text, prompt,
navigation labels and refusal messages. The navigation labels are:

| Install | Name prompt | Play | Reset | Back |
|---|---|---|---|---|
| EN | `Character name:` | `Accept` | `Reset` | `Back` |
| RU | `Имя персонажа:` | `Принять` | `Сбросить` | `Назад` |

Forward is the source pre-create confirmation bitmap with its baked `OK` pixels. “Forward” names
its transition here; the renderer MUST NOT replace those pixels with a translated text label. It is
painted over the lower `BOOK` marking and activated by that source-mask region. The far-right ornate
brooch is pre-create Back to the main menu; no separate Back text is painted near the name scroll.

Pre-create opens with editable `Danath`. The first accepted typed character replaces that untouched
default. Backspace or any earlier accepted edit ends replacement mode; later accepted characters
append. Other controls MUST NOT alter the name.

The selected install's language selector, read when the generator is built, is the character-input
selector. Each typed Unicode character MUST first encode to one Windows-1251 byte. Unencodable
characters, bytes below `0x20`, DEL and characters exceeding ten stored bytes MUST be ignored.
Backspace removes one stored byte. Selector 1 then maps `C0..EF` to `80..AF` and `F0..FF` to
`E0..EF`; every other selector leaves accepted bytes unchanged. The field MUST draw stored bytes
through that install's normal glyph mapping.

Forward MUST enter the detailed stage without validating the name; the transition tests nothing.
The page reaches it through OK, Enter on OK and the hero double-click only with a non-empty name
(`TEXT-CHARGEN-028`, `VIDEO-SFX-058`, `DIV-1507`). Each activation rebuilds the
opening spread, configured skill position, ordinary weapon, Card and Doll; abandoned detailed
edits do not return through it.

Reset MUST restore only the spread and remainder to `25/25/25/25` and 100. It preserves name,
class-and-sex choice, skill position and campaign difficulty, then reconstructs Card, Doll and
starting equipment. Repeating Reset has no further observable effect.

Detailed Back MUST return to pre-create with name, class-and-sex choice and campaign difficulty
preserved. It discards spread, skill and preview state; the next Forward rebuilds their declared
opening values. Pre-create Back MUST return to the existing mission picker without creating a party
or world. Re-entering that row creates the declared opening pre-create state rather than restoring
the abandoned Draft.

Play MUST first refuse an empty stored name, then an ASCII-case-insensitive whole-name match of its
stored bytes for `Self` or `Computer`. The selected install's empty-name message or reserved-name
message, respectively, MUST be shown while the detailed stage and Draft remain open. Refusal creates
no party or world. An accepted Draft
commits the stored name bytes exactly, creates one player from the displayed draft, and invokes the
mission-launch action captured when that picker row opened the page exactly once. Later picker-list
changes cannot retarget it. It proceeds directly to that mission without returning to the picker.
If the captured action fails, the error is shown and the page remains open; the activation that
successfully leaves the page MUST NOT reach the mission screen.

EN and RU runs MUST read their own label bytes and statistic plate. The four pre-create controls,
both five-control skill banks and every visible Doll layer MUST resolve from the selected lawful
install. Replacing a supplied image or localized plate changes only its pixels. A different name
limit, sixth skill, third class bank or split class/sex axes changes executable behavior but no
shipped file bytes.

## Acceptance criteria

| Id | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | UI | A synthetic 640-by-480 source with distinguishable backgrounds, masks and control images | Both stages draw and each pointer edge is exercised | Pre-create paints four patches once, restores/highlights lower BOOK-OK and the far-right Back brooch from source states, and a timed same-choice double-click performs one Forward; detailed paints one native source crop and five compact patches; exact statistic rectangles use game-font values and source +/- states; source hot pixels agree with hits; black-key pixels preserve the background; no yellow overlay exists. |
| **AC-2** | unit | Opening and boundary-adjacent spreads | Every plus and minus is attempted | One value changes only when its bound and the cumulative budget allow it; remainder equals the formula; every refusal leaves Draft, Card and Doll unchanged. |
| **AC-3** | unit + UI | All 20 class/sex/skill combinations | Each pre-create choice is forwarded and sword→axe→mace→pike→bow is clicked | The exact five-control class bank, one level-10 slot, four zero slots, stated weapon, Card and dressed Doll agree in one frame; exactly the clicked picture is selected and every prior picture has returned to rest. |
| **AC-4** | API + UI | Synthetic definition values distinct in every production-sheet field | A Draft is previewed, confirmed and selected in its first mission frame | Generator and normal gameplay call one shared full native panel component. Its framed pixels are equal for the same subject. Preview and live statements have equal labels, order, conditional rows and values except that the unplaced preview states `CELL -, -` and the live panel states the actual cell. Fighter and staff damage use the production presenter's invoked-attack rule. Deleting or bypassing the shared component call or selected-index read makes the parity test fail. |
| **AC-5** | UI, error | Distinct-colour base, primary and secondary Doll layers; one optional miss; then a missing base | Each Doll is composed | Pixels establish equality with the in-game order; the optional miss preserves later layers; the base miss clears old pixels and shows `PREVIEW UNAVAILABLE`. |
| **AC-6** | unit + UI, error | Pre-create with untouched `Danath`, ASCII, Cyrillic, control, unencodable, ten-byte and over-capacity input | Text and Backspace are applied under selectors 0 and 1 | Replacement mode, accepted stored bytes, conversion, display and refusal follow FR-5; refused input changes no Draft field. |
| **AC-7** | unit + UI | A modified name, choice, skill and spread | Reset is activated twice | Both results preserve name, choice, skill and difficulty, restore the opening spread and remainder, and reconstruct identical Card and Doll. |
| **AC-8** | UI | Modified pre-create and detailed state | Detailed Back, Forward, pre-create Back and re-entry are exercised | The first Back preserves name/choice but the next Forward rebuilds stats/skill/previews; the second Back creates nothing; re-entry restores `Danath` and all opening values. |
| **AC-9** | unit + UI, error | Empty, `Self`, `computer` and valid names forwarded from pre-create | Play is attempted | The two invalid classes show their source message, retain the detailed Draft and create nothing; the valid name advances. |
| **AC-10** | e2e | A valid EN or RU Draft and non-default campaign difficulty | Play succeeds, then the player is saved, loaded and carried across a campaign transition | Exactly one player has the displayed spread, sole level-10 skill, weapon, dressed loadout and exact stored name throughout; difficulty is unchanged and the leaving input is not replayed. |
| **AC-11** | API | Fixed persistent application state | Arbitrary edits, refused edits, Reset and refused Play are replayed | World, party, digest, clock, purse, documents, inventory, difficulty, random stream and saves equal their initial values. |
| **AC-12** | e2e + manual | Lawful EN and RU installs | Both stages exercise all labels, image states, choices and skill banks | Each root draws its own text and plate; all declared controls and visible Doll layers resolve; both stages are visible in the runnable build. |

## Derived properties

- **P-1 — invariant.** For any reachable spread, each statistic is in `[15,45]`, total cost is at
  most 140 and remainder is the exact difference.
- **P-2 — completeness.** For either class, the five controls cover the five visible slots exactly
  once and exactly one confirmed visible slot equals 10.
- **P-3 — invariant.** For any detailed Draft, Card, Doll and successful player project the same name,
  class, sex, skill, spread, weapon and equipment.
- **P-4 — negative invariant.** For any refused statistic edit, name byte or Play, persistent and
  hashed state do not change; a Play refusal may change only the visible message and transient focus.
- **P-5 — idempotence.** For any Draft, `Reset(Reset(Draft))` is observably equal to
  `Reset(Draft)`.
- **P-6 — negative invariant.** For any missing optional Doll layer, later layers remain; for any
  missing base, no pixel from the preceding Doll remains.

## I/O examples

| Input | Observable result |
|---|---|
| Fighter position 3 | Bludgeon 10, other visible fighter skills 0, Uncommon Bronze Mace. |
| Mage position 4 | Earth 10, other visible mage skills 0, Fire Arrow staff and spell damage. |
| Untouched `Danath`, then `A` | Stored and displayed name `A`. |
| Empty name, Forward, then Play | The detailed stage opens; Play shows the source empty-name message and creates no party or world. |

## Constraints

| Choice | Credible alternative | Observable consequence |
|---|---|---|
| Combined class-and-sex choice | Independent selectors | Four supplied pictures remain the complete choice set. |
| Keep rejected Play on the detailed stage | Return to pre-create on refusal | The Draft remains intact; Back is the explicit route to correct its name. |
| Name outside canonical world bytes | Hash the display name | Save and campaign continuity preserve the name without changing simulation identity. |
| Source-positioned keyed art and mask hits | Stretch controls into authored bands | Each patch is painted once over its matching background; only its pure-black key is skipped and source hot pixels remain authoritative. |

Automated fixtures MUST be synthetic. No game image, font, string payload or other game byte may
enter the repository. Live EN/RU checks read only the owner's lawful roots.

## Out of scope

- Multiplayer participant, lobby, transport and conflict UI.
- A sixth skill, third class, additional sex value or split class-and-sex axes.
- New or translated game art, fonts or strings, and changes to either lawful install.
- Changes to money, documents, experience rules, simulation bytes, combat or mission rules.

## Verification mapping

| Acceptance | Method |
|---|---|
| AC-2, AC-6, AC-7, AC-9, AC-11 | CI-automatable unit/API tests over synthetic setup and input. |
| AC-1, AC-3 through AC-5, AC-8 | CI-automatable UI/API tests over synthetic images, fonts and definitions. |
| AC-10 | CI-automatable front-end/save/campaign integration test over synthetic sources. |
| AC-12 | Environment-gated EN/RU drive and owner-visible runnable build. |
| Clean-room constraint | Repository asset scan and deletion diff. |

Every FR is covered: FR-1 by AC-1 and AC-12; FR-2 by AC-2; FR-3 by AC-3 and AC-10; FR-4 by
AC-3 through AC-5 and AC-11; FR-5 by AC-6 through AC-12. Error cases yield P-4 or P-6.
