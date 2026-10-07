# Story `1028` — the mission command panel

## Result

The mission column's second slot draws the original's command panel: eight 34x34 cells composed from
the four shipped 160x80 bitmaps, with the disabled cells and the selected cell drawn the way the
original draws them, and the label of the cell under the cursor. Its eight keys are the original's —
`A`, `M`, `G`, `D`, `C`, `S`, `T`, `R` — and the five bindings that collided move elsewhere. This
build's four display-switch buttons leave the slot; `I`, `S`, `D` and `E` keep switching their boxes
as keys.

Pointable in `builds/current/`: the strip under the minimap shows the game's own command bar instead
of four authored letter boxes, and pressing `G` orders Guard.

`DIV-201` moves from FIDELITY-DEBT to closed for the cells this story builds, and keeps a narrowed
row for the three orders it does not.

## Why this story exists

Two owner answers, 2026-08-22. Full text and the options they were chosen from:
`pipeline/OWNER-RULINGS.md`, the entry of that date.

1. **The four display switches lose their buttons and keep their keys.** He was offered relocation
   to the character panel's slot, keys-only, or deferring the story, and chose keys-only. This
   retires the visible half of his 2026-08-11 control-panel directive — nothing draws a pressed
   state any more — and it is his own retirement of it, recorded as such.
2. **All eight of the original's accelerators are adopted.** He was offered all eight, only the free
   letters, or keeping this build's keys, and chose all eight.

Both sit under his 2026-08-21 ruling that where an authored value stands in for a decoded fact and
no owner directive drove the difference, the decoded value wins.

## What is decoded

Five rows carry this panel, all in the pin (`fb372e2`). **Read each whole** with
`go run ./tools/claim <ID>` from `implementation/research`. What follows names which row answers
which question; it is not the evidence and must not be cited in place of it. A row's own next
sentence has cost this project a landing before.

| Question | Row | Grade |
|---|---|---|
| Where the panel is, what class it is, and that it is the column's second child | `MENU-COMBAT-017` | High |
| The 8-cell grid: pitch, cell origin, the skip mask, the selected index, the screen gate | `TOWN-092` | High for the shape / Unknown for what a cell represents |
| What is drawn: the four bitmaps, the cell size, the draw order, the four state fields | `MENU-COMBAT-018` | High |
| The vocabulary, the labels, the accelerators, and every mouse edge | `MENU-COMBAT-019` | High |
| What each cell does, the two dispatch tables, and the arming gate | `AI-PANEL-053` | High for the tables and action routes |

`AI-PANEL-053` is **partially retracted twice** and the claim tool prints both overturns beside it.
Read them. The capability clause is gone: the gate is ownership through `view+0x144 & 4`, with cast
added by `& 0x200`, and no shipped class is tested. The labels of opcodes `0x18` and `0x14` moved:
`0x18` is Stand Ground and `0x14` is Retreat, not "aggressive" and not Patrol.

**Measured at this seat before this contract was written.** Both are premises for the lane to check,
not facts to inherit:

- The four bitmaps are in `graphics.res` at `interface/headsr.bmp`, `interface/commandbarr.bmp`,
  `interface/commandempr.bmp` and `interface/commanddnr.bmp`, each 38456 bytes on both preserved
  roots. Command: `go run ./cmd/restool list <root>/graphics.res`. A fifth entry,
  `interface/commandbarl.bmp` at 3896 bytes, is **not** one of the four and is not this story's.
- `main.txt` lines 0..7 in `main.res` at `text/main.txt` read `Attack <A>`, `Move <M>`, `Guard <G>`,
  `Defend <D>`, `Cast <C>`, `Swarm <S>`, `Stand Ground <T>`, `Retreat <R>`. Command:
  `go run ./cmd/restool cat <root>/main.res text/main.txt`. The accelerator travels inside the label
  in angle brackets, so the label teaches its own key and no second table is needed.

## What is not decoded

- **What each cell's picture is.** `TOWN-092` is explicitly Unknown for what a slot represents in
  play, and `MENU-COMBAT-018` establishes that the pictures are not separate assets at all: they are
  regions of the two full-panel bitmaps. So this story never chooses an icon. It blits rectangles.
