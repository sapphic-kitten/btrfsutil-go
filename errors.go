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

//go:generate sh -c "go run cmd/gen_errors.go $(pkg-config --variable=includedir libbtrfsutil)/btrfsutil.h"

// #cgo LDFLAGS: -lbtrfsutil
// #include <btrfsutil.h>
import "C"
import (
	"fmt"
)

type ErrorCode uint32

func (e ErrorCode) Error() string {
	return C.GoString(C.btrfs_util_strerror(C.enum_btrfs_util_error(e)))
}

type Error struct {
	Code ErrorCode
	Err  error
}

func (e Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.Err)
	}
	return e.Code.Error()
}

func (e Error) Unwrap() []error {
	if e.Err != nil {
		return []error{e.Code, e.Err}
	}
	return []error{e.Code}
}

func wrapCError(cError C.enum_btrfs_util_error, err error) error {
	if cError == C.BTRFS_UTIL_OK {
		return nil
	}
	return Error{ErrorCode(cError), err}
}
