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

// btrfsutil is a wrapper for the libbtrfsutil C library
package btrfsutil

//#cgo LDFLAGS: -lbtrfsutil
//#include <stdlib.h>
//#include <btrfsutil.h>
import "C"
import (
	"time"
	"unsafe"
	"uuid"
)

// A SubvolumeInfo is a representation of a Btrfs subvolume or snapshot.
type SubvolumeInfo struct {
	Id           uint64
	ParentId     uint64
	DirId        uint64
	Flags        uint64
	UUID         uuid.UUID
	ParentUUID   uuid.UUID
	ReceivedUUID uuid.UUID
	Generation   uint64
	Ctransid     uint64
	Otransid     uint64
	Stransid     uint64
	Rtransid     uint64
	Ctime        time.Time
	Otime        time.Time
	Stime        time.Time
	Rtime        time.Time
}

// newSubvolumeInfo creates a SubvolumeInfo from a C struct btrfs_util_subvolume_info.
//
// info is the C structure to copy fields from.
func newSubvolumeInfo(info *C.struct_btrfs_util_subvolume_info) *SubvolumeInfo {
	subvol := SubvolumeInfo{
		Id:           uint64(info.id),
		ParentId:     uint64(info.parent_id),
		DirId:        uint64(info.dir_id),
		Flags:        uint64(info.flags),
		UUID:         *(*uuid.UUID)(unsafe.Pointer(&info.uuid)),
		ParentUUID:   *(*uuid.UUID)(unsafe.Pointer(&info.parent_uuid)),
		ReceivedUUID: *(*uuid.UUID)(unsafe.Pointer(&info.received_uuid)),
		Generation:   uint64(info.generation),
		Ctransid:     uint64(info.ctransid),
		Otransid:     uint64(info.otransid),
		Stransid:     uint64(info.stransid),
		Rtransid:     uint64(info.rtransid),
		Ctime:        time.Unix(int64(info.ctime.tv_sec), int64(info.ctime.tv_nsec)),
		Otime:        time.Unix(int64(info.otime.tv_sec), int64(info.otime.tv_nsec)),
		Stime:        time.Unix(int64(info.stime.tv_sec), int64(info.stime.tv_nsec)),
		Rtime:        time.Unix(int64(info.rtime.tv_sec), int64(info.rtime.tv_nsec)),
	}
	return &subvol
}

// uint64SliceFromC copies a C array of uint64_t values pointed to by ptr
// with size length into a new Go slice.
func uint64SliceFromC(ptr *C.uint64_t, length C.size_t) []uint64 {
	src := unsafe.Slice((*uint64)(unsafe.Pointer(ptr)), int(length))
	dst := make([]uint64, int(length))
	copy(dst, src)
	return dst
}

// FsSync forces a sync on a specific Btrfs filesystem.
//
// path is a path on a Btrfs filesystem.
func FsSync(path string) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	errorCode, err := C.btrfs_util_fs_sync(cPath)
	return wrapCError(errorCode, err)
}

// See FsSync.
func FsSyncFd(fd uintptr) error {
	errorCode, err := C.btrfs_util_fs_sync_fd(C.int(fd))
	return wrapCError(errorCode, err)
}

// FsStartSync starts a sync on a specific Btrfs filesystem but does not wait
// for it to complete.
//
// path is a path on a Btrfs filesystem.
// Returns the transaction ID which can be waited on with FsWaitSync.
func FsStartSync(path string) (uint64, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var transid C.uint64_t

	errorCode, err := C.btrfs_util_fs_start_sync(cPath, &transid)
	if errorCode != C.BTRFS_UTIL_OK {
		return 0, wrapCError(errorCode, err)
	}
	return uint64(transid), nil
}

// See FsStartSync.
func FsStartSyncFd(fd uintptr) (uint64, error) {
	var transid C.uint64_t

	errorCode, err := C.btrfs_util_fs_start_sync_fd(C.int(fd), &transid)
	if errorCode != C.BTRFS_UTIL_OK {
		return 0, wrapCError(errorCode, err)
	}
	return uint64(transid), nil
}

// FsWaitSync waits for a transaction with a given ID to be synced.
//
// path is a path on a Btrfs filesystem. transid is the transaction ID to wait
// for, or zero for the current transaction.
func FsWaitSync(path string, transid uint64) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	errorCode, err := C.btrfs_util_fs_wait_sync(cPath, C.uint64_t(transid))
	return wrapCError(errorCode, err)
}

