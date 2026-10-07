package ui

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"slices"

	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
	"github.com/hajimehoshi/ebiten/v2"
)

// DialogueBackdrop selects packed lookup and clipping.
type DialogueBackdrop struct {
	Layout       backdrop.Layout
	Mode         backdrop.Mode
	Clip         image.Rectangle
	Clipped      bool
	FrameClip    image.Rectangle
	FrameClipped bool
}

type dialogueBackdropState struct {
	policy            DialogueBackdrop
	policySet         bool
	appliedViewer     *Viewer
	shows             uint64
	lookup            *backdrop.Lookup
	texture           *ebiten.Image
	scratch           *ebiten.Image
	shadowLookup      *backdrop.Lookup
	shadowTexture     *ebiten.Image
	shadowMask        *image.RGBA
	shadowMaskTexture *ebiten.Image
}

func (s *dialogueBackdropState) shadowTable() *backdrop.Lookup {
	if s.shadowLookup == nil || s.shadowLookup.Layout != s.policy.Layout || s.shadowLookup.Mode != s.policy.Mode {
		var err error
		s.shadowLookup, err = backdrop.NewLevel(s.policy.Layout, s.policy.Mode, 6)
		if err != nil {
			panic(err)
		}
		if s.shadowTexture != nil {
			s.shadowTexture.Dispose()
			s.shadowTexture = nil
		}
	}
	return s.shadowLookup
}

func remapCapturedMask(calls []text.DrawCall, r image.Rectangle, l *backdrop.Lookup, mask *image.RGBA, at image.Point, counts bool, logs ...*pixelLog) {
	for i := range calls {
		c := &calls[i]
		if c.Glyph == nil || c.Glyph.Width <= 0 {
			continue
		}
		original := *c
		c.RasterColors = make([]color.RGBA, len(c.Glyph.Pixels))
		c.Under = slices.Clone(c.Under)
		for n, p := range c.Glyph.Pixels {
			ink := original.NativeColor(p.Level, n)
			point := image.Pt(c.X+n%c.Glyph.Width, c.Y+n/c.Glyph.Width)
			visible := p.Painted && (c.Clip.Empty() || point.In(c.Clip))
			if visible {
				if n < len(c.Under) {
					c.Under[n] = text.Tinted(original.Tint, c.Under[n])
				}
				if len(logs) > 0 && logs[0] != nil {
					base := original
					base.Tint = color.RGBA{}
					untinted := base.NativeColor(p.Level, n)
					if tint, verdict := logs[0].tint(point.X, point.Y, untinted); verdict == textsmooth.Matches {
						ink = text.Tinted(tint, untinted)
						if n < len(c.Under) {
							c.Under[n] = text.Tinted(tint, original.Under[n])
						}
					} else if tint, verdict := logs[0].tint(point.X, point.Y, ink); verdict == textsmooth.Matches {
						ink = text.Tinted(tint, ink)
						if n < len(c.Under) {
							c.Under[n] = text.Tinted(tint, c.Under[n])
						}
					}
				}
			}
			if visible && point.In(r) && dialogueMaskAt(mask, at, point.X, point.Y) {
				count := uint8(1)
				if counts {
					count = mask.RGBAAt(mask.Rect.Min.X+point.X-at.X, mask.Rect.Min.Y+point.Y-at.Y).A
				}
				for pass := uint8(0); pass < count; pass++ {
					ink = l.Color(ink)
					if n < len(c.Under) {
						c.Under[n] = l.Color(c.Under[n])
					}
				}
			}
			c.RasterColors[n] = ink
		}
		c.Tint, c.SourceOver, c.Flat = color.RGBA{}, false, true
	}
}

const dialogueShadowShader = `//kage:unit pixels
package main
var GreenBits float
var MaskAt vec2
func Fragment(dst vec4, src vec2, colour vec4) vec4 {
	c := imageSrc0At(src)
	count := floor(imageSrc2At(src-MaskAt).a*255.0+0.5)
	for i := 0; i < 255; i++ {
		if float(i) >= count { break }
		b := floor(c.rgb*255.0+0.5)
		g := floor(b.y/4.0)
		rshift := 2048.0
		if GreenBits == 5.0 { g = floor(b.y/8.0); rshift = 1024.0 }
		p := floor(b.x/8.0)*rshift+g*32.0+floor(b.z/8.0)
		at := vec2(mod(p,256.0),floor(p/256.0))+imageSrc0Origin()+vec2(0.5)
		c = vec4(imageSrc1At(at).rgb,c.a)
	}
	return c
}`

var dialogueShadowProgram *ebiten.Shader

