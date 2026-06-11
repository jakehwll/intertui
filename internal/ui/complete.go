package ui

import (
	"slices"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"intertui/internal/intercept"
)

// completionKey identifies one cached directory listing. local distinguishes
// the player's own system (-l/--local flag) from the connected host.
type completionKey struct {
	dir   string
	local bool
}

type completionEntry struct {
	name    string
	isDir   bool
	isEmpty bool // directory with no children ("name/ (empty)" in ls output)
}

// probeResultMsg carries the response of a silent ls probe.
type probeResultMsg struct {
	seq     int
	key     completionKey
	listing string
	err     error
}

// indexedListResultMsg carries indices parsed from a numbered list probe.
type indexedListResultMsg struct {
	seq    int
	probe  string
	indices []string
	err    error
}

// subcommandResultMsg carries subcommands parsed from probing a parent command.
type subcommandResultMsg struct {
	seq     int
	cmd     string
	listing string
	names   []string
	err     error
}

// vocabResultMsg carries the command vocabulary learned from a cmds probe
// chain: categories, all commands, and filesystem-category commands.
type vocabResultMsg struct {
	seq        int
	categories []string
	commands   []string
	filesystem []string
	err        error
}

// parseListing parses ls output into per-directory entries. Unindented
// "name/" lines open a directory whose indented children follow, so a single
// listing can populate several directories.
func parseListing(msg string) map[string][]completionEntry {
	out := make(map[string][]completionEntry)
	var stack []string
	for _, line := range strings.Split(intercept.Clean(msg), "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		depth := min(indentDepth(line), len(stack))
		stack = stack[:depth]

		isEmpty := strings.HasSuffix(name, " (empty)")
		if isEmpty {
			name = strings.TrimSuffix(name, " (empty)")
		}
		dir := strings.Join(stack, "/")
		isDir := strings.HasSuffix(name, "/")
		name = strings.TrimSuffix(name, "/")
		out[dir] = append(out[dir], completionEntry{name: name, isDir: isDir, isEmpty: isEmpty})
		if isDir {
			stack = append(stack, name)
		}
	}
	return out
}

// parseNameList parses one-name-per-line cmds output, skipping blank lines
// and headers like "Categories:".
// parseSubcommandSection extracts subcommand names under a header. Lines may
// be bare ("list"), prefixed ("bits balance"), or flag docs ("-a, …") which
// are skipped.
func parseSubcommandSection(msg, sectionHeader, cmd string) []string {
	var names []string
	seen := make(map[string]bool)
	inSection := false
	for _, line := range strings.Split(intercept.Clean(msg), "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, sectionHeader) {
			inSection = true
			continue
		}
		if !inSection {
			continue
		}
		if strings.HasPrefix(name, "-") {
			continue
		}
		if strings.HasSuffix(name, ":") && !strings.Contains(name, " ") {
			break
		}

		var sub string
		if cmd != "" && strings.HasPrefix(name, cmd+" ") {
			rest := strings.TrimSpace(strings.TrimPrefix(name, cmd+" "))
			sub, _, _ = strings.Cut(rest, " ")
		} else {
			sub, _, _ = strings.Cut(name, " ")
		}
		if sub == "" || seen[sub] {
			continue
		}
		seen[sub] = true
		names = append(names, sub)
	}
	return names
}

// parseIndexedList extracts index tokens from lines like "0: atom_probe (probe)"
// under a section header (e.g. "Software:").
func parseIndexedList(msg, sectionHeader string) []string {
	var indices []string
	inSection := false
	for _, line := range strings.Split(intercept.Clean(msg), "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, sectionHeader) {
			inSection = true
			continue
		}
		if !inSection {
			continue
		}
		before, _, ok := strings.Cut(name, ":")
		if !ok {
			continue
		}
		if idx := strings.TrimSpace(before); idx != "" {
			indices = append(indices, idx)
		}
	}
	return indices
}

func parseNameList(msg string) []string {
	var names []string
	for _, line := range strings.Split(intercept.Clean(msg), "\n") {
		name := strings.TrimSpace(line)
		if name == "" || strings.HasSuffix(name, ":") {
			continue
		}
		names = append(names, name)
	}
	return names
}