// See FsWaitSync.
func FsWaitSyncFd(fd uintptr, transid uint64) error {
	errorCode, err := C.btrfs_util_fs_wait_sync_fd(C.int(fd), C.uint64_t(transid))
	return wrapCError(errorCode, err)
}

// SubvolumeIsValid returns whether a given path is a Btrfs subvolume.
//
// path is the path to check.
func SubvolumeIsValid(path string) (bool, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	errorCode, err := C.btrfs_util_subvolume_is_valid(cPath)
	return errorCode == C.BTRFS_UTIL_OK, wrapCError(errorCode, err)
}

// See SubvolumeIsValid.
func SubvolumeIsValidFd(fd uintptr) (bool, error) {
	errorCode, err := C.btrfs_util_subvolume_is_valid_fd(C.int(fd))
	return errorCode == C.BTRFS_UTIL_OK, wrapCError(errorCode, err)
}

// SubvolumeGetId gets the ID of the subvolume containing a path.
//
// path is a path on a Btrfs filesystem.
func SubvolumeGetId(path string) (uint64, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var id_ret C.uint64_t
	errorCode, err := C.btrfs_util_subvolume_get_id(cPath, &id_ret)
	if errorCode != C.BTRFS_UTIL_OK {
		return 0, wrapCError(errorCode, err)
	}

	return uint64(id_ret), nil
}

// See SubvolumeGetId.
func SubvolumeGetIdFd(fd uintptr) (uint64, error) {
	var id_ret C.uint64_t

	errorCode, err := C.btrfs_util_subvolume_get_id_fd(C.int(fd), &id_ret)
	if errorCode != C.BTRFS_UTIL_OK {
		return 0, wrapCError(errorCode, err)
	}

	return uint64(id_ret), nil
}

// SubvolumeGetPath gets the path of the subvolume with a given ID relative to
// the filesystem root.
//
// path is a path on a Btrfs filesystem.
// id is the ID of the subvolume; if zero, the subvolume ID of path is used.
func SubvolumeGetPath(path string, id uint64) (string, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var cSubvolPath *C.char
	errorCode, err := C.btrfs_util_subvolume_get_path(cPath, C.uint64_t(id), &cSubvolPath)

	subvolPath := C.GoString(cSubvolPath)
	C.free(unsafe.Pointer(cSubvolPath))
	return subvolPath, wrapCError(errorCode, err)
}

// See SubvolumeGetPath.
func SubvolumeGetPathFd(fd uintptr, id uint64) (string, error) {
	var cSubvolPath *C.char
	errorCode, err := C.btrfs_util_subvolume_get_path_fd(C.int(fd), C.uint64_t(id), &cSubvolPath)

	subvolPath := C.GoString(cSubvolPath)
	C.free(unsafe.Pointer(cSubvolPath))
	return subvolPath, wrapCError(errorCode, err)
}

// SubvolumeGetInfo gets information about a subvolume.
//
// path may be any path in a Btrfs filesystem; it does not have to refer to a
// subvolume unless id is zero. id is the ID of the subvolume to get
// information about, or zero to use the subvolume ID of path.
func SubvolumeGetInfo(path string, id uint64) (*SubvolumeInfo, error) {
	var info C.struct_btrfs_util_subvolume_info

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	errorCode, err := C.btrfs_util_subvolume_get_info(cPath, C.uint64_t(id), &info)
	if errorCode != C.BTRFS_UTIL_OK {
		return nil, wrapCError(errorCode, err)
	}
	return newSubvolumeInfo(&info), nil
}

// See SubvolumeGetInfo.
func SubvolumeGetInfoFd(fd uintptr, id uint64) (*SubvolumeInfo, error) {
	var info C.struct_btrfs_util_subvolume_info

	errorCode, err := C.btrfs_util_subvolume_get_info_fd(C.int(fd), C.uint64_t(id), &info)
	if errorCode != C.BTRFS_UTIL_OK {
		return nil, wrapCError(errorCode, err)
	}
	return newSubvolumeInfo(&info), nil
}

// SubvolumeGetReadOnly gets whether a subvolume is read-only.
//
// path is the subvolume path.
func SubvolumeGetReadOnly(path string) (bool, error) {
	var ret C.bool

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	errorCode, err := C.btrfs_util_subvolume_get_read_only(cPath, &ret)
	return bool(ret), wrapCError(errorCode, err)
}

