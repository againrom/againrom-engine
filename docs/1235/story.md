# Two save points

## Intent and authority

Owner direction: SAVE has exactly two states, mission and city. ROM1 city starts at mission 30. Homeward world-map travel is presentation between these states and cannot become a third saved state. Legacy pending SAV files remain loadable and resolve to their city without repeating mission rewards.

## As-built behaviour

Town F2 and menu SAVE ask the game whether the city has been reached. Pending inbound travel denies both. Snapshot capture blocks automatic and named SAVE, and the common SAV producer rejects a legacy pending snapshot. Mission SAVE, F3 and ordinary city rooms retain their routes. An idle or outward world map saves the city without a travel field. The campaign's existing TownBegins boundary still determines when a city exists.

Legacy WorldMapReturn remains readable. LOAD resolves it through the ordinary return arrival writer before publishing the city. That writer sets the home position and first-map-point latch, clears travel and opens the square. It does not complete a mission or pay a reward. Missing registry art still yields a settled city. No game state field or save version is added.

Shop transactions with staged items are outside this story. Ordinary shop eligibility does not prove staged-item preservation.

## Proof

Red evidence at review/story1235-two-save-points: F2 and menu admitted pending SAVE; simultaneous F2 swallowed F3; ExportCurrentSave wrote a pending state; legacy LOAD resumed travel. Receipts record exit 1 and no remaining processes.

Focused UI/game tests pass. They cover denied homeward SAVE, mission/city F2, F3, named and automatic producer admission, legacy pending SAV LOAD, party/gold/journal/campaign parity against ordinary arrival, cold city SAVE and the next school debit. Function-key and menu regression tests pass. Storyguard and the gated-test population check pass; comment bytes fall by 159 and numbered test identifiers by three.

EN and RU pass TestReleaseTwoSavePointsMissionAndCity. It drives mission 10/20 SAVE, the first city, pending F2/menu and F3, normal city rooms, idle and outward world-map SAVE, and legacy pending LOAD through the application. The resolved city is saved, loaded into a fresh front end, and the next Gates action starts at home. TestReleaseVictoryAppReturnsHomeAndBlocksPendingSave passes four controlled victory/Continue cases in both installs, including automatic arrival, denied pending SAVE and accepted city SAVE. Both bounded installed receipts report exit 0 and no remaining processes.

The edited generated-city and native town-return release controls pass on EN. Imported town-return subcases were skipped because this lane did not set AGAINROM_SAVE_CORPUS; the final acceptance gate owns that corpus. Receipt and JSON evidence stays under review/story1235-two-save-points.

## Open debt

Final gates, sole review and promotion are pending. No new ROM1 claim is asserted. The script-gap census is unaffected by this save admission change and was not run in the lane. No windowed original-runtime claim is made.
