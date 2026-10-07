# 0066 — analysis

**Intensity: spec-first / static. Terrain: brownfield** for `pkg/render/text`, `pkg/ui` and
`pkg/game`'s map driver — all three have shipped behaviour this story changes — and **greenfield**
for the event-text reader and the announcement derivation.

The profile's default for rendering and UI work is spec-first, and this story is that: a bounded
delivery whose behaviour settles once. The durable contracts it stands on — the compiled script,
the world's byte form, the archive addressing — are owned by other stories' specs and are not
re-opened here.

## What we did not know

Three things, and the third is the one that shaped the story.

**Where the words come from.** Nothing in this tree reads a mission's event text, and nothing in
`pkg/formats/alm` carries one — a map stores a *number*, not a sentence.

**What "the mission ended" looks like to a consumer.** `pkg/sim` reports an `Outcome`; what a
player should see, and what happens to the world while they see it, was not established here.

**How an announcement gets out of the simulation.** This is the one that mattered.

## The announcement problem, and why it did not become a `pkg/sim` change

The authored action that raises a mission's text is instant opcode **2**, and `pkg/sim` does not
run it: `scriptInstantSupported` covers opcodes 3, 4, 5 and 8 only, and 2 is reported as a
`ScriptGap`. It is not a marginal arm — it is the dominant one in the shipped corpus.

The obvious move is to give `pkg/sim` an arm for it and somewhere to put the result. Three
independent things refuse that:

- `nostate_test.go` pins `World`'s and `Entity`'s field sets as a literal table compared for
  **exact equality**. A field added to `World` fails it, and that pin exists precisely to catch
  state the byte form cannot reach.
- The byte form is at version 9 and the next version is spoken for elsewhere. Encoding an
  announcement would need one.
- An announcement carries no simulation consequence in the original either. It is a client window
  message; the server's own state does not branch on it.

So the question became whether the fact is **already observable** from what a world exposes. It
is. `scriptPass` writes the latch array in a shape that answers it exactly:

- an inert trigger is skipped **before** the latch is touched, so it leaves no trace;
- a one-shot that has already fired is skipped whole, so its latch stays as it was;
- every other trigger has its latch **cleared** at the top of the pass and set to 1 only if its
  conditions hold.

Sampled immediately after the pass, that gives *fired on this pass* with no residue: for a
repeating trigger the latch is 1 exactly on the passes it fires, and for a one-shot the 0-to-1
transition is the firing. `World.ScriptLatched`, `World.Script`, `Script.Triggers` and
`Script.Instants` are all already exported, so the derivation needs nothing new below it.

That is why this story changes no file under `pkg/sim`. The cost is a disclosure rather than a
compromise: `Script.Unsupported` still names opcode 2, because `pkg/sim` still does not execute it, and
that report stays a statement about the *simulation* rather than about the game.

## What the claims settled that we would have guessed wrong

Reading the ledgers rather than reasoning from the shape of the problem changed four decisions.

**A missing text file is not an error.** The natural design is a fallback string or a logged miss.
The engine does neither — the guard is read, the window builder is skipped, and the arm returns as
if it had worked. So "no file" and "nothing happened" are the same observable, and 19 of the 242
numbers the shipped campaign raises are silent by that route.

**A second announcement is dropped, not queued.** A queue is what one would build. The engine
tests a bit the panel sets and returns without building anything. A consumer that queues shows
text the original never showed — so the queue would have been a bug that looked like polish.

**"Did we win" is not `counter > 0`.** The counters only ever rise, nothing clears them inside a
mission, and one repeating check can drive the lose counter up without bound. The reporter tests
`== 1`. `pkg/sim` already implements this correctly; what the analysis established is that nothing
*above* it may re-derive the outcome from the counters — which is why this story reads `Outcome()`
and never `ScriptCounters()`.

**Every drawn byte passes a converter first.** The atlas subscript is not `byte - 0x20`; it is
`conv(byte) - 0x20`, where `conv` is language-conditional and moves two blocks. `pkg/render/text`
implements the un-converted rule, which is correct for English and draws the wrong glyph for every
Russian byte — measured at 0 of 1 895 localised high bytes landing on inked records under the
identity rule. The goal names both releases, so this is not a later polish item.

## What we looked at and did not take

- **The portrait.** The window has two layouts and the shipped corpus only ever exercises one of
  them. Reproducing it needs `npc.reg` and face art, neither of which this story owns.
- **`~` as markup.** Both byte loops treat a single `~` as a rule that occupies no width. No census
  of `~` in shipped strings exists, so we do not know whether any mission text contains one; the
  cost of being wrong is a mis-measured line, not a wrong glyph.
- **Scrolling.** The control can scroll, and this window is given no scrollbar and no arm that
  answers one — so the clamp *is* the behaviour rather than a simplification of it.
- **The town.** The win chain routes there. There is no town, and inventing one would be the worst
  kind of scope: a destination nothing else in the tree agrees with.
