// Command kinex-mark serves the Kinex Mark bookmark manager: the JSON API
// and the embedded web interface in a single binary.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/kaszasg/kinex-mark/internal/app"
	"github.com/kaszasg/kinex-mark/internal/config"
)

var version = "dev" // -ldflags "-X main.version=..."

func main() {
	cfg, err := config.Load(flag.CommandLine, os.Args[1:], os.Getenv)
	if err != nil {
		slog.New(slog.NewTextHandler(os.Stderr, nil)).Error("invalid configuration", "err", err)
		os.Exit(2)
	}
	if cfg.ShowVersion {
		fmt.Println(version)
		return
	}
	if cfg.HealthCheck {
		os.Exit(app.CheckHealth(cfg))
	}
	log := app.NewLogger(cfg, os.Stdout)
	if cfg.AdminCommand != nil { // -unlock-user, -reset-password: offline recovery, then exit
		os.Exit(app.RunAdminCommand(context.Background(), cfg, os.Stdout, log))
	}
	if err := app.Run(context.Background(), cfg, version, log); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
