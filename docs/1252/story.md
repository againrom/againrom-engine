# Town refusal lines and sounds

## Intent and authority

After Sleep the tavern posted `the shop is restocked`, and training and squad
refusals posted authored lines. `TAVERN-LINES-022` (High for the local hire and
training refusal branches, Medium for the absence of room message posts and of
a Sleep action) states: the hire refusal arm requests tavern sound slot `+0xa0`
and posts no line; the two training refusal tests return through one exit that
posts no line and requests no sound; no Sleep action was found in the inspected
tavern handlers; indirect message-poster callers are Unknown. The refusal
condition for a hire is a price above the money left (`TAVERN-CLICK-019`).
Pin: knowledge snapshot k106.

## As-built behaviour

- Sleep restocks the shop and posts no line, on success and on refusal.
  The control is the owner's addition and is unchanged otherwise.
- Train posts a line only for a completed training. No selected skill, a price
  above the purse and every other refusal post nothing and change nothing.
- Hire refused for a price above the money left posts no line and records one
  request of tavern slot `0xa0` (`townScreen.tavernSlotRequests`), by the
  button and by a double click. No file is bound to the slot, so nothing plays.
  Other hire or return refusals post no line and request no slot.
- Hire and return success keep their authored lines and request no slot.

## Proof

- `TestTownRefusalsPostNoLineAndHireRefusalRequestsItsSlot` (synthetic
  fixture): lines and slot requests for hire past the purse by button and
  double click, hire within the purse, and Train with no selection, past the
  purse and within it.
- `TestReleaseTownRefusalsPostNoLineOnInstall` (installed EN and RU roots):
  Sleep, hire with an empty purse and Train one gold below the price through
  the production tavern and school, checking lines, slot requests, roster,
  purse and skill level.
- Existing assertions that expected the removed lines are updated.

## Open debt

`DIV-1647` to `DIV-1650` (town ledger) carry the remaining Unknowns: the
Sleep action and its indirect poster, the file bound to slots `+0x98`, `+0x9c`
and `+0xa0`, and whether hire, return and completed training post a line.
`DIV-133` notes that the message strip is the engine's own affordance.
