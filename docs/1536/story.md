# One town talk producer for both games

## Intent

The owner requires the screens and components both games have to go one
way, in the second game's shape, with the first game as profile data. Town
pages go in order: the town host, the tavern or inn page, the inn offers,
the hero pane, then the talk box. Before this story the talk box had one
window (`ui.TownTalk` through `townTalkSeat`), one cursor (`talkCursor`) and
per-game profile values, but two view producers, two page turns and two face
resolvers: `chapterTownTalk.talkView` and `turnTownTalk` with
`townScreen.speakerFace` (first game), `sectionTownTalk.talkView` and
`turnTownTalk` with the screen's `talkFaces` resolver (second game). A fix to
what the box shows, how it turns a page or which face it draws reached one
game only.

After this story one producer, one page turn, one part lookup and one face
resolver serve both games. What differed is `townTalkProfile` data keyed by
the town host JSON talk rule. Every captured frame, capture record and SAV of
both games is byte-equal to its base.

## Authority

- Owner rulings: one builder per kind (`AGENTS.md`); the first game is a
  reduced second game; screens both games have go one way, town pages in
  order, the talk box after the hero pane.
- Census `cross-game-kinds-8b7ee0e5`, rank 12: one window, one cursor and
  profile data, two view producers, two page turns and two face resolvers;
  recommended shape `townTalkSeat.view` over {cursor, part source, speaker
  resolver}, with the section part and its trim, the portrait when the
  section names a speaker and the face by tag as the shared shape, and the
  talk profile, the diagnostic line and the mercenary portrait as the first
  game's data.
- No claim is cited or changed: frames and SAV bytes are equal to the base,
  so no behaviour moved. No DIV row is added.

## As built

Kinds and builders:

- Town talk view producer: `townTalkSeat.view` (`pkg/game/towntalk.go`).
- Town talk page turn: `townTalkSeat.turnTownTalk`; `AdvanceTownDialogue`
  passes it to `ui.TownTalk.Advance`.
- Town talk part lookup: `townDialogueState.lookup(cursor, audience,
  profile)` (`pkg/game/towndialogue.go`), read by the seat's `readTalk` and by
  `townLines`.
- Town talk face resolver: `townScreen.talkFace`, over
  `speakerResolver.SpeakerFace` (`pkg/game/speakers.go`), the resolver the
  mission notice reads (`missionNotices.faces`). Under the profile's
  `mercenaryFaces` it answers an open mercenary talk's candidate and a
  platoon record's unit picture in the record's window, as the first game did.
- Talk cursor: `talkCursor`, reused. The first game's cursor stays the
  screen's `talkPage`; the second game's stays `secondCampaign.talk`. The
  campaign adapter answers which one through `talkCursor()` and what the
  talk's end does through `endTalk()` (`pkg/game/towncampaign.go`,
  `campaignfirst.go`, `campaignsecond.go`).
- Town talk window painter: `ui.TownTalk`, unchanged.
- Town talk profile: `townTalkProfile` with the window's
  `ui.TownTalkProfile` and four fields, `trim`, `openBeforePart`,
  `standingFace` and `mercenaryFaces`. `townTalkRules` maps the host talk
  rule of the town JSON (`building` in `pkg/game/towns/rom1.json`, `section`
  in the three second-game towns) to its profile; a rule it does not name
  takes the first game's. No town JSON field is added, so no CHANGELOG line.

The seat reads its screen through `townTalkHost` (cursor, open, end,
audience, face, art, profile); `townScreen` is the one host. The seat holds
the page (`townDialogueState`, now with `shown`) and the diagnostic line.
Removed: `townTalkSource`, `chapterTownTalk`, `sectionTownTalk`,
`boundTalkSource`, `dialoguePart`, `speakerFace`, `lookupTownDialogue`,
`noteSpeaker`, and the screen's `frame` and `talkFaces` fields (the window's
frame comes from `t.art.menuFrame()`, the second game's speakers from
`t.resolver`). No branch on game identity remains in the producer, the page
turn or the resolver; the profile's fields are the only per-game answers.

One kept difference in timing: the second game's face is now resolved when a
part reads (open and turn), as the first game's was, rather than at every
view. The resolver is built at the talk's open and its inputs do not change
during a talk; the frames below are equal.

## Proof

### Tests

`pkg/game/towntalkprofile_test.go` changes one datum and reads the talk under
both games' tag rules and profiles:

