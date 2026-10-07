# Provenance — the campaign mission start

Intensity: **spec-anchored / static**. Terrain: **brownfield** in `pkg/mapload` (the placement
arms already ship and this story changes what they answer), **greenfield** for the campaign
entry, the npc table and the drop reader.

## Backing

| Spec anchor | Claim | Confidence, and what carries it |
|---|---|---|
| FR-1 — a mission number names `scenario/<n>.alm` | `SESS-MAP-010`, `SESS-LOAD-009`, `RES-IDENT-034` | High for the container, the key and the name form — one routine read whole, the `"Scenario\"` prefix and the `"%d.alm"` name are its own literals, and the leading-segment dispatch is the archive set's own rule |
| FR-3 — the four arms, ordered, class key outermost | `MISSION-ARM-006` | High — the nesting is `R0151`'s own, read at instruction level, and it discriminates against the standing reading on named records rather than by corpus fit |
| FR-3 — the two search key columns, byte-truncated | `ALM-CLS-038` (as amended by EXP-0049) | High — `typeID`/`face` (`0x1d`/`0x1e`) for Units, `typeID` (`0x10`) and `serverID` (`0x18`) for Humans |
| FR-4 — the npc arm's route, and the `26` sentinel | `MISSION-ARM-006`, `MISSION-DEF-007`, `REG-NPC-058` | High for the route (`npc+0x14` fetched by two named routines) and for the search's direction, its skipped index 0 and its miss value / **Medium** for what the sentinel *means*, and its arithmetic is Unknown |
| FR-5 — the drop node: opcode, and the two value bytes | `TRIG-DROP-013`, `TRIG-ACT-004` | High — both ends are named instructions in two routines read independently, and the opcode is shown never to reach the instant dispatch |
| FR-5 — the type-7 payload framing and the 796-byte node | `ALM-TRIG-044`, `ALM-TRIG-045` | **Medium** — the three-array framing is corpus-only (exact on 38/38, sole survivor of a 150 000-candidate sweep); the record's own field map is the loader's read sequence, and its `+0x4c`/`+0x50` values land at the runtime `+0x48`/`+0x4c` that `TRIG-DROP-013` reads, which is a cross-check the two could have failed |
| FR-6 — the uniform pick, and the 30..100 inclusive fallback | `MISSION-DROP-002` | High — the index, the override gate and the generator's inclusive bound are named instructions; the 38/38 one-node census is exhaustive over the corpus |
| FR-7 — the party is positioned, not placed by the map | `MISSION-START-001` | High — slot 1 owns 0 of 35 type-6 records on `10.alm`, the routine iterates the player's own unit list, and the hero goes to the drop cell at radius 0 |
| AC-2 — the arm census | `MISSION-ARM-006` (6672 / 1405 / 15 / 2 over 8094) | High — exhaustive over both corpora; this story's tool re-derives it independently |
| AC-4 — `10.alm`'s drop cell `(17,66)` and its 19/14/2/0 split | `MISSION-M10-009` | High — the whole type-7 payload is consumed exactly and every parameter resolves |

## Ours by choice

| Decision | Why it is ours |
|---|---|
| The generator behind the pick and the fallback, and the seed it starts from | Nothing recovers the game's `rand()`; `SHOP-RNG-008` shows the engine seeds from a clock, which is the one thing a reproducible loader must not do. The *interface* is the engine's — a uniform draw inclusive of its bound — the bits are ours |
| Where each party member past the first stands | `MISSION-START-001` gives a radius `ftol(max(5.0, sqrt(nUnits) + [L03914]))`, whose global is unread, and `R1145`, which consumes it, is not decoded. The anchor cell is the claim's; the outward walk is ours, and it is deterministic where the original may not be |
| A party member's class key, health and speed | There is no chargen in this tree and the humans arm yields no stat block (0049), so a party member carries the caller's class key and the same provisional pair every unresolved placement carries |
| Party entities follow the map's placements in the entity list | `MOVE-TICK-014` puts the party *first* at map load. Ours is the opposite, and deliberately: a placement's entity id is its record index (0019) and this story does not renumber them |

## Open / undecoded

- What `player+0x60` holds and what writes it (`MISSION-DROP-002`) — the per-player drop override.
  There is no player object in this tree, so the override is unreachable rather than skipped.
- The composition arithmetic behind `DataBinID == 26` (`MISSION-DEF-007`) — the `== 0x1a` test, the
  four `Flags` tokens and their tester are read; what turns the tokens into an id is not.
- `R1145` and the constant at `[L03914]` — the placement radius and what it does with it.
- The six remaining unresolved `DataBinID` values `42..46` (`REG-NPC-058`, `MISSION-DEF-007`).
- Whether a map may carry more than one drop node. 38 of 38 shipped maps carry exactly one, so the
  pick is degenerate on everything that ships and the multi-cell behaviour is unwitnessed.

## Disclosed divergences

- **The framing under FR-5 is Medium and reaches hashed simulation state**, through the cell the
  party stands on. The story does not take it on trust: the walk requires the three-array framing to
  tile the payload exactly and yields no cell at all when it does not, which is `ALM-TRIG-044`'s own
  discriminator asked per map at load instead of once over a corpus. A map the framing does not fit
  falls to FR-6's fallback and says so.
- **The arm correction moves worlds that already had a digest.** Every placement whose class key is
  at or above 26 and which also carries a nonzero definition id resolved, before this story, against
  the Humans collection and so reached no stat block; it now reaches its Units entry, with that
  entry's health, speed and movement domain. `docs/0049-databin-classes/verification.md` records
  `Horror.alm` as `npc 0 / server-id 474 / humans 1 / units 1340`; the corrected arms make it
  `units 1394 / npc 0 / defid 420 / typeID 1`, which is `MISSION-ARM-006`'s own figure for that map.
  Five loose maps move; the byte form does not.
- **The class-key band test is on the signed key, the search keys on the low byte.** `CMP …,0x1a`
  compares the sign-extended word the record stores; `R0495` takes both keys as bytes. The
  shipped domain is `1..80`, so nothing witnesses either edge.
- **No front-end door.** The campaign start is a library entry and a developer census tool. Nothing
  in `cmd/againrom` starts a mission, so the deliverable is not visible on a screen.

## Removed, and refused

- **`ALM-CLS-038`'s key band — `rec+0x08 < 0x40 && ∉ {0x1a, 0x1b}` selects Humans — is not built
  on.** It is `R0495`'s test, and this story implements `R0151`'s, which is `>= 0x1a`
  and decides which C++ class is constructed. The two disagree over keys `0x1c..0x3f`, and where they
  do, `MISSION-ARM-006` is the later reading, of the routine that owns the decision, and it names the
  records it changes. **This is a live disagreement between two active claims and is reported to
  research rather than resolved here**; nothing below the band is affected, and no shipped record is
  known to sit inside it.
- **Refused: reading the type-7 node array in `pkg/formats/alm`.** That tier preserves the payload
  raw by contract, the grammar is Medium, and the script runtime is another story's.
- **Refused: giving a humans-arm or npc-arm placement a stat block.** 0049's reason stands — the
  humans slot list is not published and a hero's maximum is derived, not read from a column — so this
  story resolves the npc arm to an *index* and stops there.
- **Refused: the mission-number bound.** `REG-SCN-059` bounds a campaign request at
  `10 × ScenarioMissionCount`. This story lets the archive answer instead: a number naming no entry
  fails at the read, with the address in the message.
