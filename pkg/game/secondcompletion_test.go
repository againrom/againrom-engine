package game

import "testing"

func TestSecondCompletionUsesTheSelectedOutputTable(t *testing.T) {
	src := missionSource{mainPrefix + "text/cutpaths.txt": []byte("unused\r\none\r\ntwo\r\nthree\r\nfour\r\nfive\r\n")}
	for _, output := range []int{-1, 0, 1, 2, 3, 4, 5, 6, 10} {
		want := ""
		if output >= 1 && output <= 5 {
			want = []string{"", "one", "two", "three", "four", "five"}[output]
		}
		if got := secondGameCompletionDirectory(src, TextCode{}, output); got != want {
			t.Fatalf("output%d selected %q, want %q", output, got, want)
		}
	}
	if got := secondGameCompletionDirectory(nil, TextCode{}, 1); got != "" {
		t.Fatal("missing table manufactured a movie", got)
	}
}
