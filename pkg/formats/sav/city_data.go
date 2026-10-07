package sav

// CityData is the versioned, acyclic persistence DTO for a semantic no-world
// city. Objects use one-based table indices (zero means nil), not addresses,
// archive tags or source offsets. Named opaque class members retain their
// existing bounded semantics; no whole SAV/body/store/campaign blob is carried.
// It deliberately implements no GobEncoder/GobDecoder: the native envelope's
// generic count/depth preflight must see every nested allocation.
type CityData struct {
	Version              uint32
	FileVersion          uint32
	Counter04, Counter00 uint32
	MapName              string
	Head                 [13]uint32
	PlayerList           uint32
	Players, DeadActors  []uint16
	Marker, GlobalDWord  uint32
	TrailerState         []byte
	Objects              []CityObjectData
	State                CityStateData
	Campaign             CityCampaignData
}

type CityObjectData struct {
	Class  string
	Player *CityPlayerData
	Unit   *CityUnitData
	Item   *CityItemData
	Effect *CityEffectData
	Spell  *CitySpellData
	Diary  *CityDiaryData
}

type CityStateData struct {
	RootKind uint32
	// Version 1 compatibility only. New saves never emit maps: gob map order
	// is not stable. Keeping the names/types lets existing AGS decode safely.
	Directories map[string]uint32
	Values      map[string]CityStateValueData
	// Version 2 uses strictly path-sorted records. They remain ordinary gob
	// slices so generic wire preflight sees all counts and nested byte arrays.
	DirectoryRecords []CityStateDirectoryData
	ValueRecords     []CityStateRecordData
}
type CityStateDirectoryData struct {
	Path string
	Kind uint32
}
type CityStateRecordData struct {
	Path  string
	Value CityStateValueData
}
type CityStateValueData struct {
	Kind  uint32
	Int32 int32
	Bytes []byte
}

type CityDiaryData struct {
	DWords    []uint32
	Words     []uint16
	Reference uint32
}

type CityGroupData struct {
	Words20 []uint16
	Raw80   []byte
	Words3c []uint16
	Actors  []uint16
	F1c     uint32
	F40     uint32
	F44     uint32
}

type CityPlayerData struct {
	Name   string
	Fixed  []byte
	Groups []CityGroupData
	Raw32  []byte
	Diary  CityDiaryData
}

type CityUnitData struct {
	Token          []byte
	Effects        []uint16
	Words15c       []uint16
	Words178       []uint16
	RawA6          []byte
	RawBE          []byte
	Raw114         []byte
	RawD4          []byte
	Raw154         []byte
	Raw158         []byte
	Words158       []uint16
	Scalar1        []byte
	Reference74    uint16
	Reference78    uint16
	Name           string
	Scalar2        []byte
	Reference68    uint16
	ContainerFlag  byte
	Container      []uint16
	ContainerTails [2]uint32
	SpellbookFlag  byte
	SpellbookDWord uint32
	SpellbookCount uint32
	Spells         []uint16
	ScalarTail     []byte
	XP             []byte
	Equipment      []uint16
}

type CityItemData struct {
	Token       []byte
	Effects     []uint16
	Fields      []byte
	Derived     []byte
	WeaponExtra uint16
}

type CityEffectData struct {
	Token  []byte
	Fields []byte
}

type CitySpellData struct {
	Fields []byte
}

type CityCampaignBaseData struct {
	DWords [6]uint32
	Arrays [2][]uint16
}

type CityCampaignChildData struct {
	Base CityCampaignBaseData
	Age  uint32
}

type CityCampaignMarkerData struct {
	Value uint32
	Text  []byte
	Tail  [8]byte
}

type CityCampaignData struct {
	Base      CityCampaignBaseData
	Children  []CityCampaignChildData
	Parallel  [2][]uint16
	DWords    []uint32
	Arrays    [6][]uint16
	Documents [][2]uint32
	// Carriers are the document pairs that carry the Againrom-only payload.
	Carriers [][2]uint32
	Scalars  [7]uint32
	Markers  []CityCampaignMarkerData
}

func cityDataSlice[A, B any](v []A, f func(A) B) []B {
	if v == nil {
		return nil
	}
	out := make([]B, len(v))
	for i := range v {
		out[i] = f(v[i])
	}
	return out
}

func cityDataCopy[T any](v []T) []T { return append([]T(nil), v...) }
func cityDataArrays2(v [2][]uint16) [2][]uint16 {
	return [2][]uint16{cityDataCopy(v[0]), cityDataCopy(v[1])}
}
func cityDataArrays6(v [6][]uint16) (out [6][]uint16) {
	for i := range v {
		out[i] = cityDataCopy(v[i])
	}
	return
}

func cityToDataDiary(v cityDiary, ref func(*cityObject) uint16) CityDiaryData {
	return CityDiaryData{
		DWords:    cityDataCopy(v.dwords),
		Words:     cityDataCopy(v.words),
		Reference: v.reference,
	}
}

