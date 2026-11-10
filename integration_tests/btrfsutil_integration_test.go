//go:build integration

package integrationtests

import (
	"errors"
	"log"
	"os"
	"reflect"
	"strconv"
	"syscall"
	"testing"

	"github.com/sapphic-kitten/btrfsutil-go"
)

var mountpoint *mountPoint

var defaultSubvolumeLayout []subvolume = []subvolume{
	{"subvolume1", false},
	{"subvolume2", true},
	{"subvolume1/subvolume3", false},
}

func TestMain(m *testing.M) {
	var err error
	mountpoint, err = setupTestFilesystem(defaultSubvolumeLayout...)

	if err != nil {
		log.Printf("setup failed: %v", err)
		if err := mountpoint.cleanup(); err != nil {
			log.Printf("cleanup failed: %v", err)
		}
		os.Exit(1)
	}

	rc := m.Run()
	if err = mountpoint.cleanup(); err != nil {
		log.Printf("cleanup failed: %v", err)
		os.Exit(1)
	}
	os.Exit(rc)
}

func TestFsSync(t *testing.T) {
	tests := []struct {
		name string
		path string
		err  error
	}{
		{"fs root", mountpoint.path(), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := btrfsutil.FsSync(tt.path); !errors.Is(err, tt.err) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestFsSyncFd(t *testing.T) {
	tests := []struct {
		name string
		path string
		err  error
	}{
		{"fs root", mountpoint.path(), nil},
		{"invalid fd", "", syscall.EBADF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapFdTest(t, tt.path, func(t *testing.T, fd uintptr) {
				if err := btrfsutil.FsSyncFd(fd); !errors.Is(err, tt.err) {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		})
	}
}

func TestStartThenWaitSync(t *testing.T) {
	transid, err := btrfsutil.FsStartSync(mountpoint.path())
	if err != nil {
		t.Fatalf("unexpected error while starting FsSync: %v", err)
	}
	if transid == 0 {
		t.Fatal("expected non-zero transid")
	}

	if err := btrfsutil.FsWaitSync(mountpoint.path(), transid); err != nil {
		t.Fatalf("unexpected error while waiting on %d: %v", transid, err)
	}

	if err := btrfsutil.FsWaitSync(mountpoint.path(), 0); err != nil {
		t.Fatalf("unexpected error while waiting on %d: %v", 0, err)
	}
}

func TestSubvolumeIsValid(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
		err  error
	}{
		{"valid subvolume", mountpoint.path("subvolume1"), true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := btrfsutil.SubvolumeIsValid(tt.path)
			if !errors.Is(err, tt.err) {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
			}
		})
	}
}

func TestSubvolumeIsValidFd(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
		err  error
	}{
		{"valid subvolume", mountpoint.path("subvolume1"), true, nil},
		{"invalid fd", "", false, syscall.EBADF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapFdTest(t, tt.path, func(t *testing.T, fd uintptr) {
				got, err := btrfsutil.SubvolumeIsValidFd(fd)
				if !errors.Is(err, tt.err) {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
				}
			})
		})
	}
}

func TestSubvolumeGetId(t *testing.T) {
	tests := []struct {
		name string
		path string
		want uint64
		err  error
	}{
		{"subvolume", mountpoint.path("subvolume1"), 256, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := btrfsutil.SubvolumeGetId(tt.path)
			if !errors.Is(err, tt.err) {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
			}
		})
	}
}

func TestSubvolumeGetIdFd(t *testing.T) {
	tests := []struct {
		name string
		path string
		want uint64
		err  error
	}{
		{"subvolume", mountpoint.path("subvolume1"), 256, nil},
		{"invalid fd", "", 0, syscall.EBADF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapFdTest(t, tt.path, func(t *testing.T, fd uintptr) {
				got, err := btrfsutil.SubvolumeGetIdFd(fd)
				if !errors.Is(err, tt.err) {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
				}
			})
		})
	}
}

func TestSubvolumeGetPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		id   uint64
		want string
		err  error
	}{
		{"by id", mountpoint.path(), 256, "subvolume1", nil},
		{"by path", mountpoint.path("subvolume2"), 0, "subvolume2", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := btrfsutil.SubvolumeGetPath(tt.path, tt.id)
			if !errors.Is(err, tt.err) {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
			}
		})
	}
}

func TestSubvolumeGetPathFd(t *testing.T) {
	tests := []struct {
		name string
		path string
		id   uint64
		want string
		err  error
	}{
		{"by id", mountpoint.path(), 256, "subvolume1", nil},
		{"by fd", mountpoint.path("subvolume2"), 0, "subvolume2", nil},
		{"invalid fd", "", 0, "", syscall.EBADF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapFdTest(t, tt.path, func(t *testing.T, fd uintptr) {
				got, err := btrfsutil.SubvolumeGetPathFd(fd, tt.id)
				if !errors.Is(err, tt.err) {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
				}
			})
		})
	}
}

func TestSubvolumeGetInfo(t *testing.T) {
	type subvolumeInfo struct {
		id       uint64
		parentId uint64
		dirId    uint64
		flags    uint64
	}

	convert := func(info *btrfsutil.SubvolumeInfo) *subvolumeInfo {
		if info == nil {
			return nil
		}
		return &subvolumeInfo{info.Id, info.ParentId, info.DirId, info.Flags}
	}

	tests := []struct {
		name string
		path string
		id   uint64
		want *subvolumeInfo
		err  error
	}{
		{"by id", mountpoint.path(), 256, &subvolumeInfo{256, 5, 256, 0}, nil},
		{"by path", mountpoint.path("subvolume2"), 0, &subvolumeInfo{257, 5, 256, 1}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := btrfsutil.SubvolumeGetInfo(tt.path, tt.id)
			if !errors.Is(err, tt.err) {
				t.Fatalf("unexpected error: %v", err)
			}

			got := convert(info)
			if !reflect.DeepEqual(tt.want, got) {
				t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
			}
		})
	}
}

func TestSubvolumeGetInfoFd(t *testing.T) {
	type subvolumeInfo struct {
		id       uint64
		parentId uint64
		dirId    uint64
		flags    uint64
	}

	convert := func(info *btrfsutil.SubvolumeInfo) *subvolumeInfo {
		if info == nil {
			return nil
		}
		return &subvolumeInfo{info.Id, info.ParentId, info.DirId, info.Flags}
	}

	tests := []struct {
		name string
		path string
		id   uint64
		want *subvolumeInfo
		err  error
	}{
		{"by id", mountpoint.path(), 256, &subvolumeInfo{256, 5, 256, 0}, nil},
		{"by path", mountpoint.path("subvolume2"), 0, &subvolumeInfo{257, 5, 256, 1}, nil},
		{"invalid fd", "", 0, nil, syscall.EBADF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapFdTest(t, tt.path, func(t *testing.T, fd uintptr) {
				info, err := btrfsutil.SubvolumeGetInfoFd(fd, tt.id)
				if !errors.Is(err, tt.err) {
					t.Fatalf("unexpected error: %v", err)
				}

				got := convert(info)
				if !reflect.DeepEqual(tt.want, got) {
					t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
				}
			})
		})
	}
}

func TestSubvolumeGetReadOnly(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
		err  error
	}{
		{"rw subvolume", mountpoint.path("subvolume1"), false, nil},
		{"ro subvolume", mountpoint.path("subvolume2"), true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := btrfsutil.SubvolumeGetReadOnly(tt.path)
			if !errors.Is(err, tt.err) {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
			}
		})
	}
}

