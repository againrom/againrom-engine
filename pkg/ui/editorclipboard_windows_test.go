package ui

import (
	"os"
	"testing"
)

func TestWindowsEditorClipboardRoundTrip(t *testing.T) {
	if os.Getenv("AGAINROM_TEST_SYSTEM_CLIPBOARD") != "1" {
		t.Skip("real clipboard requires AGAINROM_TEST_SYSTEM_CLIPBOARD=1")
	}
	desktop, _, err := clipboardUser.NewProc("OpenInputDesktop").Call(0, 0, 0x100)
	if desktop == 0 {
		t.Skipf("no interactive desktop: %v", err)
	}
	clipboardUser.NewProc("CloseDesktop").Call(desktop)
	type savedFormat struct {
		format uintptr
		data   []byte
	}
	var saved []savedFormat
	err = withEditorClipboard(func() error {
		var format uintptr
		for {
			format, _, _ = clipboardUser.NewProc("EnumClipboardFormats").Call(format)
			if format == 0 {
				break
			}
			b, err := clipboardBytes(format)
			if err != nil {
				t.Skipf("cannot preserve existing clipboard formats without loss: %v", err)
			}
			saved = append(saved, savedFormat{format, b})
		}
		defer func() {
			if ok, _, err := clipboardEmpty.Call(); ok == 0 {
				t.Errorf("restore original clipboard: %v", err)
				return
			}
			for _, f := range saved {
				if err := putClipboardBytes(f.format, f.data); err != nil {
					t.Errorf("restore original clipboard: %v", err)
				}
			}
		}()
		const want = "Привет, мир! Ёж и Latin\nВторая строка"
		if err := writeEditorClipboardText(want); err != nil {
			t.Fatal(err)
		}
		if got, err := readEditorClipboardText(); err != nil || got != want {
			t.Fatalf("real clipboard got %q, %v", got, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
