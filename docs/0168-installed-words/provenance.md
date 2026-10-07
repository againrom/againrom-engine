# 0168 — provenance

Research pin: `7747b9d`.

## Claims consumed

| Claim | Confidence | What this story takes from it |
|---|---|---|
| `TEXT-STRTAB-023` | High | The loader contract of `R0661`: a line ends at `CR`, that byte is overwritten with `NUL` in place, the cursor advances by 2, and the bytes are used exactly as shipped with no code-page pass at load. Also the sixteen-file order, `main.txt` first at global base 0 with 274 lines, `dialogs.txt` at base 402 with 166, and the two indexing styles — a global subscript into `[L04369]` against `R0668(table, i)`'s table-local one. |
| `MENU-STRTAB-008` | Medium | Global slot 77 is the dialogue panel's button (`L06181`), and slots 140 and 141 are the mission-outcome panels (`L06182`, `L06183`). The Medium is explicit about why: each named index is a cited instruction that agrees with the English text at that line, which is corroboration across two artefacts rather than a read of the drawing call. |
| `MENU-ITEM-011` | High for the entry list, its order and the label file binding | The mission menu's seven built rows, their screen order, and their label indices `0x22 0x23 0x4c 0x24 0x25 0x26 0x27 0x28` on the descriptor at `L06197`, which `L06198 PUSH L06200` / `L06199 MOV ECX,L06197` loads from `main\text\dialogs.txt`. |
| `MENU-ITEM-012` | High for the entry list and its order | The town menu's five rows and `Abort Game` at label index `0x4d`. |
| `MENU-KEY-013` | High | The accelerator is the letter after a single `~` in the label, `~~` is an escape that skips two, and the constructor's immediate is only the fallback. On the Russian root all thirteen labels carry a `~` and the marked bytes are CP866. |
| `TEXT-CHARGEN-027` | High for the static draw contract | Global slots 238, 239 and 260 are the generator's navigation rows and 171..180 its skill hover prose. Read here only to confirm the shared reader reproduces what the private one resolved. |
| `TEXT-CHARGEN-028` | High | Global slot 125 is the name prompt and 256 the entry field's tooltip. Same use. |
| `TEXT-CHARGEN-029` | High for the source and selector mechanism | `R0616` reads `main\id` and stores the digit at `[L06203]`; the value gates byte conversion and branches no cited consumer to another index. This is why the seam is one index set with two byte sets and not two index sets. |
| `TEXT-NAMETAB-026` | High for the census | In `main.txt`, 17 of 274 lines are byte-identical across roots and they are seven structural lines plus the ten hall-of-fame names. Every line this story reads is outside that set, so each is expected to differ by root. |
| `TEXT-DOM-010`, `TEXT-SEL0-012` | High | Selector 0 is the identity on all 256 values; selector 1 moves two blocks. The build already applies this at draw through `text.Font.Selector`, so an install word needs no conversion at load. |
| `UNIT-PANEL-011` | — | Cited for the negative: the original's information-panel layout cannot be recovered from the executable, which is why the panel's 32 captions are category (c) rather than an index this story failed to find. |

## What is ours by choice

- The **fallback direction**. Nothing decoded says a missing line should produce this build's English
  word rather than an error. `TEXT-STRTAB-023` states that no reader bounds-checks a subscript, so
  the original's behaviour at an absent index is undefined rather than authored. Falling back keeps
  a partial install playable, which is a project choice.
- **Which surfaces the seam covers.** The eleven words are those with a decoded index that this
  build already draws. That intersection is a scope decision, not a finding.
- The **English strings themselves**, unchanged from the stories that authored them.

## What is open

- **The 91 category-(c) words.** `MENU-STRTAB-008` states its own coverage as 5 of 47 owners of
  `[L04369]`; the remaining sites fold the index into a memory displacement and were not resolved.
  The information panel's captions and the shop screen's words are in that unresolved remainder.
  `main.txt` lines 15..46 and 60..82 read as those surfaces' words, and that resemblance is not
  evidence: this story does not bind a word to an index on it.
- **The 26 in-mission message indices.** `MENU-STRTAB-008` resolves that `R0509` reads
  85..89, 129, 142..149, 204..209 and 221..226, and matches them to message families by contiguity.
  No single index has a decoded meaning.
- **Slots 193, 194 and 195.** `MENU-STRTAB-008` states these three name-validation messages have no
  located consumer. This tree already read 193 and 194 for the generator's two name refusals before
  this story; that binding is inherited unchanged and is not strengthened here.
- **`0158` AU-2 is closed, not open.** It held the menu descriptor's base undecoded.
  `MENU-ITEM-011` binds the descriptor at `L06197` to `dialogs.txt` at two adjacent instructions
  and gives `R0668`'s table-local resolution, so the eight label indices are readable.
