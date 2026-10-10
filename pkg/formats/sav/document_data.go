package sav

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"sync"

	"againrom/pkg/graphcopy"
)

// DocumentDataVersion versions the complete-document persistence contract,
// independently of the original SAV container version.
const DocumentDataVersion uint32 = 1

const (
	maxDocumentDataObjects  = int(instanceTagBit) - 1
	maxDocumentDataElements = 1 << 20
	maxDocumentDataBytes    = 64 << 20
	maxDocumentDataDepth    = 64
	maxDocumentRecordFields = 128
)

// DocumentData is a detached, map-free SAV document, suitable for ordinary gob
// persistence. References are one-based local Objects indices; zero is nil.
// Indices follow first encounter through roots, sorted RefSlots, Inline, then
// Groups. They are neither CArchive indices nor saved address keys. Address
// keys remain explicit named scalar/cell values; this codec does not remint
// them or supply constructor defaults or a live-game projection.
//
// There are deliberately no custom gob methods: an enclosing native format can
// preflight every ordinary field/count. Archive cycles use integer references,
// never recursive Go pointers. Inline records are bounded value trees.
type DocumentData struct {
	Version, FileVersion uint32
	Label                []byte
	Head                 DocumentHeadData
	Players, DeadActors  []uint16
	World                *DocumentWorldData
	Marker, GlobalDWord  uint32
	// Trailer stays the flat 100-dword array this DTO has always gob-encoded
	// (frozen native envelopes rely on the exact wire shape; gob distinguishes
	// an array from a struct of the same total width). TrailerBody is the
	// named split for the decoder and the narrow reader; trailerBodyToArray/
	// trailerBodyFromArray convert at the two boundary sites below.
	Trailer  [100]uint32
	Objects  []DocumentRecordData
	State    DocumentStateData
	Campaign CityCampaignData
}

// DocumentHeadData omits source offsets and the redundant Player count.
type DocumentHeadData struct {
	CounterA, CounterB                   uint32
	MapName                              string
	Reserved                             [11]uint32
	Mission, Difficulty, PlayerListField uint32
}

type DocumentWorldData struct {
	Buildings, Effects, Sacks []uint16
	Blocks                    []BlockRecord
	Cells                     []DocumentCellData
	TerrainIdentity           uint32
	Session                   DocumentSessionData
}

// These named, exported value types retain every field of the typed wire
// layouts, including unnamed values. Neither carries source byte spans.
type DocumentCellData archiveCell
type DocumentSessionData archiveSession

// DocumentRecordData is one complete class programme. Named fields are strictly
// name-sorted and unique within their own category. RefSlots is the sole
// authority for nullable archive references. Compact Refs, SpellSlots and
// WornSlots are reconstructed, not persisted as competing authorities.
// Inline and Groups are NOT archive-tagged objects and take no Objects index.
type DocumentRecordData struct {
	Class    string
	Values   []DocumentValueData
	Texts    []DocumentTextData
	Raw      []DocumentRawData
	Counts   []DocumentCountData
	RefSlots []DocumentRefsData
	Inline   []DocumentInlineData
	Groups   []DocumentRecordData
}

type DocumentValueData struct {
	Name  string
	Value uint32
}
type DocumentTextData struct {
	Name  string
	Value string
}
type DocumentRawData struct {
	Name  string
	Bytes []byte
}
type DocumentCountData struct {
	Name  string
	Count uint32
}
type DocumentRefsData struct {
	Name    string
	Objects []uint16
}
type DocumentInlineData struct {
	Name   string
	Record DocumentRecordData
}

// Unlike CityStateData this contract has no legacy map fields. The existing
// record/value types are shared, and the world grammar admits Fog/Projectiles.
type DocumentStateData struct {
	RootKind         uint32
	DirectoryRecords []CityStateDirectoryData
	ValueRecords     []CityStateRecordData
}

// DecodeDocumentData reads a complete supported city or world SAV into an
// independently owned persistence DTO; no File/body/store/tail replay remains.
func DecodeDocumentData(raw []byte) (DocumentData, error) {
	data, _, err := decodeDocumentDataKept(raw, false)
	return data, err
}

// DocumentObjectOrigin is a transient import binding, not persisted state.
// ArchiveIndex belongs to this input stream; ObjectIndex belongs to its DTO.
// Rows include every tagged object once and exclude inline Diary/Groups.
type DocumentObjectOrigin struct {
	ArchiveIndex, ObjectIndex uint16
}

// DecodeDocumentDataWithOrigins also returns exact source bindings captured
// during graph detachment. Equal/zero address keys do not participate in this
// join. Consumers must bind current objects before discarding these rows;
// never preserve ArchiveIndex as an identity in native persistence.
func DecodeDocumentDataWithOrigins(raw []byte) (DocumentData, []DocumentObjectOrigin, error) {
	return decodeDocumentDataKept(raw, true)
}

// decodedDocuments keeps the last few decoded containers by their exact
// bytes. A load parses one file several times over (equipment repair, game
// validation, campaign and document import); each reader gets its own copy.
var decodedDocuments struct {
	mu      sync.Mutex
	entries []decodedDocument
}

type decodedDocument struct {
	raw         string
	withOrigins bool
	data        *DocumentData
	origins     []DocumentObjectOrigin
}

const decodedDocumentEntries = 4

