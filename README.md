# btrfsutil-go

Go bindings for [libbtrfsutil](https://github.com/kdave/btrfs-progs/blob/devel/libbtrfsutil/README.md), the C utility library shipped with [btrfs-progs](https://github.com/kdave/btrfs-progs/blob/devel/README.md). The bindings use CGo and mirror the C API almost 1:1, with small deviations for Go idioms.

## Requirements

- Linux with a Btrfs filesystem.
- `libbtrfsutil` from btrfs-progs, resolvable via `pkg-config`.
- Go ≥ 1.27 (requires `uuid` from the [standard library](https://pkg.go.dev/uuid)).

## Usage
Install like any other Go library:

```sh
go get github.com/sapphic-kitten/btrfsutil-go
```

Then import it:

```go
import "github.com/sapphic-kitten/btrfsutil-go"

err := btrfsutil.FsSync("/mnt/btrfs")
```

## Overview

The bindings mirror libbtrfsutil almost 1:1. Every C function and type has a Go counterpart.
Function and type names are derived from their names in C.

`SubvolumeInfo` mirrors `struct btrfs_util_subvolume_info`. The following fields have been converted to Go types:

| Fields | C type | Go type |
|---|---|---|
| `uuid`, `parent_uuid`, `received_uuid` | `uint8_t[16]` | `uuid.UUID` |
| `ctime`, `otime`, `stime`, `rtime` | `struct timespec` | `time.Time` |

### Subvolume iteration (`subvolumeIterator.go`)

`NewSubvolumeIterator(path, topId, postOrder)` and `NewSubvolumeIteratorFd(fd, …)` return a `SubvolumeIterator`. It wraps `btrfs_util_subvolume_iterator`, `btrfs_util_subvolume_iterator_next` and `btrfs_util_subvolume_iterator_next_info`.

```go
// BTRFS_FS_TREE_OBJECTID (i.e. 5) is the top-level subvolume (lists everything).
it, err := btrfsutil.NewSubvolumeIterator(mount, 5, false)
if err != nil {
    return err
}
defer it.Close()

for path, info := range it.ListSubvolumeInfo() {
    fmt.Printf("%s %d\n", path, info.Id)
}
if err := it.Err(); err != nil {
    return err
}
```

| Method | Returns |
|---|---|
| `ListSubvolumes()` | `iter.Seq2` of `(path, id)` |
| `ListSubvolumeInfo()` | `iter.Seq2` of `(path, *SubvolumeInfo)` |
| `Next()` | one `(path, id)` and a validity flag |
| `NextSubvolumeInfo()` | one `(path, *SubvolumeInfo)` and a validity flag |
| `Fd()` | the iterator's file descriptor, only while it is open |
| `Err()` | the last error, excluding the `ErrStopIteration` end marker |
| `Close()` | destroys the C iterator, idempotent |

- Yielded paths are relative to the subvolume ID the iterator was created with.
- The validity flag is `false` once the list is exhausted or an error occurred.
- `ListSubvolumes()`, `ListSubvolumeInfo()`, `Next()` and `NextSubvolumeInfo()` all read from the same C iterator. When one of them reaches the end of the list (`ErrStopIteration`), the others are exhausted as well.
- `Close()` frees the underlying C structure and must be called after use.

### Qgroup inheritance (`qgroup.go`)

`NewQgroupInherit()` creates an empty qgroup inheritance specifier, mirroring `btrfs_util_create_qgroup_inherit`.

- `AddGroup(groupId)` adds a qgroup to inherit from.
- `GetGroups()` returns the qgroups added so far.
- `Close()` frees the underlying C structure and must be called after use.

### Error handling (`errors.go`)

C errors are wrapped in an `Error` that carries both the libbtrfsutil error code and the `errno`. Each part can be matched separately with `errors.Is`:

```go
err := btrfsutil.FsSync("/mnt/btrfs")

// Match the libbtrfsutil error code.
if errors.Is(err, btrfsutil.ErrOpenFailed) {
    // ...
}

// Match the errno.
if errors.Is(err, syscall.ENOENT) {
    // ...
}
```

- `Error.String()` returns the message from `btrfs_util_strerror`.
- `Error.Unwrap()` returns `[code, errno]`, so `errors.Is`, `errors.Join` and `switch` matching work on both parts.
- The error code constants (`ErrNoMemory`, `ErrSyncFailed`, …) are generated from the C header, see [Code generation](#code-generation).

## Memory management

- Arguments are copied into the corresponding C types where needed and freed automatically with `defer`.
- Strings and arrays allocated by C are copied into Go values and freed before returning.
- `SubvolumeInfo`, `[]uint64` and the iterator's `Fd()` are plain Go values.
- `SubvolumeIterator` and `QgroupInherit` are the only resources the caller releases, with `Close()`.

## Development

### Nix shell

`flake.nix` and `.envrc` (`use flake`) provide a dev shell with `go`, `gopls`, `delve`, `gotools`, `pkg-config`, `btrfs-progs` and `util-linux` installed.  
To preserve the Nix environment for the root-only integration tests a `sudo` wrapper (`sudo -E env "PATH=$PATH"`) is also included.

### Testing

- **Unit tests** (`btrfsutil_test.go`, `errors_test.go`, `qgroup_test.go`) cover struct marshalling, C-Go type conversion, error wrapping and basic qgroup tests. They use CGo-generated types such as `_Ctype_struct_btrfs_util_subvolume_info`, so they only compile if the bindings compile.
- **Integration tests** (`integration_tests/`, build tag `integration`) create small throwaway loopback images (~16 MiB) with `mkfs.btrfs`, mount them, create a subvolume layout and exercise the path- and fd-based API. These operations require elevated privileges, so they must run as root; otherwise they abort with `Integration tests must be run as root`.

### Code generation

`cmd/gen_errors.go` generates `errors_generated.go` from the C header. The `//go:generate` directive is declared in `errors.go`. To run the generator manually:

```sh
go run cmd/gen_errors.go <path to btrfsutil.h>
```

The generator extracts the `enum btrfs_util_error` body with a regular expression, skips `BTRFS_UTIL_OK` and emits `ErrorCode` constants with Go-style names (`BTRFS_UTIL_ERROR_SNAP_CREATE_FAILED` → `ErrSnapCreateFailed`).

Do not edit `errors_generated.go` by hand. Regenerate it and re-run the integration tests whenever btrfs-progs changes the error enum.

## Limitations

- All limitations of `libbtrfsutil` apply.
- The bindings depend on the C library version at build time.

## License

Copyright © 2022 Jana Marlou Rettig. Licensed under the GNU Lesser General Public License v2.1 or later (see `LICENSE`), matching libbtrfsutil.