- **What the minimap's own paint routine dispatches through.** A research experiment on the adjacent
  widget is open and unmerged as this contract is written, and its rows are **not in the pin**. Cite
  only rows the pin carries: `go run ./tools/claim <ID>` must print the row, or it is not citable
  here. No id from that experiment appears in this document, deliberately -- a sentence naming an id
  is a citation to `scripts/check-claim-citations.sh`, which cannot tell a citation from a sentence
  about one.
- **The full route from the panel's own message post to the order handler.** `AI-PANEL-053` names the
  handler and both tables; the intervening dispatch is not traced. This story routes a cell to an
  order through this build's own seams, which is a reconstruction of the effect and not of the path.

## What this build does today

- `pkg/ui/hudtoggles.go` draws four authored 32x32 letter switches centred in the 160x80 slot, with
  an authored on/off palette, and `hudTogglePanelAt` hit-tests them. The whole file is this story's
  to replace or retire.
- `pkg/ui/hud.go` puts the slot at `hudMinimapReserve` (158) with height `hudToggleBarH` (80),
  summing to `hudPanelTopY` (238) and pinned by `TestHudColumnSlotsSumToThePanelsOwnTop`. **The
  geometry is already right and this story does not move it.**
- The order vocabulary exists at the seams. `MapStance(entity, guard bool)` is Guard and Stand
  Ground; `MapMarch(entity, patrol bool, x, y int)` is Patrol and March; attack is armed by
  `Viewer.armAttack` and spent by the next press; a cast is armed by a spellbook selection; a plain
  ground tap is a move through `MapOrder`. All are in `pkg/ui/flow.go` and `pkg/ui/command.go`.
- `canArmAttack` gates on ownership and on no class, which is what `AI-PANEL-053`'s retraction says
  the original does. That agreement is already recorded in `armCommand`'s own comment.
- The keys today: attack `F`; Guard `U`; Stand Ground `T`; Patrol `P`; March `R`; pick-up `G`;
  minimap toggle `M`; pack `I`; worn `E`; camera pan `W`, `A`, `S`, `D` and the four arrows.

## Scope — five behaviours

**B1. The panel draws.** The slot composes from the four shipped bitmaps by
`MENU-COMBAT-018`'s own order: `headsr.bmp` when the panel is inactive; otherwise `commandbarr.bmp`,
then each disabled cell's own source rectangle from `commandempr.bmp`, then the selected cell from
`commanddnr.bmp`. Cells are 34x34 at panel-local `(8 + 34c, 7 + 34r)`, row-major, four columns by two
rows. There is no hover bitmap, no border and no separate icon, because the original has none.

The cell under the cursor draws its `main.txt` label. This is part of B1 rather than a sixth
behaviour, and it is the one part that must not be cut: B3 rebinds five keys, and the label is where
the shipped game teaches its own bindings.

**B2. The cells act.** Left down acts; a double-click aliases it; a left-drag re-enters it on every
delivered move; a right up cancels; the other right edges are no-ops. A margin or disabled left-down
clears the selected overlay and selects nothing, leaving an armed mode intact. Guard, Stand Ground
and Retreat issue immediately; Attack, Move, Defend, Cast and Swarm arm. The gate is ownership of the
selection and nothing else.

**B3. The eight keys are the original's.** `A`, `M`, `G`, `D`, `C`, `S`, `T`, `R`, each doing exactly
what its cell does.

**B4. The colliding bindings move.** The camera loses `W`, `A`, `S`, `D` and keeps the arrows; it
gains screen-edge panning so that a player with no arrow keys is not stranded. Pick-up leaves `G`,
the minimap toggle leaves `M`, and the attack arm leaves `F` for `A`. Every replacement letter is
verified free against the whole of `pkg/ui/app.go` and `readInput` in `pkg/ui/viewer.go`, the way
each existing binding's own comment already records.

**B5. The display switches lose their buttons.** `I`, `S`, `D` and `E` still switch the pack bar, the
spellbook bar, the doll and the worn set. `E` and `I` are unaffected by B3. `S` and `D` collide with
Swarm and Defend: **the collision is real and the lane must resolve it**, because a key cannot both
switch a box and order a unit. Resolve it in the spec with a reason, and record whichever side moves
as a divergence row.

