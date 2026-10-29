# One town square view adapter for both games

## Intent

The owner requires the screens and components both games have to go one
way, in the second game's shape, with the first game as profile data. The
town square view is the page after the room tip in the owner's order. Before
this story one `townScreen.TownSquareView` already served both games, but
the square's character pane and dolls were room data only: `paneSquareView`
read `RoomSpec.Pane` and `RoomSpec.Seats`, so a square could not take a pane
from its description, and the first game's square showed none because no
room data reached it, not because its data said so.

After this story the square's pane is description data, as a room's. The
one adapter fills the pane from `Description.Pane(room)` for the square and
for every page. The first game's square has no pane because its shipped
description names none. Every captured frame, state log and SAV of both
games is byte-equal to its base.

## Authority

- Owner rulings: one builder per kind (`AGENTS.md`); the first game is a
  reduced second game; screens both games have go one way.
- Census `cross-game-kinds-8b7ee0e5`, rank 14: one composer (`town.Scene`
  over `town.Description`) with two adapters, `townSquareScene` with its
  `TownSquareView` and `secondTownView` with its own; recommended shape the
  second game's adapter as the one adapter, folding into rank 3 (the town
  host); the first game's square has no pane or dolls.
- The base contradicts the census row. Story 1519 already folded the two
  adapters: on base `db698044` neither `townSquareScene` nor
  `secondTownView` exists, `townScreen.TownSquareView` (`townhost.go`) is
  the one producer of `ui.TownSquareView`, and `townRoomView`
  (`secondtown.go`) is the one scene adapter. The gap that remained is the
  pane and dolls being room-only data.
- No claim is cited or changed: frames, state logs and SAV bytes are equal
  to the base, so no behaviour moved. No DIV row is added.

## As built

Kinds and builders:

- Town square view adapter: `townScreen.TownSquareView` (`pkg/game/townhost.go`)
  with the scene adapter `townRoomView` (`pkg/game/secondtown.go`) and the
  pane and doll filler `townScreen.paneSquareView`
  (`pkg/game/secondheropane.go`). `paneSquareView` now takes the description
  and the shown room's name and reads the pane through `Description.Pane`;
  dolls stay with a page that has seats.
- Square composer: `town.Scene` over `town.Description`, reused.
- Town screen host: the host of story 1519, reused.
- Town descriptions: `towns/*.json`, extended with data only.
  `town.SquareSpec` takes an optional `pane` (`*PaneSpec`), the room pane's
  type. `Description.Pane(room)` answers the square's pane for the square's
  name, a room's own for a room, nil otherwise.
  `Validate` checks a square pane as a room's (non-empty rectangle, font empty
  or `install`), through one `validatePane`. The chapter check
  (`checkChapterDescription`, `pkg/game/towncatalog.go`) refuses a first-game
  square pane whose rectangle is not the character region or that carries the
  spellbook corner or a `book_offset`.
- Town SAVE producer: unchanged.
- Hero pane: `townScreen.heroPane`, reused.

No shipped description changes. `docs/MODS.md` names the square's `pane` and
the chapter rule. `CHANGELOG.md` has one line for map makers and modders.

Guard: `internal/archtest/townsquare.go`, `CheckTownSquareView`. Over
non-test `pkg/game` sources it allows a `ui.TownSquareView` composite literal
only in `townScreen.TownSquareView` and a `ui.TownSquareDoll` literal only in
`townScreen.paneSquareView`, lets no other function answer a
`ui.TownSquareView`, and lets neither producer nor any `townRoomView` method
read `SecondGame`, `GameOf`, `Edition`, `profile` or `hostRules`. Each type
must be built at least once. Test: `TestTownSquareHasOneViewAdapter` on the
tree; `TestTownSquareGuardFindsASecondAdapter` on synthetic sources (a
first-game view, a doll built elsewhere, a game branch in
`townRoomView.TipAt`, a `hostRules` branch in `townRoomView.Size`).

