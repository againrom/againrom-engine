# School diamond animation

The school displays the shipped diamond animation above the skill column.
This closes the missing paint step in DIV-147 without changing training prices,
skill gains, saves, or simulation state.

## Authority and scope

TOWN-154 at research pin `4b49d6524016a32cc0f3414c122a7c327f6fe613`
establishes nine `interface/training/diamond/on0000.bmp` through `on0008.bmp`
frames, each 80x76. The school paints the current frame opaque at (200,60)
only while the animator is active. Promoted TOWN-379 through TOWN-382 supply
the trigger, updater, completion and entry relation.

Locally admitted Train sets step +1 without resetting phase or current frame.
Refusal preserves idle or active state. Each production school room paint
advances once through 1,2,3,4,5,6,7,8,7,6,5,4,3,2,1,0. Completion caches
frame zero but hides it. A retrigger during descent reverses direction; one at
the peak repeats the peak once. Client update ticks, pure view composition,
snapshots and hit tests do not advance state.

The party picker and statistics toggle do not reset or rearm. School entry
from the square and new-game reset clear the animation. Leaving stops school
painting; reentry resets even though this build reuses its room object.
Dialogue overlays keep painting the school background. Art remains in the
front end's immutable startup cache. No school column rotation, training-rule
change, simulation field, or save-format change is included.

DIV-524 retains original display cadence, later reply effects and indirect
event Unknowns. DIV-525 records startup frame caching rather than original
entry/leave resource allocation. No fixed duration or original GUI runtime
equivalence is claimed.

## Proof

Independent fixtures check every frame address, dimensions, opacity, and the
literal 80x76 destination at (200,60). Literal phase tables cover the ordinary
orbit and retrigger from every reachable phase/direction. App tests separate
client updates from room paints. EN/RU release tests compare production
composition with independently decoded installed frames. `verification.md`
records commands, observables and remaining release work.
