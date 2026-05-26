package categories

import "testing"

func TestNormalizeSlug(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Tutorials", "tutorials"},
		{"TEST", "test"},
		{"  Some Title!  ", "some-title"},
		{"  --hello world--  ", "hello-world"},
		{"a__b", "a-b"},
		{"a---b", "a-b"},
		{"foo/bar/baz", "foo-bar-baz"},
		{"with 123 numbers", "with-123-numbers"},
		{"café", "caf"},   // non-ASCII letters are stripped
		{"한글", ""},       // entirely non-ASCII collapses to empty
		{"  ", ""},        // only whitespace
		{"-----", ""},     // only separators
		{"!@#$%", ""},     // only punctuation
		{"already-clean", "already-clean"},
	}

	for _, c := range cases {
		got := NormalizeSlug(c.in)
		if got != c.want {
			t.Errorf("NormalizeSlug(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeSlug_IsIdempotent(t *testing.T) {
	inputs := []string{
		"Hello World",
		"foo--bar",
		"  trim  me  ",
		"a/b/c",
	}
	for _, in := range inputs {
		once := NormalizeSlug(in)
		twice := NormalizeSlug(once)
		if once != twice {
			t.Errorf("NormalizeSlug not idempotent for %q: once=%q twice=%q", in, once, twice)
		}
	}
}
