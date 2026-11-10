package btrfsutil

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorCodeError(t *testing.T) {
	if got := ErrNotBtrfs.Error(); got == "" {
		t.Fatal("ErrorCode.Error() returned an empty string")
	}
}

func TestErrorWithoutCause(t *testing.T) {
	want := ErrNotBtrfs.Error()
	got := Error{Code: ErrNotBtrfs}.Error()

	if got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestErrorWithCause(t *testing.T) {
	cause := errors.New("underlying error")
	want := fmt.Sprintf("%s: %v", ErrNotBtrfs, cause)
	got := Error{Code: ErrNotBtrfs, Err: cause}.Error()

	if got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestErrorUnwrap(t *testing.T) {
	want := ErrNotBtrfs
	err := Error{Code: want, Err: nil}

	if got := err.Unwrap(); len(got) != 1 || got[0] != want {
		t.Fatalf("Unwarp() = %v, want %v", got, want)
	}
}

func TestErrorUnwrapWithCause(t *testing.T) {
	cause := errors.New("underlying error")
	wantErr := ErrNotBtrfs
	err := Error{Code: ErrNotBtrfs, Err: cause}

	if got := err.Unwrap(); len(got) != 2 || got[0] != wantErr || got[1] != cause {
		t.Fatalf("Unwrap() = %v, want [%v, %v]", got, ErrNotBtrfs, cause)
	}

	if !errors.Is(err, wantErr) {
		t.Fatalf("errors.Is() did not match ErrorCode")
	}

	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is() did not match underlying error")
	}
}

func TestWrapCErrorOK(t *testing.T) {
	if err := wrapCError(0, nil); err != nil {
		t.Fatalf("wrapCError(OK) = %v, want nil", err)
	}
}

func TestWrapCError(t *testing.T) {
	cause := errors.New("underlying error")
	want := ErrNotBtrfs
	err := wrapCError(_Ctype_enum_btrfs_util_error(want), cause)

	if err == nil {
		t.Fatal("wrapCError() returned nil, want non-nil error")
	}

	if !errors.Is(err, want) {
		t.Fatalf("errors.Is() did not match ErrorCode")
	}

	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is() did not match underlying error")
	}

	var wrapped Error
	if !errors.As(err, &wrapped) {
		t.Fatalf("errors.As() did not find Error type")
	}
	if wrapped.Code != want || wrapped.Err != cause {
		t.Fatalf("wrapped error = %v, want code=%v, err=%v", wrapped, want, cause)
	}
}