- `TestTownTalkTrimIsProfileData`: the part trim on and off.
- `TestTownTalkSpeakerAndFaceAreData`: `npc=3` against `npc=4`, a changed
  face and window for the same speaker, and a part naming nobody.
- `TestTownTalkStandingFaceIsProfileData`: the standing face on and off over
  a part naming nobody, and the end of the talk after its last part.
- `TestTownTalkOpenBeforePartIsProfileData`: a talk whose first part does not
  read, with and without opening before a part.
- `TestFirstTownScreenTalkFollowsData` and
  `TestSecondTownScreenTalkFollowsData`: on each game's own screen a trim set
  on the host rule's profile changes the page, and a changed record window
  moves the face window.

The existing town talk tests were moved to the one seat and pass
(`towntalk_test.go`, `towntalkreader_test.go`, `offercontinuation_test.go`,
`townscreen_test.go`).

### Guard and loss control

`internal/archtest/towntalk_test.go`, `TestTownTalkHasOneProducer`: in
`pkg/game` non-test code only `townTalkSeat.view` in `towntalk.go` names
`ui.TownTalkView` or `ui.TownTalkPage`, and only `townTalkSeat` declares a
`turnTownTalk` method. Loss control on `e035cc99`: a first-game
`chapterTownTalk` with its own `talkView` and `turnTownTalk` put back in
`pkg/game/townhost.go` builds, and the guard fails with five findings (four
view names outside the producer, one second page turn) and its count check
(9 view names, 2 page turns); after
`git checkout e035cc99 -- pkg/game/townhost.go` it passes.

### Frames, state and SAV

Instrument: `againrom.exe --headless` (scenario version 10, `-seed 1`) built
with `scripts/build-tools.sh` at the base (a `git archive` of `326e1853`) and
at the head `e035cc99`. The base binary carries the base's
`chapterTownTalk` symbols and the head's does not; its build stamp reads the
worktree's HEAD, which reaches no `pkg` code. Inputs: the first game's EN
kit saves 9711, 9713, 9716, 9718 and the original's 9721; the second game's
owner saves 0000, 0002, 0003, 9998 and a new game; the installs `en`, `ru`,
`rom2-en` and `rom2-ru`. Each state opens one talk, takes a frame and a
capture record on every page, turns each page with the talk's own control
(`dialogue`, `notice`), takes a frame and a capture after the talk, leaves
the room with Escape and saves through the game menu. The page count of each
talk is the count of turns after which the talk stops refusing other
controls, measured on the head and used for both.

States: first game tavern offer talks (NPC 22, 25, 30, 62; up to 12 pages),
mercenary talks (types 1, 7, 9, 10, 14; up to 3 pages; EN only), the shop's
entry talk and the gate talk; second game inn talks `TALK 207` (1 page),
`TALK 2108` (2 pages) and `TALK 517` (8 or 12 pages) and the gate talk.
Every multi-page talk is captured on its first, every middle and its last
page, and after its end.

A state's digest is the SHA-256 of its manifest: every frame PNG, every SAV
in the run's save folders after the run, and every capture record (state
and town, the run folder's path removed), each by name and SHA-256. The
town screen has no World, so the capture record stands for the hashed state.
59 states, 59 equal:

