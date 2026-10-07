package game

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unsafe"
)

// memoryReach records every pointer target, slice backing array and map the
// value v reaches, keyed by address and type. It reads unexported fields, and
// does not look inside slices of pointer-free elements (their backing array
// is recorded, their bytes cannot reach more memory).
type memoryReach struct {
	seen  map[reach]string
	stops map[reflect.Type]bool
}

type reach struct {
	addr uintptr
	typ  reflect.Type
}

func newMemoryReach() *memoryReach {
	return &memoryReach{seen: map[reach]string{}, stops: map[reflect.Type]bool{}}
}

func pointerFree(t reflect.Type, memo map[reflect.Type]bool) bool {
	if v, ok := memo[t]; ok {
		return v
	}
	memo[t] = true
	free := false
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		free = true
	case reflect.Array:
		free = pointerFree(t.Elem(), memo)
	case reflect.Struct:
		free = true
		for i := 0; i < t.NumField(); i++ {
			if !pointerFree(t.Field(i).Type, memo) {
				free = false
				break
			}
		}
	}
	memo[t] = free
	return free
}

func (m *memoryReach) walk(v reflect.Value, path string, memo map[reflect.Type]bool) {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		key := reach{v.Pointer(), v.Type()}
		if _, ok := m.seen[key]; ok {
			return
		}
		m.seen[key] = path
		if m.stops[v.Type()] {
			return
		}
		m.walk(v.Elem(), path+"*", memo)
	case reflect.Interface:
		if v.IsNil() {
			return
		}
		e := v.Elem()
		// An interface holds its dynamic value by copy; copy it to something
		// addressable so its own pointers can be followed.
		c := reflect.New(e.Type()).Elem()
		c.Set(e)
		m.walk(c, path+"."+e.Type().String(), memo)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if !f.CanAddr() {
				c := reflect.New(f.Type()).Elem()
				c.Set(f)
				f = c
			}
			f = reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
			m.walk(f, path+"."+v.Type().Field(i).Name, memo)
		}
	case reflect.Slice:
		if v.IsNil() || v.Cap() == 0 {
			return
		}
		key := reach{v.Pointer(), reflect.ArrayOf(v.Cap(), v.Type().Elem())}
		if _, ok := m.seen[key]; ok {
			return
		}
		m.seen[key] = path
		if pointerFree(v.Type().Elem(), memo) {
			return
		}
		for i := 0; i < v.Len(); i++ {
			m.walk(v.Index(i), fmt.Sprintf("%s[]", path), memo)
		}
	case reflect.Array:
		if pointerFree(v.Type().Elem(), memo) {
			return
		}
		for i := 0; i < v.Len(); i++ {
			m.walk(v.Index(i), path+"[]", memo)
		}
	case reflect.Map:
		if v.IsNil() {
			return
		}
		key := reach{v.Pointer(), v.Type()}
		if _, ok := m.seen[key]; ok {
			return
		}
		m.seen[key] = path
		it := v.MapRange()
		for it.Next() {
			k := reflect.New(it.Key().Type()).Elem()
			k.Set(it.Key())
			m.walk(k, path+".key", memo)
			e := reflect.New(it.Value().Type()).Elem()
			e.Set(it.Value())
			m.walk(e, path+"[]", memo)
		}
	}
}

func reachOf(x any, skipFields ...string) *memoryReach {
	m := newMemoryReach()
	v := reflect.ValueOf(x)
	memo := map[reflect.Type]bool{}
	if v.Kind() == reflect.Pointer && v.Elem().Kind() == reflect.Struct && len(skipFields) != 0 {
		// Walk the struct's fields one by one so named ones can be skipped.
		s := v.Elem()
		skip := map[string]bool{}
		for _, n := range skipFields {
			skip[n] = true
		}
		for i := 0; i < s.NumField(); i++ {
			if skip[s.Type().Field(i).Name] {
				continue
			}
			f := reflect.NewAt(s.Field(i).Type(), unsafe.Pointer(s.Field(i).UnsafeAddr())).Elem()
			m.walk(f, s.Type().Field(i).Name, memo)
		}
		return m
	}
	m.walk(v, "", memo)
	return m
}

func sharedMemory(a, b, except *memoryReach) []string {
	var out []string
	for k, path := range a.seen {
		if except != nil {
			if _, ok := except.seen[k]; ok {
				continue
			}
		}
		if other, ok := b.seen[k]; ok {
			out = append(out, fmt.Sprintf("%s (%s) <-> %s", path, k.typ, other))
		}
	}
	sort.Strings(out)
	return out
}

// liveGame is everything the frame thread may write after a capture: the whole
// front end except the install tables, which are read-only after launch.
func liveGame(f *FrontEnd) *memoryReach {
	m := newMemoryReach()
	memo := map[reflect.Type]bool{}
	for _, part := range []any{&f.RuntimeServices, &f.CampaignSession, &f.Presentation, &f.PersistenceContext} {
		m.walk(reflect.ValueOf(part), "", memo)
	}
	return m
}

// readOnlyShared is the memory a capture shares with the live game by design,
// because nothing writes it after it is built: the mission's decoded map, the
// loaded world-map registry and the town's group membership (replaced, never
// edited in place).
func readOnlyShared(f *FrontEnd) *memoryReach {
	m := newMemoryReach()
	memo := map[reflect.Type]bool{}
	if f.live != nil && f.live.mission != nil && f.live.mission.state != nil {
		m.walk(reflect.ValueOf(f.live.mission.state.Map), "", memo)
	}
	if a := f.worldMapCache.Value(); a != nil {
		m.walk(reflect.ValueOf(a.data), "", memo)
	}
	if f.Town != nil {
		m.walk(reflect.ValueOf(&f.Town.cityGroups), "", memo)
	}
	return m
}

// control is a value that does hold live memory: the detector must report it.
func checkNoSharedWritableMemory(t *testing.T, f *FrontEnd, s Snapshot, control any) {
	t.Helper()
	if n := len(sharedMemory(reachOf(control), liveGame(f), nil)); n < 10 {
		t.Fatalf("loss control: the detector found only %d regions shared with a live value", n)
	}
	view, detached := f.detachedExporter(s)
	live, readOnly := liveGame(f), readOnlyShared(f)
	for _, c := range []struct {
		name string
		got  *memoryReach
	}{
		{"snapshot", reachOf(&detached)},
		{"detached front end", reachOf(view, "InstallResources")},
	} {
		if bad := sharedMemory(c.got, live, readOnly); len(bad) != 0 {
			t.Errorf("%s shares %d writable regions with the live game: %s", c.name, len(bad), strings.Join(bad, "; "))
		}
	}
}

func TestCapturedMissionSharesNoWritableMemoryWithTheLiveGame(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	app := f.App("alias mission")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		f.live.tick()
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	checkNoSharedWritableMemory(t, f, s, &struct{ W *mapWorld }{f.live})
}

func TestCapturedTownSharesNoWritableMemoryWithTheLiveGame(t *testing.T) {
	f := currentTrainingCity(t)
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	checkNoSharedWritableMemory(t, f, s, &struct{ T *Town }{f.Town})
}
