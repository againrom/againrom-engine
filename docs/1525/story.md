# One tavern or inn page builder for both games

## Intent

The owner requires the screens both games have to go one way, town pages
next and the tavern first. Before this story the tavern or inn page was
built two ways. The first game's tavern held its seat cells, card and figure
offsets in Go (`pkg/ui/townshell.go`). The second game's inn read its room's
`seats`, `pane` and `buttons` and painted them with its own seat painter and
its own patron card producer (`pkg/game/secondtown.go`,
`pkg/game/secondinnpatron.go`). A card or seat fix reached one game only, and
the first game's tavern could not be moved by data.

After this story one room page builder lays out both pages from room data
and one painter draws them, with one hit test for seats and buttons. Every
frame of both pages is byte-equal to its base.

## Authority

- Owner rulings: one builder per kind (`AGENTS.md`); the first game is a
  reduced second game; screens both games have go one way, town pages next,
  the tavern first. These accepted the recommendations of architecture audit
  `3d1771a7`, whose row 5 names the two tavern or inn page builders.
- TOWN-467: the tavern roster places list position `i` in rect `i` of an
  18-rect table filled bottom row first, column `i%6`, row `⌊i/6⌋` counted
  up from the bottom. The first game's seat grid is 6 columns by 3 rows.
- TOWN-468: mercenary cells come first and talk-only cells after them, with
  different backgrounds. The first `hires` seats take the hire back.
- TAVERN-TALKSTATS-017: the character panel is drawn in rect
  `(12,0)-(172,238)`. The card is the rect `(0,0)-(160,238)` drawn at
  offset `(12,0)`.
- The figure's centring in `(0,238)-(160,480)` and its `(8,1)` shift are
  the owner's, as the base drew them; the data cites `owner`.

## As built

- Kind: tavern or inn page. Builder: `buildRoomPage` and `placeRoomPerson`
  in `pkg/game/roompageview.go`; painter `ui.DrawRoomPage` in
  `pkg/ui/roompage.go`; hit test `ui.RoomPage.SeatAt` and `ButtonAt`.
  `buildRoomPage` takes the room, the page art lookup, the seat count and
  the hire count. It returns the seat rectangles with their backs and the
  button rectangles. `placeRoomPerson` places the selected seat's card and
  figure by the room's `card` and `figure` data. There is no branch on game
  identity.
- The first game's tavern: `TownSurface` builds `TownSurfaceView.Page`
  through `buildRoomPage` from the town's tavern room, or the profile's when
  a mod town names no seats. `ComposeTownSurface` fills the seats' frames and
  captions from the roster cells and draws the page through
  `DrawRoomPage`. `TownSurfaceControlAt`, the hover lines and the card
  tooltip read the page. The candidate detail and talk detail render the
  person through `ui.RenderRoomPerson`.
- The second game's inn: `townRoomView.page` replaces `paintSeats`, and
  `patron` replaces `paintPatron`. `ControlAt` and the route witness hit the
  page built by `buildRoomPage`. The hero pane doll places its figure
  through the same `RoomFigure.Origin`; the hero pane keeps its builder.
- Seat data (`pkg/town/seats.go`, data only): `SeatFigure.Centre` selects
  the first game's centred figure. A seat grid with `frame-ms` 0 runs no seat
  clock; a negative period is refused. `SeatSpec.SeatAt` and `town.ButtonAt`
  are removed; the page's hit test replaces them.
- Profile data (`pkg/game/towns/rom1.json`, tavern room `seats`): first
  `[176,416,224,480]`, step `[48,0]`, row step `[0,-64]`, 6 columns,
  3 rows, back `man-back-talk`, hire back `man-back`, card and figure as
  above.
- Guard: `internal/archtest` `CheckRoomPage` fails when a non-test file in
  `pkg/` outside the builder and `pkg/town` reads a seat grid's `First`,
  `Step`, `RowStep`, `Columns`, `Rows`, `Count`, `Seat`, `Card` or `Figure`,
  or when the builder or the painter passes a number literal to `image.Rect`
  or `image.Pt`.
- Untouched kinds, each with its builder: town screen host, inn offers and
  hire rows, hero pane, talk box, room tip, world-map page.

