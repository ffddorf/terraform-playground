package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ffddorf/tf-preview-github/pkg/quicpunch"
)

func main() {
	var session string
	flag.StringVar(&session, "session", "", "WebRTC session (base64)")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	f, err := os.Create("workspace.tar.gz")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := quicpunch.FetchWorkspace(ctx, session, f); err != nil {
		panic(err)
	}
	fmt.Println("Done downloading")
}
