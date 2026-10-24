package semver

import "testing"

func TestParse(t *testing.T) {
	ok := map[string]Version{
		"1.2.3":    {1, 2, 3},
		"v1.2.3":   {1, 2, 3},
		"0.0.0":    {0, 0, 0},
		"10.20.30": {10, 20, 30},
	}
	for in, want := range ok {
		got, err := Parse(in)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "v", "1", "1.2", "1.2.3.4", "1.2.3-rc.1", "1.2.3+build", "a.b.c", "-1.2.3", "1.-2.3", "01.2.3", "1.2.x", " 1.2.3"} {
		if v, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) = %v, want error", in, v)
		}
	}
}

func TestStringAndLess(t *testing.T) {
	if s := (Version{1, 2, 3}).String(); s != "1.2.3" {
		t.Errorf("String = %q", s)
	}
	cases := []struct {
		a, b Version
		less bool
	}{
		{Version{1, 2, 3}, Version{1, 2, 4}, true},
		{Version{1, 2, 3}, Version{1, 3, 0}, true},
		{Version{1, 9, 9}, Version{2, 0, 0}, true},
		{Version{1, 2, 3}, Version{1, 2, 3}, false},
		{Version{2, 0, 0}, Version{1, 9, 9}, false},
	}
	for _, c := range cases {
		if got := c.a.Less(c.b); got != c.less {
			t.Errorf("%v.Less(%v) = %v", c.a, c.b, got)
		}
	}
}

func TestBump(t *testing.T) {
	v := Version{1, 2, 3}
	cases := map[Bump]Version{
		None:  {1, 2, 3},
		Patch: {1, 2, 4},
		Minor: {1, 3, 0},
		Major: {2, 0, 0},
	}
	for k, want := range cases {
		if got := v.Bump(k); got != want {
			t.Errorf("Bump(%d) = %v, want %v", k, got, want)
		}
	}
}
