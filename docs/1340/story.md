# 1340 — Campaign progression regression

## Result

`TestReleaseCampaignChainFromMission10` chains the Main line on a lawful
install: fresh mission 10, win, mission 20 (SAVE and cold LOAD inside it), win,
town, town SAVE and cold LOAD, mission 30 through the world map, win, town,
mission 40 through the world map. It runs on EN and RU in about 1 s per root
after the install loads. No defect was found. No divergence row and no
engine behaviour change.

## Intent

Owner direction (backlog: wider campaign regression). Per-mission release
tests enter each mission from a constructed prerequisite, and the document
test chains only the document grant. Nothing asserted payment, transition
reward, companion carry, town services and world-map offers as one chain.

## Authority

No ROM1 behaviour is claimed. Expected payments are read from the installed
`scenario.reg` `Payment` keys, independently of the campaign decoder. The 500
gold paid on completing mission 20 is the product's authored transition reward.

## As built

`pkg/game/campaignchain_release_test.go`. Wins reuse the objective actions of
`scenarios/1060-campaign-NNN-reachable.json` and the App's frames, so the
Victory notice, return and town arrival are production routes. This is a
progression regression, not a combat playthrough.

| step | assertions |
|---|---|
| mission 10 entry | three collected pages, one physical document |
| mission 10 win | won; purse = purse at victory + payment 10; auto-entry of 20 |
| SAVE in mission 20, cold LOAD | live mission 20, mission 10 done, 20 open |
| mission 20 win, town | purse = victory purse + payment 20 + 500; chapter 30; companion 22 carried; tavern offers 30, shop offers 31, school none; pages kept |
| SAVE in town, cold LOAD | town screen; same state; same availability |
| mission 30 via world map | offer accepted and available; enabled scroll; travel; companion in the mission party |
| mission 30 win, town | purse + payment 30; chapter 40; companion kept |
| mission 40 via world map | enabled scroll; entry; pages kept |

Loss controls (`loss_controls` subtest) each remove one carrier from the town
after mission 20 and require the town check to fail: the third collected page
in the SAV, the carried companion, and the 500 reward from the purse.

The population manifest gains the test. Seven shared test helpers lose the
`1176` suffix so the storyguard identifier count falls; the baseline moves
accordingly (identifiers 5310 to 5242, comment bytes up for the new test).

## Proof

EN and RU release run of the test, the two root scenario sets are unchanged,
`go test -trimpath -count=1 -p 2 ./...`, `scripts/check-no-game-assets.sh`.

## Open debt

The chain stops at the entry of mission 40. Missions 50 and later, and the
Side and Offered lines, stay with the per-mission and document tests.
