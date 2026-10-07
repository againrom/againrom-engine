# 0159-join-persistence — provenance

## Claims this story is built on

| Claim | Confidence | What it supplies |
|---|---|---|
| `PARTY-JOIN-025` | High | Trigger instants 19 and 22 are one ownership-change routine: the actor leaves its old group, moves to the new owner's index, and arrives in a fresh group of its own. It writes no `player+0x34` and allocates no fresh runtime id. |
| `PARTY-ENDCULL-026` | High | The end-of-mission cull keeps a surviving player's actor iff `0x21 <= word[actor+0x0e] < 0x40`, resets eight named fields, and writes no inventory field. |
| `PARTY-M20-030` | High | Mission 20 transfers Sarindar and three NPC14_1 actors whose original-save TypeIDs remain `0x17` and `0x0a`. |
| `PARTY-M20-031` | High | The Humans streamer preserves table TypeID in zero constructor mode; only the npc arm's exact `Hero` lookup requests the player-character overwrite. Both mission-end culls remove low-type actors. |
| `PARTY-M20-032` | High | The four transferred actors are absent in town; later tavern type-14 Clubmen and Reniesta are fresh stock. |
| `PARTY-PERSIST-028` | High, narrowed by EXP-0192 | Persistence is unbounded for actors admitted by the band; it is not universal to all Humans. |
| `PARTY-JOINCORPUS-029` | Medium, narrowed by EXP-0192 | Mission 40's `npc25` survives because its exact `Hero` flag selects player-character constructor mode. |
| `PARTY-GATE-013` | — | The primary-character gate tests `player+0x34`. Cross-read to establish that a joined companion is not a primary character. |
| `PARTY-INSTALL-012` | — | The full install sequence, of which the join is a subset. Cross-read for the two writes the join omits. |
| `DAT-ACT-006` | High | The Humans creation arm's own instruction for `actor+0xe`. |
| `HERO-KILL-027` | — | The death-gold roll's `> 0x40` gate, cross-read to establish that writing a Humans-band typeID cannot enable gold. |

## Confidence carried into the contract

FR-1 through FR-5 rest on High claims. The earlier `PARTY-BAND-027` census was retracted by
EXP-0192 and supplies no premise to this story. The build instead applies the decoded constructor
mode and the end-cull's exact band test to the type ID each entity actually carries.

`PARTY-JOINCORPUS-029` is Medium and is used only to name the mission the story is verified against.
No contract clause depends on it.

## Ours by choice

- **One player-character-band value rather than four.** The constructor condition is reproduced;
  within its true arm this build writes `0x21`, which every current consumer treats like the other
  values in `0x21..0x24`.
- **The reset list is not reproduced field by field.** `PARTY-ENDCULL-026` names eight resets on a
  kept actor in place. This build does not keep an actor across the boundary; it re-mints one from a
  roster member on the next map, which is the boundary shape `mapload.CarryParty` has had since the
  continuity hotfix. See spec DIV-2.
- **The joined companion's display name is his placement's own template name.** Nothing decoded says
  what name the original shows for a script-handed companion.

## Open

- Which of `0x21..0x24` a player-character-mode actor receives. Not needed by a current consumer.
- Whether the original draws a joined companion's portrait from the npc record or from his class
  record. This build resolves art through the class record, as it does for every placed unit.

## Removed from scope

- Reading a joined companion out of an owner-produced `.sav`. The loader decodes what 0144 decodes
  and no more.
- The `player+0x34` primary-character write. `PARTY-JOIN-025` establishes positively that the join
  does not perform it, so there is nothing here to implement.
