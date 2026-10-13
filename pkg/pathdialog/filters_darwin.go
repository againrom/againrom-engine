//go:build darwin

package pathdialog

import (
	"sync"

	"github.com/ebitengine/purego/objc"
)

var (
	filterTargetOnce  sync.Once
	filterTargetClass objc.Class
	filterTargetError error
	filterActions     = make(map[objc.ID]func(objc.ID))
)

func macFilterClass() (objc.Class, error) {
	filterTargetOnce.Do(func() {
		filterTargetClass, filterTargetError = objc.RegisterClass("AGRMPathDialogFilterTarget", objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{{
			Cmd: objc.RegisterName("selectFilter:"),
			Fn: func(target objc.ID, _ objc.SEL, sender objc.ID) {
				if action := filterActions[target]; action != nil {
					action(sender)
				}
			},
		}})
	})
	return filterTargetClass, filterTargetError
}

func applyMacFilter(panel objc.ID, filter Filter) {
	if len(filter.Extensions) == 1 && filter.Extensions[0] == "*" {
		send(panel, "setAllowedFileTypes:", objc.ID(0))
		send(panel, "setAllowsOtherFileTypes:", true)
		return
	}
	types := classSend("NSMutableArray", "array")
	for _, extension := range filter.Extensions {
		send(types, "addObject:", macString(extension))
	}
	send(panel, "setAllowedFileTypes:", types)
	send(panel, "setAllowsOtherFileTypes:", false)
}

type macRect struct {
	x, y, width, height float64
}

func configureMacFilters(panel objc.ID, request Request) (func(), error) {
	if request.Kind == ChooseFolder || len(request.Filters) == 0 {
		return func() {}, nil
	}
	applyMacFilter(panel, request.Filters[0])
	if len(request.Filters) == 1 {
		return func() {}, nil
	}
	class, err := macFilterClass()
	if err != nil {
		return nil, err
	}
	target := send(objc.ID(class), "new")
	popup := classSend("NSPopUpButton", "alloc")
	popup = send(popup, "initWithFrame:pullsDown:", macRect{width: 240, height: 28}, false)
	for _, filter := range request.Filters {
		send(popup, "addItemWithTitle:", macString(filter.Name))
	}
	send(popup, "selectItemAtIndex:", 0)
	send(popup, "sizeToFit")
	send(popup, "setTarget:", target)
	send(popup, "setAction:", objc.RegisterName("selectFilter:"))
	filterActions[target] = func(sender objc.ID) {
		index := int(send(sender, "indexOfSelectedItem"))
		if index >= 0 && index < len(request.Filters) {
			applyMacFilter(panel, request.Filters[index])
		}
	}
	send(panel, "setAccessoryView:", popup)
	if request.Kind == OpenFile {
		send(panel, "setAccessoryViewDisclosed:", true)
	}
	return func() {
		send(popup, "setTarget:", objc.ID(0))
		send(panel, "setAccessoryView:", objc.ID(0))
		delete(filterActions, target)
		send(popup, "release")
		send(target, "release")
	}, nil
}