## Proof

Base `eb45e6dc` (`origin/main`, the merge's first parent; story 1536 and
release 0.139.0 on top of the brief's base `db698044`). Head `912c6ef9`.
Both built with `go build -trimpath -buildvcs=false` and the revision stamp
of `scripts/build-tools.sh` (`cmd/againrom` only), and run headlessly with
`-assets <root> -seed 1 -nomusic --headless`, scenario version 10, against a
private saves directory per run.

Runs: the first game EN and RU over the town kit saves `game9711`,
`game9717` and `game9719`; the second game EN and RU over the owner saves
`game0000`, `game0002`, `game0003`, `game9998` and a new game. Each run
frames the square on entry; at 3, 6 and 9 s of square time (150 hover steps
of 20 ms each), where the first game's horse, woman and dervish layers run;
two hotspots hovered for one second each with the welcome tip open (the state
log shows the hover box visible, targets `town/233` and `town/234`); the
square after the tip's Close button; the TAVERN entered; in the second game
the patron doll hovered (`room-item/0` visible), the pane hovered and the
pane's statistics (Tab); the return to the square (Escape); after a town SAVE
named "square proof"; and after its LOAD. The SAV rows are every file of the
run's saves directory after the run. The state log row is the machine log
with the run directory and the load list's file times written out. The town
screen has no simulation World, so no World hash exists here.

Every row is equal: 252 items over 16 runs, folded below to one row per
state and root. A row's SHA-256 is the digest of the lines
`<save> <item SHA-256>` of that root's runs, sorted.

| State | Root | Items | Base SHA-256 | Head SHA-256 | Equal |
|---|---|---|---|---|---|
| entry | en | 3 | `0af99dea8322f81d674213361759c1092951f32b871a43d9092b942eb4d30ac8` | `0af99dea8322f81d674213361759c1092951f32b871a43d9092b942eb4d30ac8` | yes |
| anim-3s | en | 3 | `6e290ea028ec1ca622a239119f42cdb9b1e32a3ac61e3012a0589184654ffc23` | `6e290ea028ec1ca622a239119f42cdb9b1e32a3ac61e3012a0589184654ffc23` | yes |
| anim-6s | en | 3 | `b1cca265ed886d60c8c104a18484c1dbccde9a4896f697dd30feaa3b66cde2d3` | `b1cca265ed886d60c8c104a18484c1dbccde9a4896f697dd30feaa3b66cde2d3` | yes |
| anim-9s | en | 3 | `8e929d629b51c72c6a71110b369efa694b11a8e5605550c28573542174056665` | `8e929d629b51c72c6a71110b369efa694b11a8e5605550c28573542174056665` | yes |
| hover-a | en | 3 | `1c0107b65b80806c11899118d3f73a2c76fade28afde934b3534d9cf28f0a1b2` | `1c0107b65b80806c11899118d3f73a2c76fade28afde934b3534d9cf28f0a1b2` | yes |
| hover-b | en | 3 | `bf7f44a5bed0344a8695d60be476434d6c0e6f9260b42638478371a434915bf1` | `bf7f44a5bed0344a8695d60be476434d6c0e6f9260b42638478371a434915bf1` | yes |
| tip-closed | en | 3 | `4e1ccf272c34f4aa774ac61a0eefe94bce2bd65a0d844dd52ada6824b91f4601` | `4e1ccf272c34f4aa774ac61a0eefe94bce2bd65a0d844dd52ada6824b91f4601` | yes |
| building | en | 3 | `5d4d4a7476605b1b3e392e40429277039a8b4596ace3879a4c6a0390c22954e7` | `5d4d4a7476605b1b3e392e40429277039a8b4596ace3879a4c6a0390c22954e7` | yes |
| return | en | 3 | `e0b68d142d931b64620203909748373df3b5b0c30ba252366882e9a79648b809` | `e0b68d142d931b64620203909748373df3b5b0c30ba252366882e9a79648b809` | yes |
| after-save | en | 3 | `458fc6c9445cdedc1bfdd45743a3de35c57a13bd7680a5f06184b8b5420d6a15` | `458fc6c9445cdedc1bfdd45743a3de35c57a13bd7680a5f06184b8b5420d6a15` | yes |
| after-load | en | 3 | `77ee3704751b2d4e9641dd1dc14e0f1e5030ff37677dac5f2659a17441fdfae5` | `77ee3704751b2d4e9641dd1dc14e0f1e5030ff37677dac5f2659a17441fdfae5` | yes |
| SAV | en | 6 | `e3b869599683ab4abfa7da91b61560a6b0257b7f6d89ca2fc7d7ea3eaa10e4fa` | `e3b869599683ab4abfa7da91b61560a6b0257b7f6d89ca2fc7d7ea3eaa10e4fa` | yes |
| state log | en | 3 | `2206de08952b2afa3f5f5963b5b28ba0a619df604965cf0fa7cc148646d9e579` | `2206de08952b2afa3f5f5963b5b28ba0a619df604965cf0fa7cc148646d9e579` | yes |
| entry | ru | 3 | `acbcedaa44420f494d9bef32120b97c58a7c931bb4a7da022c78b9bd491f6e1a` | `acbcedaa44420f494d9bef32120b97c58a7c931bb4a7da022c78b9bd491f6e1a` | yes |
| anim-3s | ru | 3 | `83450ff1ec0240ac1b697d6446483d17e1c94bdb32a0bb77d95571edb11bbc17` | `83450ff1ec0240ac1b697d6446483d17e1c94bdb32a0bb77d95571edb11bbc17` | yes |
| anim-6s | ru | 3 | `0554039be776ecd09694426f0128cdb4bc4a0e2f6ae55c2700afc06a75d2ec11` | `0554039be776ecd09694426f0128cdb4bc4a0e2f6ae55c2700afc06a75d2ec11` | yes |
| anim-9s | ru | 3 | `531b92fe91712b2998c5d1284a8d543607645c0f089512e06f28721f13488add` | `531b92fe91712b2998c5d1284a8d543607645c0f089512e06f28721f13488add` | yes |
| hover-a | ru | 3 | `21d38f8678961ad1aaec9f33205756013e6f78a7898fabee13bec24af1cfd19d` | `21d38f8678961ad1aaec9f33205756013e6f78a7898fabee13bec24af1cfd19d` | yes |
| hover-b | ru | 3 | `6c11f6d43733aa1deb8cf4021cd022292591f6e5475a19c0ce360295592a54bd` | `6c11f6d43733aa1deb8cf4021cd022292591f6e5475a19c0ce360295592a54bd` | yes |
| tip-closed | ru | 3 | `4e1ccf272c34f4aa774ac61a0eefe94bce2bd65a0d844dd52ada6824b91f4601` | `4e1ccf272c34f4aa774ac61a0eefe94bce2bd65a0d844dd52ada6824b91f4601` | yes |
| building | ru | 3 | `ca91003ea1e2750d79b8b4f1cc88ae15ac9c09164a9e09a9be158611275a5345` | `ca91003ea1e2750d79b8b4f1cc88ae15ac9c09164a9e09a9be158611275a5345` | yes |
| return | ru | 3 | `e9eda76a2943e32517d574a1bd40d51cb656c8476cc8fc92d6e929a4eca7dd3f` | `e9eda76a2943e32517d574a1bd40d51cb656c8476cc8fc92d6e929a4eca7dd3f` | yes |
| after-save | ru | 3 | `33319285afcca9744fc3d8bbcebf7ec4015bb05ebcc6dde5fc5093d878f88613` | `33319285afcca9744fc3d8bbcebf7ec4015bb05ebcc6dde5fc5093d878f88613` | yes |
| after-load | ru | 3 | `fbd3c19193bf3c830a0f35d5659b14cf9656d9a54b8e92db5daa948b32c0156d` | `fbd3c19193bf3c830a0f35d5659b14cf9656d9a54b8e92db5daa948b32c0156d` | yes |
| SAV | ru | 6 | `e3b869599683ab4abfa7da91b61560a6b0257b7f6d89ca2fc7d7ea3eaa10e4fa` | `e3b869599683ab4abfa7da91b61560a6b0257b7f6d89ca2fc7d7ea3eaa10e4fa` | yes |
| state log | ru | 3 | `9cba18e48c1dd10dfb5a09b396791083e7408fb91a429810c2931a98eb1d0cbd` | `9cba18e48c1dd10dfb5a09b396791083e7408fb91a429810c2931a98eb1d0cbd` | yes |
| entry | rom2-en | 5 | `b458ac2134217e0bdd86f7a5cb907e5e34e1eb3f2ee5a311c9f9df329566ac31` | `b458ac2134217e0bdd86f7a5cb907e5e34e1eb3f2ee5a311c9f9df329566ac31` | yes |
| anim-3s | rom2-en | 5 | `9c63b2d1b43c9f8033160e8f7b0b24b6b3533efdf40cfc7a6e1b66a59949859c` | `9c63b2d1b43c9f8033160e8f7b0b24b6b3533efdf40cfc7a6e1b66a59949859c` | yes |
| anim-6s | rom2-en | 5 | `c232331822f79271d2823b62165292c40123d7f13904a5cced66020f86d33da5` | `c232331822f79271d2823b62165292c40123d7f13904a5cced66020f86d33da5` | yes |
| anim-9s | rom2-en | 5 | `4b59007f42fc30377b2528f9c11bd6bf3607be66da0950db88d7d369f5fe4987` | `4b59007f42fc30377b2528f9c11bd6bf3607be66da0950db88d7d369f5fe4987` | yes |
| hover-a | rom2-en | 5 | `cdbfe1aab115560b1fecfda935e6c43ad25f4f6606f9fdcc92c9fcccd154f8da` | `cdbfe1aab115560b1fecfda935e6c43ad25f4f6606f9fdcc92c9fcccd154f8da` | yes |
| hover-b | rom2-en | 5 | `b53db8766f088368886b400aa6a239c2a0eb537e28a10ed00fbb5e1f280f86dc` | `b53db8766f088368886b400aa6a239c2a0eb537e28a10ed00fbb5e1f280f86dc` | yes |
| tip-closed | rom2-en | 5 | `ca98e8444cf1c1e3d5766a3fafda6e62eb88e7548065dc7c0927f420edaa4938` | `ca98e8444cf1c1e3d5766a3fafda6e62eb88e7548065dc7c0927f420edaa4938` | yes |
| building | rom2-en | 5 | `89337bbaab0c3e335eee3dff0d4abd45f04f40b86728ce9d09b78e4bcec74c01` | `89337bbaab0c3e335eee3dff0d4abd45f04f40b86728ce9d09b78e4bcec74c01` | yes |
| inn-doll-hover | rom2-en | 5 | `708cab5ca0f2b780c3541af0f698d6c342b3cd9a8f0dd1d3949bf48ac03b74fa` | `708cab5ca0f2b780c3541af0f698d6c342b3cd9a8f0dd1d3949bf48ac03b74fa` | yes |
| inn-pane-hover | rom2-en | 5 | `1ec6903b79c04b58d5f7a238ee882541cf6142a50d16e55260b6f6b3859bd5b7` | `1ec6903b79c04b58d5f7a238ee882541cf6142a50d16e55260b6f6b3859bd5b7` | yes |
| inn-pane-stats | rom2-en | 5 | `e30678180ddd8b25178dc0c4c9d81d48a8e085b6f25d82995858e6f7760a634f` | `e30678180ddd8b25178dc0c4c9d81d48a8e085b6f25d82995858e6f7760a634f` | yes |
| return | rom2-en | 5 | `6245dfc0326fdca191154f924d551002653492f321f6b7d4dcf940e29f1c8ebf` | `6245dfc0326fdca191154f924d551002653492f321f6b7d4dcf940e29f1c8ebf` | yes |
| after-save | rom2-en | 5 | `137c8b470bde830a2810a5be1499ebc87b3db6dca937f6ab04c23318bedf12d1` | `137c8b470bde830a2810a5be1499ebc87b3db6dca937f6ab04c23318bedf12d1` | yes |
| after-load | rom2-en | 5 | `03aef0f8edd42683fff03cf08826e7da6de71e149021c799dc9255c54385d7e8` | `03aef0f8edd42683fff03cf08826e7da6de71e149021c799dc9255c54385d7e8` | yes |
| SAV | rom2-en | 9 | `c46dcfd260f447c8517822481e4680596d1c6f4bf1012b7e31452752107b4de0` | `c46dcfd260f447c8517822481e4680596d1c6f4bf1012b7e31452752107b4de0` | yes |
| state log | rom2-en | 5 | `422d96f9e67bf5305ffbc0a2d27ec78d33e98abd7aa1a4844965755bf97340c1` | `422d96f9e67bf5305ffbc0a2d27ec78d33e98abd7aa1a4844965755bf97340c1` | yes |
| entry | rom2-ru | 5 | `c1cd91ca1428a1af6ebe16906f4ad44f7110165b10ecfce6fd798089c885047b` | `c1cd91ca1428a1af6ebe16906f4ad44f7110165b10ecfce6fd798089c885047b` | yes |
| anim-3s | rom2-ru | 5 | `7e07b06e8c7d9466515513d5a4494b34ae48dab010b6a3baedf68014c492523e` | `7e07b06e8c7d9466515513d5a4494b34ae48dab010b6a3baedf68014c492523e` | yes |
| anim-6s | rom2-ru | 5 | `669952d1fed9714bf52aafc5ed7dbf04d776fe3429c0fa400eb5c773a696f869` | `669952d1fed9714bf52aafc5ed7dbf04d776fe3429c0fa400eb5c773a696f869` | yes |
| anim-9s | rom2-ru | 5 | `bfd1f0b7b22c9c4ccb9b290a983eed36ae404ca9d8318efa957a33935994a564` | `bfd1f0b7b22c9c4ccb9b290a983eed36ae404ca9d8318efa957a33935994a564` | yes |
| hover-a | rom2-ru | 5 | `85866e6e69023a5d4580faf38dd51a9b3ff1dc2201c31fefa23328ebc93b1c88` | `85866e6e69023a5d4580faf38dd51a9b3ff1dc2201c31fefa23328ebc93b1c88` | yes |
| hover-b | rom2-ru | 5 | `01af79fedcaf865cdacdc00b620446dc2efa14d928b6fe752060daa4e213c7d8` | `01af79fedcaf865cdacdc00b620446dc2efa14d928b6fe752060daa4e213c7d8` | yes |
| tip-closed | rom2-ru | 5 | `ca98e8444cf1c1e3d5766a3fafda6e62eb88e7548065dc7c0927f420edaa4938` | `ca98e8444cf1c1e3d5766a3fafda6e62eb88e7548065dc7c0927f420edaa4938` | yes |
| building | rom2-ru | 5 | `d6193483b764d062903817bfd74bbdea1f1b3236e1ff2c84e5c3131af52e3c29` | `d6193483b764d062903817bfd74bbdea1f1b3236e1ff2c84e5c3131af52e3c29` | yes |
| inn-doll-hover | rom2-ru | 5 | `2c2b8b82b60be6ecfd3035c452ba8847813afe058ced71ac74cd80d9a19333bf` | `2c2b8b82b60be6ecfd3035c452ba8847813afe058ced71ac74cd80d9a19333bf` | yes |
| inn-pane-hover | rom2-ru | 5 | `7189aaf95034bf3da46e54c4d72e1665b55f07b583f1b806206039e10c735bd7` | `7189aaf95034bf3da46e54c4d72e1665b55f07b583f1b806206039e10c735bd7` | yes |
| inn-pane-stats | rom2-ru | 5 | `290524efbc5c8c5694800685420a5bd838c58fe089d62f02dcd29d22e9b36e23` | `290524efbc5c8c5694800685420a5bd838c58fe089d62f02dcd29d22e9b36e23` | yes |
| return | rom2-ru | 5 | `5910098cb8cdb81a3cabdc7f65d9a2afa9f5d5d35a32d36e670c26eef9c1ba8b` | `5910098cb8cdb81a3cabdc7f65d9a2afa9f5d5d35a32d36e670c26eef9c1ba8b` | yes |
| after-save | rom2-ru | 5 | `1c0ed6a6fe54b379a2ed86474d112a5d4c6af629adace75f92a8fc0a58192f41` | `1c0ed6a6fe54b379a2ed86474d112a5d4c6af629adace75f92a8fc0a58192f41` | yes |
| after-load | rom2-ru | 5 | `46cb44d741b98d326bc3cf47f14a767da18fadd7635cd12e520217ae247c544d` | `46cb44d741b98d326bc3cf47f14a767da18fadd7635cd12e520217ae247c544d` | yes |
| SAV | rom2-ru | 9 | `5af60f93d85bb3be3eacb121c1ccf852e474d3229f64cacd807f789ca8648e9a` | `5af60f93d85bb3be3eacb121c1ccf852e474d3229f64cacd807f789ca8648e9a` | yes |
| state log | rom2-ru | 5 | `6d539a674e2791308f340e89f682d3ff8b067ba98ad238dcf2da07e61d748928` | `6d539a674e2791308f340e89f682d3ff8b067ba98ad238dcf2da07e61d748928` | yes |

Focused tests (`pkg/game/townsquareview_test.go`):

- `TestShippedSquaresCarryNoPane`: the first game's square and the second
  game's three town squares carry no pane; the inn and shop panes still
  answer through `Description.Pane`.
- `TestFirstSquarePaneIsTheSquareData`: a first-game town mod whose square
  names a pane shows it (`TownSquareView().Pane`, the character region, a
  subject), the composed square frame differs from the one without it, and
  the chapter check refuses a square pane with the spellbook corner, a
  `book_offset`, a moved rectangle or an empty one.
- `TestSecondSquareViewFollowsTheRoomData`: the second game's inn shows its
  pane and the selected patron's doll; with the pane removed from the data
  the doll stays and the pane goes; with the seats removed the pane stays and
  the doll goes.

Loss control: a first-game-only adapter put back (`firstSquareView`
answering `ui.TownSquareView` in a new `pkg/game` file, and a
`hostRules().Pages == "retained"` branch in `TownSquareView` returning it)
fails `TestTownSquareHasOneViewAdapter` with three violations: the
`hostRules` read in the adapter, a second producer, and a
`ui.TownSquareView` built outside the adapter. The tree was restored from
the story's commit.

## Open debt

- The square scene's lifecycle still has two paths selected by the host's
  `pages` value: `TownSquareActive` and `AdvanceTownSquareAnimation` run the
  page lifecycle (`roomActive`, `advanceRoom`) for `pages` `entry` and the
  retained square scene (`squareView().SetActive`, `Advance`) for
  `retained`, and `viewScene` hands the retained square its long-lived
  scene. Folding them moves the page lifecycle and the room tip's arrival,
  outside this story.
- Dolls come from a page's seats only. A square has no seats in its
  description; a square doll needs a seat source first.
- The pointer's cursor picture is not in a headless frame, and the state log
  does not carry it; the proof covers the hover box, not the cursor shape.
  The change does not touch the cursor code.
