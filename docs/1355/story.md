# First ROM2 success presentation

The first ordinary mission victory shows installed main row 140 and dialogs
rows 43/154. Victory acknowledges the existing campaign transition once,
selects cutpaths row 1, plays the present numbered teleport member from the
selected install's video.res, and returns to available mission 20. Continue
retains the mission; the later End Quest/Victory input reaches the same boundary.

## Authority and scope

Public pin 78ef66d4dd72631adcddd69368679c8e918003e5 supplies
R2-ENGINE-093..097 and R2-ENGINE-073..077/048..050. Report source joins and
movie resource, companion and native completion predicates retain their
published Medium/High confidence and Unknowns. Native outer result one includes
open failure and a terminating tick. It is never used as decoded-success evidence.

Only the selected ordinary mission 10 output 1 is implemented. No later movie
output, main-hero constructor, additional party roster, ROM2 SAV or native
playthrough follows from this slice. The first transition and portable party
remain the existing campaign implementation.

## Implementation and policies

CutsceneBank reads ROM2 video.res at the install root, with no loose fallback.
The existing portable Smacker decoder, sidecar parser, App movie overlay and
audio teardown own playback. No external decoder or library is shipped.
The named movie family comes from installed cutpaths, rather than a hard-coded
teleport family or a ROM1 mission-number family.
ROM2 startup remains disabled at App composition. Other archive members do
not establish their startup consumers. Pointer input during a movie uses the
same wall clock as media idle input; the simulation input clock remains frozen.

The report uses the shared notice layout and installed font. RU report words
use Windows Cyrillic to the existing DOS font alphabet; independent conversion
checks and rendered artifacts witness that engine policy. Native glyph/layout
and generic binding fidelity remain Unknown (DIV-2396).

The existing bounded member copy, first-track audio, 640x360 composition,
companion fades, wait/final-frame scheduling and input drain are disclosed
engine policies at unresolved native library, receiver and scheduling boundaries
(DIV-2397..2400). A native close message and a common App close flag are not
claimed to have identical process-lifetime effects.

## Verification

External receipts: `review/story1355-rom2-completion/` in the coordination seat.
Completed RED on base dbb34860607065a2cf6fe36fc1b7b14a440ba513 reaches a
real authored win/report but no movie on EN/RU. After normal reconciliation
to d66caec15873a0a42605d4b2f1338cb66f73f0e1, the same failure is witnessed
through initial town, TALK, mission selection, ordinary move and acknowledgement.
The only gameplay acceleration relocates the hero near the authored exit;
ordinary App selection/move and script execution produce the outcome.

The installed audiovisual witness compares all 167 native 640x360 RGBA frames
and 981844 PCM bytes per locale against FFmpeg and the installed Smacker library
through a separately stamped, library-only 386 helper. The helper opens an
external copied member with its existing 0x1000 flag policy. It never starts
the original game or exercises original client/UI receivers.

The App witness renders every delivered frame, compares selected frames against
independent FFmpeg RGBA under the disclosed fade/letterbox policy, checks the
final black frame and compares all PCM forwarded to its controlled streaming
device with FFmpeg. Report PNGs and movie frames/PCM stay external. This proves
decoded data delivery and composition; physical listening and hardware receiver
fidelity are not claimed.

Controls cover key, both mouse buttons, close, Enter, Escape, teardown,
Continue/End Quest/Victory, missing data, corrupt copied data, a decoder error
after real frame/PCM delivery, duplicate
acknowledgement, menu return and fresh New Game. The campaign advances once,
the movie stops its started audio once, and movie presentation advances no
simulation tick.

Installed EN/RU witnesses passed:
`pkg/game.TestReleaseSecondGameFirstSuccessPresentation`,
`pkg/game.TestReleaseSecondGameFirstMovieControls` and
`pkg/video/smacker.TestReleaseSecondGameFirstMovieAgainstOracles` on ROM2;
`pkg/game.TestReleaseCutsceneNativeAppCompletionAndSkip` on ROM1.
Fixture-only checks do not stand in for these installed witnesses.

Game 0.85.0 advances the reconciled 0.84.0 minor version. Starter 0.4.0 is unchanged.
Final ordinary Go, public compile, field/coordinator/command/comment guards,
registered release population and no-assets checks passed on the coherent source.
The known unrelated global citation failure is reported in the external receipts.
No source self-review, main landing or release promotion is performed here.