## Proof

### Frames

Instrument: the game built at engine main `29c6b4f9` (from `git archive`)
and at the merged candidate `82b937bb` (`scripts/build-tools.sh`). Headless
scenarios (frontend stage v10) loaded each save and opened the tavern or
inn. `cmd/screenshot` captured the first game's new-game tavern screens.
Inputs: the first game's EN town kit saves `game9711`, `game9714`,
`game9715` and `game9716`, read by the `en` and `ru` installs, with and
without the room tip; the second game's owner files `game0000`,
`game0002`, `game0003` and `game9998`, read by `rom2-en` and `rom2-ru`.
71 frames, 71 byte-equal:

| state | main `29c6b4f9` SHA-256 | candidate `82b937bb` SHA-256 | equal |
|---|---|---|---|
| en-game9711 tavern-entry | f06937efc70ff8d6893877f8180ae98001b7966d4ee588fd79412b35fd0f33ef | f06937efc70ff8d6893877f8180ae98001b7966d4ee588fd79412b35fd0f33ef | yes |
| en-game9711 tavern-mercenary | f06937efc70ff8d6893877f8180ae98001b7966d4ee588fd79412b35fd0f33ef | f06937efc70ff8d6893877f8180ae98001b7966d4ee588fd79412b35fd0f33ef | yes |
| en-game9711 tavern-talker | 9c540b2865cc1b4ff8eb276ffcacb342207cc22a25c216f31cbe57b7ccf2cf23 | 9c540b2865cc1b4ff8eb276ffcacb342207cc22a25c216f31cbe57b7ccf2cf23 | yes |
| en-game9711-notip tavern-entry | c851fe7031c3ffbf0b340422cd6b0c04f659262a2003a746798dc7e16de99412 | c851fe7031c3ffbf0b340422cd6b0c04f659262a2003a746798dc7e16de99412 | yes |
| en-game9711-notip tavern-mercenary | c851fe7031c3ffbf0b340422cd6b0c04f659262a2003a746798dc7e16de99412 | c851fe7031c3ffbf0b340422cd6b0c04f659262a2003a746798dc7e16de99412 | yes |
| en-game9711-notip tavern-talker | 23858a5bf23542d8987ea3a8a7d2d0d2ab95f9c937ef8a7b39233250a4f84bb1 | 23858a5bf23542d8987ea3a8a7d2d0d2ab95f9c937ef8a7b39233250a4f84bb1 | yes |
| en-game9714 tavern-entry | 4f776cf2ee66cc48d309badb764f5a7804bcf903855e32704772a07c059129e3 | 4f776cf2ee66cc48d309badb764f5a7804bcf903855e32704772a07c059129e3 | yes |
| en-game9714 tavern-mercenary | 94559287b85ed5dac597c6aee0c3051a5fe5e734604d824a306d8a45cfb3e10d | 94559287b85ed5dac597c6aee0c3051a5fe5e734604d824a306d8a45cfb3e10d | yes |
| en-game9714 tavern-talker | a8f8f456a1cbaa2280e4055273cbddc7388221671927c2a0dbacaaba192d3c13 | a8f8f456a1cbaa2280e4055273cbddc7388221671927c2a0dbacaaba192d3c13 | yes |
| en-game9714-notip tavern-entry | 7c907c9ce42ffc6165da7f3a5f9b8172f319f765d9792326887eaf4eda8c0206 | 7c907c9ce42ffc6165da7f3a5f9b8172f319f765d9792326887eaf4eda8c0206 | yes |
| en-game9714-notip tavern-mercenary | 05e562eaf69af8e9053d2312c39d6d6c3b187c0f7eb693c2b2de4b9165bc8739 | 05e562eaf69af8e9053d2312c39d6d6c3b187c0f7eb693c2b2de4b9165bc8739 | yes |
| en-game9714-notip tavern-talker | 357253c924447926bd1760bc56ca0908d572d6082d194cae35548d915502222b | 357253c924447926bd1760bc56ca0908d572d6082d194cae35548d915502222b | yes |
| en-game9715 tavern-entry | 4581c767ba9642ebc40c1125d593e6da5a0930bbc40a3aaf3e1b9ad54002a863 | 4581c767ba9642ebc40c1125d593e6da5a0930bbc40a3aaf3e1b9ad54002a863 | yes |
| en-game9715 tavern-mercenary | 1e00e114235b9f63f989000453673c25c586125d4763023b77aca26b4824d1b0 | 1e00e114235b9f63f989000453673c25c586125d4763023b77aca26b4824d1b0 | yes |
| en-game9715 tavern-talker | f467020b0ff823430d49951ea9fd78f93331db9d0bbec548e0a114e253d1c118 | f467020b0ff823430d49951ea9fd78f93331db9d0bbec548e0a114e253d1c118 | yes |
| en-game9715-notip tavern-entry | a75af3be2c9e963f5ae3d5320cc046f3b34588eec375c99ea0205b1185d1c58b | a75af3be2c9e963f5ae3d5320cc046f3b34588eec375c99ea0205b1185d1c58b | yes |
| en-game9715-notip tavern-mercenary | a6ed574c46dd85b29a125899ff1dd97c6e0d9cb7c7ce0b84952cb8665d93c29c | a6ed574c46dd85b29a125899ff1dd97c6e0d9cb7c7ce0b84952cb8665d93c29c | yes |
| en-game9715-notip tavern-talker | 869db041882fd7c745fcd5dae9bd16ec5cd29ba16b761a83e2d4e2cd2ebd9dcd | 869db041882fd7c745fcd5dae9bd16ec5cd29ba16b761a83e2d4e2cd2ebd9dcd | yes |
| en-game9716 tavern-entry | 3482157fd442c43b6c1bb888b2edf76c5b138800e8b8aa101316ca49d13e8f72 | 3482157fd442c43b6c1bb888b2edf76c5b138800e8b8aa101316ca49d13e8f72 | yes |
| en-game9716 tavern-mercenary | d530ee015422f4e4991d2c6c669bd2c1d37f89b465dd052b385fb1f7dc9e2142 | d530ee015422f4e4991d2c6c669bd2c1d37f89b465dd052b385fb1f7dc9e2142 | yes |
| en-game9716 tavern-talker | e24d0788d4c83fc456578b640d43075c2aa54b7f1de612d9c89d7e6a2a19b43a | e24d0788d4c83fc456578b640d43075c2aa54b7f1de612d9c89d7e6a2a19b43a | yes |
| en-game9716-notip tavern-entry | dfb7abc490c50b532b8d8a10fb41a733bee83f7d440efbaddd8bbbc4dfa7b4b2 | dfb7abc490c50b532b8d8a10fb41a733bee83f7d440efbaddd8bbbc4dfa7b4b2 | yes |
| en-game9716-notip tavern-mercenary | 93574742bc54393459db5beae60b7a1633d72bf58c06bd3b1f8bc14ef1befcef | 93574742bc54393459db5beae60b7a1633d72bf58c06bd3b1f8bc14ef1befcef | yes |
| en-game9716-notip tavern-talker | bf953fbe6096a217c72622b27df6457cd1abc306c2252a44adb6c9b88205ca83 | bf953fbe6096a217c72622b27df6457cd1abc306c2252a44adb6c9b88205ca83 | yes |
| rom2-en-game0000 inn-entry | 7da02977920a1ddcf602855e0c5f7d4a2d1d2dd24ad55c93c1aa9f53554b023c | 7da02977920a1ddcf602855e0c5f7d4a2d1d2dd24ad55c93c1aa9f53554b023c | yes |
| rom2-en-game0000 inn-tip-closed | 1c117c15b548d19b8cbf1b75f02edbe4ddd063a0b986644351c922f3565ad4e9 | 1c117c15b548d19b8cbf1b75f02edbe4ddd063a0b986644351c922f3565ad4e9 | yes |
| rom2-en-game0002 inn-entry | 4a4b8c3d01dc1123bfd51161d31300a8a98c777ad37e8c136fbfc44fbf9b5a4f | 4a4b8c3d01dc1123bfd51161d31300a8a98c777ad37e8c136fbfc44fbf9b5a4f | yes |
| rom2-en-game0002 inn-tip-closed | 534ce37b200cbe68615b6a7fdc3614c9fbcdc0ba89fc6981b59eddce200eb9c5 | 534ce37b200cbe68615b6a7fdc3614c9fbcdc0ba89fc6981b59eddce200eb9c5 | yes |
| rom2-en-game0003 inn-entry | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | yes |
| rom2-en-game0003 inn-tip-closed | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | yes |
| rom2-en-game9998 inn-entry | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | d7f81b28a578d07266e75ee2f690fb4534020aa0dd8b2408b1c74cb51ba07c03 | yes |
| rom2-en-game9998 inn-tip-closed | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | 87a5ed5f739a68815ce7884a63243cc711def168169475aa491254c3dadc682a | yes |
| rom2-ru-game0000 inn-entry | 80cd6f81afe5b84bb066589b8be9903f343149be88e2f19a6fbf587cedb03269 | 80cd6f81afe5b84bb066589b8be9903f343149be88e2f19a6fbf587cedb03269 | yes |
| rom2-ru-game0000 inn-tip-closed | d30f33469342e571bb4b65458a293ecfcd401b752b4508944614721d1617939a | d30f33469342e571bb4b65458a293ecfcd401b752b4508944614721d1617939a | yes |
| rom2-ru-game0002 inn-entry | 03ec45e3eaa0aebdb069c126f2e5057ad48e2823088522696a2e51da1b2915e0 | 03ec45e3eaa0aebdb069c126f2e5057ad48e2823088522696a2e51da1b2915e0 | yes |
| rom2-ru-game0002 inn-tip-closed | ffdfcbff89a11853cc7f5e8866f5b6c6fbe101e786011c7792a344b7652e2a38 | ffdfcbff89a11853cc7f5e8866f5b6c6fbe101e786011c7792a344b7652e2a38 | yes |
| rom2-ru-game0003 inn-entry | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | yes |
| rom2-ru-game0003 inn-tip-closed | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | yes |
| rom2-ru-game9998 inn-entry | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | bc4bf3d7fb6128b4afc4a7f5ac9267da51606415d43823dcdd2d0a4acc88922e | yes |
| rom2-ru-game9998 inn-tip-closed | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | 18e6f37903fb83e61a8cac9577ef604f9db67db50831034411b96115e78cfa54 | yes |
| ru-game9711 tavern-entry | 05b9420d599d5cc70542bdd4048b200bb28fef2cc157f9393b0d8a8e3d491b14 | 05b9420d599d5cc70542bdd4048b200bb28fef2cc157f9393b0d8a8e3d491b14 | yes |
| ru-game9711 tavern-mercenary | 05b9420d599d5cc70542bdd4048b200bb28fef2cc157f9393b0d8a8e3d491b14 | 05b9420d599d5cc70542bdd4048b200bb28fef2cc157f9393b0d8a8e3d491b14 | yes |
| ru-game9711 tavern-talker | 0635fde3ca03a157598f3d68edaafbb9b862665bd9763a3b1a8de678b882cf25 | 0635fde3ca03a157598f3d68edaafbb9b862665bd9763a3b1a8de678b882cf25 | yes |
| ru-game9711-notip tavern-entry | ad5cb23daf163de531227475481af181f62cb71f646b8c39a62854eea309b9ed | ad5cb23daf163de531227475481af181f62cb71f646b8c39a62854eea309b9ed | yes |
| ru-game9711-notip tavern-mercenary | ad5cb23daf163de531227475481af181f62cb71f646b8c39a62854eea309b9ed | ad5cb23daf163de531227475481af181f62cb71f646b8c39a62854eea309b9ed | yes |
| ru-game9711-notip tavern-talker | 041eeaf8d1f69f6c6506e8e8e8b1e8a924974d8169f7c552c7d720c18e1e95d1 | 041eeaf8d1f69f6c6506e8e8e8b1e8a924974d8169f7c552c7d720c18e1e95d1 | yes |
| ru-game9714 tavern-entry | 2da8cef2dd0704b67df791a2c4661ece9fcbb3901655a35fa455886e427c9e85 | 2da8cef2dd0704b67df791a2c4661ece9fcbb3901655a35fa455886e427c9e85 | yes |
| ru-game9714 tavern-mercenary | 524de376cc63bc49bbcdd16d990c888d6fffad64b0d7f8c2167e912a15dedee8 | 524de376cc63bc49bbcdd16d990c888d6fffad64b0d7f8c2167e912a15dedee8 | yes |
| ru-game9714 tavern-talker | acb89c2c46ba7b9486cf69923796935da2bf4006dc01e5673208dcb650ba6893 | acb89c2c46ba7b9486cf69923796935da2bf4006dc01e5673208dcb650ba6893 | yes |
| ru-game9714-notip tavern-entry | e3ff0bbb6fb57131119eecf29127e2946de9b2de7d9a2d598df2d660652d0358 | e3ff0bbb6fb57131119eecf29127e2946de9b2de7d9a2d598df2d660652d0358 | yes |
| ru-game9714-notip tavern-mercenary | 357c79b8e93631334cc903bafd75983e9c6c2e732d58c67b7eb309ff613e8507 | 357c79b8e93631334cc903bafd75983e9c6c2e732d58c67b7eb309ff613e8507 | yes |
| ru-game9714-notip tavern-talker | 84303137fc05a1b5f5e16d6d14730070b97af69a418c8840d17f175733f28449 | 84303137fc05a1b5f5e16d6d14730070b97af69a418c8840d17f175733f28449 | yes |
| ru-game9715 tavern-entry | 12313adf383e89a0716914d45a8b613992d7225ec61cf36781561b7401d2e064 | 12313adf383e89a0716914d45a8b613992d7225ec61cf36781561b7401d2e064 | yes |
| ru-game9715 tavern-mercenary | 1b2160987bd487541c2c2d3c1ec6ea66e4252e84bd7d4152c81bb340406025f1 | 1b2160987bd487541c2c2d3c1ec6ea66e4252e84bd7d4152c81bb340406025f1 | yes |
| ru-game9715 tavern-talker | ffa99cfeed0dcdbd91812b8bc6a9d95049686b74992e623193e13e02ddb42ba8 | ffa99cfeed0dcdbd91812b8bc6a9d95049686b74992e623193e13e02ddb42ba8 | yes |
| ru-game9715-notip tavern-entry | f1e49eef08a01568760bd165f00f9944daa5485580550e24100ee6935961bb32 | f1e49eef08a01568760bd165f00f9944daa5485580550e24100ee6935961bb32 | yes |
| ru-game9715-notip tavern-mercenary | 4d7629b72508ea548fbdef5ee523ba232bea641e7b53885868cb94710ba048c1 | 4d7629b72508ea548fbdef5ee523ba232bea641e7b53885868cb94710ba048c1 | yes |
| ru-game9715-notip tavern-talker | 7f473ac3b25d0d7c5596c65a9ba2ac6cb56a321401241b30c4c2358c0471000c | 7f473ac3b25d0d7c5596c65a9ba2ac6cb56a321401241b30c4c2358c0471000c | yes |
| ru-game9716 tavern-entry | 7681cfb1b27913fd9d7d49ec4f031ff890786ec3b12fa68cdc58865cbc625fe3 | 7681cfb1b27913fd9d7d49ec4f031ff890786ec3b12fa68cdc58865cbc625fe3 | yes |
| ru-game9716 tavern-mercenary | 28ca97f54e131897d1f4d1f8864d17626ce488173ebb062f4422f48f6e87e57b | 28ca97f54e131897d1f4d1f8864d17626ce488173ebb062f4422f48f6e87e57b | yes |
| ru-game9716 tavern-talker | dc01b4f1f4332e187917aba13cd1bd149c561d7b34458227cec2a742b200cb09 | dc01b4f1f4332e187917aba13cd1bd149c561d7b34458227cec2a742b200cb09 | yes |
| ru-game9716-notip tavern-entry | 5f155a7c1e8cb0a2681185430556e097a194d463447cfe3fe8c840b8a2e9059d | 5f155a7c1e8cb0a2681185430556e097a194d463447cfe3fe8c840b8a2e9059d | yes |
| ru-game9716-notip tavern-mercenary | 82eb56e27e8b606dfc3fd30a87305920ab643aa1d20301029a7bd50d760c86ca | 82eb56e27e8b606dfc3fd30a87305920ab643aa1d20301029a7bd50d760c86ca | yes |
| ru-game9716-notip tavern-talker | ef532f59faee99ba03e200e7de00266289c68c47563202f9f36f6b53bbd71d35 | ef532f59faee99ba03e200e7de00266289c68c47563202f9f36f6b53bbd71d35 | yes |
| en new-game town-tavern-next-frame | 157f7e4d0611b02250c71ae509f6b964ce26f075ceadb6da281187a0bf1a1bb9 | 157f7e4d0611b02250c71ae509f6b964ce26f075ceadb6da281187a0bf1a1bb9 | yes |
| en new-game town-tavern-selected | 8be774cf4dace6d2b08ecdd991dc4896aa1ece9a11a8c3c543a374685ebc88f9 | 8be774cf4dace6d2b08ecdd991dc4896aa1ece9a11a8c3c543a374685ebc88f9 | yes |
| en new-game town-tavern-talk | c410357dda48f7dc5098f59b1c9a9f428e2c87b1b5d6a3f80e1fcad393fdb4cd | c410357dda48f7dc5098f59b1c9a9f428e2c87b1b5d6a3f80e1fcad393fdb4cd | yes |
| en new-game town-tavern | 8be774cf4dace6d2b08ecdd991dc4896aa1ece9a11a8c3c543a374685ebc88f9 | 8be774cf4dace6d2b08ecdd991dc4896aa1ece9a11a8c3c543a374685ebc88f9 | yes |
| ru new-game town-tavern-next-frame | 8097176bf1bec32f6087eb0748f003106f35208bbc83992411214790106a2711 | 8097176bf1bec32f6087eb0748f003106f35208bbc83992411214790106a2711 | yes |
| ru new-game town-tavern-selected | 3c57df7729ebfc0cd559b31ffd2432eafd488ee0b3ac2095d9ef2d5d606a7e17 | 3c57df7729ebfc0cd559b31ffd2432eafd488ee0b3ac2095d9ef2d5d606a7e17 | yes |
| ru new-game town-tavern | 3c57df7729ebfc0cd559b31ffd2432eafd488ee0b3ac2095d9ef2d5d606a7e17 | 3c57df7729ebfc0cd559b31ffd2432eafd488ee0b3ac2095d9ef2d5d606a7e17 | yes |

