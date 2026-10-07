# Divergences -- Mission screen (UI/rendering split)

2 row(s). Part of the split described in `docs/DIVERGENCES.md`: read that file first for what a row means, what each column holds, and how a row is found.

## Authored where research is silent

No claim answers the question, so the implementation authored one.

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|
| DIV-487 | mission UI / selected-hire portrait and inventory eligibility | The owner requests visible mercenary portraits. The earlier NPC map-sprite and read-only inventory boundaries remain. | `MERC-TYPE-001` establishes human mercenary types 3 through 15. `UNIT-PICT-035` places their low class IDs on the composed-figure path. No promoted claim establishes the selected-hire inventory eligibility rule. | `partyFigures` supplies a Human hire its saved figure and face identity; `unitPicture` composes its portrait with live worn equipment. Siege hires keep their class picture. NPC map art is unchanged, and `switchInventorySubject` still refuses hired members. | UNKNOWN | The missing Human portrait is fixed. Read-only inventory remains owner policy; the generic class-picture claim does not close that selected-hire eligibility question. | A claim establishing the original inventory eligibility of a selected hired mercenary. | OPEN |

