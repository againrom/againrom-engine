# Analysis — a dialog window stops the world

What was not known before this story, and what was measured to find out. Three questions: does the
world actually keep running behind an open notice, does anything already darken the map, and — the
one that decided the design — what does the world's own pacing do when the front-end simply stops
asking it to advance.

## The baseline, measured rather than read

A throwaway probe drove `App.step` over the production map arm with a cadence seam recording every
call, then pushed a notice into the viewer and drove ten more frames:

```
no notice:    10 advances over 10 frames
notice open:  10 advances over 10 frames, 0 cadence calls
Space with a notice open: 1 cadence call, flow.stopped=true
```

So the world advances behind an open notice at exactly the rate it advances without one, nothing at
all crosses the cadence seam when a notice opens, and the three cadence keys are fully live under a
notice — Space toggles the player's own pause while a box is up. All three are the shipped state and
all three are what this story changes.

Nothing dims. The draw path composes terrain, the two art planes, the overlay passes, the path
strokes, the unit panel, the readout and then the notice; there is no darkening step anywhere in it.
The one dim the tree does own scales a cell's four corner light multipliers inside the engine margin
— per tile, terrain alone, margin only. It is not a backdrop and cannot be made into one.

## The measurement that chose the mechanism

The obvious way to suspend the advance is for the front-end to stop calling it. A second probe drove
the world's own pacing both ways over the same three-second span — once by not calling at all, once
by setting the stop and going on calling every frame:

```
ordinary frame:                            0 ticks
after a 3s gap with no calls:              4 ticks in ONE call
after a stopped span of the same length:   0 ticks in the resuming call
period=62000  maxCatchUp=4
```

Not calling leaves the pacing baseline where it was, so the whole span is still owed when the calls
resume and the first one pays as much of it as its catch-up bound allows: a quarter-second of world
time in a single frame, at the map-load period four ticks at once. Every walking unit jumps. The
stop path takes the baseline on every call and returns nothing, so the span is consumed as it passes
and the resume runs one ordinary frame.

That is not incidental. The pacing is written so that a stop cannot owe time — the elapsed span is
consumed by the same write the running path makes — and the catch-up bound exists precisely to keep
a stall from lurching. Declining to call turns a suspension into a stall, which is the one shape the
bound was built to absorb rather than the shape it was built to produce.

So the suspension is the stop, and the advance call stays where it is and stays unconditional. That
inverts what the shape of the change looked like before it was measured.

## What the stop already reaches, and what it does not

The stop the player sets with Space crosses one seam carrying two scalars and is applied on the far
side by the one function that re-rates the clock and sets the flag; it touches no world field, no
queue and no tick. So a suspension expressed this way needs nothing new on the far side at all — it
re-uses the flag the pause already uses, nothing reaches simulation state, no byte form moves and no
version is owed.

Two consequences fall out and are inherited rather than decided here. The water keeps animating while
the world is stopped, because the stop reaches the seam and not the viewer. And the readout keeps
being pushed while the world is stopped, because the push is deferred and fires on the stopped
return — so a suspended world reports itself as stopped, for free, which the not-calling design
would have frozen instead.

## The assumption that had to be hunted

The front-end writes to the cadence seam only when the cadence *changed*, and it decides that by
comparing against its own two fields. Those fields are the player's ladder position and the player's
switch — they are the request, not a record of what crossed. That is sound while the player is the
only thing that can move the cadence, and it stops being sound the moment a second cause exists: a
notice opening changes what the far side ought to be holding without changing either field, so the
existing comparison would see nothing to write. Whatever carries the suspension has to compare
against what the far side was actually told, and that value is stored nowhere today.

## Where the owner's ruling sits

The behaviour is the owner's, given as an author's ruling on what we ship rather than as testimony
about the original: there must be a pause and a dimming on any dialog window. A research lane is
asking in parallel what the original does with its clock and its backdrop while a panel is up. That
answer is not needed to build this and is not waited for. What it can later change is the dim's
strength, which this story fixes as a supplied value, and whether an open notice suspends at all,
which this story confines to one predicate.