// See SubvolumeGetReadOnly.
func SubvolumeGetReadOnlyFd(fd uintptr) (bool, error) {
	var ret C.bool

	errorCode, err := C.btrfs_util_subvolume_get_read_only_fd(C.int(fd), &ret)
	return bool(ret), wrapCError(errorCode, err)
}

// SubvolumeSetReadOnly sets whether a subvolume is read-only.
//
// path is the subvolume path. readOnly is the new value of the read-only flag.
func SubvolumeSetReadOnly(path string, readOnly bool) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	errorCode, err := C.btrfs_util_subvolume_set_read_only(cPath, C.bool(readOnly))
	return wrapCError(errorCode, err)
}

// See SubvolumeSetReadOnly.
func SubvolumeSetReadOnlyFd(fd uintptr, readOnly bool) error {
	errorCode, err := C.btrfs_util_subvolume_set_read_only_fd(C.int(fd), C.bool(readOnly))
	return wrapCError(errorCode, err)
}

// SubvolumeGetDefault gets the default subvolume for a filesystem.
//
// path is a path on a Btrfs filesystem. Returns the ID of the default
// subvolume.
func SubvolumeGetDefault(path string) (uint64, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var id_ret C.uint64_t

	errorCode, err := C.btrfs_util_subvolume_get_default(cPath, &id_ret)
	return uint64(id_ret), wrapCError(errorCode, err)
}

// See SubvolumeGetDefault.
func SubvolumeGetDefaultFd(fd uintptr) (uint64, error) {
	var id_ret C.uint64_t
	errorCode, err := C.btrfs_util_subvolume_get_default_fd(C.int(fd), &id_ret)
	return uint64(id_ret), wrapCError(errorCode, err)
}

// SubvolumeSetDefault sets the default subvolume for a filesystem.
//
// path may be any path in a Btrfs filesystem; it does not have to refer to a
// subvolume unless id is zero. id is the ID of the subvolume to set as the
// default, or zero to use the subvolume ID of path.
func SubvolumeSetDefault(path string, id uint64) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	cError, err := C.btrfs_util_subvolume_set_default(cPath, C.uint64_t(id))
	return wrapCError(cError, err)
}

// See SubvolumeSetDefault.
func SubvolumeSetDefaultFd(fd uintptr, id uint64) error {
	cError, err := C.btrfs_util_subvolume_set_default_fd(C.int(fd), C.uint64_t(id))
	return wrapCError(cError, err)

}

// SubvolumeCreate creates a new subvolume at path.
// qgroupInherit is a qgroup inheritance specifier (see QgroupInherit), or
// nil for none.
func SubvolumeCreate(path string, qgroupInherit *QgroupInherit) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	cError, err := C.btrfs_util_subvolume_create(cPath, 0, nil, qgroupInherit.ptr())
	return wrapCError(cError, err)
}

// SubvolumeCreateFd creates a new subvolume named name inside the directory at
// parentFd. See SubvolumeCreate for more details.
func SubvolumeCreateFd(parentFd uintptr, name string, qgroupInherit *QgroupInherit) error {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	cError, err := C.btrfs_util_subvolume_create_fd(C.int(parentFd), cName, 0, nil, qgroupInherit.ptr())
	return wrapCError(cError, err)
}

// calculateSnapshotFlags combines recursive and readOnly into the
// BTRFS_UTIL_CREATE_SNAPSHOT_* bitmask used by the snapshot functions.
func calculateSnapshotFlags(recursive bool, readOnly bool) int {
	flags := 0
	if recursive {
		flags |= C.BTRFS_UTIL_CREATE_SNAPSHOT_RECURSIVE
	}

	if readOnly {
		flags |= C.BTRFS_UTIL_CREATE_SNAPSHOT_READ_ONLY
	}
	return flags
}

// SubvolumeSnapshot creates a new snapshot from a source subvolume path.
//
// source is the path of the existing subvolume to snapshot, and path is where
// to create the snapshot. recursive also snapshots subvolumes beneath source
// onto the same location on the new snapshot; readOnly creates a read-only
// snapshot. qgroupInherit is a qgroup inheritance specifier (see QgroupInherit), or
// nil for none.
//
// The recursive flag is currently implemented in userspace non-atomically,
// cannot be combined with readOnly, and requires CAP_SYS_ADMIN.
func SubvolumeSnapshot(source string, path string, recursive bool, readOnly bool, qgroupInherit *QgroupInherit) error {
	cSource := C.CString(source)
	defer C.free(unsafe.Pointer(cSource))

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	flags := calculateSnapshotFlags(recursive, readOnly)

	cError, err := C.btrfs_util_subvolume_snapshot(cSource, cPath, C.int(flags), nil, qgroupInherit.ptr())
	return wrapCError(cError, err)
}

