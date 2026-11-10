# DOMAINS.md — the game's large vertical domains

Owner directive: name the game's large vertical nodes; loose coupling, high cohesion; do not
slice too finely. This file is the semantic half of coupling: which system owns which rules and
state, what each may know of the others, and where each lives. The mechanical half is the import
DAG in `internal/archtest` and `docs/ARCHITECTURE.md`, which stay authoritative for package-level
edges.

A story document names the domains it touches. The adversarial reviewer walks the touched
domains' interfaces, not only the changed packages: a feature is complete when every touched
domain carries its share, not when one package compiles.

Three packages hold most of the game: `pkg/game`, `pkg/ui` and `pkg/sim`. Inside them a
subsystem is a set of files that share a name prefix, not a package. The location tables below
name those prefixes.

## The twelve domains

Domains 1 to 9 keep the numbers earlier story documents cite.

| # | Domain | Owns | Primary packages | May know |
|---|---|---|---|---|
| 1 | **Assets** | The install's bytes as typed data: archives, registries, maps, sprites, palettes, `Data.bin` tables, text tables, sound and video streams; which game and language an install is. | `pkg/formats/*`, `pkg/vfs`, `pkg/data`, `pkg/base`, `pkg/locale`, `pkg/installinfo`, `pkg/installpath` | Nothing above it. Knows no game rule. |
| 2 | **Sim Core** | The deterministic world: the tick, state, hashing, occupancy, movement and pathing, command intake, event output, the one random source. | `pkg/sim`, `pkg/random`, `pkg/rules` | Assets' typed data arrives through `pkg/mapload`; Sim Core never reads a file. |
| 3 | **Combat & Magic** | To-hit, damage, resistances, death; spells, casting, spell effects (point and area), projectiles as state. | inside `pkg/sim` | A cohesion region of Sim Core; shares its state, publishes events. |
| 4 | **AI & Orders** | Unit decisions: acquisition, chase and cover, group orders, escort and defend, scripted orders' per-tick behaviour. | inside `pkg/sim` | Same as Combat & Magic. Reads world state; issues the same actions a player command would. |
| 5 | **Party, Items & Heroes** | Stats, experience and training, inventory, equipment and the wear rule, spellbook, party membership and joins. | inside `pkg/sim`, `pkg/data` columns, `pkg/herokind` | Same region rules. Item semantics come from Assets' tables, never hardcoded. |
| 6 | **Campaign & Scripts** | Mission scripts (checks and instants), triggers, mission flow, briefings, the world map, the town return, campaign session state. | `pkg/sim` script files, `pkg/scriptprofile`, `pkg/game` flow | Drives Sim Core through commands and script execution; owns what "mission" means. |
| 7 | **Town & Economy** | The town composer and its rooms; shop stock generation, pricing, buy and sell, inn, school, mercenaries, the purse. | `pkg/town`, `pkg/shopstock`, `pkg/shopitem`, `pkg/secondshop`, rules in `pkg/game` | The rules never depend on a screen; screens live in Client. |
| 8 | **Client** | Screens and input (mouse, keyboard, wheel), HUD and panels, widgets, the rendering pipeline (terrain, sprites, palettes to pixels, camera), fonts and text drawing, dialogue and message panels, sound, music and cutscenes. | `pkg/render/*`, `pkg/ui`, `pkg/audio`, `pkg/video`, `pkg/screenart`, `pkg/words`, view parts of `pkg/game` | Renders sim state and submits commands. Never mutates sim state directly. |
| 9 | **Persistence** | SAV, the only save format: reading an original save into the world, writing the current world as SAV, save slots, autosave and quick save. | `pkg/formats/sav`, `pkg/sim` binary state, `pkg/game` save paths, `pkg/hostfile` | One producer writes every save from the current state; knows nothing of Client. |
| 10 | **Mods** | Mod folders and manifests, resolved layers over the install's data, mod scripts, the parameters a mod may change. | `pkg/mod`, `pkg/modrt`, `pkg/rules`, `pkg/game` mod files | Layers over Assets; reaches Sim Core only through `pkg/rules` and the surfaces `docs/MODS.md` names. |
| 11 | **Map editor** | The headless edit model over one `.alm` map, the editor window and its tools, the editor's MCP surface. | `pkg/mapedit`, `pkg/editorapp`, `pkg/editormcp`, `pkg/ui` and `pkg/game` editor files | Reads Assets and writes maps and mods to explicit paths; test play runs the game's own mission entry. |
| 12 | **Programs** | The launcher, the game executable, the headless runner and scenarios, the inspection tools. | `cmd/*`, `pkg/launch`, `pkg/ini`, `pkg/pathdialog`, `scenarios/` | Thin: a program composes domains and owns no game rule. |

## Coupling rules

