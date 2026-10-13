//go:build windows

package pathdialog

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestWindowsFilters(t *testing.T) {
	specs := windowsFilters([]Filter{{Name: "Карты", Extensions: []string{"alm", "map"}}, {Name: "All files", Extensions: []string{"*"}}})
	if len(specs) != 2 || windows.UTF16PtrToString(specs[0].name) != "Карты" || windows.UTF16PtrToString(specs[0].pattern) != "*.alm;*.map" || windows.UTF16PtrToString(specs[1].pattern) != "*.*" {
		t.Fatalf("incorrect native filter marshalling: %+v", specs)
	}
}

func TestWindowsOptions(t *testing.T) {
	for _, test := range []struct {
		kind Kind
		want uint32
	}{{OpenFile, 0x1848}, {SaveFile, 0x1084a}, {ChooseFolder, 0x868}} {
		if got := windowsOptions(test.kind); got != test.want {
			t.Errorf("options(%d) = %#x, want %#x", test.kind, got, test.want)
		}
	}
}

func TestHRESULT(t *testing.T) {
	for _, success := range []uintptr{0, 1} {
		if err := hresult("COM", success); err != nil {
			t.Errorf("successful HRESULT %d: %v", success, err)
		}
	}
	if err := hresult("COM", 0x80004005); err == nil {
		t.Fatal("failed HRESULT accepted")
	}
}

func TestConfigureWindowsWithoutDialog(t *testing.T) {
	var slots [29]uintptr
	var options uint32
	var title, name, extension string
	var filterName, filterPattern string
	var selectedIndex uintptr
	slots[9] = windows.NewCallback(func(_ *comObject, flags uint32) uintptr { options = flags; return 0 })
	slots[17] = windows.NewCallback(func(_ *comObject, text *uint16) uintptr { title = windows.UTF16PtrToString(text); return 0 })
	slots[15] = windows.NewCallback(func(_ *comObject, text *uint16) uintptr { name = windows.UTF16PtrToString(text); return 0 })
	slots[22] = windows.NewCallback(func(_ *comObject, text *uint16) uintptr { extension = windows.UTF16PtrToString(text); return 0 })
	slots[5] = windows.NewCallback(func(_ *comObject, index uintptr) uintptr { selectedIndex = index; return 0 })
	slots[4] = windows.NewCallback(func(_ *comObject, count uint32, filter *filterSpec) uintptr {
		if count != 1 {
			return 0x80004005
		}
		filterName = windows.UTF16PtrToString(filter.name)
		filterPattern = windows.UTF16PtrToString(filter.pattern)
		return 0
	})
	dialog := comObject{vtable: &slots}
	request := Request{Kind: SaveFile, Title: "Сохранить", DefaultName: "карта.alm", Filters: []Filter{{Name: "Maps", Extensions: []string{"alm"}}}}
	if err := configureWindows(&dialog, request); err != nil {
		t.Fatal(err)
	}
	if options != 0x1084a || title != request.Title || name != request.DefaultName || extension != "alm" || filterName != "Maps" || filterPattern != "*.alm" || selectedIndex != 1 {
		t.Fatalf("native configuration = %#x %q %q %q %q %q %d", options, title, name, extension, filterName, filterPattern, selectedIndex)
	}
	slots[17] = windows.NewCallback(func(*comObject, *uint16) uintptr { return 0x80004005 })
	if err := configureWindows(&dialog, request); err == nil {
		t.Fatal("native configuration error was discarded")
	}
}
