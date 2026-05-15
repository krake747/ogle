// Package profiling provides one-shot pprof dump commands for runtime
// diagnosis of goroutine and heap leaks.
package profiling

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	tea "charm.land/bubbletea/v2"
)

// ProfilesDumped is delivered by DumpCmd with the paths of written profile files.
type ProfilesDumped struct {
	GoroutinePath string
	HeapPath      string
	Err           error
}

// DumpCmd returns a Cmd that writes goroutine and heap profiles to
// timestamped .pb.gz files in the current directory.
func DumpCmd() tea.Cmd {
	return func() tea.Msg {
		ts := time.Now().UTC().Format("20060102T150405Z")
		prefix := "ogle-profile-" + ts

		goPath := filepath.Join(".", prefix+"-goroutine.pb.gz")

		f, err := os.Create(goPath)
		if err != nil {
			return ProfilesDumped{
				GoroutinePath: "",
				HeapPath:      "",
				Err:           fmt.Errorf("create goroutine profile: %w", err),
			}
		}

		if err = pprof.Lookup("goroutine").WriteTo(f, 0); err != nil {
			_ = f.Close()

			return ProfilesDumped{
				GoroutinePath: "",
				HeapPath:      "",
				Err:           fmt.Errorf("write goroutine profile: %w", err),
			}
		}

		_ = f.Close()

		heapPath := filepath.Join(".", prefix+"-heap.pb.gz")

		f2, err := os.Create(heapPath)
		if err != nil {
			return ProfilesDumped{
				GoroutinePath: "",
				HeapPath:      "",
				Err:           fmt.Errorf("create heap profile: %w", err),
			}
		}

		if err = pprof.Lookup("heap").WriteTo(f2, 0); err != nil {
			_ = f2.Close()

			return ProfilesDumped{
				GoroutinePath: "",
				HeapPath:      "",
				Err:           fmt.Errorf("write heap profile: %w", err),
			}
		}

		_ = f2.Close()

		return ProfilesDumped{
			GoroutinePath: goPath,
			HeapPath:      heapPath,
			Err:           nil,
		}
	}
}
