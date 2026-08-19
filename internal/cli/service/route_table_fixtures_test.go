package service

import "hop.top/aps/internal/core/msgroute"

func routingInline() *msgroute.Config {
	return &msgroute.Config{
		Routes: []msgroute.Route{
			{Match: "+15551234567", Profile: "acme", Action: "inbox"},
			{Match: "unknown", Action: "triage"},
		},
	}
}

func routingWithoutTerminal() *msgroute.Config {
	return &msgroute.Config{
		Routes: []msgroute.Route{
			{Match: "+15551234567", Profile: "acme", Action: "inbox"},
		},
	}
}
