//go:build windows

package pathdialog

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ole32             = windows.NewLazySystemDLL("ole32.dll")
	coCreateInstance  = ole32.NewProc("CoCreateInstance")
	coInitializeEx    = ole32.NewProc("CoInitializeEx")
	createShellItem   = windows.NewLazySystemDLL("shell32.dll").NewProc("SHCreateItemFromParsingName")
	openDialogClass   = windows.GUID{Data1: 0xdc1c5a9c, Data2: 0xe88a, Data3: 0x4dde, Data4: [8]byte{0xa5, 0xa1, 0x60, 0xf8, 0x2a, 0x20, 0xae, 0xf7}}
	saveDialogClass   = windows.GUID{Data1: 0xc0b4e2f3, Data2: 0xba21, Data3: 0x4773, Data4: [8]byte{0x8d, 0xba, 0x33, 0x5e, 0xc9, 0x46, 0xeb, 0x8b}}
	fileDialogID      = windows.GUID{Data1: 0x42f85136, Data2: 0xdb7e, Data3: 0x439c, Data4: [8]byte{0x85, 0xf1, 0xe4, 0x07, 0x5d, 0x13, 0x5f, 0xc8}}
	shellItemID       = windows.GUID{Data1: 0x43826d1e, Data2: 0xe718, Data3: 0x42ee, Data4: [8]byte{0xbc, 0x55, 0xa1, 0xe2, 0x61, 0xc3, 0x7b, 0xfe}}
	findOwnerCallback = windows.NewCallback(findOwnerWindow)
)

type comObject struct {
	vtable *[29]uintptr
}

//go:uintptrescapes
func (object *comObject) call(method int, args ...uintptr) uintptr {
	args = append([]uintptr{uintptr(unsafe.Pointer(object))}, args...)
	result, _, _ := syscall.SyscallN(object.vtable[method], args...)
	runtime.KeepAlive(object)
	return result
}

func hresult(operation string, value uintptr) error {
	if int32(value) < 0 {
		return fmt.Errorf("%s: HRESULT 0x%08x", operation, uint32(value))
	}
	return nil
}

type ownerSearch struct {
	process uint32
	window  windows.HWND
}

func findOwnerWindow(window windows.HWND, parameter unsafe.Pointer) uintptr {
	search := (*ownerSearch)(parameter)
	var process uint32
	if _, err := windows.GetWindowThreadProcessId(window, &process); err != nil || process != search.process || !windows.IsWindowVisible(window) {
		return 1
	}
	var class [64]uint16
	if _, err := windows.GetClassName(window, &class[0], int32(len(class))); err == nil && windows.UTF16ToString(class[:]) == "GLFW30" {
		search.window = window
		return 0
	}
	return 1
}

func dialogOwner() (windows.HWND, error) {
	search := ownerSearch{process: windows.GetCurrentProcessId()}
	err := windows.EnumWindows(findOwnerCallback, unsafe.Pointer(&search))
	if search.window != 0 {
		return search.window, nil
	}
	if err != nil {
		return 0, fmt.Errorf("%w: find program window: %v", ErrUnavailable, err)
	}
	return 0, fmt.Errorf("%w: no visible program window", ErrUnavailable)
}

func windowsOptions(kind Kind) uint32 {
	const forceFilesystem = 0x40
	const noChangeDir = 0x8
	const pathMustExist = 0x800
	options := uint32(forceFilesystem | noChangeDir | pathMustExist)
	switch kind {
	case OpenFile:
		options |= 0x1000
	case SaveFile:
		// The dialog must not create a probe file before the caller's install fence.
		options |= 0x2 | 0x10000
	case ChooseFolder:
		options |= 0x20
	}
	return options
}

type filterSpec struct {
	name    *uint16
	pattern *uint16
}

func windowsFilters(filters []Filter) []filterSpec {
	specs := make([]filterSpec, 0, len(filters))
	for _, filter := range filters {
		patterns := make([]string, len(filter.Extensions))
		for i, extension := range filter.Extensions {
			if extension == "*" {
				patterns[i] = "*.*"
			} else {
				patterns[i] = "*." + extension
			}
		}
		name, _ := windows.UTF16PtrFromString(filter.Name)
		pattern, _ := windows.UTF16PtrFromString(strings.Join(patterns, ";"))
		specs = append(specs, filterSpec{name, pattern})
	}
	return specs
}

