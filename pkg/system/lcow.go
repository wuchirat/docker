package system

import (
	"fmt"
	"runtime"
	"strings"

	specs "github.com/opencontainers/image-spec/specs-go/v1"
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

// IsPlatformEmpty determines if an OCI image-spec platform structure is not populated.
// TODO This is a temporary function - can be replaced by parsing from
// https://github.com/containerd/containerd/pull/1403/files at a later date.
// @jhowardmsft
func IsPlatformEmpty(platform specs.Platform) bool {
	if platform.Architecture == "" &&
		platform.OS == "" &&
		len(platform.OSFeatures) == 0 &&
		platform.OSVersion == "" &&
		platform.Variant == "" {
		return true
	}
	return false
}