## Out of scope

- **Defend, Swarm and Retreat as orders.** This build has no such orders and building three is its
  own story. Their cells ship **disabled through the original's own skip mask**, which is the
  mechanism the original already has for a cell that cannot be used — so the gap needs nothing
  authored to express it. A row records that the mask is carrying a gap rather than a game state.
- **Patrol.** It is not one of the eight; `AI-PANEL-053` finds no route that leaves mode 8 armed. It
  keeps key `P` and gains no cell. A row records the addition.
- **The minimap's content (id 5) and the character panel's content (id 7).** Untouched.
- **The panel's own message protocol.** This story routes a cell to an existing seam. It does not
  reconstruct the original's message post.
- **800x600.** `MENU-COMBAT-017` gives the panel's rect there too. This build composes at 1024x768
  and the middle arm is not built on the owner's silence.

## Domains touched

`DOMAINS.md` names nine. This story touches three:

- **8 Client** — the panel, its composition, its hit test, the keys, the camera's own keys.
- **4 AI & Orders** — at the seam only. Every order this story issues already exists; nothing new
  reaches the sim.
- **1 Assets** — four new archive entries read at runtime.

**Never in the repository.** The four bitmaps are game assets. They are read from the asset root
through the existing archive tier, addressed by `graphicsPrefix + "interface/..."`, and no byte of
them enters git or a test fixture. `check-no-game-assets.sh` is irreversible if violated.

## Ceiling

**Four adversarial passes.** Three domains and five behaviours would allow three, but a cell issues
an order and an order changes hashed simulation state, so this story reaches it — through existing
seams and with no new hashed field, which is why four and not more.

Five behaviours is at the limit the project's rule allows without recording a reason. The reason it
is not split: B1 through B5 are one result the owner asked for in one answer, and a panel whose cells
draw but do not act, or act but under the wrong keys, is not a thing anyone can point at.

## The witness must see real art, and it must exist from the first commit

This story is the exact shape that produced `1016`'s two returned passes: production composes shipped
bitmaps, and the only test that can see the result needs an install, which golden rule 2 keeps out of
the repository's own chain. A synthetic-image test will pass over a panel that is visibly wrong.

So the install-gated release test is **part of B1's fix, not a follow-up**: compose the panel through
the production composer, from the real four bitmaps, on both roots, and assert something a wrong
composition would fail. Write it with B1, not after it.

The repository's own chain cannot reach it. Run `pipeline/check-release-tests.sh` and
`pipeline/check-scenarios.sh` from `<seat>` on **both** roots, by name, and report
the number each prints. A skip and a pass both print `ok`; both scripts exit 2 with no install, so a
bare run is never a pass.

## Divergence ids

`DIV-230`..`DIV-234` are reserved for this story, allocated with `pipeline/next-div-id.sh` on
2026-08-22 against a highest id in use of `DIV-229`. Allocate from that range, record any tail as
returned unused, and never reissue a returned id. Rows owed, at least:

- The three cells whose orders do not exist, shipped disabled.
- Patrol's key with no cell.
- Whichever of `S`/`D` moves in B5.
- Screen-edge panning, which is authored and which the original does not have.

`DIV-201` is rewritten at the landing rather than closed outright: the ordering half was never a
divergence, the content half is what this story pays down, and what remains is the three orders.

## Gates

Both repositories' Go chains by glob — run the glob, never a remembered list — with exit codes
captured as `out=$(bash scripts/check-x.sh 2>&1); code=$?`, never read off a pipe. Then
`check-release-tests.sh` and `check-scenarios.sh` on both roots, and `check-milestone.sh` with
`AGAINROM_MILESTONE_DRIVE` pointing at a `missionrun` the lane built itself: the default drive is
`builds/current/`, which every worktree on this machine shares.

Report the number each script prints, not its verdict. Deletion set empty or explained:
`git diff --diff-filter=D --name-only <before> <after>`.

## Style

`PROSE.md` governs every document this story writes. English.
