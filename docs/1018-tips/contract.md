# Contract — 1018-tips

## Result

After this story the town square, the shop, the school, the tavern and the character generator each
show that screen's own shipped tip text in a bordered panel drawn over the screen. The panel carries
a close control and a labelled toggle. Closing the panel removes it for the rest of the visit.
Turning the toggle off suppresses the panel on every screen, and the suppression survives quitting
and restarting the game.

The owner asked for this on 2026-08-19: "у окон есть такое понятие как «Подсказка» это некое
floating окно, которое можно «Закрыть» и оно закрывается и больше не появляется", and then named the
generator's, the shop's, the school's and the tavern's texts one by one.

**Observable result.** Run `builds/current/againrom.exe -assets <install>`, start a campaign, and
enter each of the five screens. Each shows its text in the panel. Close one: it does not return on
re-entry in that session. Turn the toggle off, quit, restart: no screen shows a tip.

## What this build does today

Story `1011` (2026-08-18) built the shop's tip and nothing else. `pkg/game/shoptip.go` reads one tip
node whole (`ReadShopTip`), and `pkg/ui/shopscreen.go` wraps it and writes it as plain text over the
shop floor at `shopTipRect` = (164,162,476,298). There is no border, no close control and no gate:
the text is drawn on every shop entry for the life of the process. `DIV-132` and `DIV-133` record
what that story left open.

The other four screens draw no tip. `DIV-154` records the town square's.

This build persists no user preference of any kind. `audio.Settings` is playback configuration handed
to the mixer, not a stored preference. `DefaultSaveDir` resolves `saves/` beside the running
executable, and that directory is the only per-user location the game writes to.

## The shipped texts

`main.res` carries eleven nodes under `text/tips/` and nine more under `text/battle/m10/` and
`text/battle/m20/`. Both roots carry all twenty. Sizes in bytes, measured at this seat with
`restool list`:

| node | EN | RU | screen this story places it on |
|---|---|---|---|
| `text/tips/town.txt` | 444 | 493 | town square |
| `text/tips/shop1.txt` | 208 | 150 | shop |
| `text/tips/shop2.txt` | 222 | 183 | none — see out of scope |
| `text/tips/training.txt` | 505 | 368 | school |
| `text/tips/inn.txt` | 443 | 421 | tavern |
| `text/tips/chrgen1f.txt` | 416 | 323 | generator, fighter |
| `text/tips/chrgen1m.txt` | 415 | 376 | generator, mage |
| `text/tips/chrgen2.txt` | 258 | 321 | none — see out of scope |
| `text/tips/chrsel1.txt` | 168 | 149 | none — see out of scope |
| `text/tips/chrsel2.txt` | 199 | 165 | none — see out of scope |
| `text/tips/chrsel3.txt` | 160 | 147 | none — see out of scope |

**`chrgen1f` is the fighter text and `chrgen1m` is the mage text.** This is established from the
files' own content, not from their names: `chrgen1f` reads "one of the five weapon skills" and
`chrgen1m` reads "one of the five magic spheres". One directory over, in
`graphics.res` under `equipment/`, the same two letters mean gender — `ffighter`, `fmage`,
`mfighter`, `mmage`. A lane that reads the names alone will pick the wrong file for half the heroes,
and both files are the right length and the right shape, so nothing will fail.

The trap caught a reader already, and was corrected before this story's pin. `TOWN-187` reads the
generator's own branch at instruction level. As first published it called the selection
gender-conditional, on `TOWN-149`'s own "class/gender" hedge. Research corrected it at `8990406`,
which is an ancestor of the pin: `+0x18c & 0x2` is the mage/non-mage class split, read the same way
by `TOWN-149` and `HERO-FIGURE-059`, and `+0x18c & 0x4` is the sex bit, read by `HERO-APPEAR-055`
and `HERO-APPEAR-045`. The corrected row also names the `1f`/`1m` filename trap directly and
corroborates the class reading against both files' own shipped English text. **This story selects by
class, and the pinned claim agrees, so no divergence row is owed for it.**

