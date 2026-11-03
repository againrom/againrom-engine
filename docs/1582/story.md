# The second game's inn Sleep

## Intent

Sleep restocks the current town's shop in each of the three second-game inns.
It keeps the party, gold, campaign, room and selected character. A SAV carries
no shop stock, before or after Sleep; a town LOAD draws the stock anew
(DIV-3045). The first game's tavern behaviour stays unchanged.

## Authority

Sleep is owner direction, recorded in DIV-4440. The first
game's existing Sleep rule is DIV-483, DIV-1647 and DIV-1763. This is an added
control: R2-ENGINE-246 describes the selected original inn's three controls.
R2-ENGINE-265 (High) supplies the second game's clear-fill stock rule.

The pinned snapshot is k406, c6410e6d8023e4e5a695f9ec0761185c2159493b.
R2-ENGINE-388 (Medium) gives the EN sheet's (8,8,8) shadow at (1,1). The engine
applies the one generator card profile on both roots; the claim leaves the RU
sheet binding Unknown. R2-ENGINE-748 and R2-ENGINE-749 give the offer gates
already adopted with the shared hero selector in story 1572.

The remaining inn questions were searched in the pinned ROM2 claim indexes,
then the relevant claims were read individually. R2-ENGINE-051 and
R2-ENGINE-220 narrow DIV-2635's dialogue-part uncertainty without changing
the existing dialogue consumer. R2-ENGINE-685 narrows DIV-2964's page+0x110
writer for Kaarg. R2-ENGINE-766 and R2-ENGINE-767 narrow the update and slot
questions behind DIV-4086, without establishing the missing sheet/shop
receiver joins. DIV-3109, DIV-4081, DIV-4083 and DIV-4087 remain unchanged.
The per-row search result is in the lane's RESULT.md.

## As-built behaviour

| Kind | One builder and the changed input |
|---|---|
| Inn control | Each town description adds one button to its existing RoomSpec.Buttons strip. townRoomView.Paint uses DrawRoomButtons; ControlAt resolves the added button to the existing row dispatch, townScreen.innAct. secondCampaign.innRows adds the corresponding row before Exit. The original three button positions and all seat indices stay unchanged. |
| Control word | ButtonSpec.Word is an optional engine-word ID, mutually exclusive with Caption. The existing button builder resolves it through Words.Engine. Sleep uses tavern.sleep, the same word as the first game's tavern, on both languages. |
| Shop stock | secondShop.restock clears and fills all four shelves through the existing Tables.Fill stock rule. It keeps the current town's assortment record and ceiling. There is no additional stock generator. |
| Shop random draw | The existing random.ShopStock stream supplies Raw, as on arrival. No new generator or random stream is introduced. |
| Statistics-card text | ChargenPresentation.CardView already reads the ROM2 generator profile's card-shadow. The patron and hero-pane callers use it, and composeItems paints it. No new painter or profile branch was needed. DIV-4082 was stale and moves to the closed ledger. |

The visible Sleep plaque is below Exit at (484,185)-(624,231), inside the
existing 160x238 button panel. It reuses button/2's plaque art. A list row
alone would not appear on the composed inn page. Both the plaque and fallback
row reach innAct; neither selects a seat or requests a sound or message.

The shop record is unchanged: assortment records and an older hero-list
compatibility field. A SAV written after Sleep carries no stock, stock
location or stream cursor, as every other second-game SAV (DIV-3045, owner
ruling: shop stock is never stored in a SAV). The shelves a LOAD shows are
drawn anew from the shop stream.

## Proof

The test-first commit c7aa554ec1386e1b94d15d11f7fd34325089e066 fails with
"the inn lists no Sleep control" in all three towns. The tests then gained
valid synthetic item tables and a starting-hero fixture for the complete SAV
producer and LOAD route.

- TestSecondInnSleepUsesTheStockStreamAndChangesOnlyShelves compares each of
  two Sleeps, in all three towns, with an independent service's next Tables.Fill
  result and next raw draw. It checks party, gold, bank, room, destination,
  assortment records and selected hero.
- TestSecondInnSleepSAVCarriesNoStock captures through Snapshot, writes a SAV
  with ExportCurrentSave, checks that the shop record holds Records alone, and
  calls RestoreOriginal on a cold front with a different seed and no party or
  town. The restored shelves equal the first draw of the shop stream, as a LOAD
  without Sleep draws, and differ from the slept stock.
- TestSecondInnSleepButtonAndCardProfiles checks each town's fourth plaque,
  both language word books, and the patron and hero-pane shadow profile.
- TestCardTextDrawsOverItsShadow checks the shared painter's pixels and the
  first game's unshadowed profile. TestTavernSleepOnlyRestocksTheShop keeps
  the first game's two deterministic restocks and unchanged non-shop state.

cmd/innsleep is the read-only installed-data witness. It visits town 1's shop
before Sleep and after two native-coordinate plaque clicks. It enumerates
the existing shop grid, printing counts and the first three names per shelf.
Installed witness, read-only, on the merged tree: town 1's shop has six rows
on each of its four shelves before Sleep, after one Sleep and after two, on EN
and on RU. First rows before Sleep, EN: shelf 1 Iron Buckler, Bronze Small
Shield, Bronze Small Shield; shelf 2 Bronze Dagger, Bronze Dagger, Bronze Short
Sword; shelf 3 Wood Staff x3; shelf 4 Potion of Health Regeneration, Potion of
Mana Regeneration, Scroll of Heal. After one Sleep shelves 1, 2 and 4 change
(Bronze Buckler first; Iron Dagger third; Scroll of Stone Missile); after two
they change again (Scroll of Slow on shelf 4). RU shows the same stock with the
RU names. The tool first failed to leave the shop on its second visit (it
dropped the town action its shop click returns); it now leaves by the pointer
on the Exit button. check-preserved-installs.sh: ok, 691 files as recorded.

The next release must run the existing ROM2 inn, shop, card and town-SAV
release tests on EN and RU. No release test is run in this lane.

## Open debt

DIV-3109, DIV-4081, DIV-4083 and DIV-4087 remain open unchanged. DIV-2964,
DIV-2635 and DIV-4086 remain open with narrower research questions.
DIV-4440 is accepted owner direction; DIV-4441 through DIV-4445 are unused.

Installed EN/RU witnesses and release tests remain for the seat. No original
runtime was launched, no asset was written and no native reload is claimed.
