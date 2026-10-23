package secondshop

import (
	"reflect"
	"testing"
)

// TestStackingPlaceholder: an emptied entry takes any equal item once.
func TestStackingPlaceholder(t *testing.T) {
	book := Item{Table: Book, Spell: 3, Count: 1, Price: 10}
	potion := Item{Table: Magic, Row: 70, Count: 4, Price: 83}
	list := []Item{{Table: Book, Spell: 3, Price: 10}, {Table: Magic, Row: 70, Count: 1, Price: 80}}
	list = insert(list, book)
	list = insert(list, book)
	list = insert(list, potion)
	want := []Item{{Table: Book, Spell: 3, Count: 1, Price: 10}, {Table: Magic, Row: 70, Count: 5, Price: 80}, book}
	if !reflect.DeepEqual(list, want) {
		t.Fatalf("shelf %+v, want %+v", list, want)
	}
}

// TestStackingFirstOfTwoPlaceholders: of two emptied entries of one key, the
// first takes the item.
func TestStackingFirstOfTwoPlaceholders(t *testing.T) {
	book := Item{Table: Book, Spell: 3, Count: 1, Price: 10}
	empty := Item{Table: Book, Spell: 3, Price: 10}
	list := insert([]Item{empty, empty}, book)
	if want := []Item{book, empty}; !reflect.DeepEqual(list, want) {
		t.Fatalf("shelf %+v, want %+v", list, want)
	}
}