var submitDialogueShadow = func(dst *ebiten.Image, vertices []ebiten.Vertex, shader *ebiten.Shader, op *ebiten.DrawTrianglesShaderOptions) {
	dst.DrawTrianglesShader(vertices, []uint16{0, 1, 2, 1, 2, 3}, shader, op)
}

func dialogueShadowMask(art *DialogFrame, body image.Rectangle, at image.Point, scale float64, clip image.Rectangle) *image.RGBA {
	if !art.valid() || scale <= 0 {
		return nil
	}
	tiles := art.dialogueShadows(body)
	transform := func(r image.Rectangle) image.Rectangle {
		return image.Rect(at.X+int(math.Floor(float64(r.Min.X)*scale)), at.Y+int(math.Floor(float64(r.Min.Y)*scale)), at.X+int(math.Ceil(float64(r.Max.X)*scale)), at.Y+int(math.Ceil(float64(r.Max.Y)*scale)))
	}
	var bounds image.Rectangle
	for _, tile := range tiles {
		bounds = bounds.Union(transform(image.Rectangle{Min: tile.at, Max: tile.at.Add(art.Pieces[tile.piece].Bounds().Size())}))
	}
	bounds = bounds.Intersect(clip)
	if bounds.Empty() {
		return nil
	}
	mask := image.NewRGBA(bounds)
	for _, tile := range tiles {
		pic := art.Pieces[tile.piece]
		r := transform(image.Rectangle{Min: tile.at, Max: tile.at.Add(pic.Bounds().Size())}).Intersect(bounds)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				sx := int(math.Floor((float64(x-at.X)+0.5)/scale)) - tile.at.X
				sy := int(math.Floor((float64(y-at.Y)+0.5)/scale)) - tile.at.Y
				if sx < 0 || sy < 0 || sx >= pic.Rect.Dx() || sy >= pic.Rect.Dy() || pic.RGBAAt(pic.Rect.Min.X+sx, pic.Rect.Min.Y+sy).A == 0 {
					continue
				}
				i := mask.PixOffset(x, y) + 3
				if mask.Pix[i] == 255 {
					panic("dialogue shadow mask overlap exceeds 255")
				}
				mask.Pix[i]++
			}
		}
	}
	return mask
}

func (s *dialogueBackdropState) applyFrameShadows(dst *image.RGBA, art *DialogFrame, body image.Rectangle, at image.Point, calls []text.DrawCall) {
	if dst == nil {
		return
	}
	mask := dialogueShadowMask(art, body, at, 1, dialoguePolicyClip(s.policy, dst.Bounds()))
	if mask == nil {
		return
	}
	l := s.shadowTable()
	for y := mask.Rect.Min.Y; y < mask.Rect.Max.Y; y++ {
		for x := mask.Rect.Min.X; x < mask.Rect.Max.X; x++ {
			c := dst.RGBAAt(x, y)
			for n := uint8(0); n < mask.RGBAAt(x, y).A; n++ {
				c = l.Color(c)
			}
			dst.SetRGBA(x, y, c)
		}
	}
	remapCapturedMask(calls, mask.Rect, l, mask, mask.Rect.Min, true)
}

func (s *dialogueBackdropState) drawFrameShadows(dst *ebiten.Image, log *pixelLog, art *DialogFrame, body image.Rectangle, at image.Point, scale float64, calls []text.DrawCall) {
	if dst == nil {
		return
	}
	mask := dialogueShadowMask(art, body, at, scale, dialoguePolicyClip(s.policy, dst.Bounds()))
	if mask == nil {
		s.shadowMask = nil
		return
	}
	l := s.shadowTable()
	if dialogueShadowProgram == nil {
		var err error
		dialogueShadowProgram, err = ebiten.NewShader([]byte(dialogueShadowShader))
		if err != nil {
			panic(err)
		}
	}
	if s.shadowTexture == nil {
		s.shadowTexture = ebiten.NewImageFromImage(l.Texture())
	}
	if s.shadowMaskTexture == nil || s.shadowMaskTexture.Bounds().Size() != mask.Rect.Size() {
		if s.shadowMaskTexture != nil {
			s.shadowMaskTexture.Dispose()
		}
		s.shadowMaskTexture = ebiten.NewImageFromImage(mask)
	} else if s.shadowMask == nil || !bytes.Equal(s.shadowMask.Pix, mask.Pix) {
		s.shadowMaskTexture.WritePixels(mask.Pix)
	}
	s.shadowMask = mask
	b := dst.Bounds()
	if s.scratch == nil || s.scratch.Bounds().Size() != b.Size() {
		if s.scratch != nil {
			s.scratch.Dispose()
		}
		s.scratch = ebiten.NewImage(b.Dx(), b.Dy())
	}
	var copyOp ebiten.DrawImageOptions
	copyOp.Blend = ebiten.BlendCopy
	s.scratch.DrawImage(dst, &copyOp)
	bits := float32(6)
	if s.policy.Layout == backdrop.RGB555 {
		bits = 5
	}
	op := ebiten.DrawTrianglesShaderOptions{Blend: ebiten.BlendCopy, Images: [4]*ebiten.Image{s.scratch, s.shadowTexture, s.shadowMaskTexture}, Uniforms: map[string]any{"GreenBits": bits, "MaskAt": []float32{float32(mask.Rect.Min.X), float32(mask.Rect.Min.Y)}}}
	vertices := []ebiten.Vertex{{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0}, {DstX: float32(b.Dx()), DstY: 0, SrcX: float32(b.Dx()), SrcY: 0}, {DstX: 0, DstY: float32(b.Dy()), SrcX: 0, SrcY: float32(b.Dy())}, {DstX: float32(b.Dx()), DstY: float32(b.Dy()), SrcX: float32(b.Dx()), SrcY: float32(b.Dy())}}
	submitDialogueShadow(dst.SubImage(mask.Rect).(*ebiten.Image), vertices, dialogueShadowProgram, &op)
	remapCapturedMask(calls, mask.Rect, l, mask, mask.Rect.Min, true, log)
	log.remapMask(mask.Rect, l, mask, mask.Rect.Min)
}

