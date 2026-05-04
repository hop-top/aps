package cli

import "hop.top/aps/internal/cli/skill"

func init() {
	rootCmd.AddCommand(skill.NewSkillCmd())
}
