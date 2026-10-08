# againrom

againrom is a new engine for **Rage of Mages** (Allods, 1998). You play the
original game's campaign with your own copy of the game, on Windows or macOS.

It ships no game files. It reads the maps, graphics, sounds, music and movies
from a Rage of Mages install that you already own. Both the English and the
Russian releases work.

The game is a work in progress. The campaign, towns, shops, the tavern,
battles, spells, movies and saving all run. Some details still differ from the
original. Each known difference is listed in `docs/divergences/`.

## What you need

- 64-bit Windows, or macOS on Intel or Apple Silicon.
- An installed copy of Rage of Mages, English or Russian. It is sold
  digitally, for example on GOG.

againrom only reads the install. It never changes the original game's files.

## Install on Windows

1. Download the Windows AMD64 ZIP from
   [GitHub Releases](https://github.com/againrom/againrom-engine/releases/latest).
   Unpack it into its own folder, for example
   `C:\Games\againrom`. Do not unpack it into the original game's folder.
2. Run `starter.exe`.
3. Under **Base game**, paste the folder of your Rage of Mages install into
   **Add path** and press **Add**. The starter checks the folder and shows
   whether it is the English or the Russian release.
4. Press **Play**.

The starter remembers your choices in `starter.ini` beside it. You can add
both the English and the Russian install and switch between them.
The package starts with default settings. It includes no saves, mods or personal
configuration files.

## Install on macOS

1. Download the macOS AMD64 ZIP for an Intel Mac, or the macOS ARM64 ZIP for
   Apple Silicon. Unpack it into a separate folder outside the game install.
2. These builds have no Developer ID signature or notarization. For a download
   you trust, open Terminal in the unpacked `againrom` folder and remove its
   quarantine attributes:

   ```sh
   xattr -d com.apple.quarantine againrom starter
   ```

   An attribute-not-found message means that executable is already unquarantined.
3. Run `./starter` from that Terminal. Under **Base game**, type the absolute
   path of the owned install, press **Add**, then **Play**. The Mac starter
   currently requires typed paths; clipboard paste and Command+V are pending.

The ZIP preserves executable permissions and contains no game assets or player
profiles. It is a pair of command-line-launched executables, without an `.app`
bundle. Alternatively, run `./againrom -assets "/path/to/Rage of Mages"`.

## Starter settings

| setting | what it does |
|---|---|
| Sound, volume | sound on or off, and its loudness |
| Movies | play the intro and the mission movies |
| Saves folder | where saves go; empty means the default below |
| Mods | which mods to load, in order |
| Load unmarked saves | with mods on, also load saves made without them |
| Close after Play | close the starter when the game starts |

## Saves

againrom saves in the original game's save format. LOAD also lists the saves
the original game made in its own install folder, and they load in againrom.

By default, saves go to the `saves` folder beside `againrom.exe`. If the game
runs from inside an install folder, they go to `Againrom\saves` there instead,
or to the user configuration directory's `Againrom` when that folder is
read-only (`%APPDATA%\Againrom` on Windows,
`~/Library/Application Support/Againrom` on macOS). The Mac executable is named
`againrom`; its install-local profile is `Againrom/saves` beside it.

## Mods

Mods live in the `mods` folder beside the starter. Each mod is one folder.
Tick a mod in the starter, set its options under **Mod settings**, and press
**Play**. **Rescan** picks up a mod folder you added while the starter was
open.

A save made with mods remembers which mods were on. The game refuses to load
it without the same mods, so the save stays consistent. A save made without
mods loads with mods on only when **Load unmarked saves** is ticked.

## Differences from the original

- Item bonuses can raise a skill above 100. Training still stops at 100, or
  at the Skill cap mod's limit. The save still records the trained level and
  the bonus separately, as the original does.
- The game window can be resized.

The full list of known differences is in `docs/divergences/`.

## Problems

When you report a problem, say which release you use (English or Russian),
what you did, and attach the save if the problem is in one.

Requests about pirated copies of the game are closed without reply.

## Building from source

You need Go, at the version named in `go.mod`.
macOS builds also require Xcode Command Line Tools and `CGO_ENABLED=1`.

```
go build ./...
go run ./cmd/againrom -assets "C:\path\to\Rage of Mages"
go test ./...
```

The tests need no game install. `docs/ARCHITECTURE.md` describes the code,
and `docs/PROVENANCE.md` describes how the original game's behaviour was
established without copying its code or data.
The version-tagged Windows packaging workflow is described in
[`docs/windows-release.md`](docs/windows-release.md).
That workflow also builds macOS Intel and Apple Silicon packages through
`.github/workflows/macos-release.yml` on native macOS runners.

## Legal

- This project is not affiliated with or endorsed by Nival Interactive, Buka,
  Monolith, 1C or any other rights holder.
- "Rage of Mages", "Allods" and the Russian title of the game are trademarks
  of their owners. They are used here only to describe compatibility.
- No original game data, executable or asset is distributed with this
  project. A lawfully owned install is required to run it. Naming a store is
  not an endorsement of it.
- The game's behaviour was established by reverse engineering a lawfully owned
  copy for interoperability, in a separate research process. Its public
  findings are the `knowledge/` snapshot.
- Copyright of the project's own code: the Againrom authors.

## License

GPL-3.0-or-later (`LICENSE`). Open-source libraries compiled into the game are
listed in `THIRD_PARTY_NOTICES.md`, with their license texts under
`LICENSES/`.
