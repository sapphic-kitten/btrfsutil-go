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

import (
	"errors"
	"reflect"
	"syscall"
	"testing"
	"time"
	"unsafe"
	"uuid"
)

func TestSubvolumeInfo(t *testing.T) {
	want := &SubvolumeInfo{
		Id:           0,
		ParentId:     1,
		DirId:        2,
		Flags:        3,
		UUID:         uuid.New(),
		ParentUUID:   uuid.New(),
		ReceivedUUID: uuid.New(),
		Generation:   4,
		Ctransid:     5,
		Otransid:     6,
		Stransid:     7,
		Rtransid:     8,
		Ctime:        time.Unix(100, 200),
		Otime:        time.Unix(300, 400),
		Stime:        time.Unix(500, 600),
		Rtime:        time.Unix(700, 800),
	}

	var cInfo _Ctype_struct_btrfs_util_subvolume_info
	cInfo.id = _Ctype_uint64_t(want.Id)
	cInfo.parent_id = _Ctype_uint64_t(want.ParentId)
	cInfo.dir_id = _Ctype_uint64_t(want.DirId)
	cInfo.flags = _Ctype_uint64_t(want.Flags)
	*(*uuid.UUID)(unsafe.Pointer(&cInfo.uuid)) = want.UUID
	*(*uuid.UUID)(unsafe.Pointer(&cInfo.parent_uuid)) = want.ParentUUID
	*(*uuid.UUID)(unsafe.Pointer(&cInfo.received_uuid)) = want.ReceivedUUID
	cInfo.generation = _Ctype_uint64_t(want.Generation)
	cInfo.ctransid = _Ctype_uint64_t(want.Ctransid)
	cInfo.otransid = _Ctype_uint64_t(want.Otransid)
	cInfo.stransid = _Ctype_uint64_t(want.Stransid)
	cInfo.rtransid = _Ctype_uint64_t(want.Rtransid)
	cInfo.ctime.tv_sec = _Ctype_long(want.Ctime.Unix())
	cInfo.ctime.tv_sec = _Ctype_long(want.Ctime.Unix())
	cInfo.ctime.tv_nsec = _Ctype_long(want.Ctime.Nanosecond())
	cInfo.otime.tv_sec = _Ctype_long(want.Otime.Unix())
	cInfo.otime.tv_nsec = _Ctype_long(want.Otime.Nanosecond())
	cInfo.stime.tv_sec = _Ctype_long(want.Stime.Unix())
	cInfo.stime.tv_nsec = _Ctype_long(want.Stime.Nanosecond())
	cInfo.rtime.tv_sec = _Ctype_long(want.Rtime.Unix())
	cInfo.rtime.tv_nsec = _Ctype_long(want.Rtime.Nanosecond())

	if got := newSubvolumeInfo(&cInfo); !reflect.DeepEqual(got, want) {
		t.Fatalf("newSubvolumeInfo() = %v, want %v", got, want)
	}
}

func TestUint64SliceFromC(t *testing.T) {
	cArray := []_Ctype___uint64_t{1, 2, 3, 4, 5}
	n := _Ctype_size_t(len(cArray))
	cPtr := (*_Ctype___uint64_t)(unsafe.Pointer(&cArray[0]))

	got := uint64SliceFromC(cPtr, n)
	want := []uint64{1, 2, 3, 4, 5}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uint64SliceFromC() = %v, want %v", got, want)
	}
}

func TestCalculateSnapshotFlags(t *testing.T) {
	tests := []struct {
		name      string
		recursive bool
		readOnly  bool
		want      uint64
	}{
		{"non-recursive, read-write", false, false, 0b00},
		{"recursive, read-write", true, false, 0b01},
		{"non-recursive, read-only", false, true, 0b10},
		{"recursive, read-only", true, true, 0b11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateSnapshotFlags(tt.recursive, tt.readOnly)
			if got != int(tt.want) {
				t.Errorf("calculateSnapshotFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFsSync(t *testing.T) {
	t.Run("invalid path", func(t *testing.T) {
		want := Error{ErrOpenFailed, syscall.ENOENT}
		if err := FsSync("/does/not/exist/"); !errors.Is(err, want) {
			t.Errorf("want %v, got %v", want, err)
		}
	})

	t.Run("invalid fd", func(t *testing.T) {
		if err := FsSyncFd(^uintptr(0)); !errors.Is(err, ErrSyncFailed) {
			t.Errorf("want %v, got %v", ErrSyncFailed, err)
		}
	})
}
