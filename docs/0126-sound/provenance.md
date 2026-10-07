# Provenance — 0126

Research pin: the `research/` submodule at this story's commit. Claims are cited by id; no
experiment folder is cited.

## Claims this story rests on

| Claim | Confidence | What it supplies |
|---|---|---|
| `ANIM-SND-022` | High | Two sounds per blow. The swing is `Sound[0]`, fired on the tick the attack-run counter equals the class's `AttackDelay`, and silent when that element is 0. The grunt plays `Sound[k+1]` with `k` = 0 when the health did not change, 2 when the new health is below half the maximum and 1 otherwise — the latter two only while the new health is above −10, and throttled per drawable to one in 1500 ms by `timeGetTime`. Both positional: pan and attenuation from map position, each clamped to 10000. Three classes ship a silent 0 on `Sound[0]`. |
| `REG-SFX-057` | High / Medium | `sfx.res::sfx.reg` is the slot table: `[Global] SfxCount = 564` is the highest slot id, not the entry count, over 115 sparse `Sfx<n>` entries. Each value is a path relative to `sfx.res` with **no extension**. Every nonzero element of a class's `Sound[]` array is an existing slot — 153 of 153. |
| `ANIM-RUN-004`, `ANIM-PHASE-003`, `ANIM-STATE-023` | — | Already spent, in `pkg/game`'s swing clock: the attack arm counts one per tick from zero and a fresh run begins at the swing start. This story reads that clock and adds nothing to it. |
| `ANIM-CLOCK-024` | — | The reason the swing tick is a **disclosed divergence**: the swing frame, the swing sound and the damage are scheduled from three different numbers, and binding any two reproduces the original on at most 2 of 157 shipped pairs. |
| `ANIM-BLOW-019` | High | The engine sends the victim's new health as a **level** and never a delta; the client subtracts. This is what makes the grunt free here and what makes the unchanged-health arm impossible here. |
| `RES-HDR-012` | — | Records the RU `SFX.RES` breaking a header law, which is why both roots were surveyed rather than one. |

The **index arithmetic** is worth stating separately because it is easy to get wrong by one:
`Sound[k+1]` over `k` in {0,1,2} reads elements **1, 2 and 3**. Element 1 is the unchanged-health
grunt, element 2 the ordinary wound, element 3 the wound that leaves the victim below half. Element
0 is the swing. Element 4 is read by no hook `ANIM-SND-022` names.

## What is ours by choice

- **The attenuation curve and the pan curve.** The decode names an `FSQRT` distance, a `log10` and a
  clamp at 10000. The clamp is taken; the constants either side of the log are not decoded, so the
  shape here is linear in cell distance and named as ours in `spec.md`. It is the one number a later
  decode would move.
- **The listener.** The original's listener is not decoded. Ours is the centre of the camera's
  visible tile range, which is what "where the player is looking" means in a tree with a zoom the
  original does not have.
- **The device sample rate**, 22050 Hz — the corpus's own rate for 288 of its 290 leaves. Anything
  else is resampled to it by nearest neighbour, which is ours.
- **The master volume and the mute**, their defaults and their flags. Nothing in the game names
  them; they exist because a mechanism with no seam is a mechanism nailed shut.
- **The falloff and pan radii**, named constants in one file.

## What is open

- `Sound[4]` — no claim in the pin names a reader for it. Named unused rather than guessed.
- The unchanged-health grunt (`Sound[1]`), refused for the reason `analysis.md` gives: no blow
  message crosses this tree's seam. Reopening it needs a seam change, not a sound change.
- The original's own channel limit, priority and stealing rule. `R0386` is identified as the
  channel manager in `ANIM-SND-022` but its policy is not decoded, so nothing here reproduces one.
- Everything in `sfx.res` that is not a unit's `Sound[]` array: ambience, music, interface clicks,
  spell sounds, death sounds. The loader can reach them; nothing in this story asks it to.

## What was removed

Nothing. This is the first code in the subsystem.