func TestSubvolumeGetReadOnlyFd(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
		err  error
	}{
		{"rw subvolume", mountpoint.path("subvolume1"), false, nil},
		{"ro subvolume", mountpoint.path("subvolume2"), true, nil},
		{"invalid fd", "", false, syscall.EBADF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapFdTest(t, tt.path, func(t *testing.T, fd uintptr) {
				got, err := btrfsutil.SubvolumeGetReadOnlyFd(fd)
				if !errors.Is(err, tt.err) {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
				}
			})
		})
	}
}

func TestSubvolumeSetReadOnly(t *testing.T) {
	mp := setupTest(t, subvolume{"subvolume1", false})

	tests := []struct {
		name     string
		path     string
		readOnly bool
		want     error
	}{
		{"set ro", mp.path("subvolume1"), true, nil},
		{"set rw", mp.path("subvolume1"), false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := btrfsutil.SubvolumeSetReadOnly(tt.path, tt.readOnly); !errors.Is(err, tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.want == nil {
				ro, err := btrfsutil.SubvolumeGetReadOnly(tt.path)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if ro != tt.readOnly {
					t.Fatalf("got: \"%v\", want: \"%v\"", ro, tt.readOnly)
				}
			}
		})
	}
}

func TestSubvolumeSetReadOnlyFd(t *testing.T) {
	mp := setupTest(t, subvolume{"subvolume1", false})

	tests := []struct {
		name     string
		path     string
		readOnly bool
		want     error
	}{
		{"set ro", mp.path("subvolume1"), true, nil},
		{"set rw", mp.path("subvolume1"), false, nil},
		{"invalid fd", "", false, syscall.EBADF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapFdTest(t, tt.path, func(t *testing.T, fd uintptr) {
				if err := btrfsutil.SubvolumeSetReadOnlyFd(fd, tt.readOnly); !errors.Is(err, tt.want) {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.want == nil {
					ro, err := btrfsutil.SubvolumeGetReadOnly(tt.path)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if ro != tt.readOnly {
						t.Fatalf("got: \"%v\", want: \"%v\"", ro, tt.readOnly)
					}
				}
			})
		})
	}
}

func TestSubvolumeGetDefault(t *testing.T) {
	tests := []struct {
		name string
		path string
		want uint64
		err  error
	}{
		{"fs root", mountpoint.path(), 5, nil},
		{"subvolume", mountpoint.path("subvolume1"), 5, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := btrfsutil.SubvolumeGetDefault(tt.path)
			if !errors.Is(err, tt.err) {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
			}
		})
	}
}

func TestSubvolumeGetDefaultFd(t *testing.T) {
	tests := []struct {
		name string
		path string
		want uint64
		err  error
	}{
		{"fs root", mountpoint.path(), 5, nil},
		{"subvolume", mountpoint.path("subvolume1"), 5, nil},
		{"invalid fd", "", 0, syscall.EBADF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapFdTest(t, tt.path, func(t *testing.T, fd uintptr) {
				got, err := btrfsutil.SubvolumeGetDefaultFd(fd)
				if !errors.Is(err, tt.err) {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
				}
			})
		})
	}
}

func TestSubvolumeSetDefault(t *testing.T) {
	mp := setupTest(t, defaultSubvolumeLayout...)

	tests := []struct {
		name string
		path string
		id   uint64
		want uint64
		err  error
	}{
		{"by id", mp.path(), 257, 257, nil},
		{"by path", mp.path("subvolume1"), 0, 256, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := btrfsutil.SubvolumeSetDefault(tt.path, tt.id); !errors.Is(err, tt.err) {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.err == nil {
				got, err := btrfsutil.SubvolumeGetDefault(mp.path())
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
				}
			}
		})
	}
}

