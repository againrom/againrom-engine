# Working graphics options

The owner can switch Smoothing, Shadows, Dynamic Lighting and Object Animations
from Game Options. Each control must change its production renderer, survive a
restart and reach both fresh missions and loaded checkpoints. Rendering choices
must not change the simulation hash. Global autohealing is a separate gameplay
and save obligation and follows this graphics slice.

## Authority and scope

Knowledge k46 includes TOWN-OPTIONS-457, TOWN-GRAPHICS-459 and TOWN-SMOOTH-460.
The measured Smoothing consumer is the boundary sprite paired with the backpack
sprite on the map. Shadows gate the shadow pass; dynamic lighting is independent
of day/night; disabling object animation selects the initial scenery frame and
disables dynamic lighting. Water, actor action, town and UI animation are outside
the measured population and retain their existing clocks.

The current Game Options preference path applies immediately. This slice retains
that transaction policy and records its difference from original OK/Cancel.
Graphics options are process preferences, like pathfinding display; importing an
original SAV must not rewrite its simulation or original application carriers.

## Proof

- Synthetic rendered witnesses distinguish each switch and keep actor bodies,
  unrelated animation and day/night active.
- Installed EN/RU panels use original labels and controls. Smoothing decodes the
  installed base/boundary pair with the base palette.
- Profile restart, cold native LOAD, write failure and world-hash checks reach
  the production settings route.
- One fresh adversarial review, final Go/assets and paired release tests on the
  landing; rebuild and run the exact current executable.

## Open debt

DIV1298 covers broader local shadow/light populations and unresolved defaults.
DIV1299 records RGBA8 half alpha instead of native RGB565/555 masked halves and
incomplete Smoothing families. Existing DIV1289 retains immediate application
instead of original OK/Cancel. DIV1300/1301 remain unused and reserved until
landing; no ID is returned.

Focused UI/game tests pass. Installed EN/RU tests decode six base/boundary pairs
and 120 opaque boundary pixels, operate all four controls and Ctrl+L, keep the
held world unchanged and restore the same process preferences after native LOAD.
The EN/RU panels were rendered and inspected; long labels wrap within their
control. The sole review returned one defect: an intact building still animated
when another building was ruined. The correction honors the animation flag in
that branch. A Viewer regression holds ruin art, stops the intact building and
resumes its animation without stopping the shared clock. Final landing gates and
exact executable proof remain the seat's release obligations.
