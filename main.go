// Bilibili Region Proxy - Go port of Bilibili-Region-Proxy.py
//
// Serves as a regional HTTP resolver for Bilibili main-site content.
package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/pflag"

	"bilibili-region-proxy/internal/proxy"
)

const (
	defaultListenAddress = "0.0.0.0"
	defaultPort          = 10001
)

// ─────────────────────────────────────────────────────────────────────────
// Server
// ─────────────────────────────────────────────────────────────────────────

func main() {
	var configFile string
	pflag.StringVarP(&configFile, "config", "c", "", "path to the JSON config file")
	port := pflag.Int("port", defaultPort, "listen port")
	listenAddress := pflag.String("listen-address", defaultListenAddress, "listen address")
	pflag.Usage = func() {
		out := pflag.CommandLine.Output()
		fmt.Fprintf(out, "Usage: %s [options]\n\nOptions:\n", filepath.Base(os.Args[0]))
		pflag.PrintDefaults()
	}
	pflag.Parse()

	if configFile != "" {
		if absolute, err := filepath.Abs(configFile); err == nil {
			configFile = absolute
		}
		proxy.SetConfigFile(configFile)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/", proxy.Handler)

	addr := net.JoinHostPort(*listenAddress, strconv.Itoa(*port))
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("bilibili-region-proxy listening on %s", addr)
	log.Printf("config file: %s", proxy.ConfigFile())
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
