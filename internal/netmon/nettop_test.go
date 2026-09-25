package netmon

import (
	"strings"
	"testing"
)

func TestParseNettopAndDelta(t *testing.T) {
	first, err := parseNettop(strings.NewReader(",bytes_in,bytes_out,\ncodex.123,100,80,\nCodex (Service).456,200,40,\n"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := parseNettop(strings.NewReader(",bytes_in,bytes_out,\ncodex.123,150,90,\nCodex (Service).456,210,65,\ncodex.789,300,100,\n"))
	if err != nil {
		t.Fatal(err)
	}
	delta := sampleDelta(
		networkSnapshot{Samples: first, Processes: map[int]processInfo{123: {}, 456: {}, 789: {}}},
		networkSnapshot{Samples: second},
	)
	if delta[123].Received != 50 || delta[123].Sent != 10 {
		t.Fatalf("codex delta = %+v", delta[123])
	}
	if delta[456].Received != 10 || delta[456].Sent != 25 {
		t.Fatalf("service delta = %+v", delta[456])
	}
	if _, ok := delta[789]; ok {
		t.Fatal("a process that was already running must establish a baseline")
	}
}

func TestNewProcessCountsFromStart(t *testing.T) {
	current := networkSnapshot{Samples: map[int]processSample{
		789: {Name: "mise", PID: 789, Kind: processKindCommand, Received: 300, Sent: 100},
	}}
	delta := sampleDelta(networkSnapshot{Processes: map[int]processInfo{}}, current)
	if delta[789].Received != 300 || delta[789].Sent != 100 {
		t.Fatalf("new process delta = %+v", delta[789])
	}
}
