# 0151 — layers and names

**Intensity:** spec-first / static. **Terrain:** brownfield in figure composition, moving-depth
ordering, inventory UI and equipment state; greenfield for the item-name parser and popup
composition. **Threshold:** High for unequip because it changes hashed simulation state; Medium for
the display-only paths.

## Problem

Equipment layers were painted in slot-number order, paired pieces showed one side, and a moving unit
could pass behind the sack or corpse it was crossing. Item pickup used a code's digits when the game
carried a stored name. The inventory had no characteristics popup and no route for taking a worn item
off. Russian stored names reached a byte-indexed font in the wrong representation.

## Terms

- **Equipment slot**: one of twelve numbered fields. Slot 1 holds the weapon or staff and slot 2 the
  shield. A **held layer** is either of those two layers.
- **Primary layer**: an occupied slot's ordinary figure sheet. A **secondary layer** is the opposite
  side shipped for paired equipment in slots 4, 8, 9 and 10.
- **Body name**: the line selected by slot 1's item row from the figure body-name list.
- **Crossing**: the interval in which a unit is interpolated between an origin cell and its current
  simulation cell, the destination.
- **Stored item name**: a text line paired positionally with a little-endian `u16` packed item-code
  key. The line consists of the bytes in the selected install, not UTF-8 text.
- **Seven-digit code**: for material `M`, class `C`, shape `S` and row `R`, decimal
  `%02d%02d%1d%02d` (`M,C,S,R`), except class 14 uses `%02d%02d%03d` (`M,C,R`).
- **Damage base and spread**: the weapon roll is `base + U[0, spread]`; its visible bounds are
  therefore `base` and `base + spread`. The two fields are not a minimum and a maximum.

## Functional requirements

**FR-1 — Figure layers follow one explicit order.**

Both the map figure and the inventory doll MUST paint the base figure first and then each available
layer for every occupied slot. The primary slot order MUST be:

`12, 11, 7, 4, 5, 9, 10, 8, 6, 3`, followed by slots 1 and 2.

The order of the held slots is conditional. If slot 1 resolves to body name `bowman`, `archer`,
`xbowman`, `axeman2h`, `swordsman2h` or `mage_st`, slot 2 MUST paint next and slot 1 MUST paint last.
For every other body name, an empty slot 1, or a body row that cannot be resolved, slot 1 MUST paint
next and slot 2 MUST paint last. An unoccupied slot contributes no layer and does not change the
order of occupied slots.

For occupied slots 4, 8, 9 and 10, an available secondary layer MUST paint immediately after that
slot's primary layer. No other slot MUST request a secondary layer. A missing primary or secondary
sheet MUST be skipped without preventing the rest of the figure from being painted. If the base
figure is unavailable, no composed figure is produced. The same order applies to fighter and mage
figures. Body-name matching is exact and case-sensitive over the complete stored line.

**FR-2 — A crossing unit keeps the later depth row.**

For the whole crossing, a moving unit's depth row MUST be the greater row number of its origin and
destination cells. A northward crossing therefore keeps the origin row; a southward crossing uses
the destination row; a sideways crossing uses the common row. When that row ties a sack or a corpse,
the sack or corpse MUST paint first and the living unit second. A stationary unit's depth rule MUST
not change.

**FR-3 — Item names come from the selected install's stored bytes.**

The item-name key stream MUST be read as consecutive little-endian `u16` values and paired by index
with lines from the item-name text stream. CRLF and bare LF MUST be accepted. A trailing newline
MUST NOT add a line. Pairing MUST stop at the shorter stream; an odd final key byte MUST be ignored;
an empty line MUST produce no stored name. If a key occurs more than once, the last paired non-empty
line MUST win.

An item's raw packed code MUST be looked up in that table before any composed fallback. A stored
line MUST be used unchanged, byte for byte. If no stored line exists, a code that resolves as a
weapon MUST use that weapon's existing name; a code that resolves through neither route MUST show
its seven-digit code rather than disappear. Failure to read either table stream MUST not prevent a
mission from opening.

