# Read-only mission hover inspection

## Result and authority

The mission figure and statistics card follow one tagged `kind + id` subject.
Hovering a visible unit or structure does not select it. Leaving eligible map
content restores the existing selection presentation, including a group summary
for multiple selected units. This switching policy is owner
direction, recorded as `DIV-536`; `AI-CURSOR-231` does not establish panel
auto-switching.

Structure names and class IDs come from `structures.reg`. Current/max health
and location come from the existing simulation projection (`ALM-CLS-053`,
`DAT-BLD-005`). A structure uses the flat `infowindow/<Picture>.bmp` leaf with
no tier (`UNIT-PICT-037`). Missing art remains blank (`REG-PICT-083`).
`TERR-STRUCT-102` still governs ruin graphics; `Indestructible` is not treated
as immunity. No simulation, save, footprint, ownership or diplomacy rule changes.

## As built

`pkg/ui/inspection.go` resolves the pointer against the production sprite order,
opaque pixels, camera transform and mirror. Noninspectable opaque objects block
objects behind them. Flat structures remain behind the sprite pass. The
frameless-unit diagnostic square remains inspectable through its existing full-cell
pick rectangle, which can also cover underlying art outside that small square.
The independent mission-10/20 census found no such missing-sprite units on either
install; other missions were not censused. Hidden units and explored
but not currently visible structures expose no live inspection. Modal, HUD,
held-button, marquee, item-drag and out-of-window states restore selection.

Both mission panels use that subject. Humans read live equipment through the
existing `unitFigure` cache without receiving inventory interaction masks.
Selection, commands, pack, spellbook and doll slot operations keep their original
selected actor. Structure registry selection rectangles are retained metadata,
not substituted for the measured sprite hit geometry.

## Focused proof at checkpoint

`TestInspectionProductionPointerKeepsSelectionAndBothPanelsTogether`,
`TestInspectionGatesAndFogDoNotLeakOrChangeInventory`,
`TestInspectionUsesDrawDepthAndOpaquePixels` and
`TestInspectionLeavesCommandHotkeysOnActualSelection` exercise App pointer and
hotkey dispatch, identity collisions, overlap, fog, modal/drag/HUD refusal,
pointer leave, cache refresh and selected-actor command ownership.

`TestReleaseInspectionLivePanels` passes separately on both lawful EN and RU
roots. Mission-10 structures 0/7/11 show 30000/30000, 1000/1000 and 0/0.
Expected bitmap pixels are read independently from raw BMP headers/BGR rows.
RU structure 11 names absent `graphics/infowindow/ruins.bmp`; its central figure
area matches the unadorned installed pane body. EN shows that file's pixels.
Non-owned Human 22 matches an independently painted `mfighter/17.256` base plus
`primary/0801007.256` weapon. A real unequip and damage step changes its figure
and stats. The paused hover matrix preserves world hash `332d187ef80b0e43`,
selection and inventory. Doll clicks after both foreign subject kinds enqueue
unequip for selected hero 35; Guard hotkeys while hovering act on the selected
unit only.

Measured window/frame/scale pairs: 640x480 -> 1024x768 / 0.625;
800x600 -> 1024x768 / 0.78125; 1024x768 -> 1024x768 / 1;
1280x720 -> 1366x768 / 0.9370424597;
1920x1080 -> 1366x768 / 1.4055636896. These are five tested window sizes, not
five distinct layout branches. The installed selected hero's pack/book leave
viewports 864x581 and 1206x581 respectively.

The observable result is the production headless mission panels, not a lower
script-gap count. No desktop window was driven and no install was written.
The EN one-tick `missionrun -trace` probes report UNSUPPORTED 0 for mission 10
and 0 for mission 20. The seat baseline has no unsupported rows and lists
16/27/12 and 14/15/11 checks/instants/triggers respectively; this slice moves
neither population. The probe binary used `-buildvcs=false` after sandbox VCS
stamping failed with exit status 128; simulation and trace inputs were unchanged.

## Pending and exclusions

This checkpoint includes master `0e2604f32f6eba5992d048a44a0648a79ace08f0` and
research pin `240c6279008ef0339be01af8f3784aeb41f0887a`. Story 1082's campaign
difficulty, transactional mission entry and save changes remain intact. The only
merge conflict was adjacent ledger entries; both DIV-532 and DIV-536 are retained.
No hover production code needed reconciliation edits. Focused hover, difficulty
and EN/RU inspection checks are repeated on this composition. The root-owned
fresh adversarial review and final merge gates remain pending. No landing or
`builds/current/` rebuild is claimed.

`DIV-284`, `DIV-285` and `DIV-286` remain open: inspection does not add structure
selection or cursor capability-mask bits. Physical structure attacks (`DIV-457`)
remain separate from inspection. Story 1086 closes the multi-cell spell
exclusion (`DIV-458`), with retained lifecycle debt in `DIV-552`. Exact ROM1 panel
switching and hit geometry are not claimed. DIV-537..539 are unused.