func toEntries(names []string) []completionEntry {
	entries := make([]completionEntry, len(names))
	for i, name := range names {
		entries[i] = completionEntry{name: name}
	}
	return entries
}

func dedupeSorted(ss []string) []string {
	sort.Strings(ss)
	return slices.Compact(ss)
}

func indentDepth(line string) int {
	tabs, i := 0, 0
	for ; i < len(line) && (line[i] == '\t' || line[i] == ' '); i++ {
		if line[i] == '\t' {
			tabs++
		}
	}
	if tabs == 0 && i > 0 {
		return 1
	}
	return tabs
}

func filterEntries(entries []completionEntry, style PathStyle) []completionEntry {
	switch style {
	case PathRm:
		out := make([]completionEntry, 0, len(entries))
		for _, e := range entries {
			if !e.isEmpty {
				out = append(out, e)
			}
		}
		return out
	case PathDirsOnly:
		out := make([]completionEntry, 0, len(entries))
		for _, e := range entries {
			if e.isDir {
				out = append(out, e)
			}
		}
		return out
	default:
		return entries
	}
}

// completeToken matches partial against entries, returning the replacement
// fill and the matching entries.
func completeToken(entries []completionEntry, partial string, style PathStyle) (string, []completionEntry) {
	entries = filterEntries(entries, style)
	var matches []completionEntry
	for _, e := range entries {
		if strings.HasPrefix(e.name, partial) {
			matches = append(matches, e)
		}
	}
	switch len(matches) {
	case 0:
		return "", nil
	case 1:
		if matches[0].isDir && style != PathDirsOnly {
			return matches[0].name + "/", matches
		}
		return matches[0].name, matches
	}
	fill := matches[0].name
	for _, e := range matches[1:] {
		fill = commonPrefix(fill, e.name)
	}
	return fill, matches
}

func commonPrefix(a, b string) string {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return a[:i]
		}
	}
	return a[:n]
}

func candidateColumns(matches []completionEntry, width int) string {
	names := make([]string, len(matches))
	for i, e := range matches {
		names[i] = e.name
		if e.isDir {
			names[i] += "/"
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return ""
	}
	if width <= 0 {
		width = 80
	}

	maxLen := 0
	for _, name := range names {
		maxLen = max(maxLen, len(name))
	}
	const gap = 2
	colWidth := maxLen + gap
	ncol := max(1, width/colWidth)
	if ncol > len(names) {
		ncol = len(names)
	}
	nrow := (len(names) + ncol - 1) / ncol

	var rows []string
	for r := 0; r < nrow; r++ {
		var parts []string
		for c := 0; c < ncol; c++ {
			idx := r*ncol + c
			if idx >= len(names) {
				break
			}
			parts = append(parts, names[idx]+strings.Repeat(" ", maxLen-len(names[idx])))
		}
		rows = append(rows, strings.Join(parts, strings.Repeat(" ", gap)))
	}
	return strings.Join(rows, "\n")
}

func lsQuery(key completionKey) string {
	parts := []string{"ls"}
	if key.local {
		parts = append(parts, "-l")
	}
	if key.dir != "" {
		parts = append(parts, key.dir)
	}
	return strings.Join(parts, " ")
}

// completeInput routes Tab completion through the command registry: argument
// position 0 uses the vocabulary probe; each declared ArgSpec handles its
// argument kind.
func (m *Model) completeInput(allowProbe bool) tea.Cmd {
	line := m.input.Value()
	cursor := m.input.Position()
	if cursor > len(line) {
		cursor = len(line)
	}
	t, ok := parseInputTarget(line, cursor)
	if !ok {
		return nil
	}
	if t.argPos == 0 {
		return m.completeVocab(t, allowProbe)
	}

	arg, ok := t.argSpec(m.commands)
	if !ok {
		return nil
	}
	switch arg.Kind {
	case ArgCategories:
		return m.completeEntries(t, m.categories, allowProbe)
	case ArgSubcommand:
		return m.completeSubcommand(t, arg.Section, allowProbe)
	case ArgIndexedList:
		return m.completeIndexedList(t, arg, allowProbe)
	case ArgChoices:
		m.completeAgainst(t.prefix, t.partial, t.wordSuffix, t.suffix, toEntries(arg.Choices), "", PathDefault)
		return nil
	case ArgPath:
		return m.completePath(t, arg.Path, allowProbe)
	}
	return nil
}

func commandEntries(commands map[string]CommandSpec) []completionEntry {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return toEntries(names)
}

func (m *Model) completeVocab(t inputTarget, allowProbe bool) tea.Cmd {
	if m.vocab == nil {
		if cmd := m.startVocabProbe(allowProbe); cmd != nil {
			return cmd
		}
		m.completeAgainst("", t.partial, t.wordSuffix, t.suffix, commandEntries(m.commands), " ", PathDefault)
		return nil
	}
	m.completeAgainst("", t.partial, t.wordSuffix, t.suffix, m.vocab, " ", PathDefault)
	return nil
}

func (m *Model) completeIndexedList(t inputTarget, arg ArgSpec, allowProbe bool) tea.Cmd {
	entries, ok := m.indexedLists[arg.Probe]
	if !ok {
		if !allowProbe || m.client == nil {
			return nil
		}
		m.indexedListSeq++
		return probeIndexedList(m.client, arg.Probe, arg.Section, m.indexedListSeq)
	}
	// Empty partial: fill the lowest index (e.g. "software uninstall " → "… 0").
	if t.partial == "" && len(entries) > 0 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.name
		}
		sort.Strings(names)
		m.input.SetValue(t.prefix + names[0] + t.suffix)
		m.input.CursorEnd()
		return nil
	}
	m.completeAgainst(t.prefix, t.partial, t.wordSuffix, t.suffix, entries, "", PathDefault)
	return nil
}

