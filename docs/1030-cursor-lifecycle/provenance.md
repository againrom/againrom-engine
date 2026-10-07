# Story `1030` — provenance

Research pin at implementation: `21e760a`. Every row below was read whole with
`go run ./tools/claim <ID>` from `research/` on 2026-08-22, not summarized from `contract.md`.

| ID | Grade | What it establishes | Where used in `spec.md` |
|---|---|---|---|
| `SPR16A-CURSOR-067` | High | The 28-slot order and names | B1 table, slot column |
| `SPR16A-CURSOR-046` | High for paths/hotspots/periods | 23 `.16a` registrations' art, hotspot, period argument | B1 table, rows 0–17, 23–27 |
| `SPR256-CURSOR-046` | High | 5 `.256` registrations' art, hotspot, period argument | B1 table, rows 18–22 |
| `AI-CURSOR-172` | High for the mechanism, Medium for whether observed past index 1 | One manager, one current cursor, the `timeGetTime` counter, the wrap | B2 (idempotence's own SetCursor shape), B3 (Advance) |
| `SPR16A-CURSOR-061` | High for the mechanism and the wrap, G2 | The wrap read from the sheet side; frame count/period as independent executable constants | B3 (Advance's wrap), G2 |
| `AI-CURSOR-193` | High, resting on a read-only scan | No path outside the set-cursor adapter clears the current cursor; the idempotence guard | B2 |
| `TOWN-372` | High for the pattern, Medium for which surface each routine is | 17 surface-transition routines: 11 set a wait/exit pair (10 `default`, 1 `select`), 1 sets `default` only, 5 set none. Nine are identified by a pushed asset path, four of them the town family (shop, inn, school, town square) | B4 |
| `MENU-CURSOR-046` | High for the sets, Medium for surface identification | The main menu's own `wait`/`select` pair | B4 (main menu row) |
| `AI-CURSOR-196` | Medium | The `town` slot (24) is selected only by mission-view routines, condition undecoded | Cut-from-scope note; not built in this story |

`research/experiments/EXP-0190-combat-controls/evidence/cursor-construction.tsv` is the transcription
source for B1's 28-row table: the complete join of name, path, format, hotspot x/y, period argument,
frame count and frame dimensions this story cites the claim rows above for. Only the claim rows are
cited in `spec.md`; this path is not part of the spec's own authority.

## Reading discipline followed

Every row in the table above was read through the pinned reader, not through `contract.md`'s own
one-line paraphrase of it — `contract.md` itself records that four of its own sentences did not
survive a whole re-read of these same fifteen rows, and this story does not repeat that shortcut. No
row's confidence grade was rounded up in `spec.md`; the two Medium caps this story's own behaviour
touches (`AI-CURSOR-172` and `SPR16A-CURSOR-061` on whether animation is observed past index 1) are
carried into `closure.md` as open items and into a divergence row, not resolved by assertion.
