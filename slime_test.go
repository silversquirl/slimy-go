package slimy

import "testing"

func TestZ23(t *testing.T) {
	if World(1).CalcChunk(-1, 23) {
		t.Error("-1, 23 should not be a slime chunk")
	}
}
