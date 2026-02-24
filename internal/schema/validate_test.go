package schema

import "testing"

func TestInstanceLocationToPointer(t *testing.T) {
	tests := []struct {
		name string
		loc  []string
		want string
	}{
		{name: "root", loc: nil, want: "/"},
		{name: "simple", loc: []string{"db", "host"}, want: "/db/host"},
		{name: "escaped", loc: []string{"a/b", "x~y"}, want: "/a~1b/x~0y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := instanceLocationToPointer(tt.loc)
			if got != tt.want {
				t.Fatalf("unexpected pointer: got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFirstLine(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: "validation failed"},
		{in: "  \n\t ", want: "validation failed"},
		{in: "one line", want: "one line"},
		{in: "first\nsecond", want: "first"},
	}

	for _, tt := range tests {
		got := firstLine(tt.in)
		if got != tt.want {
			t.Fatalf("unexpected line for %q: got %q, want %q", tt.in, got, tt.want)
		}
	}
}
