# Provenance — the attack affordance

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 — a held modifier raises the attack mode, and it is a LATCH: raised on the key going down, lowered on it coming up | `AI-KEYMOD-059` — the three gate globals are written on WM_KEYDOWN/WM_SYSKEYDOWN of one virtual key each and cleared on the matching WM_KEYUP; the claim states no second model fits | High for the mechanism (the jump table and the message map are PE reads on both roots, every store and clear a cited instruction, `refto:` accounting for every writer) |
| FR-1 — the modifier that produces the attack pointer is the one at `[L00627]`, and it is vkey `0x11` | `AI-KEYMOD-059`, `AI-CURSOR-052` — `[L00627]` held is what turns the hover into the attack cursor | High for **which global**, and the claim itself labels the vkey-to-physical-key naming a Windows platform contract that "never raises this grade". So the mechanism is backed and the choice of Ctrl is a binding, carried in *Ours by choice* |
| FR-2 — the mode does not survive the window losing focus | `AI-KEYMOD-059` — `R0230`, reached from the `0x008` WM_KILLFOCUS message-map entry, clears all three globals together (`L00675`/`L00676`/`L00677`) | High |
| FR-3 — the pointer is what states the mode, and a press made under it is an attack | `AI-CLICK-050` — the map-click handler dispatches on the current cursor against each registered cursor's own handle, one arm per cursor, so the order a click becomes is a function of the pointer and not of what it hit | High (every arm cited; the cursor-to-builder map read out of the PE on both roots) |
| FR-3 — the pointer drawn is the game's own attack cursor, and its art is `graphics\cursors\attack\sprites.16a` | `AI-CURSOR-052` — `R0220` registers 28 cursors by art path and the tool pairs each `PUSH <path>` with its `MOV [global],EAX`; the attack cursor's global is `L00628` on both roots | High |
| FR-4 — a `.16a` literal word is palette index in bits 1-8 and a four-bit level in bits 9-12, and the level scales the source | `SPR16A-PIX-011` (promoted) — LUT mode 4 with 16 levels, source multiplier `level+1`, data flow closed across named addresses, altered masks and strides falsified against the corpus | High. Its own stated residual is the exact per-display-mode framebuffer packing, which a consumer compositing into RGBA does not reach |
| FR-4 — the leading 1024 bytes are a per-file `[B,G,R,x]` palette | `SPR16A-PAL-008`, `SPR16A-STRUCT-001` | High |
| FR-6 — the attack mode is refused for a selection the local participant does not own, at the authored key alone | `AI-PANEL-060`, `AI-PANEL-061` — the capability routine reads one view-side flags word, never the selection's classes, and bit `0x4` is set exactly when the primary object's `CPlayer` differs from the local participant's | High for each bit's set site and its guard. This is 0075 FR-1's own backing and is **unchanged**; it is listed because FR-6 states which of the two arming surfaces carries it |
| FR-6 — the modifier surface is NOT gated by ownership | `AI-CURSOR-052` (the hover-to-cursor routine reads no player value), `AI-CLICK-050` ("no diplomacy is consulted at click time at all") | High |

## Ours by choice

| What the spec fixes | Why it is free to fix |
|---|---|
| **Ctrl is the modifier.** Either Ctrl key, the two being one modifier | `AI-KEYMOD-059` labels the vkey-to-key-name mapping a platform contract rather than a reading of the image, and says every clause survives its withdrawal. What is reconstructed is the latch; which physical key raises it is one line in the one place bindings live |
| **The pointer is the attack pointer for the whole time the mode is up**, over a unit or over empty ground | The original swaps to a **second** cursor over empty ground — the swarm cursor — and issues a swarm order under it (`AI-CURSOR-052`, `AI-CLICK-050`). This tree has no swarm order, so the second pointer would name an order that cannot be issued. Carried as a disclosed divergence in the contract, not smoothed over |
| **Frame 0 of the ten, drawn still** | Nothing read gives the animation's cadence |
| **The hotspot is the frame's top-left corner** | Nothing read gives it. Frame 0's point sits about three pixels in from that corner, so the choice is within a few pixels of the art's own reading of itself, and the pressed point is the system cursor's either way |
| **The target marker**: an outline around the unit a press would name | Nothing decoded describes one — the original's affordance is the pointer alone. It is ours, and it exists because it is the one thing that answers *what* a press will hit rather than *that* the mode is up |
| **The authored fallback pointer** drawn when no art was supplied | An install this build cannot read the cursor out of must still show the mode. Nothing asserts anything about the original here |
| **A popup lowers the mode** | 0077 ruled that a popup takes every map-screen input. Whether the original's latch survives a modal panel is not established: `AI-KEYMOD-059` reads the clears on key-up and focus loss and names no panel arm |

## Open

- **The swarm cursor and opcode `0x1a`.** Decoded (`AI-CURSOR-052`, `AI-CLICK-050`, `AI-MINIMAP-062`)
  and deliberately not built. No meaning is assigned to it here.
- **The other two latches.** `AI-KEYMOD-059` reads three; this story wires the attack one. The Shift
  latch's decoded effect — a marquee that *adds* — is not built, and this tree's existing use of
  Shift to suppress the selection rectangle is left exactly as it is.
- **What the original does with a modifier held while a modal panel is up.** Not established, so the
  ruling above is authored rather than reproduced.
- **Whether the attack cursor's ten frames animate, and at what rate.** Not established.

## Removed

| Dropped | Why |
|---|---|
| A clause replacing the authored arming key with the modifier | 0075 chose that key while nothing decoded bore on the question, and its reasoning is recorded in the binding site. `AI-KEYMOD-059` removes the premise for the *modifier* surface without bearing on the panel surface the key stands in for, and `AI-PANEL-060` shows that surface is real and separately gated. Two surfaces, two bindings — so the key is kept rather than overturned, and the contract says so at FR-6 instead of leaving it emergent |
| A clause deriving the pointer's colours from the display mode | `SPR16A-PIX-011` names per-display-mode packing as its own residual. Compositing into RGBA does not reach it, so the spec asserts nothing about it |
