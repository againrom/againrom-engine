package ui

import "testing"

// TestNewChoiceRowsAreOneTable pins the create rows' refusal, admission and
// message marker reads.
func TestNewChoiceRowsAreOneTable(t *testing.T) {
	if word, ok := NewChoiceRefusal(NewGlobalChoice); !ok || word == "" {
		t.Fatalf("global marker gave %q %v", word, ok)
	}
	if word, ok := NewChoiceRefusal(NewMessageChoice); !ok || word == "" {
		t.Fatalf("message marker gave %q %v", word, ok)
	}
	if _, ok := NewChoiceRefusal(1234); ok {
		t.Fatal("a record value gave a create refusal")
	}
	if got := createChoice(ScriptMessageParameter); got != NewMessageChoice {
		t.Fatalf("message create choice %d, want %d", got, NewMessageChoice)
	}
}
