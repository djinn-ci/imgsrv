package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var DJINN_SERVER string

func main() {
	argv0 := os.Args[0]

	var config string

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&config, "config", "djinn-imgsrv.conf", "the config file")
	fs.Parse(os.Args[1:])

	f, err := os.Open(config)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", argv0, err)
		os.Exit(1)
	}

	defer f.Close()

	srv, close, err := DecodeConfig(f)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", argv0, err)
		os.Exit(1)
	}

	if val := os.Getenv("DJINN_SERVER"); val != "" {
		DJINN_SERVER = val
	}

	scanCtx, cancelScan := context.WithCancel(context.Background())
	defer cancelScan()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	ch := make(chan os.Signal, 1)

	signal.Notify(ch, os.Interrupt)

	go func() {
		if err := srv.Serve(scanCtx); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				srv.Log.Error.Println("serve error", err)

				if nerr, ok := err.(net.Error); ok && !nerr.Temporary() {
					ch <- os.Kill
					return
				}
			}
		}
	}()

	srv.Log.Info.Println(argv0, "started on", srv.Addr)

	sig := <-ch

	cancelScan()
	srv.Shutdown(ctx)

	if sig == os.Kill {
		close()
		os.Exit(1)
	}

	srv.Log.Info.Println("received signal", sig, "shutting down")

	close()
}