The drawing path MUST apply the selector encoded by the trailing ASCII digit of `main/id`. Selector
1 maps each source byte in `0x80..0xAF` upward by `0x30` and each in `0xE0..0xEF` upward by `0x10`;
every other byte is unchanged. Other selectors leave every byte unchanged. The font MUST receive one
shipped byte per stored-name byte before that conversion; the item-name parser MUST NOT transcode a
line to UTF-8 or apply the conversion early.

**FR-4 — Hovering an inventory item shows a framed popup.**

While the cursor is over an occupied pack cell or worn slot, a popup MUST state the item's name. A
resolvable weapon MUST add `Damage <base>-<base+spread>` and `To-hit <value>  Defence <value>` lines.
A resolvable armour MUST add `Defence <value>  Absorption <value>`. A shield or class-14 magic item
MUST show its name alone because neither has a decoded characteristics route in this build.

The popup MUST use the inventory's fill and border colours, a one-pixel border, and three pixels of
clearance between border and glyph area. Text MUST use the existing face-and-shadow style. The popup
MUST begin fourteen pixels right and below the cursor when space permits and MUST paint after every
other screen element. On each axis, it MUST move back to fit when it is no larger than the view; if
it is larger than the view, its coordinate MUST be zero and overflow is allowed. No popup MUST
appear over an empty cell, outside both item grids, before a cursor has been observed, or when no
usable font is available. Hovering MUST issue no command and MUST change no click state.

**FR-5 — A worn item can return to the pack.**

Two primary presses on the same worn slot within the inventory double-click window MUST raise one
one-shot unequip request. The worn and pack double-click windows MUST be independent: a press in one
MUST neither complete, spend nor extend the other. Presses on different worn slots MUST restart the
worn-slot match rather than unequip either.

An accepted request MUST move the code occupying the selected slot when the deterministic step runs
into that entity's carried container as one item and clear only that slot. If the container already
has the same code, its count MUST increase; otherwise the item MUST be appended. The result shown
after that step MUST carry recomputed equipment-derived combat values and refreshed worn figure,
worn popup text, pack icons, counts and popup text.

The accepted boundary includes the last frame of the window. A successful pair consumes the pending
match; a later press starts a new pair. A press outside both item grids decrements time normally but
neither completes nor relocates either grid's match.

A request for an absent entity, a slot outside 1 through 12, or an already empty slot MUST make no
state change and produces no separate diagnostic. A geometric double-click on an empty worn cell may
raise a request; the deterministic step is the boundary that refuses it.

**FR-6 — Existing state and display boundaries remain intact.**

No serialized field or byte-form version MUST be added. Unequip is an ordinary transition over the
existing equipment and carried-container fields. Item names and popup text MUST remain outside
simulation state. Figure composition, popup rendering and moving-depth selection MUST not alter the
world digest. The pickup log MUST retain its bare background; only the hover popup gains a frame.

## Acceptance criteria

| Id | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | A figure with armour in slots 7 and 12 and a held item in slot 1 | The map figure and inventory doll are composed | Both use the declared slot order and the held layer paints over the armour. |
| **AC-2** | unit | Both held slots are occupied | Slot 1 resolves once to a listed two-handed/ranged body and once to another or unresolved body | Slot 1 paints last in the first case and slot 2 paints last in the second. |
| **AC-3** | unit | A paired slot and an unpaired slot each have primary and secondary fixture sheets | Each figure is composed | The paired slot paints both sheets; the unpaired slot does not request or paint the secondary sheet. |
| **AC-4** | UI | A living unit crosses a sack or corpse northward, southward and sideways | Plane depth order is built during the crossing | The sack or corpse precedes the living unit in every direction; the stationary tie remains unchanged. |
| **AC-5** | unit | Synthetic key and line streams contain stored names, CRLF/LF boundaries, an empty line, unequal lengths, an odd final key byte and a duplicate key | The table is parsed | Keys pair positionally up to the shorter stream, the odd byte and empty line are absent, the duplicate's last non-empty line wins, and every retained line has its original bytes. |
| **AC-6** | integration | A stored name exists for a weapon or armour code | Pickup text and inventory popup text are built | The stored line wins over weapon recovery and over the seven-digit fallback. |
| **AC-7** | UI | The cursor hovers a worn weapon, worn armour, shield, magic item and carried item | The inventory is drawn | Each gets a framed name popup; only weapon and armour receive the specified decoded lines, and weapon damage is a range. |
| **AC-8** | UI, error | The cursor has not been observed, names no occupied item cell, or the viewer has no usable font | Popup presentation is requested | No popup is produced and no command or click state changes. |
| **AC-9** | unit, error | Either item-name stream is absent, one stream is short, or a code has no stored line | Definitions and a display name are requested | Mission loading remains available; parsing stays in bounds; naming follows the fallback chain. |
| **AC-10** | UI + simulation | The same occupied worn slot is pressed twice within the double-click window | The next deterministic step runs | Exactly that item moves to the pack, the slot clears, combat values are recomputed and both inventory surfaces refresh. |
| **AC-11** | simulation, error | An unequip command names an absent entity, invalid slot or empty slot | One step runs | The whole canonical byte form and digest equal a quiet step from the same state. |
| **AC-12** | simulation | A successful unequip has completed | The world is marshalled, loaded and marshalled again | The bytes and digest round-trip identically without a format-version change. |

