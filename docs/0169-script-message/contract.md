# 0169 — contract

## What will work

A campaign map's script can raise a numbered announcement. The action is instant opcode 2. After
this story the simulation runs it, the number resolves to a text file inside the install, and the
text is shown in the dialogue panel this build already draws for mission events.

Before this story opcode 2 was the one instant the simulation could not run. It is 254 nodes over
the 28 campaign maps on each preserved root, and it was every remaining instant in the script-gap
census.

## The observable result

`bash scripts/campaign-sweep.sh`'s `unsupported` column falls from 313 to 59 over the 28 EN
campaign maps. The 254-node fall is the whole shipped instant-2 corpus. The 59 that remain are
**check** opcodes, which this story does not touch.

Driving mission 10 headless prints the announcement it raised and the text a player would have
read, on both roots.

## Aspects that apply

| Aspect | Applies |
|---|---|
| Data | Yes — event text files are read from `main.res` at run time. |
| Runtime state | Yes — the announcer's rising-edge memory and the open panel. |
| Simulation | Yes, as a no-op arm. The opcode runs and writes nothing. |
| Player input | Yes — the panel's dismissal and paging, both already built. |
| AI | No. |
| UI / HUD | Yes — the dialogue panel. |
| Triggers / scripts | Yes — this is the whole subject. |
| Inventory / equipment | No. |
| Persistence / save-load | Yes — the announcer must not re-raise on resume. |
| Campaign / session | Yes — the mission number selects the text directory. |
| Shipped content | Yes — 254 nodes, 242 pairs, 19 EN pairs naming no file. |
| Interactions with existing mechanics | Yes — the outcome notice, and the drop-when-open rule. |

## Domains touched

**Campaign & Scripts** (the opcode, the raise list, the mission number), **Client** (the panel, the
conditional tag arms, the audience), **Sim Core** (the supported-instant set and the empty arm).
**Persistence** is touched only in that resume must not re-raise; no serialized field is added.

## Claims this story is built on

| Claim | Confidence | What it supplies |
|---|---|---|
| `MISSION-TEXT-005` | High for the path / Medium for the number-to-file mapping | `main\text\battle\m<mission>\event<NN>.txt` inside `main.res`. |
| `DLG-PATH-002` | High for the transport and the three opcode arms | Opcode `0xb6` posts window message `0x433` with the number as `wParam`; the lose arm posts the same message with `0xff`. |
| `DLG-ABSENT-003` | High | A named event text that does not ship produces nothing: no window, no fallback, no fault. |
| `DLG-LIFE-005` | High for the drop | A second announcement arriving while any dialog is open is discarded, not queued. |
| `DLG-WIN-001` | High | The `0x84`-byte panel, its geometry, the portrait condition, the button and its paging command. |
| `DLG-WRAP-009` | High for the clamp formula | Lines clamp to `Height/(fontHeight+2)` with no scrollbar. |
| `DLG-ENTRY-016` | High for the shared path | All six entries reach the same show routine. |
| `DLG-MSGNUM-025` | High | The number is a dword. `L03537 CMP ESI,0xff` cannot separate a script raise of 255 from the mission-lost sentinel, so 255 produces the lose panel. |
| `DLG-MISSION-026` | High for the field identity | The `<mission>` of the path is `campaign+0x660`, never a packet field. |
| `DLG-TAGARM-027` | High for the eight arms and the two gates / Medium that a nesting-suppressed part is absent in play | The eight conditional tag arms, the rejection polarity, the substring nesting, the `male` arm's `female` guard, and the two gates on the speaker four. |
| `DLG-SOUND-028` | High for the arm and the caller | `sound=` is an out-parameter that plays nothing. |
| `DLG-NPCTAG-019` (amended) | High for the per-family counts | `Start` is the `npc.reg` key read at `L03431` that gates the four speaker arms. The retracted literal-run clause is not used. |
| `TRIG-MSGCORPUS-049` | High | 254 instant-2 nodes over 28 campaign maps on both roots, 242 distinct pairs, numbers 0..25, and the per-file resolution against `main.res`. |
| `TRIG-MSGLIMIT-050` | High for five limits / Medium for limit (6)'s completeness | The six G2 limits with class and cost. |

## What is ours by choice

1. **Instant 2 changes no simulation state.** The claims put the whole effect on the client. The
   simulation gets a named empty arm and the byte form does not move.
2. **The speaker's sex and class, when the speaker gate passes.** No claim states where the four
   speaker arms read a synthesised speaker's sex and class from. This story takes them from the
   `npc.reg` record's own `Flags` tokens, with `MySex`, `MyClass` and `Me` taking the player's
   hero. It is unreachable on shipped data.
3. **Message 255 shows the mission-lost notice.** `DLG-MSGNUM-025` establishes the collision. The
   lose panel this build has is the mission-outcome notice, so that is what a raise of 255 opens.

## Design decisions

- **DD-1** — The opcode is a named no-op arm in `runInstant`, not an omission from the switch. An
  omission reads the same at run time and says nothing.
- **DD-2** — The byte form does not move. `formatVersion` stays 50.
- **DD-3** — One part-selection function with the audience as its third parameter, rather than a
  second selector beside the existing one.
- **DD-4** — The eight arms are a table walked in order, not eight `if`s, because the order and
  the fall-through are the contract.
- **DD-5** — The reserved number is tested in the driver, before any address is composed.
- **DD-6** — The audience's hero half is settled once when the mission opens; the speaker half is
  resolved per tag. This is what makes paging safe.
- **DD-7** — The no-window runner accumulates announcements and reports them on the steps that
  already carry per-entity rows.
- **DD-8** — The reported text is decoded to UTF-8 for JSON; the drawn text is not decoded,
  because the font path applies the code page itself.
- **DD-9** — A `pN` reference is not how the runner knows the hero.

## Measurement taken for this story

`cmd/regtool dump <en>/scenario.res npc.reg` over the preserved EN install: no section carries a
key named `Start`. Four sections carry `Start` as a token inside their `Flags` value. Read against
`DLG-TAGARM-027`'s wording — a `Start` **key** at `L03431` — the gate does not pass for any
shipped section, so the four speaker arms are inert over shipped data and the four `iam*` arms are
what shipped data exercises. This is a statement about the key lookup only.

## Unknown

- Whether the speaker gate is a key lookup or a `Flags` token test. The measurement above makes
  the two disagree for exactly four `npc.reg` sections. `spec.md` SC-2 states which reading this
  build takes and what lifting it costs.
- What `npcalive=` and `npcdead=` do once matched. All four are unused by every shipped
  `text/battle` file, so nothing here depends on it.

## Out of scope

Audio (`sound=` composes no wave), the `npcalive=`/`npcdead=` conditionals, widening the path's
two-digit format, and every unimplemented **check** opcode. The cut list in `spec.md` carries each
with its cost to lift.