func decodeDocumentDataKept(raw []byte, withOrigins bool) (DocumentData, []DocumentObjectOrigin, error) {
	decodedDocuments.mu.Lock()
	for i, e := range decodedDocuments.entries {
		if e.withOrigins == withOrigins && e.raw == string(raw) {
			if data, ok := graphcopy.Clone(e.data); ok {
				copy(decodedDocuments.entries[1:i+1], decodedDocuments.entries[:i])
				decodedDocuments.entries[0] = e
				decodedDocuments.mu.Unlock()
				return *data, slices.Clone(e.origins), nil
			}
			break
		}
	}
	decodedDocuments.mu.Unlock()
	d, err := parseSaveDocument(raw)
	if err != nil {
		return DocumentData{}, nil, err
	}
	var origins []DocumentObjectOrigin
	var data DocumentData
	if withOrigins {
		data, err = documentDataFromDocument(d, &origins)
	} else {
		data, err = documentDataFromDocument(d, nil)
	}
	if err != nil {
		return DocumentData{}, nil, err
	}
	if kept, ok := graphcopy.Clone(&data); ok {
		decodedDocuments.mu.Lock()
		entries := append([]decodedDocument{{raw: string(raw), withOrigins: withOrigins, data: kept, origins: slices.Clone(origins)}}, decodedDocuments.entries...)
		decodedDocuments.entries = entries[:min(len(entries), decodedDocumentEntries)]
		decodedDocuments.mu.Unlock()
	}
	return data, origins, nil
}

// EncodeDocumentData validates the whole DTO and builds a new SAV container.
// Unknown/inactive fields, conflicting derived counts, noncanonical indices
// and unreachable table objects are errors, not silently discarded data.
func EncodeDocumentData(data DocumentData) ([]byte, error) {
	d, err := saveDocumentFromData(data)
	if err != nil {
		return nil, err
	}
	return serializeSaveDocument(d)
}

// CloneDocumentData validates and independently copies the complete graph.
// Empty slices are canonicalized; EncodeDocumentData checks compressed size.
func CloneDocumentData(data DocumentData) (DocumentData, error) {
	d, err := saveDocumentFromDataIndexedMode(data, nil, true)
	if err != nil {
		return DocumentData{}, err
	}
	return copyValidatedDocumentData(data, d, nil)
}

// ReindexDocumentData validates and independently copies an edited graph,
// allowing a different first-encounter order. All objects must be reachable.
// Callers must apply the old-to-new permutation to every external binding.
// Slot zero stays nil; numeric wire keys and inline Group ordinals stay intact.
// Decode, Clone and Encode still require canonical ordering.
func ReindexDocumentData(data DocumentData) (DocumentData, []uint16, error) {
	if len(data.Objects) > maxDocumentDataObjects {
		return DocumentData{}, nil, fmt.Errorf("sav: document object count exceeds bound")
	}
	permutation := make([]uint16, len(data.Objects)+1)
	d, err := saveDocumentFromDataIndexedMode(data, permutation, true)
	if err != nil {
		return DocumentData{}, nil, err
	}
	out, err := copyValidatedDocumentData(data, d, permutation)
	if err != nil {
		return DocumentData{}, nil, err
	}
	if err := remapNativeActionObjects(&out.State, permutation); err != nil {
		return DocumentData{}, nil, err
	}
	return out, permutation, nil
}

func documentSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// documentRecordShape follows the same complete programme as the wire codec.
// It checks all named authorities, including values ignored by an inactive
// branch. Empty slot sites and derived Counts get one canonical representation.
// It does not traverse archive edges, so a cycle does not recurse here.
type documentRecordShape struct {
	values, texts, raw map[string]bool
	counts, slots      map[string]int
	inline             map[string]string
	groups             bool
}