// SubvolumeSnapshotFd creates a snapshot at path from the subvolume at fd.
// See SubvolumeSnapshot for details on recursive, readOnly and qgroupInherit.
func SubvolumeSnapshotFd(fd uintptr, path string, recursive bool, readOnly bool, qgroupInherit *QgroupInherit) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	flags := calculateSnapshotFlags(recursive, readOnly)

	cError, err := C.btrfs_util_subvolume_snapshot_fd(C.int(fd), cPath, C.int(flags), nil, qgroupInherit.ptr())
	return wrapCError(cError, err)
}

// SubvolumeSnapshotFd2 creates a snapshot named name inside the directory at
// parentFd from the subvolume at fd.
// See SubvolumeSnapshot for details on recursive, readOnly and qgroupInherit.
func SubvolumeSnapshotFd2(fd uintptr, parentFd uintptr, name string, recursive bool, readOnly bool, qgroupInherit *QgroupInherit) error {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	flags := calculateSnapshotFlags(recursive, readOnly)

	cError, err := C.btrfs_util_subvolume_snapshot_fd2(C.int(fd), C.int(parentFd), cName, C.int(flags), nil, qgroupInherit.ptr())
	return wrapCError(cError, err)
}

// SubvolumeDelete deletes a subvolume or snapshot.
//
// path is the path of the subvolume to delete. If recursive is true,
// subvolumes beneath path are deleted first; otherwise, deleting a subvolume
// with child subvolumes is an error.
func SubvolumeDelete(path string, recursive bool) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	flags := 0
	if recursive {
		flags |= C.BTRFS_UTIL_DELETE_SUBVOLUME_RECURSIVE
	}

	cError, err := C.btrfs_util_subvolume_delete(cPath, C.int(flags))
	return wrapCError(cError, err)
}

// SubvolumeDeleteFd deletes the subvolume named name inside the directory at
// parentFd. See SubvolumeDelete for details on recursive.
func SubvolumeDeleteFd(parentFd uintptr, name string, recursive bool) error {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	flags := 0
	if recursive {
		flags |= C.BTRFS_UTIL_DELETE_SUBVOLUME_RECURSIVE
	}

	cError, err := C.btrfs_util_subvolume_delete_fd(C.int(parentFd), cName, C.int(flags))
	return wrapCError(cError, err)
}

// SubvolumeDeleteByIdFd deletes a subvolume or snapshot identified by its
// subvolume ID.
//
// parentFd is a file descriptor of the subvolume's parent directory;
// subvolid is the subvolume ID of the subvolume or snapshot to be deleted.
func SubvolumeDeleteByIdFd(parentFd uintptr, subvolid uint64) error {
	cError, err := C.btrfs_util_subvolume_delete_by_id_fd(C.int(parentFd), C.uint64_t(subvolid))
	return wrapCError(cError, err)
}

// SubvolumeListDeleted returns IDs of subvolumes which have been deleted but
// not yet cleaned up.
//
// path is a path on a Btrfs filesystem.
func SubvolumeListDeleted(path string) ([]uint64, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var n C.size_t
	var cIds *C.uint64_t

	cError, err := C.btrfs_util_subvolume_list_deleted(cPath, &cIds, &n)
	if err := wrapCError(cError, err); err != nil {
		return nil, err
	}

	ids := uint64SliceFromC(cIds, n)
	defer C.free(unsafe.Pointer(cIds))

	return ids, nil
}

// See SubvolumeListDeleted.
func SubvolumeListDeletedFd(fd uintptr) ([]uint64, error) {
	var n C.size_t
	var cIds *C.uint64_t

	cError, err := C.btrfs_util_subvolume_list_deleted_fd(C.int(fd), &cIds, &n)
	if err := wrapCError(cError, err); err != nil {
		return nil, err
	}

	ids := uint64SliceFromC(cIds, n)
	defer C.free(unsafe.Pointer(cIds))

	return ids, nil
}
