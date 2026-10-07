# Shared delayed hover help

The owner requests one delay for item, spell and other supported hover help,
selectable as 0, 100, 200, 300, 400 or 500 milliseconds. Game Options now has
that cycling row in both mission and town menus. Missing or invalid values use
500 ms; zero displays immediately. TooltipDelay is stored beside the existing
options, preserves other keys and is independent of introductory TipsMode.
A failed write keeps the live value and displays the error.

The application and mission viewer share one presentation controller. Pointer
movement, target identity or text changes, focus loss, buttons, keyboard input,
modals, drags, screen changes and resize invalidate its wait. Visible help
expires after25 seconds. No simulation, campaign, AGS or SAV state is changed.
The town input snapshot is reused for tooltip resolution instead of rebuilding
the tavern/school model on neutral Update.

Coverage: existing worn/backpack/doll/shop item and spellbook descriptions,
command buttons including selected spell names, character-pane buttons and
visible/disclosure-filtered stat rows, backpack arrows/background, shopkeeper,
stock selectors, shelf arrows and empty cells/money; tavern candidate gear and
stats; both school's five semantic skill cells; town doors/menu; eligible world
map regions; generator name, difficulty, class pictures, navigation, skill
icons, stats/points and card. Bindings read main.txt and sites.txt from the
lawful install. Missing text stays absent; no source prose enters this repo.

Authority: owner direction and knowledge k41 TEXT-HOVER-048,
TEXT-HOVERSET-049, TEXT-HOVERCHAR-050, TEXT-HOVERROOM-051,
TEXT-HOVERTEXT-052 and TEXT-HOVERPAINT-053. The knowledge pin advances from k40.
DIV-1269 records configurable timing/town access; DIV-1270 records the retained
popup framing/fonts, wrapping and consistent reset policy. Original inherited
getters, map-list/monster-spell formatted help and native pixel/timing parity
remain unclaimed. Retained source-specific item formatting debt is unchanged.

Focused proof covers all six delay boundaries, expiry, changed text under an
unmoved pointer, item/spell/command sharing, both school masks, stock indices,
card disclosure, modal/focus/resize handling and CPU overlay pixels. Cold-start
preference tests preserve unrelated keys. The paired installed pause-menu
witness clicks all six values, verifies row width and opens a fresh FrontEnd
for each; its paused world hash stays unchanged. The registered installed
shop/generator witness proves the500 ms boundary and45 generator targets per
locale, with every composed popup inside the640x480 frame. Owner-only PNGs
and logs live under review/story1182 outside Git.

Headless event snapshots expose delay, target identity, visibility and bounds
from the same renderer, without installed text bytes. The seat runs the final
Go, asset, paired release, divergence and relevant headless checks on the local
merge after the sole adversarial review. Work remains local under the owner's
deferred-push instruction. Physical window/input acceptance remains unavailable
under the existing control denial; no workaround is used.
