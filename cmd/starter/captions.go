package main

import (
	"errors"
	"fmt"

	"againrom/pkg/locale"
)

// captionID names one caption the starter itself prints in its window.
type captionID string

// Every caption of the starter, in the order the window shows them. Mod
// setting labels are the mod's own and are not here.
const (
	cWindowTitle  captionID = "window-title"
	cTitle        captionID = "title"
	cLaunches     captionID = "launches"
	cSettingsFile captionID = "settings-file"
	cUnknown      captionID = "unknown"
	cText         captionID = "text"

	cHeadBase        captionID = "head-base"
	cHeadMods        captionID = "head-mods"
	cHeadParams      captionID = "head-params"
	cHeadAdvanced    captionID = "head-advanced"
	cHeadStarter     captionID = "head-starter"
	cHeadModSettings captionID = "head-mod-settings"

	cBaseOK             captionID = "base-ok"
	cBaseBad            captionID = "base-bad"
	cNoBase             captionID = "no-base"
	cAddPath            captionID = "add-path"
	cAdd                captionID = "add"
	cBrowse             captionID = "browse"
	cChooseBaseFolder   captionID = "choose-base-folder"
	cFolderDialogFailed captionID = "folder-dialog-failed"
	cRemove             captionID = "remove"
	cFolder             captionID = "folder"
	cRescan             captionID = "rescan"
	cModsDefault        captionID = "mods-default"
	cModsReadError      captionID = "mods-read-error"
	cNoMods             captionID = "no-mods"
	cTickHint           captionID = "tick-hint"
	cModsFound          captionID = "mods-found"

	cSound                  captionID = "sound"
	cVolume                 captionID = "volume"
	cMovies                 captionID = "movies"
	cMissionMovies          captionID = "mission-movies"
	cMarkers                captionID = "markers"
	cSavesFolder            captionID = "saves-folder"
	cWindowed               captionID = "windowed"
	cMissionNumber          captionID = "mission-number"
	cMapPicker              captionID = "map-picker"
	cChicken                captionID = "chicken"
	cSkill                  captionID = "skill"
	cShopStockMultiplier    captionID = "shop-stock-multiplier"
	cBadShopStockMultiplier captionID = "bad-shop-stock-multiplier"
	cExtraArgs              captionID = "extra-args"
	cCloseAfterPlay         captionID = "close-after-play"
	cGameProgram            captionID = "game-program"
	cLoadUnmarked           captionID = "load-unmarked"
	cOn                     captionID = "on"
	cOff                    captionID = "off"
	cValueDefault           captionID = "value-default"
	cValueNormal            captionID = "value-normal"

	cSelectMod     captionID = "select-mod"
	cModState      captionID = "mod-state"
	cModNotEnabled captionID = "mod-not-enabled"
	cModEnabled    captionID = "mod-enabled"
	cSettingsToml  captionID = "settings-toml"
	cNoModSettings captionID = "no-mod-settings"
	cDefaults      captionID = "defaults"
	cMoreSettings  captionID = "more-settings"

	cPlay         captionID = "play"
	cCheck        captionID = "check"
	cSaveSettings captionID = "save-settings"
	cMapEditor    captionID = "map-editor"
	cSettings     captionID = "settings"
	cClose        captionID = "close"

	cReadSettingsFailed captionID = "read-settings-failed"
	cSettingsLine       captionID = "settings-line"
	cFolderNotWritable  captionID = "folder-not-writable"
	cModNoFolder        captionID = "mod-no-folder"
	cModError           captionID = "mod-error"
	cModFolderMissing   captionID = "mod-folder-missing"
	cProblemMissing     captionID = "problem-folder-missing"
	cModCannotEnable    captionID = "mod-cannot-enable"
	cTypeFolder         captionID = "type-folder"
	cNotFolder          captionID = "not-folder"
	cAdded              captionID = "added"
	cAddedNotInstall    captionID = "added-not-install"
	cSaveFailed         captionID = "save-failed"
	cSaved              captionID = "saved"
	cNoBaseSelected     captionID = "no-base-selected"
	cBaseNotInstall     captionID = "base-not-install"
	cModUnusable        captionID = "mod-unusable"
	cGameNotFound       captionID = "game-not-found"
	cCannotPlay         captionID = "cannot-play"
	cCannotStart        captionID = "cannot-start"
	cStarted            captionID = "started"
	cCannotCheck        captionID = "cannot-check"
	cRunningCheck       captionID = "running-check"
	cEditorNotFound     captionID = "editor-not-found"
	cCannotOpenEditor   captionID = "cannot-open-editor"
	cCheckingEditor     captionID = "checking-editor"
	cEditorCannotOpen   captionID = "editor-cannot-open"
	cCheckFailed        captionID = "check-failed"
	cCheckPassed        captionID = "check-passed"

	cUnrecognised  captionID = "unrecognised-build"
	cHasLimits     captionID = "has-limits"
	cInstallOf     captionID = "install-of"
	cNoCharGen     captionID = "no-chargen"
	cSavesRefused  captionID = "saves-refused"
	cBadVolume     captionID = "bad-volume"
	cPickerMission captionID = "picker-mission"
	cBadMission    captionID = "bad-mission"
	cExtraQuote    captionID = "extra-quote"
)

