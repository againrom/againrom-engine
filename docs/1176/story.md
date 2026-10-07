# 1176 — campaign documents and terminal recovery

Fresh Documents survive mixed shop sales. The item retains its source price
of -1; the sale keeps non-positive-price goods on the tray and preserves
ordinary positive-price accounting.

## Reproduced defect

The EN witness generated a fresh hero, entered mission 10, completed the installed
objective scripts in 10 and 20, acknowledged Victory and returned to town.
The hero's physical `0x0e1c` item reached the shop with price -1. Selling it
beside a priced armour item removed both items and changed gold from 1100 to 1175.
`Shop.Sell` retained only price 0 goods. Its existing synthetic document control
used price 0 and did not cover the fresh item.

The production change is the sale's per-place eligibility check. It preserves
the item and price. Fresh grant, campaign progression, rewards and terminal
routing are unchanged.

## Authority

- `DAT-DOC-021` identifies `MagicItems[28]`, code `0x0e1c`, and source price -1.
  The unconditional fresh grant remains owner direction under `DIV-305`.
- `REG-SCN-097` establishes text tuples `(1,1),(2,1),(3,1)` at mission 10,
  text `(4,1)` at 50 and picture `(1,0)` at 60. The pair-deduplicating append is
  High; its forward-only loader guard remains Medium.
- `MISSION-DOC-021` separates that campaign collection from the physical
  access item and names its resources and saved tuple representation.
- `SHOP-SELL-010` supplies the ordinary stack payout calculation. Retaining
  the negative-price physical item is the owner-directed no-loss result;
  this witness does not establish the original runtime's negative-price sale.
- `REG-SCN-063` names 150 as the only shipped `LastMission=1` section.
  `SAV-890` additionally requires selected-main routing, divisible by 10.
  `SAV-970/971` describe conditional credits/FAME/reset dispatch; natural
  arrival and unconditional completion of that callback chain remain Unknown.

The lane uses public knowledge snapshot 37,
`c4073aef7e22bbc849b0efdd2a04a13047e37ae5`.

## Proof

`TestReleaseDocuments1176ActualGrantsReturnShopColdSave` follows one fresh
physical item through actual mission entry, the installed 10/20 objective
scripts, automatic 10-to-20 entry, Victory acknowledgment and town return.
The shop branch drives the ordinary room, equipment and sale controls, then
cold-loads AGS and city SAV. Standalone and mixed sales retain the item;
positive-price payout remains exact.

Only earlier wins 30/40 are seeded for later document grants. Missions 50/60
enter through actual building offers and `MissionOpenerWith`; existing 1060
death/teleport actions accelerate their objective conditions. App frames
execute each installed script and display Victory. These are controlled
prerequisites and objective actions, not an unassisted campaign playthrough.

The item opens the real document panel by its double-click equip gesture.
The collection grows 3/4/5 while the physical count remains one. Ordinary SAVE
and a fresh FrontEnd/App LOAD preserve each AGS stage. Two successive city
SAVs are independently read for physical items and campaign tuples, then
mission 70 remains usable. Separate source controls remove one page or the
physical item from the emitted SAV; LOAD and next mission entry retain each
absence without rebuilding it from the other carrier.

`TestReleaseCampaign1176TerminalAndSideRecovery` seeds earlier victories,
takes actual offers and enters 150 or 151. Unit 188's death for 150 and player 3/4
deaths for 151 reach installed script Victory through App frames. Actual AGS
files capture pending Victory and settled town. Fresh App LOAD preserves the
pending acknowledgment, completion and purse. Repeated acknowledgment does
not pay again. The town and main menu work after both routes. After 151,
main 150 remains unfinished, available and enterable after cold LOAD.

`TestShop1176UnpricedDocumentsSurviveMixedAndSingleSale` keeps the existing
price 0 arm, adds price -1 and checks the ordinary stack of three at price 5:
display 9, payout 8, all three goods returned to their shelf.

The focused EN and RU witnesses, existing price 0 control, new shop controls,
and release-population scanner pass. Both release witnesses are registered in
`population.txt`. The seat runs the
final Go, asset and EN/RU release chain after the sole adversarial pass.
Local evidence is under `review/story1176`, outside every repository.

## Open debt

No crash, hang or lost completion was reproduced in the measured 150/151 App
recovery routes. Current 150 recovery uses town and AGS. It does not implement
the conditional original credits/FAME/reset chain. Mission 151 is an ordinary
side mission, not a second ending. No terminal reward is inferred from the
absence of a successor or pinned as original behaviour by these tests.

Native city SAV at chapter 0 retains the existing `DIV-906` refusal. This story
does not invent a completed-campaign original city record or run the original
game. No lawful install or owner save is written.
