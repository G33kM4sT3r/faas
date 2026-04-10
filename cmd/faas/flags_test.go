package main

import "testing"

// TestSetupFlagsRegistersExpectedNames asserts that each command exposes the
// flags its docs and CLI reference advertise. If a flag is renamed or removed
// these will fail loudly.
func TestSetupFlagsRegistersExpectedNames(t *testing.T) {
	setupUpFlags()
	setupDownFlags()
	setupLsFlags()
	setupLogsFlags()
	setupInvokeFlags()

	cases := []struct {
		cmdName string
		flags   []string
	}{
		{"up", []string{"port", "name", "env", "force", "no-cache"}},
		{"down", []string{"all", "keep-image"}},
		{"ls", []string{"json", "quiet"}},
		{"logs", []string{"follow", "no-follow", "lines", "json", "level"}},
		{"invoke", []string{"method", "data", "header", "path"}},
	}

	for _, c := range cases {
		t.Run(c.cmdName, func(t *testing.T) {
			var cmd = upCmd
			switch c.cmdName {
			case "down":
				cmd = downCmd
			case "ls":
				cmd = lsCmd
			case "logs":
				cmd = logsCmd
			case "invoke":
				cmd = invokeCmd
			}
			for _, f := range c.flags {
				if cmd.Flags().Lookup(f) == nil {
					t.Errorf("%s: flag %q not registered", c.cmdName, f)
				}
			}
		})
	}
}
