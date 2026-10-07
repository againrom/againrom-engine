# 1011 — contract

## What this story settles

`docs/DIVERGENCES.md` row `DIV-018` names two undrawn shop affordances: the four shelves' own
eleven-frame animation and a tip widget showing one of two shipped texts. Story `1009` drew the
shelves' static first frame and split the animation cadence into its own row, `DIV-127`. This
story draws the tip widget, the remaining half of `DIV-018`.

## What will work after this story

Entering the shop room loads the install's own `main/text/tips/shop1.txt` and shows it, wrapped,
inside the merchant panel. An install shipping no tip file draws nothing, matching every other
install-text reader in this tree.

## The observable result

The shop room, composed against a real campaign session on a lawful install, shows the install's
own tip text where `SHOP-TIP-045` places it. `DIV-018` closes; its residual authored decisions are
recorded as new rows rather than left implicit.

## Claims this story is built on

| Claim | Confidence | What it supplies |
|---|---|---|
| `SHOP-TIP-045` | Established for the load and the placement / graded Unknown for the construction gate's setter | Activation constructs a widget of a different class than the merchant panel's own children, at merchant-panel-relative (0,162,312,298) — view-relative (164,162,476,298) — holding the whole of `main/text/tips/shop1.txt`, handed to the merchant panel as a child. `R1760` can replace the text with `shop2.txt` at most once, behind a latch reached from the shop view's own `0x402` message arm. The whole construction is gated on global `[L03631]`, whose setter is not read. |
| `MISSION-DOC-021` | Established | `R1293` opens a node, reads its whole length and appends one NUL — no per-line split. This is the reader the tip text goes through, distinct from `installtext.go`'s `TextTable`, which reproduces a different loader's own CR-consuming line grammar for `main.txt`/`dialogs.txt`. |
| `TEXT-API-007` | Established for the call, Unknown for the break rule | The wrap pass `R0727` is built on the font's own measurer; no exact line-break algorithm is published. |
| `DLG-WRAP-009` | Established | The wrapping window's own default line pitch is the font's height plus 2; every shipped consumer of that window takes the default. |
| `SHOP-SCREEN-038` | Established (file sizes) | `main/text/tips/shop1.txt` is 208 bytes on the EN root, 150 bytes on RU. |

## Aspects that apply

| Aspect | Applies |
|---|---|
| Data | No — no new format decoded; the file is read whole through the existing archive reader. |
| Runtime state | Yes — `townScreen.shopTip` is new front-end state, loaded on shop entry. |
| Simulation | No — `pkg/sim` is untouched. |
| Player input | No. |
| AI | No. |
| UI / HUD | Yes — the whole subject: a new widget drawn on the shop screen. |
| Triggers / scripts | No. |
| Inventory / equipment | No. |
| Persistence / save-load | No — the tip is re-read from the install on every entry; nothing is serialized. |
| Campaign / session | No. |
| Shipped content | Yes — verified against both preserved installs' own `shop1.txt`. |
| Interactions with existing mechanics | Yes — the widget's own rect overlaps `shopMessageRect`, an existing widget (the answer to the last shop press); the two must not visually collide. |

## Domains touched

**Town & Economy** (the tip is shop content, loaded per shop entry) and **Client** (the widget's
draw, wrap and layering). Persistence and Sim Core are not touched.

## Divergence allocation

`DIV-132` and `DIV-133` are reserved for this story. Both are expected to be spent: `SHOP-TIP-045`
leaves the construction gate's setter and the second-text swap's trigger unread, and no claim
gives an exact wrap algorithm or states how a message-line affordance and the tip widget share the
panel. `DIV-018` is closed in place, since both of its named subjects (shelf animation, tip
widget) are drawn once this story lands; its residual shelf half already has its own successor row,
`DIV-127`.

## Out of scope

The shelf animation's own cadence (`DIV-127`, unrelated undecoded fact). The `shop2.txt` swap and
its `0x402` trigger (the swap's own user-action trigger is not traced by any claim read for this
story; implementing a guessed trigger is deferred, not built here). The merchant's animated Yes/No
poses (`DIV-020`, a separate row).
