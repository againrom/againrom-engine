# Story `1043` — as-built specification

## Result

Every current production consumer named by the contract reads its program-chosen words from the
active install. English and Russian therefore share one code path and differ only where their five
retained text tables differ. The change does not add a language switch, a general translation API,
or a new interaction.

## Install text

`InstallWords` retains `main.txt`, `dialogs.txt`, `unitname.txt`, `stats.txt` and `sites.txt` from
the active archive set. Its five accessors keep global-main and table-local indices separate.
`Words` begins with `ui.AuthoredWords` and replaces each available field independently, so a
missing, short or empty table cannot blank an otherwise visible control. The executable input
census reads all sixteen original text tables but production retains only these five.

## Character panel

`PanelSubject` carries the subject's actual presentation name, presentation-only `UnitNameIndex`,
`DetailLevel`, `DetailSet`, `Words` and the bounded original actor projection used by the two
conditional captions. Town/shop, chargen and mission adapters resolve their Human or Unit
definition before writing the name index and carry the actor flags at `+0x18c`, experience value at
`+0x1c` and derived byte at `+0x14a`. All current owner-character and preview routes write level 7.
The shared composer uses the subject name when it is non-empty and otherwise falls back to
`unitname.txt`; fixed captions still come from their published `main.txt` slots. This owner-directed
priority is the accepted `DIV-428` conflict with `TEXT-UI-034`.

When `DetailSet` is true, sight and speed require a level greater than 1; attack and damage greater
than 2; defence and absorption greater than 3; primary attributes greater than 4; and the skill and
resistance families greater than 6. A positive maximum gates Health or Mana. Values keep the
existing integer, pool, fractional and damage-range grammar. On the fixed 160-pixel card a caption
wider than its column is shortened before its numeric value; the complete value is retained and the
card geometry does not move. A subject assembled by legacy code without `DetailSet` keeps the
pre-story full-detail behavior.

At full detail, `main.txt[190]` paints in the existing right cell of the Weight row when actor flag
bit 0 is clear and actor `+0x1c` is nonzero. `main.txt[191]` paints in the existing right cell of the
XP row when actor flag bits 0 and 4 are both clear and actor `+0x14a` is nonzero. The last byte is 2
for type ID `0x49` and zero otherwise. These are original actor predicates, not synonyms for this
build's `Mage` or `AlwaysHits` fields. The captions are headings, not authored yes/no values, and
they add no row or geometry to the fixed card.

## Item descriptions

The two final boundaries remain `itemInstanceInfoLines` and
`itemInstanceInfoLinesWithWeaponDamage`. Both resolve labels from `stats.txt` and apply the ten
published executable format forms below those boundaries. Mission worn and pack consumers retain
the live weapon-damage resolver. Shop doll, shelf, table and pack consumers retain the nil resolver.
Identity, effects, price, equipment state, cell identity, spell resolution and drag behavior are
unchanged.

The later staff-tooltip hotfix composes at those same boundaries. When a weapon has a resolved
attached cast, its dedicated `Magic`, `Casts`, `Damage` and `Range` block replaces the unused
physical interval and the duplicate raw cast-effect line. The spell name, powered interval and
range still come from the resolved table and live/stored spell projection. Ordinary weapon lines,
item values and every remaining raw effect continue to use the active `stats.txt` words and forms.

## Notices, menus and map cards

A successful save states `main.txt[203]` and no longer exposes the generated filename. The existing
message lifetime, error paths and save transition are unchanged. Mission outcomes retain main
slots 140 and 141. Mission and town Esc-menu populations retain the dialog-table fields and
behavior built by story `1042`. Shop Undo, Buy, Sell and Exit retain main slots 72, 70, 71 and 73.

The world-map home card paints `sites.txt[0]` followed by `main.txt[261]` at the existing two text
origins. A positive mission payment paints `main.txt[262]` through `%s: %d c`. Card rectangles,
route state, marker behavior and click geometry do not change.

## Selection presentation

The mission pane branches on the current `presentSelected` result. Zero presented actors paint
main slots 47 and 48. Exactly one actor follows the complete character-panel path. More than one
paint main slots 49 and 50 plus the current count. The pane cache key includes both status strings,
so changing the plural count rebuilds the pane. Town and chargen keep their existing empty state.

## Boundaries

The story changes Assets & Formats and Client & Presentation only. No canonical simulation field,
hash, AI order, trigger, inventory rule, price rule, campaign rule or save form changes. Save form
remains 61. Fixed character-card geometry, shared spellbook/no-op hover behavior, doll dragging,
compact tavern cells and both item-description helper boundaries remain intact.

The category/shopkeeper hover strings, identify prompt and marker-site labels are decoded but have
no current production painter. They remain excluded. Exact world-card joining beyond the preserved
two origins and the source of the original panel detail level remain Unknown and are disclosed in
the divergence ledger.
