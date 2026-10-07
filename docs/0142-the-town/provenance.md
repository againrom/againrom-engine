# 0142 — provenance

## Claims relied on

| claim | confidence | what this story takes from it |
|---|---|---|
| `REG-SCN-064` | High / Medium / Unknown | `InnMission`, `ShopMission` and `TCMission` are the offer lists of tavern, shop and school, one reader each. Shop and school take element **0** and remove it. The inn is **per NPC**: `InnMission[i]` pairs positionally with `InnNPC[i]`, and `InnMission[i] == 0` is the sentinel for an NPC with nothing to give — the zero arm speaks and queues nothing. Accepting removes the entry from **both** arrays at that index. `TC` is the school. |
| `REG-SCN-067` | High / **Medium** | `[Mission<n>] Payment` is the mission's cash reward, returned by vtable slot 1 and handed over on the mission-end arm. FR-9 rests on the **Medium** half: the getter, the picker and the call site are named instructions, but nothing was checked about sign or currency. |
| `REG-SCN-059` | High / High / Unknown | The per-mission key domains — `InnNPC` 13, `InnMission` 13, `ShopMinPrice`/`ShopMaxPrice` 13, `Payment` 9, `ShopMission` 4, `TCMission` 3 over 24 sections — which is what the reader in `campaign.go` is written to the shape of. |
| `REG-SCN-062` | High / Medium / Unknown | That a value in any of the three arrays is a mission number and that a multiple of ten is how a building hands you the **main** mission. This is why the town's chapter can be derived from the main-mission ladder at all. |

Read through `research/tools/claim` against the pin, never by opening a ledger.

## Ours by choice

- **The chapter rule** (spec FR-2). `REG-SCN-064` says which section holds which offers. Nothing
  read says which section is live during a given town visit, so "the lowest main mission the
  campaign offers that is not yet won" is authored. It is a *derivation over the file's own data*
  rather than a constant, which is the cheapest form the choice could take.
- **The gates**, and that the world map stands behind them later. The owner's own naming and his
  own design (2026-08-11). Nothing decoded is involved.
- **Everything the town says.** Every line of text in `townscreen.go` is composed by this project.
  The original's inn speech is addressed as `inn\NPC\npc%02dm%d` (`REG-SCN-064`) and this tree
  reads no speech; the words here stand in for it and resemble it in no way.
- **That a lost mission returns to the town** (spec FR-3). Nothing read says what a defeat does.
- **The four doors and their order**, and that a room is a text list.

## Open, and deliberately not opened from here

- What the original's town screen looks like and what its five-way hit test does. `EXP-0060`
  reached the three views from the campaign screen's slot table; the screen that *renders* an offer
  list was not read (`REG-SCN-062`'s own Medium).
- What is said in the inn, and by whom. The speech path is known; the text is not read.
- Whether a fourth building exists (`REG-SCN-064`'s **Medium** on exactly that).
- What `Payment` is denominated in (`REG-SCN-067`'s **Medium**).
- Mercenary hire — the inn's other half, decoded in `MERC-*` and `research/docs/status/tavern.md`,
  and deliberately built as nothing here. It is a named seam in `town.go` and out of scope.

## Data read from the lawful install

`scenario.res::scenario.reg` and `scenario.res::npc.reg` were dumped with `cmd/regtool` against
`gameversions/en` to check that the reader's shape matches the file — the counts and the per-section
values are recorded in `verification.md`. **No asset, and no converted asset, is in this repository**
(golden rule 1); the tests are synthetic and read no install (golden rule 2).

## Removed

Nothing. This story cited no claim it later dropped.
