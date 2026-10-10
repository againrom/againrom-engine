# Engine words per locale as data

## Intent

The engine's own words come from one table per language, keyed by a stable
message id, and a screen takes the language from the install's profile. Before
this story ten `go:embed` Russian tables in `pkg/ui` and one in `pkg/game`
each had a loader beside English Go literals, and thirteen sites chose between
them by the menu font's selector. Base: `67d7187c`.

Owner direction: engine words per locale as data after the widget kit; one
builder per kind. No word changes.

## Authority

Owner direction only. The words are the engine's own; no ROM1 claim covers
them. The Drain Life alias keeps DIV-2177. No divergence row is added: no
player-visible word changes.

## One builder

Kind: engine-authored word table. Its one builder is `mod.Lookup`
(`pkg/mod/items.go`): a lookup over `text/<lang>/strings.toml` tables that
reads the language's own text, else the English one. A mod's folder reads
through it (`mod.TextLookup`, unchanged in behaviour), and so do the engine's
own tables (`pkg/words`). The table format and parser are the mods'
(`mod.ParseStrings`).

## As built

### Tables

`pkg/words/text/en/strings.toml` and `pkg/words/text/ru/strings.toml` hold 69
message ids each. English is the reference set. `words.For(language)` turns
the profile's language entry (`english`, `russian`) into the table language
(`en`, `ru`) and returns that language's book, loaded once; a language with no
table reads English. `words.Load(fsys, lang)` reads a book from any file system
in the same layout. `Book.Text(id)` returns the word, English for an id the
language lacks, and the id itself for an id no table holds.

### Language

`pkg/game/frontend.go` sets `InstallWords.Language` from the detected profile
(`archives.Base.Profile.Language`). `InstallWords.Words()` puts the book in
`ui.Words.Engine`, which reaches the flow through `SetWords`. `pkg/ui` reads
words through `flow.word(id)`; `flow.menuWord(id)` converts a word for the
menu font. ASCII is the same byte in both installed alphabets, so an ASCII
word is returned unconverted, as the English branch drew it before.

The font selector stays the font's: it converts bytes. On every shipped root
the profile language and the selector agree (`main.res` entry `id`: `english
0` on `en`, `rom2-en` and the demo; `russian 1` on `ru` and `rom2-ru`).

### Former sites

| former site | reads now |
|---|---|
| `ending.go` `endingWords`, `ending_ru.json` | `ending.*` |
| `gamemenuspeed.go` `pauseLabel`, `gameOptionsRows`, `gamemenuspeed_ru.json` | `speed.*` |
| `gamemenutooltip.go` `tooltipDelayRow`, `gamemenutooltip_ru.json` | `tooltip.delay`, `tooltip.unit` |
| `save.go` tooltip failure prefix | `tooltip.not_saved` |
| `gameoptions.go` `SetGameOptionControls`, `pathfinding_ru.txt`, `graphicsoptions_ru.txt` | `options.*` |
| `gameoptions_draft.go` speed and tooltip rows | `speed.slower`, `speed.faster`, `tooltip.delay_options`, `tooltip.unit` |
| `media_menu.go` `mediaMessages`, `media_ru.json` | `media.*` |
| `questobjectives.go` `questLines`, `questobjectives_ru.json` | `quest.empty` |
| `save_dialog_words.go` `saveWords`, `save_dialog_ru.json` | `save.*` |
| `soundoptions.go` `SetSoundOptionControls`, `DefaultSoundOptionWords`, `soundoptions_ru.json` | `sound.*` |
| `timedautosave.go` `timedAutosaveLabels`, `timedautosave_ru.txt` | `autosave.*` |
| `pkg/game/installtext.go` `InstallWords.Words`, `itemlabels_ru.txt`, the Drain Life rune literal | `item.*`, `tavern.sleep`, `item.drain_life_alias` |
| `words.go` `AuthoredWords` item, sleep and save dialog literals | the English table |

`itemlabels_ru.txt` line 2 (a word for "book") had no reader and is not
carried. The English `item.drain_life_alias` is empty: the English install
keeps its installed spell name. The install's own `stats.txt` words for casts,
damage and range are read when the book's language is not English, as before
for the Russian selector.

### Ratchet

`internal/archtest/enginewords.go` scans the production files of `pkg/ui` and
`pkg/game` for a `//go:embed` directive and for an `==` or `!=` between a font
selector and a selector value. On `67d7187c` it finds 30 sites: 12 embeds and
18 branches. Now it finds one allowed embed (`pkg/game/townsquare.go`, the
town description) and three branches that convert the alphabet, kept as
falling debt:

| file | branch |
|---|---|
| `pkg/game/secondgametext.go` | the second game's RU mission text code page |
| `pkg/ui/gamemenu.go` | the accelerator's CP866 lowercase fold |
| `pkg/ui/load_draw.go` | the EN font's ASCII fallback for a save name |

`pkg/ui` gains the `pkg/words` edge in the dependency table
(`docs/ARCHITECTURE.md`, `internal/archtest/dag.go`).

## Proof

- No word changes: `TestTablesEqualTheReplacedWords` (`pkg/words`). The
  replaced code on `67d7187c` was driven on both languages through its own
  functions (ending, speed rows, tooltip rows and failure prefix, option
  captions, Game Options page rows, autosave labels, media, quest, save
  dialog, sound options, item words) and its output decoded back to UTF-8:
  72 site rows per language, `pkg/words/testdata/replaced.tsv`. Three site rows
  read one id each (the page's slower, faster and unit words are the menu's),
  so 69 ids. Every table word equals the drawn word; changed words: 0.
- Missing ids: `TestShippedTablesLackNoID` lists each shipped table's missing
  ids; `en` and `ru` lack none.
- Third language: `TestAThirdLanguageTableReachesTheScreens` (`pkg/ui`)
  loads a synthetic `xx` table and shows its words in the game menu's speed
  rows and the save dialog; an id it lacks draws English.
  `TestAThirdLanguageIsATableAndFallsBackToEnglish` (`pkg/words`) covers the
  fallback and the missing-id report.
- Selector no longer chooses: `TestTheFontSelectorDoesNotChooseTheLanguage`.
- Every id read by a production source is in the English table and every
  English id is read: `TestEveryUsedIDIsInTheEnglishTable`.
- Mods: `pkg/mod` tests unchanged and green.
- Screens: the release results are in the lane return.

## Open debt

- English-only engine literals with no Russian word today stay Go literals and
  draw English on every install, as before: for example the Game Options words
  in `pkg/game/gameoptions.go`, the Sound Options fallbacks, "TIPS MODE
  UNAVAILABLE" and the setting failure messages. Each is one table row plus a
  Russian word.
- A mod's `strings.toml` cannot yet replace an engine word: the mod tables are
  not laid over the engine book.
- `pkg/modrt.LanguageFor` and the starter derive `ru` or `en` from the base ID
  suffix, beside `words.Language`.
- The three alphabet branches above.
- A Russian install whose menu font fails to load now selects Russian words,
  converted by the selector-0 encoder; before it drew English. No shipped root
  has this failure.
- An unrecognised `main.res` whose language is neither english nor russian
  leaves `Profile.Language` empty, so the engine words fall back to English.
  No shipped root has such a file.
