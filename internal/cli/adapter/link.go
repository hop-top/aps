package adapter

import (
	"encoding/json"
	"fmt"
	"strings"

	coreadapter "hop.top/aps/internal/core/adapter"
	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/events"

	"github.com/spf13/cobra"
)

func newLinkCmd() *cobra.Command {
	var profileID string
	var jsonOutput bool
	var dryRun bool

	// Messenger-specific flags
	var mappings []string
	var addMapping string
	var removeMapping string
	var defaultAction string

	cmd := &cobra.Command{
		Use:   "link <device>",
		Short: "Link a device to a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := linkOpts{
				deviceName:    args[0],
				profileID:     profileID,
				jsonOut:       jsonOutput,
				dryRun:        dryRun,
				mappings:      mappings,
				addMapping:    addMapping,
				removeMapping: removeMapping,
				defaultAction: defaultAction,
			}
			return runLink(opts)
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile to link (required)")
	cmd.MarkFlagRequired("profile")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be linked without linking")

	// Messenger-specific flags
	cmd.Flags().StringSliceVar(&mappings, "mapping", nil,
		"Channel=Action mapping (messenger devices only, repeatable)")
	cmd.Flags().StringVar(&addMapping, "add-mapping", "",
		"Add a single channel=action mapping to an existing link")
	cmd.Flags().StringVar(&removeMapping, "remove-mapping", "",
		"Remove a mapping by channel ID from an existing link")
	cmd.Flags().StringVar(&defaultAction, "default-action", "",
		"Set default action for unmapped channels")

	return cmd
}

type linkOpts struct {
	deviceName    string
	profileID     string
	jsonOut       bool
	dryRun        bool
	mappings      []string
	addMapping    string
	removeMapping string
	defaultAction string
}

func runLink(opts linkOpts) error {
	dev, err := coreadapter.LoadAdapter(opts.deviceName)
	if err != nil {
		return err
	}

	// Handle messenger-specific mutation flags on existing links
	if dev.Type == coreadapter.AdapterTypeMessenger {
		if opts.addMapping != "" {
			return runAddMapping(opts)
		}
		if opts.removeMapping != "" {
			return runRemoveMapping(opts)
		}
		if opts.defaultAction != "" && len(opts.mappings) == 0 {
			return runSetDefaultAction(opts)
		}
	}

	if opts.dryRun {
		return renderLinkDryRun(dev, opts.profileID, opts.mappings)
	}

	// Link the device at the device level
	if !dev.IsLinkedToProfile(opts.profileID) {
		err = defaultManager.LinkAdapter(opts.deviceName, opts.profileID)
		if err != nil {
			return err
		}

		publishEvent(string(events.TopicAdapterLinked), "", events.AdapterLinkedPayload{
			ProfileID:   opts.profileID,
			AdapterType: string(dev.Type),
			AdapterID:   opts.deviceName,
		})
	}

	// Handle messenger mappings
	if dev.Type == coreadapter.AdapterTypeMessenger && len(opts.mappings) > 0 {
		parsedMappings, err := parseMappings(opts.mappings)
		if err != nil {
			return err
		}

		err = messengerManager.LinkMessengerToProfile(opts.deviceName, opts.profileID, parsedMappings)
		if err != nil {
			// If link already exists, add mappings individually
			if isLinkAlreadyExists(err) {
				for channelID, action := range parsedMappings {
					if addErr := messengerManager.AddMapping(
						opts.deviceName, opts.profileID, channelID, action,
					); addErr != nil {
						return addErr
					}
				}
			} else {
				return err
			}
		}

		// Set default action if provided alongside mappings
		if opts.defaultAction != "" {
			if err := messengerManager.SetDefaultAction(
				opts.deviceName, opts.profileID, opts.defaultAction,
			); err != nil {
				return err
			}
		}
	}

	if opts.jsonOut {
		return renderLinkJSON(dev, opts.profileID, opts.mappings)
	}

	if dev.Type == coreadapter.AdapterTypeMessenger && len(opts.mappings) > 0 {
		fmt.Printf("Linked %s to %s with %d mapping(s)\n",
			opts.deviceName, opts.profileID, len(opts.mappings))
	} else {
		fmt.Printf("Linked %s to %s\n", opts.deviceName, opts.profileID)
	}
	return nil
}

