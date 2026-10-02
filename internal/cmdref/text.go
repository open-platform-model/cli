package cmdref

import (
	"regexp"
	"strings"
	"unicode"
)

// block is one part of a command's description: a prose paragraph, already
// formatted as Markdown, or a preformatted run of lines kept verbatim.
type block struct {
	pre   bool
	lines []string
}

// parsedLong is a command's Long help split into the description and the
// examples the help text lists under an "Examples:" line.
type parsedLong struct {
	desc     []block
	examples []string
}

// parseLong splits a cobra Long text into prose paragraphs, preformatted
// blocks and examples.
//
// Long texts are raw Go strings, so every line after the first carries the
// indentation of the source that declares it (none, one tab, two tabs). The
// base indentation is the shallowest first line of any paragraph after the
// first; a line indented beyond the base is preformatted, a line at the base
// is prose. The first paragraph is always prose. A prose line reading
// "Examples:" starts the examples: the preformatted lines after it, up to the
// next prose line.
func parseLong(long string) parsedLong {
	lines := splitLines(long)
	paras := paragraphs(lines)
	if len(paras) == 0 {
		return parsedLong{}
	}
	base := baseIndent(lines, paras)

	var p parsedLong
	p.desc = append(p.desc, block{lines: []string{joinProse(lines[paras[0][0]:paras[0][1]])}})

	b := &longBuilder{}
	for _, para := range paras[1:] {
		b.paragraphBreak()
		for _, line := range lines[para[0]:para[1]] {
			rel, rest := relIndent(line, base)
			if rel == "" {
				b.prose(rest)
				continue
			}
			b.preLine(rel + rest)
		}
	}
	b.flush()
	p.desc = append(p.desc, b.desc...)
	p.examples = dedent(b.examples)
	return p
}

// longBuilder accumulates the blocks of a Long text line by line.
type longBuilder struct {
	desc       []block
	examples   []string
	inExamples bool
	cur        *block // open prose or preformatted block
	gap        bool   // a blank line separates the next preformatted line
}

func (b *longBuilder) paragraphBreak() {
	if b.cur != nil && !b.cur.pre {
		b.flush()
	}
	b.gap = true
}

func (b *longBuilder) prose(text string) {
	if b.cur != nil && b.cur.pre {
		b.flush()
	}
	if t := strings.TrimSpace(text); t == "Examples:" || t == "Example:" {
		b.flush()
		b.inExamples = true
		b.gap = false
		return
	}
	b.inExamples = false
	if b.cur == nil {
		b.cur = &block{}
	}
	b.cur.lines = append(b.cur.lines, text)
	b.gap = false
}

func (b *longBuilder) preLine(line string) {
	if b.inExamples {
		if b.gap && len(b.examples) > 0 {
			b.examples = append(b.examples, "")
		}
		b.examples = append(b.examples, line)
		b.gap = false
		return
	}
	if b.cur != nil && !b.cur.pre {
		b.flush()
	}
	if b.cur == nil {
		b.cur = &block{pre: true}
	} else if b.gap {
		b.cur.lines = append(b.cur.lines, "")
	}
	b.cur.lines = append(b.cur.lines, line)
	b.gap = false
}

func (b *longBuilder) flush() {
	if b.cur == nil {
		return
	}
	if b.cur.pre {
		b.desc = append(b.desc, block{pre: true, lines: dedent(b.cur.lines)})
	} else {
		b.desc = append(b.desc, block{lines: []string{joinProse(b.cur.lines)}})
	}
	b.cur = nil
}

// splitLines normalizes line endings and drops trailing whitespace.
func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	return lines
}

// paragraphs returns the [start, end) line ranges of the runs of non-blank lines.
func paragraphs(lines []string) [][2]int {
	var out [][2]int
	start := -1
	for i, l := range lines {
		switch {
		case l == "" && start >= 0:
			out = append(out, [2]int{start, i})
			start = -1
		case l != "" && start < 0:
			start = i
		}
	}
	if start >= 0 {
		out = append(out, [2]int{start, len(lines)})
	}
	return out
}