// captionRow is one caption in every language the starter has it in. source
// names the installed game text a Russian word was taken from (file and the
// English entry it translates), or is empty when the word is the starter's own.
type captionRow struct {
	id     captionID
	by     map[string]string
	source string
}

func row(id captionID, en, ru, source string) captionRow {
	return captionRow{id: id, by: map[string]string{locale.Fallback: en, "ru": ru}, source: source}
}

// captionRows is the one caption table. A format caption keeps the same verbs
// in the same order in every language.
var captionRows = []captionRow{
	row(cWindowTitle, "Againrom starter %s", "Стартер Againrom %s", ""),
	row(cTitle, "Starter %s (%s)", "Стартер %s (%s)", ""),
	row(cLaunches, "launches againrom %s", "запускает againrom %s", ""),
	row(cSettingsFile, "settings: %s", "установки: %s", "dialogs.txt: Reset settings"),
	row(cUnknown, "unknown", "неизвестно", ""),
	row(cText, "%s", "%s", ""),

	row(cHeadBase, "Base game", "Основная игра", ""),
	row(cHeadMods, "Mods", "Моды", ""),
	row(cHeadParams, "Parameters", "Параметры", "dialogs.txt: Modem Settings"),
	row(cHeadAdvanced, "Advanced", "Дополнительно", ""),
	row(cHeadStarter, "Starter", "Стартер", ""),
	row(cHeadModSettings, "Mod settings", "Установки мода", "dialogs.txt: Reset settings"),

	row(cBaseOK, "%s  %s  OK %s", "%s  %s  OK %s", ""),
	row(cBaseBad, "%s  %s  not an install, missing %s", "%s  %s  не копия игры, нет файлов: %s", ""),
	row(cNoBase, "No base game yet. Type or paste a folder below.", "Игры пока нет. Введите или вставьте папку ниже.", ""),
	row(cAddPath, "Add path:", "Путь:", ""),
	row(cAdd, "Add", "Внести", ""),
	row(cBrowse, "Browse", "Обзор", ""),
	row(cChooseBaseFolder, "Choose the base game folder", "Выберите папку основной игры", ""),
	row(cFolderDialogFailed, "folder dialog unavailable: %v; type or paste the folder in Add path", "выбор папки недоступен: %v; введите или вставьте папку в поле пути", ""),
	row(cRemove, "Remove", "Удалить", "dialogs.txt: Delete"),
	row(cFolder, "Folder:", "Папка:", ""),
	row(cRescan, "Rescan", "Обновить", ""),
	row(cModsDefault, "(empty: %s)", "(пусто: %s)", ""),
	row(cModsReadError, "cannot read the mods folder: %s", "не удалось прочитать папку модов: %s", ""),
	row(cNoMods, "No mods found.", "Модов не найдено.", ""),
	row(cTickHint, "Tick to enable. Load order is the order shown.", "Отметьте мод, чтобы включить. Порядок загрузки - как в списке.", ""),

	row(cModsFound, "%d mod folder(s) found in %s", "Найдено папок с модами: %d, в %s", ""),

	row(cSound, "Sound", "Звук", ""),
	row(cVolume, "Volume 0-100", "Громкость 0-100", "dialogs.txt: Music volume"),
	row(cMovies, "Movies", "Видеоролики", "dialogs.txt: View Cutscenes"),
	row(cMissionMovies, "Mission movies", "Видеоролики миссий", "dialogs.txt: View Cutscenes; main.txt: Mission Completed"),
	row(cMarkers, "Markers", "Метки", ""),
	row(cSavesFolder, "Saves folder", "Папка сохранений", "dialogs.txt: Saved Games"),
	row(cWindowed, "In a window", "В окне", ""),
	row(cMissionNumber, "Mission number", "Номер миссии", "dialogs.txt: Enter Number & Name; main.txt: Mission Completed"),
	row(cMapPicker, "Map picker", "Выбор карты", "dialogs.txt: Select map to play"),
	row(cChicken, "#Chicken each mission", "#Chicken за миссию", ""),
	row(cSkill, "Skill", "Навык", "main.txt: Skills"),
	row(cShopStockMultiplier, "Shop stock multiplier (%d..%d, ROM1)", "Множитель товаров (%d..%d, ROM1)", ""),
	row(cBadShopStockMultiplier, "shop stock multiplier %q must be a whole number from %d to %d", "множитель товаров %q должен быть целым числом от %d до %d", ""),
	row(cExtraArgs, "Extra arguments", "Доп. аргументы", ""),
	row(cCloseAfterPlay, "Close after Play", "Закрыть после запуска", "main.txt: Close"),
	row(cGameProgram, "Game program", "Программа игры", ""),
	row(cLoadUnmarked, "Load unmarked saves", "Сохранения без метки", ""),
	row(cOn, "on", "Вкл", "dialogs.txt: On"),
	row(cOff, "off", "Выкл", "dialogs.txt: Off"),
	row(cValueDefault, "default", "по умолчанию", ""),
	row(cValueNormal, "normal", "обычное", ""),

	row(cSelectMod, "Select a mod in the list to edit its settings.", "Выберите мод в списке, чтобы изменить его установки.", ""),
	row(cModState, "%s (%s)", "%s (%s)", ""),
	row(cModNotEnabled, "not enabled", "не включен", ""),
	row(cModEnabled, "enabled", "включен", ""),
	row(cSettingsToml, "settings.toml: %s", "settings.toml: %s", ""),
	row(cNoModSettings, "This mod has no settings.", "У этого мода нет установок.", ""),
	row(cDefaults, "Defaults", "Сбросить", "dialogs.txt: Reset settings"),
	row(cMoreSettings, "%d more setting(s) are not shown.", "Не показано установок: %d.", ""),

	row(cPlay, "Play", "Играть", ""),
	row(cCheck, "Check", "Проверить", ""),
	row(cSaveSettings, "Save settings", "Сохранить установки", "dialogs.txt: Save Game, Reset settings"),
	row(cMapEditor, "Map editor", "Редактор карт", ""),
	row(cSettings, "Settings", "Настройки", ""),
	row(cClose, "Close", "Закрыть", "main.txt: Close"),

	row(cReadSettingsFailed, "cannot read settings %s: %v", "не удалось прочитать установки %s: %v", ""),
	row(cSettingsLine, "settings %s: %s (%d such line(s); the lines are kept as they are)", "установки %s: %s (таких строк: %d; строки оставлены как есть)", ""),
	row(cFolderNotWritable, "settings folder is not writable, Save settings will fail: %v", "папка установок закрыта для записи, \"Сохранить установки\" не сработает: %v", ""),
	row(cModNoFolder, "mod %s has no usable folder", "у мода %s нет пригодной папки", ""),
	row(cModError, "mod %s: %v", "мод %s: %v", ""),
	row(cModFolderMissing, "%s: folder not found in the mods directory", "%s: папка не найдена среди модов", ""),
	row(cProblemMissing, "folder not found", "папка не найдена", ""),
	row(cModCannotEnable, "mod %s cannot be enabled: %s", "мод %s нельзя включить: %s", ""),
	row(cTypeFolder, "type or paste the folder of a game install first", "сначала введите или вставьте папку установленной игры", ""),
	row(cNotFolder, "%s is not a folder", "%s - не папка", ""),
	row(cAdded, "added %s: %s", "добавлено %s: %s", ""),
	row(cAddedNotInstall, "added %s, but it is not an install: missing %s", "добавлено %s, но это не копия игры: нет файлов: %s", ""),
	row(cSaveFailed, "cannot save settings: %v", "не удалось сохранить установки: %v", "dialogs.txt: Save Game, Reset settings"),
	row(cSaved, "settings saved to %s", "установки сохранены в %s", "dialogs.txt: Reset settings"),
	row(cNoBaseSelected, "no base game selected", "основная игра не выбрана", ""),
	row(cBaseNotInstall, "base %s is not an install: missing %s", "%s - не копия игры, нет файлов: %s", ""),
	row(cModUnusable, "mod %s is enabled but cannot be used: %s", "мод %s включен, но непригоден: %s", ""),
	row(cGameNotFound, "game program not found: %s", "программа игры не найдена: %s", ""),
	row(cCannotPlay, "cannot play: %v", "не удалось начать игру: %v", ""),
	row(cCannotStart, "cannot start %s: %v", "не удалось запустить %s: %v", ""),
	row(cStarted, "started %s", "запущено: %s", ""),
	row(cCannotCheck, "cannot check: %v", "не удалось проверить: %v", ""),
	row(cRunningCheck, "running %s -check ...", "выполняется %s -check ...", ""),
	row(cEditorNotFound, "map editor not found: %s", "редактор карт не найден: %s", ""),
	row(cCannotOpenEditor, "cannot open the map editor: %v", "не удалось открыть редактор карт: %v", ""),
	row(cCheckingEditor, "checking %s on %s ...", "проверка %s на %s ...", ""),
	row(cEditorCannotOpen, "the map editor cannot open this base: %v", "редактор карт не может открыть эту игру: %v", ""),
	row(cCheckFailed, "check failed: %v", "проверка не пройдена: %v", ""),
	row(cCheckPassed, "check passed", "проверка пройдена", ""),

	row(cUnrecognised, " (unrecognised build)", " (неизвестная сборка)", ""),
	row(cHasLimits, ", has limits", ", есть ограничения", ""),
	row(cInstallOf, "%s install", "копия игры: %s", ""),
	row(cNoCharGen, "no character generation, new game opens mission %d", "нет создания персонажа, новая игра открывает миссию %d", "main.txt: New Character, Mission Completed"),
	row(cSavesRefused, "original saves refused", "оригинальные сохранения не принимаются", "dialogs.txt: Saved Games"),
	row(cBadVolume, "volume %q is not a number from 0 to 100", "громкость %q - не число от 0 до 100", "dialogs.txt: Music volume"),
	row(cPickerMission, "picker and mission cannot both be set", "выбор карты и номер миссии нельзя задать вместе", ""),
	row(cBadMission, "mission %q is not a positive number", "миссия %q - не положительное число", ""),
	row(cExtraQuote, "extra arguments have an unmatched quote", "в дополнительных аргументах не закрыта кавычка", ""),
}

var captionIndex = func() map[captionID]captionRow {
	m := make(map[captionID]captionRow, len(captionRows))
	for _, r := range captionRows {
		m[r.id] = r
	}
	return m
}()

// caption is the caption id in lang, formatted with args, else in the
// reference language.
func caption(lang string, id captionID, args ...any) string {
	r, ok := captionIndex[id]
	if !ok {
		panic("starter: no caption " + string(id))
	}
	s, ok := r.by[lang]
	if !ok {
		s = r.by[locale.Fallback]
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

// captionError is an error whose text is a caption, so the window can show it
// in the language of the selected base.
type captionError struct {
	id   captionID
	args []any
}

func (e *captionError) Error() string { return caption(locale.Fallback, e.id, e.args...) }

func captionErr(id captionID, args ...any) error { return &captionError{id, args} }

// t is a caption in the language of the selected base.
func (a *app) t(id captionID, args ...any) string { return caption(a.lang(), id, args...) }

// errText is err as the window shows it: a caption error in the language of
// the selected base, any other error as it reads.
func (a *app) errText(err error) string {
	var ce *captionError
	if errors.As(err, &ce) {
		return caption(a.lang(), ce.id, ce.args...)
	}
	return err.Error()
}
