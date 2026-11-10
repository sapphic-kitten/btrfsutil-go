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
	"unsafe"
)

// A QgroupInherit is a qgroup inheritance specifier, as used by the SubvolumeCreate
// and SubvolumeSnapshot functions.
type QgroupInherit struct {
	cPtr *C.struct_btrfs_util_qgroup_inherit
}

// NewQgroupInherit creates an empty qgroup inheritance specifier, to be used
// with SubvolumeCreate or SubvolumeSnapshot.
//
// The returned specifier must be destroyed with Close.
// A fresh specifier inherits from no qgroups and may be used
// when no inheritance is wanted.
func NewQgroupInherit() (*QgroupInherit, error) {
	q := new(QgroupInherit)
	cError, err := C.btrfs_util_qgroup_inherit_create(0, &q.cPtr)
	if err := wrapCError(cError, err); err != nil {
		return nil, err
	}

	return q, nil
}

func (q *QgroupInherit) ptr() *C.struct_btrfs_util_qgroup_inherit {
	if q == nil {
		return nil
	}
	return q.cPtr
}

// Close destroys the unerlaying C qgroup inheritance specifier and
// marks the specifier as closed, so it must not be used afterwards.
//
// Calling Close more than once has no effect. Calling Close on an empty
// inheritance specifier has no effect.
func (q *QgroupInherit) Close() {
	if q.cPtr != nil {
		C.btrfs_util_qgroup_inherit_destroy(q.cPtr)
		q.cPtr = nil
	}
}

// AddGroup adds inheritance from the given qgroup
//
// groupId is the ID of the qgroup to inherit from.
func (q *QgroupInherit) AddGroup(groupId uint64) error {
	cError, err := C.btrfs_util_qgroup_inherit_add_group(&q.cPtr, C.uint64_t(groupId))
	return wrapCError(cError, err)
}

// GetGroups returns the IDs of the qgroups a QgroupInherit inherits from, in the
// order they were added.
func (q *QgroupInherit) GetGroups() []uint64 {
	var n C.size_t
	var cGroups *C.uint64_t
	defer C.free(unsafe.Pointer(cGroups))

	C.btrfs_util_qgroup_inherit_get_groups(q.cPtr, &cGroups, &n)
	return uint64SliceFromC(cGroups, n)
}