func shapeDocumentRecord(r *Record, group bool) (*documentRecordShape, error) {
	if r == nil {
		return nil, fmt.Errorf("sav: nil document record")
	}
	prog, ok := programmes[r.Class]
	if group && r.Class == "Group" {
		prog, ok = groupProgramme, true
	}
	if !ok || len(r.Class) > maxClassName {
		return nil, fmt.Errorf("sav: unsupported document class %q", r.Class)
	}
	if len(r.Value)+len(r.Text)+len(r.Raw)+len(r.Counts)+len(r.RefSlots)+len(r.Refs) > maxDocumentRecordFields*2 {
		return nil, fmt.Errorf("sav: document record field bound exceeded")
	}
	s := &documentRecordShape{values: make(map[string]bool, len(r.Value)), texts: make(map[string]bool, len(r.Text)), raw: make(map[string]bool, len(r.Raw)), counts: make(map[string]int, len(r.Counts)), slots: make(map[string]int, len(r.RefSlots)), inline: map[string]string{}}
	value := func(name string) (uint32, error) {
		v, found := r.Value[name]
		if !found {
			return 0, fmt.Errorf("sav: %s missing scalar %s", r.Class, name)
		}
		s.values[name] = true
		return v, nil
	}
	count := func(name string) (int, error) {
		n, err := archiveCount(r, name)
		if err == nil {
			s.counts[name] = n
		}
		return n, err
	}
	var walk func([]step) error
	walk = func(prog []step) error {
		for _, step := range prog {
			switch step.op {
			case stpRun:
				for _, m := range step.members {
					switch m.Kind {
					case KindCString:
						v, found := r.Text[m.Name]
						if !found || len(v) >= 0xff {
							return fmt.Errorf("sav: %s invalid text %s", r.Class, m.Name)
						}
						s.texts[m.Name] = true
					case KindRaw:
						v, found := r.Raw[m.Name]
						if !found || len(v) != m.Len {
							return fmt.Errorf("sav: %s invalid raw field %s", r.Class, m.Name)
						}
						s.raw[m.Name] = true
					default:
						v, err := value(m.Name)
						if err != nil {
							return err
						}
						if (m.Kind == KindU8 && v > 0xff) || (m.Kind == KindU16 && v > 0xffff) || (m.Kind == KindU16Saturated && v > saturate) {
							return fmt.Errorf("sav: %s.%s exceeds its lossless field range", r.Class, m.Name)
						}
					}
				}
			case stpClass:
				if err := walk(programmes[step.class]); err != nil {
					return err
				}
			case stpFlagged:
				flag, err := value(step.name)
				if err != nil || flag > 1 {
					return fmt.Errorf("sav: %s invalid flag %s", r.Class, step.name)
				}
				if flag == 1 {
					if err := walk(step.sub); err != nil {
						return err
					}
				}
			case stpObjRef:
				s.slots[step.name] = 1
			case stpRefRun:
				s.slots[step.name], s.counts[step.name] = step.n, step.n
			case stpList, stpContainer, stpSpellbook:
				n, err := count(step.name)
				if err != nil {
					return err
				}
				s.slots[step.name] = n
				if step.op == stpContainer {
					for _, suffix := range []string{"1C", "20"} {
						if _, err := value(step.name + suffix); err != nil {
							return err
						}
					}
				} else if step.op == stpSpellbook {
					s.slots[step.name] = max(0, n-1)
					if _, err := value(step.name + "Header"); err != nil {
						return err
					}
				}
			case stpU16List, stpDWordArray, stpWordArray, stpRawArray:
				n, err := count(step.name)
				if err != nil {
					return err
				}
				width := 2
				if step.op == stpDWordArray {
					width = 4
				} else if step.op == stpRawArray {
					width = step.n
				}
				b, found := r.Raw[step.name]
				if !found || len(b) != width*n {
					return fmt.Errorf("sav: %s invalid array %s", r.Class, step.name)
				}
				s.raw[step.name] = true
			case stpInline:
				refs := r.Refs[step.name]
				if len(refs) != 1 || refs[0] == nil || refs[0].Class != step.class {
					return fmt.Errorf("sav: %s invalid inline %s", r.Class, step.name)
				}
				s.inline[step.name], s.counts[step.name] = step.class, 1
			case stpGroups:
				n, err := count(step.name)
				if err != nil || len(r.Groups) != n {
					return fmt.Errorf("sav: %s invalid group count", r.Class)
				}
				s.groups = true
				actors := 0
				for _, g := range r.Groups {
					if g == nil || g.Class != "Group" {
						return fmt.Errorf("sav: invalid inline Group")
					}
					n, err := archiveCount(g, "Actors")
					if err != nil || actors > maxDocumentDataElements-n {
						return fmt.Errorf("sav: document group actor count exceeds bound")
					}
					actors += n
				}
				s.counts["Actors"] = actors
			default:
				return fmt.Errorf("sav: unsupported document operation %d", step.op)
			}
		}
		return nil
	}
	if err := walk(prog); err != nil {
		return nil, err
	}
	if len(r.Value) != len(s.values) || len(r.Text) != len(s.texts) || len(r.Raw) != len(s.raw) {
		return nil, fmt.Errorf("sav: %s has unknown or inactive named fields", r.Class)
	}
	for k, n := range r.Counts {
		want, ok := s.counts[k]
		if !ok || n != want {
			return nil, fmt.Errorf("sav: %s has unknown or conflicting count %s", r.Class, k)
		}
	}
	for k, refs := range r.RefSlots {
		n, ok := s.slots[k]
		if !ok || len(refs) != n {
			return nil, fmt.Errorf("sav: %s has unknown or wrong-width slots %s", r.Class, k)
		}
	}
	for k, n := range s.slots {
		if len(r.RefSlots[k]) != n {
			return nil, fmt.Errorf("sav: %s missing complete slots %s", r.Class, k)
		}
	}
	for k := range r.Refs {
		_, slots := s.slots[k]
		_, inline := s.inline[k]
		if !slots && !inline && !(s.groups && k == "Actors") {
			return nil, fmt.Errorf("sav: %s has unknown reference site %s", r.Class, k)
		}
	}
	if !s.groups && len(r.Groups) != 0 {
		return nil, fmt.Errorf("sav: %s has inactive Groups", r.Class)
	}
	return s, nil
}

// This preflight counts repeated slice aliases once PER OCCURRENCE, before any
// conversion copies them. Its finite depth also catches adversarial recursive
// inline slice values even though the ordinary contract is an acyclic tree.
type documentDataBudget struct{ bytes, elements uint64 }

func (b *documentDataBudget) add(n, elements uint64) error {
	if n > maxDocumentDataBytes-b.bytes || elements > maxDocumentDataElements-b.elements {
		return fmt.Errorf("sav: document data byte/element budget exceeded")
	}
	b.bytes += n
	b.elements += elements
	return nil
}

func (b *documentDataBudget) check(v reflect.Value, depth int) error {
	if depth > maxDocumentDataDepth {
		return fmt.Errorf("sav: document data depth bound exceeded")
	}
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			if err := b.add(uint64(v.Type().Elem().Size()), 1); err != nil {
				return err
			}
			return b.check(v.Elem(), depth+1)
		}
	case reflect.String:
		return b.add(uint64(v.Len()), 0)
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice {
			count := uint64(v.Len())
			elements := count
			if v.Type().Elem().Kind() == reflect.Uint8 {
				elements = 0
			}
			if count > maxDocumentDataBytes/uint64(max(1, v.Type().Elem().Size())) {
				return fmt.Errorf("sav: document data slice byte bound exceeded")
			}
			if err := b.add(count*uint64(v.Type().Elem().Size()), elements); err != nil {
				return err
			}
		}
		switch v.Type().Elem().Kind() {
		case reflect.Struct:
			// One field plan serves every element.
			if depth+1 > maxDocumentDataDepth && v.Len() > 0 {
				return fmt.Errorf("sav: document data depth bound exceeded")
			}
			plan := budgetedFields(v.Type().Elem())
			for i := 0; i < v.Len(); i++ {
				if err := b.checkStruct(v.Index(i), depth+1, plan); err != nil {
					return err
				}
			}
		case reflect.Array, reflect.Slice, reflect.Pointer, reflect.String:
			for i := 0; i < v.Len(); i++ {
				if err := b.check(v.Index(i), depth+1); err != nil {
					return err
				}
			}
		}
	case reflect.Struct:
		return b.checkStruct(v, depth, budgetedFields(v.Type()))
	case reflect.Map, reflect.Interface:
		return fmt.Errorf("sav: document data cannot contain maps or interfaces")
	}
	return nil
}

