package listing

// Predicate decides whether a row of type T survives a filter pass.
// Zero-value predicates (nil) MUST be treated as match-all by callers
// using All() — never invoke a nil Predicate directly.
type Predicate[T any] func(T) bool

// All returns a Predicate that matches when every non-nil predicate
// matches. nil predicates are skipped (treated as match-all).
//
// The empty case — All() with no arguments — returns a predicate
// that matches every row. This is the natural identity for filter
// composition and the right behavior for "no filters supplied".
func All[T any](preds ...Predicate[T]) Predicate[T] {
	return func(v T) bool {
		for _, p := range preds {
			if p == nil {
				continue
			}
			if !p(v) {
				return false
			}
		}
		return true
	}
}

// Any returns a Predicate that matches when at least one non-nil
// predicate matches. nil predicates are skipped.
//
// The empty case returns a predicate that matches NO row — opposite
// identity from All. Callers using Any should generally feed it at
// least one predicate; the empty-Any case exists so combinators can
// be built generically.
func Any[T any](preds ...Predicate[T]) Predicate[T] {
	return func(v T) bool {
		for _, p := range preds {
			if p == nil {
				continue
			}
			if p(v) {
				return true
			}
		}
		return false
	}
}

// Not inverts a predicate. A nil inner predicate (which All treats as
// match-all) becomes match-none under Not.
func Not[T any](p Predicate[T]) Predicate[T] {
	return func(v T) bool {
		if p == nil {
			return false
		}
		return !p(v)
	}
}

// Filter applies a predicate to a slice and returns the matches.
// A nil predicate matches every row (consistent with All).
func Filter[T any](rows []T, p Predicate[T]) []T {
	if p == nil {
		out := make([]T, len(rows))
		copy(out, rows)
		return out
	}
	out := make([]T, 0, len(rows))
	for _, r := range rows {
		if p(r) {
			out = append(out, r)
		}
	}
	return out
}

// MatchString returns a Predicate that matches when the field
// extracted by getter equals want. An empty want string is the
// "flag not supplied" case and the predicate matches every row.
//
// Use this for set-membership filters where the user provides a
// concrete value: --capability webhooks, --status active, etc.
func MatchString[T any](getter func(T) string, want string) Predicate[T] {
	if want == "" {
		return nil
	}
	return func(v T) bool {
		return getter(v) == want
	}
}

// MatchSlice returns a Predicate that matches when want is contained
// in the slice extracted by getter. An empty want string matches
// every row (flag-not-supplied case).
//
// Use for filters where the row's field is a list and the flag value
// is a single item: --member <profile> against Squad.Members,
// --capability <name> against Profile.Capabilities.
func MatchSlice[T any](getter func(T) []string, want string) Predicate[T] {
	if want == "" {
		return nil
	}
	return func(v T) bool {
		for _, item := range getter(v) {
			if item == want {
				return true
			}
		}
		return false
	}
}

// BoolFlag returns a Predicate gated on a tristate flag value.
// kit/cli's bool flags expose Changed() to distinguish "user did not
// supply the flag" from "user supplied --flag=false". The standard
// callsite shape is:
//
//	listing.BoolFlag(cmd.Flags().Changed("has-identity"),
//	    func(p Profile) bool { return p.Identity != nil },
//	    mustBoolFlag(cmd, "has-identity"))
//
// changed=false → predicate is nil (match-all). changed=true → row
// must satisfy the getter == want check.
func BoolFlag[T any](changed bool, getter func(T) bool, want bool) Predicate[T] {
	if !changed {
		return nil
	}
	return func(v T) bool {
		return getter(v) == want
	}
}
