package main

import (
	"io"
	"os"

	"github.com/soulteary/stargate/src/internal/doctor"
)

// dispatchDoctor runs before any application, audit, tracing or SDK initialization.
func dispatchDoctor(args []string, out, errOut io.Writer) (bool, int) {
	if len(args) == 0 || args[0] != "doctor" {
		return false, 0
	}
	return true, doctor.Run(args[1:], os.LookupEnv, out, errOut)
}
