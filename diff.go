package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const lineLimit = 5000

// DiffSummary holds the count of added, removed, and changed differences.
type DiffSummary struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
	Changed int `json:"changed"`
	Total   int `json:"total"`
}

// ─── Text Diff ───────────────────────────────────────────────────────────────

// LineDiffOp is a single operation in a text diff result.
//
//   - type "equal"   – line present in both sides; Value, LeftLine, RightLine set.
//   - type "added"   – line only in right;          Value, RightLine set.
//   - type "removed" – line only in left;            Value, LeftLine set.
//   - type "changed" – consecutive remove+add pair;  LeftValue, RightValue, LeftLine, RightLine set.
type LineDiffOp struct {
	Type       string `json:"type"`
	Value      string `json:"value,omitempty"`
	LeftValue  string `json:"left_value,omitempty"`
	RightValue string `json:"right_value,omitempty"`
	LeftLine   int    `json:"left_line,omitempty"`
	RightLine  int    `json:"right_line,omitempty"`
}

// TextDiffResult is the response body for the text diff endpoint.
type TextDiffResult struct {
	Diffs   []LineDiffOp `json:"diffs"`
	Summary DiffSummary  `json:"summary"`
}

// lcsOp is an internal operation produced by the LCS backtrack.
type lcsOp struct {
	opType string // "equal", "add", or "remove"
	value  string
	ln     int // left line number (equal/remove) or right line number (add)
	rln    int // right line number (equal only)
}

// lcsDiff computes an LCS-based diff of two string slices, returning a list of
// operations in document order (equal / add / remove).
func lcsDiff(a, b []string) []lcsOp {
	m, n := len(a), len(b)

	// Build LCS DP table.
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] > dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to build the operation list (initially in reverse order).
	ops := make([]lcsOp, 0, m+n)
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
			ops = append(ops, lcsOp{opType: "equal", value: a[i-1], ln: i, rln: j})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			ops = append(ops, lcsOp{opType: "add", value: b[j-1], ln: j})
			j--
		} else {
			ops = append(ops, lcsOp{opType: "remove", value: a[i-1], ln: i})
			i--
		}
	}

	// Reverse in-place to get document order.
	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}
	return ops
}

// splitLines splits text by newline, returning an empty slice for empty text.
func splitLines(text string) []string {
	if text == "" {
		return []string{}
	}
	return strings.Split(text, "\n")
}

// TextDiff performs a line-by-line diff of left and right. A consecutive
// remove+add pair is folded into a single "changed" entry, mirroring the
// behaviour of the browser-based diff tool.
func TextDiff(left, right string) (*TextDiffResult, error) {
	leftLines := splitLines(left)
	rightLines := splitLines(right)

	if len(leftLines) > lineLimit || len(rightLines) > lineLimit {
		return nil, fmt.Errorf("input exceeds %d-line limit", lineLimit)
	}

	rawOps := lcsDiff(leftLines, rightLines)
	diffs := make([]LineDiffOp, 0, len(rawOps))
	var added, removed, changed int

	for i := 0; i < len(rawOps); i++ {
		op := rawOps[i]

		// Pair a remove immediately followed by an add → "changed".
		if op.opType == "remove" && i+1 < len(rawOps) && rawOps[i+1].opType == "add" {
			next := rawOps[i+1]
			diffs = append(diffs, LineDiffOp{
				Type:       "changed",
				LeftValue:  op.value,
				RightValue: next.value,
				LeftLine:   op.ln,
				RightLine:  next.ln,
			})
			changed++
			i++ // consume the paired add
			continue
		}

		switch op.opType {
		case "remove":
			diffs = append(diffs, LineDiffOp{Type: "removed", Value: op.value, LeftLine: op.ln})
			removed++
		case "add":
			diffs = append(diffs, LineDiffOp{Type: "added", Value: op.value, RightLine: op.ln})
			added++
		case "equal":
			diffs = append(diffs, LineDiffOp{
				Type: "equal", Value: op.value,
				LeftLine: op.ln, RightLine: op.rln,
			})
		}
	}

	return &TextDiffResult{
		Diffs: diffs,
		Summary: DiffSummary{
			Added: added, Removed: removed, Changed: changed,
			Total: added + removed + changed,
		},
	}, nil
}

// ─── JSON Diff ───────────────────────────────────────────────────────────────

// JSONDiffOp is a single difference entry in a JSON diff result.
//
//   - type "added"   – key present only in right; RightValue set.
//   - type "removed" – key present only in left;  LeftValue set.
//   - type "changed" – key present in both but values differ; both set.
type JSONDiffOp struct {
	Path       string          `json:"path"`
	Type       string          `json:"type"`
	LeftValue  json.RawMessage `json:"left_value,omitempty"`
	RightValue json.RawMessage `json:"right_value,omitempty"`
}