func setDialogText(dialog *comObject, method int, text string) error {
	if text == "" {
		return nil
	}
	value, err := windows.UTF16PtrFromString(text)
	if err != nil {
		return err
	}
	err = hresult("configure dialog text", dialog.call(method, uintptr(unsafe.Pointer(value))))
	runtime.KeepAlive(value)
	return err
}

func configureWindows(dialog *comObject, request Request) error {
	if err := hresult("set dialog options", dialog.call(9, uintptr(windowsOptions(request.Kind)))); err != nil {
		return err
	}
	if err := setDialogText(dialog, 17, request.Title); err != nil {
		return err
	}
	if err := setDialogText(dialog, 15, request.DefaultName); err != nil {
		return err
	}
	if request.Kind != ChooseFolder && len(request.Filters) > 0 {
		specs := windowsFilters(request.Filters)
		err := hresult("set dialog filters", dialog.call(4, uintptr(len(specs)), uintptr(unsafe.Pointer(&specs[0]))))
		runtime.KeepAlive(specs)
		if err != nil {
			return err
		}
		if err := hresult("select dialog filter", dialog.call(5, 1)); err != nil {
			return err
		}
		if request.Kind == SaveFile && request.Filters[0].Extensions[0] != "*" {
			if err := setDialogText(dialog, 22, request.Filters[0].Extensions[0]); err != nil {
				return err
			}
		}
	}
	if request.StartDir != "" {
		path, _ := windows.UTF16PtrFromString(request.StartDir)
		var folder *comObject
		hr, _, _ := createShellItem.Call(uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(&shellItemID)), uintptr(unsafe.Pointer(&folder)))
		runtime.KeepAlive(path)
		if err := hresult("resolve start folder", hr); err != nil {
			return err
		}
		if folder == nil {
			return fmt.Errorf("system path dialog returned no start folder")
		}
		defer folder.call(2)
		if err := hresult("set start folder", dialog.call(12, uintptr(unsafe.Pointer(folder)))); err != nil {
			return err
		}
	}
	return nil
}

func show(request Request) (Result, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for _, proc := range []*windows.LazyProc{coInitializeEx, coCreateInstance, createShellItem} {
		if err := proc.Find(); err != nil {
			return Result{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
	}
	// COM and its native modal message loop stay on this STA worker.
	hr, _, _ := coInitializeEx.Call(0, 2)
	if err := hresult("initialize dialog apartment", hr); err != nil {
		return Result{}, err
	}
	defer windows.CoUninitialize()
	owner, err := dialogOwner()
	if err != nil {
		return Result{}, err
	}
	class := &openDialogClass
	if request.Kind == SaveFile {
		class = &saveDialogClass
	}
	var dialog *comObject
	hr, _, _ = coCreateInstance.Call(uintptr(unsafe.Pointer(class)), 0, 1, uintptr(unsafe.Pointer(&fileDialogID)), uintptr(unsafe.Pointer(&dialog)))
	if err := hresult("create system path dialog", hr); err != nil {
		return Result{}, err
	}
	if dialog == nil {
		return Result{}, fmt.Errorf("system path dialog returned no interface")
	}
	defer dialog.call(2)
	if err := configureWindows(dialog, request); err != nil {
		return Result{}, err
	}
	hr = dialog.call(3, uintptr(owner))
	if uint32(hr) == 0x800704c7 {
		return selected("", true)
	}
	if err := hresult("show system path dialog", hr); err != nil {
		return Result{}, err
	}
	var item *comObject
	if err := hresult("get selected item", dialog.call(20, uintptr(unsafe.Pointer(&item)))); err != nil {
		return Result{}, err
	}
	if item == nil {
		return Result{}, fmt.Errorf("system path dialog returned no item")
	}
	defer item.call(2)
	var path *uint16
	if err := hresult("get selected path", item.call(5, 0x80058000, uintptr(unsafe.Pointer(&path)))); err != nil {
		return Result{}, err
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(path))
	return selected(windows.UTF16PtrToString(path), false)
}
