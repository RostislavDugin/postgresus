package main

import (
	"log/slog"
	"os"
	"syscall"
)

func reexecAfterUpgrade(log *slog.Logger) {
	selfPath, err := os.Executable()
	if err != nil {
		log.Error("Failed to resolve executable for re-exec", "error", err)
		os.Exit(1)
	}

	log.Info("Re-executing after upgrade...")

	if err := syscall.Exec(selfPath, os.Args, os.Environ()); err != nil {
		log.Error("Failed to re-exec after upgrade", "error", err)
		os.Exit(1)
	}
}
