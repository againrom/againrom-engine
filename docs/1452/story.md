# 1452 — Campaign chain from mission 40 to the end

## Result

`TestReleaseCampaignChainFromMission10` now chains the first game's Main line
on a lawful install from a fresh mission 10 to the end of the campaign: the
Victory of mission 150, the credits, the hall of fame and the main menu. Every
town between missions is saved and cold-loaded, every Main line mission from
40 on is saved and cold-loaded inside, and each of the 13 side missions the
towns offer (31, 41, 51, 61, 71, 81, 91, 101, 111, 121, 131, 141, 151) is
entered from a cold LOAD of the town that offers it, won and returned. It
passes on EN and RU in about 21 s per root inside the release set (12
workers; 19 s on EN alone). No engine defect was found; no divergence row and no
engine behaviour change.

## Intent

Owner direction (backlog: wider campaign regression). Story1340 stopped at the
entry of mission 40, so a defect in missions 40 to 150 or in the side missions
reached the owner first.

## Authority

No ROM1 behaviour is claimed beyond the installed data and these claims.

- Payments, offers, pages and unlocks are read from the installed
  `scenario.reg` with `pkg/formats/reg`, independently of the campaign
  decoder: `Payment` (REG-SCN-067), `InnMission` paired with `InnNPC`,
  `ShopMission` and `TCMission`, the inn's zero as an NPC with nothing to give
  (REG-SCN-064), `EnableMercenary` (REG-SCN-067), `AddTextDocument` and
  `AddPictureDocum` as one append deduplicated on the pair (REG-SCN-097).
- Companions: `AddHero` 22 in `[Mission30]` is the one town grant; the map
  companions are mission 40 unit 6 `npc25`, mission 100 unit 245 `npc24` and
  mission 140 unit 443 `npc26` (HERO-JOIN-120, HERO-JOIN-121). The chain's
  objective actions bring exactly these three into the party.
- Campaign end: `LastMission` ships once, `[Mission150] = 1` (REG-SCN-063).
  A selected main mission divisible by 10 with that flag takes the terminal
  dispatch (SAV-890), which binds the credits (SAV-970), then the hall of fame
  and the campaign reset (SAV-971); no save point follows (SAV-972, DIV-1264).
- The 500 gold paid on completing mission 20 is the product's authored
  transition reward, the only one.

## Campaign end on shipped data

Mission 150 is the last Main line mission and the only `LastMission` section.
Its script Victory, acknowledged, shows the credits; the town is not reopened.
At the credits the campaign is complete, the chapter is 0, mission 150 is
done, the purse is the victory purse plus payment 150 (none), and the current
state is refused as a SAV. One key ends the credits on the hall of fame; the
hall leaves to the main menu. Side mission 151 is offered beside 150 in the
same town and does not end the campaign.

## As built

`pkg/game/campaignchain_release_test.go`. Wins reuse the objective actions of
`scenarios/1060-campaign-NNN-reachable.json` against the App's World and App
frames, so Victory notice, return and town arrival are production routes. The
objective helper in `pkg/game/documents1176_release_test.go` gains the four
actions the later scenarios use: `kill_player`, `heal`, `order` (appended to
the front end's order queue) and `pick_item` (the player's pick-up order, the
sack taken by App frames). This is a progression regression, not a combat
playthrough.

Mission 130 is the one exception. Its scenario wins by waiting, but only with a
party without NPC22, the controlled prerequisite of scenario 1060. With NPC22
carried, as in the chain, waiting 3000 frames does not win. The chain uses the
installed betrayal route of `TestReleaseMission130KilledCompanionLeavesParty`:
NPC22 walks to Veglud (unit 135), the installed Give Unit hands it to player
8, and Veglud's side is defeated.

