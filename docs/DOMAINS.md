# DOMAINS.md — the game's large vertical domains

Owner directive, 2026-08-15: name the game's large vertical nodes; loose coupling, high cohesion;
do not slice too finely. This file is the semantic half of coupling — which system owns which
rules and state, and what each is allowed to know of the others. The mechanical half is the
import DAG in `internal/archtest`, which stays authoritative for package-level edges.

A story's `contract.md` names the domains it touches. The closure pass and the adversarial
reviewer walk the touched domains' interfaces, not only the changed packages: a feature is
complete when every touched domain carries its share, not when one package compiles.

## The nine domains

| # | Domain | Owns | Primary packages | May know |
|---|---|---|---|---|
| 1 | **Assets** | The install's bytes as typed data: archives, registries, maps, sprites, palettes, `Data.bin` tables, text tables. | `pkg/formats/*`, `pkg/vfs`, `pkg/data` | Nothing above it. Knows no game rule. |
| 2 | **Sim Core** | The deterministic world: the tick, state and its serialization (`formatVersion`), hashing, occupancy, movement and pathing, command intake, event output. | `pkg/sim` (stdlib only) | Assets' typed data arrives through `mapload`; Sim Core never reads a file. |
| 3 | **Combat & Magic** | To-hit, damage, resistances, death; spells, casting, spell effects (point and area), projectiles as state. | inside `pkg/sim` | A cohesion region of Sim Core; shares its state, publishes events. |
| 4 | **AI & Orders** | Unit decisions: acquisition, chase and cover, group orders, escort and defend, scripted orders' per-tick behaviour. | inside `pkg/sim` | Same as Combat & Magic. Reads world state; issues the same actions a player command would. |
| 5 | **Party, Items & Heroes** | Stats, experience and training, inventory, equipment and the wear rule, spellbook, party membership and joins. | inside `pkg/sim` + `pkg/data` columns | Same region rules. Item semantics come from Assets' tables, never hardcoded. |
| 6 | **Campaign & Scripts** | Mission scripts (checks and instants), triggers, mission flow, briefings, the town return, campaign session state. | `pkg/sim` script files + `pkg/game` flow | Drives Sim Core through commands and script execution; owns what "mission" means. |
| 7 | **Town & Economy** | Shop stock generation, pricing, buy/sell, inn, school, mercenaries, the purse. | server rules in `pkg/sim` / `pkg/game` | The rules never depend on a screen; screens live in Client. |
| 8 | **Client** | Screens and input (mouse, keyboard, wheel), HUD and panels, the rendering pipeline (terrain, sprites, palettes to pixels, camera), fonts and text drawing, dialogue and message panels. | `pkg/render*`, `pkg/ui`, view parts of `pkg/game` | Renders sim state and submits commands. Never mutates sim state directly. |
| 9 | **Persistence** | Save and load of campaign plus sim state, save slots, compatibility across `formatVersion`. | `pkg/sim` binary + `pkg/game` save paths | Serializes through the sim's own writer; knows nothing of Client. |

## Coupling rules

- Assets is a leaf: everyone may read its types; it knows no one.
- Domains 3–5 (and the sim half of 6–7) are cohesion regions inside the determinism wall. They
  share the world state; the boundary between them is subject ownership, not a package border.
  A story in one region that needs another region's rule uses that rule where it lives — it does
  not copy it.
- Client is the only domain that touches the player, and the only one ebiten reaches. Everything
  it shows is sim state or Assets data; everything it does to the world is a command.
- Persistence round-trips the sim's own serialization. A field that does not survive save/load is
  a Persistence defect even when the feature that added it lives elsewhere — this is the standing
  aspect-matrix check.
- Cross-domain features (most real features) list every touched domain in `contract.md`. The
  0157 shop round is the canonical example: Town & Economy (stock, prices), Client (screen, doll,
  picker, wheel), Party & Items (whose inventory), Persistence (nothing — N/A row).

## Slicing rule

Nine is the intended count. A tenth domain earns its row only as a system with its own rules, its
own state and its own screens — not as a large feature inside an existing domain. Audio, when it
arrives, is the one anticipated candidate.
