# One town hero pane producer for both games

## Intent

The owner requires the screens and components both games have to go one
way, in the second game's shape, with the first game as profile data. Town
pages go in order: the town host, the tavern or inn page, the inn offers,
then the hero pane. Before this story the one painter
`ui.DrawTownCharacterRegion` had two producers: `townCharacterView`
(`pkg/game/townshell.go`, first game) and `heroPane`
(`pkg/game/secondheropane.go`, second game). A fix to the pane's contents or
placement reached one game only, and the first game's pane could not be
moved by data.

After this story one producer, `heroPane(p *town.PaneSpec)`, builds the pane
of both games from room data. Every captured frame and SAV of both games is
byte-equal to its base.

## Authority

- Owner rulings: one builder per kind (`AGENTS.md`); the first game is a
  reduced second game; screens both games have go one way, town pages in
  order, the hero pane after the inn offers.
- Census `cross-game-kinds-8b7ee0e5`, rank 11: one widget with the two
  producers above; recommended shape `heroPane(p *town.PaneSpec)` with the
  pane rectangle, the book flag and offset and the card by the generator's
  presentation as data, and a `pane` spec on the first game's tavern, shop
  and school.
- TOWN-353: the mission frame and the shop view carry one instance of the
  character pane widget. TOWN-354: each of the six corner rectangles has its
  own art, gated by the pane session. The first game's room `pane` entries
  cite both.
- The story's adversarial review returned two findings, both on first-game
  mod towns: a shop with no `pane` record lost the drawn book corner while
  the book still opened, and a pane's rectangle and book flag moved the
  drawing but not the input. The correction answers both (As built, Proof).

## As built

- Kind: town hero pane producer. Builder: `heroPane` in
  `pkg/game/heropane.go`. It reads the rectangle, the book flag and the
  chrome font from `town.PaneSpec`; the card from
  `ChargenPresentation.CardView`; the corner and pane art from the screen's
  `townArt`; the party, the shown member and the book state from the shop
  controller; the mode from `townStats`; the figure, slot map and worn item
  text from the shop host's `face` hook. There is no branch on game identity.
- First game: `TownSurface` (tavern and school) and the chapter shop's
  `dress` call `heroPane` with the room's `pane` entry (`chapterPane`). A
  room without one takes the same room's entry in the screen's shipped
  description (`townScreen.defaults`, `rom1`), the fallback `TownMusic`
  already uses for a room's music rule. The shop host's `face` is the shop's
  figure cache and `shopDollHover`. `townCharacterView` and the tavern and
  school's `NoBook` override are removed; the book corner follows the data.
- Pane input (`pkg/ui`): the shop's picker arrows, mode corner and book
  corner (`ShopLayout.controlAt`), the doll (`shopDollSlotAt`,
  `shopDollAreaAt`), the item drop (`shopPaneTakesItemAt`), the headless doll
  targets and the tavern and school hover (`TownHeroHoverLines`) read the
  rectangle of the view the painter draws (`TownCharacterView.paneRect`), not
  `TownCharacterRegion`, `shopCharRegion` or `shopFigureRect`. The town
  surface's corners already read the view (`CharacterPaneCornerAt`). The book
  control is the pane's: `bookControl` needs the layout's book and a pane
  with the book corner; the corner hit, the B key and the open strip read it,
  and the shop host's new `pane` hook gates the game's book toggle
  (`paneBook`). A pane without the book has no book control.
- Chapter town check (`checkChapterDescription`): a tavern, shop or school
  `pane` whose rectangle is not `TownCharacterRegion`, a tavern or school
  pane with the book, and a pane with a `book_offset` are refused, naming the
  room. The first game's pages tile the rest of the screen with their own
  controls, so a pane moved onto them would overlap controls the page tests
  first; the first game's shop page does not move its book.
- Second game: the room pane and the shop page call `heroPane` unchanged.
  The shop host's `face` is `heroFaceView` over the composed `heroFace`.
  `townSceneState.heroStats` is removed; both games keep the pane mode in
  `townStats`, which the same reset clears. The `paneArt` callback is
  removed; it returned the same presentation caches `townArt` returns.
- Room data (`pkg/town/description.go`, `pkg/town/validate.go`, data only):
  `PaneSpec.Font`, `"install"` or empty; any other value is refused. The
  first game's tavern, shop and school in `pkg/game/towns/rom1.json` carry
  `"pane": {"rect": [480, 238, 640, 480], "book": <shop only>, "font":
  "install"}`. The rectangle is `ui.TownCharacterRegion`, where the base
  drew the pane from a zero `PaneRect`.
- Painter: `ui.DrawTownCharacterRegion`, unchanged. Card presentation:
  `ChargenPresentation.CardView`, reused. With the first game's generator,
  `CardView` gives the card font `tipFont` gave (`tip-font` "text") and no
  card shadow.
- Guard: `internal/archtest` `TestTownHeroPaneHasOneProducer` fails when a
  non-test function in `pkg/game` other than `heroPane` names
  `ui.CharacterPaneShop` or reads `characterCorners`.
- Untouched kinds: the tavern or inn page, the inn offers, the talk box, the
  room tip, the world-map page, the shop page layout.

## Proof

### Tests

- `pkg/game/heropane_test.go`: the first game's rooms carry the pane
  (`TestFirstTownRoomsCarryTheHeroPane`) and the first game's rooms read it
  (`TestFirstTownPaneIsTheRoomData`); a rectangle or book flag changed in
  the data moves the view's rectangle, its book corner and the drawn frame
  in both games (`TestHeroPaneFollowsTheRoomData`); with no party member
  either game's pane shows nobody (`TestHeroPaneWithNoPartyMember`).
- `TestModTownShopPaneFollowsItsRecord` (`pkg/game/heropane_test.go`), through
  the mod layer (`minimalEastMods`, `layeredFront`): the minimal mod town of
  `docs/MODS.md` has no `pane`; its three rooms take the shipped panes, and
  its shop draws the book corner closed, answers the book control, and
  after it draws the corner open with the book page shown, as on main
  (synthetic corner art). A shop pane with `"book": false` draws no corner,
  and the book control leaves the book shut. A moved shop pane, a tavern or
  school pane with the book, and a `book_offset` are refused with the room
  named.
- `pkg/ui/shoppanehit_test.go`: with the pane moved to `(200,20)-(360,262)`,
  every pixel where the region's picker, mode or book corner answers answers
  the same control at the moved place and none at the region; the doll's
  marked pixel, the item drop, the headless doll region, the town hover and
  the town's next arrow follow
  (`TestShopPaneHitTestsFollowThePaneRectangle`). A pane without the book
  answers no book pixel, keeps the strip shut and the B key sends nothing;
  with it the key sends the book control
  (`TestShopPaneWithoutBookHasNoBookControl`).
- `pkg/town/seats_test.go`: a pane font other than empty or `"install"` is
  refused with the room named; both accepted values decode.
- Loss control: `townCharacterView` put back into `pkg/game/townshell.go`
  fails `TestTownHeroPaneHasOneProducer` on both names; the file restored
  from the candidate commit passes. `chapterPane` without the fallback fails
  `TestModTownShopPaneFollowsItsRecord`; `controlAt` at the region constant
  fails `TestShopPaneHitTestsFollowThePaneRectangle`; the book toggle and the
  B key without the pane's book fail `TestShopPaneWithoutBookHasNoBookControl`
  and `TestModTownShopPaneFollowsItsRecord`; each file restored from the
  commit.

### Frames, state and SAV

Instrument: the game built at engine main `05289791` (from `git archive`)
and at the candidate (`scripts/build-tools.sh`), run headlessly with
`--headless` scenarios (front-end stage v10) and `-seed 1`, each run on a
fresh profile with a private copy of one save. Inputs: the first game's EN
town kit saves `game9711` (two members, the second a mage), `game9714` and
`game9717` (full pack), read by the `en` and `ru` installs; the second
game's owner files `game0000` (a fighter), `game0002`, `game0003` and
`game9998` (a mage), and a new game, read by `rom2-en` and `rom2-ru`.
States: each pane room entered (tavern, shop, school; inn and shop), the
room's dialogues dismissed, the figure, the statistics card (Tab) and the
book (B; drawn in the shop only); in the shop the picker's next member and
the same three states; in the first game the tavern and school after the
shop's member step. After each run the state lines (`machine.jsonl`, the
run directory's path replaced by a token) are compared; a save after
loading, and in the first game after the shop's states and member step, is
compared byte for byte. The town holds no World; the state lines and the
SAV are its hashed surfaces.

Result: 72 runs per side, every run exit 0; 316 rows (198 frames, 46 SAV,
72 state lines), 316 equal. The table's head is the corrected candidate
`d9a46cb7`, which carries engine main `40c1ba08`; the first candidate
`2de05bf9` gave the same 316 equal rows. In the second game the figure, statistics and
book frames of one room differ from each other (`tools/pngdiff.py` over
`rom2-en` `game0002`), so the states were reached. A first-game frame named
`book` outside the shop equals the room's figure frame: the tavern and school
have no book corner, in the base and at the candidate.

