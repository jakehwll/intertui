package ui

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Real payloads observed on the wire (¬ color codes, tab-indented children).
const (
	wireListing    = "¬blogs/¬*\n\t¬gxfer.log¬*"
	wireCategories = "Categories:\n¬gclient¬*\n¬gfilesystem¬*\n¬ginformation¬*\n¬gmisc¬*\n¬gremote¬*\n¬gsoftware¬*\n¬gsystem¬*\n¬gweb¬*"
	wireFsCmds     = "¬gcat¬*\n¬gls¬*\n¬gmkdir¬*\n¬grm¬*\n¬grmdir¬*"
)

func TestDedupeSorted(t *testing.T) {
	t.Parallel()

	got := dedupeSorted([]string{"rm", "cat", "ls", "cat", "rm", "mkdir"})
	want := []string{"cat", "ls", "mkdir", "rm"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dedupeSorted = %#v, want %#v", got, want)
	}
}

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

func setInput(t *testing.T, m *Model, value string) {
	t.Helper()
	m.input.SetValue(value)
	m.input.CursorEnd()
}

func pathTargetFromLine(line string) (pathTarget, bool) {
	t, ok := parseInputTarget(line, len(line))
	if !ok {
		return pathTarget{}, false
	}
	arg, ok := t.argSpec(builtinCommands)
	if !ok || arg.Kind != ArgPath {
		return pathTarget{}, false
	}
	return pathTargetFrom(t, arg.Path)
}

func TestParseInputTargetCursor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		line   string
		cursor int
		want   inputTarget
	}{
		{
			name:   "cursor at end matches last token",
			line:   "software install 1",
			cursor: len("software install 1"),
			want: inputTarget{
				cmd: "software", argPos: 2,
				prefix: "software install ", token: "1", partial: "1",
			},
		},
		{
			name:   "cursor on earlier word",
			line:   "software install 1",
			cursor: len("software inst"),
			want: inputTarget{
				cmd: "software", argPos: 1,
				prefix: "software ", token: "install", partial: "inst",
				wordSuffix: "all", suffix: " 1",
			},
		},
		{
			name:   "cursor on command name",
			line:   "software install",
			cursor: len("soft"),
			want: inputTarget{
				cmd: "software", argPos: 0,
				token: "software", partial: "soft", wordSuffix: "ware",
				suffix: " install",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := parseInputTarget(tt.line, tt.cursor)
			if !ok {
				t.Fatal("parseInputTarget returned false")
			}
			if got != tt.want {
				t.Fatalf("parseInputTarget(%q, %d) = %#v, want %#v", tt.line, tt.cursor, got, tt.want)
			}
		})
	}
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

func TestCommonPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		a, b, want string
	}{
		{"cat", "clear", "c"},
		{"文件", "文本文", "文"},
		{"café", "caffeine", "caf"},
	}
	for _, tt := range tests {
		if got := commonPrefix(tt.a, tt.b); got != tt.want {
			t.Fatalf("commonPrefix(%q, %q) = %q, want %q", tt.a, tt.b, got, tt.want)
		}
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
		if ansi.StringWidth(row) > 40 {
			t.Fatalf("row wider than terminal: width=%d %q", ansi.StringWidth(row), row)
		}
	}
	if !strings.Contains(got, "colors/colours") {
		t.Fatalf("full name missing:\n%s", got)
	}
}