// JSONDiffResult is the response body for the JSON diff endpoint.
type JSONDiffResult struct {
	Diffs   []JSONDiffOp `json:"diffs"`
	Summary DiffSummary  `json:"summary"`
}

// flattenJSON recursively flattens a JSON value into dot-notation paths.
// Arrays use [n] notation; the root primitive / empty collection uses "(root)".
func flattenJSON(v any, prefix string) map[string]any {
	result := make(map[string]any)

	switch val := v.(type) {
	case map[string]any:
		if len(val) == 0 {
			key := prefix
			if key == "" {
				key = "(root)"
			}
			result[key] = val
			return result
		}
		for k, child := range val {
			newKey := k
			if prefix != "" {
				newKey = prefix + "." + k
			}
			for k2, v2 := range flattenJSON(child, newKey) {
				result[k2] = v2
			}
		}

	case []any:
		if len(val) == 0 {
			key := prefix
			if key == "" {
				key = "(root)"
			}
			result[key] = val
			return result
		}
		for i, item := range val {
			var newKey string
			if prefix != "" {
				newKey = fmt.Sprintf("%s[%d]", prefix, i)
			} else {
				newKey = fmt.Sprintf("[%d]", i)
			}
			for k, v2 := range flattenJSON(item, newKey) {
				result[k] = v2
			}
		}

	default:
		key := prefix
		if key == "" {
			key = "(root)"
		}
		result[key] = val
	}
	return result
}

// stableStringify produces a deterministic JSON string for deep-equality
// comparison (object keys are sorted, matching the JS stableStringify).
// json.Marshal errors are safe to ignore here: v originates from json.Unmarshal,
// which only produces Go types (string, float64, bool, nil, map, slice) that
// json.Marshal always handles without error.
func stableStringify(v any) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			kb, _ := json.Marshal(k) // string key: cannot fail
			parts = append(parts, string(kb)+":"+stableStringify(val[k]))
		}
		return "{" + strings.Join(parts, ",") + "}"
	case []any:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = stableStringify(item)
		}
		return "[" + strings.Join(parts, ",") + "]"
	default:
		b, _ := json.Marshal(val) // primitive from json.Unmarshal: cannot fail
		return string(b)
	}
}

// JSONDiff performs a key-value diff of two JSON strings. Each top-level and
// nested field is identified by its dot-notation path (arrays use [n] notation).
func JSONDiff(leftJSON, rightJSON string) (*JSONDiffResult, error) {
	var leftObj, rightObj any
	if err := json.Unmarshal([]byte(strings.TrimSpace(leftJSON)), &leftObj); err != nil {
		return nil, fmt.Errorf("left JSON parse error: %w", err)
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(rightJSON)), &rightObj); err != nil {
		return nil, fmt.Errorf("right JSON parse error: %w", err)
	}

	leftFlat := flattenJSON(leftObj, "")
	rightFlat := flattenJSON(rightObj, "")

	// Union of all keys, sorted for deterministic output.
	allKeys := make(map[string]struct{}, len(leftFlat)+len(rightFlat))
	for k := range leftFlat {
		allKeys[k] = struct{}{}
	}
	for k := range rightFlat {
		allKeys[k] = struct{}{}
	}
	sortedKeys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	diffs := make([]JSONDiffOp, 0)
	var added, removed, changed int

	// json.Marshal errors below are safe to ignore: lv and rv are values
	// produced by json.Unmarshal and therefore always re-marshalable.
	for _, key := range sortedKeys {
		lv, hasLeft := leftFlat[key]
		rv, hasRight := rightFlat[key]

		switch {
		case !hasLeft:
			rvBytes, _ := json.Marshal(rv)
			diffs = append(diffs, JSONDiffOp{
				Path: key, Type: "added", RightValue: rvBytes,
			})
			added++
		case !hasRight:
			lvBytes, _ := json.Marshal(lv)
			diffs = append(diffs, JSONDiffOp{
				Path: key, Type: "removed", LeftValue: lvBytes,
			})
			removed++
		case stableStringify(lv) != stableStringify(rv):
			lvBytes, _ := json.Marshal(lv)
			rvBytes, _ := json.Marshal(rv)
			diffs = append(diffs, JSONDiffOp{
				Path: key, Type: "changed",
				LeftValue: lvBytes, RightValue: rvBytes,
			})
			changed++
		}
	}

	return &JSONDiffResult{
		Diffs: diffs,
		Summary: DiffSummary{
			Added: added, Removed: removed, Changed: changed,
			Total: added + removed + changed,
		},
	}, nil
}