// checkStruct is check's struct case at depth, which check's own depth test
// has admitted, with budgetedFields of v's type in plan.
func (b *documentDataBudget) checkStruct(v reflect.Value, depth int, plan []int) error {
	if depth+1 > maxDocumentDataDepth {
		if v.NumField() > 0 {
			return fmt.Errorf("sav: document data depth bound exceeded")
		}
		return nil
	}
	// A field whose check only tests the depth is skipped: depth+1 is
	// within the bound here.
	for _, i := range plan {
		if err := b.check(v.Field(i), depth+1); err != nil {
			return err
		}
	}
	return nil
}

var budgetFieldPlans sync.Map // reflect.Type -> []int

// budgetedFields lists the fields of struct type t whose check can add to the
// budget or fail beyond the depth test: every field but a scalar or an array
// of scalars.
func budgetedFields(t reflect.Type) []int {
	if plan, ok := budgetFieldPlans.Load(t); ok {
		return plan.([]int)
	}
	var plan []int
	for i := 0; i < t.NumField(); i++ {
		if budgetVisits(t.Field(i).Type) {
			plan = append(plan, i)
		}
	}
	budgetFieldPlans.Store(t, plan)
	return plan
}

func budgetVisits(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Pointer, reflect.String, reflect.Slice, reflect.Struct, reflect.Map, reflect.Interface:
		return true
	case reflect.Array:
		switch t.Elem().Kind() {
		case reflect.Struct, reflect.Array, reflect.Slice, reflect.Pointer, reflect.String:
			return true
		}
	}
	return false
}

type documentDataBuilder struct {
	objects []DocumentRecordData
	ids     map[*Record]uint16
	inline  map[*Record]bool
	budget  documentDataBudget
}

func (b *documentDataBuilder) ref(r *Record, depth int) (uint16, error) {
	if r == nil {
		return 0, nil
	}
	if b.inline[r] {
		return 0, fmt.Errorf("sav: inline record also used as an archive reference")
	}
	if id, ok := b.ids[r]; ok {
		return id, nil
	}
	if len(b.objects) == maxDocumentDataObjects {
		return 0, fmt.Errorf("sav: document object count exceeds bound")
	}
	id := uint16(len(b.objects) + 1)
	b.ids[r] = id // Publish before children, including a self reference.
	b.objects = append(b.objects, DocumentRecordData{})
	data, err := b.record(r, false, depth)
	if err != nil {
		return 0, err
	}
	b.objects[id-1] = data
	return id, nil
}