Three of the shipped texts name buttons this story does not change, and they are the reason two
button labels in `1017` are correct rather than defects. `training.txt`: "The sum located on the
`<train>` button shows the training price, and the sum located on the `<exit>` button shows the
amount of money in your possession." `inn.txt` names `<Talk>`, `<Hire>`, `<Fire>` and `<Exit>`.

## Behaviours

1. **A tip panel widget, drawn from shipped art.** A panel holding the screen's tip text, wrapped to
   the panel's own width by the existing `wrapShopTip` measurer at `DLG-WRAP-009`'s pitch. It carries
   a close control and a labelled toggle. It draws over the screen's other content and takes input
   before it.

   **The art for it ships and this build reads none of it.** Measured and rendered at this seat,
   2026-08-19:

   | node | size | what it is |
   |---|---|---|
   | `interface/t_back.bmp` | 160x240 | dark mottled brown fill texture, no border of its own |
   | `interface/t_border.bmp` | 88x108 | ornate gold frame with worked corners, on a keyed black field |
   | `interface/t_border.256` | 88x108, 1 frame | the same frame as a sprite |
   | `interface/radiob.256` | 6 frames: 24x24, 24x24, 24x24, 24x24, 16x16, 16x16 | gem toggles, a dark and a light of each shape and size: round, square, small square |

   The three together are a panel fill, a frame and an off/on toggle gem, which is what the owner's
   photographs of the original show the tip window to be. **That pairing is inference, not a read
   fact**: no claim names these nodes, and this tree does not treat a filename as evidence. The lane
   verifies it the only way available — compose the panel from them and compare against the
   photographs — and records the result either way. The panel is larger than 160x240 on every screen,
   so the fill tiles and the frame is used as a nine-patch; both are the lane's to establish from the
   art, and neither is researched.
2. **Each of the five screens shows its own text on entry.** Town square `town.txt`, shop
   `shop1.txt`, school `training.txt`, tavern `inn.txt`, generator `chrgen1f.txt` or
   `chrgen1m.txt` by the hero's class. A screen whose node the install does not ship draws no panel,
   matching every other install-text reader in this tree.
3. **The close control dismisses the panel for the visit.** Re-entering the screen in the same
   session does not bring it back. This is per-screen: closing the shop's tip does not close the
   school's.
4. **The toggle suppresses every tip permanently.** Turning it off stops every screen's tip, and the
   state is written and read back at startup. A missing store means tips are on, which is the
   original's own default (`TOWN-186`). The original writes the value `TipsMode` under
   `HKEY_LOCAL_MACHINE\SOFTWARE\1C\Allods`, value name and key path both from `TOWN-186`.
   **This build does not write the Windows Registry**: it is a Go program that must build and run
   where there is none. The value goes to a store beside `saves/`, resolved by `DefaultSaveDir`'s own
   rule, and the store is shaped to hold the other twelve options `TOWN-186` names rather than this
   one alone. That is a `DEVIATION` row and not an `UNKNOWN` one: the original's location is read,
   and this build departs from it deliberately.
5. **The shop's existing floor-drawn tip is replaced by the widget.** `pkg/ui/shopscreen.go` stops
   drawing bare text at `shopTipRect` and shows the panel instead. `DIV-133`'s clip against
   `shopMessageRect` is re-read against the panel's own geometry: the reason for the clip was
   bleed-through between unbordered text and the message strip, and a bordered panel may not need it.

Five behaviours, one mechanism over five screens plus one store. This is at the ceiling a contract
may state without splitting and it does not split: behaviours 2, 3 and 5 are the same widget over
different screens, and behaviour 4 is the only part with a second domain in it.

## Research

**This section is an index, not evidence.** It names which claim answers which question and nothing
more. The lane reads each row whole before building on it, from the pin, with
`cd research && go run ./tools/claim <ID>`. A summary here is a selection, and a selection drops the
sentence the reader needed: story `1017` shipped ornate empty plaques because four accurate bullets
about blits omitted the same row's next sentence about labels. Do not build from this list. Do not
quote this list in `spec.md`.

