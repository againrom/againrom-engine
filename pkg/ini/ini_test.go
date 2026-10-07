package ini

import (
	"reflect"
	"testing"
)

const sample = "; settings\r\n# another comment\r\n\r\n[starter]\r\nAgainrom = C:\\Games\\a;b#c\\againrom.exe\r\nunknown-key = keep me\r\n\r\n[bases]\r\nen=C:\\en\r\n[Options]\r\nvolume =\r\n"

func TestRoundTripIsByteIdentical(t *testing.T) {
	f, problems := Parse([]byte(sample))
	if len(problems) != 0 {
		t.Fatalf("problems %v", problems)
	}
	if got := string(f.Bytes()); got != sample {
		t.Fatalf("round trip changed the text:\n%q\n%q", got, sample)
	}
	again, _ := Parse(f.Bytes())
	if string(again.Bytes()) != sample {
		t.Fatal("second round trip differs")
	}
}

func TestGetIgnoresCaseAndKeepsValueText(t *testing.T) {
	f, _ := Parse([]byte(sample))
	if v, ok := f.Get("STARTER", "againrom"); !ok || v != `C:\Games\a;b#c\againrom.exe` {
		t.Fatalf("got %q %v", v, ok)
	}
	if v, ok := f.Get("bases", "en"); !ok || v != `C:\en` {
		t.Fatalf("got %q %v", v, ok)
	}
	if v, ok := f.Get("options", "volume"); !ok || v != "" {
		t.Fatalf("empty value: %q %v", v, ok)
	}
	if _, ok := f.Get("bases", "ru"); ok {
		t.Fatal("missing key found")
	}
	if _, ok := f.Get("nosuch", "en"); ok {
		t.Fatal("missing section found")
	}
}

func TestSetKeepsUnknownKeysAndComments(t *testing.T) {
	f, _ := Parse([]byte(sample))
	f.Set("starter", "againrom", `D:\x\againrom.exe`)
	f.Set("bases", "ru", `C:\ru`)
	f.Set("mods", "dir", `C:\mods`)
	f.Set("options", "volume", "50")
	f.Set("starter", "againrom", `D:\x\againrom.exe`)
	want := "; settings\r\n# another comment\r\n\r\n[starter]\r\nAgainrom = D:\\x\\againrom.exe\r\nunknown-key = keep me\r\n\r\n[bases]\r\nen=C:\\en\r\nru = C:\\ru\r\n[Options]\r\nvolume = 50\r\n[mods]\r\ndir = C:\\mods\r\n"
	if got := string(f.Bytes()); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	if got := f.Keys("starter"); !reflect.DeepEqual(got, []string{"Againrom", "unknown-key"}) {
		t.Fatalf("keys %v", got)
	}
}

func TestMalformedLinesAreReportedAndKept(t *testing.T) {
	src := "[a]\nx = 1\nthis is not a pair\n[broken\n= nokey\ny = 2\n"
	f, problems := Parse([]byte(src))
	want := []Problem{{3, "this is not a pair"}, {4, "[broken"}, {5, "= nokey"}}
	if !reflect.DeepEqual(problems, want) {
		t.Fatalf("problems %v", problems)
	}
	if v, _ := f.Get("a", "y"); v != "2" {
		t.Fatalf("pair after the malformed lines lost: %q", v)
	}
	f.Set("a", "x", "9")
	if got := string(f.Bytes()); got != "[a]\nx = 9\nthis is not a pair\n[broken\n= nokey\ny = 2\n" {
		t.Fatalf("got %q", got)
	}
}

func TestDeleteAndPreamble(t *testing.T) {
	f, _ := Parse([]byte("top = 1\n[a]\nk = v\n"))
	if v, ok := f.Get("", "top"); !ok || v != "1" {
		t.Fatalf("preamble: %q %v", v, ok)
	}
	if !f.Delete("a", "k") || f.Delete("a", "k") {
		t.Fatal("delete result")
	}
	if got := string(f.Bytes()); got != "top = 1\n[a]\n" {
		t.Fatalf("got %q", got)
	}
}

func TestEmptyFileAndNewFileUseLF(t *testing.T) {
	f, problems := Parse(nil)
	if len(problems) != 0 || len(f.Bytes()) != 0 {
		t.Fatal("empty file")
	}
	f.Set("s", "k", "v\nw")
	if got := string(f.Bytes()); got != "[s]\nk = v\n" {
		t.Fatalf("got %q", got)
	}
}

func TestFirstDuplicateWins(t *testing.T) {
	f, _ := Parse([]byte("[a]\nk = 1\nk = 2\n"))
	if v, _ := f.Get("a", "k"); v != "1" {
		t.Fatal(v)
	}
	f.Set("a", "k", "3")
	if got := string(f.Bytes()); got != "[a]\nk = 3\nk = 2\n" {
		t.Fatalf("got %q", got)
	}
}

func TestSectionsListsEachNameOnceInFileOrder(t *testing.T) {
	f, _ := Parse([]byte("top = 1\n[b]\nk = v\n[A]\n[B]\n[mod.skill-cap]\nx = 1\n"))
	got := f.Sections()
	want := []string{"b", "A", "mod.skill-cap"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v", got)
		}
	}
	if empty, _ := Parse(nil); len(empty.Sections()) != 0 {
		t.Fatal("an empty file lists sections")
	}
}