func (m *Model) completeSubcommand(t inputTarget, section string, allowProbe bool) tea.Cmd {
	entries, ok := m.subcommands[t.cmd]
	if !ok {
		if !allowProbe || m.client == nil {
			return nil
		}
		m.subcommandSeq++
		return probeSubcommand(m.client, t.cmd, section, m.subcommandSeq)
	}
	m.completeAgainst(t.prefix, t.partial, t.wordSuffix, t.suffix, entries, "", PathDefault)
	return nil
}

func (m *Model) completeEntries(t inputTarget, entries []completionEntry, allowProbe bool) tea.Cmd {
	if entries == nil {
		return m.startVocabProbe(allowProbe)
	}
	m.completeAgainst(t.prefix, t.partial, t.wordSuffix, t.suffix, entries, "", PathDefault)
	return nil
}

func (m *Model) completePath(t inputTarget, style PathStyle, allowProbe bool) tea.Cmd {
	pt, ok := pathTargetFrom(t, style)
	if !ok {
		return nil
	}
	entries, cached := m.completions[pt.cacheKey()]
	if !cached {
		if !allowProbe || m.client == nil {
			return nil
		}
		m.probeSeq++
		return probeListing(m.client, pt.cacheKey(), m.probeSeq)
	}
	m.completeAgainst(pt.tokenPrefix(), pt.partial, t.wordSuffix, t.suffix, entries, "", style)
	return nil
}

func (m *Model) completeAgainst(prefix, partial, wordSuffix, suffix string, entries []completionEntry, uniqueSuffix string, style PathStyle) {
	fill, matches := completeToken(entries, partial, style)
	switch {
	case len(matches) == 0:
	case len(matches) == 1:
		m.input.SetValue(prefix + fill + uniqueSuffix + suffix)
		m.input.CursorEnd()
	case fill != partial:
		m.input.SetValue(prefix + fill + wordSuffix + suffix)
		m.input.CursorEnd()
	default:
		display := matches
		if style == PathRm {
			display = filesOnly(matches)
		}
		m.log(dim.Render(candidateColumns(display, m.width)))
	}
}

func filesOnly(entries []completionEntry) []completionEntry {
	out := make([]completionEntry, 0, len(entries))
	for _, e := range entries {
		if !e.isDir {
			out = append(out, e)
		}
	}
	return out
}

func probeIndexedList(client *intercept.Client, probe, section string, seq int) tea.Cmd {
	return func() tea.Msg {
		env, err := client.Query(probe)
		msg := indexedListResultMsg{seq: seq, probe: probe, err: err}
		if err == nil {
			msg.indices = parseIndexedList(env.Msg, section)
		}
		return msg
	}
}

