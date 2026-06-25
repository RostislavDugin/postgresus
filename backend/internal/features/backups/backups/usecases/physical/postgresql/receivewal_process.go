package usecases_physical_postgresql

import (
	"os/exec"
	"syscall"
)

// setReceivewalProcessAttributes makes the kernel send pg_receivewal a SIGTERM
// if the Databasus process dies, so a crashed supervisor never leaks an orphaned
// receiver that keeps holding the replication slot. Pdeathsig is Linux-only;
// Databasus ships Linux containers exclusively.
func setReceivewalProcessAttributes(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
}
