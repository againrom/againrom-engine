# Game Options

The five existing formation, retreat, health, damage and day/night controls
become visible in Game Options. Pointer controls and existing shortcuts share
one preference writer. The panel uses installed EN/RU captions, checkbox and
radio sprites, and the common dialog frame. Tips, speed and tooltip delay remain
available.

TOWN-186 establishes the original preference keys. AI-KEY-125 establishes the
keyboard routes. Existing simulation commands remain the authority for
formation and retreat. Preferences initialize fresh missions only. Loading AGS
or original SAV retains the checkpoint's application state and pending orders.
A failed preference write leaves the requested live setting unchanged.

Touched surfaces: UI, local preferences, fresh mission initialization and
checkpoint boundaries. An additive AGS field retains pending option commands;
old saves default to no pending options. Simulation rules are unchanged.
Sound channel controls and the unresolved original rendering/autohealing
consumers remain separate work.

## Proof

Focused tests cover shared pointer/shortcut behavior, malformed and failed
preference writes, town changes with a retained mission and invalid saved
commands. The EN/RU installed witness covers fresh preferences, an AGS captured
before queued changes execute, 64 identical successor states, SAV loading with
opposite profile preferences, and original captions/sprites in the drawn panel.
The sole adversarial review returned two P2 findings: stale retreat metadata
on LOAD -> NEW GAME, and a clipped RU speed label. One correction resolves
fresh retreat before application creation, tests the actual NewGameOpener
with empty and opposite profiles, and fits both installed speed labels with
an inset. The executable scenario explicitly acknowledges its mission dialogue
before SAVE. The aggregate release gate then exposed native option persistence
restoring transient spell/camera state, and an obsolete speed-button witness.
The seat hotfix restricts that native carrier to settings and updates the
pointer witness. Imported application restoration retains its existing contract.
Final gate logs and exact build receipts are under seat review/story1186/.
Streaming RC1 remains at fc4f3af.

## Open debt

The layout and immediate application of changes are owner-directed. Original
dialog transaction semantics and exact geometry remain unmeasured. The local
preference file replaces the original registry. Physical input acceptance is
not claimed by CPU rendering and App input witnesses.

The EN M20 witness exposed an existing SAV export loss: selected Medium retreat
reloads with zero absolute-health thresholds. No original field producer is
inferred from this engine result. Ordinary SAVE now keeps active retreat
thresholds and unexecuted option commands in AGS; an executed Never setting can
again use SAV. Original-SAV threshold production remains DIV-1290 debt. Other
pending gesture commands retain their existing save policy.
