package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"againrom/pkg/game"
	"againrom/pkg/ui"
)

func startMCP(e *ui.MapEditor, transport, address string, pace time.Duration, self, config, root string, in io.ReadCloser, out, diagnostic io.Writer) (func(), error) {
	server := e.EnableMCP(pace)
	if transport == "stdio" {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		e.SetMCPConnection("MCP stdio")
		go func() {
			defer close(done)
			if err := server.Serve(ctx, in, out); err != nil && ctx.Err() == nil {
				fmt.Fprintf(diagnostic, "MCP: %v\n", err)
			}
		}()
		return func() { e.CloseMCP(); cancel(); _ = in.Close(); <-done }, nil
	}
	profile, err := game.ResolveRuntimeProfile("", self, config, root)
	if err != nil {
		e.CloseMCP()
		return nil, err
	}
	directory, err := profile.PrepareDirectory(root)
	if err != nil {
		e.CloseMCP()
		return nil, err
	}
	listener, err := server.Listen(address, directory)
	if err != nil {
		e.CloseMCP()
		return nil, err
	}
	e.SetMCPConnection("MCP "+listener.URL, listener.URL, "Bearer token: "+listener.Token, "Token file: "+listener.TokenFile)
	fmt.Fprintf(diagnostic, "MCP: %s; bearer token in %s\n", listener.URL, listener.TokenFile)
	return func() { e.CloseMCP(); _ = listener.Close() }, nil
}
