package ui

import (
	"maps"
	"strings"
)

// ArgKind describes how a single command argument is completed.
type ArgKind int

const (
	ArgCategories ArgKind = iota // cmds categories (from vocab probe)
	ArgPath                      // filesystem path (from ls probe)
	ArgSubcommand                // subcommand from probing the parent (e.g. slaves → list)
	ArgIndexedList               // numbered index from a list probe (e.g. software list)
	ArgChoices                   // fixed argument choices (e.g. pass see/reset)
)

// PathStyle controls directory traversal and filtering for path arguments.
type PathStyle int

const (
	PathDefault PathStyle = iota // dirs gain a trailing /
	PathRm                      // traverse dirs; skip empty; list files when ambiguous
	PathDirsOnly                // dirs only; no trailing slash (rmdir)
)

// ArgSpec configures completion for one argument position.
type ArgSpec struct {
	Kind    ArgKind
	Path    PathStyle // valid when Kind == ArgPath
	Section string    // section header for ArgSubcommand / ArgIndexedList
	Probe   string    // silent query for ArgIndexedList, e.g. "software list"
	Choices []string  // valid when Kind == ArgChoices
}

// CommandSpec declares how a command's arguments are completed.
// Args[0] is the first argument after the command name. Further arguments
// after a subcommand are declared in Subcommands.
type CommandSpec struct {
	Args         []ArgSpec
	Subcommands  map[string][]ArgSpec // args following a named subcommand
	InvalidateFS bool                 // submitting clears cached directory listings
}

func pathArg(style PathStyle) ArgSpec {
	return ArgSpec{Kind: ArgPath, Path: style}
}

func indexedArg(probe, section string) ArgSpec {
	return ArgSpec{Kind: ArgIndexedList, Probe: probe, Section: section}
}

func choicesArg(choices ...string) ArgSpec {
	return ArgSpec{Kind: ArgChoices, Choices: choices}
}

var softwareIndexedSubs = []string{"install", "uninstall", "transfer", "destroy"}

func softwareSpec() CommandSpec {
	subs := make(map[string][]ArgSpec, len(softwareIndexedSubs))
	arg := indexedArg("software list", "Software:")
	for _, name := range softwareIndexedSubs {
		subs[name] = []ArgSpec{arg}
	}
	return CommandSpec{
		Args:        []ArgSpec{{Kind: ArgSubcommand, Section: "Commands:"}},
		Subcommands: subs,
	}
}

func jobsSpec() CommandSpec {
	return CommandSpec{
		Args: []ArgSpec{{Kind: ArgSubcommand, Section: "Commands:"}},
		Subcommands: map[string][]ArgSpec{
			"kill": {indexedArg("jobs list", "Jobs:")},
		},
	}
}

// builtinCommands are known before the cmds vocabulary probe runs.
var builtinCommands = map[string]CommandSpec{
	"cmds":   {Args: []ArgSpec{{Kind: ArgCategories}}},
	"bits":      {Args: []ArgSpec{{Kind: ArgSubcommand, Section: "Commands:"}}},
	"hardware": {Args: []ArgSpec{{Kind: ArgSubcommand, Section: "Commands:"}}},
	"jobs":     jobsSpec(),
	"pass":     {Args: []ArgSpec{choicesArg("see", "reset")}},
	"slaves":   {Args: []ArgSpec{{Kind: ArgSubcommand, Section: "Commands:"}}},
	"software": softwareSpec(),
	"cat":    {Args: []ArgSpec{pathArg(PathDefault)}},
	"mkdir": {Args: []ArgSpec{pathArg(PathDefault)}, InvalidateFS: true},
	"rm":    {Args: []ArgSpec{pathArg(PathRm)}, InvalidateFS: true},
	"rmdir": {Args: []ArgSpec{pathArg(PathDirsOnly)}, InvalidateFS: true},
}

// vocabNoPath excludes filesystem vocabulary entries that take no path argument.
var vocabNoPath = map[string]bool{
	"ls": true,
}

func cloneCommands(src map[string]CommandSpec) map[string]CommandSpec {
	dst := make(map[string]CommandSpec, len(src))
	maps.Copy(dst, src)
	return dst
}

