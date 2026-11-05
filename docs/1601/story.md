# One shop state in both games

## Intent

Keep one location-indexed shop state, in the second game's shape. Preserve both
games' existing stock, random draws, items, inventories and SAV mappings. Apply
the three explicit owner changes separately.

## Authority

The owner accepted all nine decisions of the shop-parity table: the first game
is a reduced second game represented by profile data. The table's S01-S40 and
the implementation brief define this scope. Existing generation and native
claims retain their stated confidence; fixture equality is engine evidence.
The corrected and closed implementation differences are DIV-3044, DIV-3065,
DIV-4440 and DIV-4452. No new claim or divergence ID is introduced.

## As-built behaviour

`Shop` and `Shop.rules` in `pkg/game/shop.go` replace the two state types and
two rules methods. `newShopState` builds the state. `shopRecords` maps catalog
IDs to assortment records without allocating their numeric range. Chapter
arrival derives records from the live ceiling and town catalog. Scenario
departure, stage and druid-unlock patches are profile rows; the chapter profile
has none. `nativeRecords` and `shopRecordRows` keep the four-record SAV mapping
separate from the state population. Neither stock nor a pending presentation
is persisted. The existing `shopDeal.settle` returns unpaid player goods only
in the detached capture, through the existing party and SAV producers.

| Rules | Profile field or policy row |
|---|---|
| S03-S05, S08-S09 | `fill`, `rawStream`, `ordinalSeed`: existing `stockProfile` and `secondshop.Tables.Profile`, both using `shopstock.Generate`; candidate order, budgets, book/scroll recipes, enchant scoring and streams stay unchanged |
| S10-S13 | `fillOnArrival`, `refillOnEntry`, `undoOnEntry`, `blockRestock`, `resetCategoryOnFill`, `derivedRecords`, `departures`, `stages`, `unlocks` |
| S16-S17, S30-S32, S35, S37-S38, S40 | `binding` selects the existing `bindShop` row: shown/offer prices, inventory and codec, pack bound, item forms, mod policy and speech. `fill` keeps each generation and mod-stock recipe. Native record conversion stays with the existing SAV codecs; town-layer validation is unchanged |
| S20-S21, S24-S28 | `wholeBasket`, `unpricedBack`, `keepEmptied`, `fallback`, `split`, `home`, `shelve`, `put`, `order`, `join` |
| Shop visibility and mission visits | `shown`, `missionVisits`; callers read the state profile |

Both profiles now refuse an unaffordable purchase whole and refuse Sleep
restock while any trade place remains. Other refill events are preserved;
this does not add unconditional ROM2 town-arrival refill. A successful ROM2
Sleep still selects category zero. Pending buy, sale and exit totals use the
cells' shown-price source through the existing deal sums. Settlement retains
server prices and per-place sale rounding.

The state holds no inventory copy. ROM1 retains member packs, owner provenance,
rich item identity and weapon-Spell policy. ROM2 retains the hero bag, its codec
and the 8192-unit bound. `bindShop` and `shopController` remain the presentation
owners. Mod stocking, chapter-town layers and trader speech keep their existing
policies. There is no new import edge.

## Proof

The first commit records synthetic stock, quantities, prices, order, next stock
draws, zero through three Sleeps, chapter ceilings, departure/stage changes,
mod stock, buys, sales, unpriced sales, single-place return, Undo, Exit, owners
and actual SAV digests in `TestShopStateRecord`. The fixture remains byte equal
after the refactor and the owner changes. Reversing the recorded first-game
armour shelf fails it: expected SHA256
`341a033c660fa81672cb53431fef064fc9deabe403691676f1793c2227dba7c4`.
The actual native shop record bytes are covered by the SAV digest; no install
or owner save supplies the fixture.

`TestShopStreamStateRecord` supplements the initial next-draw witness with
54 full-state observations: seeded Go-source scalar state hashes and exact
shared MSVC words, in both random modes, at arrival, entry, three Sleeps,
chapter change and Scenario record writes/re-entry. Its fixture is acquired
by running only that ordinary test on a `git archive` of `d46f4bbf1`, whose
production code equals the base, then compared unchanged on the candidate.
The stream fixture SHA256 is
`fb994e211fc1694f64a51445b9e02f32ff7ac8938e0f3fb838880b46715c0eb5`.
This supplemental recording occurs after implementation, rather than in the
initial record commit; the immutable baseline remains independently inspectable.

Each owner change has a failing-before witness: `TestShopBuyRefusesWholeBasket`,
`TestShopRestockKeepsOpenDeal` and
`TestShopPendingTotalsUseShownPricesAndSettleServerPrices`. Existing prefix-buy
assertions in `TestShopDealOneCoreBothGames` and
`TestSecondShopPageKeepsUnstackableLotsApart` now expect whole refusal. All
other behavioural expectations remain unchanged. Fixture-only adaptations
replace the removed type, adapt record containers, name transient location
state and explicitly configure a previously zero-initialized campaign shop.

`TestShopStateCatalogRecordsAndEmptyUpdates` covers a catalog ID beyond the
native mapping, ID `1<<30`, empty chapter patch lists and detached record
ownership. `TestShopRestockResetsTheBankCategory` preserves the old refill reset.
The cheap `ShopStateSites` guard checks state/rules shapes and caller game
selectors. Eleven planted shapes name their source. Its named non-state entries
are the subordinate `shopDeal`, `currentSecondShop` wire codec and
`originalCityTrade` source-sale comparison snapshot.

The final production focused selection from the brief passes in 37.498 s with
exactly the 17 already-denied base editor/mod operations excluded. Stock-package,
architecture/DAG, storyguard and divergence checks pass; the build and vet pass.
The canonical source census falls from four lines in two files to two lines in
one file. No release test runs in this lane. The next release must run the
complete `internal/gatedtests/testdata/population.txt` in release mode on EN/RU
of both games: this includes every shop, buy, sale, Sleep and town SAV witness.

## Open debt

The base broad focused run receives `Access is denied` in 17 editor/mod tests.
Those operations are not repeated or moved to an archive copy. The fast gate
would repeat them, so it is not run here; an unchanged fast gate outside the
sandbox and the denied base comparison remain seat-owned. Native and installed
parity, including the unread original stock serializer and existing pricing,
sort and speech Unknowns, are not established by this synthetic record.

No live nonempty deal at an inn was demonstrated, so restock has no changelog
entry. Sparse-record and remaining caller corrections are additional state
commits after the first owner change, departing from the requested commit order.
Versions and resources are left for landing. Publication and the sole fresh
adversarial review are seat-owned.
