package sav

import (
	"reflect"
	"strings"
	"testing"
)

func TestActorOwnerUsesBoundPlayerKeyNotContainerOrSlotIndex(t *testing.T) {
	first := &Record{Off: 75, Class: "Player", Value: map[string]uint32{"This": 0x11223344, "Slot": 9}}
	second := &Record{Off: 1000, Class: "Player", Value: map[string]uint32{"This": 0x55667788, "Slot": 1}}
	actor := func(off int, key uint32) *Record {
		return &Record{Off: off, Class: "Human", Value: map[string]uint32{"Reference": key}, Raw: map[string][]byte{"Block12": make([]byte, 12), "U154": make([]byte, 180)}}
	}
	a := actor(200, 0x11223344)
	forward := actor(300, 0x55667788)
	first.Refs = map[string][]*Record{"Actors": {a, forward, actor(400, 0), actor(500, 0xdeadbeef)}}
	second.Refs = map[string][]*Record{"Actors": {a, actor(1200, 0x11223344), actor(1300, 0x55667788), nil}}
	got, err := ownerActors([]*Record{first, nil, second, first})
	if err != nil {
		t.Fatal(err)
	}
	var slots []uint16
	for _, a := range got {
		slots = append(slots, a.OwnerSlot)
	}
	if !reflect.DeepEqual(slots, []uint16{9, 0, 0, 0, 9, 1}) {
		t.Fatalf("owners=%v: only prior Player keys bind; aliases emit once", slots)
	}
}

func TestActorOwnerRejectsAmbiguousPlayerIdentity(t *testing.T) {
	p := &Record{Class: "Player", Value: map[string]uint32{"This": 42, "Slot": 1}}
	q := &Record{Class: "Player", Value: map[string]uint32{"This": 42, "Slot": 2}}
	if _, err := ownerActors([]*Record{p, q}); err == nil || !strings.Contains(err.Error(), "duplicate Player identity") {
		t.Fatalf("duplicate owner key: %v", err)
	}
	// A null key is never registered as an owner, even in synthetic input.
	p.Value["This"], q.Value["This"] = 0, 0
	if _, err := ownerActors([]*Record{p, q}); err != nil {
		t.Fatal(err)
	}
}
