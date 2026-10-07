# Mod companion join condition

## Intent and authority

A mod declares that a named town companion joins the party only when a named tavern conversation opens. Authority is owner direction on tester item 18: with the mod, Reniesta joins after the hero talks to her in the Plagat tavern and not on arrival in the town, as in Allods 2. The game without the mod is unchanged. The data seam is the one of `docs/1272/story.md`, `docs/1273/story.md` and `docs/1277/story.md`. ROM1 differences are rows `DIV-1912` to `DIV-1914` in `docs/divergences/mods.md`. `DIV-1915` to `DIV-1917` are returned unused.

## As built

Declaration form. `data/companions.toml`, loaded by `game.data.add("data/companions.toml")` in the mod's `init` (`pkg/mod/companions.go`, `pkg/modrt/data.go`). `[[join]]` tables, every key required:

| key | meaning |
|---|---|
| `key` | identifier, `a-z0-9_`, unique in the file |
| `companion` | the registry's `AddHero` value the town grants; 22 is the only town grant |
| `chapter` | the town chapter that grants it |
| `building` | `tavern`; the only building wired |
| `talk` | the `InnNPC` record of one of that chapter's tavern conversations |

Refusals name the mod, file and line: an unknown table or key, a value that is not an integer in 1 to 65535, a building other than `tavern`, a duplicate key, a second condition for one companion and chapter, and (from `SetModCompanions`, which reads the install's campaign) a chapter the campaign lacks, a companion other than the town grant, a grant the chapter does not hold and a conversation record the chapter's tavern does not hold. The conditions reach the game through `FrontEnd.SetModCompanions`, stored in `mapload.ModContext.Companions`; `FrontEnd` gained no field. `-check` prints `againrom: companion joins=N` when a mod declares conditions.

Join path. A town grants its companion on arrival through `addChapterCompanions` (win flow, cold LOAD of a town, the city SAVE copy). The grant is the chapter record's `AddHero` array, consumed with the companion's creation (`REG-SCN-098`, `SAV-CAMPAIGN-084`). `addChapterCompanions` now asks `holdsCompanion` and leaves a held grant in the array (`Town.takeAddHeroesExcept`, native and restored-campaign forms). The creation body is `carryTownCompanion`, unchanged. When a tavern conversation opens (`townScreen.openOfferDialogue`), `joinOnTalk` takes the held grant of the current chapter whose `talk` equals the cell's `InnNPC`, creates the companion with `carryTownCompanion`, settles the city groups and rebuilds the tavern cast before the dialogue draws (`DIV-1912`, `DIV-1914`). The party, the shop and school character switch, the roster and every save read the live party, so a held companion is absent from all of them.

Example mod (`pkg/modrt/testdata/mods/reniesta-joins-on-talk`, test data): holds the chapter-30 grant (companion 22, the mage Reniesta for a male starting hero) until the tavern conversation with record 22 opens. The seat copies it into `builds/current/mods/`.

SAV. The held state is the chapter record's `AddHero` array still holding 22 with no companion in the Player's groups; the joined state is the array cleared and the companion present, as for any joined companion. No leaf is added; the mod mark records the condition (`DIV-1913`).

## Proof

- Unit: `pkg/mod` (the file, each refusal with its line), `pkg/modrt` (the example mod, refusals naming mod, file and line, two mods holding one companion), `pkg/game` (`SetModCompanions` refusals against a fixture campaign, `takeAddHeroesExcept` and `takeAddHero` on the native and restored-campaign towns, `SetMods` keeping the conditions), `cmd/againrom` (the launch refusal, the `-check` line).
- Release, EN and RU, one root at a time (`pkg/game/modjoin_release_test.go`, gated by `AGAINROM_ASSETS`, `TestReleaseModJoinWaitsForTheTavernTalk`): chargen, mission 10, mission 20 through the production completion path.
  - Control without the mod: Reniesta is in the party on arrival, the SAV's `AddHero` is empty and a cold LOAD keeps her.
  - With the mod: the party is the hero alone, the picker shows one name and the character switch answers that nobody else is with the hero, in the tavern and the shop. The SAV holds `AddHero` [22]; a game without the mod refuses that SAV; the cold LOAD keeps her absent, mission 30 is not at the gates and a second SAV holds the same array.
  - The talk: pressing her tavern cell through the App adds her as the second member; the shop picker then shows two names; mission 30 is at the gates once the tavern is left; the SAV after holds an empty array; the cold LOAD restores both members with equal names, identities, companion record, worn sets and packs; hearing her on the loaded game adds nobody.
  - Screenshots, never skipped, written to `AGAINROM_SHOT_DIR` or a temporary directory: `join-tavern-before-talk`, `join-shop-before-talk`, `join-tavern-talk`, `join-tavern-after-talk`, `join-shop-after-talk`, each `-en.png` and `-ru.png`, copied to `review/story1280-mod-join/`.

## Open debt

- Only tavern conversations start a join; shop and school conversations, and the companions handed over on a map (mission 40, 70, 100, 140), are not wired.
- The held state is read from the `AddHero` array by this build; the original's LOAD of a save in that state was not observed (`DIV-1913`).
- A held companion's tavern cell is drawn as a speaker no party member answers for until she joins.
- Mission 30 reaches the gates only through the tavern conversation (asserted in the witness), so a held grant cannot be skipped by play; a save edited to open it without the conversation would run without her, and no rule joins her at mission entry.
