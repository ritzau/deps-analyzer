package ldd

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Scanner handles dynamic dependency scanning of binaries
type Scanner struct {
	// Function to execute commands (can be mocked for testing)
	Executor func(name string, args ...string) ([]byte, error)
}

// NewScanner creates a new Scanner instance
func NewScanner() *Scanner {
	return &Scanner{
		Executor: func(name string, args ...string) ([]byte, error) {
			cmd := exec.Command(name, args...)
			return cmd.CombinedOutput()
		},
	}
}

// ScanBinary runs ldd (Linux) or otool -L (macOS) on the given binary path
// and returns a list of shared library paths it depends on.
func (s *Scanner) ScanBinary(path string) ([]string, error) {
	if runtime.GOOS == "darwin" {
		return s.scanMacOS(path)
	}
	return s.scanLinux(path)
}

func (s *Scanner) scanLinux(path string) ([]string, error) {
	output, err := s.Executor("ldd", path)
	if err != nil {
		return nil, fmt.Errorf("ldd failed: %w", err)
	}

	var libs []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Typical ldd output:
		// 	linux-vdso.so.1 (0x00007ffc5d7dd000)
		// 	libpthread.so.0 => /lib/x86_64-linux-gnu/libpthread.so.0 (0x00007f0c2a559000)
		// 	libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6 (0x00007f0c2a367000)
		// 	/lib64/ld-linux-x86-64.so.2 (0x00007f0c2a57e000)

		parts := strings.Split(line, "=>")
		var libPath string
		if len(parts) == 2 {
			// case: libname => /path/to/lib (addr)
			right := strings.TrimSpace(parts[1])
			// remove address at the end: /path/to/lib (0x...)
			if idx := strings.Index(right, " ("); idx != -1 {
				libPath = right[:idx]
			} else {
				libPath = right
			}
		} else {
			// case: /path/to/lib (addr)
			// or: statically linked
			if strings.Contains(line, "statically linked") {
				continue
			}
			// remove address
			if idx := strings.Index(line, " ("); idx != -1 {
				libPath = line[:idx]
			} else {
				libPath = line
			}
		}

		if libPath != "" && libPath != "linux-vdso.so.1" { // skip virtual objects
			libs = append(libs, libPath)
		}
	}
	return libs, nil
}

func (s *Scanner) scanMacOS(path string) ([]string, error) {
	output, err := s.Executor("otool", "-L", path)
	if err != nil {
		return nil, fmt.Errorf("otool failed: %w", err)
	}
	if len(output) == 0 {
		// otool returned empty output, which can happen for some binaries
		// We don't need to log this as an error, just return an empty list.
		return []string{}, nil
	}

	var libs []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	// First line is usually the binary name followed by ":"
	if scanner.Scan() {
		// skip first line
		_ = scanner.Text()
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Typical otool -L output:
		// 	/usr/lib/libSystem.B.dylib (compatibility version 1.0.0, current version 1311.100.3)
		// 	@rpath/libfoo.dylib (compatibility version 0.0.0, current version 0.0.0)

		// Parse the path (everything before " (compatibility version ...)")
		if idx := strings.Index(line, " ("); idx != -1 {
			libPath := strings.TrimSpace(line[:idx])
			if libPath != "" {
				libs = append(libs, libPath)
			}
		}
	}
	return libs, nil
}
