package doctor

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"time"
)

const usage = "Usage: stargate doctor [--config manifest.json] [--health] [--gateway-url URL] [--gateway-ca-file FILE] [--json]\nDefault: scoped offline inspection of documented gateway environment fields. No application initialization.\nExit: 0 requested checks passed; 1 failure; 2 usage/input error; 3 incomplete. See diagnostics documentation for limits.\n"

// Run executes diagnostics. It neither initializes application configuration nor starts service clients.
func Run(args []string, lookup func(string) (string, bool), out, errOut io.Writer) int {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	file := flags.String("config", "", "explicit JSON manifest")
	online := flags.Bool("health", false, "check configured health endpoints")
	asJSON := flags.Bool("json", false, "write JSON")
	gatewayURL := flags.String("gateway-url", "", "explicit gateway health root")
	gatewayCA := flags.String("gateway-ca-file", "", "gateway health CA file")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			_, _ = io.WriteString(out, usage)
			return 0
		}
		_, _ = io.WriteString(errOut, "Invalid doctor arguments.\n"+usage)
		return 2
	}
	if flags.NArg() != 0 {
		_, _ = io.WriteString(errOut, "Unexpected doctor arguments.\n"+usage)
		return 2
	}
	var m Manifest
	if *file != "" {
		var err error
		m, err = LoadFile(*file)
		if err != nil {
			_, _ = io.WriteString(errOut, "Cannot load doctor configuration; require bounded JSON, unique fields, string values and restricted credential-file permissions.\n")
			return 2
		}
	}
	if *file == "" {
		m = CurrentEnvironment(lookup)
	}
	if *gatewayURL != "" {
		m.GatewayURL = *gatewayURL
	}
	if *gatewayCA != "" {
		m.GatewayCAFile = *gatewayCA
	}
	report := Inspect(m)
	if *online {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		Health(ctx, m, &report)
	}
	if *asJSON {
		if err := json.NewEncoder(out).Encode(report); err != nil {
			return 2
		}
	} else {
		_, _ = fmt.Fprintf(out, "Stargate doctor: %s (scoped checks only)\n", report.Mode)
		for _, check := range report.Checks {
			_, _ = fmt.Fprintf(out, "[%s] %s: %s\n", check.Status, check.ID, check.Message)
		}
		for _, limit := range report.Limits {
			_, _ = fmt.Fprintf(out, "Limit: %s\n", limit)
		}
	}
	return report.ExitCode()
}
