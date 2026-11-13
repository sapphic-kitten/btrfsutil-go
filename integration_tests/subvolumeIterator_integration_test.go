//go:build integration

package integrationtests

import (
	"errors"
	"reflect"
	"syscall"
	"testing"

	"github.com/sapphic-kitten/btrfsutil-go"
)

type iterSubvolume struct {
	path string
	id   uint64
}

type iterSubvolumeInfo struct {
	path     string
	id       uint64
	parentId uint64
	dirId    uint64
	flags    uint64
}

func TestNewSubvolumeIterator(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		topId     uint64
		postOrder bool
		err       error
	}{
		{"by path, pre-order", mountpoint.path("subvolume1"), 0, false, nil},
		{"by id, pre-order", mountpoint.path(), 5, false, nil},
		{"by path, post-order", mountpoint.path("subvolume2"), 0, true, nil},
		{"by id, post-order", mountpoint.path(), 258, true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter, err := btrfsutil.NewSubvolumeIterator(tt.path, tt.topId, tt.postOrder)
			t.Cleanup(iter.Close)
			if !errors.Is(err, tt.err) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewSubvolumeIteratorFd(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		topId     uint64
		postOrder bool
		err       error
	}{
		{"by fd, pre-order", mountpoint.path("subvolume1"), 0, false, nil},
		{"by id, pre-order", mountpoint.path(), 5, false, nil},
		{"by fd, post-order", mountpoint.path("subvolume2"), 0, true, nil},
		{"by id, post-order", mountpoint.path(), 258, true, nil},
		{"invalid fd", "", 0, false, syscall.EBADF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapFdTest(t, tt.path, func(t *testing.T, fd int) {
				iter, err := btrfsutil.NewSubvolumeIteratorFd(fd, tt.topId, tt.postOrder)
				t.Cleanup(iter.Close)
				if !errors.Is(err, tt.err) {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		})
	}
}

func TestListSubvolumes(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		topId     uint64
		postOrder bool
		want      []iterSubvolume
	}{
		{"from root, pre-order", mountpoint.path(), 5, false, []iterSubvolume{{"subvolume1", 256}, {"subvolume1/subvolume3", 258}, {"subvolume2", 257}}},
		{"from root, post-order", mountpoint.path(), 5, true, []iterSubvolume{{"subvolume1/subvolume3", 258}, {"subvolume1", 256}, {"subvolume2", 257}}},
		{"from subvolume1 by id", mountpoint.path(), 256, false, []iterSubvolume{{"subvolume3", 258}}},
		{"from subvolume1 by path", mountpoint.path("subvolume1"), 0, false, []iterSubvolume{{"subvolume3", 258}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter, err := btrfsutil.NewSubvolumeIterator(tt.path, tt.topId, tt.postOrder)
			if err != nil {
				t.Fatalf("NewSubvolumeIterator returned unexpected error: %v", err)
			}
			defer iter.Close()

			var got []iterSubvolume
			for subvolPath, id := range iter.ListSubvolumes() {
				got = append(got, iterSubvolume{subvolPath, id})
			}

			if err := iter.Err(); err != nil {
				t.Fatalf("iterator closed early: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
			}
		})
	}
}

func TestNext(t *testing.T) {
	t.Run("walk full", func(t *testing.T) {
		want := []iterSubvolume{{"subvolume1", 256}, {"subvolume1/subvolume3", 258}, {"subvolume2", 257}}

		iter, err := btrfsutil.NewSubvolumeIterator(mountpoint.path(), 0, false)
		if err != nil {
			t.Fatalf("NewSubvolumeIterator returned unexpected error: %v", err)
		}
		defer iter.Close()

		var got []iterSubvolume
		for subvolPath, id, hasMore := iter.Next(); hasMore; subvolPath, id, hasMore = iter.Next() {
			got = append(got, iterSubvolume{subvolPath, id})
		}

		if err := iter.Err(); err != nil {
			t.Fatalf("iterator closed early: %v", err)
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got: \"%v\", want: \"%v\"", got, want)
		}
	})

	t.Run("advance Next() inside ListSubvolumes", func(t *testing.T) {
		want := []iterSubvolume{{"subvolume1", 256}, {"subvolume2", 257}}

		iter, err := btrfsutil.NewSubvolumeIterator(mountpoint.path(), 0, false)
		if err != nil {
			t.Fatalf("NewSubvolumeIterator returned unexpected error: %v", err)
		}
		defer iter.Close()

		var got []iterSubvolume
		for subvolPath, id := range iter.ListSubvolumes() {
			if id == 256 { //on subvolume1
				iter.Next() // take one more step
			}
			got = append(got, iterSubvolume{subvolPath, id})
		}

		if err := iter.Err(); err != nil {
			t.Fatalf("iterator closed early: %v", err)
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got: \"%v\", want: \"%v\"", got, want)
		}
	})

	t.Run("Next after end", func(t *testing.T) {
		iter, err := btrfsutil.NewSubvolumeIterator(mountpoint.path(), 257, false)
		if err != nil {
			t.Fatalf("NewSubvolumeIterator returned unexpected error: %v", err)
		}
		defer iter.Close()

		for _, _, hasMore := iter.Next(); hasMore; _, _, hasMore = iter.Next() {
			//no op
		}

		if subvolPath, id, hasMore := iter.Next(); subvolPath != "" || id != 0 || hasMore {
			t.Fatalf("Next returned %v %v %v after the iterator was exhausted", subvolPath, id, hasMore)
		}

		if err := iter.Err(); err != nil {
			t.Fatalf("Err returned unexpected error: %v", err)
		}
	})
}

func TestListSubvolumeInfo(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		topId     uint64
		postOrder bool
		want      []iterSubvolumeInfo
	}{
		{"from root, pre-order", mountpoint.path(), 5, false, []iterSubvolumeInfo{{"subvolume1", 256, 5, 256, 0}, {"subvolume1/subvolume3", 258, 256, 256, 0}, {"subvolume2", 257, 5, 256, 1}}},
		{"from root, post-order", mountpoint.path(), 5, true, []iterSubvolumeInfo{{"subvolume1/subvolume3", 258, 256, 256, 0}, {"subvolume1", 256, 5, 256, 0}, {"subvolume2", 257, 5, 256, 1}}},
		{"from subvolume1 by id", mountpoint.path(), 256, false, []iterSubvolumeInfo{{"subvolume3", 258, 256, 256, 0}}},
		{"from subvolume1 by path", mountpoint.path("subvolume1"), 0, false, []iterSubvolumeInfo{{"subvolume3", 258, 256, 256, 0}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter, err := btrfsutil.NewSubvolumeIterator(tt.path, tt.topId, tt.postOrder)
			if err != nil {
				t.Fatalf("NewSubvolumeIterator returned unexpected error: %v", err)
			}
			defer iter.Close()

			var got []iterSubvolumeInfo
			for subvolPath, info := range iter.ListSubvolumeInfo() {
				got = append(got, iterSubvolumeInfo{subvolPath, info.Id, info.ParentId, info.DirId, info.Flags})
			}

			if err := iter.Err(); err != nil {
				t.Fatalf("iterator closed early: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
			}
		})
	}
}

func TestNextSubvolumeInfo(t *testing.T) {
	want := []iterSubvolumeInfo{
		{"subvolume1", 256, 5, 256, 0},
		{"subvolume1/subvolume3", 258, 256, 256, 0},
		{"subvolume2", 257, 5, 256, 1},
	}

	iter, err := btrfsutil.NewSubvolumeIterator(mountpoint.path(), 0, false)
	if err != nil {
		t.Fatalf("NewSubvolumeIterator returned unexpected error: %v", err)
	}
	defer iter.Close()

	var got []iterSubvolumeInfo
	for subvolPath, info, hasMore := iter.NextSubvolumeInfo(); hasMore; subvolPath, info, hasMore = iter.NextSubvolumeInfo() {
		got = append(got, iterSubvolumeInfo{subvolPath, info.Id, info.ParentId, info.DirId, info.Flags})
	}

	if err := iter.Err(); err != nil {
		t.Fatalf("iterator closed early: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got: \"%v\", want: \"%v\"", got, want)
	}
}
