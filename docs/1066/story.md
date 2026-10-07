# 1066 — original mission-session continuity

## Player result

Loading a lawful original mid-mission SAV restores its 1000 fire-once trigger
latches and directional 50x50 diplomacy matrix before announcements and the
first gameplay tick. Already-earned one-shot rewards and messages therefore do
not repeat, and relations changed by a script, join or combat do not reset to
the fresh ALM defaults.

## As built

- `sav.File.SessionState` copies the two complete populations through the
  existing witnessed `TriggerLatch` and `Diplomacy` accessors.
- `sim.World.ImportOriginalSession` validates both lengths and every latch byte,
  builds both replacements, then commits them together. A refusal leaves the
  canonical byte state unchanged; caller buffers are never retained.
- Both direct resume and the Load Game opener validate the handoff before
  mutation. The opener imports it after `StartMissionFrom` compiles the map and
  before `openMission` constructs `NewAnnouncer`.
- The load disclosure and counted report now name latches and diplomacy as
  restored while retaining the explicit boundary for the world state that still
  restarts.

## Authority and boundary

`TRIG-FIRE-007` establishes the 1000-byte fire-once latch gate.
`SAV-SESS-031` establishes the world-half session layout and exact populations.
`AI-DIPLO-085` establishes verbatim mission diplomacy continuity.
`TRIG-SAVE-008` establishes session-read-before-trigger-builder ordering.

The trigger result registers are deliberately excluded: the builder can preset
a result slot after the session read and the complete post-builder differential
is Unknown. Counters, outcome, unknown session regions, actors, cells, sacks,
corpses, ground items, orders, casts and area effects are also excluded. The
native save form and research pin do not move. `DIV-026` is narrowed and remains
OPEN for these unapplied world axes.

## Proof

- Focused synthetic tests cover complete format handoff, exact latch/diplomacy
  import, directional relations, one-shot suppression, caller-copy ownership,
  and byte-for-byte transactional refusal.
- A gated lawful-save witness opens save 666 through the production Load Game
  path and compares all 1000 latches plus the playable 49x49 diplomacy cells at
  tick zero; it inspects the already-created announcer baseline and then proves
  that all seven saved message latches stay silent through 64 production ticks.
- The existing save-666 ownership witness no longer dismisses those correctly
  suppressed notices. It still proves the saved hero and independently joined
  companion remain the same two characters when mission 30 is constructed.
- Exact-candidate full Go, no-asset, divergence and paired EN/RU release results
  are recorded by the landing seat after reconciliation with the preceding
  story.
