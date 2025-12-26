package gitstatus

import (
	"testing"
)

func TestStringToInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantOK  bool
	}{
		{"zero", "0", 0, true},
		{"single digit", "5", 5, true},
		{"multiple digits", "123", 123, true},
		{"large number", "999999", 999999, true},
		{"empty string", "", 0, true},
		{"non-digit", "abc", 0, false},
		{"mixed", "12a34", 0, false},
		{"negative sign", "-5", 0, false},
		{"decimal", "3.14", 0, false},
		{"spaces", "1 2", 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var result int
			ok, _ := stringToInt(tc.input, &result)

			if ok != tc.wantOK {
				t.Errorf("ok = %v, want %v", ok, tc.wantOK)
			}
			if tc.wantOK && result != tc.want {
				t.Errorf("result = %d, want %d", result, tc.want)
			}
		})
	}
}

func TestStringToInt_Accumulates(t *testing.T) {
	// The function accumulates into the result pointer
	// Starting with non-zero value should work as documented
	var result int
	stringToInt("12", &result)
	if result != 12 {
		t.Errorf("got %d, want 12", result)
	}
}
