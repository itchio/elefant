package version

import "testing"

// Greater than or equal
func Test_GTOrEq(t *testing.T) {
	type args struct {
		v1 string
		v2 string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"lesser", args{"0.9.0", "1.0.0"}, false},
		{"lesser", args{"1.0.0", "1.0.1"}, false},
		{"lesser", args{"1.0.0", "1.1.0"}, false},
		{"lesser", args{"1.0.0", "1.1.1"}, false},
		{"equal", args{"1.0.0", "1.0.0"}, true},
		{"equal", args{"12.3", "12.3"}, true},
		{"greater", args{"2.11", "2.2"}, true},
		{"greater", args{"2.3.11", "2.2"}, true},
		{"greater", args{"2.3", "2.2.11"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GTOrEq(tt.args.v1, tt.args.v2); got != tt.want {
				t.Errorf("GtOrEq() = %v, want %v", got, tt.want)
			}
		})
	}
}
