// TAREA: test del enmascarado de device ids ("DEV-****-XC54" para non-admin).
package vehicles

import "testing"

func TestMaskDeviceID(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"DEV-1234-XC54", "DEV-****-XC54"},
		{"DEV-AB-CD-EF", "DEV-****-****-EF"},
		{"DEV-1234", "DEV-****"},
		{"ABCDEFGH", "A******H"},
		{"AB", "**"},
		{"", ""},
	}

	for _, tc := range cases {
		if got := MaskDeviceID(tc.in); got != tc.want {
			t.Errorf("MaskDeviceID(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