`EXP-0197` merged at research `eb92f53` and the pin carries it as of `1017`'s landing on 2026-08-19.
It answered four of this contract's five original UNKNOWNs.

| question | claim |
|---|---|
| what `R1261`'s six operands are, and what `0x467` is | `TOWN-184` (High) |
| what controls the popup builds, their ids, captions and rects | `TOWN-185` |
| the enable flag's writer, its default, and whether it survives the process | `TOWN-186` (High) |
| the generator's own call site and its second-text latch | `TOWN-187` |
| whether the flag is one global or one per view | `TOWN-188` (High) |
| the town's, tavern's, school's and shop's own call sites | `TOWN-165`, `TOWN-015`, `TOWN-021`, `SHOP-TIP-045` |
| that the mechanism is uniform across the three town views | `TOWN-025` |
| the text reader's own behaviour | `MISSION-DOC-021` |
| the wrap pass and the line pitch | `TEXT-API-007`, `DLG-WRAP-009` |

**Two of these rows need reading with particular care, for reasons already established:**

1. `TOWN-021` grades the school's own text-file operand **Medium** and says so in its own words: the
   operand was not observed at the school's call site. `TOWN-188` discharges that caveat. Read both.
2. `TOWN-185` names the instrument it lacks. This repository has no Win32 `STRINGTABLE` decoder, so
   the two control captions, string-table ids `0x7f` and `0x80`, were not resolved to text, and
   which of controls `0xd` and `0xe` is the close control was not confirmed. What those captions say
   is authored here from the owner's own photographs of the original, recorded under **The shipped
   texts** above. That is evidence for the implementation only; B1 keeps it out of any research
   brief.

**Still open, and each an authored decision here rather than a hold:**

- What raises the second text on the shop and on the generator. `TOWN-187` reads the generator's
  second call site, `R1986`, and its once-only latch at `+0x100`, but not what calls it;
  `SHOP-TIP-045` is in the same position for the shop. Neither second text is shown by this story,
  so nothing is guessed.
- How the popup is painted. `TOWN-185` gives the three child controls' ids, sizes and rectangles and
  nothing about how any of them is drawn.

The registry subkey is **no longer open**: the same correction commit resolved it. `TOWN-186` now
reads both routines' own single callers, `R1984` and `R1985`, each of which calls
`RegOpenKeyExA((HKEY)0x80000002, "SOFTWARE\1C\Allods", …)`, and `0x80000002` is
`HKEY_LOCAL_MACHINE`. Behaviour 4 above states that path and it is now sourced rather than assumed.

**Owner directive outweighs research absence.** The story proceeds on the behaviour the owner
described. Each gap above becomes a typed row, never a hold.

## Twelve aspects

| aspect | applies | why |
|---|---|---|
| data | yes | eleven text nodes; six read |
| runtime state | yes | per-screen dismissal for the session |
| simulation | no | nothing reaches `pkg/sim` |
| player input | yes | the close control and the toggle |
| AI | no | — |
| UI/HUD | yes | the panel itself, on five screens |
| triggers/scripts | no | — |
| inventory/equipment | no | — |
| persistence/save-load | yes | the suppression store |
| campaign/session | no | the panel is per-screen, not per-campaign |
| shipped content | yes | six of the eleven shipped texts on both roots |
| interactions with existing mechanics | yes | the shop's existing tip and its message strip |

## Domains

**Client** (8) for behaviours 1, 2, 3 and 5. **Persistence** (9) for behaviour 4. Two domains, no
hashed simulation state.

## Out of scope

- **`shop2.txt` and `chrgen2.txt`.** Both are second texts installed into an existing popup rather
  than into a new one, on a once-only latch (`SHOP-TIP-045`, `TOWN-187`). Neither row names the user
  action that fires the latch, which is `DIV-132`'s open half. A guessed trigger wires a user action
  to the wrong effect. Both stay unread and the row stays open.
- **The three `chrsel*.txt` texts.** They belong to two character-select screens, which `TOWN-188`
  confirms exist as call sites of the same reader and explicitly did not read. This build draws that
  stage through `preControlRegion`. The owner named the skills text, which is `chrgen1*`.