func (m *Model) applyIndexedListResult(msg indexedListResultMsg) {
	if msg.seq != m.indexedListSeq || msg.err != nil {
		return
	}
	m.indexedLists[msg.probe] = toEntries(msg.indices)
	if len(msg.indices) == 0 {
		m.indexedLists[msg.probe] = nil
	}
	m.completeInput(false)
}

func probeSubcommand(client *intercept.Client, cmd, section string, seq int) tea.Cmd {
	return func() tea.Msg {
		env, err := client.Query(cmd)
		msg := subcommandResultMsg{seq: seq, cmd: cmd, err: err}
		if err == nil {
			msg.listing = env.Msg
			msg.names = parseSubcommandSection(env.Msg, section, cmd)
		}
		return msg
	}
}

func (m *Model) applySubcommandResult(msg subcommandResultMsg) {
	if msg.seq != m.subcommandSeq || msg.err != nil {
		return
	}
	m.subcommands[msg.cmd] = toEntries(msg.names)
	// Remember probed-but-empty so Tab doesn't re-probe in a loop.
	if len(msg.names) == 0 {
		m.subcommands[msg.cmd] = nil
	}
	m.completeInput(false)
}

func probeListing(client *intercept.Client, key completionKey, seq int) tea.Cmd {
	return func() tea.Msg {
		env, err := client.Query(lsQuery(key))
		msg := probeResultMsg{seq: seq, key: key, err: err}
		if err == nil {
			msg.listing = env.Msg
		}
		return msg
	}
}

func (m *Model) startVocabProbe(allowProbe bool) tea.Cmd {
	if !allowProbe || m.vocabLoading || m.client == nil {
		return nil
	}
	m.vocabSeq++
	m.vocabLoading = true
	return probeVocab(m.client, m.vocabSeq)
}

func probeVocab(client *intercept.Client, seq int) tea.Cmd {
	return func() tea.Msg {
		env, err := client.Query("cmds")
		if err != nil {
			return vocabResultMsg{seq: seq, err: err}
		}
		msg := vocabResultMsg{seq: seq, categories: parseNameList(env.Msg)}
		for _, cat := range msg.categories {
			env, err := client.Query("cmds " + cat)
			if err != nil {
				return vocabResultMsg{seq: seq, err: err}
			}
			names := parseNameList(env.Msg)
			msg.commands = append(msg.commands, names...)
			if cat == "filesystem" {
				msg.filesystem = names
			}
		}
		msg.commands = dedupeSorted(msg.commands)
		return msg
	}
}

func (m *Model) applyVocabResult(msg vocabResultMsg) {
	if msg.seq != m.vocabSeq {
		return
	}
	m.vocabLoading = false
	if msg.err != nil {
		return
	}
	m.categories = toEntries(msg.categories)
	m.vocab = toEntries(msg.commands)
	for _, name := range msg.filesystem {
		if spec, ok := specFromFilesystem(name); ok {
			m.commands[name] = spec
		}
	}
	m.completeInput(false)
}

func (m *Model) applyProbeResult(msg probeResultMsg) {
	if msg.seq != m.probeSeq || msg.err != nil {
		return
	}
	for dir, entries := range parseListing(msg.listing) {
		m.completions[completionKey{dir: dir, local: msg.key.local}] = entries
	}
	if _, ok := m.completions[msg.key]; !ok {
		m.completions[msg.key] = nil
	}
	m.completeInput(false)
}

func (m *Model) invalidateCompletions(value string) {
	fields := strings.Fields(value)
	switch {
	case len(fields) >= 2 && fields[0] == "software":
		switch fields[1] {
		case "install", "uninstall", "transfer", "destroy":
			delete(m.indexedLists, "software list")
			m.indexedListSeq++
		}
	case len(fields) >= 2 && fields[0] == "jobs" && fields[1] == "kill":
		delete(m.indexedLists, "jobs list")
		m.indexedListSeq++
	}

	if len(fields) == 0 {
		return
	}
	cmd := fields[0]
	if spec, ok := m.commands[cmd]; ok && spec.InvalidateFS {
		clear(m.completions)
		m.probeSeq++
		return
	}
	if cmd == "connect" {
		clear(m.completions)
		m.probeSeq++
	}
}
