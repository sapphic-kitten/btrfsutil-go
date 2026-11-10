package btrfsutil

import (
	"reflect"
	"testing"
)

func TestNewQgroupInherit(t *testing.T) {
	q, err := NewQgroupInherit()
	if err != nil {
		t.Fatalf("NewQgroupInherit() error = %v", err)
	}
	defer q.Close()

	if got := q.GetGroups(); len(got) != 0 {
		t.Fatalf("initial groups = %v, want empty", got)
	}
}

func TestQgroupInheritAddGroup(t *testing.T) {
	q, err := NewQgroupInherit()
	if err != nil {
		t.Fatalf("NewQgroupInherit() error = %v", err)
	}
	defer q.Close()

	want := []uint64{1, 256, 1001}
	for _, id := range want {
		if err := q.AddGroup(id); err != nil {
			t.Fatalf("AddGroup(%d) error = %v", id, err)
		}
	}

	if got := q.GetGroups(); !reflect.DeepEqual(got, want) {
		t.Fatalf("GetGroups() = %v, want %v", got, want)
	}
}

func TestQgroupInheritCloseIsIdempotent(t *testing.T) {
	q, err := NewQgroupInherit()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	q.Close()
	q.Close() // Must not panic

	if q.cPtr != nil {
		t.Fatalf("pointer was not cleared")
	}
}
