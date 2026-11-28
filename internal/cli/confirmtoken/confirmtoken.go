// Package confirmtoken holds the help paragraph shared by every
// destructive command gated by kit's typed confirmation token.
package confirmtoken

// Help is appended to the Long help of every command that opts into the
// typed confirmation token (kitcli.SetDestructiveToken). The gate runs
// before the command and is the same on a terminal and off one, so the
// command's own --yes / --force only skip its extra interactive prompt.
const Help = `

Confirmation token: this command refuses to run without --confirm-token,
on a terminal and off one; --yes, --force and --confirm=yes do not
replace it. Run the command once without the token: it changes nothing,
exits with code 5 (UNAUTHORIZED), and prints the token (under --format
json or yaml it is in the error's message and suggested_fix). Then run
it again with --confirm-token=<token>. The token is fixed per command:
it is the same on every run and does not depend on the arguments.`
