# F2/F3 verification

## Boundary and observable result

Base `befe948d1e28f3135cf17ac961fcb106896735d1`; reconciled master
`512eb42b69240a374e92175a2a82c8cae4bada8e` before corrected-candidate checks.
The research pin advances with that master to
`0d829843102d57aa4778aa4407749a2044fe3938`. The F2/F3 authority was read at the
earlier accepted pin `4b49d6524016a32cc0f3414c122a7c327f6fe613`.

Previously neither physical F2 nor F3 reached its original action. The new
bindings reach existing Save and Load from campaign map/town, and Diplomacy
from a standalone map. On each preserved EN/RU install, the production release
test writes one new native save from each campaign surface and restores that
surface through F3. Standalone F2 writes none; F3 shows Diplomacy (EN four rows,
RU five). Empty-store F3 opens Load without relying on the disabled menu row.

The built executable also runs `scenarios/1080-function-keys.json` on both
installs: all 17 steps pass, including F2 save, Load cancellation and immediate
menu Escape, F3 restoration, and equality of the restored character. Each run
writes one 58,160-byte native save to an explicit untracked review directory.
This is a production headless drive, not a live-window or original-game witness;
no synthetic desktop input was sent. The seat owns the landed current build.

```text
againrom.exe -assets <en-or-ru-root> -saves <external-review-dir> --headless scenarios/1080-function-keys.json
AGAINROM_ASSETS=<en-or-ru-root> go test -trimpath -count=1 ./pkg/game -run '^TestReleaseFunctionKeysSaveLoadAndDiplomacy$' -v
```

## Independent checks

AI-KEY-125 supplies keyboard.tsv rows 12-15. The test table transcribes actions
and contexts independently of menuRows; bindingSource checks exact physical
F2/F3 and unchanged F9/F10/F11 expressions. MENU-ITEM-011/012 supply existing
destinations, not the key's empty-store gate. SESS-INPUT-037 supplies focused
child precedence. App.step tests cover map/town, campaign/standalone, Save and
Load errors, acknowledgement, empty stores, successful Load, unfocused input,
notices including dismissal, popup/text/other-screen guards, town dialogue,
same-frame side input and the explicitly retained simultaneous-key policy.

An independent elapsed-time fixture holds Load for 31 seconds, then sends
Escape to the menu and immediate Escape or Return to the map. No simulation or
ambient frame is repaid on dismissal; the next ordinary 100-ms frame advances
only 1..2 simulation ticks and at most one ambient frame. The external scenario
does not claim this timing check: its wait_ticks command accepts map only.

Focused tests and gofmt pass. The final-code asset-free
`go test -trimpath -count=1 ./...` passes at corrected code commit
`195dda1136b98dc97cbd149a526edf7d64a80af5`; the evidence-record edit is prose
only. `git diff --check` passes and no commit has a
Co-Authored-By trailer. `check-no-game-assets.sh` prints `clean (tree scan)`.
Both installed release witnesses pass; the new test is registered in the gated
population. The seat's final paired release invocation must run on its merge.

Allocation sweeps before/after the ledger edits print `missing answers: 0`.
Only DIV-526 is consumed; DIV-527 is unused. `check-div-claims.sh` exits 0 at
pin 0d82984: 263 live rows, 340 cited IDs, 64 existing partial-retraction matches;
neither changed row is among those matches. The preserved-install check after
the executable runs prints `ok — 181 file(s), both roots as recorded`.

## Mission census and builds

The master baseline is unchanged. VCS-stamped missionrun with EN assets and
`-mission <N> -trace -ticks 1` exits 0 for both missions:

| mission | baseline checks / instants / triggers | master unsupported | candidate unsupported |
|---|---|---|---|
| 10 | 16 / 27 / 12 | 0 | 0 |
| 20 | 14 / 15 / 11 | 0 | 0 |

The population is not affected by this input slice. Builds use command-scoped
GIT_DIR/GIT_WORK_TREE and ephemeral safe.directory entries for this worktree;
no build disables VCS stamping. go version -m reports the exact code revision
and vcs.modified=false for both againrom and missionrun. Untracked binaries,
scenario traces and save outputs live under seat review/1080/.

## Open debt

DIV-099 retains the authored Save/Load dialogs and immediate generated-name
save/acknowledgement. DIV-294 is not wholly closed: F1 Help remains absent.
DIV-526 records press-edge-only repeats and F2 priority for simultaneous valid
F2/F3; standalone F2 refusal leaves F3 eligible. No Windows key-repeat runtime
equivalence is claimed. Future story 1074 must keep its cutscene early return
before these actions; it was not merged and no unaccepted research was used.
Correction verification, landing and current-build publication remain seat work.

## Sole review correction

Seat report `pipeline/reviews/1080-adversarial.md` returns only the continuing
mouse gesture. The regression uses literal destination coordinates, not
menuRows: all five earlier-down routes fail before the fix; their same-frame
controls pass. The fix extends release suppression to the sampled PrimaryDown
level. All ten cases now pass, including two intervening held frames and a
usable fresh click afterward. Focus, notice and popup guards remain unchanged.
The focused function-key/popup/release tests and both EN/RU release tests pass.
The corrected executable at 195dda11 stamps that exact revision with
vcs.modified=false; its 17-step scenario passes again on both roots. These
traces and saves are under review/1080/corrected-{en,ru} at the seat. The
preserved-install check still reports all 181 files unchanged. No second
review is requested; the seat verifies this one correction and lands it.
