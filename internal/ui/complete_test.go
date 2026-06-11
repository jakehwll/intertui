package ui

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Real payloads observed on the wire (¬ color codes, tab-indented children).
const (
	wireListing    = "¬blogs/¬*\n\t¬gxfer.log¬*"
	wireCategories = "Categories:\n¬gclient¬*\n¬gfilesystem¬*\n¬ginformation¬*\n¬gmisc¬*\n¬gremote¬*\n¬gsoftware¬*\n¬gsystem¬*\n¬gweb¬*"
	wireFsCmds     = "¬gcat¬*\n¬gls¬*\n¬gmkdir¬*\n¬grm¬*\n¬grmdir¬*"
)

func TestParseListing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  string
		want map[string][]completionEntry
	}{
		{
			name: "real wire payload",
			msg:  wireListing,
			want: map[string][]completionEntry{
				"":     {{name: "logs", isDir: true}},
				"logs": {{name: "xfer.log"}},
			},
		},
		{
			name: "nested dirs and root file",
			msg:  "logs/\n\txfer.log\n\tsub/\n\t\tdeep.txt\nreadme.txt",
			want: map[string][]completionEntry{
				"":         {{name: "logs", isDir: true}, {name: "readme.txt"}},
				"logs":     {{name: "xfer.log"}, {name: "sub", isDir: true}},
				"logs/sub": {{name: "deep.txt"}},
			},
		},
		{
			name: "space indentation",
			msg:  "logs/\n    xfer.log",
			want: map[string][]completionEntry{
				"":     {{name: "logs", isDir: true}},
				"logs": {{name: "xfer.log"}},
			},
		},
		{
			name: "blank lines ignored",
			msg:  "\nlogs/\n\n\txfer.log\n",
			want: map[string][]completionEntry{
				"":     {{name: "logs", isDir: true}},
				"logs": {{name: "xfer.log"}},
			},
		},
		{
			name: "empty directory marker",
			msg:  "logs/\n\txfer.log\nfoo/ (empty)",
			want: map[string][]completionEntry{
				"":     {{name: "logs", isDir: true}, {name: "foo", isDir: true, isEmpty: true}},
				"logs": {{name: "xfer.log"}},
			},
		},
		{
			name: "empty message",
			msg:  "",
			want: map[string][]completionEntry{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parseListing(tt.msg)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseListing(%q) = %#v, want %#v", tt.msg, got, tt.want)
			}
		})
	}
}

func pathTargetFromLine(line string) (pathTarget, bool) {
	t, ok := parseInputTarget(line)
	if !ok {
		return pathTarget{}, false
	}
	arg, ok := t.argSpec(builtinCommands)
	if !ok || arg.Kind != ArgPath {
		return pathTarget{}, false
	}
	return pathTargetFrom(t, arg.Path)
}

func TestParsePathTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  pathTarget
		ok    bool
	}{
		{
			name:  "root partial",
			value: "cat l",
			want:  pathTarget{prefix: "cat ", partial: "l", style: PathDefault},
			ok:    true,
		},
		{
			name:  "dir partial",
			value: "cat logs/xf",
			want:  pathTarget{prefix: "cat ", dir: "logs", partial: "xf", style: PathDefault},
			ok:    true,
		},
		{
			name:  "trailing slash lists dir",
			value: "cat logs/",
			want:  pathTarget{prefix: "cat ", dir: "logs", partial: "", style: PathDefault},
			ok:    true,
		},
		{name: "ls takes no arguments", value: "ls ", ok: false},
		{
			name:  "local flag",
			value: "cat -l lo",
			want:  pathTarget{prefix: "cat -l ", partial: "lo", local: true, style: PathDefault},
			ok:    true,
		},
		{
			name:  "long local flag with dir",
			value: "cat --local logs/x",
			want:  pathTarget{prefix: "cat --local ", dir: "logs", partial: "x", local: true, style: PathDefault},
			ok:    true,
		},
		{name: "no argument started", value: "cat", ok: false},
		{name: "unknown command", value: "echo foo", ok: false},
		{name: "flag under cursor", value: "cat -", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := pathTargetFromLine(tt.value)
			if ok != tt.ok {
				t.Fatalf("pathTargetFromLine(%q) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Fatalf("pathTargetFromLine(%q) = %#v, want %#v", tt.value, got, tt.want)
			}
		})
	}
}