func (s *dialogueBackdropState) table() *backdrop.Lookup {
	if s.lookup == nil || s.lookup.Layout != s.policy.Layout || s.lookup.Mode != s.policy.Mode {
		var err error
		s.lookup, err = backdrop.New(s.policy.Layout, s.policy.Mode)
		if err != nil {
			panic(err)
		}
		if s.texture != nil {
			s.texture.Dispose()
			s.texture = nil
		}
	}
	return s.lookup
}

func (s *dialogueBackdropState) rect(bounds image.Rectangle) image.Rectangle {
	if s.policy.Clipped {
		return bounds.Intersect(s.policy.Clip)
	}
	return bounds
}

func (s *dialogueBackdropState) apply(pic *image.RGBA, shows uint64) {
	if shows == 0 || pic == nil {
		return
	}
	s.table().Apply(pic, pic.Bounds(), s.rect(pic.Bounds()), shows)
}

func (a *App) SetDialogueBackdrop(p DialogueBackdrop) error {
	if _, err := backdrop.New(p.Layout, p.Mode); err != nil {
		return err
	}
	a.dialogueBackdrop.policy = p
	a.dialogueBackdrop.policySet = true
	if a.flow != nil {
		if town, ok := a.flow.town.(interface {
			TownDialogueVisual(DialogueButtonState, DialogueBackdrop)
		}); ok {
			state := DialogueButtonState{}
			if current, ok := a.flow.town.(interface{ TownDialogueButtonState() DialogueButtonState }); ok {
				state = current.TownDialogueButtonState()
			}
			town.TownDialogueVisual(state, p)
		}
	}
	if a.flow.viewer != nil {
		a.dialogueBackdrop.appliedViewer = a.flow.viewer
		return a.flow.viewer.SetDialogueBackdrop(p)
	}
	return nil
}

func (v *Viewer) SetDialogueBackdrop(p DialogueBackdrop) error {
	if _, err := backdrop.New(p.Layout, p.Mode); err != nil {
		return err
	}
	v.dialogueBackdrop.policy = p
	v.noticeBackdropCustom = false
	return nil
}

func townDialogueShows(t TownScreen) uint64 {
	if _, onMap := townWorldMapScreen(t); onMap {
		return 0
	}
	if state, ok := t.(interface{ TownDialogueShows() uint64 }); ok {
		return state.TownDialogueShows()
	}
	return 1
}

// DialogueBackdropPlan reports the GPU policy.
func (v *Viewer) DialogueBackdropPlan() (DialogueBackdrop, uint64, image.Rectangle) {
	if !v.noticeOpen || v.noticeKind != NoticeDialogue || v.font == nil || v.noticeBackdropCustom {
		return v.dialogueBackdrop.policy, 0, image.Rectangle{}
	}
	b := image.Rect(0, 0, v.frameW, v.frameH)
	return v.dialogueBackdrop.policy, v.dialogueBackdrop.shows, v.dialogueBackdrop.rect(b)
}

func (a *App) DialogueBackdropPixel(x, y int) (color.RGBA, bool) {
	if a.flow.mapShowing() && a.flow.viewer != nil {
		return a.flow.viewer.DialogueBackdropPixel(x, y)
	}
	return a.presentLog.value(len(a.presentLog.ops), x, y)
}

