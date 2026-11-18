# Side panels with visible resize handles

## Result

The docked panels resize from the edge facing the map. That edge and the
corner grip of a floating panel light up under the pointer. Minimizing a
panel gives its space to the map, and the arrow brings it back. With no
object selected, Properties shows one status line with help in its title.

## Authority

- Owner approved T9 and chose the inner splitter and visible hover feedback.
- The editor UI framework defines how titles, controls and their sizes are shared.

## Rules

1. A docked panel resizes at its inner edge. A floating panel keeps its corner grip.
2. The splitter and the floating corner grip change ink and cursor on hover.
3. The title sets strip thickness. A fresh dock takes its size from its contents at 1920×1080 and above.
4. The map gains the freed space on minimize. The strip arrow restores the panel and a saved resize keeps its size.
5. Empty Properties shows one localized status line and explains selection behind the title help mark.

## Code

| kind | its one builder | change |
|---|---|---|
| Panel frame | `panelsLaid`, `sidePanelRect`, `dockSpec`, `stripSpec` | Measure fresh frames and carry the dock edge into layout |
| Resize handle | `panelInnerEdge`, `panelEdgeAt`, `popupLayout`, `drawPopupResizeHandle` | Share the splitter and grip geometry and highlight |
| Title and help | `popupTitleHeight`, `popupTipAt`, `layoutTip`, `drawPopupTip` | Measure strips from the title and explain selection in one tip |
| Panel profile | `readPanelProfile`, `panelProfile.bytes`, `savePanels` | Keep user sizes and encode automatic sizing as zero |
| Panel description | `buildMockupPanels`, `layoutPopup`, `writePopupParts` | Use the live panel frames and their layout for previews |
| Words | `pkg/words` | Add the empty Properties status in English and Russian |

## Proof

| test | rules |
|---|---|
| `TestMapEditorDockHasOnlyAnInnerSplitter` | 1 |
| `TestMapEditorResizeHandlesHighlightWithTheCursor` | 2 |
| `TestMapEditorPanelFrameLayout` | 1, 3, 4, 5 |
| `TestPanelDescriptionUsesTheEditorFrame` | 1, 2, 4, 5 |
| `TestPanelProfileRoundTripAndBytes` | 4 |

Release tests at the release in EN and RU: `TestReleaseMapEditorObjectsAndGroupsFrames`.

## Open debt

- Installed-font panel fit and appearance await the EN and RU release witness.