| State | Root | Frames | SAV files | Captures | Base SHA-256 | Head SHA-256 | Equal |
|---|---|---|---|---|---|---|---|
| 9711-gates | en | 2 | 2 | 3 | `3397c120f7b71b587f9562aef4838d31f1b75e27a9dd26be0662487028d57504` | `3397c120f7b71b587f9562aef4838d31f1b75e27a9dd26be0662487028d57504` | yes |
| 9711-shop-entry | en | 5 | 2 | 5 | `83c346174db77e8cf0a63d3b90bead97b6a6415cb422c566a86a1d9da2486120` | `83c346174db77e8cf0a63d3b90bead97b6a6415cb422c566a86a1d9da2486120` | yes |
| 9711-tavern-merc14 | en | 4 | 2 | 4 | `c21060b8164b2ce755551d010de633f7234ffb97e610202d24f8b716514a4b81` | `c21060b8164b2ce755551d010de633f7234ffb97e610202d24f8b716514a4b81` | yes |
| 9711-tavern-npc22 | en | 6 | 2 | 6 | `6dbfd342f2c3c05b4bfd48bdcda3b3e20ac92dad2be08d2a4f33a7bfcaf094e3` | `6dbfd342f2c3c05b4bfd48bdcda3b3e20ac92dad2be08d2a4f33a7bfcaf094e3` | yes |
| 9713-tavern-merc10 | en | 3 | 2 | 3 | `c2ccc2e8ebfd69c69e14bb20b61a3902a529581f3f5095529d7518ac672b0681` | `c2ccc2e8ebfd69c69e14bb20b61a3902a529581f3f5095529d7518ac672b0681` | yes |
| 9713-tavern-npc25 | en | 3 | 2 | 3 | `ccc9f6e21d6e8a7543f9ec366a7f81331d1043481e1fd483c36dbd3a7339bfcd` | `ccc9f6e21d6e8a7543f9ec366a7f81331d1043481e1fd483c36dbd3a7339bfcd` | yes |
| 9716-tavern-merc1 | en | 4 | 2 | 4 | `eaed8df7bafe087b8fd7170567b39238170bbd634676b0b48d13eea00c8cc754` | `eaed8df7bafe087b8fd7170567b39238170bbd634676b0b48d13eea00c8cc754` | yes |
| 9716-tavern-npc62 | en | 5 | 2 | 5 | `0a938b7da39ad6c5259c0800f0fea27f78a4c1f35c04acbf4e1727aa01b01583` | `0a938b7da39ad6c5259c0800f0fea27f78a4c1f35c04acbf4e1727aa01b01583` | yes |
| 9718-tavern-merc7 | en | 3 | 2 | 3 | `db95215e7c5f3084cf3806a7d22b4473e14ef84af73a08a6b49bcf53efdcb2ba` | `db95215e7c5f3084cf3806a7d22b4473e14ef84af73a08a6b49bcf53efdcb2ba` | yes |
| 9718-tavern-npc30 | en | 8 | 2 | 8 | `bc42ea8ebc8540fd5ea3744f3424d9c7e3cff7ef4b381ea51d205dffb1803515` | `bc42ea8ebc8540fd5ea3744f3424d9c7e3cff7ef4b381ea51d205dffb1803515` | yes |
| 9721-tavern-merc9 | en | 5 | 2 | 5 | `bf24e61b131ad7ec89f2a864283055b7ba5e9d6727b365c408d08e0abc5085e6` | `bf24e61b131ad7ec89f2a864283055b7ba5e9d6727b365c408d08e0abc5085e6` | yes |
| 9721-tavern-npc25 | en | 14 | 2 | 14 | `a0373ac2aecc0d17c6d81f7ef06ad65e30e4715bd8c0c711d5b8121a7594162f` | `a0373ac2aecc0d17c6d81f7ef06ad65e30e4715bd8c0c711d5b8121a7594162f` | yes |
| 0000-gates | rom2-en | 2 | 2 | 3 | `761bd119a8e4a76a187b44255ce50c2261263b85fc53dc6126af8346699f7d76` | `761bd119a8e4a76a187b44255ce50c2261263b85fc53dc6126af8346699f7d76` | yes |
| 0000-inn-207 | rom2-en | 3 | 2 | 3 | `9b6205453a291cd1998ad57ba138714b7b09e8d95447862f26eb5f157e702d25` | `9b6205453a291cd1998ad57ba138714b7b09e8d95447862f26eb5f157e702d25` | yes |
| 0000-inn-2108 | rom2-en | 4 | 2 | 4 | `ff453674159e3df1a4688c7d7674f4abf654be1f3aca7b82c3bef8cf454f25a2` | `ff453674159e3df1a4688c7d7674f4abf654be1f3aca7b82c3bef8cf454f25a2` | yes |
| 0000-inn-517 | rom2-en | 10 | 2 | 10 | `8bbfdf1a4e235ba4180e3f8dcc0eb13d15d1ad26edd1d6896b2cd54de69da0b1` | `8bbfdf1a4e235ba4180e3f8dcc0eb13d15d1ad26edd1d6896b2cd54de69da0b1` | yes |
| 0002-gates | rom2-en | 2 | 2 | 3 | `634f45ec59120c58a464d17e1918cc822e76f75d8281d3474463081dc516a7f9` | `634f45ec59120c58a464d17e1918cc822e76f75d8281d3474463081dc516a7f9` | yes |
| 0002-inn-207 | rom2-en | 3 | 2 | 3 | `4ecf0a38eeebdcc47624f577c828031050c93c62a6a601928aa7dcaa23d8bd3b` | `4ecf0a38eeebdcc47624f577c828031050c93c62a6a601928aa7dcaa23d8bd3b` | yes |
| 0002-inn-2108 | rom2-en | 4 | 2 | 4 | `e65737cdc0e6dfbf03a1f8f1d25ba3f71fb0c0ccf663162dba464e8b4edfadd4` | `e65737cdc0e6dfbf03a1f8f1d25ba3f71fb0c0ccf663162dba464e8b4edfadd4` | yes |
| 0002-inn-517 | rom2-en | 14 | 2 | 14 | `da452f0fd92e56eb14e8a84fccd22fc191cb079d65a9d62bf93462ec7b9aded1` | `da452f0fd92e56eb14e8a84fccd22fc191cb079d65a9d62bf93462ec7b9aded1` | yes |
| 0003-gates | rom2-en | 2 | 2 | 3 | `3f0f539675fe0fc8ac3c3f249fcc247536226f151ddb7fe4126b98107175a1dd` | `3f0f539675fe0fc8ac3c3f249fcc247536226f151ddb7fe4126b98107175a1dd` | yes |
| 0003-inn-207 | rom2-en | 3 | 2 | 3 | `7868f812fb73fa541c0a2c2b91f76081d923014413025c58537d756d89c353da` | `7868f812fb73fa541c0a2c2b91f76081d923014413025c58537d756d89c353da` | yes |
| 0003-inn-2108 | rom2-en | 4 | 2 | 4 | `62dad4619694b8ab9aee2e9bb8289c1a367425d72b21a0e8ae1115e381033296` | `62dad4619694b8ab9aee2e9bb8289c1a367425d72b21a0e8ae1115e381033296` | yes |
| 0003-inn-517 | rom2-en | 14 | 2 | 14 | `0c0ad964d5821565961884d3330ab7f9a29ef8bd4114c274961106300f433b68` | `0c0ad964d5821565961884d3330ab7f9a29ef8bd4114c274961106300f433b68` | yes |
| 9998-gates | rom2-en | 1 | 1 | 1 | `ef49fc5e964310c18a189e07d97ded411567a4183f6aa8f554cb3e9f8435f571` | `ef49fc5e964310c18a189e07d97ded411567a4183f6aa8f554cb3e9f8435f571` | yes |
| 9998-inn-207 | rom2-en | 3 | 2 | 3 | `54dc7c9bc829af9fd40191fb62f8e8a4edbc345ced8ccb506f8f838b46b8ee80` | `54dc7c9bc829af9fd40191fb62f8e8a4edbc345ced8ccb506f8f838b46b8ee80` | yes |
| 9998-inn-2108 | rom2-en | 4 | 2 | 4 | `d9c9190eecb7b9b07bf9bdc9bc7de38c6bd5a1e589aca7dbeccb5f5c38dc82dd` | `d9c9190eecb7b9b07bf9bdc9bc7de38c6bd5a1e589aca7dbeccb5f5c38dc82dd` | yes |
| 9998-inn-517 | rom2-en | 14 | 2 | 14 | `93676903527d5cbc14c6ba7f8b5ad064e24565fa672bf2e2082490c931d7ffe6` | `93676903527d5cbc14c6ba7f8b5ad064e24565fa672bf2e2082490c931d7ffe6` | yes |
| new-gates | rom2-en | 2 | 1 | 3 | `7ce615670cfe123e4942067aa8195e92394285060cdac06a26034c31dd308a48` | `7ce615670cfe123e4942067aa8195e92394285060cdac06a26034c31dd308a48` | yes |
| new-inn-207 | rom2-en | 3 | 1 | 3 | `82ef84048d1796af65b1fd80a31711b8d8cfae7154ee43be7a7506a9da13a74b` | `82ef84048d1796af65b1fd80a31711b8d8cfae7154ee43be7a7506a9da13a74b` | yes |
| new-inn-2108 | rom2-en | 4 | 1 | 4 | `183b8cda18b52d7bd35c4c321b27b6b3432db14ebb947b7de1b193556a0ee268` | `183b8cda18b52d7bd35c4c321b27b6b3432db14ebb947b7de1b193556a0ee268` | yes |
| new-inn-517 | rom2-en | 10 | 1 | 10 | `30eed7da2bee4a177ea89af43224d743980e5eaf03c718aed2d7df862dac36c1` | `30eed7da2bee4a177ea89af43224d743980e5eaf03c718aed2d7df862dac36c1` | yes |
| 0000-gates | rom2-ru | 2 | 2 | 3 | `23efd509487bee9f5d3e9755f9943969e8f5ca91276e4e090117632ae96ef0f7` | `23efd509487bee9f5d3e9755f9943969e8f5ca91276e4e090117632ae96ef0f7` | yes |
| 0000-inn-207 | rom2-ru | 3 | 2 | 3 | `13aafb26874c6d065e3346ba3c57dc2199ea9d6f99e53995beac0984001d80cd` | `13aafb26874c6d065e3346ba3c57dc2199ea9d6f99e53995beac0984001d80cd` | yes |
| 0000-inn-2108 | rom2-ru | 4 | 2 | 4 | `f6eb87d2a9adb7f42503154ab39a696dca13662f9f9e805bd7cb771cbf1a7d35` | `f6eb87d2a9adb7f42503154ab39a696dca13662f9f9e805bd7cb771cbf1a7d35` | yes |
| 0000-inn-517 | rom2-ru | 10 | 2 | 10 | `9cdbd3bd216601011ab76d03dedb8956597691c949a9bb71ef066ab26119dc03` | `9cdbd3bd216601011ab76d03dedb8956597691c949a9bb71ef066ab26119dc03` | yes |
| 0002-gates | rom2-ru | 2 | 2 | 3 | `e256f6a2ae83380ba0127f30ec7f7dba0abcf47270c0460831e1fd689251e445` | `e256f6a2ae83380ba0127f30ec7f7dba0abcf47270c0460831e1fd689251e445` | yes |
| 0002-inn-207 | rom2-ru | 3 | 2 | 3 | `1a5fae1f585bf37f41a3e3da8cd97e979d67bb8e90d49a597c0a791ff8161bc9` | `1a5fae1f585bf37f41a3e3da8cd97e979d67bb8e90d49a597c0a791ff8161bc9` | yes |
| 0002-inn-2108 | rom2-ru | 4 | 2 | 4 | `5acd3718734acc4fde89a963796107353870c912a58ee1cf99589b79d90577fc` | `5acd3718734acc4fde89a963796107353870c912a58ee1cf99589b79d90577fc` | yes |
| 0002-inn-517 | rom2-ru | 14 | 2 | 14 | `742d6789663f9d0047fb84058da7560923c3fa317c26ff00e8c4da9446000e41` | `742d6789663f9d0047fb84058da7560923c3fa317c26ff00e8c4da9446000e41` | yes |
| 0003-gates | rom2-ru | 2 | 2 | 3 | `a4dd655cbe35cc09ddd131f1d0e97b98572bcb193a4b37df40f77bb9237dfd24` | `a4dd655cbe35cc09ddd131f1d0e97b98572bcb193a4b37df40f77bb9237dfd24` | yes |
| 0003-inn-207 | rom2-ru | 3 | 2 | 3 | `af7579a517277ef0dfea8d15e0bfe83a0a6d42812e79a0b6e457aa3cfe0744da` | `af7579a517277ef0dfea8d15e0bfe83a0a6d42812e79a0b6e457aa3cfe0744da` | yes |
| 0003-inn-2108 | rom2-ru | 4 | 2 | 4 | `7e9a6639ac9fba8af757efa969e2ec7eeb40f263d1612255da92c6181b98804f` | `7e9a6639ac9fba8af757efa969e2ec7eeb40f263d1612255da92c6181b98804f` | yes |
| 0003-inn-517 | rom2-ru | 14 | 2 | 14 | `8f7a0c508d69fb4a1520ef50bb8442ecd0ae49c47af1df450a560b1e73c0eb00` | `8f7a0c508d69fb4a1520ef50bb8442ecd0ae49c47af1df450a560b1e73c0eb00` | yes |
| 9998-gates | rom2-ru | 1 | 1 | 1 | `ed61213403022f8cc8e760d3ac015c1d908bc5932c2dddd7064e2eb9b0fef675` | `ed61213403022f8cc8e760d3ac015c1d908bc5932c2dddd7064e2eb9b0fef675` | yes |
| 9998-inn-207 | rom2-ru | 3 | 2 | 3 | `63469aa0d25da523cab1209530fa02d1c33481d0fd0eb693999d1f973e735f44` | `63469aa0d25da523cab1209530fa02d1c33481d0fd0eb693999d1f973e735f44` | yes |
| 9998-inn-2108 | rom2-ru | 4 | 2 | 4 | `189ea8880092969f4aadb278b06613a71f04b2483d6e93b4d98fb435af6cdaa9` | `189ea8880092969f4aadb278b06613a71f04b2483d6e93b4d98fb435af6cdaa9` | yes |
| 9998-inn-517 | rom2-ru | 14 | 2 | 14 | `84d8ffc951b1483b6ac293df80021f4b33a27da3c6df7e0ad4fdc0bb73bc14f1` | `84d8ffc951b1483b6ac293df80021f4b33a27da3c6df7e0ad4fdc0bb73bc14f1` | yes |
| new-gates | rom2-ru | 2 | 1 | 3 | `15b8ee4887a66902382194672d7728c35a2541017fc49427f070e23d3f73fca1` | `15b8ee4887a66902382194672d7728c35a2541017fc49427f070e23d3f73fca1` | yes |
| new-inn-207 | rom2-ru | 3 | 1 | 3 | `6486813278badee58c5dcb0a6f01c38c7f1e60d238bbf194efe3f8887093ecc1` | `6486813278badee58c5dcb0a6f01c38c7f1e60d238bbf194efe3f8887093ecc1` | yes |
| new-inn-2108 | rom2-ru | 4 | 1 | 4 | `fc0e42c725a27c843beddb79498a121cf0e9bf0f638c3d14b146ed768ae6ec0b` | `fc0e42c725a27c843beddb79498a121cf0e9bf0f638c3d14b146ed768ae6ec0b` | yes |
| new-inn-517 | rom2-ru | 10 | 1 | 10 | `953bb6e95cc4176000390a6ff770678bd709f086d327d10a35a00a02105e2d3f` | `953bb6e95cc4176000390a6ff770678bd709f086d327d10a35a00a02105e2d3f` | yes |
| 9711-gates | ru | 2 | 2 | 3 | `6f2be42d2fec83688fac98be73af9c70cde57439ac7e795f44c885d4b9b0414b` | `6f2be42d2fec83688fac98be73af9c70cde57439ac7e795f44c885d4b9b0414b` | yes |
| 9711-shop-entry | ru | 5 | 2 | 5 | `e435992ffe11c542ae3952e063a39ce61accd1e3b904d5b8c25b6e3ca026d89c` | `e435992ffe11c542ae3952e063a39ce61accd1e3b904d5b8c25b6e3ca026d89c` | yes |
| 9711-tavern-npc22 | ru | 6 | 2 | 6 | `af376f4e5659bfe1314ee0ba4f4050699a162f266680a4f02a5ab5db01d31311` | `af376f4e5659bfe1314ee0ba4f4050699a162f266680a4f02a5ab5db01d31311` | yes |
| 9713-tavern-npc25 | ru | 3 | 2 | 3 | `08ed1fc20bda2264c60282ce20f711a834e5efc8eaf3e405892a4d39eb4caa35` | `08ed1fc20bda2264c60282ce20f711a834e5efc8eaf3e405892a4d39eb4caa35` | yes |
| 9716-tavern-npc62 | ru | 5 | 2 | 5 | `7306a8b588e9e753aca4944a63f9c2e4603f6124e0cef5015962b358f71a129f` | `7306a8b588e9e753aca4944a63f9c2e4603f6124e0cef5015962b358f71a129f` | yes |
| 9718-tavern-npc30 | ru | 7 | 2 | 7 | `a6851d6652314e90d8153ecae7b95a42f04a1c3a6b7603217442baf48365f3a6` | `a6851d6652314e90d8153ecae7b95a42f04a1c3a6b7603217442baf48365f3a6` | yes |
| 9721-tavern-npc25 | ru | 12 | 2 | 12 | `50b691c4efee4f191c4fece917618f5891118bc84d5042ecc5b403e03a516947` | `50b691c4efee4f191c4fece917618f5891118bc84d5042ecc5b403e03a516947` | yes |

## Open debt

- Not captured: the second game's inn talk once Dina has joined, and a
  second-game inn talk without a speaker portrait. The owner saves are all
  in town 1 with a fresh campaign; every inn and gate talk there names a
  speaker, and the headless scenario has no step that types a chat line, so
  the campaign could not be walked to Dina. The face path is the one the
  captured portrait talks use (`townSpeakers` over the carried party, then
  `speakerResolver.SpeakerFace`). The release test
  `TestReleaseSecondDinaTalksDrawHerPortrait` covers Dina; the unit tests
  cover a part naming nobody in both games.
- Not captured on RU: the first game's mercenary talk. The RU tavern's Talk
  button label is not valid UTF-8 and a scenario cannot name it; the EN
  mercenary talks are equal.
- A host talk rule that `townTalkRules` does not name takes the first game's
  profile, as `boundTalkSource` chose the first game's source for any rule
  but `section`.