- Assets is a leaf: everyone may read its types; it knows no one.
- Domains 3 to 5 (and the sim half of 6 and 7) are cohesion regions inside the determinism wall.
  They share the world state; the boundary between them is subject ownership, not a package
  border. A story in one region that needs another region's rule uses that rule where it lives;
  it does not copy it.
- Client is the only domain that touches the player, and the only one ebiten reaches. Everything
  it shows is sim state or Assets data; everything it does to the world is a command.
- Persistence writes SAV from the current state. A field that does not survive save and load is a
  Persistence defect even when the feature that added it lives elsewhere.
- The second game is not a domain. It is profile data and `second*` or `rom2*` files inside the
  domain that owns the subject; one builder serves both games.
- Cross-domain features (most real features) list every touched domain in the story document.
  The 0157 shop round is the canonical example: Town & Economy (stock, prices), Client (screen,
  doll, picker, wheel), Party & Items (whose inventory), Persistence (nothing).

## Slicing rule

A further domain earns its row only as a system with its own rules, its own state and its own
screens, not as a large feature inside an existing domain. Sound and music stay in Client: they
present state and own none.

## Where each subsystem lives

Paths are relative to the repository root. A `*` stands for the rest of a file name.

### Assets

| Subsystem | Location |
|---|---|
| `.res` archives, virtual filesystem | `pkg/formats/res`, `pkg/vfs`, `pkg/installpath`, `pkg/namefold` |
| `.reg` registries, `Data.bin`, typed classes | `pkg/formats/reg`, `pkg/formats/databin`, `pkg/data` |
| `.alm` map container | `pkg/formats/alm` |
| Sprites, palettes, BMP | `pkg/formats/spr256`, `pkg/formats/spr16`, `pkg/formats/pal`, `pkg/formats/bmp` |
| WAV sound, Smacker video | `pkg/formats/wav`, `pkg/video`, `pkg/video/smacker` |
| Texts, item names, typed-text codec | `pkg/formats/itemname`, `pkg/formats/itempkt`, `pkg/formats/textinput`, `pkg/game/installtext.go` |
| Hall of fame | `pkg/formats/fame`, `pkg/game/fame*.go` |
| Game and language of an install | `pkg/base`, `pkg/locale`, `pkg/installinfo`, `pkg/sim/gameprofile.go`, `pkg/game/runtimeprofile.go` |

### Simulation (`pkg/sim`)

| Subsystem | Files |
|---|---|
| World, tick, hash | `world.go`, `step.go`, `hash.go`, `turn*.go`, `sessionclock.go` |
| Movement and pathing | `route.go`, `mover*.go`, `stride*.go`, `walkclaim*.go`, `facing.go`, `areacost*.go`, `pursuitsearch*.go` |
| Combat | `combat.go`, `engage.go`, `guard*.go`, `withdraw.go`, `deathroll.go`, `damage*.go`, `structurecombat.go` |
| Magic | `spell*.go`, `cast*.go`, `effect*.go`, `celleffect.go`, `flight*.go`, `creaturecast.go`, `usescroll.go`, `usepotion.go`, `usebook.go` |
| AI, groups, orders | `group*.go`, `escort.go`, `command.go`, `*order*.go`, `formation.go`, `target*.go`, `reacquire.go`, `nativepatrol.go` |
| Sight and relations | `sight.go`, `relations.go`, `presence.go` |
| Items, inventory, sacks | `item*.go`, `carry.go`, `equip.go`, `sack*.go`, `drop*.go`, `weight.go`, `rearm.go`, `wornchange.go` |
| Hero: skills, experience, regeneration | `skill.go`, `training.go`, `regen.go`, `autohealing.go`, `experiencevalue.go` |
| Mission scripts | `script*.go`, `rom2script.go`, `pkg/scriptprofile` |
| Structures | `structure*.go`, `buildingcaster.go` |
| Binary world state | `binary.go`, `*binary.go`, `saved*.go`, `original*.go`, `native*.go`, `current*.go` |
| Randomness | `rng.go`, `pkg/random` |
| Parameters a mod may change | `pkg/rules` |

### Persistence

| Subsystem | Location |
|---|---|
| SAV format: reading and writing bytes | `pkg/formats/sav` |
| An original save read into the world | `pkg/game/original*.go`, `pkg/game/resume.go`, `pkg/game/nativecity*.go` |
| The current world written as SAV | `pkg/game/sav*.go`, `pkg/game/current*.go`, `pkg/game/worldsave*.go`, `pkg/game/city*.go` |
| Slots, autosave, quick save | `pkg/game/savestore.go`, `savescan.go`, `autosavequeue.go`, `timedautosave.go`, `quicksave.go` |
| Save and Load screens | `pkg/ui/save*.go`, `pkg/ui/load*.go`, `pkg/ui/savepage*.go` |
| Player file writes | `pkg/hostfile` |

### Campaign, town, world map

