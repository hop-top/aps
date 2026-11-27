package core

import "strings"

// renderProfileGitConfig builds the gitconfig seeded into a new profile
// directory. user.name falls back to the profile id when DisplayName is
// empty; user.email is omitted when Email is empty rather than filled
// with a placeholder, so git refuses to commit instead of silently
// misattributing work.
func renderProfileGitConfig(id string, p Profile) string {
	name := p.DisplayName
	if name == "" {
		name = id
	}

	var b strings.Builder
	b.WriteString("[user]\n")
	b.WriteString("\tname = " + quoteGitConfigValue(name) + "\n")
	if p.Email != "" {
		b.WriteString("\temail = " + quoteGitConfigValue(p.Email) + "\n")
	}
	return b.String()
}

// gitConfigValueEscaper applies the git-config escape sequences for a
// double-quoted value (git-config(1), "Syntax"): backslash and double
// quote are escaped, newline and tab use \n and \t. NUL has no escape
// in git-config (git truncates the value at it), so it is dropped.
var gitConfigValueEscaper = strings.NewReplacer(
	`\`, `\\`,
	`"`, `\"`,
	"\n", `\n`,
	"\t", `\t`,
	"\x00", "",
)

// quoteGitConfigValue returns v as a double-quoted git-config value.
// Quoting keeps leading/trailing whitespace and the comment characters
// ';' and '#' literal; escaping keeps the value on one line so it cannot
// open new keys or sections.
func quoteGitConfigValue(v string) string {
	return `"` + gitConfigValueEscaper.Replace(v) + `"`
}