## Derived properties

- **P-1 — completeness.** For any equipment set, the declared figure order contains each slot from
  1 through 12 exactly once before occupancy filtering.
- **P-2 — invariant.** For any crossing, the selected depth row is `max(origin row, destination
  row)` for every transit frame.
- **P-3 — invariant.** For any stored item-name line, the byte sequence presented to the font before
  its existing per-byte conversion equals the line's byte sequence.
- **P-4 — negative invariant.** For any popup refusal in AC-8 or name-table failure in AC-9, no
  simulation state changes and mission availability is not reduced.
- **P-5 — negative invariant.** For any refused unequip in AC-11, every canonical world byte remains
  equal to the corresponding quiet-step byte.

## I/O examples

| Case | Observable text or order |
|---|---|
| Weapon with `DamageBase=5`, `DamageSpread=3`, `ToHit=2`, `Defence=1` | `Damage 5-8`; `To-hit 2  Defence 1` |
| Armour with defence 4 and absorption 2 | `Defence 4  Absorption 2` |
| Shield or class-14 magic item | Stored name only |
| One-handed slot-1 body | Armour order, then slot 1, then slot 2 |
| `swordsman2h` slot-1 body | Armour order, then slot 2, then slot 1 |

## Constraints

| Choice | Alternative | Observable consequence |
|---|---|---|
| Preserve stored item-name bytes until the font path | Decode to UTF-8 at load time | Preserving supplies one source byte per glyph lookup on both language selectors; decoding supplies multiple bytes for Russian letters and draws the wrong glyphs. |
| Refuse invalid unequip as a no-op | Reject outside the deterministic step or mutate partly | A no-op replays identically and leaves the canonical byte form unchanged. |
| Use only decoded popup characteristics | Infer shield or magic-item numbers | The popup is incomplete for those classes but never presents invented values. |

## Out of scope

- A faithful mage-specific head/equipment interleave or the original's invisible mage slot-9 rule.
- Defining what an item row beyond the body-name list means.
- Decoding shield characteristics or class-14 magic-item characteristics.
- Lifting the font path's byte-wide glyph limit or changing the selected install's encoding.
- Drag-to-unequip, clicks on the doll picture, dropping an item on the ground, or container capacity.
- Reframing the pickup log.

## Verification mapping

| Acceptance | Method |
|---|---|
| AC-1, AC-2, AC-3, AC-5, AC-6, AC-9, AC-11, AC-12 | CI-automatable unit/integration tests with synthetic fixtures. |
| AC-4, AC-7, AC-8, AC-10 | CI-automatable UI and game-seam tests with synthetic fixtures. |
| Russian appearance under the selected install | Manual run through the shipped font path; no game data enters tests. |

Every FR is covered: FR-1 by AC-1–AC-3; FR-2 by AC-4; FR-3 by AC-5, AC-6 and AC-9; FR-4 by AC-7
and AC-8; FR-5 by AC-10–AC-12; FR-6 by AC-8, AC-11 and AC-12. Every error case yields a negative
invariant in P-4 or P-5.