func runAddMapping(opts linkOpts) error {
	channelID, action, err := parseMapping(opts.addMapping)
	if err != nil {
		return err
	}

	err = messengerManager.AddMapping(opts.deviceName, opts.profileID, channelID, action)
	if err != nil {
		if msgtypes.IsMappingConflict(err) {
			fmt.Printf("Error: %s\n", err)
			return err
		}
		return err
	}

	if opts.jsonOut {
		data := map[string]any{
			"device":     opts.deviceName,
			"profile":    opts.profileID,
			"channel_id": channelID,
			"action":     action,
			"operation":  "add_mapping",
		}
		out, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	fmt.Printf("Added mapping %s=%s for %s -> %s\n",
		channelID, action, opts.deviceName, opts.profileID)
	return nil
}

func runRemoveMapping(opts linkOpts) error {
	err := messengerManager.RemoveMapping(opts.deviceName, opts.profileID, opts.removeMapping)
	if err != nil {
		return err
	}

	if opts.jsonOut {
		data := map[string]any{
			"device":     opts.deviceName,
			"profile":    opts.profileID,
			"channel_id": opts.removeMapping,
			"operation":  "remove_mapping",
		}
		out, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	fmt.Printf("Removed mapping for channel '%s' from %s -> %s\n",
		opts.removeMapping, opts.deviceName, opts.profileID)
	return nil
}

func runSetDefaultAction(opts linkOpts) error {
	err := messengerManager.SetDefaultAction(opts.deviceName, opts.profileID, opts.defaultAction)
	if err != nil {
		return err
	}

	if opts.jsonOut {
		data := map[string]any{
			"device":         opts.deviceName,
			"profile":        opts.profileID,
			"default_action": opts.defaultAction,
			"operation":      "set_default_action",
		}
		out, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	fmt.Printf("Set default action '%s' for %s -> %s\n",
		opts.defaultAction, opts.deviceName, opts.profileID)
	return nil
}

func parseMappings(raw []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, m := range raw {
		channelID, action, err := parseMapping(m)
		if err != nil {
			return nil, err
		}
		result[channelID] = action
	}
	return result, nil
}

func parseMapping(raw string) (string, string, error) {
	parts := strings.SplitN(raw, "=", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf(
			"invalid mapping format '%s': expected 'channel_id=action'", raw)
	}
	return parts[0], parts[1], nil
}

func isLinkAlreadyExists(err error) bool {
	if e, ok := err.(*msgtypes.MessengerError); ok {
		return e.Code == msgtypes.ErrCodeLinkAlreadyExists
	}
	return false
}

func renderLinkDryRun(dev *coreadapter.Adapter, profileID string, mappings []string) error {
	fmt.Printf("Dry run: linking %s to %s\n\n", dev.Name, profileID)
	fmt.Printf("  Device:     %s (%s)\n", dev.Name, dev.Type)
	fmt.Printf("  Profile:    %s\n", profileID)
	if dev.IsLinkedToProfile(profileID) {
		fmt.Printf("  Status:     already linked\n")
	} else {
		fmt.Printf("  Status:     will be linked\n")
	}

	if dev.Type == coreadapter.AdapterTypeMessenger && len(mappings) > 0 {
		fmt.Printf("  Mappings:\n")
		for _, m := range mappings {
			fmt.Printf("    %s\n", m)
		}
	}

	fmt.Println()
	fmt.Println("No changes made. Remove --dry-run to link.")
	return nil
}

func renderLinkJSON(dev *coreadapter.Adapter, profileID string, mappings []string) error {
	data := map[string]any{
		"device":   dev.Name,
		"profile":  profileID,
		"linked":   true,
		"strategy": dev.Strategy,
	}
	if len(mappings) > 0 {
		data["mappings"] = mappings
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
