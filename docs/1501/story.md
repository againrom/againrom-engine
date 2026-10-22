# One table of launch settings for the starter and the game

## Intent

Each launch setting the starter keeps in its `[options]` section and passes to the game is one row
of one table, `launch.Settings` in `pkg/launch/settings.go`. A row states the settings file key,
the kind, the default, the bounds, the game flag, the game's default and how the game reads a
number. The starter's window, its settings file read and write, the argument list it builds and the
game's flag parse all walk that table, so a bound is stated once. Behaviour is unchanged: the same
settings, defaults, bounds, captions in both languages, settings file bytes, argument lists and
refusals.

## Authority

- Owner decision on audit `pipeline/reviews/arch-audit-29d32509.md`, rank 28 (launch settings):
  one table of setting rows {key, type, default, bound, caption IDs, game flag, apply}.
- Seat rule "One builder per kind". Refactor only; no ROM1 fact moves and no claim is cited.

## As built

### The table

`pkg/launch/settings.go` (the package already owns how one Againrom program starts another) holds:

- `Setting`: `Key`, `Kind`, `Default`, `Options`, `Min`, `Max`, `Read`, `GameDefault`, `Flag`,
  `Excludes`.
- Kinds: `Switch` (passes `-flag` or `-flag=false` when it differs from the default), `Override`
  (sound: default, on, off; on and off set the game's bool flag), `Choice` (video: each non-first
  value is its own game flag, `-4x`, `-8x`), `Number` (decimal whole number from `Min` to `Max`;
  blank means the default; the default value passes nothing), `Text` (passed when not blank),
  `Arguments` (the free extra arguments, split by `launch.Split`, passed last).
- `Read` says how the game reads a number: `ReadInt` (flag package integer, refused after parsing:
  volume), `ReadWhole` (decimal, refused while parsing: shop stock multiplier), `ReadAtStart`
  (flag package integer; the game refuses a mission number that names no mission when it starts).
- The bounds `MaxVolume = 100` and `MaxShopStockMultiplier = 1000` are constants beside the rows;
  `game.MaxShopStockMultiplier` is that constant.
- Row methods: `Load` (settings file text to the kept value), `Next` (the button's next value),
  `Args` (one row's arguments, or a `*Refusal`), `Whole`, `Bounds`, `InBounds`, `GameBool`,
  `GameInt`. `launch.Build` walks the rows in table order and refuses a value set while the switch
  its row excludes is on (mission with picker).

The table order is the order the settings file keeps the keys and the order the starter passed the
arguments, so both stay byte for byte.

### The starter

`cmd/starter/options.go` holds `optionViews`, one view per row: the `settings` field holding the
value, the field or button that edits it, the settings view row, the label, refusal and clash
captions, and the captions of an Override's or Choice's values. `defaultSettings`, `loadSettings`,
`store`, `launchArgs`, the settings view layout, the value buttons, the switch toggles,
`app.field` and `fieldShown` walk the table and these views. The local `parseBool`, `formatBool`,
`splitArgs` and the sound and video value constants are gone; `launch.ParseBool`,
`launch.FormatBool` and `launch.Split` replace them.

### The game

`cmd/againrom/launchflags.go` binds each game flag of the table to the `options` field it sets and
its help text (`launchFlags`), defines the flags from the rows (`registerLaunchFlags`), and checks
`ReadInt` bounds and row exclusions after parsing. `parse` no longer defines those thirteen flags
one by one. `pkg/game.ParseShopStockMultiplier` and `FrontEnd.SetShopStockMultiplier` read the
shop row.

### Before and after

| Statement | Before (base `41408076`) | After |
|---|---|---|
| Shop multiplier bound | `pkg/game/shopmultiplier.go:8`, literal `1..1000` in `cmd/againrom/main.go:109` and `:474`, checks at `shopmultiplier.go:12`, `:19` and through them `cmd/starter/settings.go:389`, `cmd/againrom/main.go:475` | `pkg/launch/settings.go` constant and row; `Whole`/`InBounds` are the one check; help texts and captions format the row's bounds |
| Volume bound | `cmd/starter/settings.go:342` (`100`), `cmd/againrom/main.go:539` (`audio.MasterUnit`), help text `:189` | `launch.Volume`; the game's check is `Bounds`; a test holds it equal to `audio.MasterUnit` |
| Defaults | `cmd/starter/settings.go:79`, `:107-127`; game `main.go:118`, `:173-176`, `:471-526` | row `Default` and `GameDefault` |
| Settings file keys | `settings.go:107-127` (read), `:164-176` (write) | row `Key`, one loop each |
| Arguments | `settings.go:334-400` | `Setting.Args`, `launch.Build` |
| Game flags | `main.go:471-526`, Visit `:534-538`, checks `:539-547` | `registerLaunchFlags`, `checkLaunchBounds`, `checkLaunchExclusions` |
| Window controls | `app.go:379-390`, `:478-504`, `:666-678`, `:791-810`, `:839` | `optionViews` and loops in `app.go` |

## Settings outside the table

- `[starter] againrom`, `close-on-play`, `last-base` and `[bases]`: the starter's own; no game
  flag.
- `[mods] dir`, `enabled`, `accept-unmarked` and the `[mod.<id>]` values: their arguments
  (`-mods`, `-mods-dir`, `-mod-setting`, `-mods-accept-unmarked`) are passed only when mods are
  enabled, and each mod declares its own settings in its manifest. That is a different row shape.
- The map editor's test play (`pkg/game/mapedittestplay.go`) builds `-assets -base -mods
  -mods-dir -mission` from the mod it plays; it reads no starter setting.

## Proof

- `cmd/starter/launchsettings_test.go`, `TestLaunchSettingsTableReachesTheStarter`: every row has
  one view whose controls fit its kind; each row's default reaches `defaultSettings`; each accepted
  value is written, read back and passed as the expected arguments; each value one past a bound is
  refused with the value and, for the shop multiplier, the bounds in the text.
  `TestLaunchSettingsBoundsInCaptions` holds the volume captions' written bound to the row and shows
  the shop label with the row's bounds in both languages. `TestDefaultSettingsFileBytes` holds the
  default settings file to its bytes at base.
- `cmd/againrom/launchsettings_test.go`, `TestLaunchSettingsTableReachesTheGame`: every defined
  flag is bound; each row's game default reaches `parse(nil)`; the arguments `Setting.Args` builds
  for each value parse back to that value; each value one past a bound is refused by `Setting.Args`
  and by `parse` (the mission number by `game.MissionMap` at start); mission with picker is refused
  by both.
- Base comparison: one dump test run on base `41408076` and on the candidate printed 442 settings
  file and argument cases over 13 keys and 34 raw texts, the settings view layout and value steps in
  both languages, and 47 game argument vectors with the usage line. The dumps are identical after
  replacing the temporary directory name.
- Loss control: the result file of the lane names the runs.

## Open debt

- The volume label and refusal captions keep the bound as text (`0-100`, `0 to 100`): existing
  starter tests compare them without format arguments. `TestLaunchSettingsBoundsInCaptions` fails if
  the row's bounds move without them.
- `launchFlags` keys the game's binding by flag name; the walk test fails on a flag without a
  binding or a binding without a flag.
