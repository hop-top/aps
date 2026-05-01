package tui

import (
	"fmt"

	"hop.top/aps/internal/core"
	apsstyles "hop.top/aps/internal/styles"
)

// All list items implement hop.top/kit/go/console/tui.Item (Renderer): they
// expose Render(width int) string and let kit/tui.List manage scrolling.
// Selection styling is baked into each item via the selected flag so the
// kit list — which doesn't itself track focus — can stay generic.

type profileListItem struct {
	id       string
	selected bool
}

func (p profileListItem) Render(_ int) string {
	if p.selected {
		return selectedItemStyle.Render(p.id)
	}
	return itemStyle.Render(p.id)
}

type capabilityListItem struct {
	cap      capItem
	selected bool
}

func (c capabilityListItem) Render(_ int) string {
	dot := apsstyles.StatusDot(c.cap.Enabled)
	kind := apsstyles.KindBadge(c.cap.Kind)
	typ := ""
	if c.cap.Kind == "external" {
		typ = "  " + apsstyles.TypeBadge(c.cap.Type)
	}
	desc := apsstyles.Dim.Render(c.cap.Description)
	line := fmt.Sprintf("%s %-18s %s%s  %s", dot, c.cap.Name, kind, typ, desc)
	if c.selected {
		return selectedItemStyle.Render(line)
	}
	return itemStyle.Render(line)
}

type actionListItem struct {
	action   core.Action
	selected bool
}

func (a actionListItem) Render(_ int) string {
	label := a.action.ID
	if a.action.Title != "" {
		label += " - " + a.action.Title
	}
	if a.selected {
		return selectedItemStyle.Render(label)
	}
	return itemStyle.Render(label)
}
