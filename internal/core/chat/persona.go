package chat

import (
	"fmt"
	"strings"

	"hop.top/aps/internal/core"
)

// RenderSystemPrompt converts profile identity and boundaries into the system
// prompt shared by CLI/TUI and messenger chat surfaces.
func RenderSystemPrompt(p *core.Profile) string {
	if p == nil {
		return ""
	}

	var sentences []string
	if name := strings.TrimSpace(p.DisplayName); name != "" {
		sentences = append(sentences, "You are "+name+".")
	}
	if tone := strings.TrimSpace(p.Persona.Tone); tone != "" {
		sentences = append(sentences, "Tone: "+tone+".")
	}
	if style := strings.TrimSpace(p.Persona.Style); style != "" {
		sentences = append(sentences, "Style: "+style+".")
	}
	if risk := strings.TrimSpace(p.Persona.Risk); risk != "" {
		sentences = append(sentences, "Risk posture: "+risk+".")
	}

	var lines []string
	if len(sentences) > 0 {
		lines = append(lines, strings.Join(sentences, " "))
	}
	lines = append(lines, scopeLines(p.Scope)...)
	lines = append(lines, limitLines(p.Limits)...)

	return strings.Join(lines, "\n")
}

func scopeLines(scope *core.ScopeConfig) []string {
	if scope == nil {
		return nil
	}
	var lines []string
	if len(scope.FilePatterns) > 0 {
		lines = append(lines, "Scope file patterns: "+strings.Join(scope.FilePatterns, ", ")+".")
	}
	if len(scope.Operations) > 0 {
		lines = append(lines, "Allowed operations: "+strings.Join(scope.Operations, ", ")+".")
	}
	if len(scope.Tools) > 0 {
		lines = append(lines, "Allowed tools: "+strings.Join(scope.Tools, ", ")+".")
	}
	if len(scope.Secrets) > 0 {
		lines = append(lines, "Allowed secrets: "+strings.Join(scope.Secrets, ", ")+".")
	}
	if len(scope.Networks) > 0 {
		lines = append(lines, "Allowed networks: "+strings.Join(scope.Networks, ", ")+".")
	}
	return lines
}

func limitLines(limits core.Limits) []string {
	var lines []string
	if limits.MaxConcurrency > 0 {
		lines = append(lines, fmt.Sprintf("Max concurrency: %d.", limits.MaxConcurrency))
	}
	if limits.MaxRuntimeMinutes > 0 {
		lines = append(lines, fmt.Sprintf("Max runtime minutes: %d.", limits.MaxRuntimeMinutes))
	}
	return lines
}
