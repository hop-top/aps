package adapter

import (
	"hop.top/aps/internal/styles"
)

// T-0479 — `tableHeader` (lipgloss bold-dim style) was removed when
// the presence / pending / channels tables migrated to
// listing.RenderList (T-0474..T-0476). Header styling now flows from
// the active kit/cli theme via the styled table renderer on TTY
// writers; the lipgloss/v2 import is dropped with it.
var (
	headerStyle  = styles.Title
	dimStyle     = styles.Dim
	boldStyle    = styles.Bold
	errorStyle   = styles.Error
	successStyle = styles.Success
	warnStyle    = styles.Warn
)
