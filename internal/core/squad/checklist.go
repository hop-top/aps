package squad

import "fmt"

// CheckResult represents one item in the topology design checklist.
type CheckResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

func pass(name string) CheckResult { return CheckResult{Name: name, Passed: true} }
func fail(name, detail string) CheckResult {
	return CheckResult{Name: name, Passed: false, Detail: detail}
}

// ValidateTopology runs the 8-item design checklist from spec §225-234.
func ValidateTopology(t Topology, contracts []Contract, exitConditions []ExitCondition, contextLoads []ContextLoad) []CheckResult {
	return []CheckResult{
		checkStreamIndependent(t),
		checkSubsystemInterfaceClean(t, contracts),
		checkEnablingExitDefined(t, exitConditions),
		checkPlatformGoldenPath(t),
		checkCollabTimeboxed(contracts),
		checkNoPlatformInternals(t, contracts),
		checkRoutingTypeAware(t, contracts),
		checkTopologyFirst(contextLoads),
	}
}

// 1. Every stream squad can complete work without cross-squad coordination.
func checkStreamIndependent(t Topology) CheckResult {
	for _, s := range t.Squads {
		if s.Type == SquadTypeStream && len(s.Members) == 0 {
			return fail("stream-independent", fmt.Sprintf("stream squad %q has no members", s.ID))
		}
	}
	return pass("stream-independent")
}

// 2. All subsystem logic is behind a clean, versioned interface (XaaS contract).
func checkSubsystemInterfaceClean(t Topology, contracts []Contract) CheckResult {
	for _, s := range t.Squads {
		if s.Type != SquadTypeSubsystem {
			continue
		}
		found := false
		for _, c := range contracts {
			if c.ProviderSquad == s.ID && c.Mode == ModeXaaS {
				found = true
				break
			}
		}
		if !found {
			return fail("subsystem-interface-clean",
				fmt.Sprintf("subsystem squad %q has no XaaS contract as provider", s.ID))
		}
	}
	return pass("subsystem-interface-clean")
}

// 3. Every enabling squad has a defined exit condition.
func checkEnablingExitDefined(t Topology, exitConditions []ExitCondition) CheckResult {
	for _, s := range t.Squads {
		if s.Type != SquadTypeEnabling {
			continue
		}
		found := false
		for _, e := range exitConditions {
			if e.SquadID == s.ID {
				found = true
				break
			}
		}
		if !found {
			return fail("enabling-exit-defined",
				fmt.Sprintf("enabling squad %q has no exit condition defined", s.ID))
		}
	}
	return pass("enabling-exit-defined")
}

// 4. Platform squad has a self-service golden path.
func checkPlatformGoldenPath(t Topology) CheckResult {
	for _, s := range t.Squads {
		if s.Type == SquadTypePlatform && !s.GoldenPathDefined {
			return fail("platform-golden-path",
				fmt.Sprintf("platform squad %q missing golden path definition", s.ID))
		}
	}
	return pass("platform-golden-path")
}

// 5. All collaboration contracts are time-boxed.
func checkCollabTimeboxed(contracts []Contract) CheckResult {
	for _, c := range contracts {
		if c.Mode == ModeCollaboration && c.Timebox == nil {
			return fail("collab-timeboxed",
				fmt.Sprintf("collaboration contract %q has no timebox", c.ID))
		}
	}
	return pass("collab-timeboxed")
}

// 6. No stream squad collaborates with a platform squad (would mean platform internals exposed).
func checkNoPlatformInternals(t Topology, contracts []Contract) CheckResult {
	platformIDs := map[string]bool{}
	for _, s := range t.Squads {
		if s.Type == SquadTypePlatform {
			platformIDs[s.ID] = true
		}
	}
	for _, c := range contracts {
		if c.Mode == ModeCollaboration && platformIDs[c.ProviderSquad] {
			return fail("no-platform-internals",
				fmt.Sprintf("contract %q: platform squad %q as collaboration provider", c.ID, c.ProviderSquad))
		}
	}
	return pass("no-platform-internals")
}

// 7. Routing is type-aware (at least one contract declaring type exists).
func checkRoutingTypeAware(t Topology, contracts []Contract) CheckResult {
	if len(t.Squads) > 1 && len(contracts) == 0 {
		return fail("routing-type-aware", "multiple squads but no interaction contracts declared")
	}
	return pass("routing-type-aware")
}

// 8. Context loads are well-scoped (coordination ratio < 0.5).
func checkTopologyFirst(contextLoads []ContextLoad) CheckResult {
	for _, cl := range contextLoads {
		if !cl.IsWellScoped() {
			return fail("topology-first",
				fmt.Sprintf("squad %q coordination ratio %.2f >= 0.5 — topology may be wrong",
					cl.SquadID, cl.CoordinationRatio()))
		}
	}
	return pass("topology-first")
}
