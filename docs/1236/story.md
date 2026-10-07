# City SAVE settles pending shop goods

## Intent and authority

A city SAVE conserves player goods and gold when the shop table contains an unsettled trade. Owner direction: return goods to their actual pretrade owners without implicit payment, then write a resolved city through the common SAV producer. Changing the selected hero must not change ownership. ROM1's SAVE-time table policy is Unknown; DIV-1443 records this owner-directed boundary.

## As built

Pack and equipment staging record the actual owner and quantity. Table merges retain the destination identity and combine ownership receipts. Partial transfers consume receipts in order. Existing shared object aliases remain shared.

Snapshot captures a deep copy of pending goods and receipts. ExportCurrentSave resolves that captured city on a detached copy before ordinary SAV projection. Goods return to their owners without payment; merchant goods return to merchant stock. The live table remains pending after SAVE, cancellation or failure, and can still complete Buy, Sell or Clear. Cold LOAD opens a resolved city.

A mixed-owner merged place is split by the existing typed split constructor. The first owner retains the current merged identity and child graph; other owners receive fresh identities and independent children. Previously retired merge inputs are never reconstructed. Separate pack removals are restored in reverse table order to retain their insertion positions. No table presentation state is added to the SAV wire.

## Proof

The pre-fix focused run failed: `^TestShopSave`, package execution 0.173 seconds, bounded receipt exit 1 with no remaining processes. The final 93-test shop regression run passed in 0.560 seconds without skips. Controls cover whole graphs, changed selection, partial stacks, merged owners, equipment, fresh city topology, aliases, repeated exports, live Buy/Sell/Clear, unchanged merchant identity floors and no added zero-capacity load-refresh refusal. Ordinary Inventory references are checked before cold continuation. Storyguard and the release-test population guard passed.

`TestReleaseShopStagedGoodsF2CurrentSAV` passed on both lawful installs without skips: EN package 1.76 seconds and RU package 1.75 seconds. It drives the first native city SAVE with a staged item, then pack drag, F2, two named SAV writes, cold LOAD and a next shop action against current and original city sessions. It also checks live table retention and cancellation. The imported city fixture is `2026-08-15/game0010.sav`, SHA-256 `89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4`. No original runtime or screen observation is claimed.

Local evidence: `review/story1236-shop-save/final-receipt.json`, `final-regression.jsonl`, `final-guards.jsonl`, `final-en.jsonl`, `final-ru.jsonl`. Code/test tree: `67b28089e46b37b48164ff67a9da358dc12e7e70`, reconciled with main `4d169830`. The serial bounded run took 70.815 seconds including compilation, used four CPUs and left no child processes. The player result is that staged goods survive city SAVE and return to their owners on LOAD. The script-gap census is outside this shop slice.

## Open debt

Original-runtime SAVE-time shop behavior remains Unknown. The cold shop retains existing stock generation; this slice adds no stock persistence or table presentation encoding. The story adds no third save point, alternate save format, or new refusal for a legitimate city SAVE. Legacy reads remain unchanged. Sole review and final merge gates remain the landing authority.