func cityFromDataDiary(v CityDiaryData, ref func(uint16) *cityObject) cityDiary {
	return cityDiary{
		dwords:    cityDataCopy(v.DWords),
		words:     cityDataCopy(v.Words),
		reference: v.Reference,
	}
}

func cityToDataGroup(v cityGroup, ref func(*cityObject) uint16) CityGroupData {
	return CityGroupData{
		Words20: cityDataCopy(v.words20),
		Raw80:   cityDataCopy(v.raw80),
		Words3c: cityDataCopy(v.words3c),
		Actors:  cityDataSlice(v.actors, ref),
		F1c:     v.f1c,
		F40:     v.f40,
		F44:     v.f44,
	}
}

func cityFromDataGroup(v CityGroupData, ref func(uint16) *cityObject) cityGroup {
	return cityGroup{
		words20: cityDataCopy(v.Words20),
		raw80:   cityDataCopy(v.Raw80),
		words3c: cityDataCopy(v.Words3c),
		actors:  cityDataSlice(v.Actors, ref),
		f1c:     v.F1c,
		f40:     v.F40,
		f44:     v.F44,
	}
}

func cityToDataPlayer(v cityPlayer, ref func(*cityObject) uint16) CityPlayerData {
	return CityPlayerData{
		Name:   v.name,
		Fixed:  cityDataCopy(v.fixed),
		Groups: cityDataSlice(v.groups, func(x cityGroup) CityGroupData { return cityToDataGroup(x, ref) }),
		Raw32:  cityDataCopy(v.raw32),
		Diary:  cityToDataDiary(v.diary, ref),
	}
}

func cityFromDataPlayer(v CityPlayerData, ref func(uint16) *cityObject) cityPlayer {
	return cityPlayer{
		name:   v.Name,
		fixed:  cityDataCopy(v.Fixed),
		groups: cityDataSlice(v.Groups, func(x CityGroupData) cityGroup { return cityFromDataGroup(x, ref) }),
		raw32:  cityDataCopy(v.Raw32),
		diary:  cityFromDataDiary(v.Diary, ref),
	}
}

func cityToDataUnit(v cityUnit, ref func(*cityObject) uint16) CityUnitData {
	return CityUnitData{
		Token:          cityDataCopy(v.token),
		Effects:        cityDataSlice(v.effects, ref),
		Words15c:       cityDataCopy(v.words15c),
		Words178:       cityDataCopy(v.words178),
		RawA6:          cityDataCopy(v.rawA6),
		RawBE:          cityDataCopy(v.rawBE),
		Raw114:         cityDataCopy(v.raw114),
		RawD4:          cityDataCopy(v.rawD4),
		Raw154:         cityDataCopy(v.raw154),
		Raw158:         cityDataCopy(v.raw158),
		Words158:       cityDataCopy(v.words158),
		Scalar1:        cityDataCopy(v.scalar1),
		Reference74:    ref(v.reference74),
		Reference78:    ref(v.reference78),
		Name:           v.name,
		Scalar2:        cityDataCopy(v.scalar2),
		Reference68:    ref(v.reference68),
		ContainerFlag:  v.containerFlag,
		Container:      cityDataSlice(v.container, ref),
		ContainerTails: v.containerTails,
		SpellbookFlag:  v.spellbookFlag,
		SpellbookDWord: v.spellbookDWord,
		SpellbookCount: v.spellbookCount,
		Spells:         cityDataSlice(v.spells, ref),
		ScalarTail:     cityDataCopy(v.scalarTail),
		XP:             cityDataCopy(v.xp),
		Equipment:      cityDataSlice(v.equipment, ref),
	}
}

func cityFromDataUnit(v CityUnitData, ref func(uint16) *cityObject) cityUnit {
	return cityUnit{
		token:          cityDataCopy(v.Token),
		effects:        cityDataSlice(v.Effects, ref),
		words15c:       cityDataCopy(v.Words15c),
		words178:       cityDataCopy(v.Words178),
		rawA6:          cityDataCopy(v.RawA6),
		rawBE:          cityDataCopy(v.RawBE),
		raw114:         cityDataCopy(v.Raw114),
		rawD4:          cityDataCopy(v.RawD4),
		raw154:         cityDataCopy(v.Raw154),
		raw158:         cityDataCopy(v.Raw158),
		words158:       cityDataCopy(v.Words158),
		scalar1:        cityDataCopy(v.Scalar1),
		reference74:    ref(v.Reference74),
		reference78:    ref(v.Reference78),
		name:           v.Name,
		scalar2:        cityDataCopy(v.Scalar2),
		reference68:    ref(v.Reference68),
		containerFlag:  v.ContainerFlag,
		container:      cityDataSlice(v.Container, ref),
		containerTails: v.ContainerTails,
		spellbookFlag:  v.SpellbookFlag,
		spellbookDWord: v.SpellbookDWord,
		spellbookCount: v.SpellbookCount,
		spells:         cityDataSlice(v.Spells, ref),
		scalarTail:     cityDataCopy(v.ScalarTail),
		xp:             cityDataCopy(v.XP),
		equipment:      cityDataSlice(v.Equipment, ref),
	}
}