func indentOf(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// indentWidth measures indentation with a tab worth eight columns.
func indentWidth(indent string) int {
	w := 0
	for _, r := range indent {
		if r == '\t' {
			w += 8
		} else {
			w++
		}
	}
	return w
}

func baseIndent(lines []string, paras [][2]int) string {
	base, found := "", false
	for _, p := range paras[1:] {
		ind := indentOf(lines[p[0]])
		if !found || indentWidth(ind) < indentWidth(base) {
			base, found = ind, true
		}
	}
	return base
}

// relIndent returns the indentation of line beyond base and the rest of the
// line. A line not indented by base counts as prose.
func relIndent(line, base string) (rel, rest string) {
	ind := indentOf(line)
	rest = line[len(ind):]
	if !strings.HasPrefix(ind, base) {
		return "", rest
	}
	return ind[len(base):], rest
}

// dedent removes the indentation every non-blank line shares.
func dedent(lines []string) []string {
	common, found := "", false
	for _, l := range lines {
		if l == "" {
			continue
		}
		ind := indentOf(l)
		if !found {
			common, found = ind, true
			continue
		}
		for !strings.HasPrefix(ind, common) {
			common = common[:len(common)-1]
		}
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = strings.TrimPrefix(l, common)
	}
	return out
}

func joinProse(lines []string) string {
	parts := make([]string, 0, len(lines))
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}

// formatProse turns one line of plain help text into Markdown. Words that
// name code (flags, paths, placeholders, environment variables, CUE
// definitions, file names) and single-quoted spans become code spans, so the
// site never reads them as markup or links; every other Markdown character is
// escaped. The words themselves are unchanged.
func formatProse(s string) string {
	var out []string
	rest := strings.TrimSpace(s)
	for rest != "" {
		if span, after, ok := quotedSpan(rest); ok {
			out = append(out, span)
			rest = strings.TrimLeft(after, " \t")
			continue
		}
		word := rest
		if i := strings.IndexAny(rest, " \t"); i >= 0 {
			word, rest = rest[:i], strings.TrimLeft(rest[i:], " \t")
		} else {
			rest = ""
		}
		out = append(out, formatWord(word))
	}
	if len(out) > 0 {
		out[0] = escapeLineStart(out[0])
	}
	return strings.Join(out, " ")
}

// quotedSpan recognizes a single-quoted span at the start of s, such as
// 'opm module vet', and renders it as a code span with any trailing
// punctuation after it.
func quotedSpan(s string) (span, after string, ok bool) {
	if !strings.HasPrefix(s, "'") || len(s) < 3 || s[1] == ' ' {
		return "", "", false
	}
	end := strings.Index(s[1:], "'")
	if end <= 0 {
		return "", "", false
	}
	end++
	inner := s[1:end]
	if strings.HasSuffix(inner, " ") {
		return "", "", false
	}
	tail := s[end+1:]
	punct := tail[:len(tail)-len(strings.TrimLeft(tail, ".,;:!?)"))]
	next := tail[len(punct):]
	if next != "" && next[0] != ' ' && next[0] != '\t' {
		return "", "", false
	}
	return codeSpan(inner) + escapeText(punct), next, true
}

var (
	reFlag    = regexp.MustCompile(`^--?[A-Za-z]`)
	rePlus    = regexp.MustCompile(`^\+[a-z]`)
	reDef     = regexp.MustCompile(`^#[A-Za-z]`)
	reFile    = regexp.MustCompile(`^[A-Za-z0-9_.-]+\.(cue|yaml|yml|json|md|go|sh|toml)$`)
	reOrdinal = regexp.MustCompile(`^\d+[.)]`)
)

// formatWord renders one whitespace-separated word, keeping surrounding
// punctuation outside a code span.
func formatWord(w string) string {
	core := strings.TrimLeft(w, `("`)
	lead := w[:len(w)-len(core)]
	trimmed := strings.TrimRight(core, `.,;:!?)"`)
	trail := core[len(trimmed):]
	if trimmed == "" || !isCode(trimmed) {
		return escapeText(w)
	}
	if strings.HasPrefix(lead, `"`) && strings.HasPrefix(trail, `"`) {
		// A quoted code word: the code span replaces the quotes.
		lead, trail = lead[:len(lead)-1], trail[1:]
	}
	return escapeText(lead) + codeSpan(trimmed) + escapeText(trail)
}

func isCode(w string) bool {
	switch {
	case reFlag.MatchString(w), rePlus.MatchString(w), reDef.MatchString(w), reFile.MatchString(w):
		return true
	case strings.ContainsAny(w, "@_=~<>[]{}*`|\\$"):
		return strings.IndexFunc(w, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) >= 0
	case strings.Contains(w, "/"):
		return isPath(w)
	}
	return false
}

// isPath tells a path ("cue.mod/module.cue", "./src", "platform/") from a
// word pair joined by a slash ("beta/GA").
func isPath(w string) bool {
	return strings.Contains(w, ".") || strings.HasPrefix(w, "/") || strings.HasSuffix(w, "/")
}

var textEscaper = strings.NewReplacer(
	`\`, `\\`, "`", "\\`", `*`, `\*`, `_`, `\_`, `[`, `\[`, `]`, `\]`,
	`<`, `&lt;`, `>`, `&gt;`, `&`, `&amp;`, `#`, `\#`, `~`, `\~`, `|`, `\|`,
)

func escapeText(s string) string { return textEscaper.Replace(s) }

// escapeLineStart keeps a paragraph's first word from reading as a list item.
func escapeLineStart(w string) string {
	switch {
	case w == "-" || w == "+":
		return `\` + w
	case reOrdinal.MatchString(w):
		i := strings.IndexAny(w, ".)")
		return w[:i] + `\` + w[i:]
	}
	return w
}

func codeSpan(s string) string {
	ticks := "`"
	for strings.Contains(s, ticks) {
		ticks += "`"
	}
	if ticks == "`" {
		return "`" + s + "`"
	}
	return ticks + " " + s + " " + ticks
}

// fence renders lines as a fenced code block tagged lang, with a fence longer
// than any backtick run at the start of a line.
func fence(lang string, lines []string) string {
	ticks := "```"
	for _, l := range lines {
		for strings.HasPrefix(strings.TrimLeft(l, " "), ticks) {
			ticks += "`"
		}
	}
	return ticks + lang + "\n" + strings.Join(lines, "\n") + "\n" + ticks + "\n"
}

// fenceLang tags a preformatted block sh when it holds opm commands and
// nothing but commands and comments, and text otherwise.
func fenceLang(lines []string) string {
	sawCommand := false
	for _, l := range lines {
		switch {
		case l == "" || strings.HasPrefix(l, "# "):
		case strings.HasPrefix(l, "opm "):
			sawCommand = true
		default:
			return "text"
		}
	}
	if sawCommand {
		return "sh"
	}
	return "text"
}

// escapeShortcodes keeps Hugo from expanding shortcode delimiters, which it
// does even inside code fences and code spans.
func escapeShortcodes(s string) string {
	return strings.NewReplacer("{{<", "{{</*", ">}}", "*/>}}", "{{%", "{{%/*", "%}}", "*/%}}").Replace(s)
}
