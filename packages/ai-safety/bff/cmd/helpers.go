package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

func getEnvAsInt(name string, defaultVal int) int {
	if value, exists := os.LookupEnv(name); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultVal
}

func getEnvAsString(name string, defaultVal string) string {
	if value, exists := os.LookupEnv(name); exists {
		return value
	}
	return defaultVal
}

func getEnvAsBool(name string, defaultVal bool) bool {
	if value, exists := os.LookupEnv(name); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultVal
}

func parseLevel(s string) slog.Level {
	var level slog.Level
	err := level.UnmarshalText([]byte(s))
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid log level %q, valid levels are: error, warn, info, debug\n", s)
		os.Exit(1)
	}
	return level
}

func newOriginParser(allowList *[]string) func(s string) error {
	return func(s string) error {
		parsed := []string{}
		for _, str := range strings.Split(s, ",") {
			if trimmed := strings.TrimSpace(str); trimmed != "" {
				parsed = append(parsed, trimmed)
			}
		}
		*allowList = parsed
		return nil
	}
}

// newDialMapParser parses comma-separated host=address pairs (for example
// "my-guardrail.my-project.svc.cluster.local=127.0.0.1:9443") into a map used
// to redirect guardrail service connections to local port-forwards. Each
// invocation replaces any previously parsed value.
func newDialMapParser(dialMap *map[string]string) func(s string) error {
	return func(s string) error {
		parsed := map[string]string{}
		for _, entry := range strings.Split(s, ",") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			host, address, found := strings.Cut(entry, "=")
			host = strings.TrimSpace(host)
			address = strings.TrimSpace(address)
			if !found || host == "" || address == "" {
				return fmt.Errorf("invalid dial map entry %q, expected host=address", entry)
			}
			parsed[host] = address
		}
		if len(parsed) == 0 {
			*dialMap = nil
		} else {
			*dialMap = parsed
		}
		return nil
	}
}
