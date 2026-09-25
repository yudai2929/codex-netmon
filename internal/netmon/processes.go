package netmon

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type processKind string

const (
	processKindCodex   processKind = "codex"
	processKindCommand processKind = "command"
)

type processInfo struct {
	PID       int
	ParentPID int
	Name      string
}

var codexProcessNames = map[string]bool{
	"codex": true, "Codex": true, "Codex (Service)": true,
	"Codex (Renderer)": true, "codex-code-mode-host": true,
}

var commandParentNames = map[string]bool{
	"codex": true, "Codex (Service)": true, "codex-code-mode-host": true,
}

func readProcesses(ctx context.Context) (map[int]processInfo, error) {
	output, err := exec.CommandContext(ctx, "ps", "-axww", "-o", "pid=,ppid=,comm=").Output()
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}
	return parseProcesses(bytes.NewReader(output))
}

func parseProcesses(r io.Reader) (map[int]processInfo, error) {
	processes := make(map[int]processInfo)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("parse process PID %q: %w", fields[0], err)
		}
		parentPID, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("parse parent PID %q: %w", fields[1], err)
		}
		processes[pid] = processInfo{
			PID: pid, ParentPID: parentPID,
			Name: filepath.Base(strings.Join(fields[2:], " ")),
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read process list: %w", err)
	}
	return processes, nil
}

func classifyProcess(pid int, processes map[int]processInfo) (processKind, bool) {
	process, found := processes[pid]
	if !found {
		return "", false
	}
	if codexProcessNames[process.Name] {
		return processKindCodex, true
	}

	seen := map[int]bool{pid: true}
	for parentPID := process.ParentPID; parentPID > 1 && !seen[parentPID]; {
		seen[parentPID] = true
		parent, found := processes[parentPID]
		if !found {
			break
		}
		if commandParentNames[parent.Name] {
			return processKindCommand, true
		}
		if codexProcessNames[parent.Name] {
			break
		}
		parentPID = parent.ParentPID
	}
	return "", false
}
