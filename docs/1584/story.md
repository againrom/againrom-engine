# The second game's shown shop order

## Intent

Town 1, Kaarg and the Druids' town show each shop category in the client's
order. Plain equipment cells show the record price. Purchases and sales keep
their server values. The first game's stock, prices and save state stay
unchanged. No shop stock enters a SAV.

## Authority

Owner direction requires the original's shown order on EN and RU and the
first game's behaviour where a claim leaves a question open. The snapshot
moves from k406 to k408, `d76ef23`.

R2-ENGINE-798 supplies flag bit 0x2 set first, kind ascending, the weapon row
group ascending and shown price ascending. It binds tab k to category k.
R2-ASSET-207 supplies the packet layout and equal EN/RU flag and price data.
R2-ENGINE-799 supplies the CRT equal-key permutations and the Unknown server
stack slots for MagicItems lists. R2-ENGINE-800 distinguishes the reached
enchanted fill from the sibling draw and gives the town-1 shown groups.
R2-ENGINE-801 gives Kaarg's NewGame populations and shown groups.

R2-ENGINE-710 supplies the server fill keys and the unsorted merge or append
on Sale and Undo. R2-ENGINE-267 supplies settlement arithmetic. SHOP-154 has
the same client-comparator shape; DIV-4164 retains stable ROM1 ties.
R2-ENGINE-264/265/327/711 were read to narrow the remaining draw questions.
These are claim reads, not observations of an original runtime.

## As-built behaviour

| Kind | One builder |
|---|---|
| Shown shelf order | `shopstock.ShelfOrder`: flag, kind, weapon group, price and equal-key policy are profile data. `shopSortShelf` supplies the unchanged ROM1 sources. `secondshop.Tables` supplies packet flags, ID fields, shown prices and the CRT policy. |
| Server fill sort | `Tables.sortFill` sorts equipment with the server keys and CRT permutations. It is retained because equal client keys depend on server order. A MagicItems list keeps its draw order under DIV-4328. |
| Stock fill | `shopstock.Generate` over the existing `shopstock.Profile`; no new stock producer. |
| Packet decoder | `pkg/formats/itempkt.Parse`, a standard-library-only leaf. `LoadDefinitionsFor` reads the bytes at the address in `gameFiles`; ROM1 names no packet address. |
| Shelf copy and selection | `shopDeal.shelf` builds the shown copy. Its order permutation also maps a selection back to `removeFromShelf`. The mutation clone copies raw stock through the same shelf-copy builder with no order. Server lists remain the merge and Undo authority. |
| Cell price | The existing shelf/table/pack cell builders use the shop binding's optional shown-price source. `pkg/ui` still paints the same cells. |
| Book template key | The existing `itemTemplateCode` resolves a book's packet flag source; no second template lookup. |
| Headless dump | `cmd/shopdump -stock -seed 1` uses the installed profile, NewGame records, shared fill and shown-order builder for towns 1 and 2. |

Equipment without effects uses the last packet tag-1 value. Equipment with
effects uses server P. MagicItems use the loaded MagicItems price; books use
their server price with their teach effect. A missing packet key has no flag
bit and keeps server P. A malformed or absent required packet refuses the
ROM2 definition load. Mods can supply records without a shipped-count limit.

The shelf cell shows P, a sale-side cell ceil(P/2), with P the shown price.
Purchases charge quantity times server P. Sales credit
floor(quantity times server P / 2 + 0.5), as before (R2-ENGINE-267).
Pending totals retain server P; DIV-4452 names this remaining presentation
difference. Affordability and settlement still use server P.

The packet parser skips nine opaque header bytes and walks seven-byte record
prefixes plus their length-bounded attribute bytes. It retains the raw flag,
the last u32 price and whether a price occurred. Unknown tags consume one
opaque byte. Header meaning, prefix bytes +2/+3/+5, other flag meanings and
non-price attributes are not decoded. Duplicate IDs keep the last record.
There is no allocation from an installed count and no fixed 491-record cap.
Short headers, prefixes, payloads and attributes return bounded errors.

## Proof

Test-only commits precede the implementation. The initial decoder tests fail
to compile before the API exists. The two tests in `clientkeys_test.go` also
run against an archive of base `63ac65e4a`: kind/price order is `[2 1 3]`
instead of `[2 3 1]`, and two equal equipment items stay `[1 2]` instead of
the CRT's `[2 1]`. The same assertions pass after the implementation.

Focused tests separate every client key, the CRT cutoff at 8/9, mixed-key
partitions, stable Unknown MagicItems ties, packet truncation, book template
flags, MagicItems prices, and cell prices with and without effects. A page
test clicks the shown item, checks the server index, then checks Undo and a
mutation clone. Existing ROM1 tests are `TestShelfRunsCheapestFirstInsideEachGroup`,
`TestShelfSortIsStableOnEqualPrices`, `TestShelfSortWithNoTableChangesNothing`,
`TestReturnedItemLandsInOrder` and `TestArmourShelfOpensOnShields`.
`TestShopBindingRecord` retains the bindings' frozen output.

`TestReleaseSecondShopStockMatchesMain` retains its frozen digest through the
old stable server-order projection. It also compares current production fill
contents and the next draw against that projection before hashing. Its order
projection is test data; the new order is checked by the planted-list tests.
The release test is not run in this lane.

The static next-release inventory in `release-tests.txt` names 128 tests for
EN and RU. None ran here. The notes' RESULT.md records exact gate receipts.
The requested `-run DAG` selector selects no tests in this tree; the live DAG
test and the new cheap shelf-builder guard supply the actual checks.

The pre-reconciliation fast gate took 50.1 seconds. Four path-boundary tests were denied by the
sandbox. Two old caption/deal assertions also assumed the former shelf order;
they were updated and checked separately. The seat then required merging the
already-fetched `origin/main` at `6469d619b` and one fast gate on that merged
head. RESULT.md holds that receipt. Denied tests are never retried individually;
any remaining denied gate work belongs to the seat outside the sandbox.

The correction pass took the witness. `shopdump -stock -seed 1` on rom2-en and rom2-ru gives
250 equal rows. Against the pre-correction witness one pair differs (town 1, tab 0, rows 14 and
15, items 5642 and 5673); both rows are equal on all four client keys (flag 2, kind 6, price 113),
so the change is a reorder of tied rows. The CRT partition takes its pivot at lo + size/2 with
size = hi-lo+1 (R2-ENGINE-799), so n equal items exchange positions 0 and n/2 for n >= 9,
including n = 10, 12 and 14. The storyguard counters are unchanged by the correction (total
comment bytes 10480940); no baseline moved.

## Open debt

DIV-4328 narrows to MagicItems equal-key order. DIV-3042 narrows to the
phase-two chooser. DIV-4449 holds the potion draw form, DIV-4450 the enchant
random helper forms, DIV-4451 the RU client body and DIV-4452 pending totals.
All four reserved new IDs are used. No row claims an unread original body.

The cell and order changes introduce no simulation field, item field or SAV
writer. Stock remains transient and is redrawn after LOAD. Native installed
proof and the next release's EN/RU tests remain outstanding.