func TestCandidateColumnsWideChars(t *testing.T) {
	t.Parallel()

	// Sorted: "a" then "文件" (width 4). Narrow name is padded to match.
	got := candidateColumns([]completionEntry{{name: "文件"}, {name: "a"}}, 40)
	row := strings.Split(got, "\n")[0]
	want := "a     文件" // "a"+3 pad spaces (width 4), gap 2, "文件" (width 4)
	if row != want {
		t.Fatalf("row = %q, want %q (display width %d)", row, want, ansi.StringWidth(want))
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
	m.completion.completions[completionKey{}] = parseListing(listing)[""]

	t.Run("rm traverses into non-empty dir", func(t *testing.T) {
		setInput(t, &m, "rm lo")
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if got := m.input.Value(); got != "rm logs/" {
			t.Fatalf("input = %q, want %q", got, "rm logs/")
		}
	})

	t.Run("rm skips empty dir", func(t *testing.T) {
		setInput(t, &m, "rm fo")
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if got := m.input.Value(); got != "rm fo" {
			t.Fatalf("input = %q, want unchanged", got)
		}
	})

	t.Run("rmdir completes empty dir without slash", func(t *testing.T) {
		setInput(t, &m, "rmdir fo")
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if got := m.input.Value(); got != "rmdir foo" {
			t.Fatalf("input = %q, want %q", got, "rmdir foo")
		}
	})

	t.Run("rmdir skips files", func(t *testing.T) {
		setInput(t, &m, "rmdir x")
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if got := m.input.Value(); got != "rmdir x" {
			t.Fatalf("input = %q, want unchanged", got)
		}
	})
}

func TestTabCompletionFlow(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	setInput(t, &m, "cat l")
	m.completion.probeSeq = 1

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
	setInput(t, &m, "cat l")
	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "cat l" {
		t.Fatalf("input = %q, want unchanged %q", got, "cat l")
	}
}

func TestProbeErrorLogged(t *testing.T) {
	t.Parallel()

	err := errTest("timed out waiting for ls response")

	t.Run("listing", func(t *testing.T) {
		t.Parallel()

		m := connectedModel(t)
		m.completion.probeSeq = 1
		updated, _ := m.Update(probeResultMsg{seq: 1, key: completionKey{}, err: err})
		m = updated.(Model)
		if !hasMessage(m.messages, "tab completion") {
			t.Fatalf("probe error not logged: %#v", m.messages)
		}
	})

	t.Run("stale listing error is silent", func(t *testing.T) {
		t.Parallel()

		m := connectedModel(t)
		m.completion.probeSeq = 2
		updated, _ := m.Update(probeResultMsg{seq: 1, key: completionKey{}, err: err})
		m = updated.(Model)
		if hasMessage(m.messages, "tab completion") {
			t.Fatalf("stale probe error logged: %#v", m.messages)
		}
	})

	t.Run("vocab", func(t *testing.T) {
		t.Parallel()

		m := connectedModel(t)
		m.completion.vocabSeq = 1
		m.completion.vocabLoading = true
		updated, _ := m.Update(vocabResultMsg{seq: 1, err: err})
		m = updated.(Model)
		if m.completion.vocabLoading {
			t.Fatal("vocabLoading not cleared after error")
		}
		if !hasMessage(m.messages, "tab completion") {
			t.Fatalf("vocab probe error not logged: %#v", m.messages)
		}
	})
}

func TestStaleProbeResultDropped(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	setInput(t, &m, "cat l")
	m.completion.probeSeq = 2

	updated, _ := m.Update(probeResultMsg{seq: 1, key: completionKey{}, listing: wireListing})
	m = updated.(Model)
	if got := m.input.Value(); got != "cat l" {
		t.Fatalf("input = %q, want unchanged %q", got, "cat l")
	}
	if len(m.completion.completions) != 0 {
		t.Fatalf("stale probe populated cache: %#v", m.completion.completions)
	}
}

func TestAmbiguousCompletionListsCandidates(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.completion.completions[completionKey{}] = []completionEntry{
		{name: "readme.txt"},
		{name: "reports", isDir: true},
	}
	setInput(t, &m, "cat re")

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
	setInput(t, &m, "pass r")
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
	m.completion.subcommands["jobs"] = toEntries(got)
	m.completion.indexedLists["jobs list"] = toEntries(parseIndexedList(wireJobsList, "Jobs:"))

	setInput(t, &m, "jobs k")
	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "jobs kill" {
		t.Fatalf("subcommand input = %q, want %q", got, "jobs kill")
	}

	setInput(t, &m, "jobs kill ")
	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "jobs kill 0" {
		t.Fatalf("index input = %q, want %q", got, "jobs kill 0")
	}
}

func TestHardwareSubcommandCompletion(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.completion.subcommands["hardware"] = toEntries(parseSubcommandSection(wireHardwareHelp, "Commands:", "hardware"))
	setInput(t, &m, "hardware upgrade_r")
	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "hardware upgrade_ram" {
		t.Fatalf("input = %q, want %q", got, "hardware upgrade_ram")
	}
}

func TestBitsSubcommandCompletion(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.completion.subcommands["bits"] = toEntries([]string{"balance", "transfer"})
	setInput(t, &m, "bits tran")
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
	m.completion.subcommands["software"] = toEntries(parseSubcommandSection(wireSoftwareHelp, "Commands:", "software"))
	m.completion.indexedLists["software list"] = toEntries([]string{"0", "1", "2"})

	t.Run("subcommand", func(t *testing.T) {
		setInput(t, &m, "software unin")
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if got := m.input.Value(); got != "software uninstall" {
			t.Fatalf("input = %q, want %q", got, "software uninstall")
		}
	})

	t.Run("index", func(t *testing.T) {
		setInput(t, &m, "software uninstall ")
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if got := m.input.Value(); got != "software uninstall 0" {
			t.Fatalf("input = %q, want %q", got, "software uninstall 0")
		}
	})

	t.Run("partial index", func(t *testing.T) {
		setInput(t, &m, "software install 1")
		pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
		if got := m.input.Value(); got != "software install 1" {
			t.Fatalf("input = %q, want unchanged %q", got, "software install 1")
		}
	})
}

func TestTabCompletesWordAtCursor(t *testing.T) {
	t.Parallel()

	m := vocabModel(t)
	m.completion.subcommands["software"] = toEntries(parseSubcommandSection(wireSoftwareHelp, "Commands:", "software"))
	setInput(t, &m, "software install 1")
	m.input.SetCursor(len("software inst"))
	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "software install 1" {
		t.Fatalf("input = %q, want %q", got, "software install 1")
	}
}

func TestSlavesSubcommandCompletion(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.completion.subcommandSeq = 1
	updated, _ := m.Update(subcommandResultMsg{
		seq:   1,
		cmd:   "slaves",
		names: []string{"list"},
	})
	m = updated.(Model)
	setInput(t, &m, "slaves l")
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
	m.completion.vocabSeq = 1
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
		m.completion.vocabLoading = true
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
			setInput(t, &m, tt.input)
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
	setInput(t, &m, "cmds fi")
	pressKey(t, &m, tea.KeyPressMsg{Code: tea.KeyTab})
	if got := m.input.Value(); got != "cmds filesystem" {
		t.Fatalf("input = %q, want %q", got, "cmds filesystem")
	}
}

func TestVocabExtendsCommandRegistry(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.completion.vocabSeq = 1
	updated, _ := m.Update(vocabResultMsg{
		seq:        1,
		categories: []string{"filesystem"},
		commands:   []string{"shred"},
		filesystem: []string{"shred"},
	})
	m = updated.(Model)

	if spec, ok := m.completion.commands["shred"]; !ok || spec.Args[0].Kind != ArgPath {
		t.Fatal("filesystem category command not registered for path completion")
	}
	if spec, ok := m.completion.commands["cat"]; !ok || spec.Args[0].Kind != ArgPath {
		t.Fatal("builtin path command lost after vocab merge")
	}
}

func TestStaleVocabResultDropped(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.completion.vocabSeq = 2
	m.completion.vocabLoading = true
	updated, _ := m.Update(vocabResultMsg{seq: 1, commands: []string{"cat"}})
	m = updated.(Model)

	if m.completion.vocab != nil {
		t.Fatalf("stale vocab applied: %#v", m.completion.vocab)
	}
	if !m.completion.vocabLoading {
		t.Fatal("stale result must not clear the in-flight flag")
	}
}

func TestVocabProbeNotDuplicated(t *testing.T) {
	t.Parallel()

	m := connectedModel(t)
	m.completion.vocabLoading = true
	setInput(t, &m, "mkd")

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
			m.completion.completions[completionKey{}] = []completionEntry{{name: "logs", isDir: true}}
			submit(t, &m, tt.command)

			if empty := len(m.completion.completions) == 0; empty != tt.wantEmpty {
				t.Fatalf("cache empty = %v, want %v after %q", empty, tt.wantEmpty, tt.command)
			}
		})
	}
}
