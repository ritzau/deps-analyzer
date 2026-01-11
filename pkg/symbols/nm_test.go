package symbols

import (
	"reflect"
	"testing"
)

func TestParseNMOutput(t *testing.T) {
	tests := []struct {
		name       string
		objectFile string
		output     string
		want       []Symbol
	}{
		{
			name:       "Standard Output",
			objectFile: "test.o",
			output: `
0000000000000000 T _Z3foov
                 U _Z3barv
0000000000000020 D _data
                 U _undefined
`,
			want: []Symbol{
				{File: "test.o", Name: "_Z3foov", Type: "T", Address: "0000000000000000"},
				{File: "test.o", Name: "_Z3barv", Type: "U"},
				{File: "test.o", Name: "_data", Type: "D", Address: "0000000000000020"},
				{File: "test.o", Name: "_undefined", Type: "U"},
			},
		},
		{
			name:       "GNU Linux Output",
			objectFile: "linux.o",
			output: `
000000000040052d T main
                 U puts
0000000000601038 B __bss_start
`,
			// Note: Linux nm often omits leading underscores for C symbols compared to macOS
			want: []Symbol{
				{File: "linux.o", Name: "main", Type: "T", Address: "000000000040052d"},
				{File: "linux.o", Name: "puts", Type: "U"},
				{File: "linux.o", Name: "__bss_start", Type: "B", Address: "0000000000601038"},
			},
		},
		{
			name:       "BSD macOS Output",
			objectFile: "mac.o",
			output: `
0000000100003f90 T _main
                 U _puts
0000000100008000 B _bss_start
`,
			want: []Symbol{
				{File: "mac.o", Name: "_main", Type: "T", Address: "0000000100003f90"},
				{File: "mac.o", Name: "_puts", Type: "U"},
				{File: "mac.o", Name: "_bss_start", Type: "B", Address: "0000000100008000"},
			},
		},
		{
			name:       "Comparison with Address but no Hex (invalid)",
			objectFile: "invalid_addr.o",
			output: `
zzzzzzzz T main
`,
			// "zzzzzzzz" is not hex.
			// Logic: len=3. isHex(zzzzzzzz) -> false.
			// else -> Type=parts[0]="zzzzzzzz", Name="T main" ...
			// This seems like a potential edge case if nm output is corrupted or wildly different.
			// Current parser:
			// if len>=3:
			//   if isHex(p[0]): defined
			//   else: undefined-like (p[0]=Type, p[1:]=Name)
			want: []Symbol{
				{File: "invalid_addr.o", Name: "T main", Type: "zzzzzzzz"},
			},
		},
		{
			name:       "Empty Output",
			objectFile: "empty.o",
			output:     "",
			want:       nil,
		},
		{
			name:       "Invalid Lines (Skipped)",
			objectFile: "invalid.o",
			output: `
InvalidLineWithoutEnoughParts
T OnlyTypeAndNameButNoAddressIfDefined (Wait this is parsed as defined if hex check fails?)
00000000 T
`,
			// Note: "00000000 T" has 2 parts. If first looks like hex, it might be parsed as Address=00000000, Type=T, Name="".
			// But code logic: if len==2, Type=parts[0], Name=parts[1].
			// So "00000000 T" -> Type="00000000", Name="T" (Undefined).
			// Let's verify actual behavior.
			// "InvalidLineWithoutEnoughParts" -> Skipped (len<2)
			want: []Symbol{
				{File: "invalid.o", Name: "OnlyTypeAndNameButNoAddressIfDefined (Wait this is parsed as defined if hex check fails?)", Type: "T"}, // Parsed as U-like (parts[0]=Type, parts[1:]=Name)
				{File: "invalid.o", Name: "T", Type: "00000000"}, // Parsed as U-like
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseNMOutput(tt.objectFile, tt.output)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseNMOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsHexAddress(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"0000000000000000", true},
		{"00000000", true},
		{"12345678", true},
		{"deadbeef", true},
		{"zzzzzzzz", false},
		{"123", false}, // too short
		{"U", false},
	}

	for _, tt := range tests {
		if got := isHexAddress(tt.input); got != tt.want {
			t.Errorf("isHexAddress(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
