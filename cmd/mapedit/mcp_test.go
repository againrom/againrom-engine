package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func TestMCPFlagValidationIsHeadless(t *testing.T) {
	for _, args := range [][]string{{"-mcp", "bad"}, {"-mcp", "stdio", "-check"}, {"-mcp", "stdio", "-snapshot", "out.png"}, {"-mcp-pace", "-1s"}, {"-mcp-pace", "2m"}} {
		if err := run(args, io.Discard); err == nil {
			t.Fatal("invalid flags accepted", args)
		}
	}
}

func TestMCPStdioStartupAndShutdown(t *testing.T) {
	e := ui.NewMapEditor(nil, nil, nil)
	inRead, inWrite := io.Pipe()
	outRead, outWrite := io.Pipe()
	var diagnostic bytes.Buffer
	stop, err := startMCP(e, "stdio", "", 0, "", "", "", inRead, outWrite, &diagnostic)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { stop(); _ = inWrite.Close(); _ = outRead.Close(); _ = outWrite.Close() }()
	if _, err := fmt.Fprintln(inWrite, `{"jsonrpc":"2.0","id":1,"method":"ping"}`); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(outRead).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(line, `"result":{}`) || diagnostic.Len() != 0 {
		t.Fatal(line, diagnostic.String())
	}
}
