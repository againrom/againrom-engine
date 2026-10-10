package ui

// The chrgen members of sfx.res the main menu, the Hall of Fame and the school
// room request, named by their archive path under sfx/ (VIDEO-SFX-059,
// VIDEO-SFX-060). The character generator reads its own sounds from its
// description.
const ChargenSoundOK = "chrgen/ok.wav"

// ChargenSkillSounds is the member a skill slot requests, fighter row then
// mage row; index 0..4 is stored skill slot 1..5 (VIDEO-SFX-059). The ten
// live in chrgen's skill directory, as the shipped literal
// SFX\ChrGen\Skill\MAstral.wav in RES-CASE-036 spells.
var ChargenSkillSounds = [2][5]string{
	{"chrgen/skill/fsword.wav", "chrgen/skill/faxe.wav", "chrgen/skill/fclub.wav", "chrgen/skill/fpike.wav", "chrgen/skill/fbow.wav"},
	{"chrgen/skill/mfire.wav", "chrgen/skill/mwater.wav", "chrgen/skill/mair.wav", "chrgen/skill/mearth.wav", "chrgen/skill/mastral.wav"},
}
