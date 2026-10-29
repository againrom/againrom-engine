# One town room tip lifecycle for both games

## Intent

The owner requires the screens and components both games have to go one
way, in the second game's shape, with the first game as profile data. Town
pages go in order: the town host, the tavern or inn page, the inn offers,
the hero pane, the talk box, then the room tip. Before this story the one
tip builder `roomTipBuilder` (`pkg/game/tips.go`) had two lifecycles. The
first game kept a text and a closed flag per room (`townTip`, `tavernTip`,
`shopTip`, `schoolTip`, `tipClosed`), a revision counter and a shop second
text latch (`shopTipSecond`), driven by `loadTip`, `rereadTip`, `CloseTip`
and `advanceShopSecondTip`. The second game kept `secondRoomTip`
(`pkg/game/secondtowntip.go`), rebuilt at each room entry, behind the town
host's `tip-lifecycle` field. A fix to when a tip opens, rereads or closes
reached one game only.

After this story one state, `roomTip` in `pkg/game/tips.go`, carries the
popup of both games through one entry build, reread, close, view, arrival
and reset path. The game differences are tip table data. Every captured
frame, state line and SAV of both games is byte-equal to its base.

## Authority

- Owner rulings: one builder per kind (`AGENTS.md`); the first game is a
  reduced second game (`pipeline/OWNER-RULINGS.md`); screens both games
  have go one way, town pages in order, the room tip after the talk box.
- Census `cross-game-kinds-8b7ee0e5`, rank 13: one tip builder with the two
  lifecycles above.
- TOWN-516: the town, tavern, school and shop build their popup at every
  room enter while `TipsMode` is set, with no latch. TOWN-517: the shop's
  `shop2` replaces `shop1` in the open popup at most once per shop
  activation, without reading `TipsMode`.
- R2-ENGINE-308: second-game squares, inns and shops build their popups at
  every entry; `#tips4` replaces `#tips3` once per shop entry.
  R2-ENGINE-310: no show-once latch; Close and the room leave delete the
  panel; the shop latch resets at each entry.
- DIV-4063 (chosen): the second game's game menu Return builds no popup; the
  shown room builds one when the composed town is shown after hidden or
  under another campaign. DIV-2968: the second game shows no `#tips4`.

## As built

- Kind: room tip lifecycle. Builder: `roomTip` and its methods in
  `pkg/game/tips.go`. `roomTip` holds `of` (the second-game campaign it was
  built in, nil in the first game), `room`, `text`, `closed`, `revision`,
  `away` (the composed town is hidden) and `second` (the row's second text
  is shown).
  - Entry: `loadRoomTip(room)` replaces the popup at every tipped room's
    enter and reads the room's row text once while the tips option is on.
  - Reread: `rereadTip(room)` retexts the built popup at a talk's end and
    keeps its closed state, revision and second latch.
  - Close: `CloseTip` hides the popup until the next build.
  - View: `tipShown(room)` and `roomTipView(room)` show the current room's
    popup under the current campaign.
  - Arrival: `arriveRoomTip` builds the shown room's popup after the town
    was hidden or under another campaign.
  - Reset: `resetRoomTip` at a new or loaded game keeps the room, text and
    revision and drops the campaign, closed state and latches.
  - Shop second text: `advanceShopSecondTip` retexts the popup once per
    build with the row's `second` text while the shop's table holds an
    item, in both games; a row without a second text does nothing.
- Kind: room tip builder. Builder: `roomTipBuilder`, reused unchanged.
- Kind: tip rows. Builder: `tips/rom1.json` and `tips/rom2.json`, data
  only. The tip table gains `rooms` (`ui.TipRooms`), the game's lifecycle
  rules:
  - `resume`: the game menu's Return builds the shown room's popup again
    (first game `true`, second game `false`, DIV-4063);
  - `construct`: the room built when the town screen is made (first game
    `square`, second game none); the table check refuses a name no room
    row carries;
  - `entry-paint`: the popup's revision is published as the square's entry
    repaint (first game `true`; the second game publishes 0).
  The shop's second text stays a row field: the first game's shop row
  carries it, the second game's carries none (DIV-2968).
- Kind: town SAVE producer. Unchanged; the popup is not saved.
- The town host's `tip-lifecycle` field is removed from `town.HostSpec`, its
  check and the four town descriptions. The field was not documented for
  mods and the tip tables are not moddable, so no CHANGELOG line.
- The dead `townScreen.tipArt` field is removed.
- Guard: `internal/archtest` `TestTownRoomTipHasOneLifecycle` parses the
  production sources of `pkg/game` and holds four facts: `townScreen` has
  one field whose name holds `tip`, and it is `tip roomTip`; `roomTip` is
  declared once, in `pkg/game/tips.go`; every assignment or increment of
  `.tip` or one of its parts is in `pkg/game/tips.go`; and no function of
  `pkg/game/tips.go` calls `hostRules`.

## Proof

### Tests

`pkg/game/tips_test.go` drives synthetic tip tables (`withTipTable`) and
the second game's synthetic town text:

- `TestResumeRuleDecidesTheMenuReturn`: in both games, a closed popup
  reopens at the menu's Return when the table's `resume` is set and stays
  closed when it is clear; each game's table flipped turns its answer.
- `TestSecondTipRowDecidesTheRetext`: in both games, a shop row with a
  second text retexts the popup once when the table holds an item, and a
  row without one does not; each game's row changed in the data turns its
  answer.
- `TestEntryPaintRulePublishesTheRevision`, `TestTownScreenConstructionBuildsTheRulesRoom`
  and `TestTipRoomsConstructNamesARoomRow`: the other two rules and their
  check.
- `TestEachEntryBuildsItsRoomsPopup`, `TestCloseTipHoldsUntilTheNextEntry`,
  `TestTownTipRereadKeepsTheBuild`, `TestEntryWithNoShippedTextShowsNothing`,
  `TestBuildWithTipsOffNeverReadsTheInstall`,
  `TestToggleTipsFlipsSuppressionNotTheOpenPanel`,
  `TestResetRoomTipForgetsTheCampaign` and
  `TestNilTownScreenTipMethodsDoNotPanic`: the one lifecycle's paths.

Loss control of the guard, each restored from the story commit
`1adf0eae`: a first-game-only `chapterTip` per-room state (a type, a
`townScreen` field and a writer of `t.tip.closed` in a new
`pkg/game/chaptertip.go`) fails the guard three ways (the extra field, the
write outside `tips.go`, the counts); a `hostRules()` branch in `CloseTip`
fails it once.

### Frames, state and SAV

Instrument: the game built at engine main `f7247810` (from `git archive`)
and at the story commit `1adf0eae`, both with `scripts/build-tools.sh`'s
`go build` flags and their revision stamps, run headlessly with
`--headless` scenarios (front-end stage v10) and `-seed 1`, each run on a
fresh profile with a private copy of one save. Inputs: the first game's EN
town kit saves `game9711` (engine-written, chapter 30), `game9714`
(engine-written, chapter 60, purse 5000) and `game9719` (the original's,
chapter 30), read by the `en` and `ru` installs; the second game's owner
files `game0000`, `game0002`, `game0003` and `game9998`, and a new game,
read by `rom2-en` and `rom2-ru`. The first game's new game starts in a
mission, so no first-game run starts from a new game in town. One plan per
save and state was built once with the candidate (dialogue dismissals
counted, a Close sent only where a panel shows) and run unchanged by both
binaries.

States, per save (7 runs a first-game save, 6 a second-game save):

- `square`: the square's popup after the load or new game, closed, and
  after the game menu's Return (Escape on the menu).
- `room-<room>` (tavern, shop, school; inn and shop): the room entered for
  the first time, its popup closed, the square after the leave, the room
  re-entered after the Close, left with its popup open, the square, and
  entered again.
- `talk`: the tavern's first talk (an NPC cell in the first game, a TALK
  row in the second) to its end with the popup open (the reread), the popup
  closed, and a second talk to its end.
- `shop2`: the shop entered, the selector's first shelf item put on the
  table (the first game's second tip), the Undo, the item again (no second
  retext), the Undo, the Close, the leave, the shop entered again (the first
  text), and the item again (the second text).
- `save`: the square's and the tavern's popups closed, the ordinary town
  SAVE named `tip proof`, its LOAD from the LOAD list, and the tavern
  entered after the LOAD.

Every run exits 0 on both sides (102 of 102). Rows: 516 frames, 16 saved
SAVs and 102 state lines (`machine.jsonl`, the run directory's path
replaced by a token), 634 of 634 equal. The town holds no World; the state
lines and the SAV are its hashed surfaces. The human logs (102) and the
copied input saves (48) are equal as well and not listed. The same plans
run at `origin/main` `db698044` and at the merge of this story with it are
784 of 784 equal, the logs and input copies included; so are the runs at
`origin/main` `eb45e6dc` (the town talk producer) and at the merge
`cd9ef318` of this story with it, 784 of 784, and the runs at
`origin/main` `a5c60f20` (the square view adapter) and at the merge
`1e4ebf24`, 784 of 784.

