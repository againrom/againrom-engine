//go:build !windows

package widgetkit

import "github.com/hajimehoshi/ebiten/v2"

// platformDesktop is the monitor Ebitengine reports; a desktop other than
// Windows gives no work area or frame size to read without cgo.
func platformDesktop() Desktop {
	m := ebiten.Monitor()
	if m == nil {
		return Desktop{}
	}
	w, h := m.Size()
	return Desktop{Work: WorkArea{W: w, H: h}}
}