func (v *Viewer) DialogueBackdropPixel(x, y int) (color.RGBA, bool) {
	return v.canvasLog.value(len(v.canvasLog.ops), x, y)
}

func remapCaptured(calls []text.DrawCall, r image.Rectangle, l *backdrop.Lookup, shows uint64) {
	for i := range calls {
		c := &calls[i]
		if c.Glyph == nil || c.Glyph.Width <= 0 {
			continue
		}
		original := *c
		c.RasterColors = make([]color.RGBA, len(c.Glyph.Pixels))
		c.Under = slices.Clone(c.Under)
		for n, p := range c.Glyph.Pixels {
			ink := original.NativeColor(p.Level, n)
			at := image.Pt(c.X+n%c.Glyph.Width, c.Y+n/c.Glyph.Width)
			if at.In(r) {
				for pass := uint64(0); pass < shows; pass++ {
					next := l.Color(ink)
					if next == ink {
						break
					}
					ink = next
				}
				if n < len(c.Under) {
					for pass := uint64(0); pass < shows; pass++ {
						next := l.Color(c.Under[n])
						if next == c.Under[n] {
							break
						}
						c.Under[n] = next
					}
				}
			}
			c.RasterColors[n] = ink
		}
		c.Tint, c.SourceOver, c.Flat = color.RGBA{}, false, true
	}
}

const dialogueBackdropShader = `//kage:unit pixels
package main
var GreenBits float
func Fragment(dst vec4, src vec2, colour vec4) vec4 {
	c := imageSrc0At(src)
	b := floor(c.rgb*255.0+0.5)
	g := floor(b.y/4.0)
	rshift := 2048.0
	if GreenBits == 5.0 { g = floor(b.y/8.0); rshift = 1024.0 }
	p := floor(b.x/8.0)*rshift+g*32.0+floor(b.z/8.0)
	at := vec2(mod(p,256.0),floor(p/256.0))+imageSrc0Origin()+vec2(0.5)
	return vec4(imageSrc1At(at).rgb,c.a)
}`

var dialogueBackdropProgram *ebiten.Shader

var submitDialogueBackdrop = func(dst *ebiten.Image, vertices []ebiten.Vertex, shader *ebiten.Shader, op *ebiten.DrawTrianglesShaderOptions) {
	dst.DrawTrianglesShader(vertices, []uint16{0, 1, 2, 1, 2, 3}, shader, op)
}

func (s *dialogueBackdropState) draw(dst *ebiten.Image, log *pixelLog, shows uint64) {
	s.drawWithCalls(dst, log, shows, text.Captured())
}

func (s *dialogueBackdropState) drawWithCalls(dst *ebiten.Image, log *pixelLog, shows uint64, calls []text.DrawCall) {
	if shows == 0 || dst == nil {
		return
	}
	l := s.table()
	r := s.rect(dst.Bounds())
	if r.Empty() {
		return
	}
	if dialogueBackdropProgram == nil {
		var err error
		dialogueBackdropProgram, err = ebiten.NewShader([]byte(dialogueBackdropShader))
		if err != nil {
			panic(err)
		}
	}
	if s.texture == nil {
		s.texture = ebiten.NewImageFromImage(l.Texture())
	}
	b := dst.Bounds()
	if s.scratch == nil || s.scratch.Bounds().Size() != b.Size() {
		if s.scratch != nil {
			s.scratch.Dispose()
		}
		s.scratch = ebiten.NewImage(b.Dx(), b.Dy())
	}
	bits := float32(6)
	if s.policy.Layout == backdrop.RGB555 {
		bits = 5
	}
	// Every packed channel reaches a fixed point within 32 applications.
	for pass := uint64(0); pass < min(shows, 32); pass++ {
		var copyOp ebiten.DrawImageOptions
		copyOp.Blend = ebiten.BlendCopy
		s.scratch.DrawImage(dst, &copyOp)
		op := ebiten.DrawTrianglesShaderOptions{Blend: ebiten.BlendCopy, Images: [4]*ebiten.Image{s.scratch, s.texture}, Uniforms: map[string]any{"GreenBits": bits}}
		vertices := []ebiten.Vertex{{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0}, {DstX: float32(b.Dx()), DstY: 0, SrcX: float32(b.Dx()), SrcY: 0}, {DstX: 0, DstY: float32(b.Dy()), SrcX: 0, SrcY: float32(b.Dy())}, {DstX: float32(b.Dx()), DstY: float32(b.Dy()), SrcX: float32(b.Dx()), SrcY: float32(b.Dy())}}
		submitDialogueBackdrop(dst.SubImage(r).(*ebiten.Image), vertices, dialogueBackdropProgram, &op)
	}
	log.remap(r, l, shows)
	remapCaptured(calls, r, l, shows)
}
