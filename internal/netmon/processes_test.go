package netmon

import (
	"strings"
	"testing"
)

func TestClassifyProcessTree(t *testing.T) {
	processes, err := parseProcesses(strings.NewReader(`100 1 /Applications/ChatGPT.app/Contents/MacOS/Codex
101 100 /Applications/ChatGPT.app/Contents/MacOS/Codex (Service)
102 100 /Applications/ChatGPT.app/Contents/Resources/codex
103 102 /bin/zsh
104 103 /Users/example/.local/bin/mise
105 101 /usr/local/bin/npm
106 100 /Applications/ChatGPT.app/Contents/Helpers/SkyComputerUseService
107 1 /Applications/Safari.app/Contents/MacOS/Safari
108 107 /usr/bin/curl
109 100 /Applications/ChatGPT.app/Contents/MacOS/Codex (Renderer)
110 109 /usr/bin/curl
`))
	if err != nil {
		t.Fatal(err)
	}
	for pid, want := range map[int]processKind{
		100: processKindCodex, 101: processKindCodex, 102: processKindCodex,
		104: processKindCommand, 105: processKindCommand, 109: processKindCodex,
	} {
		got, included := classifyProcess(pid, processes)
		if !included || got != want {
			t.Errorf("pid %d: got %q, included %v; want %q", pid, got, included, want)
		}
	}
	for _, pid := range []int{106, 107, 108, 110} {
		if got, included := classifyProcess(pid, processes); included {
			t.Errorf("pid %d: unexpectedly included as %q", pid, got)
		}
	}
}
