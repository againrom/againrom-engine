# Mission-start autosave

Every accepted fresh mission writes an ordinary SAV at world tick 0. The LOAD
chooser displays its exact SAV label `autosave start mission N`, where N is the
mission number, followed by the ordinary row formatter's ` - mission N` metadata.
The save captures the adopted mission and its ready-view camera through
the same current-state producer as ordinary SAVE.

## Authority

Owner direction. No new ROM1 research is required for this addition. Numbered
slots use the existing next-free `SaveStore.WriteOriginal` policy. Occupied
files remain intact. Game version is 0.48.0; starter remains 0.3.1.
The candidate pins published knowledge k149.

## As built

ConfigureSaveSeams installs the observer on App. The common accepted-viewer
layout seam covers NEW GAME, configured direct/headless missions, campaign
successors, town/world-map entry and restart. WorldMapTick arrival now applies
viewer layout before returning from its entry frame. App consumes that viewer
before the callback; resize and menu return cannot retry. Late configuration
consumes the existing viewer too.

The observer uses the adopted live mission's Snapshot, playerMissionSave and
SaveStore.WriteOriginal. The current successful SAVE directory remains the
destination. Candidate preparation, raw maps and resumed SAVs write nothing.
An unconfigured App gains no destination. A failed write reports on the map,
leaves the mission running and receives one attempt. FrontEnd gains no field.

## Proof

The primary RED on source base 7d7895a7 was
TestMissionStartAutosaveWritesAcceptedTickZero: accepted entry produced zero
SAV rows. The WorldMapTick loss control removes the arrival layout call; the
entry then has zero observer calls and its unapplied zoom 2 instead of 1.

TestReleaseMissionStartAutosaveRoutes covers NEW GAME, direct mission, direct
base, town/world-map entry and restart. TestReleaseMissionStartAutosaveSuccessor
finishes mission 10 through its objective and accepts mission 20. Both pass on
installed EN and RU data during development. Each writes automatic SAV, checks
its header label/mission/tick, renders the ordinary chooser in a child process,
cold-loads there and compares tick 0 and the same move after 16 ticks.

The source oracle is the live World bytes/hash and SaveApplication immediately
after synchronous SAVE, before the first map tick or command. The child has a
fresh FrontEnd. Full campaign projection is compared when CampaignState is
known. Fresh starts have CampaignState false; their operational Main, Selected,
Gold and Won fields are compared instead of claiming campaign-wire equality.
World bytes, mission and view state are compared on every route.

The town harness loads original save 666, buys through the shop, then enters
mission 30. The shop pointer helper uses its required 640x480 geometry; App is
resized to 1024x768 before GATES and mission acceptance. Cold LOAD uses
1024x768. The started purse must equal the changed current purse; output must
differ from the source SAV. Later ticks change World and late producer output
while the automatic file stays fixed. Restart allocates a second slot and
preserves the first. Unit controls cover preparation, LOAD, occupied slots,
visible failed write, resize, raw/refused entry, late/unconfigured wiring and
the destination after a successful dialog SAVE. The existing scheduled-light
witness selects its manual SAV by name beside the new automatic row; its cold
World and Sun assertions remain intact.

Focused tests, ordinary pkg/game and pkg/ui tests, gated manifest scan,
architecture, story guards and divergence guards pass. go build ./... passes.
The reconciled source includes idle-cache main e284df46. The source scanner
selects 775 gated tests. FrontEnd composition measures 82 fields and 17
coordinators; no ratchet rises. Ordinary pkg/sim and version-resource tests
also pass. The game resource matches 0.48.0 and is retained unchanged.
The new family and the existing scheduled-light release witness pass on EN
(7.240 seconds) and RU (7.097 seconds): three top-level tests and four route
subtests per root, with no skips. The owner kit preserves six automatic SAVs
per root with exact labels, file hashes, live-state proofs and chooser frames.

The corrected seven-family EN/RU selection passes in 9.103 and 8.965 seconds,
with no skips. It includes the four release oracles that failed on merge
88d523dd, whose 3298 Go files are byte-identical to reviewed e23ad146. Name
and quick-spell witnesses now cold-load their manual SAV by its exact name
beside the validated tick-zero autosave. The quick-spell cold witness uses
one local catalog instead of cataloging that directory twice. Restored hero
names, spell bindings, cast cursor and next cast remain asserted. Fame compares
all physical pre-ending save names and bytes after credits, hall, terminal F2
and menu return. Its completed-state SAV guards remain asserted.

The correction changes release oracles and a falling comment baseline.
The full landing chain remains required on the correction merge. The asset
guard and divergence claims instrument pass. Evidence stays outside Git under
review/story1313-mission-autosave; correction-01 holds the source identity,
focused receipts and fresh owner outputs. The earlier source proofs remain in
owner-kit/en-reconciled-01 and owner-kit/ru-reconciled-01.

## Open debt

Original executable listing and loading of the emitted ordinary SAV remain
unproved until an owner runtime witness. Reconciled published candidate
e23ad146 passed its sole adversarial review without a returned finding.
Landing acceptance requires the full Go, installed EN/RU and original-save/M2
chain on the exact merge before publication and build promotion.