| state | root | base SHA-256 | head SHA-256 | equal |
|---|---|---|---|---|
| 9711-next-school frame school-next-book | en | 1f96a12952e06b08b9a469cdc9ec74df8f57586572a724ceabe2cfd0766b5e00 | 1f96a12952e06b08b9a469cdc9ec74df8f57586572a724ceabe2cfd0766b5e00 | yes |
| 9711-next-school frame school-next-figure | en | 1f96a12952e06b08b9a469cdc9ec74df8f57586572a724ceabe2cfd0766b5e00 | 1f96a12952e06b08b9a469cdc9ec74df8f57586572a724ceabe2cfd0766b5e00 | yes |
| 9711-next-school frame school-next-stats | en | 0dab49ede278f4dbc2d650fc7572a2f0fcc46973903060b2a4c022b23f3ad1ad | 0dab49ede278f4dbc2d650fc7572a2f0fcc46973903060b2a4c022b23f3ad1ad | yes |
| 9711-next-school state lines | en | 9e0706cb8e6ea5cb35d493d94e831f548d9d0844a4899c63d64bf93c0b81f21b | 9e0706cb8e6ea5cb35d493d94e831f548d9d0844a4899c63d64bf93c0b81f21b | yes |
| 9711-next-tavern frame tavern-next-book | en | c100660e14fd1b66cdd95b3c68839d754f6bd6abc60b4c50a7e62aae6a885f9d | c100660e14fd1b66cdd95b3c68839d754f6bd6abc60b4c50a7e62aae6a885f9d | yes |
| 9711-next-tavern frame tavern-next-figure | en | c100660e14fd1b66cdd95b3c68839d754f6bd6abc60b4c50a7e62aae6a885f9d | c100660e14fd1b66cdd95b3c68839d754f6bd6abc60b4c50a7e62aae6a885f9d | yes |
| 9711-next-tavern frame tavern-next-stats | en | 7ce691b4edaa9b4042618a39121143866cceeb9155ab0509129feb38d311a6a0 | 7ce691b4edaa9b4042618a39121143866cceeb9155ab0509129feb38d311a6a0 | yes |
| 9711-next-tavern state lines | en | 7daf21e5279d033a7bf3cbb738e26ff25d5573526c41b8b6fe6f1fef27f79d41 | 7daf21e5279d033a7bf3cbb738e26ff25d5573526c41b8b6fe6f1fef27f79d41 | yes |
| 9711-save SAV pane proof.sav | en | 931ef4a5ab8c27a26a41d774fde69cc45740a14099265acf24d2f397cf7697b6 | 931ef4a5ab8c27a26a41d774fde69cc45740a14099265acf24d2f397cf7697b6 | yes |
| 9711-save state lines | en | fc0e1997db82970d708117c68333916765d0d859af7cb4d3cddbde23d615bbcc | fc0e1997db82970d708117c68333916765d0d859af7cb4d3cddbde23d615bbcc | yes |
| 9711-save-after-shop SAV pane proof.sav | en | fdb5ec03e183156aa66aa4630dbb9d060896fd614aef498db4111b9965f42b36 | fdb5ec03e183156aa66aa4630dbb9d060896fd614aef498db4111b9965f42b36 | yes |
| 9711-save-after-shop state lines | en | 4cc0c1f3de3141ab33c44d6724921f1c9871b9a404b32195020852c4a90f0155 | 4cc0c1f3de3141ab33c44d6724921f1c9871b9a404b32195020852c4a90f0155 | yes |
| 9711-school frame school-book | en | d9517d9fd715a3e93cfe8bfd2e85867996d63a4f8a22f6199075ec77589b8d03 | d9517d9fd715a3e93cfe8bfd2e85867996d63a4f8a22f6199075ec77589b8d03 | yes |
| 9711-school frame school-figure | en | d9517d9fd715a3e93cfe8bfd2e85867996d63a4f8a22f6199075ec77589b8d03 | d9517d9fd715a3e93cfe8bfd2e85867996d63a4f8a22f6199075ec77589b8d03 | yes |
| 9711-school frame school-stats | en | c958637f13af4c4c7af880d8b1e0267e093d77df59e8960e84e025548a301815 | c958637f13af4c4c7af880d8b1e0267e093d77df59e8960e84e025548a301815 | yes |
| 9711-school state lines | en | 2751c9685a770876a07235bc267cdbfbaec27b6851c54f31ec64b3503dff1dd2 | 2751c9685a770876a07235bc267cdbfbaec27b6851c54f31ec64b3503dff1dd2 | yes |
| 9711-shop frame shop-book | en | 0c71bda5ae963601819f8bb42eb96dc9dba0bf7629aa6edf26dbab6b8813c78e | 0c71bda5ae963601819f8bb42eb96dc9dba0bf7629aa6edf26dbab6b8813c78e | yes |
| 9711-shop frame shop-figure | en | 034001fa0c17681c2735b19d6ab1c3b9ecf832820543ffdd5e4e2a6d8a75edaa | 034001fa0c17681c2735b19d6ab1c3b9ecf832820543ffdd5e4e2a6d8a75edaa | yes |
| 9711-shop frame shop-next-book | en | 8facaefdfcd61a73b0bf65c4063a58a1e3c7c15485dfb0f3b815b50c465a5575 | 8facaefdfcd61a73b0bf65c4063a58a1e3c7c15485dfb0f3b815b50c465a5575 | yes |
| 9711-shop frame shop-next-figure | en | 9a1d5271b8f578f08c425235363a4febd12e7ec1932b24d466f1817867c682a1 | 9a1d5271b8f578f08c425235363a4febd12e7ec1932b24d466f1817867c682a1 | yes |
| 9711-shop frame shop-next-stats | en | ba9f909abac0e3fa2bd437bf66a6dde42fce01402687990ddb67e4999671b3db | ba9f909abac0e3fa2bd437bf66a6dde42fce01402687990ddb67e4999671b3db | yes |
| 9711-shop frame shop-stats | en | df77dad6f74097b3ff837e1f91fc21bec7bd20d6296fdc614457436c30b41913 | df77dad6f74097b3ff837e1f91fc21bec7bd20d6296fdc614457436c30b41913 | yes |
| 9711-shop state lines | en | 4a18b25f1707898c4222e26d4ab9738423e81110214a097f1a0398882765ea5f | 4a18b25f1707898c4222e26d4ab9738423e81110214a097f1a0398882765ea5f | yes |
| 9711-tavern frame tavern-book | en | f06937efc70ff8d6893877f8180ae98001b7966d4ee588fd79412b35fd0f33ef | f06937efc70ff8d6893877f8180ae98001b7966d4ee588fd79412b35fd0f33ef | yes |
| 9711-tavern frame tavern-figure | en | f06937efc70ff8d6893877f8180ae98001b7966d4ee588fd79412b35fd0f33ef | f06937efc70ff8d6893877f8180ae98001b7966d4ee588fd79412b35fd0f33ef | yes |
| 9711-tavern frame tavern-stats | en | a369e8ef49c138ece6162e4ad429fa6c922832fe3ddb9c8381754613c4a7a44c | a369e8ef49c138ece6162e4ad429fa6c922832fe3ddb9c8381754613c4a7a44c | yes |
| 9711-tavern state lines | en | c92c09b078f3fe478f14aad7d78f080a4419c592a9b08c309d03ab560fd72059 | c92c09b078f3fe478f14aad7d78f080a4419c592a9b08c309d03ab560fd72059 | yes |
| 9714-next-school frame school-next-book | en | 30282e71b56e08ac5a377ee1b402c0c56c34febde5d822fff8f5f8a35c11311b | 30282e71b56e08ac5a377ee1b402c0c56c34febde5d822fff8f5f8a35c11311b | yes |
| 9714-next-school frame school-next-figure | en | 55f752cdcc8437de89229adf3b4576839ce8c75a54ef02c4ba1fb1649aef30ba | 55f752cdcc8437de89229adf3b4576839ce8c75a54ef02c4ba1fb1649aef30ba | yes |
| 9714-next-school frame school-next-stats | en | 1634ca42d6ff782b7ec03c9f5eb6e18cfa75f5d5f9d8af8b9ea7c429e9712c84 | 1634ca42d6ff782b7ec03c9f5eb6e18cfa75f5d5f9d8af8b9ea7c429e9712c84 | yes |
| 9714-next-school state lines | en | f22be515cf7900c5ad78ec8e4f652742a8b099116f87b9468477ea20b1194302 | f22be515cf7900c5ad78ec8e4f652742a8b099116f87b9468477ea20b1194302 | yes |
| 9714-next-tavern frame tavern-next-book | en | 632370630b3a763ef1abd4699e447d5cc7b07e070af7d2590d66ea1fc346f099 | 632370630b3a763ef1abd4699e447d5cc7b07e070af7d2590d66ea1fc346f099 | yes |
| 9714-next-tavern frame tavern-next-figure | en | 632370630b3a763ef1abd4699e447d5cc7b07e070af7d2590d66ea1fc346f099 | 632370630b3a763ef1abd4699e447d5cc7b07e070af7d2590d66ea1fc346f099 | yes |
| 9714-next-tavern frame tavern-next-stats | en | f5d8b65abd86b51c91df8c4a324b008844f1e0269c8ee2e3928bdecf4d8695f0 | f5d8b65abd86b51c91df8c4a324b008844f1e0269c8ee2e3928bdecf4d8695f0 | yes |
| 9714-next-tavern state lines | en | 96db360e7cb8e2fce5e674ef282cb2cc1872e701c28df21ca8e5ac918968e23e | 96db360e7cb8e2fce5e674ef282cb2cc1872e701c28df21ca8e5ac918968e23e | yes |
| 9714-save SAV pane proof.sav | en | e47d90ff3b69ff0f9dea86f8db56a274b60210533abd6599dec370018a31502a | e47d90ff3b69ff0f9dea86f8db56a274b60210533abd6599dec370018a31502a | yes |
| 9714-save state lines | en | e05e92b44583cd60b7e19716618d69aac0db922b0a5294217a16b7f3473a936e | e05e92b44583cd60b7e19716618d69aac0db922b0a5294217a16b7f3473a936e | yes |
| 9714-save-after-shop SAV pane proof.sav | en | e47d90ff3b69ff0f9dea86f8db56a274b60210533abd6599dec370018a31502a | e47d90ff3b69ff0f9dea86f8db56a274b60210533abd6599dec370018a31502a | yes |
| 9714-save-after-shop state lines | en | 3768271fac7c991b282818096e02b8b75b74ad40f5798962364ff1d54dad614d | 3768271fac7c991b282818096e02b8b75b74ad40f5798962364ff1d54dad614d | yes |
| 9714-school frame school-book | en | a03599c141a2adae86a2674d7dad8614fb739336f5ff5224e33d9511519deba0 | a03599c141a2adae86a2674d7dad8614fb739336f5ff5224e33d9511519deba0 | yes |
| 9714-school frame school-figure | en | ea73c667ec8d60200919fc62d3ed123eec095b0308d2db5c5ee97d634953894c | ea73c667ec8d60200919fc62d3ed123eec095b0308d2db5c5ee97d634953894c | yes |
| 9714-school frame school-stats | en | 070a77443a965babb2a1f49c496a01edbd74902e96581073a2921bea6cd7ced8 | 070a77443a965babb2a1f49c496a01edbd74902e96581073a2921bea6cd7ced8 | yes |
| 9714-school state lines | en | 6bb06710522decde719be7f80aac8fae310081d02a9c407e5f3c6d9bcf66f087 | 6bb06710522decde719be7f80aac8fae310081d02a9c407e5f3c6d9bcf66f087 | yes |
| 9714-shop frame shop-book | en | 7d84dc34d5427f7387372f47f45855bd14d91693d2f7716ef4ec0f6eb45471b5 | 7d84dc34d5427f7387372f47f45855bd14d91693d2f7716ef4ec0f6eb45471b5 | yes |
| 9714-shop frame shop-figure | en | 76fb1afa119e31bd1455660da23c0dc77543b1aae429daa4e6c8e2979d3deba8 | 76fb1afa119e31bd1455660da23c0dc77543b1aae429daa4e6c8e2979d3deba8 | yes |
| 9714-shop frame shop-next-book | en | da6d3bf69c2905cd97845fbebd9cb250d14eb2df2e23b40458e16de5e2d75eb6 | da6d3bf69c2905cd97845fbebd9cb250d14eb2df2e23b40458e16de5e2d75eb6 | yes |
| 9714-shop frame shop-next-figure | en | a8e7b352b3efcbe4d5506f90676d60fb3996b52b18d7f3a83b6a3c86075c17eb | a8e7b352b3efcbe4d5506f90676d60fb3996b52b18d7f3a83b6a3c86075c17eb | yes |
| 9714-shop frame shop-next-stats | en | 351145278b5195b4697de11e6995bbe1c9c599f6bde119f46d7d99f110fba661 | 351145278b5195b4697de11e6995bbe1c9c599f6bde119f46d7d99f110fba661 | yes |
| 9714-shop frame shop-stats | en | 42f4f5d517a8af961bf326d17f8faf5fe9d053ea7e2789392a89a24db56e73f5 | 42f4f5d517a8af961bf326d17f8faf5fe9d053ea7e2789392a89a24db56e73f5 | yes |
| 9714-shop state lines | en | 61a3bd10a26a23861ef86af2d85ff47d821543016e206c623b4df49e5f462e5d | 61a3bd10a26a23861ef86af2d85ff47d821543016e206c623b4df49e5f462e5d | yes |
| 9714-tavern frame tavern-book | en | 4f776cf2ee66cc48d309badb764f5a7804bcf903855e32704772a07c059129e3 | 4f776cf2ee66cc48d309badb764f5a7804bcf903855e32704772a07c059129e3 | yes |
| 9714-tavern frame tavern-figure | en | 4f776cf2ee66cc48d309badb764f5a7804bcf903855e32704772a07c059129e3 | 4f776cf2ee66cc48d309badb764f5a7804bcf903855e32704772a07c059129e3 | yes |
| 9714-tavern frame tavern-stats | en | b26f3aa8b14883df128c9d9a9f4f26c07ebd7f75506328e533386a8f2ebd2413 | b26f3aa8b14883df128c9d9a9f4f26c07ebd7f75506328e533386a8f2ebd2413 | yes |
| 9714-tavern state lines | en | f96674bcd2d01ed44ef80dc5a95b883ca2214ba7020bf70bb30092c08b56aefd | f96674bcd2d01ed44ef80dc5a95b883ca2214ba7020bf70bb30092c08b56aefd | yes |
| 9717-next-school frame school-next-book | en | 90f49f3549ce06e3947220d81f1364594da7d425d0d745c069977345a13540a8 | 90f49f3549ce06e3947220d81f1364594da7d425d0d745c069977345a13540a8 | yes |
| 9717-next-school frame school-next-figure | en | 90f49f3549ce06e3947220d81f1364594da7d425d0d745c069977345a13540a8 | 90f49f3549ce06e3947220d81f1364594da7d425d0d745c069977345a13540a8 | yes |
| 9717-next-school frame school-next-stats | en | 28c12733c0e1f95e6d9319be5e462cc05928684901ff67e397914d5319528129 | 28c12733c0e1f95e6d9319be5e462cc05928684901ff67e397914d5319528129 | yes |
| 9717-next-school state lines | en | 4e2d546ad93752ff7d07afe0498390ca641535ee9af6aae30b751043d8280fe8 | 4e2d546ad93752ff7d07afe0498390ca641535ee9af6aae30b751043d8280fe8 | yes |
| 9717-next-tavern frame tavern-next-book | en | 02ad1a915c5bbbae0af022203e5f12216a9c0e37fdf651afeb13d13106f9fe7b | 02ad1a915c5bbbae0af022203e5f12216a9c0e37fdf651afeb13d13106f9fe7b | yes |
| 9717-next-tavern frame tavern-next-figure | en | 02ad1a915c5bbbae0af022203e5f12216a9c0e37fdf651afeb13d13106f9fe7b | 02ad1a915c5bbbae0af022203e5f12216a9c0e37fdf651afeb13d13106f9fe7b | yes |
| 9717-next-tavern frame tavern-next-stats | en | d88ebac2dc9a21cc0a026e112e5810c358ecbd68d623be6e184efa18162deee1 | d88ebac2dc9a21cc0a026e112e5810c358ecbd68d623be6e184efa18162deee1 | yes |
| 9717-next-tavern state lines | en | 68bba75290709a7a840247f91ddbc18d1485774e69d5ad263c4118a54781a395 | 68bba75290709a7a840247f91ddbc18d1485774e69d5ad263c4118a54781a395 | yes |
| 9717-save SAV pane proof.sav | en | 3f98e7506d58cee831d9149d9a7edf2e76d3bb08b85f7c663a12bdff4f2afe2b | 3f98e7506d58cee831d9149d9a7edf2e76d3bb08b85f7c663a12bdff4f2afe2b | yes |
| 9717-save state lines | en | 02b90bb8e454720e98fb8f8f61fe00765023db573a1708a56c31c768d89de79b | 02b90bb8e454720e98fb8f8f61fe00765023db573a1708a56c31c768d89de79b | yes |
| 9717-save-after-shop SAV pane proof.sav | en | 3f98e7506d58cee831d9149d9a7edf2e76d3bb08b85f7c663a12bdff4f2afe2b | 3f98e7506d58cee831d9149d9a7edf2e76d3bb08b85f7c663a12bdff4f2afe2b | yes |
| 9717-save-after-shop state lines | en | 074d8d451561094a9403cae634a4fd1186dec738ef109aab9a51b50c4d0371c7 | 074d8d451561094a9403cae634a4fd1186dec738ef109aab9a51b50c4d0371c7 | yes |
| 9717-school frame school-book | en | 09fed9b2bf1c5135ea678d972dcc19f71d99eb7347ae2a662a34b50d80baa5f1 | 09fed9b2bf1c5135ea678d972dcc19f71d99eb7347ae2a662a34b50d80baa5f1 | yes |
| 9717-school frame school-figure | en | 09fed9b2bf1c5135ea678d972dcc19f71d99eb7347ae2a662a34b50d80baa5f1 | 09fed9b2bf1c5135ea678d972dcc19f71d99eb7347ae2a662a34b50d80baa5f1 | yes |
| 9717-school frame school-stats | en | 6a95d4ef1ba8352a03b4a5e1a60b8fdc07cd500b926768f3c218cd6b3cf7e31b | 6a95d4ef1ba8352a03b4a5e1a60b8fdc07cd500b926768f3c218cd6b3cf7e31b | yes |
| 9717-school state lines | en | 38c0f7d77633f3539dc806252567c515bed483d66f402ea6c2cc9cc2aee1f5f5 | 38c0f7d77633f3539dc806252567c515bed483d66f402ea6c2cc9cc2aee1f5f5 | yes |
| 9717-shop frame shop-book | en | fc2024872aa79f98a05146ab415614b20f57eac59b99e1ed2d6f5e8c86f4044b | fc2024872aa79f98a05146ab415614b20f57eac59b99e1ed2d6f5e8c86f4044b | yes |
| 9717-shop frame shop-figure | en | 861b2b55f2247e145a01bc2e6eaa976b6dafe490099b7063ccb8d2bf781298b8 | 861b2b55f2247e145a01bc2e6eaa976b6dafe490099b7063ccb8d2bf781298b8 | yes |
| 9717-shop frame shop-next-book | en | 845bb5c810ee08c7d5fe7cc6afcdd3fb71aadcb91f92e85a4e72122b2dba3fed | 845bb5c810ee08c7d5fe7cc6afcdd3fb71aadcb91f92e85a4e72122b2dba3fed | yes |
| 9717-shop frame shop-next-figure | en | a6e33f6105259de2055578e7494aaacefd94571c8c40ba52effc4e011cb0c519 | a6e33f6105259de2055578e7494aaacefd94571c8c40ba52effc4e011cb0c519 | yes |
| 9717-shop frame shop-next-stats | en | 460201ee75fc7665b23bfc2dabb9aa2cf09c8d585c361a333f3894f9b8f92c4b | 460201ee75fc7665b23bfc2dabb9aa2cf09c8d585c361a333f3894f9b8f92c4b | yes |
| 9717-shop frame shop-stats | en | 6677180d28f9eff301bf79309279c12e7fd04ccd09ba16313f36f7de4fcdbb8e | 6677180d28f9eff301bf79309279c12e7fd04ccd09ba16313f36f7de4fcdbb8e | yes |
| 9717-shop state lines | en | 3030ef2cb8c2c685cc784caa21acc8608a26c8f1e09ea132047e3a9d773796cd | 3030ef2cb8c2c685cc784caa21acc8608a26c8f1e09ea132047e3a9d773796cd | yes |
| 9717-tavern frame tavern-book | en | 1717b0ac0901e00315faba80919826123cbfe385886be95d0f6a6404172b6dac | 1717b0ac0901e00315faba80919826123cbfe385886be95d0f6a6404172b6dac | yes |
| 9717-tavern frame tavern-figure | en | 1717b0ac0901e00315faba80919826123cbfe385886be95d0f6a6404172b6dac | 1717b0ac0901e00315faba80919826123cbfe385886be95d0f6a6404172b6dac | yes |
| 9717-tavern frame tavern-stats | en | fdbf94f0c6d9e57501ca4a620bf8f33f426092da0ad3c39905ff1d51e937953e | fdbf94f0c6d9e57501ca4a620bf8f33f426092da0ad3c39905ff1d51e937953e | yes |
| 9717-tavern state lines | en | e5696ebbac132cc86e54f097e8772b0b9c28a5bcede266f5a5a744a9dcbf2562 | e5696ebbac132cc86e54f097e8772b0b9c28a5bcede266f5a5a744a9dcbf2562 | yes |
| 0000-save SAV game0000.sav | rom2-en | 5f9821363e610b76c28d3851b3dcba55a03885fb51d1a9380add675c7a774def | 5f9821363e610b76c28d3851b3dcba55a03885fb51d1a9380add675c7a774def | yes |
| 0000-save SAV game0001.sav | rom2-en | 7259eaef7a9d3f87f04124c277bf89c2e1c8a8d949125eeec77cfd48585f595e | 7259eaef7a9d3f87f04124c277bf89c2e1c8a8d949125eeec77cfd48585f595e | yes |
| 0000-save state lines | rom2-en | 6006a739cac17fbf60655f58b4abcb21ca053b4c6a827d78b6f43ea1c2515855 | 6006a739cac17fbf60655f58b4abcb21ca053b4c6a827d78b6f43ea1c2515855 | yes |
| 0000-shop frame shop-book | rom2-en | 3cc40d79dc09ed02984da1fc714a70021d31bb20c099792e5b353616c775f900 | 3cc40d79dc09ed02984da1fc714a70021d31bb20c099792e5b353616c775f900 | yes |
| 0000-shop frame shop-figure | rom2-en | ca95dbf6517e405e1fe31d291e657ac6722af29d481d1ca08627c9ce19a78c72 | ca95dbf6517e405e1fe31d291e657ac6722af29d481d1ca08627c9ce19a78c72 | yes |
| 0000-shop frame shop-next-book | rom2-en | 2ecbddcd6a58c964595ffb09e107fcec7c111c23f48ff6ae307500f2e34ef07e | 2ecbddcd6a58c964595ffb09e107fcec7c111c23f48ff6ae307500f2e34ef07e | yes |
| 0000-shop frame shop-next-figure | rom2-en | 5ef641639f37a1cc4cae8c447832d7e900996966926e23aaa98e9734107b991d | 5ef641639f37a1cc4cae8c447832d7e900996966926e23aaa98e9734107b991d | yes |
| 0000-shop frame shop-next-stats | rom2-en | be54e0591243e6308007ba7fd83142f509e45da11d38cb171c502cac4da58db0 | be54e0591243e6308007ba7fd83142f509e45da11d38cb171c502cac4da58db0 | yes |
| 0000-shop frame shop-stats | rom2-en | 70fd07017aa82212f8ffacec3e62b123efb0b1412dde814fd7ec13dbc803d416 | 70fd07017aa82212f8ffacec3e62b123efb0b1412dde814fd7ec13dbc803d416 | yes |
| 0000-shop SAV game0000.sav | rom2-en | 5f9821363e610b76c28d3851b3dcba55a03885fb51d1a9380add675c7a774def | 5f9821363e610b76c28d3851b3dcba55a03885fb51d1a9380add675c7a774def | yes |
| 0000-shop state lines | rom2-en | 81d817ac9e28b101324eff259d6881ec82a79f5710f9f931407685103f0d8a62 | 81d817ac9e28b101324eff259d6881ec82a79f5710f9f931407685103f0d8a62 | yes |
| 0000-tavern frame tavern-book | rom2-en | c641f05b26344ed57ab77fb96a81ffa5ac19eba9d96af2bd9794d755b25b15c9 | c641f05b26344ed57ab77fb96a81ffa5ac19eba9d96af2bd9794d755b25b15c9 | yes |
| 0000-tavern frame tavern-figure | rom2-en | 7da02977920a1ddcf602855e0c5f7d4a2d1d2dd24ad55c93c1aa9f53554b023c | 7da02977920a1ddcf602855e0c5f7d4a2d1d2dd24ad55c93c1aa9f53554b023c | yes |
| 0000-tavern frame tavern-stats | rom2-en | 595e10c003c975efb6a17c611336d1b7e77a7e89f9f452943b0c604425db27a3 | 595e10c003c975efb6a17c611336d1b7e77a7e89f9f452943b0c604425db27a3 | yes |
| 0000-tavern SAV game0000.sav | rom2-en | 5f9821363e610b76c28d3851b3dcba55a03885fb51d1a9380add675c7a774def | 5f9821363e610b76c28d3851b3dcba55a03885fb51d1a9380add675c7a774def | yes |
| 0000-tavern state lines | rom2-en | 4e3a96943293eeb471547efb60cd52b0d16e37f9ee9afb646715d3a04e251907 | 4e3a96943293eeb471547efb60cd52b0d16e37f9ee9afb646715d3a04e251907 | yes |
| 0002-save SAV game0000.sav | rom2-en | 1aa1c3afa01d8c8cdb1ef5ab7db4d2fcdfd3e3cdfef22726a5c68cb308b3252c | 1aa1c3afa01d8c8cdb1ef5ab7db4d2fcdfd3e3cdfef22726a5c68cb308b3252c | yes |
| 0002-save SAV game0002.sav | rom2-en | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | yes |
| 0002-save state lines | rom2-en | a50a3250024ea88e6394281ba66314630768f924ca17da8876773e507b6a8454 | a50a3250024ea88e6394281ba66314630768f924ca17da8876773e507b6a8454 | yes |
| 0002-shop frame shop-book | rom2-en | e234ed4b963e02d796fc11a148622b42c34762194d09ae983a03dc19461d86ad | e234ed4b963e02d796fc11a148622b42c34762194d09ae983a03dc19461d86ad | yes |
| 0002-shop frame shop-figure | rom2-en | 10b72b2596009f24572b49f1954b5b595f95a68567654edc14df215ad043e798 | 10b72b2596009f24572b49f1954b5b595f95a68567654edc14df215ad043e798 | yes |
| 0002-shop frame shop-next-book | rom2-en | 76d341cab3985c416483f12f9912263e7c4bd153f628811bd80b5dc12d057d34 | 76d341cab3985c416483f12f9912263e7c4bd153f628811bd80b5dc12d057d34 | yes |
| 0002-shop frame shop-next-figure | rom2-en | a5ef78f03cce7e82f0b570c5967a802ccf8ce9b997b96ce82d0145eeb69d80d1 | a5ef78f03cce7e82f0b570c5967a802ccf8ce9b997b96ce82d0145eeb69d80d1 | yes |
| 0002-shop frame shop-next-stats | rom2-en | efe97f87a4fe9df166c961a388e77ea4f8227f87fa173685c4635b7dd372ef6a | efe97f87a4fe9df166c961a388e77ea4f8227f87fa173685c4635b7dd372ef6a | yes |
| 0002-shop frame shop-stats | rom2-en | 5a095c1aac53bb44cc50b213e688ccc12127762b35ece5191999ffb170bec80e | 5a095c1aac53bb44cc50b213e688ccc12127762b35ece5191999ffb170bec80e | yes |
| 0002-shop SAV game0002.sav | rom2-en | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | yes |
| 0002-shop state lines | rom2-en | 75a7a9135eb2c4843ee63af2ee2bc98fd3bbf597f46707a0af53044aabbf9b80 | 75a7a9135eb2c4843ee63af2ee2bc98fd3bbf597f46707a0af53044aabbf9b80 | yes |
| 0002-tavern frame tavern-book | rom2-en | d2030a8267550420077be2f902fa44fe748bd607c826708ed4745dd78d78d597 | d2030a8267550420077be2f902fa44fe748bd607c826708ed4745dd78d78d597 | yes |
| 0002-tavern frame tavern-figure | rom2-en | 4a4b8c3d01dc1123bfd51161d31300a8a98c777ad37e8c136fbfc44fbf9b5a4f | 4a4b8c3d01dc1123bfd51161d31300a8a98c777ad37e8c136fbfc44fbf9b5a4f | yes |
| 0002-tavern frame tavern-stats | rom2-en | 47afb7dac7abdddd9a9752fa4aeaebf6930c693c4c0f0052deb39f6b51e4c9b4 | 47afb7dac7abdddd9a9752fa4aeaebf6930c693c4c0f0052deb39f6b51e4c9b4 | yes |
| 0002-tavern SAV game0002.sav | rom2-en | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | yes |
| 0002-tavern state lines | rom2-en | 2bca252b4749dedc4481a62a16c748e3bfe7d1fa059925376effc3f2718ccf29 | 2bca252b4749dedc4481a62a16c748e3bfe7d1fa059925376effc3f2718ccf29 | yes |
| 0003-save SAV game0000.sav | rom2-en | bb327440e06a8b4c0f79c30fb0b55441c3ad8f467bfcb865b1d83eb42bc7f533 | bb327440e06a8b4c0f79c30fb0b55441c3ad8f467bfcb865b1d83eb42bc7f533 | yes |
| 0003-save SAV game0003.sav | rom2-en | 76a64e76a06bdb03cd6e8ffc11644831aa55959ff09594004a436d4fb61e02c9 | 76a64e76a06bdb03cd6e8ffc11644831aa55959ff09594004a436d4fb61e02c9 | yes |
| 0003-save state lines | rom2-en | f6f9e4df71a620e5eb9a1de3593a99298e1040183a04f69c16ba40b0e8b87b04 | f6f9e4df71a620e5eb9a1de3593a99298e1040183a04f69c16ba40b0e8b87b04 | yes |
| 0003-shop frame shop-book | rom2-en | f9cee03f6901d4e1b8eed484c3d5f788af30a6711d3675cd61cb1dcf252b132b | f9cee03f6901d4e1b8eed484c3d5f788af30a6711d3675cd61cb1dcf252b132b | yes |
| 0003-shop frame shop-figure | rom2-en | bc23faebf945448135f74e2cd4abdde8b1b23d732a835dbe28348df7cac61fbd | bc23faebf945448135f74e2cd4abdde8b1b23d732a835dbe28348df7cac61fbd | yes |
| 0003-shop frame shop-next-book | rom2-en | 8b2e6036962c2b22fea2b421684aef556f7084187f4234e0bc1f83bc3bcedb83 | 8b2e6036962c2b22fea2b421684aef556f7084187f4234e0bc1f83bc3bcedb83 | yes |
| 0003-shop frame shop-next-figure | rom2-en | 3c710bb8da5a0c0e8755ba2e6fef7974d3370d27d135b42840a1c8d5233b04e5 | 3c710bb8da5a0c0e8755ba2e6fef7974d3370d27d135b42840a1c8d5233b04e5 | yes |
| 0003-shop frame shop-next-stats | rom2-en | 44bd03e26c9be3642231613df65d5b4f67e1b40c95e28cccbc00e73153ebee6d | 44bd03e26c9be3642231613df65d5b4f67e1b40c95e28cccbc00e73153ebee6d | yes |
| 0003-shop frame shop-stats | rom2-en | 422e77d9d492bd6ad3e9b630e08904796f0df8fcf7fc9c23064e4d152861cdd2 | 422e77d9d492bd6ad3e9b630e08904796f0df8fcf7fc9c23064e4d152861cdd2 | yes |
| 0003-shop SAV game0003.sav | rom2-en | 76a64e76a06bdb03cd6e8ffc11644831aa55959ff09594004a436d4fb61e02c9 | 76a64e76a06bdb03cd6e8ffc11644831aa55959ff09594004a436d4fb61e02c9 | yes |
| 0003-shop state lines | rom2-en | a3f96862de2b8360f91d70c42d8adbba828c2095771e80d3420b6bfdd49ebe99 | a3f96862de2b8360f91d70c42d8adbba828c2095771e80d3420b6bfdd49ebe99 | yes |
| 0003-tavern frame tavern-book | rom2-en | ae21ca46ad86682a547869ca3eae2522e93c513d79402d8f865e8eaff61bf32b | ae21ca46ad86682a547869ca3eae2522e93c513d79402d8f865e8eaff61bf32b | yes |
| 0003-tavern frame tavern-figure | rom2-en | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | yes |
| 0003-tavern frame tavern-stats | rom2-en | 4856476192ccc149f71b4026dc98b7bce4997e5b0bb27b97b60796202113ddcf | 4856476192ccc149f71b4026dc98b7bce4997e5b0bb27b97b60796202113ddcf | yes |
| 0003-tavern SAV game0003.sav | rom2-en | 76a64e76a06bdb03cd6e8ffc11644831aa55959ff09594004a436d4fb61e02c9 | 76a64e76a06bdb03cd6e8ffc11644831aa55959ff09594004a436d4fb61e02c9 | yes |
| 0003-tavern state lines | rom2-en | 89c088dce2ce6f3321f4b5e514b16b4f9f9649348746cbc428b5e3776e7a1da5 | 89c088dce2ce6f3321f4b5e514b16b4f9f9649348746cbc428b5e3776e7a1da5 | yes |
| 9998-save SAV game0000.sav | rom2-en | 3cf60927cbf706fdbc4773330c8cfad96fef0cabb4a4a54ab9b5de940be574e7 | 3cf60927cbf706fdbc4773330c8cfad96fef0cabb4a4a54ab9b5de940be574e7 | yes |
| 9998-save SAV game9998.sav | rom2-en | ee46c9257283c739b7babb822f54911fb42eaa9f0ea8ebaf257591b9d4769e6e | ee46c9257283c739b7babb822f54911fb42eaa9f0ea8ebaf257591b9d4769e6e | yes |
| 9998-save state lines | rom2-en | 1edc7ef08e2789a9ea21165fdefa9d5171b1ebd35a26a530f07d72a90c02f0bb | 1edc7ef08e2789a9ea21165fdefa9d5171b1ebd35a26a530f07d72a90c02f0bb | yes |
| 9998-shop frame shop-book | rom2-en | e2af86ab97284ed40bbc2625dbf8e1f92858b86f672c41493da2ffbe45e38172 | e2af86ab97284ed40bbc2625dbf8e1f92858b86f672c41493da2ffbe45e38172 | yes |
| 9998-shop frame shop-figure | rom2-en | 21f9ec488746116b57fa08d6fb2b38930b65642995d24a32e2de4751baf5d7e3 | 21f9ec488746116b57fa08d6fb2b38930b65642995d24a32e2de4751baf5d7e3 | yes |
| 9998-shop frame shop-next-book | rom2-en | 97af5804fe29d043f8b24ced5e7cff4a4a9565487990bb2cc3546a2411efb9e6 | 97af5804fe29d043f8b24ced5e7cff4a4a9565487990bb2cc3546a2411efb9e6 | yes |
| 9998-shop frame shop-next-figure | rom2-en | 63f129721aa51f36c0736833028113e09be9126493850ab011dadfccfc29387c | 63f129721aa51f36c0736833028113e09be9126493850ab011dadfccfc29387c | yes |
| 9998-shop frame shop-next-stats | rom2-en | 3cd4025261cae4f7d4733361800aa9e112f84ca39fd68e16f4ff5f7af55c8a8f | 3cd4025261cae4f7d4733361800aa9e112f84ca39fd68e16f4ff5f7af55c8a8f | yes |
| 9998-shop frame shop-stats | rom2-en | cb964968cf71c42c5f5da3fd057bd53ed931e73fb4dcdb829a5e98b94112e0e9 | cb964968cf71c42c5f5da3fd057bd53ed931e73fb4dcdb829a5e98b94112e0e9 | yes |
| 9998-shop SAV game9998.sav | rom2-en | ee46c9257283c739b7babb822f54911fb42eaa9f0ea8ebaf257591b9d4769e6e | ee46c9257283c739b7babb822f54911fb42eaa9f0ea8ebaf257591b9d4769e6e | yes |
| 9998-shop state lines | rom2-en | a3f96862de2b8360f91d70c42d8adbba828c2095771e80d3420b6bfdd49ebe99 | a3f96862de2b8360f91d70c42d8adbba828c2095771e80d3420b6bfdd49ebe99 | yes |
| 9998-tavern frame tavern-book | rom2-en | ae21ca46ad86682a547869ca3eae2522e93c513d79402d8f865e8eaff61bf32b | ae21ca46ad86682a547869ca3eae2522e93c513d79402d8f865e8eaff61bf32b | yes |
| 9998-tavern frame tavern-figure | rom2-en | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | yes |
| 9998-tavern frame tavern-stats | rom2-en | 4856476192ccc149f71b4026dc98b7bce4997e5b0bb27b97b60796202113ddcf | 4856476192ccc149f71b4026dc98b7bce4997e5b0bb27b97b60796202113ddcf | yes |
| 9998-tavern SAV game9998.sav | rom2-en | ee46c9257283c739b7babb822f54911fb42eaa9f0ea8ebaf257591b9d4769e6e | ee46c9257283c739b7babb822f54911fb42eaa9f0ea8ebaf257591b9d4769e6e | yes |
| 9998-tavern state lines | rom2-en | 89c088dce2ce6f3321f4b5e514b16b4f9f9649348746cbc428b5e3776e7a1da5 | 89c088dce2ce6f3321f4b5e514b16b4f9f9649348746cbc428b5e3776e7a1da5 | yes |
| new-save SAV game0000.sav | rom2-en | da43e38d54c2c9f9635c21ebe69017c3edcb76c4d1333df9758ff8285cce352d | da43e38d54c2c9f9635c21ebe69017c3edcb76c4d1333df9758ff8285cce352d | yes |
| new-save state lines | rom2-en | 2fd48fd75b3a26fe033fbd3f615b975c574f7c745a5db18e7be2308d5f250264 | 2fd48fd75b3a26fe033fbd3f615b975c574f7c745a5db18e7be2308d5f250264 | yes |
| new-shop frame shop-book | rom2-en | ada3338177701da1a355edc11dff714fee0b15b66705ba8df5f35c290b68e4b6 | ada3338177701da1a355edc11dff714fee0b15b66705ba8df5f35c290b68e4b6 | yes |
| new-shop frame shop-figure | rom2-en | ad5e59c8015de7c8eb9e0023b78aac28437bd3c3af7cbc630a58688c0632e277 | ad5e59c8015de7c8eb9e0023b78aac28437bd3c3af7cbc630a58688c0632e277 | yes |
| new-shop frame shop-next-book | rom2-en | aed4d920f73488587580c3bfb1ca943ad068da1a0918e1c817af3ee52e1aa843 | aed4d920f73488587580c3bfb1ca943ad068da1a0918e1c817af3ee52e1aa843 | yes |
| new-shop frame shop-next-figure | rom2-en | 5ffdc7119ee5a6030308dd494905dff67b5fc8fc185486e787661c0fe8c69d0f | 5ffdc7119ee5a6030308dd494905dff67b5fc8fc185486e787661c0fe8c69d0f | yes |
| new-shop frame shop-next-stats | rom2-en | 0b3e09d81b72902dc255282beb5484a693e2eaed6a5ccfdc96d9afa076b21069 | 0b3e09d81b72902dc255282beb5484a693e2eaed6a5ccfdc96d9afa076b21069 | yes |
| new-shop frame shop-stats | rom2-en | 8c438be181a6c272fa7036283e2ba575f485bdddc32d6bbaa73bcb921c062744 | 8c438be181a6c272fa7036283e2ba575f485bdddc32d6bbaa73bcb921c062744 | yes |
| new-shop state lines | rom2-en | 2ebc7d2c333d7603540b6384fc85d2659417bc1f4c1a64f4e082587442903b28 | 2ebc7d2c333d7603540b6384fc85d2659417bc1f4c1a64f4e082587442903b28 | yes |
| new-tavern frame tavern-book | rom2-en | ee0e3518f63f90865d445c3db10216e86f93f124fd76e65699bc4863292b92f3 | ee0e3518f63f90865d445c3db10216e86f93f124fd76e65699bc4863292b92f3 | yes |
| new-tavern frame tavern-figure | rom2-en | 815a06e602dde8fa87c08501bd53b6be731e52e2ce907a0f753d6c737d5682f6 | 815a06e602dde8fa87c08501bd53b6be731e52e2ce907a0f753d6c737d5682f6 | yes |
| new-tavern frame tavern-stats | rom2-en | c3e8dfd54dcc63b8410d9e1771af8f7a3e8c1731b298d690d605d1b057a8da78 | c3e8dfd54dcc63b8410d9e1771af8f7a3e8c1731b298d690d605d1b057a8da78 | yes |
| new-tavern state lines | rom2-en | 3c7bec2178cefcf61ef36dd53b11fdba540bbe43204c14fc0b1e10ec9f230b21 | 3c7bec2178cefcf61ef36dd53b11fdba540bbe43204c14fc0b1e10ec9f230b21 | yes |
| 0000-save SAV game0000.sav | rom2-ru | 5f9821363e610b76c28d3851b3dcba55a03885fb51d1a9380add675c7a774def | 5f9821363e610b76c28d3851b3dcba55a03885fb51d1a9380add675c7a774def | yes |
| 0000-save SAV game0001.sav | rom2-ru | 7259eaef7a9d3f87f04124c277bf89c2e1c8a8d949125eeec77cfd48585f595e | 7259eaef7a9d3f87f04124c277bf89c2e1c8a8d949125eeec77cfd48585f595e | yes |
| 0000-save state lines | rom2-ru | 9be7c7a961f13b972fbfc8ba694c80716d1c73fca6a8c3a2849ef0cb9aa4ab10 | 9be7c7a961f13b972fbfc8ba694c80716d1c73fca6a8c3a2849ef0cb9aa4ab10 | yes |
| 0000-shop frame shop-book | rom2-ru | 038ae9a60593362de717dedb000c0956ef56e4712dee43fa7c7eb0f8f4fc9710 | 038ae9a60593362de717dedb000c0956ef56e4712dee43fa7c7eb0f8f4fc9710 | yes |
| 0000-shop frame shop-figure | rom2-ru | ec2f3a859b3a107fed5b096d01577090b62eb9a72a8f356f6e402bd8a351133a | ec2f3a859b3a107fed5b096d01577090b62eb9a72a8f356f6e402bd8a351133a | yes |
| 0000-shop frame shop-next-book | rom2-ru | 3a84d50c040ecab2b504689855e0821de3439921367334ae32b8f22b9aa1cd35 | 3a84d50c040ecab2b504689855e0821de3439921367334ae32b8f22b9aa1cd35 | yes |
| 0000-shop frame shop-next-figure | rom2-ru | 01431fcf6b6ec9f865bcc81b61c886929ccf56040a38069113ed2cd98ac9fe49 | 01431fcf6b6ec9f865bcc81b61c886929ccf56040a38069113ed2cd98ac9fe49 | yes |
| 0000-shop frame shop-next-stats | rom2-ru | cbb4b23e37159b5f4738a96b2e24318f8b90c44a6cca70a554a740af6bba61ee | cbb4b23e37159b5f4738a96b2e24318f8b90c44a6cca70a554a740af6bba61ee | yes |
| 0000-shop frame shop-stats | rom2-ru | c6ba26dcf530db0740e9f715e9a18ee8dbc4eebfe418e32596b7f4dc03ee59b5 | c6ba26dcf530db0740e9f715e9a18ee8dbc4eebfe418e32596b7f4dc03ee59b5 | yes |
| 0000-shop SAV game0000.sav | rom2-ru | 5f9821363e610b76c28d3851b3dcba55a03885fb51d1a9380add675c7a774def | 5f9821363e610b76c28d3851b3dcba55a03885fb51d1a9380add675c7a774def | yes |
| 0000-shop state lines | rom2-ru | a73eab116d9998b3f89fec04513be09b68b821b206c68d3d119f013b7197b3de | a73eab116d9998b3f89fec04513be09b68b821b206c68d3d119f013b7197b3de | yes |
| 0000-tavern frame tavern-book | rom2-ru | b43901511b3005ec148663f7c68f89a19f8d443bd8e141aaa8d5a1e409b4744b | b43901511b3005ec148663f7c68f89a19f8d443bd8e141aaa8d5a1e409b4744b | yes |
| 0000-tavern frame tavern-figure | rom2-ru | 80cd6f81afe5b84bb066589b8be9903f343149be88e2f19a6fbf587cedb03269 | 80cd6f81afe5b84bb066589b8be9903f343149be88e2f19a6fbf587cedb03269 | yes |
| 0000-tavern frame tavern-stats | rom2-ru | 27bde5370a34f39cd16b100c371e306ed753c3d7c6937ee87dfa3d475caa179d | 27bde5370a34f39cd16b100c371e306ed753c3d7c6937ee87dfa3d475caa179d | yes |
| 0000-tavern SAV game0000.sav | rom2-ru | 5f9821363e610b76c28d3851b3dcba55a03885fb51d1a9380add675c7a774def | 5f9821363e610b76c28d3851b3dcba55a03885fb51d1a9380add675c7a774def | yes |
| 0000-tavern state lines | rom2-ru | b39d60f9526342eaa094a1e134f55e973ecf83f608e27fd5da714d7414fa8dfc | b39d60f9526342eaa094a1e134f55e973ecf83f608e27fd5da714d7414fa8dfc | yes |
| 0002-save SAV game0000.sav | rom2-ru | 1aa1c3afa01d8c8cdb1ef5ab7db4d2fcdfd3e3cdfef22726a5c68cb308b3252c | 1aa1c3afa01d8c8cdb1ef5ab7db4d2fcdfd3e3cdfef22726a5c68cb308b3252c | yes |
| 0002-save SAV game0002.sav | rom2-ru | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | yes |
| 0002-save state lines | rom2-ru | e6416bfcaab6a2744411f29d1b08e328428ad2122609df42cae6112aa1b7872f | e6416bfcaab6a2744411f29d1b08e328428ad2122609df42cae6112aa1b7872f | yes |
| 0002-shop frame shop-book | rom2-ru | f042d9fc83d161f527f0182108acdb1e2f056a653e21ad6249786ac6f93606a4 | f042d9fc83d161f527f0182108acdb1e2f056a653e21ad6249786ac6f93606a4 | yes |
| 0002-shop frame shop-figure | rom2-ru | cd48a7aba0a61dce440de29c40fbb41f6c15f19551bca28406e18886d19effbd | cd48a7aba0a61dce440de29c40fbb41f6c15f19551bca28406e18886d19effbd | yes |
| 0002-shop frame shop-next-book | rom2-ru | 25fd391e5dc2a4364b6e3262b188257e05b242ae80236c51f65a1534e9553260 | 25fd391e5dc2a4364b6e3262b188257e05b242ae80236c51f65a1534e9553260 | yes |
| 0002-shop frame shop-next-figure | rom2-ru | 949d3d62671b81dee0d3d645aabe57e04c9b16cd0a608008a8e5b097adf85f79 | 949d3d62671b81dee0d3d645aabe57e04c9b16cd0a608008a8e5b097adf85f79 | yes |
| 0002-shop frame shop-next-stats | rom2-ru | 4f71d2d88b20dc1e1e7681271849358aaec332f10bbfdeab289393981a6a56a1 | 4f71d2d88b20dc1e1e7681271849358aaec332f10bbfdeab289393981a6a56a1 | yes |
| 0002-shop frame shop-stats | rom2-ru | eeed91195b013068201ab540910ceeb1dcbacac59ba44988fd63722ead903500 | eeed91195b013068201ab540910ceeb1dcbacac59ba44988fd63722ead903500 | yes |
| 0002-shop SAV game0002.sav | rom2-ru | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | yes |
| 0002-shop state lines | rom2-ru | ead07e5f1993409a78fc1f2c0b10bc5eec85989fcb15f638a6b7eaa9eb6d46d4 | ead07e5f1993409a78fc1f2c0b10bc5eec85989fcb15f638a6b7eaa9eb6d46d4 | yes |
| 0002-tavern frame tavern-book | rom2-ru | a180f04b55c24de4de80b11b972eb9dc558a5618a119a66cf91dec028f7177b5 | a180f04b55c24de4de80b11b972eb9dc558a5618a119a66cf91dec028f7177b5 | yes |
| 0002-tavern frame tavern-figure | rom2-ru | 03ec45e3eaa0aebdb069c126f2e5057ad48e2823088522696a2e51da1b2915e0 | 03ec45e3eaa0aebdb069c126f2e5057ad48e2823088522696a2e51da1b2915e0 | yes |
| 0002-tavern frame tavern-stats | rom2-ru | 4e8cfeffda03db99578d27df2422a680e71b9a74fbea2461f70009b5914f4d9d | 4e8cfeffda03db99578d27df2422a680e71b9a74fbea2461f70009b5914f4d9d | yes |
| 0002-tavern SAV game0002.sav | rom2-ru | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | 06494ed83c3a85c1f0a5ded424b16c71b3d566188990fbd9ba3ee70744af7b8d | yes |
| 0002-tavern state lines | rom2-ru | bfdaecec7b153a394cc6b07270b505e441ea6ae888d915145bdec3d828ea7ccb | bfdaecec7b153a394cc6b07270b505e441ea6ae888d915145bdec3d828ea7ccb | yes |
| 0003-save SAV game0000.sav | rom2-ru | bb327440e06a8b4c0f79c30fb0b55441c3ad8f467bfcb865b1d83eb42bc7f533 | bb327440e06a8b4c0f79c30fb0b55441c3ad8f467bfcb865b1d83eb42bc7f533 | yes |
| 0003-save SAV game0003.sav | rom2-ru | 76a64e76a06bdb03cd6e8ffc11644831aa55959ff09594004a436d4fb61e02c9 | 76a64e76a06bdb03cd6e8ffc11644831aa55959ff09594004a436d4fb61e02c9 | yes |
| 0003-save state lines | rom2-ru | 67ab59f6421e5588fc50d984720a716c8ab44f36be0eb844dfa7f4c8cbf50f33 | 67ab59f6421e5588fc50d984720a716c8ab44f36be0eb844dfa7f4c8cbf50f33 | yes |
| 0003-shop frame shop-book | rom2-ru | fe301bae284078af9ef6ef0a2047529bf9311e294fb6dc4b6c8b380928dddeba | fe301bae284078af9ef6ef0a2047529bf9311e294fb6dc4b6c8b380928dddeba | yes |
| 0003-shop frame shop-figure | rom2-ru | fb5eae1b12af4825c9437b7b3acf8a194b8ff5b7ac695e74defcd11a798578d9 | fb5eae1b12af4825c9437b7b3acf8a194b8ff5b7ac695e74defcd11a798578d9 | yes |
| 0003-shop frame shop-next-book | rom2-ru | af12b68e5b1e981b21dfc0d8b3ee42001a4de42a94e6f1c911636c48794a1e79 | af12b68e5b1e981b21dfc0d8b3ee42001a4de42a94e6f1c911636c48794a1e79 | yes |
| 0003-shop frame shop-next-figure | rom2-ru | 9368a0f160d17ff718582fea316ca57190d789408ba948ee88af662306928ae6 | 9368a0f160d17ff718582fea316ca57190d789408ba948ee88af662306928ae6 | yes |
| 0003-shop frame shop-next-stats | rom2-ru | a65d4a890d63abf5bd30508d000181e84ac00525533bbf89acc04ed2ac9c89b4 | a65d4a890d63abf5bd30508d000181e84ac00525533bbf89acc04ed2ac9c89b4 | yes |
| 0003-shop frame shop-stats | rom2-ru | ef5c0b85174a375adc55095d927b3e8705987f2c374fb5eb718873370d1c9633 | ef5c0b85174a375adc55095d927b3e8705987f2c374fb5eb718873370d1c9633 | yes |
| 0003-shop SAV game0003.sav | rom2-ru | 76a64e76a06bdb03cd6e8ffc11644831aa55959ff09594004a436d4fb61e02c9 | 76a64e76a06bdb03cd6e8ffc11644831aa55959ff09594004a436d4fb61e02c9 | yes |
| 0003-shop state lines | rom2-ru | bc3869c91e9cb527f64a77f697fb06680cd48a77443de2025393f960e972e65a | bc3869c91e9cb527f64a77f697fb06680cd48a77443de2025393f960e972e65a | yes |
| 0003-tavern frame tavern-book | rom2-ru | b5f3950361cb8fae509cb5c78fbf0eafaf6ff9f87dc98ef79bb9df1e90c49df9 | b5f3950361cb8fae509cb5c78fbf0eafaf6ff9f87dc98ef79bb9df1e90c49df9 | yes |
| 0003-tavern frame tavern-figure | rom2-ru | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | yes |
| 0003-tavern frame tavern-stats | rom2-ru | a33a0977491512b0366311c0ed3127c61e7f2040b49ea8540fd8d64bf9400644 | a33a0977491512b0366311c0ed3127c61e7f2040b49ea8540fd8d64bf9400644 | yes |
| 0003-tavern SAV game0003.sav | rom2-ru | 76a64e76a06bdb03cd6e8ffc11644831aa55959ff09594004a436d4fb61e02c9 | 76a64e76a06bdb03cd6e8ffc11644831aa55959ff09594004a436d4fb61e02c9 | yes |
| 0003-tavern state lines | rom2-ru | 469ab1419293c2e11a633a5d1f5b73d5e72926a3289e7c7f13ec11cda30190e6 | 469ab1419293c2e11a633a5d1f5b73d5e72926a3289e7c7f13ec11cda30190e6 | yes |
| 9998-save SAV game0000.sav | rom2-ru | 3cf60927cbf706fdbc4773330c8cfad96fef0cabb4a4a54ab9b5de940be574e7 | 3cf60927cbf706fdbc4773330c8cfad96fef0cabb4a4a54ab9b5de940be574e7 | yes |
| 9998-save SAV game9998.sav | rom2-ru | ee46c9257283c739b7babb822f54911fb42eaa9f0ea8ebaf257591b9d4769e6e | ee46c9257283c739b7babb822f54911fb42eaa9f0ea8ebaf257591b9d4769e6e | yes |
| 9998-save state lines | rom2-ru | d001d053b2f7feff37a631d5ea5cf3a22372fbbacb8be2bd93637b94832f036e | d001d053b2f7feff37a631d5ea5cf3a22372fbbacb8be2bd93637b94832f036e | yes |
| 9998-shop frame shop-book | rom2-ru | 83a91ff304f2988ad562b6355e22f271f7472c57083e1904b1807575cdb710c3 | 83a91ff304f2988ad562b6355e22f271f7472c57083e1904b1807575cdb710c3 | yes |
| 9998-shop frame shop-figure | rom2-ru | 94bfa7668c856ad368575ef707ce71b2ae448bcf7b9c472d785f0ddeed64d088 | 94bfa7668c856ad368575ef707ce71b2ae448bcf7b9c472d785f0ddeed64d088 | yes |
| 9998-shop frame shop-next-book | rom2-ru | c634ad21061dc843250257a6bdd64934b94bd87485d792dd2ccb9528f0a738b3 | c634ad21061dc843250257a6bdd64934b94bd87485d792dd2ccb9528f0a738b3 | yes |
| 9998-shop frame shop-next-figure | rom2-ru | d9aa3ffffa48514948b9a256892953d377bd42833567279007b572f83de04819 | d9aa3ffffa48514948b9a256892953d377bd42833567279007b572f83de04819 | yes |
| 9998-shop frame shop-next-stats | rom2-ru | fc88394d6dd96b2e67b24084aa337b966f557a0675cd2ede8efd5a4118a7cce4 | fc88394d6dd96b2e67b24084aa337b966f557a0675cd2ede8efd5a4118a7cce4 | yes |
| 9998-shop frame shop-stats | rom2-ru | 3330ee3338843be8a9d235db5c2664b462064c597ced37313819aebc7a35e8f9 | 3330ee3338843be8a9d235db5c2664b462064c597ced37313819aebc7a35e8f9 | yes |
| 9998-shop SAV game9998.sav | rom2-ru | ee46c9257283c739b7babb822f54911fb42eaa9f0ea8ebaf257591b9d4769e6e | ee46c9257283c739b7babb822f54911fb42eaa9f0ea8ebaf257591b9d4769e6e | yes |
| 9998-shop state lines | rom2-ru | bc3869c91e9cb527f64a77f697fb06680cd48a77443de2025393f960e972e65a | bc3869c91e9cb527f64a77f697fb06680cd48a77443de2025393f960e972e65a | yes |
| 9998-tavern frame tavern-book | rom2-ru | b5f3950361cb8fae509cb5c78fbf0eafaf6ff9f87dc98ef79bb9df1e90c49df9 | b5f3950361cb8fae509cb5c78fbf0eafaf6ff9f87dc98ef79bb9df1e90c49df9 | yes |
| 9998-tavern frame tavern-figure | rom2-ru | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | yes |
| 9998-tavern frame tavern-stats | rom2-ru | a33a0977491512b0366311c0ed3127c61e7f2040b49ea8540fd8d64bf9400644 | a33a0977491512b0366311c0ed3127c61e7f2040b49ea8540fd8d64bf9400644 | yes |
| 9998-tavern SAV game9998.sav | rom2-ru | ee46c9257283c739b7babb822f54911fb42eaa9f0ea8ebaf257591b9d4769e6e | ee46c9257283c739b7babb822f54911fb42eaa9f0ea8ebaf257591b9d4769e6e | yes |
| 9998-tavern state lines | rom2-ru | 469ab1419293c2e11a633a5d1f5b73d5e72926a3289e7c7f13ec11cda30190e6 | 469ab1419293c2e11a633a5d1f5b73d5e72926a3289e7c7f13ec11cda30190e6 | yes |
| new-save SAV game0000.sav | rom2-ru | 50abf04ece3f72dedd3171ff3b2d7c4559bee665addeebcb93503e75342f1dc5 | 50abf04ece3f72dedd3171ff3b2d7c4559bee665addeebcb93503e75342f1dc5 | yes |
| new-save state lines | rom2-ru | be3b7931560a6688e2222af694f72764ce8487a65303785bc7b5833c5b3e3470 | be3b7931560a6688e2222af694f72764ce8487a65303785bc7b5833c5b3e3470 | yes |
| new-shop frame shop-book | rom2-ru | 35cc70f656e9581fe9857adcca402e23fe684daf7b9593570b6524e1f41ddbbd | 35cc70f656e9581fe9857adcca402e23fe684daf7b9593570b6524e1f41ddbbd | yes |
| new-shop frame shop-figure | rom2-ru | 58db68ccc3c9b1f2b9c11f25d61074296efee2416876d549ee52d65d3fffee74 | 58db68ccc3c9b1f2b9c11f25d61074296efee2416876d549ee52d65d3fffee74 | yes |
| new-shop frame shop-next-book | rom2-ru | 204227167760e1add7593388221205c32340292f526accd3dc24afd63b883c0c | 204227167760e1add7593388221205c32340292f526accd3dc24afd63b883c0c | yes |
| new-shop frame shop-next-figure | rom2-ru | 7d36e44130560eac5c4bf64335277f489c1188e3b2eb0e050b7c6272174398be | 7d36e44130560eac5c4bf64335277f489c1188e3b2eb0e050b7c6272174398be | yes |
| new-shop frame shop-next-stats | rom2-ru | bd21228d15ec45cd337dee5d6a0872e136c0b73663cabd248b598573080f35a5 | bd21228d15ec45cd337dee5d6a0872e136c0b73663cabd248b598573080f35a5 | yes |
| new-shop frame shop-stats | rom2-ru | 5043c76497b55116acb2aaae4e0d11a2e6d25cd658a269d923fb7f808f4b906b | 5043c76497b55116acb2aaae4e0d11a2e6d25cd658a269d923fb7f808f4b906b | yes |
| new-shop state lines | rom2-ru | 5a208aa4334cfecf7288c0342c218fb24e4c1f1799edf7c56cde570d2f3aaeb0 | 5a208aa4334cfecf7288c0342c218fb24e4c1f1799edf7c56cde570d2f3aaeb0 | yes |
| new-tavern frame tavern-book | rom2-ru | 9bea1b8a2d01844920c173ccdd2189a04f37c351d089cdf0db7bcad542542763 | 9bea1b8a2d01844920c173ccdd2189a04f37c351d089cdf0db7bcad542542763 | yes |
| new-tavern frame tavern-figure | rom2-ru | b1e5e68b492a4d8e271044c220e6ec7f27336c03f407ed471d10bded8952d6d8 | b1e5e68b492a4d8e271044c220e6ec7f27336c03f407ed471d10bded8952d6d8 | yes |
| new-tavern frame tavern-stats | rom2-ru | e96de96d8c99619f1049ea18edc552cee655591f30886f8358fb4d48f3ee857c | e96de96d8c99619f1049ea18edc552cee655591f30886f8358fb4d48f3ee857c | yes |
| new-tavern state lines | rom2-ru | 7a55ebcbd7ca69b317ab811c7bf6f6018de70ba743ce678f1bf2a9ad99047f45 | 7a55ebcbd7ca69b317ab811c7bf6f6018de70ba743ce678f1bf2a9ad99047f45 | yes |
| 9711-next-school frame school-next-book | ru | e82198dc6fb4da2f72998d281a72c5820f8748c1b72ba14761bd05eb40970887 | e82198dc6fb4da2f72998d281a72c5820f8748c1b72ba14761bd05eb40970887 | yes |
| 9711-next-school frame school-next-figure | ru | e82198dc6fb4da2f72998d281a72c5820f8748c1b72ba14761bd05eb40970887 | e82198dc6fb4da2f72998d281a72c5820f8748c1b72ba14761bd05eb40970887 | yes |
| 9711-next-school frame school-next-stats | ru | 2e43758a3fe66a10fb623cd76b18204c364574787873e121381f4650ce78ffa1 | 2e43758a3fe66a10fb623cd76b18204c364574787873e121381f4650ce78ffa1 | yes |
| 9711-next-school state lines | ru | a145c41607d78e84cb42be5bcc64754c9a7af56f84c3d0778f6dc1d3f84e1699 | a145c41607d78e84cb42be5bcc64754c9a7af56f84c3d0778f6dc1d3f84e1699 | yes |
| 9711-next-tavern frame tavern-next-book | ru | cd9c46b5d2e2303ee5baec85d9059c0556447627755d258fe2d14f77f218ee95 | cd9c46b5d2e2303ee5baec85d9059c0556447627755d258fe2d14f77f218ee95 | yes |
| 9711-next-tavern frame tavern-next-figure | ru | cd9c46b5d2e2303ee5baec85d9059c0556447627755d258fe2d14f77f218ee95 | cd9c46b5d2e2303ee5baec85d9059c0556447627755d258fe2d14f77f218ee95 | yes |
| 9711-next-tavern frame tavern-next-stats | ru | b3c170a1a9e4a2fe27da3a9bdecfa6acf02d3848b4f86c4ec721eff9faf3c8d1 | b3c170a1a9e4a2fe27da3a9bdecfa6acf02d3848b4f86c4ec721eff9faf3c8d1 | yes |
| 9711-next-tavern state lines | ru | 6f0bcabeed0976eae9c16ce5ea97943032696626a7dd9f76f72cfb1ddd2da7b4 | 6f0bcabeed0976eae9c16ce5ea97943032696626a7dd9f76f72cfb1ddd2da7b4 | yes |
| 9711-save SAV pane proof.sav | ru | 931ef4a5ab8c27a26a41d774fde69cc45740a14099265acf24d2f397cf7697b6 | 931ef4a5ab8c27a26a41d774fde69cc45740a14099265acf24d2f397cf7697b6 | yes |
| 9711-save state lines | ru | d96e175c5458e864c3a03f78a2734c7a6dea23d8c0b63885c529ef67c8eb5a16 | d96e175c5458e864c3a03f78a2734c7a6dea23d8c0b63885c529ef67c8eb5a16 | yes |
| 9711-save-after-shop SAV pane proof.sav | ru | fdb5ec03e183156aa66aa4630dbb9d060896fd614aef498db4111b9965f42b36 | fdb5ec03e183156aa66aa4630dbb9d060896fd614aef498db4111b9965f42b36 | yes |
| 9711-save-after-shop state lines | ru | 1649851d8ef4d7372bb1a7eea19b466d0eadcb6e7b4d6cc3311239da3a003fa9 | 1649851d8ef4d7372bb1a7eea19b466d0eadcb6e7b4d6cc3311239da3a003fa9 | yes |
| 9711-school frame school-book | ru | 1c794cd2299d0dfbb8ba45681ff2add80af187e67e8be69fcbefd11608289831 | 1c794cd2299d0dfbb8ba45681ff2add80af187e67e8be69fcbefd11608289831 | yes |
| 9711-school frame school-figure | ru | 1c794cd2299d0dfbb8ba45681ff2add80af187e67e8be69fcbefd11608289831 | 1c794cd2299d0dfbb8ba45681ff2add80af187e67e8be69fcbefd11608289831 | yes |
| 9711-school frame school-stats | ru | 726c5334c5f96fe90b1ddcdfde25febdf025fa468f3d17b7b809f1d0c61acb49 | 726c5334c5f96fe90b1ddcdfde25febdf025fa468f3d17b7b809f1d0c61acb49 | yes |
| 9711-school state lines | ru | 3eadbccfc113ef6e8e3a19eacd6fc47d5635bc62f664555884d5d0b87c879d49 | 3eadbccfc113ef6e8e3a19eacd6fc47d5635bc62f664555884d5d0b87c879d49 | yes |
| 9711-shop frame shop-book | ru | cd2f507ad905afdfaa6682048b3433da07d9f1d7f31cf13caf15288d1674751b | cd2f507ad905afdfaa6682048b3433da07d9f1d7f31cf13caf15288d1674751b | yes |
| 9711-shop frame shop-figure | ru | 16e97ef18775a0f5f8146d67685a56d747eab2a7f95a04e0f86340e6d496e982 | 16e97ef18775a0f5f8146d67685a56d747eab2a7f95a04e0f86340e6d496e982 | yes |
| 9711-shop frame shop-next-book | ru | 022cb83f0b7c9206ae606ddf70f243995e9efcc49142f55c074f68caafa7e412 | 022cb83f0b7c9206ae606ddf70f243995e9efcc49142f55c074f68caafa7e412 | yes |
| 9711-shop frame shop-next-figure | ru | 7402dbb90fbedbab736b9e891d81848ddf79666b352240a6e5332f89b4d56a02 | 7402dbb90fbedbab736b9e891d81848ddf79666b352240a6e5332f89b4d56a02 | yes |
| 9711-shop frame shop-next-stats | ru | c7ea12a56b4bdd97ce42ab1c5c9fa51a427e87c26195a8e2a08d76caaa042b08 | c7ea12a56b4bdd97ce42ab1c5c9fa51a427e87c26195a8e2a08d76caaa042b08 | yes |
| 9711-shop frame shop-stats | ru | 69c7a6e32c9206d5b21261383af88c0942372c71c43c8cda5a9ba1d72f72da90 | 69c7a6e32c9206d5b21261383af88c0942372c71c43c8cda5a9ba1d72f72da90 | yes |
| 9711-shop state lines | ru | 291262eb0952b2ecdf9e953cb522e903768ab8745783b3b4ac1f1870787b5fe8 | 291262eb0952b2ecdf9e953cb522e903768ab8745783b3b4ac1f1870787b5fe8 | yes |
| 9711-tavern frame tavern-book | ru | 05b9420d599d5cc70542bdd4048b200bb28fef2cc157f9393b0d8a8e3d491b14 | 05b9420d599d5cc70542bdd4048b200bb28fef2cc157f9393b0d8a8e3d491b14 | yes |
| 9711-tavern frame tavern-figure | ru | 05b9420d599d5cc70542bdd4048b200bb28fef2cc157f9393b0d8a8e3d491b14 | 05b9420d599d5cc70542bdd4048b200bb28fef2cc157f9393b0d8a8e3d491b14 | yes |
| 9711-tavern frame tavern-stats | ru | 72d4b9115f66dd0ddb68d3edc68e67d657d18741269d8f11a7586bddb0ae2054 | 72d4b9115f66dd0ddb68d3edc68e67d657d18741269d8f11a7586bddb0ae2054 | yes |
| 9711-tavern state lines | ru | 630db95af65ede0d8590dc57dea9284f8d5cb107838fe4c150b3adbe791d2e03 | 630db95af65ede0d8590dc57dea9284f8d5cb107838fe4c150b3adbe791d2e03 | yes |
| 9714-next-school frame school-next-book | ru | f36f24b34a8cba1d6c0f65296091e75496f90a2a103192dbe40f69bb14997a9e | f36f24b34a8cba1d6c0f65296091e75496f90a2a103192dbe40f69bb14997a9e | yes |
| 9714-next-school frame school-next-figure | ru | 233a5d1a5d9396426a515ca839870ab384a9779e7ce6d32c4dabc8733e4b3060 | 233a5d1a5d9396426a515ca839870ab384a9779e7ce6d32c4dabc8733e4b3060 | yes |
| 9714-next-school frame school-next-stats | ru | 3cce87c3d99dada23e3935f21068e6bd61b1be0a750490ad9c5fd6673522e3e0 | 3cce87c3d99dada23e3935f21068e6bd61b1be0a750490ad9c5fd6673522e3e0 | yes |
| 9714-next-school state lines | ru | ca88e15d935ac76d23503afd3e49e7883b7625767afa26c05db3e2b87882a110 | ca88e15d935ac76d23503afd3e49e7883b7625767afa26c05db3e2b87882a110 | yes |
| 9714-next-tavern frame tavern-next-book | ru | 50db976b0a6b72659d67cb6e542a68dc481ac7e6fc5d32dd93f5a3d1063e741f | 50db976b0a6b72659d67cb6e542a68dc481ac7e6fc5d32dd93f5a3d1063e741f | yes |
| 9714-next-tavern frame tavern-next-figure | ru | 50db976b0a6b72659d67cb6e542a68dc481ac7e6fc5d32dd93f5a3d1063e741f | 50db976b0a6b72659d67cb6e542a68dc481ac7e6fc5d32dd93f5a3d1063e741f | yes |
| 9714-next-tavern frame tavern-next-stats | ru | fece8d3af1691cd820aca82eabf62ed5d3da4206490c0c2283c3073d826e4b30 | fece8d3af1691cd820aca82eabf62ed5d3da4206490c0c2283c3073d826e4b30 | yes |
| 9714-next-tavern state lines | ru | 460b0bf5f32f39e0faedadf38fb2dc264cb5012fb797a366706a0db803becf29 | 460b0bf5f32f39e0faedadf38fb2dc264cb5012fb797a366706a0db803becf29 | yes |
| 9714-save SAV pane proof.sav | ru | e47d90ff3b69ff0f9dea86f8db56a274b60210533abd6599dec370018a31502a | e47d90ff3b69ff0f9dea86f8db56a274b60210533abd6599dec370018a31502a | yes |
| 9714-save state lines | ru | d249f00cb48a18e01a247edcbcf5c9e54884ba930792ecf6bf95d8b21851c0a0 | d249f00cb48a18e01a247edcbcf5c9e54884ba930792ecf6bf95d8b21851c0a0 | yes |
| 9714-save-after-shop SAV pane proof.sav | ru | e47d90ff3b69ff0f9dea86f8db56a274b60210533abd6599dec370018a31502a | e47d90ff3b69ff0f9dea86f8db56a274b60210533abd6599dec370018a31502a | yes |
| 9714-save-after-shop state lines | ru | bdd184891b859c04de4d874d5699ce82ade0d867fabac4f393a2a8a03fa01641 | bdd184891b859c04de4d874d5699ce82ade0d867fabac4f393a2a8a03fa01641 | yes |
| 9714-school frame school-book | ru | bdeb2a26ce083ffbe9e699dc3f1ae5cded34a0d07f94fede0026af6d9a5f0f1a | bdeb2a26ce083ffbe9e699dc3f1ae5cded34a0d07f94fede0026af6d9a5f0f1a | yes |
| 9714-school frame school-figure | ru | 0be1d338cb9d14552548b507b7602acfecdeffbe73cc0e26ea9e5dd14f38230f | 0be1d338cb9d14552548b507b7602acfecdeffbe73cc0e26ea9e5dd14f38230f | yes |
| 9714-school frame school-stats | ru | 8fc4ed71520b02816fdf4c8e30055bc47a88c5839320a7550b13df761c45821b | 8fc4ed71520b02816fdf4c8e30055bc47a88c5839320a7550b13df761c45821b | yes |
| 9714-school state lines | ru | 044f1e89f75606d6931cdbf968f9a211e78fa8e7ff16c27a1b1032b68df4f715 | 044f1e89f75606d6931cdbf968f9a211e78fa8e7ff16c27a1b1032b68df4f715 | yes |
| 9714-shop frame shop-book | ru | c53aedc7dc2a8a71d990ac29d6b2bb763e22587bd3049e44cb4fd3756f8542ed | c53aedc7dc2a8a71d990ac29d6b2bb763e22587bd3049e44cb4fd3756f8542ed | yes |
| 9714-shop frame shop-figure | ru | 1179d93b09bdc5277fb25a44fb653db071873772ddf86be3a0cbcee65e63ca22 | 1179d93b09bdc5277fb25a44fb653db071873772ddf86be3a0cbcee65e63ca22 | yes |
| 9714-shop frame shop-next-book | ru | 5176d7855a1c63fffa6c36a51f8b6a13c49fd87e0239ddc310f188361bd53d4c | 5176d7855a1c63fffa6c36a51f8b6a13c49fd87e0239ddc310f188361bd53d4c | yes |
| 9714-shop frame shop-next-figure | ru | d5f19c0bbd912b07518e1f1f2bdd7b39bcd3f3bd595774c9e43c353dc65fe44f | d5f19c0bbd912b07518e1f1f2bdd7b39bcd3f3bd595774c9e43c353dc65fe44f | yes |
| 9714-shop frame shop-next-stats | ru | 567939c8d11e59a36aff231c087eea6bfc0caae77240e5f64bd7789a0765fd93 | 567939c8d11e59a36aff231c087eea6bfc0caae77240e5f64bd7789a0765fd93 | yes |
| 9714-shop frame shop-stats | ru | 49a491b06ab1dc3bc6db2121842141942424df06b87690221025542e7b33ffc1 | 49a491b06ab1dc3bc6db2121842141942424df06b87690221025542e7b33ffc1 | yes |
| 9714-shop state lines | ru | 8311fa3b3e73b43c8db7533b983cde9b99ae37f6dfaf7bd0956e213153ea5d3a | 8311fa3b3e73b43c8db7533b983cde9b99ae37f6dfaf7bd0956e213153ea5d3a | yes |
| 9714-tavern frame tavern-book | ru | 2da8cef2dd0704b67df791a2c4661ece9fcbb3901655a35fa455886e427c9e85 | 2da8cef2dd0704b67df791a2c4661ece9fcbb3901655a35fa455886e427c9e85 | yes |
| 9714-tavern frame tavern-figure | ru | 2da8cef2dd0704b67df791a2c4661ece9fcbb3901655a35fa455886e427c9e85 | 2da8cef2dd0704b67df791a2c4661ece9fcbb3901655a35fa455886e427c9e85 | yes |
| 9714-tavern frame tavern-stats | ru | 54bb706a1e6ff4e5058d71010cb68641a94bb13d9d3758c20a39ffa84179cca6 | 54bb706a1e6ff4e5058d71010cb68641a94bb13d9d3758c20a39ffa84179cca6 | yes |
| 9714-tavern state lines | ru | b1770859a463b5005d76cfcd57c17a6b975551c3c5a9bd955f220eb7df2e1453 | b1770859a463b5005d76cfcd57c17a6b975551c3c5a9bd955f220eb7df2e1453 | yes |
| 9717-next-school frame school-next-book | ru | bebcfb7d961d216ad7bec28044da94375ab084a56903d4622e282a8e2461ee7c | bebcfb7d961d216ad7bec28044da94375ab084a56903d4622e282a8e2461ee7c | yes |
| 9717-next-school frame school-next-figure | ru | bebcfb7d961d216ad7bec28044da94375ab084a56903d4622e282a8e2461ee7c | bebcfb7d961d216ad7bec28044da94375ab084a56903d4622e282a8e2461ee7c | yes |
| 9717-next-school frame school-next-stats | ru | 88e552b8bf6cf8a4cb1fa4146b04bd5cc12945ce3e52eba637b7e0ff46695387 | 88e552b8bf6cf8a4cb1fa4146b04bd5cc12945ce3e52eba637b7e0ff46695387 | yes |
| 9717-next-school state lines | ru | 46e06a8f339ebbabb3b7e5b916974b42a598ccafc13841b48d7bf879b22c1ee5 | 46e06a8f339ebbabb3b7e5b916974b42a598ccafc13841b48d7bf879b22c1ee5 | yes |
| 9717-next-tavern frame tavern-next-book | ru | cd960e520da9df223c4514123649b8ae950507d1a42f7cbb411a17ca8facc0d0 | cd960e520da9df223c4514123649b8ae950507d1a42f7cbb411a17ca8facc0d0 | yes |
| 9717-next-tavern frame tavern-next-figure | ru | cd960e520da9df223c4514123649b8ae950507d1a42f7cbb411a17ca8facc0d0 | cd960e520da9df223c4514123649b8ae950507d1a42f7cbb411a17ca8facc0d0 | yes |
| 9717-next-tavern frame tavern-next-stats | ru | 14c0dc7b7671334372f5c994d2d0805644c39d95a426e7d4e071db35712ad9dc | 14c0dc7b7671334372f5c994d2d0805644c39d95a426e7d4e071db35712ad9dc | yes |
| 9717-next-tavern state lines | ru | 5b5efcee39c29fbc50f1e9917d7d77c1113453304ff96481971c5fd2df7ee9e3 | 5b5efcee39c29fbc50f1e9917d7d77c1113453304ff96481971c5fd2df7ee9e3 | yes |
| 9717-save SAV pane proof.sav | ru | 3f98e7506d58cee831d9149d9a7edf2e76d3bb08b85f7c663a12bdff4f2afe2b | 3f98e7506d58cee831d9149d9a7edf2e76d3bb08b85f7c663a12bdff4f2afe2b | yes |
| 9717-save state lines | ru | 5288a9fd63d33bb63e961c97c780ad8cbc3edad0fa797d3eb1e8cecc5cbcadd9 | 5288a9fd63d33bb63e961c97c780ad8cbc3edad0fa797d3eb1e8cecc5cbcadd9 | yes |
| 9717-save-after-shop SAV pane proof.sav | ru | 3f98e7506d58cee831d9149d9a7edf2e76d3bb08b85f7c663a12bdff4f2afe2b | 3f98e7506d58cee831d9149d9a7edf2e76d3bb08b85f7c663a12bdff4f2afe2b | yes |
| 9717-save-after-shop state lines | ru | 2908c0f456debca0b38cd05e4d66c462af544f674b1b98a902f111b86d1d33c7 | 2908c0f456debca0b38cd05e4d66c462af544f674b1b98a902f111b86d1d33c7 | yes |
| 9717-school frame school-book | ru | bcce242f164ab36ea117c8001109644a35e9b23ff6653ac04664bfb8344f265d | bcce242f164ab36ea117c8001109644a35e9b23ff6653ac04664bfb8344f265d | yes |
| 9717-school frame school-figure | ru | bcce242f164ab36ea117c8001109644a35e9b23ff6653ac04664bfb8344f265d | bcce242f164ab36ea117c8001109644a35e9b23ff6653ac04664bfb8344f265d | yes |
| 9717-school frame school-stats | ru | c1ac33ae644641fcaca51f660b04077cb3d07ca74e24521212c5c1447f8fcb0b | c1ac33ae644641fcaca51f660b04077cb3d07ca74e24521212c5c1447f8fcb0b | yes |
| 9717-school state lines | ru | 7565126b6743cec5b43ecf66a0855ed249e3f262b91e293704d26bc5e918d897 | 7565126b6743cec5b43ecf66a0855ed249e3f262b91e293704d26bc5e918d897 | yes |
| 9717-shop frame shop-book | ru | 97227ee2d6e8c742ef1055c1f01d24f623769ed6c722d17e753f228158431265 | 97227ee2d6e8c742ef1055c1f01d24f623769ed6c722d17e753f228158431265 | yes |
| 9717-shop frame shop-figure | ru | 006e531f912443760953569ffdae608d187f4b4f43778878e7591c050a41047c | 006e531f912443760953569ffdae608d187f4b4f43778878e7591c050a41047c | yes |
| 9717-shop frame shop-next-book | ru | 21cfa6dce3b031b67a9f1d9b611dc5b44dd89c10d6c6c6362cb52d6b37abda60 | 21cfa6dce3b031b67a9f1d9b611dc5b44dd89c10d6c6c6362cb52d6b37abda60 | yes |
| 9717-shop frame shop-next-figure | ru | 7ea9d501b86a03b926f5832c2161cbad2b3161a559114ae7ba544a8426a5ad97 | 7ea9d501b86a03b926f5832c2161cbad2b3161a559114ae7ba544a8426a5ad97 | yes |
| 9717-shop frame shop-next-stats | ru | 8720a88cb2250662791a352dbf863050ecfa46916cbeec461a37ac3ee616ddb3 | 8720a88cb2250662791a352dbf863050ecfa46916cbeec461a37ac3ee616ddb3 | yes |
| 9717-shop frame shop-stats | ru | d0e645bdd989245c82f0cb8a0b5d04c91bed0b122baa14a347ac829583e35547 | d0e645bdd989245c82f0cb8a0b5d04c91bed0b122baa14a347ac829583e35547 | yes |
| 9717-shop state lines | ru | 78dd7ff02bd24d48d1d689bfd40d3d54dde1ef2b63c28d5102713001b8dabe41 | 78dd7ff02bd24d48d1d689bfd40d3d54dde1ef2b63c28d5102713001b8dabe41 | yes |
| 9717-tavern frame tavern-book | ru | c38250f2ab5cc218d317e41c28b783994b650a8d04f1731e0bce3ad9d8b23389 | c38250f2ab5cc218d317e41c28b783994b650a8d04f1731e0bce3ad9d8b23389 | yes |
| 9717-tavern frame tavern-figure | ru | c38250f2ab5cc218d317e41c28b783994b650a8d04f1731e0bce3ad9d8b23389 | c38250f2ab5cc218d317e41c28b783994b650a8d04f1731e0bce3ad9d8b23389 | yes |
| 9717-tavern frame tavern-stats | ru | 2c0ee4e0c42599ebaee9baf3e3daddc3605735756f95ee322000956bd794026c | 2c0ee4e0c42599ebaee9baf3e3daddc3605735756f95ee322000956bd794026c | yes |
| 9717-tavern state lines | ru | 667d7d48d693750387d75603f406d73732553ee8882eaf78e877356e573dc77f | 667d7d48d693750387d75603f406d73732553ee8882eaf78e877356e573dc77f | yes |

## Open debt

- A first-game pane cannot move: the chapter town check refuses a rectangle
  other than the character region. The pane's hit tests follow the
  rectangle; moving it also needs the tavern, shop and school pages to give
  their controls a place beside it.
- `BookOffset` is read by the second game's shop page layout only; the
  chapter town check refuses it in the first game.
- A pane with no party member is reached by `TestHeroPaneWithNoPartyMember`
  only: the first game materialises a hero for an empty party and the second
  game's new game carries one, so no headless run shows it.
- The first game's new game starts in a mission, so no first-game run starts
  from a new game in town.
- The picker states cover the member the shop's next arrow selects; the other
  members of the full-pack saves are not captured.
- The pane figure has two composers, the first game's shop figure cache and
  the second game's `heroFace`; `heroPane` reads both through the shop host's
  `face` hook. Making them one is a separate kind.
