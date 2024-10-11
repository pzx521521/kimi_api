package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestGetStreamData(t *testing.T) {
	sb := &strings.Builder{}
	GetStreamData(`data: {"event":"cmpl","idx_s":0,"idx_z":0,"text":"身","view":"`, sb)
	fmt.Printf("%v\n", sb.String())
}

func TestGetData(t *testing.T) {
}
