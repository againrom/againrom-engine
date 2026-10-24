# Tavern talks and quests as town data in a mod

Status: design; not built. The game is unchanged by this document.

## Intent

A town's tavern talks and quests become rows of that town in the town catalog's entry format.
A mod's `data/towns.toml` adds a row, changes one by its key or switches one off through
`mod.Merge`, as it does for towns. The base first-game town keeps the rows `scenario.reg` gives
it today.

## Authority

- Owner direction: BACKLOG T8.
- REG-SCN-064: the tavern's `InnNPC`/`InnMission` arrays pair by position; `InnMission` 0 is a
  talk with nothing to give.
- TAVERN-023, TAVERN-024: heard missions queue without identity comparison and register in
  order; registration removes the first matching offer and sets one announce flag.
- SAV-CAMPAIGN-081, SAV-1112: an accepted pair leaves both live arrays; a talk cell follows its
  array position.

## Design

- `town.Entry` gains tavern rows: chapter, NPC record, mission. The shipped catalog states no
  rows; layer 0 fills them from `Campaign.Chapters` (the one `scenario.reg` reader), walking the
  shorter array, each row in the town the present `offeredHere` rule shows it in.
- The tavern offer list, the chapter load of the live arrays, the native SAV producer, the
  offer labels and the companion join read one accessor over the merged rows instead of
  `Chapter.InnNPC`/`Inn`. With no mod it answers the same arrays.
- Row key: town, chapter and NPC record. Settable: `mission`, `off`.
- No new SAV field: a mod row is a pair in the live arrays and its mission a campaign record;
  the SAV mod mark names the mod and LOAD without it refuses.

## Open debt

- Uniqueness of the NPC record within one chapter's tavern on EN and RU is not yet measured.
- A row whose talk leaf `main/text/inn/npc/npcNNmM.txt` the install lacks has no text source;
  the owner chooses one (a mod text field, new install-file paths, or install leaves only).
- A save without a mod mark is refused while a mod is active unless `-mods-accept-unmarked`.
- A save loaded mid-chapter keeps its live arrays; a mod row of that chapter shows from the
  next chapter.
