# The starter shows every setting of a mod

## Result

A player can now set every setting a mod declares in the starter, not only the first five.
The settings panel scrolls with the wheel like the other two lists. Each kind of setting
has a control that fits it, and a value the game would refuse is shown refused in the
game's own words.

## Authority

- Owner: «понять как настраивать мод» (mod workbench point 2). The starter is our own
  program; no original behaviour is involved.

## Rules

1. Every declared setting is reachable by the wheel; the heading shows "1-5 of 12".
2. The heading line shows the window and the count.
3. A number has a field with - and + that stop at its bounds; a switch is on or off.

## Code

| kind | its one builder | change |
|---|---|---|
| scrolling list | `cmd/starter/app.go` scrollList | the panel is its third list |

## Proof

| test | rules |
|---|---|
| `TestEverySettingOfAModIsReachableByTheWheel` | 1, 2 |
| `TestEverySettingIsEditedStoredAndPassed` | 3 |

Release tests at the release: none.

## Open debt

- A setting has no description in the declaration format; it waits for a format change.