| Subsystem | Location |
|---|---|
| Campaign and session | `pkg/game/campaign*.go`, `mission*.go`, `session*.go`, `winfinish.go`, `ending.go` |
| World map | `pkg/game/worldmap*.go`, `pkg/game/secondworldmap.go`, `pkg/ui/worldmap.go` |
| Town composer and rooms | `pkg/town`, `pkg/game/town*.go`, `pkg/game/roompage*.go`, `pkg/ui/town*.go`, `pkg/ui/room*.go` |
| Shop | `pkg/shopstock`, `pkg/shopitem`, `pkg/secondshop`, `pkg/game/shop*.go`, `pkg/ui/shopscreen.go` |
| Tavern, inn, hiring, school | `pkg/game/tavern*.go`, `inn*.go`, `hire.go`, `schoolpage.go`, `secondinn*.go` |
| Character generator | `pkg/game/chargen*.go`, `pkg/ui/chargen*.go`, `pkg/herokind` |
| Dialogue, speech, documents | `pkg/game/speech*.go`, `speakers.go`, `mapmessagepages.go`, `eventtext.go`, `pkg/ui/dialog*.go`, `pkg/ui/notice*.go`, `pkg/ui/documents.go` |

### Client

| Subsystem | Location |
|---|---|
| Application and screen flow | `pkg/ui/app.go`, `flow.go`, `screenregistry.go`, `screenhandler.go`, `pkg/game/frontend.go` |
| Mission view | `pkg/ui/viewer.go`, `overlay.go`, `fog.go`, `shadow.go`, `minimap*.go`, `pkg/render/terrain`, `pkg/render/camera` |
| Playable map assembly | `pkg/mapload` |
| Mission panels and HUD | `pkg/ui/panel.go`, `command*.go`, `inventory.go`, `spellbook.go`, `hud.go`, `characterpane.go`, `readout.go` |
| Input and cursors | `pkg/ui/missioninput.go`, `pkg/ui/cursor*.go`, `pkg/ui/heldkey`, `pkg/ui/systemclick`, `pkg/game/cursorregistry.go` |
| Widgets | `pkg/ui/widget*.go`, `pkg/ui/widgetkit`, `pkg/render/latch` |
| Text and fonts | `pkg/render/text`, `pkg/render/textfit`, `pkg/render/textsmooth`, `pkg/ui/linebreak.go`, `pkg/words` |
| Frame and window scale | `pkg/render/frame`, `pkg/render/catmullrom`, `pkg/ui/windowmode.go` |
| Screen art | `pkg/screenart`, `pkg/render/menu`, `pkg/render/backdrop` |
| Tips and tooltips | `pkg/ui/tooltip*.go`, `pkg/ui/tip*.go`, `pkg/game/tips.go`, `pkg/game/tipstore.go` |
| Menus and options | `pkg/ui/gamemenu*.go`, `pkg/ui/gameoptions*.go`, `pkg/ui/soundoptions*.go`, `pkg/game/gameoptions*.go` |
| Sound and music | `pkg/audio`, `pkg/ui/sound*.go`, `pkg/ui/music*.go`, `pkg/ui/ambient*.go`, `pkg/game/sound*.go`, `pkg/game/music*.go` |
| Cutscenes | `pkg/game/cutscene*.go`, `pkg/ui/cutscene*.go`, `pkg/video` |
| Cheat line and chat commands | `pkg/game/cheats.go`, `pkg/game/chatcommands.go`, `pkg/sim/cheats.go`, `pkg/ui/cheatinput.go` |

### Mods, map editor, programs

| Subsystem | Location |
|---|---|
| Mods: discovery, manifest, layers, scripts | `pkg/mod`, `pkg/modrt`, `pkg/game/mod*.go`, `pkg/ui/modscreens.go` |
| Map editor model | `pkg/mapedit` |
| Map editor window | `pkg/ui/mapeditor*.go`, `pkg/ui/editor*.go`, `pkg/game/mapinspection*.go`, `pkg/game/mapeditor*.go`, `pkg/editorapp` |
| Map editor over MCP | `pkg/editormcp`, `pkg/ui/mapeditormcp.go` |
| Launcher | `cmd/starter`, `pkg/launch`, `pkg/ini`, `pkg/pathdialog` |
| Game executable | `cmd/againrom` |
| Headless runner and scenarios | `pkg/game/headless*.go`, `pkg/game/scenario.go`, `pkg/ui/headless*.go`, `scenarios/`, `cmd/missionrun` |
| Inspection tools | the other programs under `cmd/` |

### Guards and build

| Subsystem | Location |
|---|---|
| Architecture guards | `internal/archtest`, `internal/storyguard` |
| Release test population | `internal/gatedtests` |
| Synthetic test data | `internal/synth`, `internal/savefixture`, `internal/cityfixture`, `internal/witness` |
| Versions and Windows resources | `internal/versionres`, `internal/buildinfo`, `internal/appicon` |
| Gates and packaging | `scripts/` |