func TestCandidateColumns(t *testing.T) {
	t.Parallel()

	entries := []completionEntry{
		{name: "abandon"}, {name: "antivirus"}, {name: "bits"},
		{name: "breach"}, {name: "broadcast"}, {name: "cat"},
		{name: "colors/colours"}, {name: "connect"},
	}
	got := candidateColumns(entries, 40)
	rows := strings.Split(got, "\n")
	if len(rows) < 2 {
		t.Fatalf("expected multiple rows, got %d:\n%s", len(rows), got)
	}
	for _, row := range rows {
		if len(row) > 40 {
			t.Fatalf("row wider than terminal: len=%d %q", len(row), row)
		}
	}
	if !strings.Contains(got, "colors/colours") {
		t.Fatalf("full name missing:\n%s", got)
	}
}

func TestCompleteToken(t *testing.T) {
	t.Parallel()

	entries := []completionEntry{
		{name: "logs", isDir: true},
		{name: "readme.txt"},
		{name: "report.txt"},
	}

	tests := []struct {
		name        string
		partial     string
		style       PathStyle
		wantFill    string
		wantMatches int
	}{
		{name: "unique dir gains slash", partial: "l", style: PathDefault, wantFill: "logs/", wantMatches: 1},
		{name: "unique file", partial: "readm", style: PathDefault, wantFill: "readme.txt", wantMatches: 1},
		{name: "common prefix fills", partial: "r", style: PathDefault, wantFill: "re", wantMatches: 2},
		{name: "ambiguous no progress", partial: "re", style: PathDefault, wantFill: "re", wantMatches: 2},
		{name: "no match", partial: "z", style: PathDefault, wantFill: "", wantMatches: 0},
		{name: "empty partial matches all", partial: "", style: PathDefault, wantFill: "", wantMatches: 3},
		{name: "rm traverses into directories", partial: "l", style: PathRm, wantFill: "logs/", wantMatches: 1},
		{name: "rm skips empty directories", partial: "f", style: PathRm, wantFill: "", wantMatches: 0},
		{name: "rmdir dir without slash", partial: "l", style: PathDirsOnly, wantFill: "logs", wantMatches: 1},
		{name: "rmdir skips files", partial: "read", style: PathDirsOnly, wantFill: "", wantMatches: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fill, matches := completeToken(entries, tt.partial, tt.style)
			if fill != tt.wantFill {
				t.Fatalf("fill = %q, want %q", fill, tt.wantFill)
			}
			if len(matches) != tt.wantMatches {
				t.Fatalf("matches = %d, want %d", len(matches), tt.wantMatches)
			}
		})
	}
}

func TestLsQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key  completionKey
		want string
	}{
		{completionKey{}, "ls"},
		{completionKey{dir: "logs"}, "ls logs"},
		{completionKey{local: true}, "ls -l"},
		{completionKey{dir: "logs", local: true}, "ls -l logs"},
	}

	for _, tt := range tests {
		if got := lsQuery(tt.key); got != tt.want {
			t.Fatalf("lsQuery(%#v) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

// TestTabCompletionFlow exercises the user-facing scenario: probe result
// arrives, then two Tabs walk "cat l" to "cat logs/xfer.log".
func TestRmRmdirCompletion(t *testing.T) {
	t.Parallel()

	listing := "logs/\n\txfer.log\nfoo/ (empty)"
	m := connectedModel(t)
	m.completions[completionKey{}] = parseListing(listing)[""]

	t.Run("rm traverses into non-empty dir", func(t *testing.T) {
		m.input.SetValue("rm lo")
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if got := m.input.Value(); got != "rm logs/" {
			t.Fatalf("input = %q, want %q", got, "rm logs/")
		}
	})

	t.Run("rm skips empty dir", func(t *testing.T) {
		m.input.SetValue("rm fo")
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if got := m.input.Value(); got != "rm fo" {
			t.Fatalf("input = %q, want unchanged", got)
		}
	})

	t.Run("rmdir completes empty dir without slash", func(t *testing.T) {
		m.input.SetValue("rmdir fo")
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if got := m.input.Value(); got != "rmdir foo" {
			t.Fatalf("input = %q, want %q", got, "rmdir foo")
		}
	})

	t.Run("rmdir skips files", func(t *testing.T) {
		m.input.SetValue("rmdir x")
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if got := m.input.Value(); got != "rmdir x" {
			t.Fatalf("input = %q, want unchanged", got)
		}
	})
}

func TestTabCompletionFlow(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.input.SetValue("cat l")
	m.probeSeq = 1

	// The probe reply both populates the cache and applies the completion.
	updated, _ := m.Update(probeResultMsg{seq: 1, key: completionKey{}, listing: wireListing})
	m = updated.(Model)
	if got := m.input.Value(); got != "cat logs/" {
		t.Fatalf("input after probe result = %q, want %q", got, "cat logs/")
	}

	// Second Tab descends into logs/ and fills its sole entry from cache.
	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "cat logs/xfer.log" {
		t.Fatalf("input after tab = %q, want %q", got, "cat logs/xfer.log")
	}
}

func TestTabIgnoredWithoutCacheOrClient(t *testing.T) {
	t.Parallel()

	// No client and no cache: Tab must neither probe nor alter the input.
	m := connectedModel(t)
	m.input.SetValue("cat l")
	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "cat l" {
		t.Fatalf("input = %q, want unchanged %q", got, "cat l")
	}
}

func TestStaleProbeResultDropped(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.input.SetValue("cat l")
	m.probeSeq = 2

	updated, _ := m.Update(probeResultMsg{seq: 1, key: completionKey{}, listing: wireListing})
	m = updated.(Model)
	if got := m.input.Value(); got != "cat l" {
		t.Fatalf("input = %q, want unchanged %q", got, "cat l")
	}
	if len(m.completions) != 0 {
		t.Fatalf("stale probe populated cache: %#v", m.completions)
	}
}

func TestAmbiguousCompletionListsCandidates(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.completions[completionKey{}] = []completionEntry{
		{name: "readme.txt"},
		{name: "reports", isDir: true},
	}
	m.input.SetValue("cat re")

	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "cat re" {
		t.Fatalf("input = %q, want unchanged %q", got, "cat re")
	}
	if !hasMessage(m.messages, "readme.txt") || !hasMessage(m.messages, "reports/") {
		t.Fatalf("candidates not logged: %#v", m.messages)
	}
}

const (
	wireSlavesHelp   = "Usage: slaves [command]\n\n0 slaves active\nCommands:\n list"
	wireSoftwareHelp = "Usage: software [command]\nCommands:\n list\n install\n uninstall\n transfer\n destroy"
	wireSoftwareList = "Software:\n0: atom_probe (probe)\n1: void_pw (getpw)\n2: zd_infiltr8tr (breach)"
	wireBitsHelp      = "Commands:\nbits balance\nbits transfer [to] [amount]\n  -a, --anonymous: Transfer anonymously."
	wireHardwareHelp = "ATOM(NET)* HARDWARE MANAGER\n\nCPU: ATOM Gamma B1000 (Level 1)\nRAM: 2GB\n\nUsage: hardware [command]\nCommands:\n  upgrade_cpu\n  upgrade_ram\n  upgrade_ports"
	wireJobsHelp     = "Active jobs: 0\nMemory usage: 0GB/2GB\n\nUsage: jobs [command]\nCommands:\n  list\n  kill [id]"
	wireJobsList     = "Jobs:\n0: scan 1.2.3.4\n1: breach target"
)

func TestParseSubcommandSection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, msg, cmd string
		want          []string
	}{
		{
			name: "bare names",
			msg:  wireSlavesHelp,
			cmd:  "slaves",
			want: []string{"list"},
		},
		{
			name: "prefixed names with flags",
			msg:  wireBitsHelp,
			cmd:  "bits",
			want: []string{"balance", "transfer"},
		},
		{
			name: "software style",
			msg:  wireSoftwareHelp,
			cmd:  "software",
			want: []string{"list", "install", "uninstall", "transfer", "destroy"},
		},
		{
			name: "preamble before commands",
			msg:  wireHardwareHelp,
			cmd:  "hardware",
			want: []string{"upgrade_cpu", "upgrade_ram", "upgrade_ports"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parseSubcommandSection(tt.msg, "Commands:", tt.cmd)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseSubcommandSection() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestPassCompletion(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.input.SetValue("pass r")
	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "pass reset" {
		t.Fatalf("input = %q, want %q", got, "pass reset")
	}
}

func TestJobsCompletion(t *testing.T) {
	t.Parallel()

	got := parseSubcommandSection(wireJobsHelp, "Commands:", "jobs")
	if !reflect.DeepEqual(got, []string{"list", "kill"}) {
		t.Fatalf("subcommands = %#v, want [list kill]", got)
	}

	m := connectedModel(t)
	m.subcommands["jobs"] = toEntries(got)
	m.indexedLists["jobs list"] = toEntries(parseIndexedList(wireJobsList, "Jobs:"))

	m.input.SetValue("jobs k")
	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "jobs kill" {
		t.Fatalf("subcommand input = %q, want %q", got, "jobs kill")
	}

	m.input.SetValue("jobs kill ")
	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "jobs kill 0" {
		t.Fatalf("index input = %q, want %q", got, "jobs kill 0")
	}
}

func TestHardwareSubcommandCompletion(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.subcommands["hardware"] = toEntries(parseSubcommandSection(wireHardwareHelp, "Commands:", "hardware"))
	m.input.SetValue("hardware upgrade_r")
	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "hardware upgrade_ram" {
		t.Fatalf("input = %q, want %q", got, "hardware upgrade_ram")
	}
}

func TestBitsSubcommandCompletion(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.subcommands["bits"] = toEntries([]string{"balance", "transfer"})
	m.input.SetValue("bits tran")
	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "bits transfer" {
		t.Fatalf("input = %q, want %q", got, "bits transfer")
	}
}

func TestParseIndexedList(t *testing.T) {
	t.Parallel()

	got := parseIndexedList(wireSoftwareList, "Software:")
	if !reflect.DeepEqual(got, []string{"0", "1", "2"}) {
		t.Fatalf("parseIndexedList() = %#v, want [0 1 2]", got)
	}
}

func TestSoftwareCompletion(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.subcommands["software"] = toEntries(parseSubcommandSection(wireSoftwareHelp, "Commands:", "software"))
	m.indexedLists["software list"] = toEntries([]string{"0", "1", "2"})

	t.Run("subcommand", func(t *testing.T) {
		m.input.SetValue("software unin")
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if got := m.input.Value(); got != "software uninstall" {
			t.Fatalf("input = %q, want %q", got, "software uninstall")
		}
	})

	t.Run("index", func(t *testing.T) {
		m.input.SetValue("software uninstall ")
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if got := m.input.Value(); got != "software uninstall 0" {
			t.Fatalf("input = %q, want %q", got, "software uninstall 0")
		}
	})

	t.Run("partial index", func(t *testing.T) {
		m.input.SetValue("software install 1")
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if got := m.input.Value(); got != "software install 1" {
			t.Fatalf("input = %q, want unchanged %q", got, "software install 1")
		}
	})
}

func TestSlavesSubcommandCompletion(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.subcommandSeq = 1
	updated, _ := m.Update(subcommandResultMsg{
		seq:   1,
		cmd:   "slaves",
		names: []string{"list"},
	})
	m = updated.(Model)
	m.input.SetValue("slaves l")
	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "slaves list" {
		t.Fatalf("input = %q, want %q", got, "slaves list")
	}
}

func TestParseNameList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  string
		want []string
	}{
		{
			name: "categories with header",
			msg:  wireCategories,
			want: []string{"client", "filesystem", "information", "misc", "remote", "software", "system", "web"},
		},
		{
			name: "plain command list",
			msg:  wireFsCmds,
			want: []string{"cat", "ls", "mkdir", "rm", "rmdir"},
		},
		{name: "empty message", msg: "", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := parseNameList(tt.msg); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseNameList(%q) = %#v, want %#v", tt.msg, got, tt.want)
			}
		})
	}
}

func vocabModel(t *testing.T) Model {
	t.Helper()

	m := connectedModel(t)
	m.vocabSeq = 1
	updated, _ := m.Update(vocabResultMsg{
		seq:        1,
		categories: []string{"client", "filesystem"},
		commands:   []string{"cat", "clear", "ls", "mkdir", "rm", "rmdir", "vol"},
		filesystem: []string{"cat", "ls", "mkdir", "rm", "rmdir"},
	})
	return updated.(Model)
}

func TestEmptyInputTabListsCommands(t *testing.T) {
	t.Parallel()

	t.Run("loaded vocab", func(t *testing.T) {
		t.Parallel()

		m := vocabModel(t)
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if !hasMessage(m.messages, "cat") || !hasMessage(m.messages, "mkdir") {
			t.Fatalf("commands not listed on tab from empty input: %#v", m.messages)
		}
	})

	t.Run("builtin fallback while probe in flight", func(t *testing.T) {
		t.Parallel()

		m := connectedModel(t)
		m.vocabLoading = true
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if !hasMessage(m.messages, "cat") || !hasMessage(m.messages, "cmds") {
			t.Fatalf("builtin commands not listed: %#v", m.messages)
		}
	})
}

func TestCommandCompletion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		wantInput     string
		wantInLog     string
		wantAlsoInLog string
	}{
		{name: "unique command gains space", input: "mkd", wantInput: "mkdir "},
		{name: "common prefix fills", input: "c", wantInput: "c", wantInLog: "cat", wantAlsoInLog: "clear"},
		{name: "ambiguous lists candidates", input: "rm", wantInput: "rm", wantInLog: "rm", wantAlsoInLog: "rmdir"},
		{name: "no match unchanged", input: "zz", wantInput: "zz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := vocabModel(t)
			m.input.SetValue(tt.input)
			pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})

			if got := m.input.Value(); got != tt.wantInput {
				t.Fatalf("input = %q, want %q", got, tt.wantInput)
			}
			if tt.wantInLog != "" && !hasMessage(m.messages, tt.wantInLog) {
				t.Fatalf("candidates not logged, want %q in %#v", tt.wantInLog, m.messages)
			}
			if tt.wantAlsoInLog != "" && !hasMessage(m.messages, tt.wantAlsoInLog) {
				t.Fatalf("candidates not logged, want %q in %#v", tt.wantAlsoInLog, m.messages)
			}
		})
	}
}