func TestSubvolumeSetDefaultFd(t *testing.T) {
	mp := setupTest(t, defaultSubvolumeLayout...)

	tests := []struct {
		name string
		path string
		id   uint64
		want uint64
		err  error
	}{
		{"by id", mp.path(), 257, 257, nil},
		{"by fd", mp.path("subvolume1"), 0, 256, nil},
		{"invalid fd", "", 270, 0, syscall.EBADF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapFdTest(t, tt.path, func(t *testing.T, fd uintptr) {
				if err := btrfsutil.SubvolumeSetDefaultFd(fd, tt.id); !errors.Is(err, tt.err) {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.err == nil {
					got, err := btrfsutil.SubvolumeGetDefault(mp.path())
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if got != tt.want {
						t.Fatalf("got: \"%v\", want: \"%v\"", got, tt.want)
					}
				}
			})
		})
	}
}

func TestSubvolumeCreate(t *testing.T) {
	mp := setupTest(t)

	tests := []struct {
		name string
		path string
		err  error
	}{
		{"simple subvolume", mp.path("subvolume1"), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := btrfsutil.SubvolumeCreate(tt.path, nil); !errors.Is(err, tt.err) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSubvolumeCreateFd(t *testing.T) {
	mp := setupTest(t)

	tests := []struct {
		name       string
		parentFd   uintptr
		subvolName string
		err        error
	}{
		{"ok", mp.fd(), "subvolume1", nil},
		{"invalid parentFd", ^uintptr(0), "will_not_create", syscall.EBADF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := btrfsutil.SubvolumeCreateFd(tt.parentFd, tt.subvolName, nil); !errors.Is(err, tt.err) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSubvolumeCreateWithQgroup(t *testing.T) {
	mp := setupTest(t)

	qgroupInherit, err := btrfsutil.NewQgroupInherit()
	if err != nil {
		t.Fatalf("failed to create new QgroupInherit: %v", err)
	}
	t.Cleanup(qgroupInherit.Close)

	if err := qgroupInherit.AddGroup(1); err != nil {
		t.Fatalf("QgroupInherit.AddGroup failed: %v", err)
	}

	if err := btrfsutil.SubvolumeCreate(mp.path("subvolume1"), qgroupInherit); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSubvolumeSnapshot(t *testing.T) {
	mp := setupTest(t, subvolume{"subvolume1", false})
	subvolPath := mp.path("subvolume1")

	tests := []struct {
		name      string
		recursive bool
		readOnly  bool
		err       error
	}{
		{"non-recursive, rw", false, false, nil},
		{"recursive, rw", true, false, nil},
		{"non-recursive, ro", false, true, nil},
		{"recursive, ro", true, true, syscall.EINVAL},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapPath := mp.path("snap" + strconv.Itoa(i))
			err := btrfsutil.SubvolumeSnapshot(subvolPath, snapPath, tt.recursive, tt.readOnly, nil)
			if !errors.Is(err, tt.err) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSubvolumeSnapshotFd(t *testing.T) {
	mp := setupTest(t, subvolume{"subvolume1", false})
	subvolPath := mp.path("subvolume1")

	f, err := os.Open(subvolPath)
	if err != nil {
		t.Fatalf("failed to open %q: %v", subvolPath, err)
	}
	t.Cleanup(func() { closeOpenedFile(t, f) })

	tests := []struct {
		name      string
		fd        uintptr
		recursive bool
		readOnly  bool
		err       error
	}{
		{"non-recursive, rw", f.Fd(), false, false, nil},
		{"recursive, rw", f.Fd(), true, false, nil},
		{"non-recursive, ro", f.Fd(), false, true, nil},
		{"recursive, ro", f.Fd(), true, true, syscall.EINVAL},
		{"invalid fd", ^uintptr(0), false, false, syscall.EINVAL},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapPath := mp.path("snap" + strconv.Itoa(i))
			err := btrfsutil.SubvolumeSnapshotFd(tt.fd, snapPath, tt.recursive, tt.readOnly, nil)
			if !errors.Is(err, tt.err) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSubvolumeSnapshotFd2(t *testing.T) {
	mp := setupTest(t, subvolume{"subvolume1", false})
	subvolPath := mp.path("subvolume1")

	f, err := os.Open(subvolPath)
	if err != nil {
		t.Fatalf("failed to open %q: %v", subvolPath, err)
	}
	t.Cleanup(func() { closeOpenedFile(t, f) })

	tests := []struct {
		name      string
		fd        uintptr
		parentFd  uintptr
		recursive bool
		readOnly  bool
		err       error
	}{
		{"non-recursive, rw", f.Fd(), mp.fd(), false, false, nil},
		{"recursive, rw", f.Fd(), mp.fd(), true, false, nil},
		{"non-recursive, ro", f.Fd(), mp.fd(), false, true, nil},
		{"recursive, ro", f.Fd(), mp.fd(), true, true, syscall.EINVAL},
		{"invalid fd", ^uintptr(0), mp.fd(), false, false, syscall.EINVAL},
		{"invalid parentFd", f.Fd(), ^uintptr(0), false, false, syscall.EBADF},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapName := "snap" + strconv.Itoa(i)
			err := btrfsutil.SubvolumeSnapshotFd2(tt.fd, tt.parentFd, snapName, tt.recursive, tt.readOnly, nil)
			if !errors.Is(err, tt.err) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSubvolumeDelete(t *testing.T) {
	mp := setupTest(t, defaultSubvolumeLayout...)

	tests := []struct {
		name      string
		path      string
		recursive bool
		err       error
	}{
		{"regular subvolume", mp.path("subvolume2"), false, nil},
		{"nested, recursive", mp.path("subvolume1"), true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := btrfsutil.SubvolumeDelete(tt.path, tt.recursive); !errors.Is(err, tt.err) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSubvolumeDeleteFd(t *testing.T) {
	mp := setupTest(t, defaultSubvolumeLayout...)

	tests := []struct {
		name          string
		parentPath    string
		subvolumeName string
		recursive     bool
		err           error
	}{
		{"regular subvolume", mp.path(), "subvolume2", false, nil},
		{"nested, recursive", mp.path(), "subvolume1", true, nil},
		{"invalid parentFd", "", "invalid", false, syscall.EBADF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapFdTest(t, tt.parentPath, func(t *testing.T, fd uintptr) {
				if err := btrfsutil.SubvolumeDeleteFd(fd, tt.subvolumeName, tt.recursive); !errors.Is(err, tt.err) {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		})
	}
}

func TestSubvolumeDeleteByIdFd(t *testing.T) {
	mp := setupTest(t, subvolume{"subvolume1", false})

	tests := []struct {
		name       string
		parentPath string
		id         uint64
		err        error
	}{
		{"regular subvolume", mp.path(), 256, nil},
		{"invalid parentFd", "", 300, syscall.EBADF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapFdTest(t, tt.parentPath, func(t *testing.T, fd uintptr) {
				if err := btrfsutil.SubvolumeDeleteByIdFd(fd, tt.id); !errors.Is(err, tt.err) {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		})
	}
}

func TestSubvolumeListDeleted(t *testing.T) {
	mp := setupTest(t, subvolume{"subvolume1", false})
	want := []uint64{256}

	if err := btrfsutil.SubvolumeDelete(mp.path("subvolume1"), false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := btrfsutil.SubvolumeListDeleted(mp.path())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got: \"%v\", want: \"%v\"", got, want)
	}
}

func TestSubvolumeListDeletedFd(t *testing.T) {
	mp := setupTest(t, subvolume{"subvolume1", false})
	if err := btrfsutil.SubvolumeDelete(mp.path("subvolume1"), false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("regular subvolume", func(t *testing.T) {
		want := []uint64{256}

		got, err := btrfsutil.SubvolumeListDeletedFd(mp.fd())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got: \"%v\", want: \"%v\"", got, want)
		}
	})

	t.Run("invalid fd", func(t *testing.T) {
		if _, err := btrfsutil.SubvolumeListDeletedFd(^uintptr(0)); !errors.Is(err, syscall.EBADF) {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