func specFromFilesystem(cmd string) (CommandSpec, bool) {
	if vocabNoPath[cmd] {
		return CommandSpec{}, false
	}
	style := PathDefault
	switch cmd {
	case "rm":
		style = PathRm
	case "rmdir":
		style = PathDirsOnly
	}
	return CommandSpec{Args: []ArgSpec{pathArg(style)}}, true
}

// inputTarget describes which part of the input line Tab is completing.
type inputTarget struct {
	cmd    string // set when completing an argument
	argPos int    // 0 = command name, 1+ = argument index
	prefix string // input before the word under the cursor
	token  string // full word under the cursor (used for path splitting)
	partial string // word prefix before the cursor (used for matching)
	wordSuffix string // remainder of the word after the cursor
	suffix string // input after the word under the cursor
	local  bool   // -l/--local among arguments before the cursor
}

// tokenBounds returns the whitespace-delimited word surrounding cursor.
func tokenBounds(line string, cursor int) (start, end int) {
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(line) {
		cursor = len(line)
	}
	start = cursor
	for start > 0 && line[start-1] != ' ' {
		start--
	}
	end = cursor
	for end < len(line) && line[end] != ' ' {
		end++
	}
	return start, end
}

func parseInputTarget(line string, cursor int) (inputTarget, bool) {
	if line == "" {
		return inputTarget{argPos: 0}, true
	}
	start, end := tokenBounds(line, cursor)
	if cursor > end {
		cursor = end
	}
	t := inputTarget{
		prefix:     line[:start],
		token:      line[start:end],
		partial:    line[start:cursor],
		wordSuffix: line[cursor:end],
		suffix:     line[end:],
	}
	t.cmd, _, _ = strings.Cut(line, " ")
	t.argPos = strings.Count(t.prefix, " ")
	fields := strings.Fields(t.prefix)
	for i := 1; i < len(fields); i++ {
		if fields[i] == "-l" || fields[i] == "--local" {
			t.local = true
		}
	}
	return t, true
}

func (t inputTarget) argSpec(commands map[string]CommandSpec) (ArgSpec, bool) {
	if t.argPos == 0 || strings.HasPrefix(t.token, "-") {
		return ArgSpec{}, false
	}
	spec, ok := commands[t.cmd]
	if !ok {
		return ArgSpec{}, false
	}
	// Flags between the command and its arguments don't consume ArgSpec slots.
	idx := t.argPos - flagCount(t.prefix)
	if idx < 1 {
		return ArgSpec{}, false
	}
	if idx <= len(spec.Args) {
		return spec.Args[idx-1], true
	}
	sub := t.subcommand()
	if sub == "" || spec.Subcommands == nil {
		return ArgSpec{}, false
	}
	subArgs, ok := spec.Subcommands[sub]
	if !ok {
		return ArgSpec{}, false
	}
	subIdx := idx - len(spec.Args)
	if subIdx < 1 || subIdx > len(subArgs) {
		return ArgSpec{}, false
	}
	return subArgs[subIdx-1], true
}

func (t inputTarget) subcommand() string {
	fields := strings.Fields(strings.TrimSpace(t.prefix))
	if len(fields) < 2 {
		return ""
	}
	return fields[1]
}

func flagCount(prefix string) int {
	fields := strings.Fields(prefix)
	if len(fields) <= 1 {
		return 0
	}
	n := 0
	for _, f := range fields[1:] {
		if strings.HasPrefix(f, "-") {
			n++
		}
	}
	return n
}

// pathTarget is the filesystem segment under completion for an ArgPath argument.
type pathTarget struct {
	prefix  string
	dir     string
	partial string
	local   bool
	style   PathStyle
}

func pathTargetFrom(t inputTarget, style PathStyle) (pathTarget, bool) {
	if strings.HasPrefix(t.token, "-") {
		return pathTarget{}, false
	}
	pt := pathTarget{
		prefix:  t.prefix,
		partial: t.token,
		local:   t.local,
		style:   style,
	}
	if i := strings.LastIndex(t.token, "/"); i >= 0 {
		pt.dir, pt.partial = t.token[:i], t.token[i+1:]
	}
	return pt, true
}

func (p pathTarget) cacheKey() completionKey {
	return completionKey{dir: p.dir, local: p.local}
}

func (p pathTarget) tokenPrefix() string {
	if p.dir != "" {
		return p.prefix + p.dir + "/"
	}
	return p.prefix
}