The same 71 states captured at the story base `243ed697` and at the
story's tree before its merge are also 71 of 71 byte-equal.

### Tests

- `pkg/ui` `TestRoomPageFollowsItsData`: the seats and the figure moved in
  the page data move their pixels, `SeatAt` follows the moved seats,
  `ButtonAt` answers the button, and a centred figure lands at its centred
  origin. `TestRoomPageHiddenSeatIsHitButNotDrawn`.
- `pkg/game` `TestRoomPageMovesWithItsRoomData`,
  `TestRoomPersonCardFollowsItsRoomData` (synthetic room data) and
  `TestFirstTavernRoomCarriesItsPageGeometry` (the shipped rom1 data: seats
  0, 5, 6 and 17, 18 seats, the card, the centred figure's origin).
- `internal/archtest` `TestRoomPageHasOneBuilder` and its synthetic cases.
  Loss controls: `image.Pt(12, 0)` put back for the card offset in
  `placeRoomPerson` fails it at `pkg/game/roompageview.go:44`; a read of
  `r.Seats.Figure` in `pkg/game/secondheropane.go` fails it at line 114.
  Both were restored.

## Open debt

- A first-game roster over 18 entries draws 18 seats, as TOWN-467's
  18-rect table states; the base drew more rows. The shipped campaign
  offers at most 16 entries. No frame covers a longer roster.
- The candidate inspection picture is 172 pixels wide (the union of the
  figure and card rectangles) where the base's was 176; its pixels are
  unchanged and the tavern frames are equal.
- The talk offer on the first game's RU new-game tavern is not reached by
  `cmd/screenshot` at the base or the candidate; the headless saves cover
  the talk-only seats in RU.
- The inn offers and hire rows (audit row 4) and the hero pane (row 11)
  keep their own builders.
