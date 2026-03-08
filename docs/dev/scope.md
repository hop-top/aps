# Scope System

The scope system controls what a profile (or squad) can access: files, operations, tools, secrets, and networks. Scopes from multiple sources are combined using intersection logic — the most restrictive wins.

## Scope Rule

A `Rule` defines access boundaries:

```go
type Rule struct {
    FilePatterns []string  // glob patterns for file access
    Operations   []string  // allowed operations
    Tools        []string  // tool names allowed
    Secrets      []string  // secret keys accessible
    Networks     []string  // network addresses/CIDRs
}
```

**Empty slice = unrestricted.** A rule with no `FilePatterns` allows access to all files. A rule with `["src/**"]` restricts to that pattern only.

## Scope Owner

A `Scope` ties a rule to an owner:

```go
type Scope struct {
    OwnerType string  // "profile", "squad", or "workspace"
    OwnerID   string
    Rules     Rule
}
```

## Intersection Logic

When multiple scopes apply, they are intersected. Each field is combined independently:

- If **both** rules specify values, only items present in **both** are kept
- If **either** rule is empty for a field, the other rule's values apply
- If **both** are empty, the field remains unrestricted

```go
// Intersect two rules
result := scope.Intersect(ruleA, ruleB)

// Intersect any number of rules
result := scope.IntersectAll(rule1, rule2, rule3)
```

**Example:**

| Rule A tools | Rule B tools | Result |
|-------------|-------------|--------|
| `[]` (unrestricted) | `[bash, python]` | `[bash, python]` |
| `[bash, python, go]` | `[bash, python]` | `[bash, python]` |
| `[bash]` | `[python]` | `[]` (nothing in common) |
| `[]` | `[]` | `[]` (unrestricted) |

## Scope Resolution

For any given operation, the effective scope is resolved from three layers:

1. **Profile scope** — the profile's own access rules
2. **Squad scopes** — scopes from all squads the profile belongs to
3. **Workspace scope** — the workspace's access rules

```go
type ResolvedScope struct {
    ProfileScope   *Scope
    SquadScopes    []Scope
    WorkspaceScope *Scope
    Effective      Rule  // intersection of all three
}

resolved := scope.Resolve(profileScope, workspaceScope, squadScopes)
// resolved.Effective is the final, most-restrictive access set
```

The effective rule is what APS enforces when the profile takes an action.

## Key Files

| File | Purpose |
|------|---------|
| `internal/core/scope/scope.go` | `Rule`, `Scope` types |
| `internal/core/scope/intersection.go` | `Intersect`, `IntersectAll`, `Resolve`, `ResolvedScope` |
