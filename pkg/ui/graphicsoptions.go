package ui

import "againrom/pkg/render/terrain"

// GraphicsOptions are presentation preferences. The zero value preserves the
// previous renderer. They never enter the simulation or imported SAV carriers.
type GraphicsOptions struct {
	Smoothing, HideShadows, DisableLighting, StaticObjects bool
}

func (v *Viewer) SetGraphicsOptions(options GraphicsOptions) {
	if options.StaticObjects {
		options.DisableLighting = true // TOWN-OPTIONS-457's commit coupling.
	}
	v.graphics = options
}

func (v *Viewer) GraphicsOptions() GraphicsOptions { return v.graphics }

func (v *Viewer) SetSackBoundaries(frames []*terrain.StaticFrame) {
	v.sackBoundaries = make(map[*terrain.StaticFrame]*terrain.StaticFrame)
	for i, boundary := range frames {
		if i < len(v.sackFrames) && boundary != nil && v.sackFrames[i] != nil {
			v.sackBoundaries[v.sackFrames[i]] = boundary
		}
	}
}
