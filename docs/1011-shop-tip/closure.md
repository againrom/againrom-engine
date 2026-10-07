# 1011 — closure

Branch `1011-shop-tip`, base `c3ac3d4`, merged forward to master `857fb43` (`1012-notice-cadence`)
before this gate ran. Research pin `02a1403`, unchanged — not bumped, per brief.

Gate, run on the merged tree: `go build ./...` clean, `go vet ./...` clean, `gofmt -l` over tracked
and untracked Go files prints nothing, `go test -trimpath -count=1 ./...` all packages `ok` (54
tested packages, 0 failures), `bash scripts/check-no-game-assets.sh` reports "clean (tree scan)".
No file under the repository is deleted by this story (`git diff --diff-filter=D` against `c3ac3d4`
is empty for this story's own changes).

## Result

The shop room now draws a tip widget over the merchant panel's lower half: activation loads
`main/text/tips/shop1.txt` whole from the install and shows it, word-wrapped, from
(169,162) to (476,283) — 121 of the decoded rectangle's 136 rows, stopping above the pre-existing
message strip. `docs/DIVERGENCES.md`'s `DIV-018` moves to `DIVERGENCES-CLOSED.md`. Two new rows are
added, both spent: `DIV-132` (the widget's own construction gate and its unimplemented
`shop2.txt` swap) and `DIV-133` (the authored wrap rule and the authored layering against the
message strip).

## The twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | N/A | No new format decoded. The file is read whole through the existing archive `entrySource`; no new parser. |
| Runtime state | PASS | `townScreen.shopTip` is new state, set on every `Choose` into `roomShop` and carried out by `ShopScreen()` as `ShopScreenView.Tip`. `TestEveryShopEntryRereadsTheTip` (`pkg/game/shoptip_test.go`) swaps the install's `Archives` between two visits and confirms the second visit's `Tip` reflects the new file, proving a fresh read each entry rather than a cached value. |
| Simulation | N/A | `pkg/sim` is untouched; no file under it is modified by this story. |
| Player input | N/A | The widget accepts no input; nothing in `pkg/ui/command.go` changes. |
| AI | N/A | Not touched. |
| UI / HUD | PASS | The whole subject. `drawShopTip`/`wrapShopTip` (`pkg/ui/shopscreen.go`), unit-tested (`pkg/ui/shopscreen_test.go`) and confirmed against a real composed frame below. |
| Triggers / scripts | N/A | Not touched. |
| Inventory / equipment | N/A | Not touched. |
| Persistence / save-load | N/A | Nothing is serialized; the tip is re-read from the install on every shop entry, never stored in the byte form. |
| Campaign / session | N/A | Not touched. |
| Shipped content | PASS | Verified against both preserved installs' own `shop1.txt` (below): byte counts match `SHOP-SCREEN-038` exactly on both roots. |
| Interactions with existing mechanics | PASS | `shopMessageRect` (the answer to the last shop press) is unaffected: `TestTheAnswerMessageIsUnaffectedByTheTip` confirms every pixel of the message strip is unchanged whether or not a tip is drawn above it; mutation-verified below. |

No in-scope GAP. `DIV-018` closes; its residual content is fully re-homed to `DIV-127` (shelf
cadence, unrelated to this story) and to `DIV-132`/`DIV-133` (this story's own authored decisions),
none of which are GAPs against this story's own stated contract — they are disclosed divergences
from ROM1 truth where research does not resolve the question.

## Integration witness

`cmd/shopdump` (pre-existing developer tool) opens a real campaign session
(`game.NewFrontEnd(root)`), finishes the campaign mission the registry marks as offering a town,
presses the shop door through the real `ui.TownScreen.Choose` flow, pages the merchant's greeting
to its end, and composes the shop screen through `ui.ComposeShopScreen` exactly as the running game
does. Run against both preserved installs:

```
AGAINROM_ASSETS=<en> shopdump.exe -png <dir>
  town opened after mission 20, chapter 30, purse 600
  tip: 208 bytes, hex 4174207468652073686F70...65786368616E67652E0D0A

AGAINROM_ASSETS=<ru> shopdump.exe
  town opened after mission 20, chapter 30, purse 600
  tip: 150 bytes, hex 82EB20ADA0E5AEA4A8E2A5...8FE0AEA4A0E2EC3E2E0D0A
```

208 bytes (EN) and 150 bytes (RU) match `SHOP-SCREEN-038`'s claimed file sizes for `shop1.txt` on
both roots exactly. The EN hex decodes to readable ASCII English text ending `\r\n` (0x0D 0x0A):
"At the shop you can buy or sell equipment. Click on a shop shelf to view its contents. Drag & drop
the items or equipment onto the table, and then click the <BUY> or <SELL> buttons to complete the
exchange." The RU hex decodes to CP866 Cyrillic text of the same shape, also ending `\r\n`. Both are
the install's own file bytes, read by the production load path (`townScreen.Choose` →
`ReadShopTip` → `ShopScreen()` → `ComposeShopScreen`), not a value constructed by the witness.

The `-png` run wrote a composed frame to a scratch directory outside the repository. Inspected
visually: the tip's four wrapped lines read legibly over the merchant panel's stone-floor
background, stop above the "weapons - 100 on the shelf" message line with no overlap, and the
message line itself is unobstructed.

**What this does not witness.** `shopdump` presses only the first shop entry of one campaign
session; it does not exercise a second visit, an install missing `shop1.txt`, or the `shop2.txt`
swap (not built, `DIV-132`). Those are covered by the synthetic tests below, per golden rule 2 (no
test may read the real install; the real-install run is this developer tool alone).

## Mutation evidence

Line reverted: `pkg/ui/shopscreen.go`, `drawShopTip`'s clip condition, from

```go
if y+font.Height() > shopMessageRect.Min.Y {
```

to

```go
if y+font.Height() > shopTipRect.Max.Y {
```

(the naive full-height version tried first, before this story settled on stopping above the
message strip). With the mutation in place, `go test ./pkg/ui/... -run 'ShopTip|TheAnswerMessage'`
reddens both `TestDrawShopTipStopsBeforeTheMessageStrip` (a long tip now paints past the bound it
asserts) and `TestTheAnswerMessageIsUnaffectedByTheTip` (a long tip shows through the message's own
unpainted margins, changing pixels the test reads directly off the composed frame at
`shopMessageRect`'s own coordinates — the same derivation `drawShopMessage` itself paints through,
not a recomputed expectation). Reverted back to the merged version; `go test ./...` is green again
(see gate above).

## Research reconciliation

- `SHOP-TIP-045`'s decoded load, placement and second-text mechanism are implemented for the first
  text; its own stated Unknown (the construction gate's setter) is authored open, and its second
  arm (`shop2.txt`) is not built because the arm's own trigger is not traced. Both recorded,
  `DIV-132`.
- `TEXT-API-007`'s stated absence of a break rule and `DLG-WRAP-009`'s one decoded pitch number are
  both carried forward exactly as stated; the wrap algorithm itself is authored. The layering
  against `shopMessageRect` answers a question no claim addresses. Both recorded, `DIV-133`.
- `MISSION-DOC-021`'s whole-file, no-line-split reader is reproduced by `ReadShopTip` and confirmed
  by `TestReadShopTipReadsTheWholeFileWithNoLineSplit`, which fixtures a payload carrying an
  embedded CR and asserts it survives unsplit.
- `SHOP-SCREEN-038`'s file-size clause is independently corroborated by the real-install witness
  above (208/150 bytes, exact) rather than merely cited.
- No claim was refuted by this story.

## Script-gap census

```
go build -o /tmp/mr ./cmd/missionrun
AGAINROM_ASSETS=<en> /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED  ->  0
AGAINROM_ASSETS=<en> /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED  ->  0
```

Both match `pipeline/milestone-baseline.txt`'s current values (0 and 0). Unchanged by this story:
no `pkg/sim` or script-execution file is touched; this story's own subject is a UI widget drawn
outside the census's measured domain.

## What remains open

`DIV-018` is closed. What remains open, split across its successor rows:

- `DIV-127` (unchanged by this story): the four shelves' own eleven-frame animation cadence is
  undecoded; frames 2-11 are never read.
- `DIV-132` (new, this story): the tip widget's construction gate (`[L03631]`) and the
  `shop2.txt` swap trigger (`0x402`) are both undecoded; this build shows shop1.txt unconditionally
  and never swaps.
- `DIV-133` (new, this story): the original's own wrap break rule is undecoded, and no claim states
  how a message-line affordance and the tip widget share the panel; both are authored.

A shipped affordance quietly deleted would have been a defect even with a more faithful
replacement; this story's own clip choice keeps the pre-existing message strip pixel-identical
(`TestTheAnswerMessageIsUnaffectedByTheTip`, mutation-verified) rather than deleting or degrading
it.
