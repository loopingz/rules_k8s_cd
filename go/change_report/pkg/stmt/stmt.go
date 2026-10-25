// Package stmt splits a Starlark file (BUILD, MODULE.bazel) into its top-level
// statements, to tell which of them a commit changed.
package stmt

import (
	"regexp"
	"strings"
)

// Stmt is a top-level statement.
type Stmt struct {
	// Text is the statement with comments removed and whitespace outside
	// strings collapsed: two statements are equal when their Text is.
	Text string
	// Assign is the variable a `x = ...` statement assigns, "" otherwise.
	Assign string
	// Call is the function a statement calls, possibly dotted (`oci.pull`),
	// for `f(...)` and `x = f(...)`; "" for any other statement.
	Call string
	// Args is the text between the call's parentheses.
	Args string
}

var (
	callRe   = regexp.MustCompile(`^(?:([A-Za-z_]\w*)\s*=\s*)?([A-Za-z_][\w.]*)\s*\((.*)\)$`)
	assignRe = regexp.MustCompile(`^([A-Za-z_]\w*)\s*=`)
	strArgRe = regexp.MustCompile(`^\s*(?:[A-Za-z_]\w*\s*=\s*)?"([^"\\]*)"`)
)

// Split returns the top-level statements of src, in order.
func Split(src string) []Stmt {
	var out []Stmt
	var cur strings.Builder
	depth := 0
	flush := func() {
		t := strings.TrimSpace(cur.String())
		cur.Reset()
		if t == "" {
			return
		}
		s := Stmt{Text: t}
		if m := callRe.FindStringSubmatch(t); m != nil {
			s.Assign, s.Call, s.Args = m[1], m[2], m[3]
		} else if m := assignRe.FindStringSubmatch(t); m != nil {
			s.Assign = m[1]
		}
		out = append(out, s)
	}
	space := false
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch {
		case c == '#':
			for i < len(src) && src[i] != '\n' {
				i++
			}
			i--
		case c == '"' || c == '\'':
			j := skipString(src, i)
			separate(&cur, &space)
			cur.WriteString(src[i:j])
			i = j - 1
		case c == '(' || c == '[' || c == '{':
			depth++
			writeByte(&cur, c, &space)
		case c == ')' || c == ']' || c == '}':
			depth--
			// a trailing comma does not change the statement
			trimComma(&cur)
			space = false
			cur.WriteByte(c)
		case c == '\n' && depth <= 0:
			depth = 0
			flush()
			space = false
		case c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\\':
			space = true
		default:
			writeByte(&cur, c, &space)
		}
	}
	flush()
	return out
}

func writeByte(b *strings.Builder, c byte, space *bool) {
	separate(b, space)
	b.WriteByte(c)
}

// separate writes the pending whitespace as one space, except after an
// opening bracket.
func separate(b *strings.Builder, space *bool) {
	if *space && b.Len() > 0 {
		if s := b.String(); !strings.HasSuffix(s, "(") && !strings.HasSuffix(s, "[") && !strings.HasSuffix(s, "{") {
			b.WriteByte(' ')
		}
	}
	*space = false
}

func trimComma(b *strings.Builder) {
	s := strings.TrimRight(b.String(), " ")
	if strings.HasSuffix(s, ",") {
		s = s[:len(s)-1]
	}
	b.Reset()
	b.WriteString(s)
}

// skipString returns the index just past the string literal (single, double
// or triple quoted) starting at i.
func skipString(s string, i int) int {
	q := s[i]
	if strings.HasPrefix(s[i:], strings.Repeat(string(q), 3)) {
		if j := strings.Index(s[i+3:], strings.Repeat(string(q), 3)); j >= 0 {
			return i + 3 + j + 3
		}
		return len(s)
	}
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++
		case q:
			return j + 1
		case '\n':
			return j
		}
	}
	return len(s)
}

// Kwarg returns the value of the string literal keyword argument key of the
// call's Args, "" when absent or not a literal.
func (s Stmt) Kwarg(key string) string {
	re := regexp.MustCompile(`(?:^|[\s,(])` + regexp.QuoteMeta(key) + `\s*=\s*"([^"\\]*)"`)
	if m := re.FindStringSubmatch(s.Args); m != nil {
		return m[1]
	}
	return ""
}

// Positional returns the n-th (0-based) argument when it is a string literal,
// positional or keyword, "" otherwise.
func (s Stmt) Positional(n int) string {
	args := splitArgs(s.Args)
	if n >= len(args) {
		return ""
	}
	if m := strArgRe.FindStringSubmatch(args[n]); m != nil {
		return m[1]
	}
	return ""
}

// splitArgs splits a call's arguments on top-level commas.
func splitArgs(a string) []string {
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(a); i++ {
		switch c := a[i]; c {
		case '"', '\'':
			i = skipString(a, i) - 1
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, a[start:i])
				start = i + 1
			}
		}
	}
	if strings.TrimSpace(a[start:]) != "" {
		out = append(out, a[start:])
	}
	return out
}

// Changed returns the statements of before and after that are not in the
// other one (as a multiset of Text): the removed ones, then the added ones.
func Changed(before, after []Stmt) []Stmt {
	count := map[string]int{}
	for _, s := range after {
		count[s.Text]++
	}
	var removed []Stmt
	for _, s := range before {
		if count[s.Text] > 0 {
			count[s.Text]--
		} else {
			removed = append(removed, s)
		}
	}
	count = map[string]int{}
	for _, s := range before {
		count[s.Text]++
	}
	var added []Stmt
	for _, s := range after {
		if count[s.Text] > 0 {
			count[s.Text]--
		} else {
			added = append(added, s)
		}
	}
	return append(removed, added...)
}
