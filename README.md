# AgainROM

AgainROM is a new engine for **Rage of Mages** (Allods, 1998). You play the
original game's campaign with your own copy of the game, on Windows, macOS or
Linux.

It ships no game files. It reads the maps, graphics, sounds, music and movies
from the files of a Rage of Mages copy that you already own. Both the English
and the Russian releases work.

The game is a work in progress. The campaign, towns, shops, the tavern,
battles, spells, movies and saving all run. Some details still differ from the
original. Each known difference is listed in `docs/divergences/`.

## What you need

- 64-bit Windows, macOS on Intel or Apple Silicon, or Linux x86_64. See
  [Install on Linux](#install-on-linux) for its system libraries.
- The game files of Rage of Mages, English or Russian, from a Windows install
  or the CD. The game is sold digitally, for example on GOG.

AgainROM only reads the game files. It never changes them.

## Game files

AgainROM reads these files from one game folder:

| file | what it gives |
|---|---|
| `main.res`, `graphics.res`, `scenario.res`, `world.res`, `movies.res` | the game itself; without any one of them AgainROM does not start |
| `sfx.res` | sounds |
| `speech.res` | voiced lines in towns |
| `patch.res` | the captions of three graphics options |
| `Allods/MUSIC.RES` | music |
| `Allods/VIDEO4.RES`, `Allods/VIDEO8.RES` | the intro and the mission movies |

A missing optional file only turns its part off: no sound, no voices, no
music or no movies. `rom.exe` and the other files of the folder are not
needed. Saves the original game made in that folder (`game*.sav`) appear in
LOAD.

### An install from the CD

The CD installer leaves the music and the movies on the disc, and the original
game reads them from there. AgainROM reads only the game folder. Copy
`MUSIC.RES`, `VIDEO4.RES` and `VIDEO8.RES` from the `Allods` folder on the disc
into a folder named `Allods` inside the game folder. A digital copy, for
example from GOG, already holds that folder.

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

Rage of Mages was never released for macOS. Copy the files listed under
[Game files](#game-files) from a Windows install or the CD into one folder on
the Mac, with `Allods` as a folder inside it.

1. Download the macOS AMD64 ZIP for an Intel Mac, or the macOS ARM64 ZIP for
   Apple Silicon. Unpack it into a separate folder, not into the game folder.
2. These builds have no Developer ID signature or notarization. For a download
   you trust, open Terminal in the unpacked `againrom` folder and remove its
   quarantine attributes:

   ```sh
   xattr -d com.apple.quarantine againrom starter mapedit
   ```

   An attribute-not-found message means that executable is already unquarantined.
3. Run `./starter` from that Terminal. Under **Base game**, type the absolute
   path of the game folder, press **Add**, then **Play**. The Mac starter
   currently requires typed paths; clipboard paste and Command+V are pending.

The ZIP preserves executable permissions and contains no game assets or player
profiles. It holds three command-line-launched executables, without an `.app`
bundle. Alternatively, run `./againrom -assets "/path/to/game folder"`.

## Install on Linux

Copy the files listed under [Game files](#game-files) from a Windows install
or the CD into one folder on Linux, with `Allods` as a folder inside it.
File names are found in any letter case, so nothing needs renaming.

1. Download the Linux x86_64 AppImage or tar.gz from
   [GitHub Releases](https://github.com/againrom/againrom-engine/releases/latest).
2. For the AppImage, make it executable and run it:

   ```sh
   chmod +x againrom-*-linux-x86_64.AppImage
   ./againrom-*-linux-x86_64.AppImage
   ```

   For the tar.gz, unpack it into a separate folder and run `./starter` in
   its `againrom` folder. The archive preserves executable permissions.
3. Under **Base game**, browse to the game folder, press **Add**, then **Play**.
   Install `zenity` or `kdialog` for the folder browser; zenity takes priority.
   Without either, type the absolute path in **Add path**. Clipboard paste
   remains pending on Linux.

The AppImage keeps `starter.ini`, `mods/`, game preferences and `saves/` in
`$XDG_CONFIG_HOME/Againrom`, or `~/.config/Againrom` when that variable is
unset. Its read-only mount holds no player files. Existing settings are kept;
explicit paths chosen in the starter still apply. With the tar.gz, settings,
mods and the player profile use the same layout beside the programs as on
Windows and macOS. Neither download contains game assets or player profiles.

Install X11 or XWayland for the display. At run time Ebitengine loads these
libraries from the system; install the distribution packages for them:

- OpenGL: `libGL`, `libOpenGL`, `libGLX` or `libEGL`.
- X11: `libX11`, `libXcursor`, `libXext`, `libXi`, `libXinerama`, `libXrandr`
  and `libXrender`.
- Sound: PulseAudio or PipeWire's PulseAudio service. Without a
  reachable server, sound falls back to `libasound.so.2` (ALSA).

## Map editor

The map editor, `mapedit.exe` (`./mapedit` on macOS and Linux), comes in the same
folder. Select a base in the starter and press **Map editor** (**Редактор карт** on a Russian install). See
`docs/MODS.md` in the source repository for its keys.

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

AgainROM saves in the original game's save format. LOAD also lists the saves
the original game made in its own install folder, and they load in AgainROM.

By default, saves go to the `saves` folder beside `againrom.exe`. If the game
runs from inside an install folder, they go to `Againrom\saves` there instead,
or to the user configuration directory's `Againrom` when that folder is
read-only (`%APPDATA%\Againrom` on Windows,
`~/Library/Application Support/Againrom` on macOS, `~/.config/Againrom` on
Linux unless `XDG_CONFIG_HOME` names another folder). The Mac and Linux
executable is named `againrom`; its install-local profile is `Againrom/saves`
beside it.

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
Linux builds use `CGO_ENABLED=0` and need no C compiler; see
[Install on Linux](#install-on-linux).

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
  project. A lawfully owned copy of the game is required to run it. Naming a store is
  not an endorsement of it.
- The game's behaviour was established by reverse engineering a lawfully owned
  copy for interoperability, in a separate research process. Its public
  findings are the `knowledge/` snapshot.
- Copyright of the project's own code: the AgainROM authors.

## License

GPL-3.0-or-later (`LICENSE`). Open-source libraries compiled into the game are
listed in `THIRD_PARTY_NOTICES.md`, with their license texts under
`LICENSES/`.