| state | root | base SHA-256 | head SHA-256 | equal |
|---|---|---|---|---|
| 9711-room-school frame school-closed | en | f2b581746127aa205c78b9f43a765cfc2d45ed72dd17b8474747f233dd9d6fb5 | f2b581746127aa205c78b9f43a765cfc2d45ed72dd17b8474747f233dd9d6fb5 | yes |
| 9711-room-school frame school-first | en | d9517d9fd715a3e93cfe8bfd2e85867996d63a4f8a22f6199075ec77589b8d03 | d9517d9fd715a3e93cfe8bfd2e85867996d63a4f8a22f6199075ec77589b8d03 | yes |
| 9711-room-school frame school-reentered-after-close | en | bcfa12ffbcf436f3cac9f07fc3cfb97ba10931f25cd2ca22ca70bc5932bf6316 | bcfa12ffbcf436f3cac9f07fc3cfb97ba10931f25cd2ca22ca70bc5932bf6316 | yes |
| 9711-room-school frame school-reentered-after-open | en | bcfa12ffbcf436f3cac9f07fc3cfb97ba10931f25cd2ca22ca70bc5932bf6316 | bcfa12ffbcf436f3cac9f07fc3cfb97ba10931f25cd2ca22ca70bc5932bf6316 | yes |
| 9711-room-school frame school-square-after-close | en | faf608acd814e436049554d01345d4130646b278a16d9c95eedad442dab78b5c | faf608acd814e436049554d01345d4130646b278a16d9c95eedad442dab78b5c | yes |
| 9711-room-school frame school-square-left-open | en | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | yes |
| 9711-room-school state lines | en | 45649d048e3dfb8876cd0c2072a62d092158cce43569531078cde4afb9ffdad9 | 45649d048e3dfb8876cd0c2072a62d092158cce43569531078cde4afb9ffdad9 | yes |
| 9711-room-shop frame shop-closed | en | ff98557c606a01c4622c0e31fcb70d96a515277dd2414f3e5c62034f529937ff | ff98557c606a01c4622c0e31fcb70d96a515277dd2414f3e5c62034f529937ff | yes |
| 9711-room-shop frame shop-first | en | 034001fa0c17681c2735b19d6ab1c3b9ecf832820543ffdd5e4e2a6d8a75edaa | 034001fa0c17681c2735b19d6ab1c3b9ecf832820543ffdd5e4e2a6d8a75edaa | yes |
| 9711-room-shop frame shop-reentered-after-close | en | 6e579077d2498f65dc0e39624e2b4638d43f2dbdfc19eb8be6cb75e3f7868e8d | 6e579077d2498f65dc0e39624e2b4638d43f2dbdfc19eb8be6cb75e3f7868e8d | yes |
| 9711-room-shop frame shop-reentered-after-open | en | 6e579077d2498f65dc0e39624e2b4638d43f2dbdfc19eb8be6cb75e3f7868e8d | 6e579077d2498f65dc0e39624e2b4638d43f2dbdfc19eb8be6cb75e3f7868e8d | yes |
| 9711-room-shop frame shop-square-after-close | en | 5db3bdb40b7368bc1ddd58d7701690f2c5a5155f747342f90b62d24a9bc38047 | 5db3bdb40b7368bc1ddd58d7701690f2c5a5155f747342f90b62d24a9bc38047 | yes |
| 9711-room-shop frame shop-square-left-open | en | a371106925492afe27a5e10ee82d9c350203bb9edae48fdd8c0ac19dda44c5ea | a371106925492afe27a5e10ee82d9c350203bb9edae48fdd8c0ac19dda44c5ea | yes |
| 9711-room-shop state lines | en | eab83d695fd516b9860abf3da1c2e1635ae351bb4219b3b42383e64313ae4425 | eab83d695fd516b9860abf3da1c2e1635ae351bb4219b3b42383e64313ae4425 | yes |
| 9711-room-tavern frame tavern-closed | en | c851fe7031c3ffbf0b340422cd6b0c04f659262a2003a746798dc7e16de99412 | c851fe7031c3ffbf0b340422cd6b0c04f659262a2003a746798dc7e16de99412 | yes |
| 9711-room-tavern frame tavern-first | en | f06937efc70ff8d6893877f8180ae98001b7966d4ee588fd79412b35fd0f33ef | f06937efc70ff8d6893877f8180ae98001b7966d4ee588fd79412b35fd0f33ef | yes |
| 9711-room-tavern frame tavern-reentered-after-close | en | 3dcbc21e8575542d4c7ccd1a5cf058e3d3d37a2a7e72429c841afed4040c4096 | 3dcbc21e8575542d4c7ccd1a5cf058e3d3d37a2a7e72429c841afed4040c4096 | yes |
| 9711-room-tavern frame tavern-reentered-after-open | en | 3dcbc21e8575542d4c7ccd1a5cf058e3d3d37a2a7e72429c841afed4040c4096 | 3dcbc21e8575542d4c7ccd1a5cf058e3d3d37a2a7e72429c841afed4040c4096 | yes |
| 9711-room-tavern frame tavern-square-after-close | en | faf608acd814e436049554d01345d4130646b278a16d9c95eedad442dab78b5c | faf608acd814e436049554d01345d4130646b278a16d9c95eedad442dab78b5c | yes |
| 9711-room-tavern frame tavern-square-left-open | en | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | yes |
| 9711-room-tavern state lines | en | 22cd729894df232235a70146e614095ca451dc8e86f1d02d48c4d8a1087d04aa | 22cd729894df232235a70146e614095ca451dc8e86f1d02d48c4d8a1087d04aa | yes |
| 9711-save frame save-loaded-tavern | en | 3dcbc21e8575542d4c7ccd1a5cf058e3d3d37a2a7e72429c841afed4040c4096 | 3dcbc21e8575542d4c7ccd1a5cf058e3d3d37a2a7e72429c841afed4040c4096 | yes |
| 9711-save frame save-loaded | en | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | yes |
| 9711-save frame save-saved | en | e195a907bbda08952dc1ce0493cfcd1fb6b09ae0f8c9895a17ddfadc2ed117d1 | e195a907bbda08952dc1ce0493cfcd1fb6b09ae0f8c9895a17ddfadc2ed117d1 | yes |
| 9711-save frame save-square | en | e64ea1112ffb68e6fbf107500cc1322dcec80a2885699915170e73575cf0d669 | e64ea1112ffb68e6fbf107500cc1322dcec80a2885699915170e73575cf0d669 | yes |
| 9711-save SAV tip proof.sav | en | af59b54981f55ee03507029a3c393ebdca038e93f59ba2214020f75c80902b60 | af59b54981f55ee03507029a3c393ebdca038e93f59ba2214020f75c80902b60 | yes |
| 9711-save state lines | en | 560768db25ef9c2d80c9171d3ccbc91d720ff0063e387e5cac6b2da87d88ad29 | 560768db25ef9c2d80c9171d3ccbc91d720ff0063e387e5cac6b2da87d88ad29 | yes |
| 9711-shop2 frame shop-closed | en | 3dd0996d1f005baa1ca4ab4d55383c4bab6c82a01504aa24d871e2aab569a49a | 3dd0996d1f005baa1ca4ab4d55383c4bab6c82a01504aa24d871e2aab569a49a | yes |
| 9711-shop2 frame shop-first | en | 034001fa0c17681c2735b19d6ab1c3b9ecf832820543ffdd5e4e2a6d8a75edaa | 034001fa0c17681c2735b19d6ab1c3b9ecf832820543ffdd5e4e2a6d8a75edaa | yes |
| 9711-shop2 frame shop-reentered-table | en | c3f24d0c21538200900b1f9af5f4a2cef1eacd65b4a8874191d941d25dab6fa5 | c3f24d0c21538200900b1f9af5f4a2cef1eacd65b4a8874191d941d25dab6fa5 | yes |
| 9711-shop2 frame shop-reentered | en | 6e579077d2498f65dc0e39624e2b4638d43f2dbdfc19eb8be6cb75e3f7868e8d | 6e579077d2498f65dc0e39624e2b4638d43f2dbdfc19eb8be6cb75e3f7868e8d | yes |
| 9711-shop2 frame shop-table-again | en | ba9f5d6bbdfeea4d0ac41754b79f4972140d3cd375de98ff7edb078f1e06f530 | ba9f5d6bbdfeea4d0ac41754b79f4972140d3cd375de98ff7edb078f1e06f530 | yes |
| 9711-shop2 frame shop-table | en | c3f24d0c21538200900b1f9af5f4a2cef1eacd65b4a8874191d941d25dab6fa5 | c3f24d0c21538200900b1f9af5f4a2cef1eacd65b4a8874191d941d25dab6fa5 | yes |
| 9711-shop2 frame shop-undo | en | 8f62123110030b853a85dbd92de82d1a1e37b58e61f2995c03fb1dde6b12df46 | 8f62123110030b853a85dbd92de82d1a1e37b58e61f2995c03fb1dde6b12df46 | yes |
| 9711-shop2 state lines | en | 9a75cfddaff37bdf34661939a6683b43dee4ab540965e91945d71350193dc590 | 9a75cfddaff37bdf34661939a6683b43dee4ab540965e91945d71350193dc590 | yes |
| 9711-square frame square-closed | en | f6affd12134ec5c4635a29f300cce98f7ae70088cdbbb1cd91a5b3e124b2f1f2 | f6affd12134ec5c4635a29f300cce98f7ae70088cdbbb1cd91a5b3e124b2f1f2 | yes |
| 9711-square frame square-load | en | 1bfc277a443c7370584020970b1b51cd4989e2292175aced250473e1aed85f64 | 1bfc277a443c7370584020970b1b51cd4989e2292175aced250473e1aed85f64 | yes |
| 9711-square frame square-return | en | 54181890c7dab0acd3044df7d7053099dc8fcafe43c6b554713d2c1af83c414f | 54181890c7dab0acd3044df7d7053099dc8fcafe43c6b554713d2c1af83c414f | yes |
| 9711-square state lines | en | 90da448d9c9cea7f22f2ad43df59733d873d070be755ac15367e8bcb842c2614 | 90da448d9c9cea7f22f2ad43df59733d873d070be755ac15367e8bcb842c2614 | yes |
| 9711-talk frame talk-closed | en | 2e6c69224e037959b1c4e71fb09d7861d630a7919c2c603fc05b7e4ac61523b4 | 2e6c69224e037959b1c4e71fb09d7861d630a7919c2c603fc05b7e4ac61523b4 | yes |
| 9711-talk frame talk-reread-closed | en | 2e6c69224e037959b1c4e71fb09d7861d630a7919c2c603fc05b7e4ac61523b4 | 2e6c69224e037959b1c4e71fb09d7861d630a7919c2c603fc05b7e4ac61523b4 | yes |
| 9711-talk frame talk-reread | en | f5ddf12b0ee8d094c4afc55e4004c0bf3108264ac38b762f59a9719030e1a91c | f5ddf12b0ee8d094c4afc55e4004c0bf3108264ac38b762f59a9719030e1a91c | yes |
| 9711-talk frame tavern-in | en | f06937efc70ff8d6893877f8180ae98001b7966d4ee588fd79412b35fd0f33ef | f06937efc70ff8d6893877f8180ae98001b7966d4ee588fd79412b35fd0f33ef | yes |
| 9711-talk state lines | en | e77788ce319af4da1aee7d33c74477fa273766102d9798742bdd0a82d8c7d34a | e77788ce319af4da1aee7d33c74477fa273766102d9798742bdd0a82d8c7d34a | yes |
| 9714-room-school frame school-closed | en | ddd091d6621e41901ab023ba2785ba559cc20100172151a1a12ce210ebe66d2e | ddd091d6621e41901ab023ba2785ba559cc20100172151a1a12ce210ebe66d2e | yes |
| 9714-room-school frame school-first | en | ea73c667ec8d60200919fc62d3ed123eec095b0308d2db5c5ee97d634953894c | ea73c667ec8d60200919fc62d3ed123eec095b0308d2db5c5ee97d634953894c | yes |
| 9714-room-school frame school-reentered-after-close | en | 53a5ba69b652847b620b9b7d51a86111ac06a6c5772dc0e7488c837fd7b04fd3 | 53a5ba69b652847b620b9b7d51a86111ac06a6c5772dc0e7488c837fd7b04fd3 | yes |
| 9714-room-school frame school-reentered-after-open | en | 53a5ba69b652847b620b9b7d51a86111ac06a6c5772dc0e7488c837fd7b04fd3 | 53a5ba69b652847b620b9b7d51a86111ac06a6c5772dc0e7488c837fd7b04fd3 | yes |
| 9714-room-school frame school-square-after-close | en | e64ea1112ffb68e6fbf107500cc1322dcec80a2885699915170e73575cf0d669 | e64ea1112ffb68e6fbf107500cc1322dcec80a2885699915170e73575cf0d669 | yes |
| 9714-room-school frame school-square-left-open | en | 0c48eaa66f5bffd6f1f2d5328c370757f10c7761e36320dbe0b9b1c5c8748523 | 0c48eaa66f5bffd6f1f2d5328c370757f10c7761e36320dbe0b9b1c5c8748523 | yes |
| 9714-room-school state lines | en | cad40f8b90f77192a4a3b391fd80c1da658fb1676a66a79437548c9f78f85565 | cad40f8b90f77192a4a3b391fd80c1da658fb1676a66a79437548c9f78f85565 | yes |
| 9714-room-shop frame shop-closed | en | 671a8faf0133e6d1d328b111fa1eaf2c85b1178be05a52dabd1deb06a7f7d86b | 671a8faf0133e6d1d328b111fa1eaf2c85b1178be05a52dabd1deb06a7f7d86b | yes |
| 9714-room-shop frame shop-first | en | 76fb1afa119e31bd1455660da23c0dc77543b1aae429daa4e6c8e2979d3deba8 | 76fb1afa119e31bd1455660da23c0dc77543b1aae429daa4e6c8e2979d3deba8 | yes |
| 9714-room-shop frame shop-reentered-after-close | en | 2aada39ab7310beb6d3ceb1e3bdff1a0935628c8e95ed46d34a541eb96b327b0 | 2aada39ab7310beb6d3ceb1e3bdff1a0935628c8e95ed46d34a541eb96b327b0 | yes |
| 9714-room-shop frame shop-reentered-after-open | en | 2aada39ab7310beb6d3ceb1e3bdff1a0935628c8e95ed46d34a541eb96b327b0 | 2aada39ab7310beb6d3ceb1e3bdff1a0935628c8e95ed46d34a541eb96b327b0 | yes |
| 9714-room-shop frame shop-square-after-close | en | 5db3bdb40b7368bc1ddd58d7701690f2c5a5155f747342f90b62d24a9bc38047 | 5db3bdb40b7368bc1ddd58d7701690f2c5a5155f747342f90b62d24a9bc38047 | yes |
| 9714-room-shop frame shop-square-left-open | en | a371106925492afe27a5e10ee82d9c350203bb9edae48fdd8c0ac19dda44c5ea | a371106925492afe27a5e10ee82d9c350203bb9edae48fdd8c0ac19dda44c5ea | yes |
| 9714-room-shop state lines | en | 7bc9682df9efc9ed9797fcaf0255166a05c29e14a93f1738b19c005d5c9ce4cb | 7bc9682df9efc9ed9797fcaf0255166a05c29e14a93f1738b19c005d5c9ce4cb | yes |
| 9714-room-tavern frame tavern-closed | en | 7c907c9ce42ffc6165da7f3a5f9b8172f319f765d9792326887eaf4eda8c0206 | 7c907c9ce42ffc6165da7f3a5f9b8172f319f765d9792326887eaf4eda8c0206 | yes |
| 9714-room-tavern frame tavern-first | en | 4f776cf2ee66cc48d309badb764f5a7804bcf903855e32704772a07c059129e3 | 4f776cf2ee66cc48d309badb764f5a7804bcf903855e32704772a07c059129e3 | yes |
| 9714-room-tavern frame tavern-reentered-after-close | en | 353c1e8a76e465147a843a06be082dd6199dd43717b7bf4416a9aec9838dc9ad | 353c1e8a76e465147a843a06be082dd6199dd43717b7bf4416a9aec9838dc9ad | yes |
| 9714-room-tavern frame tavern-reentered-after-open | en | 353c1e8a76e465147a843a06be082dd6199dd43717b7bf4416a9aec9838dc9ad | 353c1e8a76e465147a843a06be082dd6199dd43717b7bf4416a9aec9838dc9ad | yes |
| 9714-room-tavern frame tavern-square-after-close | en | faf608acd814e436049554d01345d4130646b278a16d9c95eedad442dab78b5c | faf608acd814e436049554d01345d4130646b278a16d9c95eedad442dab78b5c | yes |
| 9714-room-tavern frame tavern-square-left-open | en | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | yes |
| 9714-room-tavern state lines | en | 76de067c5192e7cafc16998f79e3d76cbe1a857fb8b1d88aef5874ed30deba18 | 76de067c5192e7cafc16998f79e3d76cbe1a857fb8b1d88aef5874ed30deba18 | yes |
| 9714-save frame save-loaded-tavern | en | 8fac5c9ebb21978073a3a487ebdf5477e91c419762d4d870c0698c58b22e2217 | 8fac5c9ebb21978073a3a487ebdf5477e91c419762d4d870c0698c58b22e2217 | yes |
| 9714-save frame save-loaded | en | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | yes |
| 9714-save frame save-saved | en | e195a907bbda08952dc1ce0493cfcd1fb6b09ae0f8c9895a17ddfadc2ed117d1 | e195a907bbda08952dc1ce0493cfcd1fb6b09ae0f8c9895a17ddfadc2ed117d1 | yes |
| 9714-save frame save-square | en | e64ea1112ffb68e6fbf107500cc1322dcec80a2885699915170e73575cf0d669 | e64ea1112ffb68e6fbf107500cc1322dcec80a2885699915170e73575cf0d669 | yes |
| 9714-save SAV tip proof.sav | en | 3fa2a334b5ad6bd862511c66c4dcbe174a864e54cf7fab22c0c473f7f410a67b | 3fa2a334b5ad6bd862511c66c4dcbe174a864e54cf7fab22c0c473f7f410a67b | yes |
| 9714-save state lines | en | 6357a8c815f1907b4b517ba3c762c0b0525c34ab5caa274bac9bec98fc84d338 | 6357a8c815f1907b4b517ba3c762c0b0525c34ab5caa274bac9bec98fc84d338 | yes |
| 9714-shop2 frame shop-closed | en | 029536d5e86c0c8610ad8f905448da88340726839463d0d3af252d477a842c07 | 029536d5e86c0c8610ad8f905448da88340726839463d0d3af252d477a842c07 | yes |
| 9714-shop2 frame shop-first | en | 76fb1afa119e31bd1455660da23c0dc77543b1aae429daa4e6c8e2979d3deba8 | 76fb1afa119e31bd1455660da23c0dc77543b1aae429daa4e6c8e2979d3deba8 | yes |
| 9714-shop2 frame shop-reentered-table | en | 65e0ee2e66c4d25d54a9d462c3918ed15527a99ee643980d1799dca07fda6810 | 65e0ee2e66c4d25d54a9d462c3918ed15527a99ee643980d1799dca07fda6810 | yes |
| 9714-shop2 frame shop-reentered | en | 2aada39ab7310beb6d3ceb1e3bdff1a0935628c8e95ed46d34a541eb96b327b0 | 2aada39ab7310beb6d3ceb1e3bdff1a0935628c8e95ed46d34a541eb96b327b0 | yes |
| 9714-shop2 frame shop-table-again | en | 65e0ee2e66c4d25d54a9d462c3918ed15527a99ee643980d1799dca07fda6810 | 65e0ee2e66c4d25d54a9d462c3918ed15527a99ee643980d1799dca07fda6810 | yes |
| 9714-shop2 frame shop-table | en | 65e0ee2e66c4d25d54a9d462c3918ed15527a99ee643980d1799dca07fda6810 | 65e0ee2e66c4d25d54a9d462c3918ed15527a99ee643980d1799dca07fda6810 | yes |
| 9714-shop2 frame shop-undo | en | 409eb2290c07621112fe43fc2e510a617604e693625024ad1401377870f14ad6 | 409eb2290c07621112fe43fc2e510a617604e693625024ad1401377870f14ad6 | yes |
| 9714-shop2 state lines | en | 9868e4895bd90e2a80c3c04b6559a1083cc6d1f258c08eff0c31241aab7da370 | 9868e4895bd90e2a80c3c04b6559a1083cc6d1f258c08eff0c31241aab7da370 | yes |
| 9714-square frame square-closed | en | f6affd12134ec5c4635a29f300cce98f7ae70088cdbbb1cd91a5b3e124b2f1f2 | f6affd12134ec5c4635a29f300cce98f7ae70088cdbbb1cd91a5b3e124b2f1f2 | yes |
| 9714-square frame square-load | en | 1bfc277a443c7370584020970b1b51cd4989e2292175aced250473e1aed85f64 | 1bfc277a443c7370584020970b1b51cd4989e2292175aced250473e1aed85f64 | yes |
| 9714-square frame square-return | en | 54181890c7dab0acd3044df7d7053099dc8fcafe43c6b554713d2c1af83c414f | 54181890c7dab0acd3044df7d7053099dc8fcafe43c6b554713d2c1af83c414f | yes |
| 9714-square state lines | en | 53d5df1fbfe58513b91559398abefe5e3ee06a574ed462797e283599dcb53d23 | 53d5df1fbfe58513b91559398abefe5e3ee06a574ed462797e283599dcb53d23 | yes |
| 9714-talk frame talk-closed | en | bbce9eb50c68bd800d974b5a9b8ca60882ce0765560cf5733b9dc134da158786 | bbce9eb50c68bd800d974b5a9b8ca60882ce0765560cf5733b9dc134da158786 | yes |
| 9714-talk frame talk-reread-closed | en | bbce9eb50c68bd800d974b5a9b8ca60882ce0765560cf5733b9dc134da158786 | bbce9eb50c68bd800d974b5a9b8ca60882ce0765560cf5733b9dc134da158786 | yes |
| 9714-talk frame talk-reread | en | 459f7b50cbc1db77f84b27444ad0a408914b9c9b4a55b4acc4ce884fa20653b7 | 459f7b50cbc1db77f84b27444ad0a408914b9c9b4a55b4acc4ce884fa20653b7 | yes |
| 9714-talk frame tavern-in | en | 4f776cf2ee66cc48d309badb764f5a7804bcf903855e32704772a07c059129e3 | 4f776cf2ee66cc48d309badb764f5a7804bcf903855e32704772a07c059129e3 | yes |
| 9714-talk state lines | en | 44743d4c62a5b0188bd3dc07fa495828ff5eda2e8459c1e2d4fa7d3d5f7fa261 | 44743d4c62a5b0188bd3dc07fa495828ff5eda2e8459c1e2d4fa7d3d5f7fa261 | yes |
| 9719-room-school frame school-closed | en | 8be3744cd2422ea3ae501f03e8d38ca95fc369b7bc56fb3c42a581f05fc3283e | 8be3744cd2422ea3ae501f03e8d38ca95fc369b7bc56fb3c42a581f05fc3283e | yes |
| 9719-room-school frame school-first | en | 0334a9df7dd7af3eb81722bdca87659b5a1080b4df11ef9b896a1d6b6e0163fe | 0334a9df7dd7af3eb81722bdca87659b5a1080b4df11ef9b896a1d6b6e0163fe | yes |
| 9719-room-school frame school-reentered-after-close | en | e0ff741ac8061444f8d23dbb6bec70fed89cf42fc979878257a7fc4ebe867b41 | e0ff741ac8061444f8d23dbb6bec70fed89cf42fc979878257a7fc4ebe867b41 | yes |
| 9719-room-school frame school-reentered-after-open | en | e0ff741ac8061444f8d23dbb6bec70fed89cf42fc979878257a7fc4ebe867b41 | e0ff741ac8061444f8d23dbb6bec70fed89cf42fc979878257a7fc4ebe867b41 | yes |
| 9719-room-school frame school-square-after-close | en | faf608acd814e436049554d01345d4130646b278a16d9c95eedad442dab78b5c | faf608acd814e436049554d01345d4130646b278a16d9c95eedad442dab78b5c | yes |
| 9719-room-school frame school-square-left-open | en | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | yes |
| 9719-room-school state lines | en | 587cf68657a4d4e0db8685860cc79db9f2624830f529f31b07ea062fc6f66770 | 587cf68657a4d4e0db8685860cc79db9f2624830f529f31b07ea062fc6f66770 | yes |
| 9719-room-shop frame shop-closed | en | 18c7e4b857b06b7414527011f78cc14214233fb0c70683b6b9ff2a89cbb5e60b | 18c7e4b857b06b7414527011f78cc14214233fb0c70683b6b9ff2a89cbb5e60b | yes |
| 9719-room-shop frame shop-first | en | 307f7f4ca7329b7f20566c131eb409beb89dfd575247282f9a2734405155d6da | 307f7f4ca7329b7f20566c131eb409beb89dfd575247282f9a2734405155d6da | yes |
| 9719-room-shop frame shop-reentered-after-close | en | ca7360aaffa530ded04322b866c9427b9203dd6be4376842555b1b2834486a47 | ca7360aaffa530ded04322b866c9427b9203dd6be4376842555b1b2834486a47 | yes |
| 9719-room-shop frame shop-reentered-after-open | en | ca7360aaffa530ded04322b866c9427b9203dd6be4376842555b1b2834486a47 | ca7360aaffa530ded04322b866c9427b9203dd6be4376842555b1b2834486a47 | yes |
| 9719-room-shop frame shop-square-after-close | en | 5db3bdb40b7368bc1ddd58d7701690f2c5a5155f747342f90b62d24a9bc38047 | 5db3bdb40b7368bc1ddd58d7701690f2c5a5155f747342f90b62d24a9bc38047 | yes |
| 9719-room-shop frame shop-square-left-open | en | a371106925492afe27a5e10ee82d9c350203bb9edae48fdd8c0ac19dda44c5ea | a371106925492afe27a5e10ee82d9c350203bb9edae48fdd8c0ac19dda44c5ea | yes |
| 9719-room-shop state lines | en | 2c67a21e6d99e57959b8eec10e137b6d7b29625fe381d80bb2f96301913408d7 | 2c67a21e6d99e57959b8eec10e137b6d7b29625fe381d80bb2f96301913408d7 | yes |
| 9719-room-tavern frame tavern-closed | en | 5df79293f3c00739d35d7934096f05fe8a392fd905f31f4980c556ac15120ad6 | 5df79293f3c00739d35d7934096f05fe8a392fd905f31f4980c556ac15120ad6 | yes |
| 9719-room-tavern frame tavern-first | en | 5ef23b0a66ec25e069368a4a0d998929da6f220b905578c788158e477cabbfd1 | 5ef23b0a66ec25e069368a4a0d998929da6f220b905578c788158e477cabbfd1 | yes |
| 9719-room-tavern frame tavern-reentered-after-close | en | 67dd422f52feaa27ad7760989fa014760658f024ac07ed36107c919d4ce59b02 | 67dd422f52feaa27ad7760989fa014760658f024ac07ed36107c919d4ce59b02 | yes |
| 9719-room-tavern frame tavern-reentered-after-open | en | 67dd422f52feaa27ad7760989fa014760658f024ac07ed36107c919d4ce59b02 | 67dd422f52feaa27ad7760989fa014760658f024ac07ed36107c919d4ce59b02 | yes |
| 9719-room-tavern frame tavern-square-after-close | en | faf608acd814e436049554d01345d4130646b278a16d9c95eedad442dab78b5c | faf608acd814e436049554d01345d4130646b278a16d9c95eedad442dab78b5c | yes |
| 9719-room-tavern frame tavern-square-left-open | en | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | yes |
| 9719-room-tavern state lines | en | 0a78ff3952103efa01d08c707edc26f6f614ace6e840c259a9ae01e7994bea12 | 0a78ff3952103efa01d08c707edc26f6f614ace6e840c259a9ae01e7994bea12 | yes |
| 9719-save frame save-loaded-tavern | en | 67dd422f52feaa27ad7760989fa014760658f024ac07ed36107c919d4ce59b02 | 67dd422f52feaa27ad7760989fa014760658f024ac07ed36107c919d4ce59b02 | yes |
| 9719-save frame save-loaded | en | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | 37e2fa6f9092a380621818961fc75d81662302088b5548da6c9c7a54d855693d | yes |
| 9719-save frame save-saved | en | e195a907bbda08952dc1ce0493cfcd1fb6b09ae0f8c9895a17ddfadc2ed117d1 | e195a907bbda08952dc1ce0493cfcd1fb6b09ae0f8c9895a17ddfadc2ed117d1 | yes |
| 9719-save frame save-square | en | e64ea1112ffb68e6fbf107500cc1322dcec80a2885699915170e73575cf0d669 | e64ea1112ffb68e6fbf107500cc1322dcec80a2885699915170e73575cf0d669 | yes |
| 9719-save SAV tip proof.sav | en | 3681d639f81726534a0a092e8d8e8f34b71ca07e95077d29dc19f4ea0464d540 | 3681d639f81726534a0a092e8d8e8f34b71ca07e95077d29dc19f4ea0464d540 | yes |
| 9719-save state lines | en | 6635dadf147275392c5c76d92461df82819e45b6c2de5b67d56cdd9fee9a7208 | 6635dadf147275392c5c76d92461df82819e45b6c2de5b67d56cdd9fee9a7208 | yes |
| 9719-shop2 frame shop-closed | en | 8390c97a8729275124b26effa73e0627a37c1d4ffb45e24f73acd5ad6bb02f5c | 8390c97a8729275124b26effa73e0627a37c1d4ffb45e24f73acd5ad6bb02f5c | yes |
| 9719-shop2 frame shop-first | en | 307f7f4ca7329b7f20566c131eb409beb89dfd575247282f9a2734405155d6da | 307f7f4ca7329b7f20566c131eb409beb89dfd575247282f9a2734405155d6da | yes |
| 9719-shop2 frame shop-reentered-table | en | ed1bc3b51a238a1574d60ccfbd2b0a628687818912c508f35f3f87068b3190ef | ed1bc3b51a238a1574d60ccfbd2b0a628687818912c508f35f3f87068b3190ef | yes |
| 9719-shop2 frame shop-reentered | en | ca7360aaffa530ded04322b866c9427b9203dd6be4376842555b1b2834486a47 | ca7360aaffa530ded04322b866c9427b9203dd6be4376842555b1b2834486a47 | yes |
| 9719-shop2 frame shop-table-again | en | 30afb8ff8f68ed2bba78617ae15691e8c20f1d416a9240aa9954cb82bd40928f | 30afb8ff8f68ed2bba78617ae15691e8c20f1d416a9240aa9954cb82bd40928f | yes |
| 9719-shop2 frame shop-table | en | ed1bc3b51a238a1574d60ccfbd2b0a628687818912c508f35f3f87068b3190ef | ed1bc3b51a238a1574d60ccfbd2b0a628687818912c508f35f3f87068b3190ef | yes |
| 9719-shop2 frame shop-undo | en | fdadbe613b4f386f741c3772225afeb21279dc52ddb89b0d06f9aa1f133fda80 | fdadbe613b4f386f741c3772225afeb21279dc52ddb89b0d06f9aa1f133fda80 | yes |
| 9719-shop2 state lines | en | 3ca67ea2a78ba1ce1e178c926090e414370bc6c6d52fcba2fab10e03ef811620 | 3ca67ea2a78ba1ce1e178c926090e414370bc6c6d52fcba2fab10e03ef811620 | yes |
| 9719-square frame square-closed | en | f6affd12134ec5c4635a29f300cce98f7ae70088cdbbb1cd91a5b3e124b2f1f2 | f6affd12134ec5c4635a29f300cce98f7ae70088cdbbb1cd91a5b3e124b2f1f2 | yes |
| 9719-square frame square-load | en | 1bfc277a443c7370584020970b1b51cd4989e2292175aced250473e1aed85f64 | 1bfc277a443c7370584020970b1b51cd4989e2292175aced250473e1aed85f64 | yes |
| 9719-square frame square-return | en | 54181890c7dab0acd3044df7d7053099dc8fcafe43c6b554713d2c1af83c414f | 54181890c7dab0acd3044df7d7053099dc8fcafe43c6b554713d2c1af83c414f | yes |
| 9719-square state lines | en | 7874238fa9933ff141826274e7c3646f48d20a06b98a802d8b107518ffb5c29e | 7874238fa9933ff141826274e7c3646f48d20a06b98a802d8b107518ffb5c29e | yes |
| 9719-talk frame talk-closed | en | 51c24b86d317bc6edd148c0ebf58c286f169d6e61b8bd1e0e302398f7a858460 | 51c24b86d317bc6edd148c0ebf58c286f169d6e61b8bd1e0e302398f7a858460 | yes |
| 9719-talk frame talk-reread-closed | en | 51c24b86d317bc6edd148c0ebf58c286f169d6e61b8bd1e0e302398f7a858460 | 51c24b86d317bc6edd148c0ebf58c286f169d6e61b8bd1e0e302398f7a858460 | yes |
| 9719-talk frame talk-reread | en | f6145c611bafb320b586aa02d79d422e1376b34485d510ca19bbb22381a29113 | f6145c611bafb320b586aa02d79d422e1376b34485d510ca19bbb22381a29113 | yes |
| 9719-talk frame tavern-in | en | 5ef23b0a66ec25e069368a4a0d998929da6f220b905578c788158e477cabbfd1 | 5ef23b0a66ec25e069368a4a0d998929da6f220b905578c788158e477cabbfd1 | yes |
| 9719-talk state lines | en | f028479a3ef05e8663735894787110f3b5b3d973a0ae89093586264d0d3f5703 | f028479a3ef05e8663735894787110f3b5b3d973a0ae89093586264d0d3f5703 | yes |
| 0000-room-shop frame shop-closed | rom2-en | d4cf1f4873bc60270b377be2e890d062d1bc3a7b7afeca0c13c6b8af8b5e5735 | d4cf1f4873bc60270b377be2e890d062d1bc3a7b7afeca0c13c6b8af8b5e5735 | yes |
| 0000-room-shop frame shop-first | rom2-en | ca95dbf6517e405e1fe31d291e657ac6722af29d481d1ca08627c9ce19a78c72 | ca95dbf6517e405e1fe31d291e657ac6722af29d481d1ca08627c9ce19a78c72 | yes |
| 0000-room-shop frame shop-reentered-after-close | rom2-en | d29334261e89adf4d29fa7b2d49e874c1c01e5546fb26e8e12fe4bdfe9be6a2b | d29334261e89adf4d29fa7b2d49e874c1c01e5546fb26e8e12fe4bdfe9be6a2b | yes |
| 0000-room-shop frame shop-reentered-after-open | rom2-en | d29334261e89adf4d29fa7b2d49e874c1c01e5546fb26e8e12fe4bdfe9be6a2b | d29334261e89adf4d29fa7b2d49e874c1c01e5546fb26e8e12fe4bdfe9be6a2b | yes |
| 0000-room-shop frame shop-square-after-close | rom2-en | e8f4faac36e494b790b58a53f5c1be572702ac3aa8d3b888c0cc5403c5e0d3a4 | e8f4faac36e494b790b58a53f5c1be572702ac3aa8d3b888c0cc5403c5e0d3a4 | yes |
| 0000-room-shop frame shop-square-left-open | rom2-en | 63e32cb1b50b38dcbc898d383a3b872d9951de192cc3e18c985f21f5ed836d8c | 63e32cb1b50b38dcbc898d383a3b872d9951de192cc3e18c985f21f5ed836d8c | yes |
| 0000-room-shop state lines | rom2-en | a76cbb44c732f783bf006ff88290e075263acd216c762b365b672bd797d09686 | a76cbb44c732f783bf006ff88290e075263acd216c762b365b672bd797d09686 | yes |
| 0000-room-tavern frame tavern-closed | rom2-en | 1c117c15b548d19b8cbf1b75f02edbe4ddd063a0b986644351c922f3565ad4e9 | 1c117c15b548d19b8cbf1b75f02edbe4ddd063a0b986644351c922f3565ad4e9 | yes |
| 0000-room-tavern frame tavern-first | rom2-en | 7da02977920a1ddcf602855e0c5f7d4a2d1d2dd24ad55c93c1aa9f53554b023c | 7da02977920a1ddcf602855e0c5f7d4a2d1d2dd24ad55c93c1aa9f53554b023c | yes |
| 0000-room-tavern frame tavern-reentered-after-close | rom2-en | 91e5a159b48f79396a6c5594db946fd316c363efd34745a2f4ed8d2c95708e76 | 91e5a159b48f79396a6c5594db946fd316c363efd34745a2f4ed8d2c95708e76 | yes |
| 0000-room-tavern frame tavern-reentered-after-open | rom2-en | 91e5a159b48f79396a6c5594db946fd316c363efd34745a2f4ed8d2c95708e76 | 91e5a159b48f79396a6c5594db946fd316c363efd34745a2f4ed8d2c95708e76 | yes |
| 0000-room-tavern frame tavern-square-after-close | rom2-en | ff223ae5892e738025758365ff08f3444d5a2bb211aa9a770ff8bce345af0b7e | ff223ae5892e738025758365ff08f3444d5a2bb211aa9a770ff8bce345af0b7e | yes |
| 0000-room-tavern frame tavern-square-left-open | rom2-en | 9409562d2b9c8a016f26c50d6f34d5d9774932ba8a726631c879ffe429d91b36 | 9409562d2b9c8a016f26c50d6f34d5d9774932ba8a726631c879ffe429d91b36 | yes |
| 0000-room-tavern state lines | rom2-en | 42b3b62dc0e5ef7d2501be4dc94905bfed5b8757595a6124e7b36fee2fe9841a | 42b3b62dc0e5ef7d2501be4dc94905bfed5b8757595a6124e7b36fee2fe9841a | yes |
| 0000-save frame save-loaded-tavern | rom2-en | 91e5a159b48f79396a6c5594db946fd316c363efd34745a2f4ed8d2c95708e76 | 91e5a159b48f79396a6c5594db946fd316c363efd34745a2f4ed8d2c95708e76 | yes |
| 0000-save frame save-loaded | rom2-en | d26397b4124f8247d76aa0346442e20f5b3e4abe96f26607f5fbab0452c3f41d | d26397b4124f8247d76aa0346442e20f5b3e4abe96f26607f5fbab0452c3f41d | yes |
| 0000-save frame save-saved | rom2-en | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | yes |
| 0000-save frame save-square | rom2-en | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | yes |
| 0000-save SAV game0001.sav | rom2-en | 1f288bec22edf47e53a3896c0c0f6ae8faa43eba33d54cc184977da7d6d3cdde | 1f288bec22edf47e53a3896c0c0f6ae8faa43eba33d54cc184977da7d6d3cdde | yes |
| 0000-save state lines | rom2-en | b1ef8e797d4ec9d1ba2e0037b166df5ada70e58aaaacab41ee5d2d70a82b7a62 | b1ef8e797d4ec9d1ba2e0037b166df5ada70e58aaaacab41ee5d2d70a82b7a62 | yes |
| 0000-shop2 frame shop-closed | rom2-en | d9d90e5633095f5e2d1e022a72a2331235ac4b3778e1c9fa24476b55b40483c3 | d9d90e5633095f5e2d1e022a72a2331235ac4b3778e1c9fa24476b55b40483c3 | yes |
| 0000-shop2 frame shop-first | rom2-en | ca95dbf6517e405e1fe31d291e657ac6722af29d481d1ca08627c9ce19a78c72 | ca95dbf6517e405e1fe31d291e657ac6722af29d481d1ca08627c9ce19a78c72 | yes |
| 0000-shop2 frame shop-reentered-table | rom2-en | bd41d6b195c9b6d5091e57916ddba655dd9af2c5d35b7a109b78fb99108b7b20 | bd41d6b195c9b6d5091e57916ddba655dd9af2c5d35b7a109b78fb99108b7b20 | yes |
| 0000-shop2 frame shop-reentered | rom2-en | 1fa742bc654b0c72b7dfc701e0b71980c956b9617a6771c0db3b14c66f62b7b9 | 1fa742bc654b0c72b7dfc701e0b71980c956b9617a6771c0db3b14c66f62b7b9 | yes |
| 0000-shop2 frame shop-table-again | rom2-en | bd41d6b195c9b6d5091e57916ddba655dd9af2c5d35b7a109b78fb99108b7b20 | bd41d6b195c9b6d5091e57916ddba655dd9af2c5d35b7a109b78fb99108b7b20 | yes |
| 0000-shop2 frame shop-table | rom2-en | affd31c82edce147592549eb16cbeefae4bb7815a99b42d93c62c5400a12595c | affd31c82edce147592549eb16cbeefae4bb7815a99b42d93c62c5400a12595c | yes |
| 0000-shop2 frame shop-undo | rom2-en | 5ef641639f37a1cc4cae8c447832d7e900996966926e23aaa98e9734107b991d | 5ef641639f37a1cc4cae8c447832d7e900996966926e23aaa98e9734107b991d | yes |
| 0000-shop2 state lines | rom2-en | 4091e4cfee5110910508d511ab0795d994e65745cc20aa115fb5c7ab1c8fb855 | 4091e4cfee5110910508d511ab0795d994e65745cc20aa115fb5c7ab1c8fb855 | yes |
| 0000-square frame square-closed | rom2-en | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | yes |
| 0000-square frame square-load | rom2-en | c1d2489e2bcdfa53e7da137923be8427e2d56ed7aa96dbd947360721fc4d7af7 | c1d2489e2bcdfa53e7da137923be8427e2d56ed7aa96dbd947360721fc4d7af7 | yes |
| 0000-square frame square-return | rom2-en | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | yes |
| 0000-square state lines | rom2-en | 7a0c8aee7315e7e5025ce969e1a014a5d35d23ba7cc9cbd3d87e4aaa71c9ba07 | 7a0c8aee7315e7e5025ce969e1a014a5d35d23ba7cc9cbd3d87e4aaa71c9ba07 | yes |
| 0000-talk frame talk-closed | rom2-en | 1c117c15b548d19b8cbf1b75f02edbe4ddd063a0b986644351c922f3565ad4e9 | 1c117c15b548d19b8cbf1b75f02edbe4ddd063a0b986644351c922f3565ad4e9 | yes |
| 0000-talk frame talk-reread-closed | rom2-en | 1c117c15b548d19b8cbf1b75f02edbe4ddd063a0b986644351c922f3565ad4e9 | 1c117c15b548d19b8cbf1b75f02edbe4ddd063a0b986644351c922f3565ad4e9 | yes |
| 0000-talk frame talk-reread | rom2-en | c641f05b26344ed57ab77fb96a81ffa5ac19eba9d96af2bd9794d755b25b15c9 | c641f05b26344ed57ab77fb96a81ffa5ac19eba9d96af2bd9794d755b25b15c9 | yes |
| 0000-talk frame tavern-in | rom2-en | 7da02977920a1ddcf602855e0c5f7d4a2d1d2dd24ad55c93c1aa9f53554b023c | 7da02977920a1ddcf602855e0c5f7d4a2d1d2dd24ad55c93c1aa9f53554b023c | yes |
| 0000-talk state lines | rom2-en | 83ef5d62a4afab898749f84d96f810cc29ad6059d437a4f98167aace77445999 | 83ef5d62a4afab898749f84d96f810cc29ad6059d437a4f98167aace77445999 | yes |
| 0002-room-shop frame shop-closed | rom2-en | 3f18140df07ac7900ba1b852596cc3f416a9679ead65acef0f3f71aa10693d2b | 3f18140df07ac7900ba1b852596cc3f416a9679ead65acef0f3f71aa10693d2b | yes |
| 0002-room-shop frame shop-first | rom2-en | 10b72b2596009f24572b49f1954b5b595f95a68567654edc14df215ad043e798 | 10b72b2596009f24572b49f1954b5b595f95a68567654edc14df215ad043e798 | yes |
| 0002-room-shop frame shop-reentered-after-close | rom2-en | f38973d1120a07c54a10551e277fb47e25de06124bb026cb0848d4aa7682161f | f38973d1120a07c54a10551e277fb47e25de06124bb026cb0848d4aa7682161f | yes |
| 0002-room-shop frame shop-reentered-after-open | rom2-en | f38973d1120a07c54a10551e277fb47e25de06124bb026cb0848d4aa7682161f | f38973d1120a07c54a10551e277fb47e25de06124bb026cb0848d4aa7682161f | yes |
| 0002-room-shop frame shop-square-after-close | rom2-en | e8f4faac36e494b790b58a53f5c1be572702ac3aa8d3b888c0cc5403c5e0d3a4 | e8f4faac36e494b790b58a53f5c1be572702ac3aa8d3b888c0cc5403c5e0d3a4 | yes |
| 0002-room-shop frame shop-square-left-open | rom2-en | 63e32cb1b50b38dcbc898d383a3b872d9951de192cc3e18c985f21f5ed836d8c | 63e32cb1b50b38dcbc898d383a3b872d9951de192cc3e18c985f21f5ed836d8c | yes |
| 0002-room-shop state lines | rom2-en | 43151fedc66d287a9ab85adcf409483c487b067d94ed55bb4c559f6042204ac7 | 43151fedc66d287a9ab85adcf409483c487b067d94ed55bb4c559f6042204ac7 | yes |
| 0002-room-tavern frame tavern-closed | rom2-en | 534ce37b200cbe68615b6a7fdc3614c9fbcdc0ba89fc6981b59eddce200eb9c5 | 534ce37b200cbe68615b6a7fdc3614c9fbcdc0ba89fc6981b59eddce200eb9c5 | yes |
| 0002-room-tavern frame tavern-first | rom2-en | 4a4b8c3d01dc1123bfd51161d31300a8a98c777ad37e8c136fbfc44fbf9b5a4f | 4a4b8c3d01dc1123bfd51161d31300a8a98c777ad37e8c136fbfc44fbf9b5a4f | yes |
| 0002-room-tavern frame tavern-reentered-after-close | rom2-en | a05a98cd64ab4bb24aa29078dd9a642d4fd5124a6f547fbcbebf2c96c3edb2ae | a05a98cd64ab4bb24aa29078dd9a642d4fd5124a6f547fbcbebf2c96c3edb2ae | yes |
| 0002-room-tavern frame tavern-reentered-after-open | rom2-en | a05a98cd64ab4bb24aa29078dd9a642d4fd5124a6f547fbcbebf2c96c3edb2ae | a05a98cd64ab4bb24aa29078dd9a642d4fd5124a6f547fbcbebf2c96c3edb2ae | yes |
| 0002-room-tavern frame tavern-square-after-close | rom2-en | ff223ae5892e738025758365ff08f3444d5a2bb211aa9a770ff8bce345af0b7e | ff223ae5892e738025758365ff08f3444d5a2bb211aa9a770ff8bce345af0b7e | yes |
| 0002-room-tavern frame tavern-square-left-open | rom2-en | 9409562d2b9c8a016f26c50d6f34d5d9774932ba8a726631c879ffe429d91b36 | 9409562d2b9c8a016f26c50d6f34d5d9774932ba8a726631c879ffe429d91b36 | yes |
| 0002-room-tavern state lines | rom2-en | bb3c49b59f50fc100720bd345b801d6df0342fce9ad1b8406803064e7b442bba | bb3c49b59f50fc100720bd345b801d6df0342fce9ad1b8406803064e7b442bba | yes |
| 0002-save frame save-loaded-tavern | rom2-en | a05a98cd64ab4bb24aa29078dd9a642d4fd5124a6f547fbcbebf2c96c3edb2ae | a05a98cd64ab4bb24aa29078dd9a642d4fd5124a6f547fbcbebf2c96c3edb2ae | yes |
| 0002-save frame save-loaded | rom2-en | d26397b4124f8247d76aa0346442e20f5b3e4abe96f26607f5fbab0452c3f41d | d26397b4124f8247d76aa0346442e20f5b3e4abe96f26607f5fbab0452c3f41d | yes |
| 0002-save frame save-saved | rom2-en | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | yes |
| 0002-save frame save-square | rom2-en | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | yes |
| 0002-save SAV game0000.sav | rom2-en | 4b47efbff08bdffb0366628982fec075284b045dfefe3ceb08fd0e296da4297d | 4b47efbff08bdffb0366628982fec075284b045dfefe3ceb08fd0e296da4297d | yes |
| 0002-save state lines | rom2-en | ed39a054b82f34f4adaf691f31579d44b4ce498fe836678d27eb8502ece2190d | ed39a054b82f34f4adaf691f31579d44b4ce498fe836678d27eb8502ece2190d | yes |
| 0002-shop2 frame shop-closed | rom2-en | 609762d4c9a4af0904154eab4eda470ffb5d3d557eb7f7437c42394bdaf6b1c1 | 609762d4c9a4af0904154eab4eda470ffb5d3d557eb7f7437c42394bdaf6b1c1 | yes |
| 0002-shop2 frame shop-first | rom2-en | 10b72b2596009f24572b49f1954b5b595f95a68567654edc14df215ad043e798 | 10b72b2596009f24572b49f1954b5b595f95a68567654edc14df215ad043e798 | yes |
| 0002-shop2 frame shop-reentered-table | rom2-en | c4ea318f2f2b92c3d4cbf52aab0daa1e9b4aafc0ed84f70264d75ee69900f2ac | c4ea318f2f2b92c3d4cbf52aab0daa1e9b4aafc0ed84f70264d75ee69900f2ac | yes |
| 0002-shop2 frame shop-reentered | rom2-en | d852d6daa031886cd4fc5b677198377e632c42777abd13582d666e03689eaf1e | d852d6daa031886cd4fc5b677198377e632c42777abd13582d666e03689eaf1e | yes |
| 0002-shop2 frame shop-table-again | rom2-en | c4ea318f2f2b92c3d4cbf52aab0daa1e9b4aafc0ed84f70264d75ee69900f2ac | c4ea318f2f2b92c3d4cbf52aab0daa1e9b4aafc0ed84f70264d75ee69900f2ac | yes |
| 0002-shop2 frame shop-table | rom2-en | 15795c31397e6f57da80f3b45f721937f2f1e53d3d17d84b1d0df38fbc335f26 | 15795c31397e6f57da80f3b45f721937f2f1e53d3d17d84b1d0df38fbc335f26 | yes |
| 0002-shop2 frame shop-undo | rom2-en | a5ef78f03cce7e82f0b570c5967a802ccf8ce9b997b96ce82d0145eeb69d80d1 | a5ef78f03cce7e82f0b570c5967a802ccf8ce9b997b96ce82d0145eeb69d80d1 | yes |
| 0002-shop2 state lines | rom2-en | 794477d3ea6c69d0ba5e23e553afb0d22f175586a22fe2a40eae6dc28923a32e | 794477d3ea6c69d0ba5e23e553afb0d22f175586a22fe2a40eae6dc28923a32e | yes |
| 0002-square frame square-closed | rom2-en | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | yes |
| 0002-square frame square-load | rom2-en | c1d2489e2bcdfa53e7da137923be8427e2d56ed7aa96dbd947360721fc4d7af7 | c1d2489e2bcdfa53e7da137923be8427e2d56ed7aa96dbd947360721fc4d7af7 | yes |
| 0002-square frame square-return | rom2-en | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | yes |
| 0002-square state lines | rom2-en | 74449629fa4f47cf8cc773efc10dc1be6cd3b1250d570b4efe7ce21498be31e1 | 74449629fa4f47cf8cc773efc10dc1be6cd3b1250d570b4efe7ce21498be31e1 | yes |
| 0002-talk frame talk-closed | rom2-en | 534ce37b200cbe68615b6a7fdc3614c9fbcdc0ba89fc6981b59eddce200eb9c5 | 534ce37b200cbe68615b6a7fdc3614c9fbcdc0ba89fc6981b59eddce200eb9c5 | yes |
| 0002-talk frame talk-reread-closed | rom2-en | 534ce37b200cbe68615b6a7fdc3614c9fbcdc0ba89fc6981b59eddce200eb9c5 | 534ce37b200cbe68615b6a7fdc3614c9fbcdc0ba89fc6981b59eddce200eb9c5 | yes |
| 0002-talk frame talk-reread | rom2-en | d2030a8267550420077be2f902fa44fe748bd607c826708ed4745dd78d78d597 | d2030a8267550420077be2f902fa44fe748bd607c826708ed4745dd78d78d597 | yes |
| 0002-talk frame tavern-in | rom2-en | 4a4b8c3d01dc1123bfd51161d31300a8a98c777ad37e8c136fbfc44fbf9b5a4f | 4a4b8c3d01dc1123bfd51161d31300a8a98c777ad37e8c136fbfc44fbf9b5a4f | yes |
| 0002-talk state lines | rom2-en | 919fc9aeb905d1917960137b09b231efd38092064ed889ff9964ea1e725c3ba2 | 919fc9aeb905d1917960137b09b231efd38092064ed889ff9964ea1e725c3ba2 | yes |
| 0003-room-shop frame shop-closed | rom2-en | a3e44e735ad2bed58e23f8cb7e2efab81f9aac2e7a6cb8362ab00e1a47fef261 | a3e44e735ad2bed58e23f8cb7e2efab81f9aac2e7a6cb8362ab00e1a47fef261 | yes |
| 0003-room-shop frame shop-first | rom2-en | bc23faebf945448135f74e2cd4abdde8b1b23d732a835dbe28348df7cac61fbd | bc23faebf945448135f74e2cd4abdde8b1b23d732a835dbe28348df7cac61fbd | yes |
| 0003-room-shop frame shop-reentered-after-close | rom2-en | 86d761f9d817fd8fac1b722a72400f068d26c2376810b2c6bc5c0629b8733fd2 | 86d761f9d817fd8fac1b722a72400f068d26c2376810b2c6bc5c0629b8733fd2 | yes |
| 0003-room-shop frame shop-reentered-after-open | rom2-en | 02dbdab495caf724a48054c4a424c9fd437f485eb3f7400bf1380659e29a4073 | 02dbdab495caf724a48054c4a424c9fd437f485eb3f7400bf1380659e29a4073 | yes |
| 0003-room-shop frame shop-square-after-close | rom2-en | e8f4faac36e494b790b58a53f5c1be572702ac3aa8d3b888c0cc5403c5e0d3a4 | e8f4faac36e494b790b58a53f5c1be572702ac3aa8d3b888c0cc5403c5e0d3a4 | yes |
| 0003-room-shop frame shop-square-left-open | rom2-en | 63e32cb1b50b38dcbc898d383a3b872d9951de192cc3e18c985f21f5ed836d8c | 63e32cb1b50b38dcbc898d383a3b872d9951de192cc3e18c985f21f5ed836d8c | yes |
| 0003-room-shop state lines | rom2-en | 9f24fb5c5ef56726c00ba0fd0885eebd56e71af00941c97a63d7b4b0c6ec2d56 | 9f24fb5c5ef56726c00ba0fd0885eebd56e71af00941c97a63d7b4b0c6ec2d56 | yes |
| 0003-room-tavern frame tavern-closed | rom2-en | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | yes |
| 0003-room-tavern frame tavern-first | rom2-en | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | yes |
| 0003-room-tavern frame tavern-reentered-after-close | rom2-en | e3887a24e35496546462e8952e474b050b3be01a9ca5a84129eb700fd9585dec | e3887a24e35496546462e8952e474b050b3be01a9ca5a84129eb700fd9585dec | yes |
| 0003-room-tavern frame tavern-reentered-after-open | rom2-en | e3887a24e35496546462e8952e474b050b3be01a9ca5a84129eb700fd9585dec | e3887a24e35496546462e8952e474b050b3be01a9ca5a84129eb700fd9585dec | yes |
| 0003-room-tavern frame tavern-square-after-close | rom2-en | ff223ae5892e738025758365ff08f3444d5a2bb211aa9a770ff8bce345af0b7e | ff223ae5892e738025758365ff08f3444d5a2bb211aa9a770ff8bce345af0b7e | yes |
| 0003-room-tavern frame tavern-square-left-open | rom2-en | 9409562d2b9c8a016f26c50d6f34d5d9774932ba8a726631c879ffe429d91b36 | 9409562d2b9c8a016f26c50d6f34d5d9774932ba8a726631c879ffe429d91b36 | yes |
| 0003-room-tavern state lines | rom2-en | 501d8b4c3454972d011a70e9558aaadfd1faf46c5330141f780d87df9634eecc | 501d8b4c3454972d011a70e9558aaadfd1faf46c5330141f780d87df9634eecc | yes |
| 0003-save frame save-loaded-tavern | rom2-en | e3887a24e35496546462e8952e474b050b3be01a9ca5a84129eb700fd9585dec | e3887a24e35496546462e8952e474b050b3be01a9ca5a84129eb700fd9585dec | yes |
| 0003-save frame save-loaded | rom2-en | d26397b4124f8247d76aa0346442e20f5b3e4abe96f26607f5fbab0452c3f41d | d26397b4124f8247d76aa0346442e20f5b3e4abe96f26607f5fbab0452c3f41d | yes |
| 0003-save frame save-saved | rom2-en | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | yes |
| 0003-save frame save-square | rom2-en | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | yes |
| 0003-save SAV game0000.sav | rom2-en | 8671daa97a0effc178188c62b7934302b1e771ae4b4a0788f500ce991b8eefd5 | 8671daa97a0effc178188c62b7934302b1e771ae4b4a0788f500ce991b8eefd5 | yes |
| 0003-save state lines | rom2-en | 9fb78b8dd1593fe357a7ab9358a832793dda55dc7b435282f524369bfac18a24 | 9fb78b8dd1593fe357a7ab9358a832793dda55dc7b435282f524369bfac18a24 | yes |
| 0003-shop2 frame shop-closed | rom2-en | edfc6d1e731ea9ce448148d0024f8fbb7a958d80633779d523a8680ad52dc4e3 | edfc6d1e731ea9ce448148d0024f8fbb7a958d80633779d523a8680ad52dc4e3 | yes |
| 0003-shop2 frame shop-first | rom2-en | bc23faebf945448135f74e2cd4abdde8b1b23d732a835dbe28348df7cac61fbd | bc23faebf945448135f74e2cd4abdde8b1b23d732a835dbe28348df7cac61fbd | yes |
| 0003-shop2 frame shop-reentered-table | rom2-en | e9217b784753248ab08b84cf5a5e698164364c2fc09e9f2b74cccbb2ab3f479c | e9217b784753248ab08b84cf5a5e698164364c2fc09e9f2b74cccbb2ab3f479c | yes |
| 0003-shop2 frame shop-reentered | rom2-en | 215229679b230974bb89757628fe673ee47201ae94b977aa2d56aaa631434fbd | 215229679b230974bb89757628fe673ee47201ae94b977aa2d56aaa631434fbd | yes |
| 0003-shop2 frame shop-table-again | rom2-en | 73cda143294d073e13aeb260dcb8e31985304fc9c8f26f436051f804372a923d | 73cda143294d073e13aeb260dcb8e31985304fc9c8f26f436051f804372a923d | yes |
| 0003-shop2 frame shop-table | rom2-en | faa7323e262244e0b0b62c4096d44171f40f30ca8a899d92e053b34ad648db25 | faa7323e262244e0b0b62c4096d44171f40f30ca8a899d92e053b34ad648db25 | yes |
| 0003-shop2 frame shop-undo | rom2-en | 0c776ef9650c9c615e1ffabe7ca0407bf1257b1c3123ebb3eeb9fec66bd00c38 | 0c776ef9650c9c615e1ffabe7ca0407bf1257b1c3123ebb3eeb9fec66bd00c38 | yes |
| 0003-shop2 state lines | rom2-en | b6f2c66bf17494c66b83e1e5637d2887a6bd1872ee45278a76bcab6146624de1 | b6f2c66bf17494c66b83e1e5637d2887a6bd1872ee45278a76bcab6146624de1 | yes |
| 0003-square frame square-closed | rom2-en | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | yes |
| 0003-square frame square-load | rom2-en | c1d2489e2bcdfa53e7da137923be8427e2d56ed7aa96dbd947360721fc4d7af7 | c1d2489e2bcdfa53e7da137923be8427e2d56ed7aa96dbd947360721fc4d7af7 | yes |
| 0003-square frame square-return | rom2-en | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | yes |
| 0003-square state lines | rom2-en | 28730beb5e6ff9343889f3bbea7c106c5dec34b7d2d80e9ae7861324e33719c8 | 28730beb5e6ff9343889f3bbea7c106c5dec34b7d2d80e9ae7861324e33719c8 | yes |
| 0003-talk frame talk-closed | rom2-en | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | yes |
| 0003-talk frame talk-reread-closed | rom2-en | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | yes |
| 0003-talk frame talk-reread | rom2-en | ae21ca46ad86682a547869ca3eae2522e93c513d79402d8f865e8eaff61bf32b | ae21ca46ad86682a547869ca3eae2522e93c513d79402d8f865e8eaff61bf32b | yes |
| 0003-talk frame tavern-in | rom2-en | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | yes |
| 0003-talk state lines | rom2-en | 3150b1f68b0991971ea1266bd102e5d6337aa730446374e1907cf1f24bc03961 | 3150b1f68b0991971ea1266bd102e5d6337aa730446374e1907cf1f24bc03961 | yes |
| 9998-room-shop frame shop-closed | rom2-en | e369cdd3efbc06990d9aa0960e49cd55f78a3c794793689f5d3b848d37c63a8f | e369cdd3efbc06990d9aa0960e49cd55f78a3c794793689f5d3b848d37c63a8f | yes |
| 9998-room-shop frame shop-first | rom2-en | 21f9ec488746116b57fa08d6fb2b38930b65642995d24a32e2de4751baf5d7e3 | 21f9ec488746116b57fa08d6fb2b38930b65642995d24a32e2de4751baf5d7e3 | yes |
| 9998-room-shop frame shop-reentered-after-close | rom2-en | f2d34547e883afe42d96d502d401b644a846953c1875911db1ead062ba3fda9e | f2d34547e883afe42d96d502d401b644a846953c1875911db1ead062ba3fda9e | yes |
| 9998-room-shop frame shop-reentered-after-open | rom2-en | 1a7ab340259fd3b58de0f5dc84610b83aa630e3f70e2617577f159fcf47d46cb | 1a7ab340259fd3b58de0f5dc84610b83aa630e3f70e2617577f159fcf47d46cb | yes |
| 9998-room-shop frame shop-square-after-close | rom2-en | e8f4faac36e494b790b58a53f5c1be572702ac3aa8d3b888c0cc5403c5e0d3a4 | e8f4faac36e494b790b58a53f5c1be572702ac3aa8d3b888c0cc5403c5e0d3a4 | yes |
| 9998-room-shop frame shop-square-left-open | rom2-en | 63e32cb1b50b38dcbc898d383a3b872d9951de192cc3e18c985f21f5ed836d8c | 63e32cb1b50b38dcbc898d383a3b872d9951de192cc3e18c985f21f5ed836d8c | yes |
| 9998-room-shop state lines | rom2-en | 9f24fb5c5ef56726c00ba0fd0885eebd56e71af00941c97a63d7b4b0c6ec2d56 | 9f24fb5c5ef56726c00ba0fd0885eebd56e71af00941c97a63d7b4b0c6ec2d56 | yes |
| 9998-room-tavern frame tavern-closed | rom2-en | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | yes |
| 9998-room-tavern frame tavern-first | rom2-en | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | yes |
| 9998-room-tavern frame tavern-reentered-after-close | rom2-en | e3887a24e35496546462e8952e474b050b3be01a9ca5a84129eb700fd9585dec | e3887a24e35496546462e8952e474b050b3be01a9ca5a84129eb700fd9585dec | yes |
| 9998-room-tavern frame tavern-reentered-after-open | rom2-en | e3887a24e35496546462e8952e474b050b3be01a9ca5a84129eb700fd9585dec | e3887a24e35496546462e8952e474b050b3be01a9ca5a84129eb700fd9585dec | yes |
| 9998-room-tavern frame tavern-square-after-close | rom2-en | ff223ae5892e738025758365ff08f3444d5a2bb211aa9a770ff8bce345af0b7e | ff223ae5892e738025758365ff08f3444d5a2bb211aa9a770ff8bce345af0b7e | yes |
| 9998-room-tavern frame tavern-square-left-open | rom2-en | 9409562d2b9c8a016f26c50d6f34d5d9774932ba8a726631c879ffe429d91b36 | 9409562d2b9c8a016f26c50d6f34d5d9774932ba8a726631c879ffe429d91b36 | yes |
| 9998-room-tavern state lines | rom2-en | 501d8b4c3454972d011a70e9558aaadfd1faf46c5330141f780d87df9634eecc | 501d8b4c3454972d011a70e9558aaadfd1faf46c5330141f780d87df9634eecc | yes |
| 9998-save frame save-loaded-tavern | rom2-en | e3887a24e35496546462e8952e474b050b3be01a9ca5a84129eb700fd9585dec | e3887a24e35496546462e8952e474b050b3be01a9ca5a84129eb700fd9585dec | yes |
| 9998-save frame save-loaded | rom2-en | d26397b4124f8247d76aa0346442e20f5b3e4abe96f26607f5fbab0452c3f41d | d26397b4124f8247d76aa0346442e20f5b3e4abe96f26607f5fbab0452c3f41d | yes |
| 9998-save frame save-saved | rom2-en | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | yes |
| 9998-save frame save-square | rom2-en | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | yes |
| 9998-save SAV game0000.sav | rom2-en | 5c06780748162a596becd0f8dbe056a5080e8f5c577fe2d59ad89822c22c6cdb | 5c06780748162a596becd0f8dbe056a5080e8f5c577fe2d59ad89822c22c6cdb | yes |
| 9998-save state lines | rom2-en | bb37a4632f9ac1ed5ec1582d969531385da89ec14e0b74d3a99c2e084d0201f2 | bb37a4632f9ac1ed5ec1582d969531385da89ec14e0b74d3a99c2e084d0201f2 | yes |
| 9998-shop2 frame shop-closed | rom2-en | 5dd582288b2d0a0a30fba86f88ff485571b72b4610a6eec72daf4da9cc12d905 | 5dd582288b2d0a0a30fba86f88ff485571b72b4610a6eec72daf4da9cc12d905 | yes |
| 9998-shop2 frame shop-first | rom2-en | 21f9ec488746116b57fa08d6fb2b38930b65642995d24a32e2de4751baf5d7e3 | 21f9ec488746116b57fa08d6fb2b38930b65642995d24a32e2de4751baf5d7e3 | yes |
| 9998-shop2 frame shop-reentered-table | rom2-en | 1db7b4a0c414ad898be4483bffa46a6752df6f7de35bb5b4142cef491b623c2d | 1db7b4a0c414ad898be4483bffa46a6752df6f7de35bb5b4142cef491b623c2d | yes |
| 9998-shop2 frame shop-reentered | rom2-en | 1546ffb305685ee249c86491d3f21fb3e9e6d32a31063a74b1a4a5a7439d3350 | 1546ffb305685ee249c86491d3f21fb3e9e6d32a31063a74b1a4a5a7439d3350 | yes |
| 9998-shop2 frame shop-table-again | rom2-en | a5d3933c5715fa1eaadf30f425c8ebeb24e609807f0e5400b4de77f3206b6ba0 | a5d3933c5715fa1eaadf30f425c8ebeb24e609807f0e5400b4de77f3206b6ba0 | yes |
| 9998-shop2 frame shop-table | rom2-en | 83fcc8a69fe328445d44f511b6cc84c206842f505119e341f159a0f0fc2f8cbf | 83fcc8a69fe328445d44f511b6cc84c206842f505119e341f159a0f0fc2f8cbf | yes |
| 9998-shop2 frame shop-undo | rom2-en | 6c59bc8011c982754c946c0f309c3e291c3e925c9b121e201c910cb1ecce2411 | 6c59bc8011c982754c946c0f309c3e291c3e925c9b121e201c910cb1ecce2411 | yes |
| 9998-shop2 state lines | rom2-en | b6f2c66bf17494c66b83e1e5637d2887a6bd1872ee45278a76bcab6146624de1 | b6f2c66bf17494c66b83e1e5637d2887a6bd1872ee45278a76bcab6146624de1 | yes |
| 9998-square frame square-closed | rom2-en | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | yes |
| 9998-square frame square-load | rom2-en | c1d2489e2bcdfa53e7da137923be8427e2d56ed7aa96dbd947360721fc4d7af7 | c1d2489e2bcdfa53e7da137923be8427e2d56ed7aa96dbd947360721fc4d7af7 | yes |
| 9998-square frame square-return | rom2-en | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | yes |
| 9998-square state lines | rom2-en | 28730beb5e6ff9343889f3bbea7c106c5dec34b7d2d80e9ae7861324e33719c8 | 28730beb5e6ff9343889f3bbea7c106c5dec34b7d2d80e9ae7861324e33719c8 | yes |
| 9998-talk frame talk-closed | rom2-en | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | yes |
| 9998-talk frame talk-reread-closed | rom2-en | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | yes |
| 9998-talk frame talk-reread | rom2-en | ae21ca46ad86682a547869ca3eae2522e93c513d79402d8f865e8eaff61bf32b | ae21ca46ad86682a547869ca3eae2522e93c513d79402d8f865e8eaff61bf32b | yes |
| 9998-talk frame tavern-in | rom2-en | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | yes |
| 9998-talk state lines | rom2-en | 3150b1f68b0991971ea1266bd102e5d6337aa730446374e1907cf1f24bc03961 | 3150b1f68b0991971ea1266bd102e5d6337aa730446374e1907cf1f24bc03961 | yes |
| new-room-shop frame shop-closed | rom2-en | 0cb278ee2727d33b2e60feb2c82ab6eccf38d25bd8b3299e7851f716d914c3f5 | 0cb278ee2727d33b2e60feb2c82ab6eccf38d25bd8b3299e7851f716d914c3f5 | yes |
| new-room-shop frame shop-first | rom2-en | ad5e59c8015de7c8eb9e0023b78aac28437bd3c3af7cbc630a58688c0632e277 | ad5e59c8015de7c8eb9e0023b78aac28437bd3c3af7cbc630a58688c0632e277 | yes |
| new-room-shop frame shop-reentered-after-close | rom2-en | 77dcbdc7a2f86e68ab23f6ce3e6f3cf11b70cc54a85d569009c6b1ef5734bdbe | 77dcbdc7a2f86e68ab23f6ce3e6f3cf11b70cc54a85d569009c6b1ef5734bdbe | yes |
| new-room-shop frame shop-reentered-after-open | rom2-en | 77dcbdc7a2f86e68ab23f6ce3e6f3cf11b70cc54a85d569009c6b1ef5734bdbe | 77dcbdc7a2f86e68ab23f6ce3e6f3cf11b70cc54a85d569009c6b1ef5734bdbe | yes |
| new-room-shop frame shop-square-after-close | rom2-en | e8f4faac36e494b790b58a53f5c1be572702ac3aa8d3b888c0cc5403c5e0d3a4 | e8f4faac36e494b790b58a53f5c1be572702ac3aa8d3b888c0cc5403c5e0d3a4 | yes |
| new-room-shop frame shop-square-left-open | rom2-en | 63e32cb1b50b38dcbc898d383a3b872d9951de192cc3e18c985f21f5ed836d8c | 63e32cb1b50b38dcbc898d383a3b872d9951de192cc3e18c985f21f5ed836d8c | yes |
| new-room-shop state lines | rom2-en | 12e6efd5a978ce2bb1d235dc9c58c3b1ccc1411171baa8546ebff04897c1f847 | 12e6efd5a978ce2bb1d235dc9c58c3b1ccc1411171baa8546ebff04897c1f847 | yes |
| new-room-tavern frame tavern-closed | rom2-en | ef2808f5c0e37678f001b2352ae057358d7f39f146fe9b4f8f3b69bca297a0ca | ef2808f5c0e37678f001b2352ae057358d7f39f146fe9b4f8f3b69bca297a0ca | yes |
| new-room-tavern frame tavern-first | rom2-en | 815a06e602dde8fa87c08501bd53b6be731e52e2ce907a0f753d6c737d5682f6 | 815a06e602dde8fa87c08501bd53b6be731e52e2ce907a0f753d6c737d5682f6 | yes |
| new-room-tavern frame tavern-reentered-after-close | rom2-en | ba30700fe9ea5e4400edd7552067009d6781785f9394cdfb0dd1e508d360ee3c | ba30700fe9ea5e4400edd7552067009d6781785f9394cdfb0dd1e508d360ee3c | yes |
| new-room-tavern frame tavern-reentered-after-open | rom2-en | ba30700fe9ea5e4400edd7552067009d6781785f9394cdfb0dd1e508d360ee3c | ba30700fe9ea5e4400edd7552067009d6781785f9394cdfb0dd1e508d360ee3c | yes |
| new-room-tavern frame tavern-square-after-close | rom2-en | ff223ae5892e738025758365ff08f3444d5a2bb211aa9a770ff8bce345af0b7e | ff223ae5892e738025758365ff08f3444d5a2bb211aa9a770ff8bce345af0b7e | yes |
| new-room-tavern frame tavern-square-left-open | rom2-en | 9409562d2b9c8a016f26c50d6f34d5d9774932ba8a726631c879ffe429d91b36 | 9409562d2b9c8a016f26c50d6f34d5d9774932ba8a726631c879ffe429d91b36 | yes |
| new-room-tavern state lines | rom2-en | 8e846dea8837abe6591e6130ecfa436215a3489083dfba8d239bd0a718966f05 | 8e846dea8837abe6591e6130ecfa436215a3489083dfba8d239bd0a718966f05 | yes |
| new-save frame save-loaded-tavern | rom2-en | ba30700fe9ea5e4400edd7552067009d6781785f9394cdfb0dd1e508d360ee3c | ba30700fe9ea5e4400edd7552067009d6781785f9394cdfb0dd1e508d360ee3c | yes |
| new-save frame save-loaded | rom2-en | d26397b4124f8247d76aa0346442e20f5b3e4abe96f26607f5fbab0452c3f41d | d26397b4124f8247d76aa0346442e20f5b3e4abe96f26607f5fbab0452c3f41d | yes |
| new-save frame save-saved | rom2-en | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | yes |
| new-save frame save-square | rom2-en | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | 5efb31f98738e9f078c67d4e28b32fa36f33e960da70daaa1873ed79db8c84e6 | yes |
| new-save SAV game0000.sav | rom2-en | 9fedcbcb6d2c97ea1227baa46aec17f4a0c66e6cf67ff26c86311fd3b47069c4 | 9fedcbcb6d2c97ea1227baa46aec17f4a0c66e6cf67ff26c86311fd3b47069c4 | yes |
| new-save state lines | rom2-en | 496c185b379a9de23d19c08871b1c2c2462e274c58f64c713fee0348eed0e77e | 496c185b379a9de23d19c08871b1c2c2462e274c58f64c713fee0348eed0e77e | yes |
| new-shop2 frame shop-closed | rom2-en | 699cd8a2c0dfcf03c58d7542cd280d9b446edbf2d2d7f1d21a3f8aa3b548715c | 699cd8a2c0dfcf03c58d7542cd280d9b446edbf2d2d7f1d21a3f8aa3b548715c | yes |
| new-shop2 frame shop-first | rom2-en | ad5e59c8015de7c8eb9e0023b78aac28437bd3c3af7cbc630a58688c0632e277 | ad5e59c8015de7c8eb9e0023b78aac28437bd3c3af7cbc630a58688c0632e277 | yes |
| new-shop2 frame shop-reentered-table | rom2-en | 6622b5496d9fe626a23a43f33f582abfdfe80e1af151af9bc7c0a6d4602525d9 | 6622b5496d9fe626a23a43f33f582abfdfe80e1af151af9bc7c0a6d4602525d9 | yes |
| new-shop2 frame shop-reentered | rom2-en | 989885ceb607e9b6e9eb8b36f6e7940fb1af0f53d1e35b4d5b1b973599f6130f | 989885ceb607e9b6e9eb8b36f6e7940fb1af0f53d1e35b4d5b1b973599f6130f | yes |
| new-shop2 frame shop-table-again | rom2-en | 6622b5496d9fe626a23a43f33f582abfdfe80e1af151af9bc7c0a6d4602525d9 | 6622b5496d9fe626a23a43f33f582abfdfe80e1af151af9bc7c0a6d4602525d9 | yes |
| new-shop2 frame shop-table | rom2-en | 6e389bd095e5e5a5f2e6a1a1c777cb454181ec83ef5e011503395cfbb17e539c | 6e389bd095e5e5a5f2e6a1a1c777cb454181ec83ef5e011503395cfbb17e539c | yes |
| new-shop2 frame shop-undo | rom2-en | 5ffdc7119ee5a6030308dd494905dff67b5fc8fc185486e787661c0fe8c69d0f | 5ffdc7119ee5a6030308dd494905dff67b5fc8fc185486e787661c0fe8c69d0f | yes |
| new-shop2 state lines | rom2-en | 8765d233553159f42e35ac0e016db595041e3c09616a9bd1d49cf5736550f631 | 8765d233553159f42e35ac0e016db595041e3c09616a9bd1d49cf5736550f631 | yes |
| new-square frame square-closed | rom2-en | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | yes |
| new-square frame square-load | rom2-en | c1d2489e2bcdfa53e7da137923be8427e2d56ed7aa96dbd947360721fc4d7af7 | c1d2489e2bcdfa53e7da137923be8427e2d56ed7aa96dbd947360721fc4d7af7 | yes |
| new-square frame square-return | rom2-en | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | yes |
| new-square state lines | rom2-en | 56c28108c957f43b154dfd551f9e0f11ebf223deb926744855e7138f3d54c2a4 | 56c28108c957f43b154dfd551f9e0f11ebf223deb926744855e7138f3d54c2a4 | yes |
| new-talk frame talk-closed | rom2-en | ef2808f5c0e37678f001b2352ae057358d7f39f146fe9b4f8f3b69bca297a0ca | ef2808f5c0e37678f001b2352ae057358d7f39f146fe9b4f8f3b69bca297a0ca | yes |
| new-talk frame talk-reread-closed | rom2-en | ef2808f5c0e37678f001b2352ae057358d7f39f146fe9b4f8f3b69bca297a0ca | ef2808f5c0e37678f001b2352ae057358d7f39f146fe9b4f8f3b69bca297a0ca | yes |
| new-talk frame talk-reread | rom2-en | ee0e3518f63f90865d445c3db10216e86f93f124fd76e65699bc4863292b92f3 | ee0e3518f63f90865d445c3db10216e86f93f124fd76e65699bc4863292b92f3 | yes |
| new-talk frame tavern-in | rom2-en | 815a06e602dde8fa87c08501bd53b6be731e52e2ce907a0f753d6c737d5682f6 | 815a06e602dde8fa87c08501bd53b6be731e52e2ce907a0f753d6c737d5682f6 | yes |
| new-talk state lines | rom2-en | 287da3768b515168752a1f5b561a064e0d8cc98f83de664124a0d4b404ff4f43 | 287da3768b515168752a1f5b561a064e0d8cc98f83de664124a0d4b404ff4f43 | yes |
| 0000-room-shop frame shop-closed | rom2-ru | 68046ef2d9f7e463a0dcef8514d5c377bb64e22d7f8d0f197d9d8c460197857f | 68046ef2d9f7e463a0dcef8514d5c377bb64e22d7f8d0f197d9d8c460197857f | yes |
| 0000-room-shop frame shop-first | rom2-ru | ec2f3a859b3a107fed5b096d01577090b62eb9a72a8f356f6e402bd8a351133a | ec2f3a859b3a107fed5b096d01577090b62eb9a72a8f356f6e402bd8a351133a | yes |
| 0000-room-shop frame shop-reentered-after-close | rom2-ru | f899d08a344195cc47504e714c1935a516fdba9846414a2ac82a72625057be02 | f899d08a344195cc47504e714c1935a516fdba9846414a2ac82a72625057be02 | yes |
| 0000-room-shop frame shop-reentered-after-open | rom2-ru | f899d08a344195cc47504e714c1935a516fdba9846414a2ac82a72625057be02 | f899d08a344195cc47504e714c1935a516fdba9846414a2ac82a72625057be02 | yes |
| 0000-room-shop frame shop-square-after-close | rom2-ru | 9b698e55a7a74c5f7d38eb08e0deb025bbfa20026bc5348dbad1b7ea454bef7e | 9b698e55a7a74c5f7d38eb08e0deb025bbfa20026bc5348dbad1b7ea454bef7e | yes |
| 0000-room-shop frame shop-square-left-open | rom2-ru | d2f89ef35877d40f8af63ab26c8da57c82d968d426946cf0bc42ee178c7558b7 | d2f89ef35877d40f8af63ab26c8da57c82d968d426946cf0bc42ee178c7558b7 | yes |
| 0000-room-shop state lines | rom2-ru | 0fdf8ddb42feee9dd34e580bb5df9d245bf5ca745867e6e041c41365a65868fe | 0fdf8ddb42feee9dd34e580bb5df9d245bf5ca745867e6e041c41365a65868fe | yes |
| 0000-room-tavern frame tavern-closed | rom2-ru | d30f33469342e571bb4b65458a293ecfcd401b752b4508944614721d1617939a | d30f33469342e571bb4b65458a293ecfcd401b752b4508944614721d1617939a | yes |
| 0000-room-tavern frame tavern-first | rom2-ru | 80cd6f81afe5b84bb066589b8be9903f343149be88e2f19a6fbf587cedb03269 | 80cd6f81afe5b84bb066589b8be9903f343149be88e2f19a6fbf587cedb03269 | yes |
| 0000-room-tavern frame tavern-reentered-after-close | rom2-ru | 58fc75d6b8c37c4fa4915247b6fece56fcfc81e5b3050112c7901588d17ec056 | 58fc75d6b8c37c4fa4915247b6fece56fcfc81e5b3050112c7901588d17ec056 | yes |
| 0000-room-tavern frame tavern-reentered-after-open | rom2-ru | 58fc75d6b8c37c4fa4915247b6fece56fcfc81e5b3050112c7901588d17ec056 | 58fc75d6b8c37c4fa4915247b6fece56fcfc81e5b3050112c7901588d17ec056 | yes |
| 0000-room-tavern frame tavern-square-after-close | rom2-ru | 7e2bfb42b0a41baec2af8755d4db4e1f9acbc15237e828401a0541313bf423ea | 7e2bfb42b0a41baec2af8755d4db4e1f9acbc15237e828401a0541313bf423ea | yes |
| 0000-room-tavern frame tavern-square-left-open | rom2-ru | 5ae4a522d5a690bb254f8300af7745bad3625308b71107b3198464d325a9dfaf | 5ae4a522d5a690bb254f8300af7745bad3625308b71107b3198464d325a9dfaf | yes |
| 0000-room-tavern state lines | rom2-ru | 921f723053abcbd56727acbd8c273b747518869138e4659f2f907075a556e048 | 921f723053abcbd56727acbd8c273b747518869138e4659f2f907075a556e048 | yes |
| 0000-save frame save-loaded-tavern | rom2-ru | 58fc75d6b8c37c4fa4915247b6fece56fcfc81e5b3050112c7901588d17ec056 | 58fc75d6b8c37c4fa4915247b6fece56fcfc81e5b3050112c7901588d17ec056 | yes |
| 0000-save frame save-loaded | rom2-ru | 030ac80ff9445771e435dd030ac430ed6304908415168d67c14096fe7c194aae | 030ac80ff9445771e435dd030ac430ed6304908415168d67c14096fe7c194aae | yes |
| 0000-save frame save-saved | rom2-ru | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | yes |
| 0000-save frame save-square | rom2-ru | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | yes |
| 0000-save SAV game0001.sav | rom2-ru | 1f288bec22edf47e53a3896c0c0f6ae8faa43eba33d54cc184977da7d6d3cdde | 1f288bec22edf47e53a3896c0c0f6ae8faa43eba33d54cc184977da7d6d3cdde | yes |
| 0000-save state lines | rom2-ru | 1097e7555580cd0326c0ae7bb3a8be96e86f60417dfa443ea51f51e396b6d327 | 1097e7555580cd0326c0ae7bb3a8be96e86f60417dfa443ea51f51e396b6d327 | yes |
| 0000-shop2 frame shop-closed | rom2-ru | 1e41f956cf1840f74d9e7d13d96a4702f1ad2727b00bb97f190e62df728c766d | 1e41f956cf1840f74d9e7d13d96a4702f1ad2727b00bb97f190e62df728c766d | yes |
| 0000-shop2 frame shop-first | rom2-ru | ec2f3a859b3a107fed5b096d01577090b62eb9a72a8f356f6e402bd8a351133a | ec2f3a859b3a107fed5b096d01577090b62eb9a72a8f356f6e402bd8a351133a | yes |
| 0000-shop2 frame shop-reentered-table | rom2-ru | 4b965f9c3fe9770a5b1b14028d8b7d762ca55c8e2b0d2537f1e23ebfcbfca0f6 | 4b965f9c3fe9770a5b1b14028d8b7d762ca55c8e2b0d2537f1e23ebfcbfca0f6 | yes |
| 0000-shop2 frame shop-reentered | rom2-ru | 10ed823dd0c39c29e318511f843405fb58a14b89144f6a5bea975f3578407ead | 10ed823dd0c39c29e318511f843405fb58a14b89144f6a5bea975f3578407ead | yes |
| 0000-shop2 frame shop-table-again | rom2-ru | 4b965f9c3fe9770a5b1b14028d8b7d762ca55c8e2b0d2537f1e23ebfcbfca0f6 | 4b965f9c3fe9770a5b1b14028d8b7d762ca55c8e2b0d2537f1e23ebfcbfca0f6 | yes |
| 0000-shop2 frame shop-table | rom2-ru | b5952dc3506d13eb2d42f97adc156c03380fb7319f6dbeddd0d8830c8941aa31 | b5952dc3506d13eb2d42f97adc156c03380fb7319f6dbeddd0d8830c8941aa31 | yes |
| 0000-shop2 frame shop-undo | rom2-ru | 01431fcf6b6ec9f865bcc81b61c886929ccf56040a38069113ed2cd98ac9fe49 | 01431fcf6b6ec9f865bcc81b61c886929ccf56040a38069113ed2cd98ac9fe49 | yes |
| 0000-shop2 state lines | rom2-ru | bb3ca093fbbbb7ce3b07bd6584a08475ad004ca5dd5ebaeef3db061311ffeb04 | bb3ca093fbbbb7ce3b07bd6584a08475ad004ca5dd5ebaeef3db061311ffeb04 | yes |
| 0000-square frame square-closed | rom2-ru | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | yes |
| 0000-square frame square-load | rom2-ru | 5b7a94f6baeb5461423e7ea72ac94c8aba79c392c66b8a3f220a0929b2d560a3 | 5b7a94f6baeb5461423e7ea72ac94c8aba79c392c66b8a3f220a0929b2d560a3 | yes |
| 0000-square frame square-return | rom2-ru | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | yes |
| 0000-square state lines | rom2-ru | 51ae6737ebbaed725a53df077dcbb646a37728f2decfad819bb82b89f9fc8125 | 51ae6737ebbaed725a53df077dcbb646a37728f2decfad819bb82b89f9fc8125 | yes |
| 0000-talk frame talk-closed | rom2-ru | d30f33469342e571bb4b65458a293ecfcd401b752b4508944614721d1617939a | d30f33469342e571bb4b65458a293ecfcd401b752b4508944614721d1617939a | yes |
| 0000-talk frame talk-reread-closed | rom2-ru | d30f33469342e571bb4b65458a293ecfcd401b752b4508944614721d1617939a | d30f33469342e571bb4b65458a293ecfcd401b752b4508944614721d1617939a | yes |
| 0000-talk frame talk-reread | rom2-ru | b43901511b3005ec148663f7c68f89a19f8d443bd8e141aaa8d5a1e409b4744b | b43901511b3005ec148663f7c68f89a19f8d443bd8e141aaa8d5a1e409b4744b | yes |
| 0000-talk frame tavern-in | rom2-ru | 80cd6f81afe5b84bb066589b8be9903f343149be88e2f19a6fbf587cedb03269 | 80cd6f81afe5b84bb066589b8be9903f343149be88e2f19a6fbf587cedb03269 | yes |
| 0000-talk state lines | rom2-ru | 7c474608c90addf4fe43f9656d596642441a7cee90d2e0752ab36e93e23decfd | 7c474608c90addf4fe43f9656d596642441a7cee90d2e0752ab36e93e23decfd | yes |
| 0002-room-shop frame shop-closed | rom2-ru | def3d42393a25a042ea1235f1810444d55b4e55596d118b11ac65843252a0faa | def3d42393a25a042ea1235f1810444d55b4e55596d118b11ac65843252a0faa | yes |
| 0002-room-shop frame shop-first | rom2-ru | cd48a7aba0a61dce440de29c40fbb41f6c15f19551bca28406e18886d19effbd | cd48a7aba0a61dce440de29c40fbb41f6c15f19551bca28406e18886d19effbd | yes |
| 0002-room-shop frame shop-reentered-after-close | rom2-ru | 27cf8dd27449cc0227ae8b3a6b1457f375b8f722c105774c8e76115bb1ef01e6 | 27cf8dd27449cc0227ae8b3a6b1457f375b8f722c105774c8e76115bb1ef01e6 | yes |
| 0002-room-shop frame shop-reentered-after-open | rom2-ru | 27cf8dd27449cc0227ae8b3a6b1457f375b8f722c105774c8e76115bb1ef01e6 | 27cf8dd27449cc0227ae8b3a6b1457f375b8f722c105774c8e76115bb1ef01e6 | yes |
| 0002-room-shop frame shop-square-after-close | rom2-ru | 9b698e55a7a74c5f7d38eb08e0deb025bbfa20026bc5348dbad1b7ea454bef7e | 9b698e55a7a74c5f7d38eb08e0deb025bbfa20026bc5348dbad1b7ea454bef7e | yes |
| 0002-room-shop frame shop-square-left-open | rom2-ru | d2f89ef35877d40f8af63ab26c8da57c82d968d426946cf0bc42ee178c7558b7 | d2f89ef35877d40f8af63ab26c8da57c82d968d426946cf0bc42ee178c7558b7 | yes |
| 0002-room-shop state lines | rom2-ru | 015b0bc16aac2f9f8e6ba688e6127ac73546b61d4dd5149be716a58cbc56b604 | 015b0bc16aac2f9f8e6ba688e6127ac73546b61d4dd5149be716a58cbc56b604 | yes |
| 0002-room-tavern frame tavern-closed | rom2-ru | ffdfcbff89a11853cc7f5e8866f5b6c6fbe101e786011c7792a344b7652e2a38 | ffdfcbff89a11853cc7f5e8866f5b6c6fbe101e786011c7792a344b7652e2a38 | yes |
| 0002-room-tavern frame tavern-first | rom2-ru | 03ec45e3eaa0aebdb069c126f2e5057ad48e2823088522696a2e51da1b2915e0 | 03ec45e3eaa0aebdb069c126f2e5057ad48e2823088522696a2e51da1b2915e0 | yes |
| 0002-room-tavern frame tavern-reentered-after-close | rom2-ru | 18133b18eac338121c7c62d6c3a55ab9946af74c88f25666a71d4f2c4d962e03 | 18133b18eac338121c7c62d6c3a55ab9946af74c88f25666a71d4f2c4d962e03 | yes |
| 0002-room-tavern frame tavern-reentered-after-open | rom2-ru | 18133b18eac338121c7c62d6c3a55ab9946af74c88f25666a71d4f2c4d962e03 | 18133b18eac338121c7c62d6c3a55ab9946af74c88f25666a71d4f2c4d962e03 | yes |
| 0002-room-tavern frame tavern-square-after-close | rom2-ru | 7e2bfb42b0a41baec2af8755d4db4e1f9acbc15237e828401a0541313bf423ea | 7e2bfb42b0a41baec2af8755d4db4e1f9acbc15237e828401a0541313bf423ea | yes |
| 0002-room-tavern frame tavern-square-left-open | rom2-ru | 5ae4a522d5a690bb254f8300af7745bad3625308b71107b3198464d325a9dfaf | 5ae4a522d5a690bb254f8300af7745bad3625308b71107b3198464d325a9dfaf | yes |
| 0002-room-tavern state lines | rom2-ru | f78c1e21f2c062c793d8d190454ad9c156037c780e3530fb32e1cf130b61272b | f78c1e21f2c062c793d8d190454ad9c156037c780e3530fb32e1cf130b61272b | yes |
| 0002-save frame save-loaded-tavern | rom2-ru | 18133b18eac338121c7c62d6c3a55ab9946af74c88f25666a71d4f2c4d962e03 | 18133b18eac338121c7c62d6c3a55ab9946af74c88f25666a71d4f2c4d962e03 | yes |
| 0002-save frame save-loaded | rom2-ru | 030ac80ff9445771e435dd030ac430ed6304908415168d67c14096fe7c194aae | 030ac80ff9445771e435dd030ac430ed6304908415168d67c14096fe7c194aae | yes |
| 0002-save frame save-saved | rom2-ru | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | yes |
| 0002-save frame save-square | rom2-ru | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | yes |
| 0002-save SAV game0000.sav | rom2-ru | 4b47efbff08bdffb0366628982fec075284b045dfefe3ceb08fd0e296da4297d | 4b47efbff08bdffb0366628982fec075284b045dfefe3ceb08fd0e296da4297d | yes |
| 0002-save state lines | rom2-ru | 367fb91ad684352abd2150c0dc471da229c6e70a2217b6ca4672cbd46649d160 | 367fb91ad684352abd2150c0dc471da229c6e70a2217b6ca4672cbd46649d160 | yes |
| 0002-shop2 frame shop-closed | rom2-ru | 7e9fdd7e7c388c548456c7d28894d7ce7c5f035b9ff414c128ff9e4b5a91acdd | 7e9fdd7e7c388c548456c7d28894d7ce7c5f035b9ff414c128ff9e4b5a91acdd | yes |
| 0002-shop2 frame shop-first | rom2-ru | cd48a7aba0a61dce440de29c40fbb41f6c15f19551bca28406e18886d19effbd | cd48a7aba0a61dce440de29c40fbb41f6c15f19551bca28406e18886d19effbd | yes |
| 0002-shop2 frame shop-reentered-table | rom2-ru | db5f16e43335adb9f409262816d61f00aaee428d32931f7510358f24cb0fc9df | db5f16e43335adb9f409262816d61f00aaee428d32931f7510358f24cb0fc9df | yes |
| 0002-shop2 frame shop-reentered | rom2-ru | 7f4e1f59fe0384a2d7374ef021a2bf4d461ac13aa9a74c82db13160d78c2ecf8 | 7f4e1f59fe0384a2d7374ef021a2bf4d461ac13aa9a74c82db13160d78c2ecf8 | yes |
| 0002-shop2 frame shop-table-again | rom2-ru | db5f16e43335adb9f409262816d61f00aaee428d32931f7510358f24cb0fc9df | db5f16e43335adb9f409262816d61f00aaee428d32931f7510358f24cb0fc9df | yes |
| 0002-shop2 frame shop-table | rom2-ru | 006bb1053d20fe202e7ff472d7a7fbbb6fe0e6e0cf411f24c75e81b88b92a290 | 006bb1053d20fe202e7ff472d7a7fbbb6fe0e6e0cf411f24c75e81b88b92a290 | yes |
| 0002-shop2 frame shop-undo | rom2-ru | 949d3d62671b81dee0d3d645aabe57e04c9b16cd0a608008a8e5b097adf85f79 | 949d3d62671b81dee0d3d645aabe57e04c9b16cd0a608008a8e5b097adf85f79 | yes |
| 0002-shop2 state lines | rom2-ru | 19101b43d70916b6d31212803072424336c04733b64ded10fbb6db575f9e9f19 | 19101b43d70916b6d31212803072424336c04733b64ded10fbb6db575f9e9f19 | yes |
| 0002-square frame square-closed | rom2-ru | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | yes |
| 0002-square frame square-load | rom2-ru | 5b7a94f6baeb5461423e7ea72ac94c8aba79c392c66b8a3f220a0929b2d560a3 | 5b7a94f6baeb5461423e7ea72ac94c8aba79c392c66b8a3f220a0929b2d560a3 | yes |
| 0002-square frame square-return | rom2-ru | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | yes |
| 0002-square state lines | rom2-ru | 7f410c5c7729db2d8b38f163a13d83ebcaff2522ca480dd1a492b8aed872815f | 7f410c5c7729db2d8b38f163a13d83ebcaff2522ca480dd1a492b8aed872815f | yes |
| 0002-talk frame talk-closed | rom2-ru | ffdfcbff89a11853cc7f5e8866f5b6c6fbe101e786011c7792a344b7652e2a38 | ffdfcbff89a11853cc7f5e8866f5b6c6fbe101e786011c7792a344b7652e2a38 | yes |
| 0002-talk frame talk-reread-closed | rom2-ru | ffdfcbff89a11853cc7f5e8866f5b6c6fbe101e786011c7792a344b7652e2a38 | ffdfcbff89a11853cc7f5e8866f5b6c6fbe101e786011c7792a344b7652e2a38 | yes |
| 0002-talk frame talk-reread | rom2-ru | a180f04b55c24de4de80b11b972eb9dc558a5618a119a66cf91dec028f7177b5 | a180f04b55c24de4de80b11b972eb9dc558a5618a119a66cf91dec028f7177b5 | yes |
| 0002-talk frame tavern-in | rom2-ru | 03ec45e3eaa0aebdb069c126f2e5057ad48e2823088522696a2e51da1b2915e0 | 03ec45e3eaa0aebdb069c126f2e5057ad48e2823088522696a2e51da1b2915e0 | yes |
| 0002-talk state lines | rom2-ru | 7578e3d29e482406f95844c1c748e3f6db74ececb4ae7ef77aa9ff0c8950a999 | 7578e3d29e482406f95844c1c748e3f6db74ececb4ae7ef77aa9ff0c8950a999 | yes |
| 0003-room-shop frame shop-closed | rom2-ru | cab0af0587da6967db5408c894096f9870755cd7041ffd6b436b628ce8183c1d | cab0af0587da6967db5408c894096f9870755cd7041ffd6b436b628ce8183c1d | yes |
| 0003-room-shop frame shop-first | rom2-ru | fb5eae1b12af4825c9437b7b3acf8a194b8ff5b7ac695e74defcd11a798578d9 | fb5eae1b12af4825c9437b7b3acf8a194b8ff5b7ac695e74defcd11a798578d9 | yes |
| 0003-room-shop frame shop-reentered-after-close | rom2-ru | c89a9dd673c5097c80cdf97333fa58dcebb54f1519d38d95e6f81a47be819fcf | c89a9dd673c5097c80cdf97333fa58dcebb54f1519d38d95e6f81a47be819fcf | yes |
| 0003-room-shop frame shop-reentered-after-open | rom2-ru | 5ce27b928dc1f0f2ca41447324e392dd77d94faff526a05c4066c913b860be9a | 5ce27b928dc1f0f2ca41447324e392dd77d94faff526a05c4066c913b860be9a | yes |
| 0003-room-shop frame shop-square-after-close | rom2-ru | 9b698e55a7a74c5f7d38eb08e0deb025bbfa20026bc5348dbad1b7ea454bef7e | 9b698e55a7a74c5f7d38eb08e0deb025bbfa20026bc5348dbad1b7ea454bef7e | yes |
| 0003-room-shop frame shop-square-left-open | rom2-ru | d2f89ef35877d40f8af63ab26c8da57c82d968d426946cf0bc42ee178c7558b7 | d2f89ef35877d40f8af63ab26c8da57c82d968d426946cf0bc42ee178c7558b7 | yes |
| 0003-room-shop state lines | rom2-ru | 7769fad47de743482406c82bfe113a7e75419b708f07f129351b30b9922c3a75 | 7769fad47de743482406c82bfe113a7e75419b708f07f129351b30b9922c3a75 | yes |
| 0003-room-tavern frame tavern-closed | rom2-ru | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | yes |
| 0003-room-tavern frame tavern-first | rom2-ru | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | yes |
| 0003-room-tavern frame tavern-reentered-after-close | rom2-ru | 925c26addc0f040b4b3380dca2eee4648389ebd62d1f69ab997ba4a3c0a7b178 | 925c26addc0f040b4b3380dca2eee4648389ebd62d1f69ab997ba4a3c0a7b178 | yes |
| 0003-room-tavern frame tavern-reentered-after-open | rom2-ru | 925c26addc0f040b4b3380dca2eee4648389ebd62d1f69ab997ba4a3c0a7b178 | 925c26addc0f040b4b3380dca2eee4648389ebd62d1f69ab997ba4a3c0a7b178 | yes |
| 0003-room-tavern frame tavern-square-after-close | rom2-ru | 7e2bfb42b0a41baec2af8755d4db4e1f9acbc15237e828401a0541313bf423ea | 7e2bfb42b0a41baec2af8755d4db4e1f9acbc15237e828401a0541313bf423ea | yes |
| 0003-room-tavern frame tavern-square-left-open | rom2-ru | 5ae4a522d5a690bb254f8300af7745bad3625308b71107b3198464d325a9dfaf | 5ae4a522d5a690bb254f8300af7745bad3625308b71107b3198464d325a9dfaf | yes |
| 0003-room-tavern state lines | rom2-ru | 5c9cee4627515c1c40085be2969587bb34ba0fce383e9186c4bdac60e9a1423a | 5c9cee4627515c1c40085be2969587bb34ba0fce383e9186c4bdac60e9a1423a | yes |
| 0003-save frame save-loaded-tavern | rom2-ru | 925c26addc0f040b4b3380dca2eee4648389ebd62d1f69ab997ba4a3c0a7b178 | 925c26addc0f040b4b3380dca2eee4648389ebd62d1f69ab997ba4a3c0a7b178 | yes |
| 0003-save frame save-loaded | rom2-ru | 030ac80ff9445771e435dd030ac430ed6304908415168d67c14096fe7c194aae | 030ac80ff9445771e435dd030ac430ed6304908415168d67c14096fe7c194aae | yes |
| 0003-save frame save-saved | rom2-ru | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | yes |
| 0003-save frame save-square | rom2-ru | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | yes |
| 0003-save SAV game0000.sav | rom2-ru | 8671daa97a0effc178188c62b7934302b1e771ae4b4a0788f500ce991b8eefd5 | 8671daa97a0effc178188c62b7934302b1e771ae4b4a0788f500ce991b8eefd5 | yes |
| 0003-save state lines | rom2-ru | 349078f236054794094f4262497cced1334366a23d90fd960360add9c6b7ae36 | 349078f236054794094f4262497cced1334366a23d90fd960360add9c6b7ae36 | yes |
| 0003-shop2 frame shop-closed | rom2-ru | 7e1a3b3dd67af434b54a23c91224a924ae525b0c078e3bc10f591f4c683b4d6a | 7e1a3b3dd67af434b54a23c91224a924ae525b0c078e3bc10f591f4c683b4d6a | yes |
| 0003-shop2 frame shop-first | rom2-ru | fb5eae1b12af4825c9437b7b3acf8a194b8ff5b7ac695e74defcd11a798578d9 | fb5eae1b12af4825c9437b7b3acf8a194b8ff5b7ac695e74defcd11a798578d9 | yes |
| 0003-shop2 frame shop-reentered-table | rom2-ru | 7a478bb058e1c65a0414a9f116285fcd565508c9fea61649893c51f6a0b6c407 | 7a478bb058e1c65a0414a9f116285fcd565508c9fea61649893c51f6a0b6c407 | yes |
| 0003-shop2 frame shop-reentered | rom2-ru | 1c8f0f8c2d319aaca38a23ecbbb0d8992b4dcc00a7d2dfc7b0a1334fe388592c | 1c8f0f8c2d319aaca38a23ecbbb0d8992b4dcc00a7d2dfc7b0a1334fe388592c | yes |
| 0003-shop2 frame shop-table-again | rom2-ru | 71c9f40e99aa37903feee8d9a08854afbe4b117c95d736219a18289151e4ca01 | 71c9f40e99aa37903feee8d9a08854afbe4b117c95d736219a18289151e4ca01 | yes |
| 0003-shop2 frame shop-table | rom2-ru | aa66e5993f549da3816d407275b5b763df47bd64c0339c57f97145d71cf97f8c | aa66e5993f549da3816d407275b5b763df47bd64c0339c57f97145d71cf97f8c | yes |
| 0003-shop2 frame shop-undo | rom2-ru | b9a501863526a97ad9019b88145821075b6b2230eb35ad490a14a0be3e5e2610 | b9a501863526a97ad9019b88145821075b6b2230eb35ad490a14a0be3e5e2610 | yes |
| 0003-shop2 state lines | rom2-ru | 9b7805464b0d4cd60624e254308b352c95c40cc0a175eb9ff7f804789620115b | 9b7805464b0d4cd60624e254308b352c95c40cc0a175eb9ff7f804789620115b | yes |
| 0003-square frame square-closed | rom2-ru | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | yes |
| 0003-square frame square-load | rom2-ru | 5b7a94f6baeb5461423e7ea72ac94c8aba79c392c66b8a3f220a0929b2d560a3 | 5b7a94f6baeb5461423e7ea72ac94c8aba79c392c66b8a3f220a0929b2d560a3 | yes |
| 0003-square frame square-return | rom2-ru | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | yes |
| 0003-square state lines | rom2-ru | bfb37abfcae7dbe0ea245b82285b989cd425a09f5ea7c6a630669e25fe865e3c | bfb37abfcae7dbe0ea245b82285b989cd425a09f5ea7c6a630669e25fe865e3c | yes |
| 0003-talk frame talk-closed | rom2-ru | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | yes |
| 0003-talk frame talk-reread-closed | rom2-ru | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | yes |
| 0003-talk frame talk-reread | rom2-ru | b5f3950361cb8fae509cb5c78fbf0eafaf6ff9f87dc98ef79bb9df1e90c49df9 | b5f3950361cb8fae509cb5c78fbf0eafaf6ff9f87dc98ef79bb9df1e90c49df9 | yes |
| 0003-talk frame tavern-in | rom2-ru | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | yes |
| 0003-talk state lines | rom2-ru | 0743f4ec3f02bfc2c5879756bf430fea3d3737601bac0df94a5684ad7946e933 | 0743f4ec3f02bfc2c5879756bf430fea3d3737601bac0df94a5684ad7946e933 | yes |
| 9998-room-shop frame shop-closed | rom2-ru | d0ed0fac8b8ab6817085a3cb81fae7cb1b27f9243814974d19f50042af5d8e55 | d0ed0fac8b8ab6817085a3cb81fae7cb1b27f9243814974d19f50042af5d8e55 | yes |
| 9998-room-shop frame shop-first | rom2-ru | 94bfa7668c856ad368575ef707ce71b2ae448bcf7b9c472d785f0ddeed64d088 | 94bfa7668c856ad368575ef707ce71b2ae448bcf7b9c472d785f0ddeed64d088 | yes |
| 9998-room-shop frame shop-reentered-after-close | rom2-ru | 7705c65bde0144495cd1b0589d471bee05642c6c49c918668d07e9499fb26b90 | 7705c65bde0144495cd1b0589d471bee05642c6c49c918668d07e9499fb26b90 | yes |
| 9998-room-shop frame shop-reentered-after-open | rom2-ru | 0c3746af7beec7a230972f97172b2d811d30b740fe0f0fb7eb96439df7c4cfbe | 0c3746af7beec7a230972f97172b2d811d30b740fe0f0fb7eb96439df7c4cfbe | yes |
| 9998-room-shop frame shop-square-after-close | rom2-ru | 9b698e55a7a74c5f7d38eb08e0deb025bbfa20026bc5348dbad1b7ea454bef7e | 9b698e55a7a74c5f7d38eb08e0deb025bbfa20026bc5348dbad1b7ea454bef7e | yes |
| 9998-room-shop frame shop-square-left-open | rom2-ru | d2f89ef35877d40f8af63ab26c8da57c82d968d426946cf0bc42ee178c7558b7 | d2f89ef35877d40f8af63ab26c8da57c82d968d426946cf0bc42ee178c7558b7 | yes |
| 9998-room-shop state lines | rom2-ru | 7769fad47de743482406c82bfe113a7e75419b708f07f129351b30b9922c3a75 | 7769fad47de743482406c82bfe113a7e75419b708f07f129351b30b9922c3a75 | yes |
| 9998-room-tavern frame tavern-closed | rom2-ru | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | yes |
| 9998-room-tavern frame tavern-first | rom2-ru | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | yes |
| 9998-room-tavern frame tavern-reentered-after-close | rom2-ru | 925c26addc0f040b4b3380dca2eee4648389ebd62d1f69ab997ba4a3c0a7b178 | 925c26addc0f040b4b3380dca2eee4648389ebd62d1f69ab997ba4a3c0a7b178 | yes |
| 9998-room-tavern frame tavern-reentered-after-open | rom2-ru | 925c26addc0f040b4b3380dca2eee4648389ebd62d1f69ab997ba4a3c0a7b178 | 925c26addc0f040b4b3380dca2eee4648389ebd62d1f69ab997ba4a3c0a7b178 | yes |
| 9998-room-tavern frame tavern-square-after-close | rom2-ru | 7e2bfb42b0a41baec2af8755d4db4e1f9acbc15237e828401a0541313bf423ea | 7e2bfb42b0a41baec2af8755d4db4e1f9acbc15237e828401a0541313bf423ea | yes |
| 9998-room-tavern frame tavern-square-left-open | rom2-ru | 5ae4a522d5a690bb254f8300af7745bad3625308b71107b3198464d325a9dfaf | 5ae4a522d5a690bb254f8300af7745bad3625308b71107b3198464d325a9dfaf | yes |
| 9998-room-tavern state lines | rom2-ru | 5c9cee4627515c1c40085be2969587bb34ba0fce383e9186c4bdac60e9a1423a | 5c9cee4627515c1c40085be2969587bb34ba0fce383e9186c4bdac60e9a1423a | yes |
| 9998-save frame save-loaded-tavern | rom2-ru | 925c26addc0f040b4b3380dca2eee4648389ebd62d1f69ab997ba4a3c0a7b178 | 925c26addc0f040b4b3380dca2eee4648389ebd62d1f69ab997ba4a3c0a7b178 | yes |
| 9998-save frame save-loaded | rom2-ru | 030ac80ff9445771e435dd030ac430ed6304908415168d67c14096fe7c194aae | 030ac80ff9445771e435dd030ac430ed6304908415168d67c14096fe7c194aae | yes |
| 9998-save frame save-saved | rom2-ru | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | yes |
| 9998-save frame save-square | rom2-ru | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | yes |
| 9998-save SAV game0000.sav | rom2-ru | 5c06780748162a596becd0f8dbe056a5080e8f5c577fe2d59ad89822c22c6cdb | 5c06780748162a596becd0f8dbe056a5080e8f5c577fe2d59ad89822c22c6cdb | yes |
| 9998-save state lines | rom2-ru | 91bbd6b76ccf0eaa522109d69dbe7bdfa56c5d20225a2112569b305c89e65b8c | 91bbd6b76ccf0eaa522109d69dbe7bdfa56c5d20225a2112569b305c89e65b8c | yes |
| 9998-shop2 frame shop-closed | rom2-ru | a53669d9f5b71ad68b9a4378feb6e66870bfc62d72e78a3409839c078a1b5579 | a53669d9f5b71ad68b9a4378feb6e66870bfc62d72e78a3409839c078a1b5579 | yes |
| 9998-shop2 frame shop-first | rom2-ru | 94bfa7668c856ad368575ef707ce71b2ae448bcf7b9c472d785f0ddeed64d088 | 94bfa7668c856ad368575ef707ce71b2ae448bcf7b9c472d785f0ddeed64d088 | yes |
| 9998-shop2 frame shop-reentered-table | rom2-ru | 561998df91deafeacdfec37db479e35b41c2881d6570c9b9fc6c690658cda9e7 | 561998df91deafeacdfec37db479e35b41c2881d6570c9b9fc6c690658cda9e7 | yes |
| 9998-shop2 frame shop-reentered | rom2-ru | 644922d54f75575dd0d6036d03ef3ab7ab8c48e5591ac38d3a4513438fb07510 | 644922d54f75575dd0d6036d03ef3ab7ab8c48e5591ac38d3a4513438fb07510 | yes |
| 9998-shop2 frame shop-table-again | rom2-ru | d1a6dc99758cdda3ece8a3d094bcc6deb6b7a76afc57bf513508ff0b653aa0aa | d1a6dc99758cdda3ece8a3d094bcc6deb6b7a76afc57bf513508ff0b653aa0aa | yes |
| 9998-shop2 frame shop-table | rom2-ru | 2cd7501e9b7fb4603983415fa79c9803ea582951004c9cb168e7045e1308e22c | 2cd7501e9b7fb4603983415fa79c9803ea582951004c9cb168e7045e1308e22c | yes |
| 9998-shop2 frame shop-undo | rom2-ru | 844927055574e291825723776d764926deb09ce7db3969d1cd7b92471b2b0e01 | 844927055574e291825723776d764926deb09ce7db3969d1cd7b92471b2b0e01 | yes |
| 9998-shop2 state lines | rom2-ru | 9b7805464b0d4cd60624e254308b352c95c40cc0a175eb9ff7f804789620115b | 9b7805464b0d4cd60624e254308b352c95c40cc0a175eb9ff7f804789620115b | yes |
| 9998-square frame square-closed | rom2-ru | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | yes |
| 9998-square frame square-load | rom2-ru | 5b7a94f6baeb5461423e7ea72ac94c8aba79c392c66b8a3f220a0929b2d560a3 | 5b7a94f6baeb5461423e7ea72ac94c8aba79c392c66b8a3f220a0929b2d560a3 | yes |
| 9998-square frame square-return | rom2-ru | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | yes |
| 9998-square state lines | rom2-ru | bfb37abfcae7dbe0ea245b82285b989cd425a09f5ea7c6a630669e25fe865e3c | bfb37abfcae7dbe0ea245b82285b989cd425a09f5ea7c6a630669e25fe865e3c | yes |
| 9998-talk frame talk-closed | rom2-ru | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | yes |
| 9998-talk frame talk-reread-closed | rom2-ru | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | yes |
| 9998-talk frame talk-reread | rom2-ru | b5f3950361cb8fae509cb5c78fbf0eafaf6ff9f87dc98ef79bb9df1e90c49df9 | b5f3950361cb8fae509cb5c78fbf0eafaf6ff9f87dc98ef79bb9df1e90c49df9 | yes |
| 9998-talk frame tavern-in | rom2-ru | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | yes |
| 9998-talk state lines | rom2-ru | 0743f4ec3f02bfc2c5879756bf430fea3d3737601bac0df94a5684ad7946e933 | 0743f4ec3f02bfc2c5879756bf430fea3d3737601bac0df94a5684ad7946e933 | yes |
| new-room-shop frame shop-closed | rom2-ru | 0a2a97226a7f94e80497ad5c70dcbcb52eefdd6e1fa0d01f702857ac1316b929 | 0a2a97226a7f94e80497ad5c70dcbcb52eefdd6e1fa0d01f702857ac1316b929 | yes |
| new-room-shop frame shop-first | rom2-ru | 58db68ccc3c9b1f2b9c11f25d61074296efee2416876d549ee52d65d3fffee74 | 58db68ccc3c9b1f2b9c11f25d61074296efee2416876d549ee52d65d3fffee74 | yes |
| new-room-shop frame shop-reentered-after-close | rom2-ru | 104606aab58ac91ca9d45bae158932b58822dedb36fe7162d3c1f06185d3faa6 | 104606aab58ac91ca9d45bae158932b58822dedb36fe7162d3c1f06185d3faa6 | yes |
| new-room-shop frame shop-reentered-after-open | rom2-ru | 104606aab58ac91ca9d45bae158932b58822dedb36fe7162d3c1f06185d3faa6 | 104606aab58ac91ca9d45bae158932b58822dedb36fe7162d3c1f06185d3faa6 | yes |
| new-room-shop frame shop-square-after-close | rom2-ru | 9b698e55a7a74c5f7d38eb08e0deb025bbfa20026bc5348dbad1b7ea454bef7e | 9b698e55a7a74c5f7d38eb08e0deb025bbfa20026bc5348dbad1b7ea454bef7e | yes |
| new-room-shop frame shop-square-left-open | rom2-ru | d2f89ef35877d40f8af63ab26c8da57c82d968d426946cf0bc42ee178c7558b7 | d2f89ef35877d40f8af63ab26c8da57c82d968d426946cf0bc42ee178c7558b7 | yes |
| new-room-shop state lines | rom2-ru | dbcb3562ba435ec7c49a901539b485d61b6de3b6baefbec67bfe9dfdbb2dcef7 | dbcb3562ba435ec7c49a901539b485d61b6de3b6baefbec67bfe9dfdbb2dcef7 | yes |
| new-room-tavern frame tavern-closed | rom2-ru | aaf846069e122d3229f0da3e75c64a0c4d1bdd8fdc7ef9eb2445ef3ddf09af4c | aaf846069e122d3229f0da3e75c64a0c4d1bdd8fdc7ef9eb2445ef3ddf09af4c | yes |
| new-room-tavern frame tavern-first | rom2-ru | b1e5e68b492a4d8e271044c220e6ec7f27336c03f407ed471d10bded8952d6d8 | b1e5e68b492a4d8e271044c220e6ec7f27336c03f407ed471d10bded8952d6d8 | yes |
| new-room-tavern frame tavern-reentered-after-close | rom2-ru | d6e0f0596aba2cfc43bcf9d290e947433c66c883ef3ba9d01fc3d9e2533877f7 | d6e0f0596aba2cfc43bcf9d290e947433c66c883ef3ba9d01fc3d9e2533877f7 | yes |
| new-room-tavern frame tavern-reentered-after-open | rom2-ru | d6e0f0596aba2cfc43bcf9d290e947433c66c883ef3ba9d01fc3d9e2533877f7 | d6e0f0596aba2cfc43bcf9d290e947433c66c883ef3ba9d01fc3d9e2533877f7 | yes |
| new-room-tavern frame tavern-square-after-close | rom2-ru | 7e2bfb42b0a41baec2af8755d4db4e1f9acbc15237e828401a0541313bf423ea | 7e2bfb42b0a41baec2af8755d4db4e1f9acbc15237e828401a0541313bf423ea | yes |
| new-room-tavern frame tavern-square-left-open | rom2-ru | 5ae4a522d5a690bb254f8300af7745bad3625308b71107b3198464d325a9dfaf | 5ae4a522d5a690bb254f8300af7745bad3625308b71107b3198464d325a9dfaf | yes |
| new-room-tavern state lines | rom2-ru | 0ea8d71d7de3671c5e9d3a8a411e030fddcb61e7c4c713be0b136ea19b506049 | 0ea8d71d7de3671c5e9d3a8a411e030fddcb61e7c4c713be0b136ea19b506049 | yes |
| new-save frame save-loaded-tavern | rom2-ru | d6e0f0596aba2cfc43bcf9d290e947433c66c883ef3ba9d01fc3d9e2533877f7 | d6e0f0596aba2cfc43bcf9d290e947433c66c883ef3ba9d01fc3d9e2533877f7 | yes |
| new-save frame save-loaded | rom2-ru | 030ac80ff9445771e435dd030ac430ed6304908415168d67c14096fe7c194aae | 030ac80ff9445771e435dd030ac430ed6304908415168d67c14096fe7c194aae | yes |
| new-save frame save-saved | rom2-ru | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | yes |
| new-save frame save-square | rom2-ru | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | dac14ab517f57db887fc9d73fb05dc93cf44644d68a314e87938cb223ef80a66 | yes |
| new-save SAV game0000.sav | rom2-ru | 40319c03ff19d83db66f94e2432395a9e969e7e3790bc81ab488d1a8b400c5e0 | 40319c03ff19d83db66f94e2432395a9e969e7e3790bc81ab488d1a8b400c5e0 | yes |
| new-save state lines | rom2-ru | ed1c6bfad37e9e4b72e46d961ea73f875e4daec4b375941665e32d78203af350 | ed1c6bfad37e9e4b72e46d961ea73f875e4daec4b375941665e32d78203af350 | yes |
| new-shop2 frame shop-closed | rom2-ru | 1fd0cc00762bd29ae51641098622c3994f1611221aeafdb6e6de2a97fbb53b51 | 1fd0cc00762bd29ae51641098622c3994f1611221aeafdb6e6de2a97fbb53b51 | yes |
| new-shop2 frame shop-first | rom2-ru | 58db68ccc3c9b1f2b9c11f25d61074296efee2416876d549ee52d65d3fffee74 | 58db68ccc3c9b1f2b9c11f25d61074296efee2416876d549ee52d65d3fffee74 | yes |
| new-shop2 frame shop-reentered-table | rom2-ru | 2d846b51ddf20c2e9e62b84428a4e5b1508b5bbccb041380e81c726e33fcb5f6 | 2d846b51ddf20c2e9e62b84428a4e5b1508b5bbccb041380e81c726e33fcb5f6 | yes |
| new-shop2 frame shop-reentered | rom2-ru | 31e68398237bd16bc735e1bde17269b724e9527fefbe6144e9e74caac53daa94 | 31e68398237bd16bc735e1bde17269b724e9527fefbe6144e9e74caac53daa94 | yes |
| new-shop2 frame shop-table-again | rom2-ru | 2d846b51ddf20c2e9e62b84428a4e5b1508b5bbccb041380e81c726e33fcb5f6 | 2d846b51ddf20c2e9e62b84428a4e5b1508b5bbccb041380e81c726e33fcb5f6 | yes |
| new-shop2 frame shop-table | rom2-ru | 1524a522987eec89a88b61880fbe16da8f40159447ae52881083c6820759f447 | 1524a522987eec89a88b61880fbe16da8f40159447ae52881083c6820759f447 | yes |
| new-shop2 frame shop-undo | rom2-ru | 7d36e44130560eac5c4bf64335277f489c1188e3b2eb0e050b7c6272174398be | 7d36e44130560eac5c4bf64335277f489c1188e3b2eb0e050b7c6272174398be | yes |
| new-shop2 state lines | rom2-ru | e706475d99ce19a8e0f3092c9c40700c992f52e13998350cd1310c9a0fc5e5aa | e706475d99ce19a8e0f3092c9c40700c992f52e13998350cd1310c9a0fc5e5aa | yes |
| new-square frame square-closed | rom2-ru | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | eb9d7102783e22342c908df19f3718f8204d5e735d4cbd7995517949b8d7d907 | yes |
| new-square frame square-load | rom2-ru | 5b7a94f6baeb5461423e7ea72ac94c8aba79c392c66b8a3f220a0929b2d560a3 | 5b7a94f6baeb5461423e7ea72ac94c8aba79c392c66b8a3f220a0929b2d560a3 | yes |
| new-square frame square-return | rom2-ru | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | 99feec150b3d32aae9b38f4cc61bf47d9225ed8afc14afd6c12be99b38c51bb2 | yes |
| new-square state lines | rom2-ru | db331f9d05088c25893e21445dc702a03756460d7accd267c252e1725e343c59 | db331f9d05088c25893e21445dc702a03756460d7accd267c252e1725e343c59 | yes |
| new-talk frame talk-closed | rom2-ru | aaf846069e122d3229f0da3e75c64a0c4d1bdd8fdc7ef9eb2445ef3ddf09af4c | aaf846069e122d3229f0da3e75c64a0c4d1bdd8fdc7ef9eb2445ef3ddf09af4c | yes |
| new-talk frame talk-reread-closed | rom2-ru | aaf846069e122d3229f0da3e75c64a0c4d1bdd8fdc7ef9eb2445ef3ddf09af4c | aaf846069e122d3229f0da3e75c64a0c4d1bdd8fdc7ef9eb2445ef3ddf09af4c | yes |
| new-talk frame talk-reread | rom2-ru | 9bea1b8a2d01844920c173ccdd2189a04f37c351d089cdf0db7bcad542542763 | 9bea1b8a2d01844920c173ccdd2189a04f37c351d089cdf0db7bcad542542763 | yes |
| new-talk frame tavern-in | rom2-ru | b1e5e68b492a4d8e271044c220e6ec7f27336c03f407ed471d10bded8952d6d8 | b1e5e68b492a4d8e271044c220e6ec7f27336c03f407ed471d10bded8952d6d8 | yes |
| new-talk state lines | rom2-ru | 38212643f1f2aeae4604928e1221774514ec017300974f1c57874d0f6f7e7939 | 38212643f1f2aeae4604928e1221774514ec017300974f1c57874d0f6f7e7939 | yes |
| 9711-room-school frame school-closed | ru | 687f566f6ab4a810a1fbb7e0afe00ffa07770cb34741a36bde3ca7bb18aba6b9 | 687f566f6ab4a810a1fbb7e0afe00ffa07770cb34741a36bde3ca7bb18aba6b9 | yes |
| 9711-room-school frame school-first | ru | 1c794cd2299d0dfbb8ba45681ff2add80af187e67e8be69fcbefd11608289831 | 1c794cd2299d0dfbb8ba45681ff2add80af187e67e8be69fcbefd11608289831 | yes |
| 9711-room-school frame school-reentered-after-close | ru | 0d1fc4809c94df94f3b38bde9d77802dc939e4dbee970f827149d0f49f0b9119 | 0d1fc4809c94df94f3b38bde9d77802dc939e4dbee970f827149d0f49f0b9119 | yes |
| 9711-room-school frame school-reentered-after-open | ru | 0d1fc4809c94df94f3b38bde9d77802dc939e4dbee970f827149d0f49f0b9119 | 0d1fc4809c94df94f3b38bde9d77802dc939e4dbee970f827149d0f49f0b9119 | yes |
| 9711-room-school frame school-square-after-close | ru | 8b3830f1bb2d99b5d8c8e7f61c251da54023749328f0e894469199ca094431fa | 8b3830f1bb2d99b5d8c8e7f61c251da54023749328f0e894469199ca094431fa | yes |
| 9711-room-school frame school-square-left-open | ru | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | yes |
| 9711-room-school state lines | ru | 7ce549452d81f817fc9030144d12e592eaac9206d1f84ff5bb02a5d81beb09e7 | 7ce549452d81f817fc9030144d12e592eaac9206d1f84ff5bb02a5d81beb09e7 | yes |
| 9711-room-shop frame shop-closed | ru | 0cfdb5919a9b9ca1fcbb62d12222e6454e0fef7b19b19bc4fb812255632060f2 | 0cfdb5919a9b9ca1fcbb62d12222e6454e0fef7b19b19bc4fb812255632060f2 | yes |
| 9711-room-shop frame shop-first | ru | 16e97ef18775a0f5f8146d67685a56d747eab2a7f95a04e0f86340e6d496e982 | 16e97ef18775a0f5f8146d67685a56d747eab2a7f95a04e0f86340e6d496e982 | yes |
| 9711-room-shop frame shop-reentered-after-close | ru | 32d9d81f3c5e21003a369b4d85e42c115b1dcf5789754b592d63ff816d166750 | 32d9d81f3c5e21003a369b4d85e42c115b1dcf5789754b592d63ff816d166750 | yes |
| 9711-room-shop frame shop-reentered-after-open | ru | 32d9d81f3c5e21003a369b4d85e42c115b1dcf5789754b592d63ff816d166750 | 32d9d81f3c5e21003a369b4d85e42c115b1dcf5789754b592d63ff816d166750 | yes |
| 9711-room-shop frame shop-square-after-close | ru | 1eb30511bd221c0c1beba3f79a5379d8369160fd9fb87ea5183c9966236591bf | 1eb30511bd221c0c1beba3f79a5379d8369160fd9fb87ea5183c9966236591bf | yes |
| 9711-room-shop frame shop-square-left-open | ru | 641251743b4573c20b7e8857bc60bcd964004cd5fadbf4afdad0a7e7606d66a1 | 641251743b4573c20b7e8857bc60bcd964004cd5fadbf4afdad0a7e7606d66a1 | yes |
| 9711-room-shop state lines | ru | d3561a3724e8103fd10a946c6fcb89bc177079a98fa8f3b9228e306d494fc12a | d3561a3724e8103fd10a946c6fcb89bc177079a98fa8f3b9228e306d494fc12a | yes |
| 9711-room-tavern frame tavern-closed | ru | ad5cb23daf163de531227475481af181f62cb71f646b8c39a62854eea309b9ed | ad5cb23daf163de531227475481af181f62cb71f646b8c39a62854eea309b9ed | yes |
| 9711-room-tavern frame tavern-first | ru | 05b9420d599d5cc70542bdd4048b200bb28fef2cc157f9393b0d8a8e3d491b14 | 05b9420d599d5cc70542bdd4048b200bb28fef2cc157f9393b0d8a8e3d491b14 | yes |
| 9711-room-tavern frame tavern-reentered-after-close | ru | 4fa63b97a20a6c4c8cc07d477e5eca7c3a0a8087a56de28c499554f43d10f824 | 4fa63b97a20a6c4c8cc07d477e5eca7c3a0a8087a56de28c499554f43d10f824 | yes |
| 9711-room-tavern frame tavern-reentered-after-open | ru | 4fa63b97a20a6c4c8cc07d477e5eca7c3a0a8087a56de28c499554f43d10f824 | 4fa63b97a20a6c4c8cc07d477e5eca7c3a0a8087a56de28c499554f43d10f824 | yes |
| 9711-room-tavern frame tavern-square-after-close | ru | 8b3830f1bb2d99b5d8c8e7f61c251da54023749328f0e894469199ca094431fa | 8b3830f1bb2d99b5d8c8e7f61c251da54023749328f0e894469199ca094431fa | yes |
| 9711-room-tavern frame tavern-square-left-open | ru | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | yes |
| 9711-room-tavern state lines | ru | e58e42af1dea3f3f191fa2ade4da9fbb6459acd3d68b9c2d7c9e65b21259a5e8 | e58e42af1dea3f3f191fa2ade4da9fbb6459acd3d68b9c2d7c9e65b21259a5e8 | yes |
| 9711-save frame save-loaded-tavern | ru | 4fa63b97a20a6c4c8cc07d477e5eca7c3a0a8087a56de28c499554f43d10f824 | 4fa63b97a20a6c4c8cc07d477e5eca7c3a0a8087a56de28c499554f43d10f824 | yes |
| 9711-save frame save-loaded | ru | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | yes |
| 9711-save frame save-saved | ru | 6afff4582a3dc364cc90cfc5a37132f4504c5c53a077d51233a32f9625bb3b56 | 6afff4582a3dc364cc90cfc5a37132f4504c5c53a077d51233a32f9625bb3b56 | yes |
| 9711-save frame save-square | ru | 1eb30511bd221c0c1beba3f79a5379d8369160fd9fb87ea5183c9966236591bf | 1eb30511bd221c0c1beba3f79a5379d8369160fd9fb87ea5183c9966236591bf | yes |
| 9711-save SAV tip proof.sav | ru | af59b54981f55ee03507029a3c393ebdca038e93f59ba2214020f75c80902b60 | af59b54981f55ee03507029a3c393ebdca038e93f59ba2214020f75c80902b60 | yes |
| 9711-save state lines | ru | fa4aef9ee4f03c7faa70eb8b39867fabc60de5eb2781545b4973e2601f7f7ffd | fa4aef9ee4f03c7faa70eb8b39867fabc60de5eb2781545b4973e2601f7f7ffd | yes |
| 9711-shop2 frame shop-closed | ru | 25d314a315196f2d9276d7281de181d6af4e01169672c1003576f73868ef8b79 | 25d314a315196f2d9276d7281de181d6af4e01169672c1003576f73868ef8b79 | yes |
| 9711-shop2 frame shop-first | ru | 16e97ef18775a0f5f8146d67685a56d747eab2a7f95a04e0f86340e6d496e982 | 16e97ef18775a0f5f8146d67685a56d747eab2a7f95a04e0f86340e6d496e982 | yes |
| 9711-shop2 frame shop-reentered-table | ru | eabae603f986d5de5bcf32684cab886d27f6c44dd673602a3c2cafd9c00f21a5 | eabae603f986d5de5bcf32684cab886d27f6c44dd673602a3c2cafd9c00f21a5 | yes |
| 9711-shop2 frame shop-reentered | ru | 32d9d81f3c5e21003a369b4d85e42c115b1dcf5789754b592d63ff816d166750 | 32d9d81f3c5e21003a369b4d85e42c115b1dcf5789754b592d63ff816d166750 | yes |
| 9711-shop2 frame shop-table-again | ru | 6442c63ee8d9dfc04d70babc6cc494fe531548fafdd89f3627e45c5a1789a63f | 6442c63ee8d9dfc04d70babc6cc494fe531548fafdd89f3627e45c5a1789a63f | yes |
| 9711-shop2 frame shop-table | ru | eabae603f986d5de5bcf32684cab886d27f6c44dd673602a3c2cafd9c00f21a5 | eabae603f986d5de5bcf32684cab886d27f6c44dd673602a3c2cafd9c00f21a5 | yes |
| 9711-shop2 frame shop-undo | ru | d846f19218658a074a3270dcf85a9e7731a458057c716a5e4f1fa505b027af20 | d846f19218658a074a3270dcf85a9e7731a458057c716a5e4f1fa505b027af20 | yes |
| 9711-shop2 state lines | ru | 432c8a7e75666eafdf858774306bc961ff7a4e6de3833866a15c9a5bafbc3308 | 432c8a7e75666eafdf858774306bc961ff7a4e6de3833866a15c9a5bafbc3308 | yes |
| 9711-square frame square-closed | ru | f6affd12134ec5c4635a29f300cce98f7ae70088cdbbb1cd91a5b3e124b2f1f2 | f6affd12134ec5c4635a29f300cce98f7ae70088cdbbb1cd91a5b3e124b2f1f2 | yes |
| 9711-square frame square-load | ru | 4833e140020c2600299d7bb0059114dc343a26bc3695b1a232c7ee615bf5eda5 | 4833e140020c2600299d7bb0059114dc343a26bc3695b1a232c7ee615bf5eda5 | yes |
| 9711-square frame square-return | ru | edd30127d29bdd2fe15793d167baab697ccb45510af0fe6808deb286102917e8 | edd30127d29bdd2fe15793d167baab697ccb45510af0fe6808deb286102917e8 | yes |
| 9711-square state lines | ru | 9deac23fe6d2bd4465ce49e9256e3166acdb23a763a4ee3692d2d3f7ff112dad | 9deac23fe6d2bd4465ce49e9256e3166acdb23a763a4ee3692d2d3f7ff112dad | yes |
| 9711-talk frame talk-closed | ru | efea4c3439883b9b148b420b5082d6545db35c1961b28f3220eada00ecadf949 | efea4c3439883b9b148b420b5082d6545db35c1961b28f3220eada00ecadf949 | yes |
| 9711-talk frame talk-reread-closed | ru | efea4c3439883b9b148b420b5082d6545db35c1961b28f3220eada00ecadf949 | efea4c3439883b9b148b420b5082d6545db35c1961b28f3220eada00ecadf949 | yes |
| 9711-talk frame talk-reread | ru | 7d4892aee4558a876f4773ad41170a118025e62b10280128477cdc6d1b65a733 | 7d4892aee4558a876f4773ad41170a118025e62b10280128477cdc6d1b65a733 | yes |
| 9711-talk frame tavern-in | ru | 05b9420d599d5cc70542bdd4048b200bb28fef2cc157f9393b0d8a8e3d491b14 | 05b9420d599d5cc70542bdd4048b200bb28fef2cc157f9393b0d8a8e3d491b14 | yes |
| 9711-talk state lines | ru | 62e35884f07972e88f260f5fd8cff6c8acdee47afeab9f050e116a9f6eb0a68e | 62e35884f07972e88f260f5fd8cff6c8acdee47afeab9f050e116a9f6eb0a68e | yes |
| 9714-room-school frame school-closed | ru | 31fe86f09e90fa344e82c79f6e35154831aef8c5849d468697c5fc90aea094f1 | 31fe86f09e90fa344e82c79f6e35154831aef8c5849d468697c5fc90aea094f1 | yes |
| 9714-room-school frame school-first | ru | 0be1d338cb9d14552548b507b7602acfecdeffbe73cc0e26ea9e5dd14f38230f | 0be1d338cb9d14552548b507b7602acfecdeffbe73cc0e26ea9e5dd14f38230f | yes |
| 9714-room-school frame school-reentered-after-close | ru | bbf8e5d139c8fdca28e27569cf75a926c27b56dfd371b886edbb071140907f63 | bbf8e5d139c8fdca28e27569cf75a926c27b56dfd371b886edbb071140907f63 | yes |
| 9714-room-school frame school-reentered-after-open | ru | bbf8e5d139c8fdca28e27569cf75a926c27b56dfd371b886edbb071140907f63 | bbf8e5d139c8fdca28e27569cf75a926c27b56dfd371b886edbb071140907f63 | yes |
| 9714-room-school frame school-square-after-close | ru | 1eb30511bd221c0c1beba3f79a5379d8369160fd9fb87ea5183c9966236591bf | 1eb30511bd221c0c1beba3f79a5379d8369160fd9fb87ea5183c9966236591bf | yes |
| 9714-room-school frame school-square-left-open | ru | 641251743b4573c20b7e8857bc60bcd964004cd5fadbf4afdad0a7e7606d66a1 | 641251743b4573c20b7e8857bc60bcd964004cd5fadbf4afdad0a7e7606d66a1 | yes |
| 9714-room-school state lines | ru | 7e3bb6901eb0b6c6c58199e952b9043a2e9bc09fdb18a7098db52432611fc0bc | 7e3bb6901eb0b6c6c58199e952b9043a2e9bc09fdb18a7098db52432611fc0bc | yes |
| 9714-room-shop frame shop-closed | ru | 4d314eea33b19ca9ab4f8956964a396305c27b57a44b441cc60561accd6126e0 | 4d314eea33b19ca9ab4f8956964a396305c27b57a44b441cc60561accd6126e0 | yes |
| 9714-room-shop frame shop-first | ru | 1179d93b09bdc5277fb25a44fb653db071873772ddf86be3a0cbcee65e63ca22 | 1179d93b09bdc5277fb25a44fb653db071873772ddf86be3a0cbcee65e63ca22 | yes |
| 9714-room-shop frame shop-reentered-after-close | ru | fc91f17660ecb85f53df9b3409178eb5fecbbfb40ebaa11e456a44601a626879 | fc91f17660ecb85f53df9b3409178eb5fecbbfb40ebaa11e456a44601a626879 | yes |
| 9714-room-shop frame shop-reentered-after-open | ru | fc91f17660ecb85f53df9b3409178eb5fecbbfb40ebaa11e456a44601a626879 | fc91f17660ecb85f53df9b3409178eb5fecbbfb40ebaa11e456a44601a626879 | yes |
| 9714-room-shop frame shop-square-after-close | ru | 1eb30511bd221c0c1beba3f79a5379d8369160fd9fb87ea5183c9966236591bf | 1eb30511bd221c0c1beba3f79a5379d8369160fd9fb87ea5183c9966236591bf | yes |
| 9714-room-shop frame shop-square-left-open | ru | 641251743b4573c20b7e8857bc60bcd964004cd5fadbf4afdad0a7e7606d66a1 | 641251743b4573c20b7e8857bc60bcd964004cd5fadbf4afdad0a7e7606d66a1 | yes |
| 9714-room-shop state lines | ru | f37a467eb72148f7491e684c485f61f7252912e6f05d8da327ea04bcfe6c826d | f37a467eb72148f7491e684c485f61f7252912e6f05d8da327ea04bcfe6c826d | yes |
| 9714-room-tavern frame tavern-closed | ru | e3ff0bbb6fb57131119eecf29127e2946de9b2de7d9a2d598df2d660652d0358 | e3ff0bbb6fb57131119eecf29127e2946de9b2de7d9a2d598df2d660652d0358 | yes |
| 9714-room-tavern frame tavern-first | ru | 2da8cef2dd0704b67df791a2c4661ece9fcbb3901655a35fa455886e427c9e85 | 2da8cef2dd0704b67df791a2c4661ece9fcbb3901655a35fa455886e427c9e85 | yes |
| 9714-room-tavern frame tavern-reentered-after-close | ru | 89bc2d9da46155bacb23859e5cbd2b4c7b254d67f883d4afe319fec45a87f856 | 89bc2d9da46155bacb23859e5cbd2b4c7b254d67f883d4afe319fec45a87f856 | yes |
| 9714-room-tavern frame tavern-reentered-after-open | ru | 89bc2d9da46155bacb23859e5cbd2b4c7b254d67f883d4afe319fec45a87f856 | 89bc2d9da46155bacb23859e5cbd2b4c7b254d67f883d4afe319fec45a87f856 | yes |
| 9714-room-tavern frame tavern-square-after-close | ru | 8b3830f1bb2d99b5d8c8e7f61c251da54023749328f0e894469199ca094431fa | 8b3830f1bb2d99b5d8c8e7f61c251da54023749328f0e894469199ca094431fa | yes |
| 9714-room-tavern frame tavern-square-left-open | ru | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | yes |
| 9714-room-tavern state lines | ru | c869d66d1eae1cb5f1108fce2b422327fe6c4d7521af124cf11cf81e983f570f | c869d66d1eae1cb5f1108fce2b422327fe6c4d7521af124cf11cf81e983f570f | yes |
| 9714-save frame save-loaded-tavern | ru | 83f0dc77185d77d8e584d4dd8228be9499cf385f7e96b2491efc899e450c49ba | 83f0dc77185d77d8e584d4dd8228be9499cf385f7e96b2491efc899e450c49ba | yes |
| 9714-save frame save-loaded | ru | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | yes |
| 9714-save frame save-saved | ru | 6afff4582a3dc364cc90cfc5a37132f4504c5c53a077d51233a32f9625bb3b56 | 6afff4582a3dc364cc90cfc5a37132f4504c5c53a077d51233a32f9625bb3b56 | yes |
| 9714-save frame save-square | ru | 1eb30511bd221c0c1beba3f79a5379d8369160fd9fb87ea5183c9966236591bf | 1eb30511bd221c0c1beba3f79a5379d8369160fd9fb87ea5183c9966236591bf | yes |
| 9714-save SAV tip proof.sav | ru | 3fa2a334b5ad6bd862511c66c4dcbe174a864e54cf7fab22c0c473f7f410a67b | 3fa2a334b5ad6bd862511c66c4dcbe174a864e54cf7fab22c0c473f7f410a67b | yes |
| 9714-save state lines | ru | c394cc43f1a56f542d8a432e9782e52c111d4677377526a11ae209b18360c616 | c394cc43f1a56f542d8a432e9782e52c111d4677377526a11ae209b18360c616 | yes |
| 9714-shop2 frame shop-closed | ru | 8ba52f6eeda35df9ab1f0b6c5e20f436eecdf0acb6b575e2d9318c372c5a3b0d | 8ba52f6eeda35df9ab1f0b6c5e20f436eecdf0acb6b575e2d9318c372c5a3b0d | yes |
| 9714-shop2 frame shop-first | ru | 1179d93b09bdc5277fb25a44fb653db071873772ddf86be3a0cbcee65e63ca22 | 1179d93b09bdc5277fb25a44fb653db071873772ddf86be3a0cbcee65e63ca22 | yes |
| 9714-shop2 frame shop-reentered-table | ru | 73ac5a24960f72157a79c95f19c27b4db53f801df397d2626b207f2f1a4137d1 | 73ac5a24960f72157a79c95f19c27b4db53f801df397d2626b207f2f1a4137d1 | yes |
| 9714-shop2 frame shop-reentered | ru | fc91f17660ecb85f53df9b3409178eb5fecbbfb40ebaa11e456a44601a626879 | fc91f17660ecb85f53df9b3409178eb5fecbbfb40ebaa11e456a44601a626879 | yes |
| 9714-shop2 frame shop-table-again | ru | 73ac5a24960f72157a79c95f19c27b4db53f801df397d2626b207f2f1a4137d1 | 73ac5a24960f72157a79c95f19c27b4db53f801df397d2626b207f2f1a4137d1 | yes |
| 9714-shop2 frame shop-table | ru | 73ac5a24960f72157a79c95f19c27b4db53f801df397d2626b207f2f1a4137d1 | 73ac5a24960f72157a79c95f19c27b4db53f801df397d2626b207f2f1a4137d1 | yes |
| 9714-shop2 frame shop-undo | ru | a71dda5fb653587eabd5fdac90b368efd8f3811c1721f090c9afa7f074292d0c | a71dda5fb653587eabd5fdac90b368efd8f3811c1721f090c9afa7f074292d0c | yes |
| 9714-shop2 state lines | ru | 3bc3a27e2bf430cc3e704233d56d59fd72539247a2e1f3ca874a2cd95e5ef113 | 3bc3a27e2bf430cc3e704233d56d59fd72539247a2e1f3ca874a2cd95e5ef113 | yes |
| 9714-square frame square-closed | ru | f6affd12134ec5c4635a29f300cce98f7ae70088cdbbb1cd91a5b3e124b2f1f2 | f6affd12134ec5c4635a29f300cce98f7ae70088cdbbb1cd91a5b3e124b2f1f2 | yes |
| 9714-square frame square-load | ru | 4833e140020c2600299d7bb0059114dc343a26bc3695b1a232c7ee615bf5eda5 | 4833e140020c2600299d7bb0059114dc343a26bc3695b1a232c7ee615bf5eda5 | yes |
| 9714-square frame square-return | ru | edd30127d29bdd2fe15793d167baab697ccb45510af0fe6808deb286102917e8 | edd30127d29bdd2fe15793d167baab697ccb45510af0fe6808deb286102917e8 | yes |
| 9714-square state lines | ru | 7ef2c4bb20d3d19d472802d86b62fabc6871bcca1a46f680128446013a62509e | 7ef2c4bb20d3d19d472802d86b62fabc6871bcca1a46f680128446013a62509e | yes |
| 9714-talk frame talk-closed | ru | f87e9b6214c7e729eb5c31b24bdcf264b4b6942d4466724a85bdd74a6cbfa1c7 | f87e9b6214c7e729eb5c31b24bdcf264b4b6942d4466724a85bdd74a6cbfa1c7 | yes |
| 9714-talk frame talk-reread-closed | ru | f87e9b6214c7e729eb5c31b24bdcf264b4b6942d4466724a85bdd74a6cbfa1c7 | f87e9b6214c7e729eb5c31b24bdcf264b4b6942d4466724a85bdd74a6cbfa1c7 | yes |
| 9714-talk frame talk-reread | ru | 94d5cd63ebb4105e3e86723f664933d69e6493349ff11eed7010b47b26438856 | 94d5cd63ebb4105e3e86723f664933d69e6493349ff11eed7010b47b26438856 | yes |
| 9714-talk frame tavern-in | ru | 2da8cef2dd0704b67df791a2c4661ece9fcbb3901655a35fa455886e427c9e85 | 2da8cef2dd0704b67df791a2c4661ece9fcbb3901655a35fa455886e427c9e85 | yes |
| 9714-talk state lines | ru | 02bd56307d2672ed49c73eec5a600ccd2067e8bf4f24b2afa86e28bb1cc2c2dc | 02bd56307d2672ed49c73eec5a600ccd2067e8bf4f24b2afa86e28bb1cc2c2dc | yes |
| 9719-room-school frame school-closed | ru | 31e310631bf4e33b18c1db3f93d613b982a5c8b39a87771134c2296ffa3293f8 | 31e310631bf4e33b18c1db3f93d613b982a5c8b39a87771134c2296ffa3293f8 | yes |
| 9719-room-school frame school-first | ru | 0cdf40282f6d0e49bdf67951dd858f447df9e7a03f803e89c6f4bf168f5aed10 | 0cdf40282f6d0e49bdf67951dd858f447df9e7a03f803e89c6f4bf168f5aed10 | yes |
| 9719-room-school frame school-reentered-after-close | ru | 23f631cf023072d33d6b0c383bb059de1202b8e85f7307e3a9d10222aa33d6ea | 23f631cf023072d33d6b0c383bb059de1202b8e85f7307e3a9d10222aa33d6ea | yes |
| 9719-room-school frame school-reentered-after-open | ru | 23f631cf023072d33d6b0c383bb059de1202b8e85f7307e3a9d10222aa33d6ea | 23f631cf023072d33d6b0c383bb059de1202b8e85f7307e3a9d10222aa33d6ea | yes |
| 9719-room-school frame school-square-after-close | ru | 8b3830f1bb2d99b5d8c8e7f61c251da54023749328f0e894469199ca094431fa | 8b3830f1bb2d99b5d8c8e7f61c251da54023749328f0e894469199ca094431fa | yes |
| 9719-room-school frame school-square-left-open | ru | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | yes |
| 9719-room-school state lines | ru | 7153546b42f77d4ce6e1bad2dda15f1b34bb7a9cae3972926650ed4b0af539bd | 7153546b42f77d4ce6e1bad2dda15f1b34bb7a9cae3972926650ed4b0af539bd | yes |
| 9719-room-shop frame shop-closed | ru | ac512d74556a024290e5fde190503a8123ebdb448ddf8ad1dd4e2ca7e43efde7 | ac512d74556a024290e5fde190503a8123ebdb448ddf8ad1dd4e2ca7e43efde7 | yes |
| 9719-room-shop frame shop-first | ru | 0c5720a676947a81e34f2bdec5312373ebc0e2fce863f33867371137864dbb47 | 0c5720a676947a81e34f2bdec5312373ebc0e2fce863f33867371137864dbb47 | yes |
| 9719-room-shop frame shop-reentered-after-close | ru | 1b1923c96243027316eae0da6bafe61f00f61490a96010007cb4bb39455177a2 | 1b1923c96243027316eae0da6bafe61f00f61490a96010007cb4bb39455177a2 | yes |
| 9719-room-shop frame shop-reentered-after-open | ru | 1b1923c96243027316eae0da6bafe61f00f61490a96010007cb4bb39455177a2 | 1b1923c96243027316eae0da6bafe61f00f61490a96010007cb4bb39455177a2 | yes |
| 9719-room-shop frame shop-square-after-close | ru | 1eb30511bd221c0c1beba3f79a5379d8369160fd9fb87ea5183c9966236591bf | 1eb30511bd221c0c1beba3f79a5379d8369160fd9fb87ea5183c9966236591bf | yes |
| 9719-room-shop frame shop-square-left-open | ru | 641251743b4573c20b7e8857bc60bcd964004cd5fadbf4afdad0a7e7606d66a1 | 641251743b4573c20b7e8857bc60bcd964004cd5fadbf4afdad0a7e7606d66a1 | yes |
| 9719-room-shop state lines | ru | 6e3973211497b82c64d0563ba8a390529e0da6588b874fe3246ca3368c9c7be8 | 6e3973211497b82c64d0563ba8a390529e0da6588b874fe3246ca3368c9c7be8 | yes |
| 9719-room-tavern frame tavern-closed | ru | 356b2853b67783c3694f89a1331558884f6d339280e63c134b9a9e48e7bcaf81 | 356b2853b67783c3694f89a1331558884f6d339280e63c134b9a9e48e7bcaf81 | yes |
| 9719-room-tavern frame tavern-first | ru | f62ee238f7075be18d149f4e4540d016093b1f1aa5094a784211bdac0c46387c | f62ee238f7075be18d149f4e4540d016093b1f1aa5094a784211bdac0c46387c | yes |
| 9719-room-tavern frame tavern-reentered-after-close | ru | c43332f9164eba6f4008681697d2821c34926f1b58a74ba1b8f2794a2ed3c3f9 | c43332f9164eba6f4008681697d2821c34926f1b58a74ba1b8f2794a2ed3c3f9 | yes |
| 9719-room-tavern frame tavern-reentered-after-open | ru | c43332f9164eba6f4008681697d2821c34926f1b58a74ba1b8f2794a2ed3c3f9 | c43332f9164eba6f4008681697d2821c34926f1b58a74ba1b8f2794a2ed3c3f9 | yes |
| 9719-room-tavern frame tavern-square-after-close | ru | 8b3830f1bb2d99b5d8c8e7f61c251da54023749328f0e894469199ca094431fa | 8b3830f1bb2d99b5d8c8e7f61c251da54023749328f0e894469199ca094431fa | yes |
| 9719-room-tavern frame tavern-square-left-open | ru | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | yes |
| 9719-room-tavern state lines | ru | e7c87b3c50ee548472f5f1ebc259aa47d0352cec67f1628665041b0234fd8f3b | e7c87b3c50ee548472f5f1ebc259aa47d0352cec67f1628665041b0234fd8f3b | yes |
| 9719-save frame save-loaded-tavern | ru | c43332f9164eba6f4008681697d2821c34926f1b58a74ba1b8f2794a2ed3c3f9 | c43332f9164eba6f4008681697d2821c34926f1b58a74ba1b8f2794a2ed3c3f9 | yes |
| 9719-save frame save-loaded | ru | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | 0843bbd3ff7226b82fdfd4f93294ab862339fefd3aad87c193cdd25b5c809701 | yes |
| 9719-save frame save-saved | ru | 6afff4582a3dc364cc90cfc5a37132f4504c5c53a077d51233a32f9625bb3b56 | 6afff4582a3dc364cc90cfc5a37132f4504c5c53a077d51233a32f9625bb3b56 | yes |
| 9719-save frame save-square | ru | 1eb30511bd221c0c1beba3f79a5379d8369160fd9fb87ea5183c9966236591bf | 1eb30511bd221c0c1beba3f79a5379d8369160fd9fb87ea5183c9966236591bf | yes |
| 9719-save SAV tip proof.sav | ru | 3681d639f81726534a0a092e8d8e8f34b71ca07e95077d29dc19f4ea0464d540 | 3681d639f81726534a0a092e8d8e8f34b71ca07e95077d29dc19f4ea0464d540 | yes |
| 9719-save state lines | ru | 63d8ad5d3b276db8c213a62d7a9974c67842470903146f9053cea6407caa9632 | 63d8ad5d3b276db8c213a62d7a9974c67842470903146f9053cea6407caa9632 | yes |
| 9719-shop2 frame shop-closed | ru | 3fa51f915596387f17de9c61fd7b0f6f876932a0ed364c9cb1fe6218f1967161 | 3fa51f915596387f17de9c61fd7b0f6f876932a0ed364c9cb1fe6218f1967161 | yes |
| 9719-shop2 frame shop-first | ru | 0c5720a676947a81e34f2bdec5312373ebc0e2fce863f33867371137864dbb47 | 0c5720a676947a81e34f2bdec5312373ebc0e2fce863f33867371137864dbb47 | yes |
| 9719-shop2 frame shop-reentered-table | ru | 7759f6eb4bafed67644e57ddc15ebaec2bde379a668f47b6f7cfcb61056680cb | 7759f6eb4bafed67644e57ddc15ebaec2bde379a668f47b6f7cfcb61056680cb | yes |
| 9719-shop2 frame shop-reentered | ru | 1b1923c96243027316eae0da6bafe61f00f61490a96010007cb4bb39455177a2 | 1b1923c96243027316eae0da6bafe61f00f61490a96010007cb4bb39455177a2 | yes |
| 9719-shop2 frame shop-table-again | ru | 04fcd2c9d434e957c4bc637c129b33041271e5842ac3bc3275700ea2cb88e61e | 04fcd2c9d434e957c4bc637c129b33041271e5842ac3bc3275700ea2cb88e61e | yes |
| 9719-shop2 frame shop-table | ru | 7759f6eb4bafed67644e57ddc15ebaec2bde379a668f47b6f7cfcb61056680cb | 7759f6eb4bafed67644e57ddc15ebaec2bde379a668f47b6f7cfcb61056680cb | yes |
| 9719-shop2 frame shop-undo | ru | 0eac55a98a8149e673d63a879ce9e5103c9f1c3859edb24f1c63f7ee4953a573 | 0eac55a98a8149e673d63a879ce9e5103c9f1c3859edb24f1c63f7ee4953a573 | yes |
| 9719-shop2 state lines | ru | a0bbd7f80b4168b217967921c49e1273b8c9d9d9a75c78c66445e1f7b924d870 | a0bbd7f80b4168b217967921c49e1273b8c9d9d9a75c78c66445e1f7b924d870 | yes |
| 9719-square frame square-closed | ru | f6affd12134ec5c4635a29f300cce98f7ae70088cdbbb1cd91a5b3e124b2f1f2 | f6affd12134ec5c4635a29f300cce98f7ae70088cdbbb1cd91a5b3e124b2f1f2 | yes |
| 9719-square frame square-load | ru | 4833e140020c2600299d7bb0059114dc343a26bc3695b1a232c7ee615bf5eda5 | 4833e140020c2600299d7bb0059114dc343a26bc3695b1a232c7ee615bf5eda5 | yes |
| 9719-square frame square-return | ru | edd30127d29bdd2fe15793d167baab697ccb45510af0fe6808deb286102917e8 | edd30127d29bdd2fe15793d167baab697ccb45510af0fe6808deb286102917e8 | yes |
| 9719-square state lines | ru | 2bf20d2296cb12abcb027292548ac30e0e74b779ed2979f1a845670f3a7026dd | 2bf20d2296cb12abcb027292548ac30e0e74b779ed2979f1a845670f3a7026dd | yes |
| 9719-talk frame talk-closed | ru | 0a683fdc79c3c903c1667d453f49fca3b4037c77596b81d332421698d7438634 | 0a683fdc79c3c903c1667d453f49fca3b4037c77596b81d332421698d7438634 | yes |
| 9719-talk frame talk-reread-closed | ru | 0a683fdc79c3c903c1667d453f49fca3b4037c77596b81d332421698d7438634 | 0a683fdc79c3c903c1667d453f49fca3b4037c77596b81d332421698d7438634 | yes |
| 9719-talk frame talk-reread | ru | 11040f68e02a03d7269eda8db8fca7cb7844b4b6a39cce7b5df1ce24a24b3ef8 | 11040f68e02a03d7269eda8db8fca7cb7844b4b6a39cce7b5df1ce24a24b3ef8 | yes |
| 9719-talk frame tavern-in | ru | f62ee238f7075be18d149f4e4540d016093b1f1aa5094a784211bdac0c46387c | f62ee238f7075be18d149f4e4540d016093b1f1aa5094a784211bdac0c46387c | yes |
| 9719-talk state lines | ru | a93dd565826a846e06e5efc2ecd1b4252b8aa3b84828870c1b68c2e06bfa256f | a93dd565826a846e06e5efc2ecd1b4252b8aa3b84828870c1b68c2e06bfa256f | yes |

## Open debt

- The first game's per-room closed state was observable only for the room
  shown, since every tipped room's entry builds its popup (TOWN-516); the
  one state keeps that. A future rule that keeps a closed popup closed
  across entries is a new `rooms` field, not a lifecycle branch.
- The second game's shop second text stays absent (DIV-2968): `rom2.json`'s
  shop row carries no `second` text, so the one shop path never retexts it.
- The tip tables are not moddable; `rooms` is engine data only. A mod town
  takes its game's rules.
- The town description decoder refuses unknown fields. A mod's town
  description that copied a shipped `host` line with `tip-lifecycle` is now
  refused; `host` is not documented in `docs/MODS.md`, and no `.json`,
  `.star`, `.toml` or `.go` file of the repository carries the field.
- The Russian first-game runs read the EN kit saves; there is no Russian
  first-game town kit.
