# Front-end and new-character fidelity

The owner reported seven defects: stale initial spell ranges, an inert Cutscenes
menu, missing generator sparkles and archetype presets, an inert Credits menu,
hover help suppressed by tips, and male hurt sounds on a female mage.

This story connects the existing installed resources to those player paths.
Encountered movies belong to the local profile; campaign saves remain independent.
Original saved spell instances retain their streamed characteristics. Newly
constructed books use the actor's current derived statistics before first use.

Authority: owner screenshots and direction, installed resource tables, and the
pinned knowledge claims named by the affected code. Presentation choices whose
original timing or composition is not established are recorded as divergences.

The generated mission constructor now refreshes all known spell ranges after
actor derivation, before serializing the initial instance graph. Learning through
the town also refreshes against current derived stats. Loading a saved original
book retains its cached range, cost and defensive bytes.

The generator chooses the installed PC_Danath, PC_Fergard, PC_Naira and
PC_Reniesta spreads. Each spends the full 140-point budget; Reset restores the
selected preset. Its 15-frame Blind sprite is driven by presentation time only.
Tips occlude hover help inside their own rectangle, leaving exposed controls
eligible for the shared delay. Human voice selection resolves the live client
class and female hurt/death samples independently of the server persistence ID.

Cutscenes uses the installed cutpaths/cutscene positional pair and scrlbars/lm
art. Successfully presented movies unlock their catalog group in options.txt;
startup logos do not. Selection replays the full numbered group and returns to
the list on completion or skip. Credits from either main menu or campaign ending
scroll the complete installed LF-separated text, preserving blank lines and
replacing available logo markers with their installed pictures. Escape/click
returns to the caller; natural completion does too. Focus loss pauses the roll.

Focused unit checks pass. TestReleaseFrontendFidelity1184 passes against EN and
RU: four full-budget presets, live sparkle/tooltip pixels, all 28 initial ranges
(Teleport 5 in the controlled mage), female sample decoding, AGS cold resume,
encounter persistence/replay, and 207/166-line credits with the Nival logo.
The prior campaign-ending witness now selects the shared scrolling credits.

Remaining original-fidelity questions are bounded in DIV-1275..1278: profile
history compatibility, decoration placement/cadence, Reset's original preset
producer, and original human voice-selection control flow. These do not prevent
the requested player paths. The sole local adversarial pass returned one P2:
a decoder still waiting for its first frame could unlock the movie. The single
correction now requires a delivered frame. A blocked-stream regression covers
both abort-before-frame and first-frame unlock, including a cold library read.
Final landing evidence follows this correction. Publication is deferred by
owner direction.