func (b *documentDataBuilder) refs(records []*Record, depth int) ([]uint16, error) {
	if len(records) > maxListElements {
		return nil, fmt.Errorf("sav: document reference count exceeds bound")
	}
	if err := b.budget.add(uint64(2*len(records)), uint64(len(records))); err != nil {
		return nil, err
	}
	var out []uint16
	if len(records) != 0 {
		out = make([]uint16, len(records))
	}
	for i, r := range records {
		var err error
		out[i], err = b.ref(r, depth)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (b *documentDataBuilder) embedded(r *Record, group bool, depth int) (DocumentRecordData, error) {
	if r == nil || b.inline[r] || b.ids[r] != 0 {
		return DocumentRecordData{}, fmt.Errorf("sav: inline record has nil, repeated or archive ownership")
	}
	b.inline[r] = true
	return b.record(r, group, depth)
}

func (b *documentDataBuilder) record(r *Record, group bool, depth int) (DocumentRecordData, error) {
	if depth > maxDocumentDataDepth {
		return DocumentRecordData{}, fmt.Errorf("sav: document graph depth bound exceeded")
	}
	s, err := shapeDocumentRecord(r, group)
	if err != nil {
		return DocumentRecordData{}, err
	}
	out := DocumentRecordData{Class: r.Class}
	for _, k := range documentSortedKeys(r.Value) {
		out.Values = append(out.Values, DocumentValueData{k, r.Value[k]})
	}
	for _, k := range documentSortedKeys(r.Text) {
		out.Texts = append(out.Texts, DocumentTextData{k, r.Text[k]})
	}
	for _, k := range documentSortedKeys(r.Raw) {
		out.Raw = append(out.Raw, DocumentRawData{k, r.Raw[k]})
	}
	for _, k := range documentSortedKeys(s.counts) {
		out.Counts = append(out.Counts, DocumentCountData{k, uint32(s.counts[k])})
	}
	// Account all raw aliases before copying the first byte of this record.
	if err := b.budget.check(reflect.ValueOf(out), 0); err != nil {
		return DocumentRecordData{}, err
	}
	for i := range out.Raw {
		out.Raw[i].Bytes = cityDataCopy(out.Raw[i].Bytes)
	}
	for _, k := range documentSortedKeys(s.slots) {
		refs, err := b.refs(r.RefSlots[k], depth+1)
		if err != nil {
			return DocumentRecordData{}, err
		}
		out.RefSlots = append(out.RefSlots, DocumentRefsData{k, refs})
	}
	for _, k := range documentSortedKeys(s.inline) {
		v, err := b.embedded(r.Refs[k][0], false, depth+1)
		if err != nil {
			return DocumentRecordData{}, err
		}
		out.Inline = append(out.Inline, DocumentInlineData{k, v})
	}
	if err := b.budget.add(uint64(len(r.Groups))*uint64(reflect.TypeFor[DocumentRecordData]().Size()), uint64(len(r.Groups))); err != nil {
		return DocumentRecordData{}, err
	}
	for _, g := range r.Groups {
		v, err := b.embedded(g, true, depth+1)
		if err != nil {
			return DocumentRecordData{}, err
		}
		out.Groups = append(out.Groups, v)
	}
	return out, nil
}

func saveDocumentToData(d *saveDocument) (DocumentData, error) {
	return documentDataFromDocument(d, nil)
}

func documentDataFromDocument(d *saveDocument, origins *[]DocumentObjectOrigin) (DocumentData, error) {
	if d == nil || d.archive == nil || d.state == nil || d.campaign == nil {
		return DocumentData{}, fmt.Errorf("sav: incomplete document")
	}
	if err := documentDataEnvelope(d.version, d.label, d.archive.head.MapName, d.archive.marker, d.archive.global); err != nil {
		return DocumentData{}, err
	}
	if _, err := worldStateSerializedSize(d.state); err != nil {
		return DocumentData{}, err
	}
	if d.archive.world == nil {
		if err := validateCityState(d.state); err != nil {
			return DocumentData{}, err
		}
	} else if err := validateWorldState(d.state); err != nil {
		return DocumentData{}, err
	}
	if _, err := archiveCampaignSize(d.campaign); err != nil {
		return DocumentData{}, err
	}
	a := d.archive
	h := a.head
	out := DocumentData{Version: DocumentDataVersion, FileVersion: d.version, Label: cityDataCopy(d.label), Head: DocumentHeadData{h.CounterA, h.CounterB, h.MapName, h.Reserved, h.Mission, h.Difficulty, h.PlayerListField}, Marker: a.marker, GlobalDWord: a.global, Trailer: trailerBodyToArray(a.trailer)}
	b := documentDataBuilder{ids: map[*Record]uint16{}, inline: map[*Record]bool{}}
	var err error
	if out.Players, err = b.refs(a.players, 0); err != nil {
		return DocumentData{}, err
	}
	if out.DeadActors, err = b.refs(a.dead, 0); err != nil {
		return DocumentData{}, err
	}
	if w := a.world; w != nil {
		if len(w.blocks) > maxListElements || len(w.cells) > maxListElements {
			return DocumentData{}, fmt.Errorf("sav: document terrain count exceeds bound")
		}
		out.World = &DocumentWorldData{Blocks: cityDataCopy(w.blocks), TerrainIdentity: w.terrainIdentity, Session: DocumentSessionData(w.session)}
		for _, c := range w.cells {
			out.World.Cells = append(out.World.Cells, DocumentCellData(c))
		}
		for _, list := range []struct {
			src []*Record
			dst *[]uint16
		}{{w.buildings, &out.World.Buildings}, {w.effects, &out.World.Effects}, {w.sacks, &out.World.Sacks}} {
			if *list.dst, err = b.refs(list.src, 0); err != nil {
				return DocumentData{}, err
			}
		}
	}
	out.Objects = b.objects
	out.State.RootKind = d.state.rootKind
	for _, k := range documentSortedKeys(d.state.directoryKinds) {
		out.State.DirectoryRecords = append(out.State.DirectoryRecords, CityStateDirectoryData{k, d.state.directoryKinds[k]})
	}
	for _, k := range documentSortedKeys(d.state.values) {
		v := d.state.values[k]
		if v.kind != 2 {
			v.int32 = 0 // Pool offsets are not retained scalar state.
		}
		out.State.ValueRecords = append(out.State.ValueRecords, CityStateRecordData{k, CityStateValueData{v.kind, v.int32, cityDataCopy(v.bytes)}})
	}
	out.Campaign = cityToDataCampaign(*d.campaign, nil)
	// The shared city helper preserves allocated empty struct slices. Gob does
	// not, so the complete DTO chooses nil for both before native adoption.
	// All other campaign slices already pass through cityDataCopy.
	if len(out.Campaign.Children) == 0 {
		out.Campaign.Children = nil
	}
	if len(out.Campaign.Markers) == 0 {
		out.Campaign.Markers = nil
	}
	if err := (&documentDataBudget{}).check(reflect.ValueOf(out), 0); err != nil {
		return DocumentData{}, err
	}
	if origins != nil {
		rows := make([]DocumentObjectOrigin, len(b.ids))
		for record, localID := range b.ids {
			archiveID := a.origins[record]
			if archiveID == 0 {
				return DocumentData{}, fmt.Errorf("sav: missing exact import origin for document object %d", localID)
			}
			rows[localID-1] = DocumentObjectOrigin{archiveID, localID}
		}
		*origins = rows
	}
	return out, nil
}

func documentDataGraph(data DocumentData, permutation []uint16) error {
	if len(data.Objects) > maxDocumentDataObjects {
		return fmt.Errorf("sav: document object count exceeds bound")
	}
	seen := make([]bool, len(data.Objects))
	next := 1
	var refs func([]uint16, int) error
	var record func(DocumentRecordData, int) error
	record = func(r DocumentRecordData, depth int) error {
		if depth > maxDocumentDataDepth {
			return fmt.Errorf("sav: document graph depth bound exceeded")
		}
		for _, slot := range r.RefSlots {
			if err := refs(slot.Objects, depth+1); err != nil {
				return err
			}
		}
		for _, inline := range r.Inline {
			if err := record(inline.Record, depth+1); err != nil {
				return err
			}
		}
		for _, g := range r.Groups {
			if err := record(g, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	refs = func(ids []uint16, depth int) error {
		if len(ids) > maxListElements {
			return fmt.Errorf("sav: document reference count exceeds bound")
		}
		for _, id := range ids {
			if id == 0 {
				continue
			}
			if int(id) > len(seen) {
				return fmt.Errorf("sav: document reference %d outside object table", id)
			}
			if seen[id-1] {
				continue
			}
			if permutation == nil && int(id) != next {
				return fmt.Errorf("sav: document object indices are not in first-encounter order")
			}
			if permutation != nil {
				permutation[id] = uint16(next)
			}
			next++
			seen[id-1] = true
			if err := record(data.Objects[id-1], depth); err != nil {
				return err
			}
		}
		return nil
	}
	lists := [][]uint16{data.Players, data.DeadActors}
	if w := data.World; w != nil {
		lists = append(lists, w.Buildings, w.Effects, w.Sacks)
	}
	for _, list := range lists {
		if err := refs(list, 0); err != nil {
			return err
		}
	}
	if next != len(data.Objects)+1 {
		return fmt.Errorf("sav: document has unreachable objects")
	}
	return nil
}

func documentFieldOrder(previous *string, name string) error {
	if name <= *previous || len(name) > maxClassName {
		return fmt.Errorf("sav: document fields are not strictly name ordered")
	}
	*previous = name
	return nil
}

func documentRecordFromData(data DocumentRecordData, out *Record, objects []*Record, group bool, depth int) error {
	return documentRecordFromDataMode(data, out, objects, group, depth, false)
}

func documentRecordFromDataMode(data DocumentRecordData, out *Record, objects []*Record, group bool, depth int, borrowRaw bool) error {
	if depth > maxDocumentDataDepth {
		return fmt.Errorf("sav: document inline depth bound exceeded")
	}
	if len(data.Values)+len(data.Texts)+len(data.Raw)+len(data.Counts)+len(data.RefSlots)+len(data.Inline) > maxDocumentRecordFields {
		return fmt.Errorf("sav: document record field bound exceeded")
	}
	prev := ""
	for _, v := range data.Values {
		if err := documentFieldOrder(&prev, v.Name); err != nil {
			return err
		}
		out.Value[v.Name] = v.Value
	}
	prev = ""
	for _, v := range data.Texts {
		if err := documentFieldOrder(&prev, v.Name); err != nil {
			return err
		}
		out.Text[v.Name] = v.Value
	}
	prev = ""
	for _, v := range data.Raw {
		if err := documentFieldOrder(&prev, v.Name); err != nil {
			return err
		}
		// Validation only reads these bytes; the public copy detaches them.
		if borrowRaw {
			out.Raw[v.Name] = v.Bytes
		} else {
			out.Raw[v.Name] = cityDataCopy(v.Bytes)
		}
	}
	prev = ""
	for _, v := range data.Counts {
		if err := documentFieldOrder(&prev, v.Name); err != nil {
			return err
		}
		if v.Count > maxDocumentDataElements {
			return fmt.Errorf("sav: document count exceeds bound")
		}
		out.Counts[v.Name] = int(v.Count)
	}
	prev = ""
	for _, v := range data.RefSlots {
		if err := documentFieldOrder(&prev, v.Name); err != nil {
			return err
		}
		out.RefSlots[v.Name] = nil
		for _, id := range v.Objects {
			var r *Record
			if id != 0 {
				if int(id) > len(objects) {
					return fmt.Errorf("sav: document reference outside object table")
				}
				r = objects[id-1]
			}
			out.RefSlots[v.Name] = append(out.RefSlots[v.Name], r)
			if r != nil {
				out.Refs[v.Name] = append(out.Refs[v.Name], r)
			}
		}
	}
	prev = ""
	for _, v := range data.Inline {
		if err := documentFieldOrder(&prev, v.Name); err != nil {
			return err
		}
		if _, conflict := out.RefSlots[v.Name]; conflict {
			return fmt.Errorf("sav: inline site also carries archive slots")
		}
		r := newRecord(v.Record.Class, 0, 0)
		if err := documentRecordFromDataMode(v.Record, r, objects, false, depth+1, borrowRaw); err != nil {
			return err
		}
		out.Refs[v.Name] = []*Record{r}
	}
	for _, v := range data.Groups {
		r := newRecord(v.Class, 0, 0)
		if err := documentRecordFromDataMode(v, r, objects, true, depth+1, borrowRaw); err != nil {
			return err
		}
		out.Groups = append(out.Groups, r)
		out.Refs["Actors"] = append(out.Refs["Actors"], r.Refs["Actors"]...)
	}
	s, err := shapeDocumentRecord(out, group)
	if err != nil {
		return err
	}
	// Unlike parsed Record's sparse zero maps, the DTO has all sites and all
	// derived counts. Requiring them prevents equivalent ignored-field models.
	if len(out.Counts) != len(s.counts) || len(out.RefSlots) != len(s.slots) || len(data.Inline) != len(s.inline) {
		return fmt.Errorf("sav: document record omits canonical counts/slots/inline sites")
	}
	out.SpellSlots = append([]*Record(nil), out.RefSlots["Spells"]...)
	out.WornSlots = append([]*Record(nil), out.RefSlots["Worn"]...)
	return nil
}

func saveDocumentFromData(data DocumentData) (*saveDocument, error) {
	return saveDocumentFromDataIndexed(data, nil)
}

func saveDocumentFromDataIndexed(data DocumentData, permutation []uint16) (*saveDocument, error) {
	return saveDocumentFromDataIndexedMode(data, permutation, false)
}

func saveDocumentFromDataIndexedMode(data DocumentData, permutation []uint16, borrowRaw bool) (*saveDocument, error) {
	if data.Version != DocumentDataVersion {
		return nil, fmt.Errorf("sav: unsupported document data version %d", data.Version)
	}
	if err := documentDataEnvelope(data.FileVersion, data.Label, data.Head.MapName, data.Marker, data.GlobalDWord); err != nil {
		return nil, err
	}
	if err := (&documentDataBudget{}).check(reflect.ValueOf(data), 0); err != nil {
		return nil, err
	}
	if permutation != nil && len(permutation) != len(data.Objects)+1 {
		return nil, fmt.Errorf("sav: invalid document index permutation extent")
	}
	if err := documentDataGraph(data, permutation); err != nil {
		return nil, err
	}
	if err := validateDocumentRoots(data); err != nil {
		return nil, err
	}
	if err := validateDocumentCampaign(data.Campaign); err != nil {
		return nil, err
	}
	objects := make([]*Record, len(data.Objects))
	for i, r := range data.Objects {
		objects[i] = newRecord(r.Class, 0, 0)
	}
	for i, r := range data.Objects {
		if err := documentRecordFromDataMode(r, objects[i], objects, false, 0, borrowRaw); err != nil {
			return nil, fmt.Errorf("sav: document object %d: %w", i+1, err)
		}
	}
	refs := func(ids []uint16) []*Record {
		var out []*Record
		for _, id := range ids {
			if id == 0 {
				out = append(out, nil)
			} else {
				out = append(out, objects[id-1])
			}
		}
		return out
	}
	h := data.Head
	a := &archiveDocument{head: Head{CounterA: h.CounterA, CounterB: h.CounterB, MapName: h.MapName, Reserved: h.Reserved, Mission: h.Mission, Difficulty: h.Difficulty, PlayerListField: h.PlayerListField}, players: refs(data.Players), dead: refs(data.DeadActors), marker: data.Marker, global: data.GlobalDWord, trailer: trailerBodyFromArray(data.Trailer)}
	if w := data.World; w != nil {
		if len(w.Blocks) > maxListElements || len(w.Cells) > maxListElements {
			return nil, fmt.Errorf("sav: document terrain count exceeds bound")
		}
		a.world = &archiveWorld{buildings: refs(w.Buildings), effects: refs(w.Effects), sacks: refs(w.Sacks), blocks: cityDataCopy(w.Blocks), terrainIdentity: w.TerrainIdentity, session: archiveSession(w.Session)}
		for _, cell := range w.Cells {
			a.world.cells = append(a.world.cells, archiveCell(cell))
		}
	}
	s, err := cityStateFromData(CityStateData{RootKind: data.State.RootKind, DirectoryRecords: data.State.DirectoryRecords, ValueRecords: data.State.ValueRecords}, 2)
	if err != nil {
		return nil, err
	}
	if a.world == nil {
		err = validateCityState(s)
	} else {
		err = validateWorldState(s)
	}
	if err != nil {
		return nil, err
	}
	campaign := cityFromDataCampaign(data.Campaign, nil)
	if _, err := archiveCampaignSize(&campaign); err != nil {
		return nil, err
	}
	if err := validateDocumentArchive(a); err != nil {
		return nil, err
	}
	return &saveDocument{version: data.FileVersion, label: cityDataCopy(data.Label), archive: a, state: s, campaign: &campaign}, nil
}

func documentDataEnvelope(version uint32, label []byte, mapName string, marker, global uint32) error {
	if version < MinVersion || len(label) >= 256 || bytes.IndexByte(label, 0) >= 0 || len(mapName) >= 0xff {
		return fmt.Errorf("sav: document version/label/map envelope is invalid")
	}
	if marker != 0xbadface1 && global != 0 {
		return fmt.Errorf("sav: document global value has no active marker arm")
	}
	return nil
}

// Called after graph range checks. Root classes/nullability and terrain order
// must be checked for Clone too, before a caller adopts the document natively.
func validateDocumentRoots(data DocumentData) error {
	type rootList struct {
		ids      []uint16
		class    string
		nullable bool
	}
	lists := []rootList{{data.Players, "Player", true}, {data.DeadActors, "Unit", false}}
	if w := data.World; w != nil {
		if len(w.Blocks) > maxListElements || len(w.Cells) > maxListElements {
			return fmt.Errorf("sav: document terrain count exceeds bound")
		}
		for i, b := range w.Blocks {
			if i > 0 && b.Cell <= w.Blocks[i-1].Cell {
				return fmt.Errorf("sav: document block keys are not strictly ordered")
			}
		}
		lists = append(lists, rootList{w.Buildings, "Building", false}, rootList{w.Effects, "SpellEffect", false}, rootList{w.Sacks, "Sack", false})
	}
	for _, list := range lists {
		for _, id := range list.ids {
			if id == 0 {
				if !list.nullable {
					return fmt.Errorf("sav: null document %s root", list.class)
				}
			} else if !groundClass(data.Objects[id-1].Class, list.class) {
				return fmt.Errorf("sav: document root is not an admitted %s", list.class)
			}
		}
	}
	return nil
}

func validateDocumentCampaign(c CityCampaignData) error {
	count := func(n int) error {
		if n > maxCampaignElements {
			return fmt.Errorf("sav: document campaign count exceeds bound")
		}
		return nil
	}
	base := func(b CityCampaignBaseData) error {
		if b.DWords[5] > 1 {
			return fmt.Errorf("sav: document campaign announced flag is not boolean")
		}
		for _, a := range b.Arrays {
			if err := count(len(a)); err != nil {
				return err
			}
		}
		return nil
	}
	if err := base(c.Base); err != nil {
		return err
	}
	for _, n := range []int{len(c.Children), len(c.Parallel[0]), len(c.Parallel[1]), len(c.DWords), len(c.Documents), len(c.Markers)} {
		if err := count(n); err != nil {
			return err
		}
	}
	if len(c.Parallel[0]) != len(c.Parallel[1]) || c.Scalars[4] > 1 {
		return fmt.Errorf("sav: document campaign parallel arrays/first marker flag are invalid")
	}
	for _, child := range c.Children {
		if err := base(child.Base); err != nil {
			return err
		}
	}
	for _, a := range c.Arrays {
		if err := count(len(a)); err != nil {
			return err
		}
	}
	for _, hired := range c.DWords {
		if hired > 1 {
			return fmt.Errorf("sav: document campaign hire flag is not boolean")
		}
	}
	for _, document := range c.Documents {
		if document[1] > 1 {
			return fmt.Errorf("sav: document campaign document kind is not boolean")
		}
	}
	if len(c.Documents)+len(c.Carriers) > maxCampaignDocumentPairs {
		return fmt.Errorf("sav: document campaign document list exceeds bound")
	}
	for _, carrier := range c.Carriers {
		if !IsCarrierKind(carrier[1]) {
			return fmt.Errorf("sav: document campaign carrier kind lacks the carrier bit")
		}
	}
	for _, marker := range c.Markers {
		text := marker.Text
		if len(text) < 2 || text[len(text)-1] != 0 || bytes.IndexByte(text[:len(text)-1], 0) >= 0 {
			return fmt.Errorf("sav: document campaign marker is not one nonempty NUL-terminated string")
		}
	}
	return nil
}

// Validate original archive nesting/index/decoded-size limits without making
// an output buffer. Clone must not admit a graph which only fails when a later
// SAVE discovers an over-deep archive or an exhausted shared tag namespace.
func validateDocumentArchive(a *archiveDocument) error {
	var size uint64
	add := func(n uint64) error {
		if n > maxArchiveOutput-size {
			return fmt.Errorf("sav: document decoded archive byte bound exceeded")
		}
		size += n
		return nil
	}
	seen := map[*Record]bool{}
	classes := map[string]bool{}
	var ref func(*Record, int) error
	var run func(*Record, []step, int) error
	refs := func(records []*Record, depth int) error {
		for _, r := range records {
			if err := ref(r, depth); err != nil {
				return err
			}
		}
		return nil
	}
	countWidth := func(n int) uint64 {
		if n >= 0xffff {
			return 6
		}
		return 2
	}
	ref = func(r *Record, depth int) error {
		if depth > maxWalkDepth {
			return fmt.Errorf("sav: document archive nesting exceeds bound %d", maxWalkDepth)
		}
		if err := add(2); err != nil {
			return err
		}
		if r == nil || seen[r] {
			return nil
		}
		if !classes[r.Class] {
			classes[r.Class] = true
			if err := add(uint64(4 + len(r.Class))); err != nil {
				return err
			}
		}
		seen[r] = true
		if len(seen)+len(classes) > maxDocumentDataObjects {
			return fmt.Errorf("sav: document archive index bound exceeded")
		}
		return run(r, programmes[r.Class], depth)
	}
	run = func(r *Record, prog []step, depth int) error {
		for _, s := range prog {
			var n uint64
			switch s.op {
			case stpRun:
				for _, m := range s.members {
					width, _ := m.Kind.width(m.Len)
					if m.Kind == KindCString {
						width = 1 + len(r.Text[m.Name])
					}
					n += uint64(width)
				}
			case stpClass:
				if err := run(r, programmes[s.class], depth); err != nil {
					return err
				}
			case stpInline:
				if err := run(r.Refs[s.name][0], programmes[s.class], depth); err != nil {
					return err
				}
			case stpObjRef, stpRefRun, stpList, stpContainer, stpSpellbook:
				if err := refs(r.RefSlots[s.name], depth+1); err != nil {
					return err
				}
				if s.op == stpList {
					n = 4
				} else if s.op == stpContainer {
					n = 12
				} else if s.op == stpSpellbook {
					n = 8
				}
			case stpU16List, stpDWordArray, stpWordArray, stpRawArray:
				n = countWidth(r.Counts[s.name]) + uint64(len(r.Raw[s.name]))
			case stpFlagged:
				n = 1
				if r.Value[s.name] == 1 {
					if err := run(r, s.sub, depth); err != nil {
						return err
					}
				}
			case stpGroups:
				n = 4
				for _, g := range r.Groups {
					if err := run(g, groupProgramme, depth); err != nil {
						return err
					}
				}
			default:
				return fmt.Errorf("sav: unsupported document archive operation %d", s.op)
			}
			if err := add(n); err != nil {
				return err
			}
		}
		return nil
	}
	if err := add(uint64(8 + 1 + len(a.head.MapName) + 44 + 12 + 1 + 4 + 400)); err != nil {
		return err
	}
	if a.marker == 0xbadface1 {
		if err := add(4); err != nil {
			return err
		}
	}
	for _, list := range a.rootLists() {
		if err := add(4); err != nil {
			return err
		}
		if err := refs(*list, 0); err != nil {
			return err
		}
	}
	if w := a.world; w != nil {
		if err := add(countWidth(len(w.blocks)) + uint64(len(w.blocks))*4 + countWidth(len(w.cells)) + uint64(len(w.cells))*cellRecLen + 4 + sessionLen); err != nil {
			return err
		}
	}
	return add(size & 1)
}