| step | assertions |
|---|---|
| mission 10 entry and win | purse 100 on entry; three pages, one physical document; won; purse = victory purse + payment 10; mission-10 `EnableMercenary` types unlocked; auto-entry of 20 |
| SAVE in mission 20, cold LOAD | live mission 20, 10 done, 20 open; the loaded mission purse equals the purse before the SAVE |
| mission 20 win, town 30 | purse = victory purse + payment 20 + 500; the transition rewards are exactly {20: 500} |
| every town 30 to 150 | town open, previous mission done, chapter and offered chapter; purse; pages as the registry grants them up to the mission won; one physical document on the primary; carried companions; tavern, shop and school missions as the chapter's section lists them |
| town SAVE, cold LOAD | town screen; the same town assertions; the same availability |
| world-map entry | the mission purse equals the town purse before the entry; offer accepted and available; scrolls are the accepted missions, each with its installed payment; the chosen scroll enabled; travel; the mission party's companions are the carried ones |
| mission entry 40 to 150 | pages through the mission's own grants (4 from 50, 5 with the picture from 60) |
| SAVE in mission 40 to 150, cold LOAD | live mission, previous done, this one open, the same companions, the same pages, the loaded mission purse equal to the purse before the SAVE |
| main win, town | purse = victory purse + payment; chapter + 10; companions: npc25 joins in 40, npc24 in 100, npc26 in 140, NPC22 leaves in 130; `EnableMercenary` unlocked |
| each side mission | from a cold LOAD of its town: entry by world map, pages kept, win, return; side done; purse = victory purse + side payment; chapter kept; main mission not done; the side offer consumed, the others kept; side `EnableMercenary` types locked before and unlocked after; town SAVE and cold LOAD keep it; the chapter's main mission can still be accepted |
| mission 150 win | credits, campaign complete, chapter 0, final purse, SAV refused; one key to the hall of fame; main menu |

The test keeps its own purse ledger, so a change the engine makes to the
purse between town and Victory fails it. Each Victory purse must equal the
mission's entry purse plus its gain, and each town purse the Victory purse plus
the installed payment (and 500 after mission 20). The gain is derived or pinned:

- Derived from the installed data: payments (`scenario.reg`); mission 141's
  Give Money of 1500000, which the test requires the installed map script to
  hold (opcode 23).
- Pinned constants, equal on EN and RU, not stated by the installed data: the
  entry purse 100 at mission 10; 500 gained inside mission 20; 783 gained
  inside mission 40; the purse 192883 at the credits.

Purse at the towns on EN and RU: 2100 at 40; 2883 from 50 to 90; 72883 at
100; 192883 from 110 to the end. Mission 30's Give Money of 1000 and mission
131's of 10000 are in the installed scripts; the chain's objective actions fire
neither, and the ledger holds no gain for them.

Loss controls, each required to make its check fail:

| subtest | carrier removed |
|---|---|
| `loss_controls/document_grant_removed` | the third page in the town-30 SAV |
| `loss_controls/companion_carry_removed` | NPC22 from the town-30 party |
| `loss_controls/payment_removed` | the 500 reward from the purse |
| `loss_controls/offer_consumed` | the shop's mission-31 offer, taken in a cold copy |
| `purse_loss_control` | the loaded mission-60 purse changed by one gold either way |
| `picture_loss_control` | the picture page in the town-70 SAV |
| `side_41/loss_controls/side_completion_removed` | side mission 41's completion |
| `side_41/loss_controls/mercenary_unlock_removed` | the type mission 41 unlocks |
| `departed_companion_kept` | NPC22 put back into the town-140 party |
| `ending_loss_control/terminal_completion_removed` | mission 150's completion at the credits |

Four numbered test helpers lose their suffix (`documentsFrame`,
`documentsState`, `campaignMenu`, and the chain's own betrayal helper), so the
storyguard identifier count falls from 4551 to 4539.

## Proof

EN and RU release runs of the test; the whole release set through
`check-release-tests.sh` on EN and RU; `go test -trimpath -count=1 ./...`;
`go vet ./pkg/game`; `gofmt -l`; `scripts/check-no-game-assets.sh`. The
release set ran EN 1141 and RU 1139 of 1245 gated tests with no failure.

## Open debt

- With NPC22 absent, mission 130 wins about 31 ticks after entry by waiting
  (scenario 1060-130 and a controlled prerequisite). Whether ROM1 does the same
  for a player whose NPC22 died earlier is Unknown; the chain does not claim it.
- The mission-70 companion `npc23`, mission 30's Give Money 1000 and mission
  131's Give Money 10000 are not reached by the scenarios' objective actions;
  the chain asserts none of them. Mission 130's win is by the headless betrayal
  route (`HeadlessPlace`, `HeadlessKillPlayer`), as in the existing
  mission-130 test.
- The 500 gained in mission 20 and the 783 gained in mission 40 have no
  installed source the test reads; they are pinned.
- Wins are objective actions, not combat; tavern hiring, shop purchases and
  school lessons are not exercised between missions.
