package netmon

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type nettopSampler struct {
	timeout time.Duration
}

type networkSnapshot struct {
	Samples   map[int]processSample
	Processes map[int]processInfo
}

func (sampler nettopSampler) read(ctx context.Context) (networkSnapshot, error) {
	sampleCtx, cancel := context.WithTimeout(ctx, sampler.timeout)
	defer cancel()
	processes, err := readProcesses(sampleCtx)
	if err != nil {
		return networkSnapshot{}, err
	}
	cmd := exec.CommandContext(sampleCtx, "nettop",
		"-P", "-L", "1", "-x", "-n", "-t", "wifi", "-J", "bytes_in,bytes_out",
	)
	output, err := cmd.Output()
	if err != nil {
		return networkSnapshot{}, fmt.Errorf("run nettop: %w", err)
	}
	samples, err := parseNettop(bytes.NewReader(output))
	if err != nil {
		return networkSnapshot{}, err
	}
	selected := make(map[int]processSample)
	for pid, sample := range samples {
		kind, found := classifyProcess(pid, processes)
		if !found {
			continue
		}
		sample.Name = processes[pid].Name
		sample.Kind = kind
		selected[pid] = sample
	}
	return networkSnapshot{Samples: selected, Processes: processes}, nil
}

type processSample struct {
	Name     string
	PID      int
	Kind     processKind
	Received uint64
	Sent     uint64
}

func parseNettop(r io.Reader) (map[int]processSample, error) {
	rows, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse nettop CSV: %w", err)
	}
	samples := make(map[int]processSample)
	for _, row := range rows {
		if len(row) < 3 || row[0] == "" {
			continue
		}
		separator := strings.LastIndexByte(row[0], '.')
		if separator < 1 {
			continue
		}
		pid, err := strconv.Atoi(row[0][separator+1:])
		if err != nil {
			continue
		}
		received, err := strconv.ParseUint(row[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse received bytes for %s: %w", row[0], err)
		}
		sent, err := strconv.ParseUint(row[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse sent bytes for %s: %w", row[0], err)
		}
		samples[pid] = processSample{
			Name: row[0][:separator], PID: pid, Received: received, Sent: sent,
		}
	}
	return samples, nil
}

func sampleDelta(previous, current networkSnapshot) map[int]processSample {
	deltas := make(map[int]processSample)
	for pid, sample := range current.Samples {
		old, found := previous.Samples[pid]
		if !found {
			if _, wasRunning := previous.Processes[pid]; !wasRunning {
				deltas[pid] = sample
			}
			continue
		}
		if old.Name != sample.Name || old.Kind != sample.Kind || sample.Received < old.Received || sample.Sent < old.Sent {
			continue
		}
		deltas[pid] = processSample{
			Name: sample.Name, PID: pid, Kind: sample.Kind,
			Received: sample.Received - old.Received,
			Sent:     sample.Sent - old.Sent,
		}
	}
	return deltas
}
