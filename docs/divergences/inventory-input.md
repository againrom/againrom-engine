# Divergences -- Inventory input

1 row(s). Part of the split described in `docs/DIVERGENCES.md`.

## Authored where research is silent

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|
| DIV-2940 | inventory / same-cell activation and the worn box double click | One detector for every screen; keep the mission pack and worn box actions and same-cell rule | `MENU-143` (High / Medium) gives the system double-click message, time and rectangle. `ITEM-USE-112` (High for the physical handlers) gives the mission-grid double click selecting its hit item and calling the character-panel action. No claim in the 34 pinned claim files searched states a same-cell restriction or a double-click action on the separate worn box; `TOWN-349` describes the character figure instead | Both boxes consume the one detector's second press (`DIV-2871`) only on the first press's cell. A pack tap raises its item action on the second release after drag arbitration; a worn-box match raises its slot action on the second press. Another primary or secondary press clears the previous cell identity. The 30-frame windows are removed. The worn box lies below the shipped frame (`DIV-316`); its installed-data input witness uses an explicit 992-line headless fixture | UNKNOWN | The shared timing follows MENU-143 and the pack action has ITEM-USE-112 authority; the same-cell restriction, pack release arbitration and separate worn-box action remain owner-directed policies beyond those claims | Claims for mission-grid first/second press arbitration, its target identity rule and the separate worn-box double-click action | OPEN |