func TestCategoryCompletion(t *testing.T) {
	t.Parallel()

	m := vocabModel(t)
	m.input.SetValue("cmds fi")
	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "cmds filesystem" {
		t.Fatalf("input = %q, want %q", got, "cmds filesystem")
	}
}

func TestVocabExtendsCommandRegistry(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.vocabSeq = 1
	updated, _ := m.Update(vocabResultMsg{
		seq:        1,
		categories: []string{"filesystem"},
		commands:   []string{"shred"},
		filesystem: []string{"shred"},
	})
	m = updated.(Model)

	if spec, ok := m.commands["shred"]; !ok || spec.Args[0].Kind != ArgPath {
		t.Fatal("filesystem category command not registered for path completion")
	}
	if spec, ok := m.commands["cat"]; !ok || spec.Args[0].Kind != ArgPath {
		t.Fatal("builtin path command lost after vocab merge")
	}
}

func TestStaleVocabResultDropped(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.vocabSeq = 2
	m.vocabLoading = true
	updated, _ := m.Update(vocabResultMsg{seq: 1, commands: []string{"cat"}})
	m = updated.(Model)

	if m.vocab != nil {
		t.Fatalf("stale vocab applied: %#v", m.vocab)
	}
	if !m.vocabLoading {
		t.Fatal("stale result must not clear the in-flight flag")
	}
}

func TestVocabProbeNotDuplicated(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.vocabLoading = true
	m.input.SetValue("mkd")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("tab during in-flight vocab probe must not start another")
	}
	// Builtin registry still completes while the remote vocab probe is in flight.
	if got := m.input.Value(); got != "mkdir " {
		t.Fatalf("input = %q, want %q", got, "mkdir ")
	}
}

func TestSubmitInvalidatesCompletions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		command   string
		wantEmpty bool
	}{
		{name: "rm clears cache", command: "rm logs/xfer.log", wantEmpty: true},
		{name: "connect clears cache", command: "connect 1.2.3.4", wantEmpty: true},
		{name: "cat keeps cache", command: "cat logs/xfer.log", wantEmpty: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := connectedModel(t)
			m.completions[completionKey{}] = []completionEntry{{name: "logs", isDir: true}}
			submit(t, &m, tt.command)

			if empty := len(m.completions) == 0; empty != tt.wantEmpty {
				t.Fatalf("cache empty = %v, want %v after %q", empty, tt.wantEmpty, tt.command)
			}
		})
	}
}
