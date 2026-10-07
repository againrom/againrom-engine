# 1012 — contract

## What this story settles

`docs/DIVERGENCES.md` row `DIV-011` states that the original's script-message broadcast is per
node, so consecutive raises of one message number produce one panel each, and that this build's
announcer instead merges consecutive same-trigger raises into one notice. The row cites
`TRIG-MSG-023` for the ROM1 side.

This story checks that citation against what `TRIG-MSG-023` actually establishes, measures how
often the shipped campaign exercises the case the row describes, and either closes the row or
sharpens it with a measured population. It changes `pkg/game/announce.go` only if the measurement
shows the current merge produces a player-visible difference on shipped data; otherwise it leaves
the code untouched and corrects the ledger row.

## What will work after this story

`DIV-011` states precisely what ROM1 evidence answers the cadence question, cites the claim that
answers it, and carries a measured shipped population instead of an assertion. No unresolved
"disclosed only in a code comment" state remains — the row and the code comment agree.

## The observable result

Either:
- `docs/DIVERGENCES.md`'s `DIV-011` moves to `CLOSED` because the announcer now matches ROM1
  broadcast-and-discard behaviour exactly, or
- `DIV-011` stays `OPEN`, its `ROM1 behaviour` and `Reason` cells cite the claim that actually
  answers the panel-count question, and its `Reason` cell states the measured population (how many
  shipped, non-once, message-raising triggers exist on each preserved root) and why the code is
  left as it stands.

## Claims this story is built on

| Claim | Confidence | What it supplies |
|---|---|---|
| `TRIG-MSG-023` | High for the send-side arm, the packet fields and the broadcast decision / **Medium** for "no simulation state" — the instrument is three routines and cannot see what a recipient does with the packet | Instant opcode 2 stamps a shared static packet and broadcasts it once per script pass its owning trigger holds. This is a SEND-side fact only: it says nothing about how many panels the client shows for repeated broadcasts. |
| `DLG-LIFE-005` | High for the three inputs, the pager loop, the absence of a timer and the drop | Nothing ends the display but player input, nothing queues, and a second announcement arriving while any dialogue panel is open is discarded outright, not shown as a second panel and not deferred. |
| `DLG-PATH-002` | High for the shared transport and the three opcode arms | The client dispatcher turns opcode `0xb6` into `PostMessage(0x433, wParam = msg+0x0a)` — one post per broadcast received, which is what feeds the handler `DLG-LIFE-005` describes. |

## Aspects that apply

| Aspect | Applies |
|---|---|
| Data | No — no data format changes. |
| Runtime state | Yes — the announcer's rising-edge memory is the subject under review. |
| Simulation | No — `pkg/sim`'s per-pass re-firing of a repeating trigger's instants is unchanged and is not itself in question; only the presentation-layer sampling above it is. |
| Player input | No. |
| AI | No. |
| UI / HUD | Yes — how many notice panels a repeated same-trigger raise produces. |
| Triggers / scripts | Yes — this is the whole subject. |
| Inventory / equipment | No. |
| Persistence / save-load | No — no serialized field changes. |
| Campaign / session | No. |
| Shipped content | Yes — the measurement is a sweep of the 28 campaign maps on both preserved roots. |
| Interactions with existing mechanics | Yes — `DIV-008`'s message-255 collision and the drop-when-open rule both touch the same recipient-side mechanism and are read alongside this row. |

## Domains touched

**Campaign & Scripts** (the trigger latch, the raise list, the announcer). **Client** is read for
reconciliation (the notice panel's open/dismiss state that `DLG-LIFE-005` describes) but no Client
package is modified.

## Divergence allocation

`DIV-011` already exists; this story edits it in place. `DIV-134` is reserved for this story and is
spent only if the measurement below requires a *second* row (for example, an unrelated EN/RU
shipped-data difference this measurement happens to surface). If the work closes with `DIV-011`
alone, `DIV-134` returns unused.

## Out of scope

Any change to `DLG-TAGARM-027`'s speaker gate, the outcome-notice panel (`DIV-008`), or the shelf
tip widget (`DIV-018`). This story is scoped to the one cadence question `DIV-011` names.
