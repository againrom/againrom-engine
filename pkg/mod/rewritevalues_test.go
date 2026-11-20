package mod

import "testing"

// RewriteValues changes the value of the lines it is given and no other byte.
func TestRewriteValuesKeepsEveryOtherByte(t *testing.T) {
	src := "# rows\n[[speaker]]\nid = 20001 # first\npicture = 'speakers/a.png'   # face\n\n[[speaker]]\nid = 20002\npicture = \"b.png\"\n"
	out, err := RewriteValues([]byte(src), []ValueEdit{{Line: 4, Value: "assets/speakers/a.png"}, {Line: 8, Value: "assets/speakers/b.png"}})
	if err != nil {
		t.Fatal(err)
	}
	want := "# rows\n[[speaker]]\nid = 20001 # first\npicture = \"assets/speakers/a.png\"   # face\n\n[[speaker]]\nid = 20002\npicture = \"assets/speakers/b.png\"\n"
	if string(out) != want {
		t.Errorf("got\n%s\nwant\n%s", out, want)
	}
	if _, err := RewriteValues([]byte(src), []ValueEdit{{Line: 99, Value: "x"}}); err == nil {
		t.Error("a line outside the file is accepted")
	}
}
