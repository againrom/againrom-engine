# 1014 — contract

## What this story settles

`docs/DIVERGENCES.md` row `DIV-006` states that a RU install draws no Cyrillic keyboard
accelerator for any in-game menu row, because the row's own label marking (`gameMenuAccelerator`,
`pkg/ui/gamemenu.go`) is already decoded and built, but its lowercase step is ASCII-only, and no
key this build read ever produced a CP866 Cyrillic byte to compare against it. The claim to build
against is `MENU-KEY-013` (High), which decodes the original's own accelerator-matching routine
whole: it lowercases ASCII always, and — gated on the install's own language flag
(`[L06203] == 1`) — lowercases CP866 Cyrillic by two fixed offsets (`+0x20` over `0x80..0x8f`,
`+0x50` over `0x90..0x9f`).

This story implements that fold, on the label-marking side and on the keyboard side, so a RU menu
row becomes reachable by its own key, matching the EN behaviour already built.

## What will work after this story

- The lowercase step folds CP866 Cyrillic, gated on the install's own selector, per `MENU-KEY-013`'s
  exact two ranges.
- The keyboard produces a byte a folded Cyrillic accelerator can match: a typed rune is encoded to
  the install's own CP866 byte through the existing seam (`pkg/formats/textinput.EncodeRune`), the
  same one `pkg/game/chargen.go` already uses for character-name entry.
- A RU in-game menu row (mission or town) is reachable by pressing its own marked letter, exactly as
  an EN row already is by its ASCII initial.
- The EN behaviour is unchanged: the eleven marked labels keep their accelerators, and the two
  unmarked ones (`Diplomacy`, `Abort Game`) keep the constructor's own immediate.

## The observable result

`docs/DIVERGENCES.md`'s `DIV-006` moves to `DIVERGENCES-CLOSED.md`: the CP866 fold `MENU-KEY-013`
decodes is implemented, mutation-tested, and witnessed against both preserved installs by a new
developer tool, `cmd/menuaccelcheck`, driving the production dispatch (the same `App.HeadlessType`
path a windowed keystroke goes through) against a real loaded save on `gameversions/ru`.
`pipeline/check-milestone.sh`'s script-gap census is not expected to move — this story touches no
script opcode — and is measured, not assumed, in `closure.md`.

## Claims this story is built on

| Claim | Confidence | What it supplies |
|---|---|---|
| `MENU-KEY-013` | High | The accelerator-matching routine (`R0700`/`R1173`) lowercases ASCII always and, when the install's own selector reads Russian, CP866 Cyrillic uppercase by two fixed offsets. Measured on both roots: EN marks eleven of thirteen labels, RU marks all thirteen, and the RU marks are placed away from the first letter specifically where the mission menu's own rows 5-7 would otherwise collide. |

## Aspects that apply

| Aspect | Applies |
|---|---|
| Data | No — no archive or table format changes; the install's own `dialogs.txt` bytes are read as they already are. |
| Runtime state | No — no new persisted or session field. |
| Simulation | No — `pkg/sim` is untouched. |
| Player input | Yes — a typed rune must reach the accelerator match; this is the story's second half. |
| AI | No. |
| UI / HUD | Yes — the whole subject: the in-game menu's own accelerator resolution. |
| Triggers / scripts | No. |
| Inventory / equipment | No. |
| Persistence / save-load | No — no serialized field changes; the integration witness loads and saves through the existing scratch-store path only, unchanged. |
| Campaign / session | No. |
| Shipped content | Yes — reconciled against both preserved installs' own `dialogs.txt` bytes (`cmd/menuaccelcheck labels`). |
| Interactions with existing mechanics | Yes — the fold is shared between the label side (already built, `0168`) and the new keyboard side; a fix to one without the other leaves the row still unreachable. |

## Domains touched

**Client** only (`pkg/ui`, the `pkg/game` wiring that hands `pkg/ui` its font selector and encoder).
No `pkg/sim`, `pkg/mapload`, `pkg/data` or script package is touched.

## Divergence allocation

`DIV-140` and `DIV-141` are reserved for this story. Both are expected to be spent: `DIV-140` for
physical keyboard-layout resolution being outside this build's own scope (an operating-system
concern, not a `pkg/ui` one), `DIV-141` for a same-byte accelerator collision the tool's own
diagnostic finds between two RU town rows (`MenuLoad`, `MenuAbort`) that `MENU-KEY-013` does not
address. Both are typed `UNKNOWN` — authored where research is silent — not `FIDELITY-DEBT`,
because neither is a case where correct ROM1 behaviour is known and built otherwise; ROM1's own
runtime accelerator matcher is undecoded for both questions.

## Concurrency

Story `1013-world-map-arrival` is open concurrently, touching `pkg/game/worldmap*.go` and
`pkg/ui/worldmap.go`. This story does not touch either file. It does touch `pkg/ui/app.go`, kept to
two small, self-contained edits: `App.SetWords`'s signature (a new `encode` parameter) and
`stepGameMenu`'s `Typed` dispatch (byte-range corrected to rune-range, a pre-existing bug this
story's own keyboard path exposes). Neither edit is near worldmap-related code in that file.

## Out of scope

- A typed save name, save overwrite/delete, or any other in-game-menu affordance beyond the
  accelerator itself (`DIV-099`, unaffected).
- Extending the JSON scenario vocabulary (`scenarios/README.md`) with a "type" command; the
  integration witness is a standalone `cmd/` tool instead, matching `cmd/shopdump`'s and
  `cmd/worldmapcheck`'s own precedent for a single-story instrument.
- ROM1's own physical-key-to-byte resolution (DOS scancode/codepage tables); this build reads
  runes the operating system's keyboard layout has already resolved (`DIV-140`).
