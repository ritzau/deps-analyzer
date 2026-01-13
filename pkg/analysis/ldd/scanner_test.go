package ldd

import (
	"reflect"
	"testing"
)

func TestScanLinux(t *testing.T) {
	mockOutput := []byte(`
	linux-vdso.so.1 (0x00007ffc5d7dd000)
	libpthread.so.0 => /lib/x86_64-linux-gnu/libpthread.so.0 (0x00007f0c2a559000)
	libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6 (0x00007f0c2a367000)
	/lib64/ld-linux-x86-64.so.2 (0x00007f0c2a57e000)
	`)

	scanner := &Scanner{
		Executor: func(name string, args ...string) ([]byte, error) {
			return mockOutput, nil
		},
	}

	libs, err := scanner.scanLinux("dummy_binary")
	if err != nil {
		t.Fatalf("scanLinux failed: %v", err)
	}

	expected := []string{
		"/lib/x86_64-linux-gnu/libpthread.so.0",
		"/lib/x86_64-linux-gnu/libc.so.6",
		"/lib64/ld-linux-x86-64.so.2",
	}

	if !reflect.DeepEqual(libs, expected) {
		t.Errorf("Expected %v, got %v", expected, libs)
	}
}

func TestScanMacOS(t *testing.T) {
	mockOutput := []byte(`
/path/to/binary:
	/usr/lib/libSystem.B.dylib (compatibility version 1.0.0, current version 1311.100.3)
	@rpath/libfoo.dylib (compatibility version 0.0.0, current version 0.0.0)
	`)

	scanner := &Scanner{
		Executor: func(name string, args ...string) ([]byte, error) {
			return mockOutput, nil
		},
	}

	libs, err := scanner.scanMacOS("dummy_binary")
	if err != nil {
		t.Fatalf("scanMacOS failed: %v", err)
	}

	expected := []string{
		"/usr/lib/libSystem.B.dylib",
		"@rpath/libfoo.dylib",
	}

	if !reflect.DeepEqual(libs, expected) {
		t.Errorf("Expected %v, got %v", expected, libs)
	}
}
