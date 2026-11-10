//go:build integration

package integrationtests

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"
)

type mountPoint struct {
	file  *os.File
	image *os.File
}

type subvolume struct {
	path     string
	readOnly bool
}

func newMountPoint() (*mountPoint, error) {
	mountPath, err := os.MkdirTemp(os.TempDir(), "btrfsutil-")
	if err != nil {
		return nil, err
	}

	image, err := os.CreateTemp(os.TempDir(), "btrfsutil-")
	if err != nil {
		return nil, errors.Join(err, abortMountPointCreation(mountPath, nil, false))
	}

	// 16MiB seems to be the minimal size of a Btrfs volume
	if err := image.Truncate(1024 * 1024 * 16); err != nil {
		return nil, errors.Join(err, abortMountPointCreation(mountPath, image, false))
	}

	err = exec.Command("mkfs.btrfs", "-q", "--mixed", image.Name()).Run()
	if err != nil {
		return nil, errors.Join(err, abortMountPointCreation(mountPath, image, false))
	}

	err = exec.Command("mount", "-o", "loop,user_subvol_rm_allowed", image.Name(), mountPath).Run()
	if err != nil {
		// image may be mounted, always try to unmount
		return nil, errors.Join(err, abortMountPointCreation(mountPath, image, true))
	}

	file, err := os.Open(mountPath)
	if err != nil {
		return nil, errors.Join(err, abortMountPointCreation(mountPath, image, true))
	}

	return &mountPoint{file, image}, nil
}

func abortMountPointCreation(mountPath string, image *os.File, tryUnmount bool) error {
	var errs []error
	if mountPath != "" {
		if tryUnmount {
			if err := exec.Command("umount", "-R", mountPath).Run(); err != nil {
				errs = append(errs, fmt.Errorf("could not unmount %q: %v", mountPath, err))
			}
		}

		if err := os.Remove(mountPath); err != nil {
			errs = append(errs, fmt.Errorf("could not remove %q: %v", mountPath, err))
		}
	}

	if image != nil {
		if err := image.Close(); err != nil {
			errs = append(errs, fmt.Errorf("could not close %q: %v", image.Name(), err))
		}
		if err := os.Remove(image.Name()); err != nil {
			errs = append(errs, fmt.Errorf("could not remove %q: %v", image.Name(), err))
		}
	}
	return errors.Join(errs...)
}

func (mp *mountPoint) cleanup() error {
	if mp == nil {
		return nil
	}
	var errs []error
	if mp.file != nil {
		if err := mp.file.Close(); err != nil {
			errs = append(errs, fmt.Errorf("could not close %q: %v", mp.file.Name(), err))
		}
		if err := exec.Command("umount", "-R", mp.file.Name()).Run(); err != nil {
			errs = append(errs, fmt.Errorf("could not unmount %q: %v", mp.file.Name(), err))
		}
		if err := os.Remove(mp.file.Name()); err != nil {
			errs = append(errs, fmt.Errorf("could not remove %q: %v", mp.file.Name(), err))
		}
	}

	if mp.image != nil {
		if err := mp.image.Close(); err != nil {
			errs = append(errs, fmt.Errorf("could not close %q: %v", mp.image.Name(), err))
		}
		if err := os.Remove(mp.image.Name()); err != nil {
			errs = append(errs, fmt.Errorf("could not remove %q: %v", mp.image.Name(), err))
		}
	}
	return errors.Join(errs...)
}

func (mp *mountPoint) path(subpath ...string) string {
	if mp.file == nil {
		return ""
	}
	return path.Join(append([]string{mp.file.Name()}, subpath...)...)
}

func (mp *mountPoint) fd() uintptr {
	if mp.file == nil {
		return 0
	}
	return mp.file.Fd()
}

func setupTestFilesystem(subvolumes ...subvolume) (*mountPoint, error) {
	if !hasPrivileges() {
		return nil, errors.New("Integration tests must be run as root")
	}

	mp, err := newMountPoint()
	if err != nil {
		return nil, fmt.Errorf("Failed to create btrfs filesystem: %v", err)
	}

	for _, subvol := range subvolumes {
		fullPath := path.Join(mp.path(), subvol.path)

		err := exec.Command("btrfs", "subvolume", "create", fullPath).Run()
		if err != nil {
			return mp, fmt.Errorf("Failed to create subvolume %q: %v", subvol.path, err)
		}

		if subvol.readOnly {
			err := exec.Command("btrfs", "property", "set", fullPath, "ro", "true").Run()
			if err != nil {
				return mp, fmt.Errorf("Failed to set ro on %q: %v", subvol.path, err)
			}
		}
	}
	return mp, nil
}

func setupTest(t *testing.T, subvolumes ...subvolume) *mountPoint {
	t.Helper()
	mp, err := setupTestFilesystem(subvolumes...)
	t.Cleanup(func() {
		if err := mp.cleanup(); err != nil {
			t.Errorf("test cleanup failed: %v", err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	return mp
}

func hasPrivileges() bool {
	return os.Geteuid() == 0
}

func wrapFdTest(t *testing.T, targetPath string, testFunc func(t *testing.T, fd uintptr)) {
	t.Helper()

	fd := ^uintptr(0)
	if targetPath != "" {
		f, err := os.Open(targetPath)
		if err != nil {
			t.Fatalf("failed to open %q: %v", targetPath, err)
		}
		t.Cleanup(func() {
			closeOpenedFile(t, f)
		})
		fd = f.Fd()
	}

	testFunc(t, fd)
}

func closeOpenedFile(t *testing.T, f *os.File) {
	if err := f.Close(); err != nil {
		t.Errorf("failed to close %q: %v", f.Name(), err)
	}
}
