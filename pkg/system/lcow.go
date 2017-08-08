package system

import (
	"fmt"
	"runtime"
	"strings"
)

// ValidatePlatform validates that a platform is valid on the daemon
func ValidatePlatform(platform string) error {
	platform = strings.ToLower(platform)
	valid := []string{runtime.GOOS}

	if LCOWSupported() {
		valid = append(valid, "linux")
	}

	for _, item := range valid {
		if item == platform {
			return nil
		}
	}
	return fmt.Errorf("invalid platform: %q", platform)
}
