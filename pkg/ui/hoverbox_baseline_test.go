package ui

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"sort"
	"testing"
)

// hoverBoxAnswers is every hover box instance over fixture fonts and inputs:
// the shared box (the corner picture present and absent, wrapped, clamped on
// each edge), the game's styled box (authored lines, shadow), the editor's
// box, and the shop and town characteristics box beside the pointer (clamped
// on both axes). Each answer is the box's rectangle in the frame and a digest
// of its pixels.
func hoverBoxAnswers(t *testing.T) map[string]string {
	t.Helper()
	digest := func(pic *image.RGBA, at image.Point, ok bool) string {
		if !ok || pic == nil {
			return "none"
		}
		h := sha256.New()
		b := pic.Bounds()
		for _, v := range []int{b.Dx(), b.Dy(), at.X, at.Y} {
			var buf [8]byte
			binary.LittleEndian.PutUint64(buf[:], uint64(int64(v)))
			h.Write(buf[:])
		}
		h.Write(pic.Pix)
		return fmt.Sprintf("%dx%d@%d,%d %s", b.Dx(), b.Dy(), at.X, at.Y, hex.EncodeToString(h.Sum(nil))[:16])
	}
	ball := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := 0; i < len(ball.Pix); i += 4 {
		ball.Pix[i], ball.Pix[i+1], ball.Pix[i+2], ball.Pix[i+3] = 200, 40, 40, 255
	}
	frame := image.Rect(0, 0, 640, 480)
	small := image.Rect(0, 0, 100, 240)
	shadow := color.RGBA{8, 8, 8, 255}
	game := func() *hoverStyle {
		return &hoverStyle{ink: color.RGBA{160, 120, 50, 255}, shadow: &shadow, authored: true}
	}
	editor := func() *hoverStyle {
		return &hoverStyle{font: messageFont(), ink: editorPaper, box: &hoverBoxInk{fill: editorPanel, light: editorGold, dark: editorInk}}
	}
	out := map[string]string{}
	fonts := map[string]func() tooltipTarget{
		"msg": func() tooltipTarget {
			return tooltipTarget{tooltipText, "t", []string{"ABC DEF GHI JKL MNO PQR STU VWX YZ ABC DEF GHI JKL MNO PQR STU", "", "#AA#", "BB"}, messageFont()}
		},
		"shape": func() tooltipTarget {
			return tooltipTarget{tooltipItem, "t", []string{"AA", "#AB", "#B", "AAAAAAAA BBBBBBBB AAAAAAAA BBBBBBBB AAAAAAAA BBBBBBBB AAAAAAAA BBBBBBBB AAAAAAAA BBBBBBBB AAAAAAAA"}, shippedShapeFont()}
		},
	}
	points := []image.Point{{0, 0}, {5, 100}, {320, 240}, {639, 479}, {99, 119}, {200, 350}, {-4, 30}}
	for name, mk := range fonts {
		for _, b := range []struct {
			n string
			r image.Rectangle
		}{{"frame", frame}, {"small", small}} {
			for _, p := range points {
				for _, st := range []struct {
					n string
					s func() *hoverStyle
					b image.Image
				}{{"shared", func() *hoverStyle { return nil }, nil}, {"sharedball", func() *hoverStyle { return nil }, ball},
					{"game", game, nil}, {"gameball", game, ball}, {"editor", editor, nil}} {
					pic, at, ok := tooltipPicture(mk(), p, b.r, st.b, st.s())
					out[fmt.Sprintf("tip/%s/%s/%v/%s", name, b.n, p, st.n)] = digest(pic, at, ok)
				}
			}
		}
	}
	for _, name := range []string{"msg", "shape"} {
		f := fonts[name]().font
		for _, lines := range [][]string{{"Sword"}, {"Sword", "Damage 4", "", "Value 9"}, {"", ""}, {"ABC DEF"}} {
			out[fmt.Sprintf("box/%s/%q", name, lines)] = digest(composeHoverBox(lines, f, ball, nil), image.Point{}, true)
			out[fmt.Sprintf("boxnil/%s/%q", name, lines)] = digest(composeHoverBox(lines, f, nil, nil), image.Point{}, true)
		}
	}
	for _, name := range []string{"msg", "shape"} {
		f := fonts[name]().font
		for _, lines := range [][]string{{"Sword"}, {"Sword", "Damage 4", "", "Value 9"}, {"", ""}, {"ABC DEF GHI JKL MNO PQR STU VWX YZ ABC DEF GHI JKL MNO PQR STU VWX YZ ABC DEF"}} {
			for _, p := range []image.Point{{20, 20}, {0, 0}, {300, 200}, {620, 470}, {639, 479}, {-30, -30}} {
				dst := image.NewRGBA(image.Rect(0, 0, shopScreenW, shopScreenH))
				for i := 0; i < len(dst.Pix); i += 4 {
					dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = 90, 70, 50, 255
				}
				DollTip{Beside: true}.DrawBeside(dst, f, lines, p)
				out[fmt.Sprintf("beside/%s/%q/%v", name, lines, p)] = digest(dst, image.Point{}, true)
			}
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, 50, 50))
	DollTip{Beside: true}.DrawBeside(dst, nil, []string{"x"}, image.Pt(1, 1))
	DollTip{Beside: true}.DrawBeside(dst, messageFont(), nil, image.Pt(1, 1))
	out["beside/empty"] = digest(dst, image.Point{}, true)
	return out
}

// The answers of every hover box instance are those recorded before the
// instances were built by one builder.
func TestHoverBoxAnswersAreUnchanged(t *testing.T) {
	got := hoverBoxAnswers(t)
	h := sha256.New()
	keys := make([]string, 0, len(got))
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(h, "%s=%s\n", k, got[k])
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if len(got) != hoverBoxAnswerCount || sum != hoverBoxAnswerDigest {
		t.Fatalf("hover box answers: %d recorded, digest %s; want %d and %s", len(got), sum, hoverBoxAnswerCount, hoverBoxAnswerDigest)
	}
}

const (
	hoverBoxAnswerCount  = 205
	hoverBoxAnswerDigest = "ed5f225c07b1f8dc4ec2e3f104e83daf753798de8ff0b36ae1bc9df89b809628"
)
