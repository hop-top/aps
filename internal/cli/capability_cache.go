package cli

import (
	"fmt"
	"os"

	"hop.top/aps/internal/core/capability"
	"hop.top/aps/internal/storage"
)

// capabilityCache is the process-wide sqlstore-backed cache for
// capability metadata. Lazily initialised in init; nil if open fails
// (in which case capability.LoadCapabilityCached falls through to the
// filesystem walk — same behaviour as before this cache existed).
var capabilityCache *storage.CapabilityCache

func init() {
	c, err := storage.NewCapabilityCache(storage.CapabilityCacheOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: capability cache: open failed (%v); falling back to filesystem\n", err)
		return
	}
	capabilityCache = c
	capability.SetCache(c)
}
