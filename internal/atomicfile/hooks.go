package atomicfile

import "os"

type opHooks struct {
	write  func(*os.File, []byte) error
	sync   func(*os.File) error
	rename func(oldpath, newpath string) error
}

var hooks = opHooks{
	write:  func(f *os.File, b []byte) error { _, err := f.Write(b); return err }, //nolint:wrapcheck // PathError passthrough
	sync:   func(f *os.File) error { return f.Sync() },
	rename: os.Rename,
}

// Stage names accepted by InjectFault.
const (
	StageWrite  = "write"
	StageSync   = "sync"
	StageRename = "rename"
)

// InjectFault makes every Write fail with err at stage until the returned
// restore func runs. Test-only: lets callers' tests prove a failed write
// leaves the target intact. The write stage writes half of the data first, to
// mimic a torn write. It swaps an unsynchronised package-level hook, so use
// it only from tests that do not run in parallel with other writers in the
// same process.
func InjectFault(stage string, err error) (restore func()) {
	prev := hooks
	switch stage {
	case StageWrite:
		hooks.write = func(f *os.File, b []byte) error { _, _ = f.Write(b[:len(b)/2]); return err }
	case StageSync:
		hooks.sync = func(*os.File) error { return err }
	case StageRename:
		hooks.rename = func(string, string) error { return err }
	default:
		panic("atomicfile: unknown stage " + stage)
	}
	return func() { hooks = prev }
}
