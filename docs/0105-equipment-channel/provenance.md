# Provenance — 0105, the equipment channel

Research pin: submodule `research` at `53f8bb7`. Every row below was read at that pin with
`go run ./tools/claim <ID>`, which prints the row with its retraction state; each was `active` when
read. No experiment folder is cited: what a row does not say, this story does not use.

## The chain the contract is built on

| Claim | Grade | What this story takes from it |
|---|---|---|
| `ITEM-APPEAR-023` | High | an equipped item's appearance word is assembled by one builder from four inputs, and its low five bits are the item's **definition row index** in its own collection. This is what makes "the row a weapon's name resolves to" the value the body list is indexed by, rather than a number this project chose. Its own Unknown — what the three bits above the index mean — is not consulted here |
| `HERO-APPEAR-042` | High for the gate and the arms | the body name is **produced**, not stored, and it is produced from equipment slot 1's five-bit field less one. `0085` already installed the seventeen-arm name-to-class chain from this row; this story installs the step in front of it. The row's **Medium** — that slot 1 is *the weapon* and slot 2 *the shield* — is the labelling, and FR-2 depends on the slot **number** only |
| `HERO-APPEAR-047` | High | the equipment slots are numbered 1..12 and the wire slot is the equipment slot minus one. This fixes the array's **width** and its **numbering**, which is what FR-1 models; the row's own statement that an empty slot is sent as zero rather than omitted is what makes "empty" a representable state rather than an absence |
| `HERO-APPEAR-052` | High for the load; Unknown past the end | the ordered list is the shipped file `text/heropicture.txt`, loaded by three named instructions, and it is indexed with `D - 1`. FR-3 reads the same shipped payload. **This story diverges from the row's published enumeration** at index 22 and above — see the divergence note below |
| `HERO-APPEAR-053` | High for the world half | nothing is composited over the world sprite: a held weapon is visible because the whole body sheet changes, and a character holds a visible weapon exactly when slot 1 is occupied and its field names a body other than `unarmed`. This is the row that authorises the story's largest cut, and it is also FR-4's own sentence |
| `SPR256-EQUIP-042` | High | the `graphics\equipment` tree is a portrait tree: 987 file nodes with identical path sets on both roots, every parsed sheet one 160x240 frame. The corpus half of the same cut, measured from the archive rather than from the image |
| `HERO-APPEAR-054` | High for the width and the entry; Medium for "nothing else edits it" | the slot array is twelve wide and an equipment change reaches the drawable only by a re-send. The Medium is why DD-2 states the derive-once divergence as a divergence rather than as an equivalence |

## Rows read and deliberately not used

| Claim | Why it is not in the contract |
|---|---|
| `HERO-FIGURE-058`, `HERO-FIGURE-059`, `HERO-FIGURE-060` | all three describe `R0745`, the figure compositor, which `HERO-APPEAR-053` and `SPR256-EQUIP-042` place in the **info window**. `-059`'s draw order and `-060`'s six-name two-handedness predicate decide which of two portrait layers is painted last; this tree draws no portrait. `-058` is cited above only as corroboration of the slot numbering, and `-060`'s own grading says the *words* weapon and shield are no more certain than `-042` left them |
| `HERO-FIGURE-061` | the eight voice banks. There is no audio tier in this tree — no package, no consumer — so a bank has nothing to feed |
| `ITEM-EQUIP-006` | the fourteen equipped pointer fields, the take-off arm of command `0x22`, and the equip and unequip virtuals. This story has no command channel and nothing that moves an item, so the row's slot map is used only as a second statement of the numbering `HERO-APPEAR-047` fixes. Its **Medium** on "no per-slot type restriction" is untouched because nothing here places an item in a slot other than 1 |
| `ITEM-CODE-029` | an authored item is a packed word whose top nibble-but-one picks the class. That is the map-placed item path; `0103` cut it and this story does not take it back. A party hero's weapon arrives as a name, not as a code |
| `HERO-START-039` | the ten start-weapon literals and the jump table. `0078` already installed them; this story changes nothing about which weapon a hero is handed, only what is derived from it. The row's **Medium** — that a shipped campaign takes the low arm — is carried unchanged and not resolved here |
| `UNIT-EQUIP-005` | a unit class's `EquipItem` cell and its `[tier ][material ]shape` grammar. `pkg/mapload` already resolves it for a placed human; extending the body derivation to the placed population is out of scope, so nothing new is taken from this row |

## The divergence from a cited row, stated plainly

`HERO-APPEAR-052` publishes `text/heropicture.txt` as **25 non-blank lines** and gives the
enumeration that follows from dropping the blank one. Read at this pin from both lawful roots with
this repo's own archive reader, the shipped file is **26 CRLF lines totalling 248 bytes** — the
byte count the row itself states — with index 22 blank. This story keeps the blank line as a list
entry, so its indices from 22 upward are one higher than the row's.

The reason is a corpus alignment taken in this story and recorded in `analysis.md`: the blank line
falls exactly on the `Weapons` row named `rem`, a removed row's placeholder, and the 22 rows below
it agree name for name. Which reading the original's own list reader takes is **not established by
any row at this pin** — the row's grading covers the load instruction and the payload, not the
reader's treatment of an empty line, and its standing Unknown is about an index past the end.

Nothing this story ships can tell the two apart: every weapon reachable through character
generation resolves to a row below the blank. The divergence is disclosed in the contract as DD-4
rather than resolved, and it is a question for research and not for a story.

## What no row at this pin says, and what the contract does about it

- **Which item occupies equipment slots 2 through 12 for a hero this tree builds.** Nothing decodes
  it and nothing here invents it: the contract models the slots and leaves them empty, and FR-2
  feeds `0085`'s existing law the same "no second slot, no armour" arguments it already receives.
- **What the original does with a slot-1 index past the end of the list.** `HERO-APPEAR-052` carries
  this as a standing Unknown. FR-3 answers it with the same total behaviour `0085` gave an unmatched
  name — a refusal reported rather than a value invented — and DD-5 names it as authored.
