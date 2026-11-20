package ui

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"sort"
	"testing"
)

// editorTipAnswers is every editor tip shape over fixture fonts: the fitted
// hover tip with and without a picture, the value tip and the help tip with
// their scroll window, and the shapes the popups ask for, each laid out in a
// page and painted. Each answer is the box and a digest of the page's pixels.
func editorTipAnswers(t *testing.T) map[string]string {
	t.Helper()
	pic := image.NewRGBA(image.Rect(0, 0, 30, 12))
	for i := 0; i < len(pic.Pix); i += 4 {
		pic.Pix[i], pic.Pix[i+1], pic.Pix[i+2], pic.Pix[i+3] = 30, 200, 90, 255
	}
	long := "ABC DEF GHI JKL MNO PQR STU VWX YZ ABC DEF GHI JKL MNO PQR STU VWX YZ ABC DEF GHI JKL MNO PQR STU VWX YZ"
	many := ""
	for i := 0; i < 30; i++ {
		many += fmt.Sprintf("row %d of the tip text ", i)
	}
	texts := []string{"Sword", "Sword damage 4 and more", long, many, ""}
	pages := map[string]image.Rectangle{"page": image.Rect(0, 0, 400, 300), "small": image.Rect(10, 10, 120, 90)}
	anchors := []image.Rectangle{image.Rect(20, 20, 60, 40), image.Rect(350, 270, 390, 290), image.Rect(0, 0, 10, 10), image.Rect(100, 60, 110, 70)}
	shapes := map[string]tipShape{
		"fit":    {Fit: true, Pull: true, Clip: true},
		"fitraw": {Fit: true},
		"wide":   {Width: 90},
		"value":  {Width: popupTipW, Pull: true, Clip: true, Scroll: true},
		"flip":   {Width: 120, Fit: true, Pull: true},
	}
	out := map[string]string{}
	for _, fname := range []string{"msg", "shape"} {
		f := messageFont()
		if fname == "shape" {
			f = shippedShapeFont()
		}
		for pname, bounds := range pages {
			for ai, a := range anchors {
				for ti, s := range texts {
					for sname, how := range shapes {
						for _, p := range []image.Image{nil, pic} {
							tip := layoutTip(bounds, f, a, s, p, how)
							dst := image.NewRGBA(image.Rect(0, 0, 400, 300))
							for i := 0; i < len(dst.Pix); i += 4 {
								dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = 90, 70, 50, 255
							}
							drawPopupTip(dst, f, tip, editorPopupInk)
							if tip.Rows > 0 {
								tip = tip.scrolled(0, 2)
								drawPopupTip(dst, f, tip, editorPopupInk)
							}
							h := sha256.Sum256(dst.Pix)
							out[fmt.Sprintf("%s/%s/%d/%d/%s/%v", fname, pname, ai, ti, sname, p != nil)] = fmt.Sprintf("%v %v %d %s", tip.Rect, tip.Lines, tip.Rows, hex.EncodeToString(h[:8]))
						}
					}
					for k, tip := range map[string]popupTip{"hover": hoverTip(bounds, f, a, s, pic), "value": valueTip(bounds, f, a, s), "help": helpTip(bounds, f, a, s)} {
						dst := image.NewRGBA(image.Rect(0, 0, 400, 300))
						drawPopupTip(dst, f, tip, editorPopupInk)
						h := sha256.Sum256(dst.Pix)
						out[fmt.Sprintf("%s/%s/%d/%d/%s", fname, pname, ai, ti, k)] = fmt.Sprintf("%v %v %v %s", tip.Rect, tip.Help, tip.Rows, hex.EncodeToString(h[:8]))
					}
				}
			}
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, 50, 50))
	drawPopupTip(dst, nil, popupTip{}, editorPopupInk)
	drawPopupTip(dst, messageFont(), popupTip{}, popupInk{Value: color.RGBA{1, 2, 3, 255}})
	out["empty"] = hex.EncodeToString(dst.Pix[:8])
	return out
}

// The answers of every editor tip are those recorded before the editor's tips
// and the game's hover boxes were built by one builder.
func TestEditorTipAnswersAreUnchanged(t *testing.T) {
	got := editorTipAnswers(t)
	keys := make([]string, 0, len(got))
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		fmt.Fprintf(h, "%s=%s\n", k, got[k])
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if len(got) != editorTipAnswerCount || sum != editorTipAnswerDigest {
		t.Fatalf("editor tip answers: %d recorded, digest %s; want %d and %s", len(got), sum, editorTipAnswerCount, editorTipAnswerDigest)
	}
}

const (
	editorTipAnswerCount  = 1041
	editorTipAnswerDigest = "99942e4cd6cd872dcb723b51f5995fbb3d6c69d03732518df976641c24e39bef"
)
