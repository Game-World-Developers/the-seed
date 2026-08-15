package generators

import (
	"fmt"
	"strings"
)

// unifiedDiff produces a compact, line-based diff between old and new
// text — enough to review what "seed sync --dry-run" would change without
// pulling in an external diff library. It uses a straightforward O(n*m)
// longest-common-subsequence over lines (generated files are small enough
// — typically well under a thousand lines — that this is fast in
// practice) and prints up to `context` unchanged lines around each
// changed region, in the same "-"/"+" style as `diff -u`/git, so it's
// immediately familiar to read.
func unifiedDiff(oldText, newText string, context int) string {
	oldLines := splitLines(oldText)
	newLines := splitLines(newText)

	ops := diffLines(oldLines, newLines)

	var b strings.Builder
	i := 0
	for i < len(ops) {
		if ops[i].kind == diffEqual {
			i++
			continue
		}
		// Found a changed region starting at i; walk back up to `context`
		// leading equal lines, then forward through consecutive
		// changed/near-changed lines, then up to `context` trailing equal
		// lines, matching how `diff -u` groups nearby hunks together.
		start := i
		for start > 0 && i-start < context && ops[start-1].kind == diffEqual {
			start--
		}
		end := i
		for end < len(ops) {
			if ops[end].kind != diffEqual {
				end++
				continue
			}
			// Peek ahead: if another change starts within 2*context, keep
			// this hunk going instead of closing it.
			run := 0
			for end+run < len(ops) && ops[end+run].kind == diffEqual && run < 2*context {
				run++
			}
			if end+run >= len(ops) || ops[end+run].kind == diffEqual {
				end += min(run, context)
				break
			}
			end += run
		}

		fmt.Fprintf(&b, "@@ line %d @@\n", ops[start].oldLine+1)
		for _, op := range ops[start:end] {
			switch op.kind {
			case diffEqual:
				fmt.Fprintf(&b, "  %s\n", op.text)
			case diffDelete:
				fmt.Fprintf(&b, "- %s\n", op.text)
			case diffInsert:
				fmt.Fprintf(&b, "+ %s\n", op.text)
			}
		}
		i = end
	}
	return b.String()
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

type diffKind int

const (
	diffEqual diffKind = iota
	diffDelete
	diffInsert
)

type diffOp struct {
	kind    diffKind
	text    string
	oldLine int // 0-indexed position in the old text, for @@ headers
}

// diffLines aligns old and new via an LCS table, then walks it to produce
// a flat, ordered list of equal/delete/insert operations.
func diffLines(old, new []string) []diffOp {
	n, m := len(old), len(new)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if old[i] == new[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case old[i] == new[j]:
			ops = append(ops, diffOp{diffEqual, old[i], i})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			ops = append(ops, diffOp{diffDelete, old[i], i})
			i++
		default:
			ops = append(ops, diffOp{diffInsert, new[j], i})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{diffDelete, old[i], i})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{diffInsert, new[j], i})
	}
	return ops
}