func cityToDataItem(v cityItem, ref func(*cityObject) uint16) CityItemData {
	return CityItemData{
		Token:       cityDataCopy(v.token),
		Effects:     cityDataSlice(v.effects, ref),
		Fields:      cityDataCopy(v.fields),
		Derived:     cityDataCopy(v.derived),
		WeaponExtra: ref(v.weaponExtra),
	}
}

func cityFromDataItem(v CityItemData, ref func(uint16) *cityObject) cityItem {
	return cityItem{
		token:       cityDataCopy(v.Token),
		effects:     cityDataSlice(v.Effects, ref),
		fields:      cityDataCopy(v.Fields),
		derived:     cityDataCopy(v.Derived),
		weaponExtra: ref(v.WeaponExtra),
	}
}

func cityToDataEffect(v cityEffect, ref func(*cityObject) uint16) CityEffectData {
	return CityEffectData{
		Token:  cityDataCopy(v.token),
		Fields: cityDataCopy(v.fields),
	}
}

func cityFromDataEffect(v CityEffectData, ref func(uint16) *cityObject) cityEffect {
	return cityEffect{
		token:  cityDataCopy(v.Token),
		fields: cityDataCopy(v.Fields),
	}
}

func cityToDataSpell(v citySpell, ref func(*cityObject) uint16) CitySpellData {
	return CitySpellData{
		Fields: cityDataCopy(v.fields),
	}
}

func cityFromDataSpell(v CitySpellData, ref func(uint16) *cityObject) citySpell {
	return citySpell{
		fields: cityDataCopy(v.Fields),
	}
}

func cityToDataCampaignBase(v cityCampaignBase, ref func(*cityObject) uint16) CityCampaignBaseData {
	return CityCampaignBaseData{
		DWords: v.dwords,
		Arrays: cityDataArrays2(v.arrays),
	}
}

func cityFromDataCampaignBase(v CityCampaignBaseData, ref func(uint16) *cityObject) cityCampaignBase {
	return cityCampaignBase{
		dwords: v.DWords,
		arrays: cityDataArrays2(v.Arrays),
	}
}

func cityToDataCampaignChild(v cityCampaignChild, ref func(*cityObject) uint16) CityCampaignChildData {
	return CityCampaignChildData{
		Base: cityToDataCampaignBase(v.base, ref),
		Age:  v.age,
	}
}

func cityFromDataCampaignChild(v CityCampaignChildData, ref func(uint16) *cityObject) cityCampaignChild {
	return cityCampaignChild{
		base: cityFromDataCampaignBase(v.Base, ref),
		age:  v.Age,
	}
}

func cityToDataCampaignMarker(v cityCampaignMarker, ref func(*cityObject) uint16) CityCampaignMarkerData {
	return CityCampaignMarkerData{
		Value: v.value,
		Text:  cityDataCopy(v.text),
		Tail:  v.tail,
	}
}

func cityFromDataCampaignMarker(v CityCampaignMarkerData, ref func(uint16) *cityObject) cityCampaignMarker {
	return cityCampaignMarker{
		value: v.Value,
		text:  cityDataCopy(v.Text),
		tail:  v.Tail,
	}
}

func cityToDataCampaign(v cityCampaign, ref func(*cityObject) uint16) CityCampaignData {
	return CityCampaignData{
		Base:      cityToDataCampaignBase(v.base, ref),
		Children:  cityDataSlice(v.children, func(x cityCampaignChild) CityCampaignChildData { return cityToDataCampaignChild(x, ref) }),
		Parallel:  cityDataArrays2(v.parallel),
		DWords:    cityDataCopy(v.dwords),
		Arrays:    cityDataArrays6(v.arrays),
		Documents: cityDataCopy(v.documents),
		Carriers:  cityDataCopy(v.carriers),
		Scalars:   v.scalars,
		Markers:   cityDataSlice(v.markers, func(x cityCampaignMarker) CityCampaignMarkerData { return cityToDataCampaignMarker(x, ref) }),
	}
}

func cityFromDataCampaign(v CityCampaignData, ref func(uint16) *cityObject) cityCampaign {
	return cityCampaign{
		base:      cityFromDataCampaignBase(v.Base, ref),
		children:  cityDataSlice(v.Children, func(x CityCampaignChildData) cityCampaignChild { return cityFromDataCampaignChild(x, ref) }),
		parallel:  cityDataArrays2(v.Parallel),
		dwords:    cityDataCopy(v.DWords),
		arrays:    cityDataArrays6(v.Arrays),
		documents: cityDataCopy(v.Documents),
		carriers:  cityDataCopy(v.Carriers),
		scalars:   v.Scalars,
		markers:   cityDataSlice(v.Markers, func(x CityCampaignMarkerData) cityCampaignMarker { return cityFromDataCampaignMarker(x, ref) }),
	}
}