- **The nine battle tips** under `text/battle/m10/` and `text/battle/m20/`. `TOWN-188` confirms a
  per-mission battle-tip call site of the same reader and did not read it. In-mission content is not
  one of the five screens.
- **The generator's window layout beyond the tip.** The owner also asked that the generator window be
  "более унифицированным с другими окнами" and, on 2026-08-19, that the generator be rebuilt because
  "композиция немного неверная у страницы". `1017` gives it the same shipped button frame the other
  three rooms have and this story gives it its floating tip, which is the part he named first. The
  page composition itself is a separate story, and its material was measured at this seat the day he
  asked, so that nothing here is lost:

  - **`interface/chrgen/centerarea.bmp` is 320x480 and this build does not load it.** That is the
    detailed page's whole middle column, x 160..480. `pkg/game/chargenassets.go` reads
    `interface/chrgen/precreate/`, the per-class `column.bmp` and `mask.bmp`, and the skill bitmaps,
    and never this node. The tavern's own `interface/inn/centerarea.bmp`, the same 320x480 shape, is
    loaded and drawn by `pkg/game/towntavernart.go`.
  - **`interface/inn/luover.bmp`, `ruover.bmp` and `ldover.bmp` are 16x238, 16x238 and 16x242, and
    none is loaded.** Rendered at this seat, all three are green stone trim with gold banding, the
    same material as the panel frames. Their widths and heights are those of the seam bands between
    the screen's three columns. Where each one goes is not established: the filenames suggest it and
    filenames are not evidence in this tree.
  - **`interface/inn/lbuttonoff.bmp`, `lbuttonon.bmp`, `rbuttonoff.bmp` and `rbuttonon.bmp` are 32x28
    off/on pairs and none is loaded.** This build draws the character strip's previous and next
    controls as authored boxes.
  - `manback.bmp` and `manbacktalk.bmp` (48x64), `tav_09.bmp` (16x242), and the `tender/`, `candle/`
    and `cauldron/` frame sequences are also unread.
- **Sound, difficulty and every other candidate preference.** Behaviour 4 builds one store with one
  value in it. It does not become a settings screen.

## Expected divergence rows

Allocated `DIV-160` through `DIV-164`; the allocator returned `DIV-160` as next free on 2026-08-19,
found at `PIPELINE-STATUS.md:60`, which is `1017`'s own reservation.

**These five ids are this story's and they are not re-derived.** `1017` landed later the same day and
wrote `DIV-165` and `DIV-166`, so `pipeline/next-div-id.sh` now returns `DIV-167` and the ledger's
highest row is `DIV-166`. `DIV-160`..`DIV-164` have no row and are reserved for this story at
`PIPELINE-STATUS.md`; running the allocator again here would allocate a sixth range and leave a hole.

Expected subjects, revised after `EXP-0197` and after reading its rows whole at the pin: the
suppression store, a `DEVIATION` because `TOWN-186` reads the original's registry location in full
and this build departs from it deliberately; the close control's identity and both captions, which
no instrument in the research repository can currently read (`TOWN-185`); the panel's own drawn
appearance, since `TOWN-185` gives the three controls' rectangles and nothing about how any of them
is painted; and `DIV-133`'s clip if the bordered panel changes the reason for it. **The generator's
class discriminant is no longer among them**: `TOWN-187`'s correction landed before this pin and
agrees with what this story builds. If the story
finds fewer than five distinct facts, the unused ids are **returned and retired, never reused** —
`pipeline/next-div-id.sh` counts mentions rather than rows, so a returned id is already spoken for by
this paragraph.

`DIV-154` is expected to close: it is the town square's missing tip widget, which behaviour 2 builds.
`DIV-132`'s first half, the always-open gate, is expected to narrow rather than close, because
behaviour 4 gives the build a gate of its own without establishing that it is the original's.

## Review ceiling

**Three passes.** Five behaviours, two domains, no hashed simulation state. Reaching the ceiling is a
scoping diagnosis: land what works, open the remainder as its own defect story or hotfix, and record
in `closure.md` that the story was cut too large.
