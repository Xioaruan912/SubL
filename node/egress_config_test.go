package node

import "testing"

func TestStatusAllowed(t *testing.T) {
	for _, tc := range []struct {
		code int
		spec string
		want bool
	}{
		{200, "200-399", true},
		{302, "200-399", true},
		{404, "200-399", false},
		{204, "200,204", true},
		{500, "200,204", false},
	} {
		if got := statusAllowed(tc.code, tc.spec); got != tc.want {
			t.Fatalf("statusAllowed(%d, %q) = %v, want %v", tc.code, tc.spec, got, tc.want)
		}
	}
}
