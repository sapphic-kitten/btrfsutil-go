/*
 * Copyright (C) 2022 Jana Marlou Rettig
 *
 * This file is part of btrfsutil-go.
 *
 * btrfsutil-go is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 2.1 of the License, or
 * (at your option) any later version.
 *
 * btrfsutil-go is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with btrfsutil-go.  If not, see <http://www.gnu.org/licenses/>.
 */

package btrfsutil

// #cgo LDFLAGS: -lbtrfsutil
// #include <stdlib.h>
// #include <btrfsutil.h>
import "C"
import (
	"errors"
	"iter"
	"unsafe"
)

// SubvolumeIterator is an iterator over the subvolumes in a Btrfs filesystem.
// It is created by NewSubvolumeIterator or NewSubvolumeIteratorFd and holds an
// open C iterator until it is destroyed by Close; list iteration closes the
// iterator automatically when it finishes or fails. Close needs to be
// called explicitly when iteration is stopped early. See NewSubvolumeIterator,
// ListSubvolumes and Close for more details.
type SubvolumeIterator struct {
	cPtr *C.struct_btrfs_util_subvolume_iterator
	err  error
}

// NewSubvolumeIterator creates an iterator over the subvolumes in a Btrfs
// filesystem. The returned iterator holds an open C iterator and must be
// destroyed with Close()
//
// path may be any path in a Btrfs filesystem; it does not have to refer to a
// subvolume unless topId is zero. topId is the ID of the subvolume beneath
// which (but not including) subvolumes will be listed; if zero is given, the
// subvolume ID of path is used. To list all subvolumes, pass the ID of the root
// subvolume, BTRFS_FS_TREE_OBJECTID (i.e., 5); the returned paths are then
// relative to the subvolume with this ID. If postOrder is true, subvolumes are
// yielded post-order, so foo/bar is yielded before foo; the default is
// pre-order, where foo is yielded before foo/bar.
//
// This requires appropriate privilege (CAP_SYS_ADMIN) unless topId is zero and
// the kernel supports BTRFS_IOC_GET_SUBVOL_ROOTREF and BTRFS_IOC_INO_LOOKUP_USER
// (kernel >= 4.18). In the latter case, subvolumes which cannot be accessed
// (e.g., due to permissions or other mounts) are silently skipped.
func NewSubvolumeIterator(path string, topId uint64, postOrder bool) (*SubvolumeIterator, error) {
	si := new(SubvolumeIterator)

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	flags := 0
	if postOrder {
		flags |= C.BTRFS_UTIL_SUBVOLUME_ITERATOR_POST_ORDER
	}

	cError, err := C.btrfs_util_subvolume_iter_create(cPath, C.uint64_t(topId), C.int(flags), &si.cPtr)
	return si, wrapCError(cError, err)
}

// NewSubvolumeIteratorFd creates an iterator over the subvolumes in a Btrfs
// filesystem.
//
// fd may be the file descriptor for any object on the filesystem.
// See NewSubvolumeIterator for details.
func NewSubvolumeIteratorFd(fd int, topId uint64, postOrder bool) (*SubvolumeIterator, error) {
	si := new(SubvolumeIterator)

	flags := 0
	if postOrder {
		flags |= C.BTRFS_UTIL_SUBVOLUME_ITERATOR_POST_ORDER
	}

	cError, err := C.btrfs_util_subvolume_iter_create_fd(C.int(fd), C.uint64_t(topId), C.int(flags), &si.cPtr)
	return si, wrapCError(cError, err)
}

// ListSubvolumes returns an iter.Seq2 which yields the path and ID of each
// (remaining) subvolume.
//
// Paths are relative to the subvolume ID used to create the iterator. Any error
// is recorded in the iterator; see NewSubvolumeIterator and Err for details.
func (si *SubvolumeIterator) ListSubvolumes() iter.Seq2[string, uint64] {
	return func(yield func(string, uint64) bool) {
		for si.cPtr != nil {
			var cPath *C.char
			defer C.free(unsafe.Pointer(cPath))

			var cId C.uint64_t

			cError, err := C.btrfs_util_subvolume_iter_next(si.cPtr, &cPath, &cId)
			if cError != C.BTRFS_UTIL_OK {
				si.err = wrapCError(cError, err)
				return
			}

			if !yield(C.GoString(cPath), uint64(cId)) {
				return
			}
		}
	}
}

// Next pulls the path and ID of the next subvolume from a SubvolumeIterator,
// along with a bool which reports whether the pair is valid: it is false once
// there are no subvolumes left or an error has occurred.
//
// Paths are relative to the subvolume ID used to create the iterator. Any error
// is recorded in the iterator; see NewSubvolumeIterator and Err for details.
func (si *SubvolumeIterator) Next() (string, uint64, bool) {
	next, stop := iter.Pull2(si.ListSubvolumes())
	defer stop()
	return next()
}

// ListSubvolumeInfo returns an iter.Seq2 which yields the path and the
// SubvolumeInfo of each (remaining) subvolume.
//
// Paths are relative to the subvolume ID used to create the iterator. Any error
// is recorded in the iterator; see NewSubvolumeIterator and Err for details.
func (si *SubvolumeIterator) ListSubvolumeInfo() iter.Seq2[string, *SubvolumeInfo] {
	return func(yield func(string, *SubvolumeInfo) bool) {
		for si.cPtr != nil {
			var cPath *C.char
			defer C.free(unsafe.Pointer(cPath))

			var info C.struct_btrfs_util_subvolume_info

			cError, err := C.btrfs_util_subvolume_iter_next_info(si.cPtr, &cPath, &info)
			if cError != C.BTRFS_UTIL_OK {
				si.err = wrapCError(cError, err)
				return
			}

			if !yield(C.GoString(cPath), newSubvolumeInfo(&info)) {
				return
			}
		}
	}
}

// NextSubvolumeInfo pulls the path and the SubvolumeInfo of the next subvolume
// from a SubvolumeIterator, along with a bool which reports whether the pair is
// valid: it is false once there are no subvolumes left or an error has occurred.
//
// Paths are relative to the subvolume ID used to create the iterator. Any error
// is recorded in the iterator; see NewSubvolumeIterator and Err for details.
func (si *SubvolumeIterator) NextSubvolumeInfo() (string, *SubvolumeInfo, bool) {
	next, stop := iter.Pull2(si.ListSubvolumeInfo())
	defer stop()
	return next()
}

// Fd gets the file descriptor associated with a SubvolumeIterator.
//
// The iterator must not have been destroyed by Close.
func (si *SubvolumeIterator) Fd() int {
	return int(C.btrfs_util_subvolume_iterator_fd(si.cPtr))
}

// Err gets the last error recorded by a SubvolumeIterator, or nil if there is
// none. Errors from iteration (ListSubvolumes, ListSubvolumeInfo, Next and
// NextSubvolumeInfo) should be checked with Err after the iteration has finished.
// The end-of-list marker ErrStopIteration is not reported as an error.
func (si *SubvolumeIterator) Err() error {
	if errors.Is(si.err, ErrStopIteration) {
		return nil
	}
	return si.err
}

// Close destroys the underlying C btrfs_util_subvolume_iterator and marks the
// SubvolumeIterator as closed.
//
// Calling Close more than once has no effect.
func (si *SubvolumeIterator) Close() {
	if si.cPtr != nil {
		C.btrfs_util_subvolume_iter_destroy(si.cPtr)
		si.cPtr = nil
	}
}
