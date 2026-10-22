// Test-only connector fixture for browser-to-Podux-to-child lifecycle QA.
// It does not connect to Cloudflare and must never be used as a real connector.
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	args := os.Args[1:]
	for _, arg := range args {
		if arg == "--version" {
			fmt.Println("cloudflared version 2025.4.0 (built 2025-04-02-1524 UTC)")
			return
		}
		if arg == "--help" {
			fmt.Println("--token-file --protocol --region --retries --grace-period --edge-ip-version --edge-bind-address --post-quantum --metrics --loglevel --no-autoupdate --config --ha-connections")
			return
		}
	}
	value := func(flag string) string {
		for i, arg := range args {
			if arg == flag && i+1 < len(args) {
				return args[i+1]
			}
		}
		return ""
	}
	token, e := os.ReadFile(value("--token-file"))
	if e != nil || strings.TrimSpace(string(token)) == "" {
		os.Exit(2)
	}
	listener, e := net.Listen("tcp4", value("--metrics"))
	if e != nil {
		os.Exit(3)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprintln(w, "cloudflared_tunnel_ha_connections 2") })
	server := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second}
	go server.Serve(listener)
	fmt.Fprintln(os.Stderr, " INF Starting metrics server on "+listener.Addr().String()+"/metrics")
	fmt.Fprintln(os.Stderr, " INF Registered tunnel connection protocol=quic")
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	<-done
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
