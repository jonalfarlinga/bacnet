package testutil

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Frame is one datagram payload read from a .hex fixture file.
type Frame struct {
	Name string // file name and line number, for subtest names
	Data []byte
}

// LoadHexFrames reads every file matching pattern. Each line holds one frame
// in hex, where spaces may group the bytes; blank lines and lines starting
// with '#' are skipped.
func LoadHexFrames(tb testing.TB, pattern string) []Frame {
	tb.Helper()

	paths, err := filepath.Glob(pattern)
	if err != nil {
		tb.Fatalf("fixture pattern %q: %v", pattern, err)
	}
	if len(paths) == 0 {
		tb.Fatalf("no fixture files match %q", pattern)
	}

	var frames []Frame
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			tb.Fatal(err)
		}
		scanner := bufio.NewScanner(f)
		for line := 1; scanner.Scan(); line++ {
			text := strings.TrimSpace(scanner.Text())
			if text == "" || strings.HasPrefix(text, "#") {
				continue
			}
			name := fmt.Sprintf("%s:%d", filepath.Base(path), line)
			data, err := hex.DecodeString(strings.ReplaceAll(text, " ", ""))
			if err != nil {
				tb.Fatalf("%s: %v", name, err)
			}
			frames = append(frames, Frame{Name: name, Data: data})
		}
		if err := scanner.Err(); err != nil {
			tb.Fatalf("%s: %v", path, err)
		}
		f.Close()
	}
	return frames
}
