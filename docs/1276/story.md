# Base profiles, and the ROM1 demo as a base

## Intent and authority

The engine names the install it runs on and states what that install lacks. The ROM1 demo may run as a base: the engine runs on the demo install, and the demo's content is not restored into the release game. Authority is owner direction (`review/mod-milestone/PROPOSAL.md`, "Bases", "Base profile"). English and Russian behaviour is unchanged except one added `-check` line. ROM1 differences are rows `DIV-1888` to `DIV-1892` in `docs/divergences/mods.md`; `DIV-1893` is returned unused.

## As built

Profiles (`pkg/base`). A `Profile` holds an id, a title, a language, extra required files, known builds (size and SHA-256 of `main.res`) and `Limits` (no character generation, first mission, original-save refusal, notes). The zero `Limits` states no limit. Three profiles are known: `rom1-en`, `rom1-ru`, `rom1-demo`. A root that holds the five archives and matches none is the generic `rom1`.

Detection (`base.Detect`). Read-only:

1. List the directory, names case-folded.
2. The five archives are required; a missing one is a `NotInstallError`.
3. Stat `main.res`. Only when its size equals a known build, read it once for the SHA-256.
4. A digest match then needs the profile's extra files; a lack is a `PartialError` naming them. The demo needs `video4.res`.
5. With no digest match, the language entry of `main.res` (read by `pkg/game`) selects `rom1-en` or `rom1-ru`; otherwise the generic profile.

The demo build is `main.res` of 4822928 bytes. `InspectInstall` carries the match (`InstallInfo.Base`) and a detection error (`BaseErr`); `Valid` is false for either. `FrontEnd` takes the match from the shared install resources (`Archives.Base`), so no `FrontEnd` field was added. A profile with no character generation skips loading generation art. `mods.BaseID` returns the profile id; an `applies-to = "rom1"` mod matches all three.

Command (`cmd/againrom`). `-base <id>` requires the detected profile and exits 2 with a message naming both on a difference, a partial base, or a root that is not an install. `-check` prints `againrom: base <id> (<title>; exact build)` and one `base limit:` line per stated limit, and omits the generation line on a no-generation base. The GUI launch prints the same lines to stderr. Without `-mission`, the default mission is the profile's first mission. An explicit `-mission` on a no-generation base is refused with the base and its first mission named. `-picker` keeps the map list.

Demo new game. NEW GAME on a no-generation base opens the profile's first mission (`ui.App.SetNewGameDirect`, `FrontEnd.DirectNewGame`) on a fresh town with the engine's default party at Normal difficulty, with no generation screen. A first mission the install cannot open falls back to the picker with the reason on its message line.

Starter (`cmd/starter`). The base choice rows and the status line name the detected profile and its limits. Play and the launch plan pass `-base <id>` after `-assets <root>` for a known, non-generic profile. `pkg/base` is registered in the `internal/archtest` DAG; the starter may import it.

## What the demo reaches

Root: `gameversions/rom1-demo` (15 files, read-only).

- Starts and shows its own main menu from its own archive.
- NEW GAME opens mission 41 and runs 200 ticks headless with two party members.
- Loading its own `game9999.sav` (through a temporary store copy) is refused by the original-save reader, because the campaign inn arrays differ in length (3 NPCs, 2 missions). The refusal appends `(base rom1-demo: <limit>)`. The profile states this limit (`DIV-1890`).

## Proof

- Unit: `pkg/base/base_test.go` (exact build, language fallback, generic, partial base refused naming the file, missing archives, unreadable root, case folding, `Require`, known profiles); `pkg/game/installinfo_test.go` (`BaseID`, typed errors, partial profile); `cmd/againrom/base_test.go` (`-base` parse and refusals, `-check` naming and limits, direct new game, fallback to the picker, `-mission` refusal, `-picker`); `cmd/starter/base_test.go`.
- Demo witnesses (`pkg/game/demobase_test.go`): found at the default absolute path, overridden by `AGAINROM_DEMO_ASSETS`; they skip only when the root is absent and name no environment variable in the skip, so the release gate does not count them as gated. They cover detection, the main-menu frame, new game to mission 41 and the refused save.
- Release, EN and RU, one root at a time (gated by `AGAINROM_ASSETS`): `TestReleaseBaseProfileMatchesTheInstall` (exact profile by language, no limit, one base line) and `TestReleaseCheckNamesTheBase` (`-check -base <id>` exits 0 naming an exact build; the other release id exits 2).
- Screenshot: `review/story1276-demo/demo-main-menu.png`, composed offscreen from the demo archive. `TestDemoBaseMainMenuFrame` writes it under `AGAINROM_DEMO_SHOTS`, otherwise under `t.TempDir()`.

## Open debt

- Town, school, documents and the screens after a won mission are unwitnessed on the demo. Only missions 41, 51 and 91 ship.
- The demo's own save is refused (`DIV-1890`). Whether the engine can read it with a tolerant inn-array reader is Unknown.
- The demo new game uses the engine's default party, not a demo-authored one (`DIV-1889`).
- No mission-map screenshot of the demo: the map screen has no CPU composite.
- The starter's base rows were tested through the model, not by driving its window.
- The demo's content is not shipped into the release game (`DIV-1891`).
