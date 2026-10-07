# Cursor rules by spell kind

## Intent and authority

Result: the cursor over the game area follows the armed order and, for a
spell, the spell's kind. Over every panel the arrow stands.

Authority: owner observations of the original, items 4, 5, 10, 11, 13 and 14
of the space and panels observation set. Pinned claims `AI-CURSOR-177`
(right-up cancel closes the popup when Cast is armed), `MENU-054`,
`MENU-055`, `MENU-064`, `MENU-065`, `AI-QUICKINVOKE-279`, `SESS-VIEW-028`
(open panel state). No claim states a cursor rule per spell kind, and none
names a self spell. Where an owner observation differs from `MAGIC-REACH-181`
the observation is built (DIV-2274).

## As built

- Item 5, game area only. `gameAreaAt` (`pkg/ui/cursor.go`) is the surface
  rectangle minus the pack, book, command panel, minimap and unit panel
  boxes. The ordinary hover cascade already answered the arrow there; the
  attack picture did not and showed over every panel. `attackShown` now
  asks `gameAreaAt`. Order-mode cursors read it through the same hover path.
- Item 10, book closed, no hook. A chosen spell is live only while the book
  shows, so the mode is none and no magic cursor draws. Unchanged code, now
  pinned.
- Item 11, kind. `castCursorAt` (`pkg/ui/castcursor.go`) replaces the old
  point-spell Move arm:
  - target: cast cursor over a unit the party sees, else the ordinary
    cursor; the click is made under the drawn cursor, so a ground click moves;
  - area: cast cursor on every cell, seen or not, and the click casts there;
  - self: cast cursor over a selected unit that holds the spell.
  An item cast keeps the cast cursor over the whole game area.
- Kinds come from the installed row (`spellSelfOnly`, `pkg/game/spell.go`;
  DIV-2275). Self is Shield alone on both installs; area is the 10 area rows
  and Teleport; target is the other 16.
- Item 14, hook survives the book. Unchanged logic, now pinned for each kind:
  after C and closing the book by hand the hook stays armed and the cursor
  follows the kind.
- Item 13, hook spent. A cast spends the hook as before. A cast click that
  issues no cast order (a miscast) now spends it too (DIV-2276).
- `AI-CURSOR-177`. `cancelMapCommand` closes the book whenever Cast is armed,
  whoever opened it. With nothing armed the right click still deselects and
  leaves the book as it is.

## Proof

Viewer tests through `App.HeadlessKey` and `HeadlessPointer`
(`pkg/ui/castcursor_test.go`, `pkg/ui/cursorarea_test.go`):

- `TestOrderModeAndOrdinaryCursorsShowOnlyOverTheGameArea`: ordinary, attack,
  move, defend, patrol, swarm and cast over ground and over five non-game
  places.
- `TestCursorByKindWithTheBookOpenByHand`,
  `TestSelfSpellCastsOnlyOnTheCasterAndTargetSpellOnlyOnAUnit`,
  `TestAreaCursorStandsOverUnseenGroundAndTargetCursorNeedsASeenUnit`.
- `TestChosenSpellIsNeverAMagicCursorWithTheBookClosedAndNoHook` (control:
  the hook makes the same pointer a magic cursor).
- `TestCastHookKeepsItsCursorByKindAfterTheBookCloses`,
  `TestCastHookIsSpentByACastAndByAMiscast`,
  `TestRightClickCancelClosesTheBookWhenCastIsArmedWhoeverOpenedIt`.

Release test, EN and RU: `TestReleaseCastCursorFollowsTheSpellKind` opens
mission 10 with two installed-rule mages and reads the cursor manager
through the pointer route for Fire Arrow, Fire Ball, Teleport and Shield: book
open, book closed, C hook with the book open and closed, then the cast click
and the spent hook.

Loss controls: with the `attackShown` gate reverted the attack case fails
("attack over the pack bar: cursor attack"); with every kind answering the
area rule the kind tests fail; with the cancel close limited to a C-opened
book, and with the miscast arm removed, their tests fail; with
`spellSelfOnly` returning false Shield shows the cast cursor over the friend
on EN.

Existing tests changed to the owner rule: a target spell on open ground no
longer casts (`TestTheSpellSelectionStaysArmedForRepeatedCasts`,
`TestACastAsksNoVoice`), an area spell casts over a cell with no visible
corner (`TestPointSpellClickCastsWhateverTheFogOverTheCell`), and a cancel
closes the book (`TestBookPointSpellKeepsClickedGroundUnderCreature` reopens it
between rounds).

## Open debt

- Heal, Invisibility, the Protections, Bless and Haste are target spells here;
  the original may treat some as self (DIV-2275).
- A cast the world refuses later (mana, range) is not seen by the viewer and
  does not spend the hook (DIV-2276).
- Guard, Stand Ground and Retreat issue their order at once and arm no
  cursor in this build; the owner's item 4 lists them among the specific
  cursors.
- A self spell with several selected casters casts each on the clicked one.
