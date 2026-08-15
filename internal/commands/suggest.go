package commands

import (
	"fmt"
	"regexp"
	"strings"
)

// suggestFor returns an actionable follow-up for a diagnostic message, or
// "" if none applies. This is pattern-matched against message text rather
// than a structured diagnostic code — generators.Diagnostic and
// gameak.Diagnostic are two different types with no shared "kind" field,
// and adding one to both just for this would be a larger refactor than a
// suggestion feature warrants. Matching prose is more fragile than a code
// would be, but every pattern here is matched against this codebase's own
// diagnostic message constants (internal/generators/semantic.go,
// internal/backend/gameak/backend.go), not user-facing free text, so the
// fragility is contained to this file breaking loudly (a missing
// suggestion, never a wrong one) if a diagnostic message's wording changes.
var missingRefPattern = regexp.MustCompile(`references missing (\w+) "([^"]+)"`)
var duplicatePattern = regexp.MustCompile(`^duplicate (\w+) "([^"]+)"`)

func suggestFor(message string) string {
	if m := missingRefPattern.FindStringSubmatch(message); m != nil {
		return fmt.Sprintf("try: seed generate %s %s", m[1], m[2])
	}
	if m := duplicatePattern.FindStringSubmatch(message); m != nil {
		return fmt.Sprintf("try: rename one of the two %q declarations, or give one a different namespace", m[2])
	}
	switch {
	case strings.Contains(message, "ambiguous"):
		return "try: qualify the reference with an explicit namespace, e.g. {name: X, namespace: Y}"
	case strings.Contains(message, "rt.define"), strings.Contains(message, "register_"):
		return "try: rename one of the conflicting declarations so their bare name (ignoring namespace) is unique"
	case strings.Contains(message, "assign distinct priorities"):
		return "try: add a 'priority: N' field to one of the conflicting transitions"
	case strings.Contains(message, "exactly one of name, all, any, or not"):
		return "try: a guard must set exactly one of 'name', 'all', 'any', or 'not' — see docs/cardinal.md §7-9"
	}
	return ""
}
