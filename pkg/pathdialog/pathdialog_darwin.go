//go:build darwin

package pathdialog

import (
	"fmt"
	"runtime"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

func show(request Request) (Result, error) {
	return waitForPanel(enqueueMain, func(finish func(Result, error)) {
		beginMacPanel(request, finish)
	})
}

func enqueueMain(fn func()) error {
	queue, err := purego.Dlsym(purego.RTLD_DEFAULT, "_dispatch_main_q")
	if err != nil {
		return fmt.Errorf("%w: main queue: %v", ErrUnavailable, err)
	}
	address, err := purego.Dlsym(purego.RTLD_DEFAULT, "dispatch_async")
	if err != nil {
		return fmt.Errorf("%w: main dispatch: %v", ErrUnavailable, err)
	}
	var dispatch func(uintptr, objc.Block)
	purego.RegisterFunc(&dispatch, address)
	block := objc.NewBlock(func(objc.Block) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		fn()
	})
	defer block.Release()
	dispatch(queue, block)
	return nil
}

func send(object objc.ID, selector string, args ...any) objc.ID {
	return object.Send(objc.RegisterName(selector), args...)
}

func classSend(class, selector string, args ...any) objc.ID {
	return send(objc.ID(objc.GetClass(class)), selector, args...)
}

func macString(text string) objc.ID {
	return classSend("NSString", "stringWithUTF8String:", text)
}

func beginMacPanel(request Request, finish func(Result, error)) {
	if !objc.Send[bool](objc.ID(objc.GetClass("NSThread")), objc.RegisterName("isMainThread")) {
		finish(Result{}, fmt.Errorf("system path panel is not on the main thread"))
		return
	}
	pool := classSend("NSAutoreleasePool", "new")
	defer send(pool, "drain")
	app := classSend("NSApplication", "sharedApplication")
	owner := send(app, "mainWindow")
	if owner == 0 {
		owner = send(app, "keyWindow")
	}
	if owner == 0 {
		finish(Result{}, fmt.Errorf("%w: no program window", ErrUnavailable))
		return
	}
	class, selector := "NSOpenPanel", "openPanel"
	if request.Kind == SaveFile {
		class, selector = "NSSavePanel", "savePanel"
	}
	panel := classSend(class, selector)
	if panel == 0 {
		finish(Result{}, fmt.Errorf("%w: no system path panel", ErrUnavailable))
		return
	}
	send(panel, "retain")
	send(panel, "setCanCreateDirectories:", false)
	if request.Title != "" {
		send(panel, "setTitle:", macString(request.Title))
	}
	if request.StartDir != "" {
		send(panel, "setDirectoryURL:", classSend("NSURL", "fileURLWithPath:", macString(request.StartDir)))
	}
	if request.DefaultName != "" {
		send(panel, "setNameFieldStringValue:", macString(request.DefaultName))
	}
	if request.Kind != SaveFile {
		send(panel, "setCanChooseFiles:", request.Kind == OpenFile)
		send(panel, "setCanChooseDirectories:", request.Kind == ChooseFolder)
		send(panel, "setAllowsMultipleSelection:", false)
	}
	cleanup, err := configureMacFilters(panel, request)
	if err != nil {
		send(panel, "release")
		finish(Result{}, err)
		return
	}
	completion := objc.NewBlock(func(_ objc.Block, response int64) {
		defer send(panel, "release")
		defer cleanup()
		defer send(panel, "orderOut:", objc.ID(0))
		if response != 0 && response != 1 {
			finish(Result{}, fmt.Errorf("system path panel response %d", response))
			return
		}
		path := ""
		if response == 1 {
			url := send(panel, "URL")
			if url != 0 && objc.Send[bool](url, objc.RegisterName("isFileURL")) {
				value := send(url, "path")
				path = objc.Send[string](value, objc.RegisterName("UTF8String"))
			}
		}
		result, err := selected(path, response == 0)
		finish(result, err)
	})
	defer completion.Release()
	// A sheet returns immediately; Ebitengine keeps pumping AppKit events.
	send(panel, "beginSheetModalForWindow:completionHandler:", owner, completion)
}
