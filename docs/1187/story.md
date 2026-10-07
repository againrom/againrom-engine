# Sound Options channels

## Intent

Music, effects and dialogue speech must have separate persistent volume controls.
Changing a control must affect sounds already playing. The owner approves this
settings slice and immediate publication after review and final gates.

## Scope

- Use the existing green/gold dialog frame and installed EN/RU volume captions.
- Add independent 0..100 music, effects and speech controls. Retain the existing
  master enable and volume, including command-line precedence and silent headless
  runs. Missing channel preferences default to full volume.
- Route dialogue and shop/teacher responses through a separate speech device.
  Effects include combat, interface sounds and ambience. Movie audio keeps its
  existing master-only gain until original track/channel routing is established.
- Update active voices without seeking or replaying them. Stop and finished-voice
  cleanup must release retained players.
- Persist before applying a change. A failed write must preserve live settings.
- Keep preferences outside mission/save state. Existing saves and process defaults
  retain their distinct lifetimes.

The independent original cells are established by VIDEO-MUSIC-010. Exact original
initializers, slider transactions and all producer-to-channel joins remain
separate questions. Track selection, Random Order and Acknowledgments are later
Sound Options surfaces; this slice does not claim the complete original panel.

## As built

The three sliders accept pointer press/drag/release and focused Left/Right
steps of five percentage points. Pointer drag previews without writing;
release commits. Escape cancels a pending drag and returns to the pause menu.
The original installed font and captions use the shared dialog frame. Control
geometry, slider art and immediate commit policy remain authored (DIV-1291).

MusicVolume, EffectsVolume and SpeechVolume are independent process preferences.
The applied percentage is master multiplied by channel, divided by 100. Dialogue
and merchant/teacher responses use SpeechPlayer; combat, interface and ambience
use the effects channel. This does not classify the original Acknowledgement
consumer. Cutscene audio retains master-only control (DIV-1255).

## Proof

Focused tests cover field defaults, unknown-key preservation, command-line
precedence, failed writes, all device routes, active voice gain/mute without
replay, concurrent stop/update, menu pointer/keyboard ownership and session reset.
TestReleaseSoundOptions1187InstalledControls passes with EN and RU resources:
installed captions fit, pointer controls persist and the mission stays paused.

The candidate executable ran twice per locale against private profiles. First
startup read full channels; the second process read 25/50/75. Both processes
read actual retained player gains for music, effects, speech, ambience and movie
audio before and after master mute. The existing town speech witness also
checks tavern, mercenary, merchant and teacher dialogue and response producers.
Candidate artifacts and both rendered panels are in the seat's untracked
review/story1187/. This proves device state, not a human listening assessment.

The landing requires one independent review, final Go/assets/release gates and
the same executable checks from clean engine main. The seat journal and review
report own their exact candidate/merge hashes and results. The EN/RU gate adds
the installed control test to the checked-in release population.
