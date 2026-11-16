package main

import "againrom/pkg/launch"

// optionView is how the starter keeps and shows one row of the launch
// settings table: the settings field holding its value, the control that
// edits it and its captions.
type optionView struct {
	key   string
	value func(*settings) any // *bool for a Switch, else *string
	edit  fieldID             // the field of a Number, Text or Arguments row
	act   actionKind          // the button of a Switch, Override or Choice row
	label captionID
	// bounds marks a Number whose label takes Min and Max and whose refusal
	// takes them after the value.
	bounds bool
	// refusal is a value the game would refuse (a Number's takes the value);
	// clash is the value set while the row it excludes is on.
	refusal, clash captionID
	values         []captionID // each option's caption; empty shows the value
}

// optionViews has one view for every row of launch.Settings.
var optionViews = []optionView{
	{key: launch.Sound.Key, value: func(s *settings) any { return &s.Sound }, act: aSound, label: cSound,
		values: []captionID{cValueDefault, cOn, cOff}},
	{key: launch.Volume.Key, value: func(s *settings) any { return &s.Volume }, edit: fVolume, label: cVolume, refusal: cBadVolume},
	{key: launch.Movies.Key, value: func(s *settings) any { return &s.Movies }, act: aMovies, label: cMovies},
	{key: launch.Video.Key, value: func(s *settings) any { return &s.Video }, act: aVideo, label: cMissionMovies,
		values: []captionID{cValueNormal, "", ""}},
	{key: launch.Markers.Key, value: func(s *settings) any { return &s.Markers }, act: aMarkers, label: cMarkers},
	{key: launch.Saves.Key, value: func(s *settings) any { return &s.Saves }, edit: fSaves, label: cSavesFolder},
	{key: launch.Window.Key, value: func(s *settings) any { return &s.Windowed }, act: aWindowed, label: cWindowed},
	{key: launch.Mission.Key, value: func(s *settings) any { return &s.Mission }, edit: fMission, label: cMissionNumber,
		refusal: cBadMission, clash: cPickerMission},
	{key: launch.Picker.Key, value: func(s *settings) any { return &s.Picker }, act: aPicker, label: cMapPicker},
	{key: launch.Chicken.Key, value: func(s *settings) any { return &s.Chicken }, act: aChicken, label: cChicken},
	{key: launch.SmoothRegen.Key, value: func(s *settings) any { return &s.SmoothRegen }, act: aSmoothRegen, label: cSmoothRegen},
	{key: launch.Skill.Key, value: func(s *settings) any { return &s.Skill }, edit: fSkill, label: cSkill},
	{key: launch.ShopStockMultiplier.Key, value: func(s *settings) any { return &s.ShopStockMultiplier }, edit: fShopStockMultiplier,
		label: cShopStockMultiplier, bounds: true, refusal: cBadShopStockMultiplier},
	{key: launch.Extra.Key, value: func(s *settings) any { return &s.Extra }, edit: fExtra, label: cExtraArgs, refusal: cExtraQuote},
}

// settingsRowKind is what one line of the settings view shows.
type settingsRowKind int

const (
	rowSetting settingsRowKind = iota
	rowHeading
	rowCloseOnPlay
	rowGameProgram
	rowAcceptUnmarked
)

// settingsRow is one line of the settings view. Its row is its index in
// settingsRows: the line's top is 66+24*row.
type settingsRow struct {
	kind settingsRowKind
	key  string    // the launch setting of a rowSetting
	head captionID // the caption of a rowHeading
	dy   int       // a heading's offset below its row's top
}

// settingsRows is the settings view in draw order.
var settingsRows = []settingsRow{
	{kind: rowSetting, key: launch.Sound.Key},
	{kind: rowSetting, key: launch.Volume.Key},
	{kind: rowSetting, key: launch.Movies.Key},
	{kind: rowSetting, key: launch.Video.Key},
	{kind: rowSetting, key: launch.Markers.Key},
	{kind: rowSetting, key: launch.Saves.Key},
	{kind: rowSetting, key: launch.Window.Key},
	{kind: rowHeading, head: cHeadAdvanced, dy: 4},
	{kind: rowSetting, key: launch.Mission.Key},
	{kind: rowSetting, key: launch.Picker.Key},
	{kind: rowSetting, key: launch.Chicken.Key},
	{kind: rowSetting, key: launch.SmoothRegen.Key},
	{kind: rowSetting, key: launch.Skill.Key},
	{kind: rowSetting, key: launch.ShopStockMultiplier.Key},
	{kind: rowSetting, key: launch.Extra.Key},
	{kind: rowHeading, head: cHeadStarter, dy: 2},
	{kind: rowCloseOnPlay},
	{kind: rowGameProgram},
	{kind: rowAcceptUnmarked},
}

// stepOption flips the Switch or steps the Override or Choice that the button
// act edits. It reports whether act is one of those buttons.
func (a *app) stepOption(act actionKind) bool {
	for _, v := range optionViews {
		if act == aNone || v.act != act {
			continue
		}
		st, _ := launch.ByKey(v.key)
		if st.Kind == launch.Switch {
			p := v.value(&a.s).(*bool)
			*p = !*p
			return true
		}
		a.s.setOption(st, st.Next(a.s.option(st)))
		return true
	}
	return false
}

// viewOf is the view of the row with key.
func viewOf(key string) optionView {
	for _, v := range optionViews {
		if v.key == key {
			return v
		}
	}
	panic("starter: no view of launch setting " + key)
}

// option is the text of the row st in s: a Switch as the settings file
// spells it.
func (s *settings) option(st launch.Setting) string {
	switch p := viewOf(st.Key).value(s).(type) {
	case *bool:
		return launch.FormatBool(*p)
	case *string:
		return *p
	}
	return ""
}

// setOption sets the row st in s to the text v.
func (s *settings) setOption(st launch.Setting, v string) {
	switch p := viewOf(st.Key).value(s).(type) {
	case *bool:
		*p = launch.ParseBool(v, false)
	case *string:
		*p = v
	}
}

// labelText is the label of the row st.
func (a *app) labelText(st launch.Setting, v optionView) string {
	if v.bounds {
		return a.t(v.label, st.Min, st.Max)
	}
	return a.t(v.label)
}

// refusalErr is the starter's caption for a value the launch table refuses.
func refusalErr(err error) error {
	r, ok := err.(*launch.Refusal)
	if !ok {
		return err
	}
	v := viewOf(r.Setting.Key)
	if r.Clash {
		return captionErr(v.clash)
	}
	if r.Setting.Kind != launch.Number {
		return captionErr(v.refusal)
	}
	if v.bounds {
		return captionErr(v.refusal, r.Value, r.Setting.Min, r.Setting.Max)
	}
	return captionErr(v.refusal, r.Value)
}
