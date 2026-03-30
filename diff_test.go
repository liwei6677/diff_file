package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── lcsDiff ─────────────────────────────────────────────────────────────────

func TestLcsDiffIdentical(t *testing.T) {
	ops := lcsDiff([]string{"a", "b", "c"}, []string{"a", "b", "c"})
	for _, op := range ops {
		if op.opType != "equal" {
			t.Errorf("expected all equal ops, got %q", op.opType)
		}
	}
}

func TestLcsDiffBothEmpty(t *testing.T) {
	ops := lcsDiff([]string{}, []string{})
	if len(ops) != 0 {
		t.Errorf("expected no ops, got %d", len(ops))
	}
}

func TestLcsDiffAddAndRemove(t *testing.T) {
	ops := lcsDiff([]string{"a", "b"}, []string{"a", "c"})
	// expected: equal(a), remove(b), add(c)
	if len(ops) != 3 {
		t.Fatalf("expected 3 ops, got %d", len(ops))
	}
	if ops[0].opType != "equal" || ops[0].value != "a" {
		t.Errorf("op[0]: want equal(a), got %+v", ops[0])
	}
	if ops[1].opType != "remove" || ops[1].value != "b" {
		t.Errorf("op[1]: want remove(b), got %+v", ops[1])
	}
	if ops[2].opType != "add" || ops[2].value != "c" {
		t.Errorf("op[2]: want add(c), got %+v", ops[2])
	}
}

func TestLcsDiffLineNumbers(t *testing.T) {
	ops := lcsDiff([]string{"x"}, []string{"x"})
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	if ops[0].ln != 1 || ops[0].rln != 1 {
		t.Errorf("expected ln=1 rln=1, got ln=%d rln=%d", ops[0].ln, ops[0].rln)
	}
}

// ─── TextDiff ────────────────────────────────────────────────────────────────

func TestTextDiffIdentical(t *testing.T) {
	result, err := TextDiff("hello\nworld", "hello\nworld")
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Total != 0 {
		t.Errorf("expected 0 differences, got %d", result.Summary.Total)
	}
	if len(result.Diffs) != 2 {
		t.Errorf("expected 2 equal diffs, got %d", len(result.Diffs))
	}
}

func TestTextDiffBothEmpty(t *testing.T) {
	result, err := TextDiff("", "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Total != 0 {
		t.Errorf("expected 0 differences, got %d", result.Summary.Total)
	}
}

func TestTextDiffAdded(t *testing.T) {
	result, err := TextDiff("line1", "line1\nline2")
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Added != 1 || result.Summary.Total != 1 {
		t.Errorf("expected added=1 total=1, got added=%d total=%d",
			result.Summary.Added, result.Summary.Total)
	}
}

func TestTextDiffRemoved(t *testing.T) {
	result, err := TextDiff("line1\nline2", "line1")
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Removed != 1 {
		t.Errorf("expected 1 removed, got %d", result.Summary.Removed)
	}
}

func TestTextDiffChanged(t *testing.T) {
	result, err := TextDiff("hello", "world")
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Changed != 1 {
		t.Errorf("expected 1 changed, got %d", result.Summary.Changed)
	}
	d := result.Diffs[0]
	if d.Type != "changed" {
		t.Errorf("expected type=changed, got %s", d.Type)
	}
	if d.LeftValue != "hello" || d.RightValue != "world" {
		t.Errorf("expected left=hello right=world, got left=%s right=%s", d.LeftValue, d.RightValue)
	}
	if d.LeftLine != 1 || d.RightLine != 1 {
		t.Errorf("expected left_line=1 right_line=1, got left_line=%d right_line=%d",
			d.LeftLine, d.RightLine)
	}
}

func TestTextDiffLineLimit(t *testing.T) {
	lines := make([]string, lineLimit+1)
	for i := range lines {
		lines[i] = "line"
	}
	_, err := TextDiff(strings.Join(lines, "\n"), "")
	if err == nil {
		t.Error("expected error for input exceeding line limit")
	}
}

// ─── flattenJSON ─────────────────────────────────────────────────────────────

func TestFlattenJSONPrimitive(t *testing.T) {
	flat := flattenJSON(float64(42), "")
	if flat["(root)"] != float64(42) {
		t.Errorf("expected (root)=42, got %v", flat["(root)"])
	}
}

func TestFlattenJSONSimple(t *testing.T) {
	var obj any
	if err := json.Unmarshal([]byte(`{"a":1,"b":"hello"}`), &obj); err != nil {
		t.Fatal(err)
	}
	flat := flattenJSON(obj, "")
	if flat["a"] != float64(1) {
		t.Errorf("expected a=1, got %v", flat["a"])
	}
	if flat["b"] != "hello" {
		t.Errorf("expected b=hello, got %v", flat["b"])
	}
}

func TestFlattenJSONNested(t *testing.T) {
	var obj any
	if err := json.Unmarshal([]byte(`{"user":{"name":"Alice","age":30}}`), &obj); err != nil {
		t.Fatal(err)
	}
	flat := flattenJSON(obj, "")
	if flat["user.name"] != "Alice" {
		t.Errorf("expected user.name=Alice, got %v", flat["user.name"])
	}
	if flat["user.age"] != float64(30) {
		t.Errorf("expected user.age=30, got %v", flat["user.age"])
	}
}

func TestFlattenJSONArray(t *testing.T) {
	var obj any
	if err := json.Unmarshal([]byte(`{"items":[1,2,3]}`), &obj); err != nil {
		t.Fatal(err)
	}
	flat := flattenJSON(obj, "")
	if flat["items[0]"] != float64(1) {
		t.Errorf("expected items[0]=1, got %v", flat["items[0]"])
	}
	if flat["items[2]"] != float64(3) {
		t.Errorf("expected items[2]=3, got %v", flat["items[2]"])
	}
}

func TestFlattenJSONEmptyObject(t *testing.T) {
	var obj any
	if err := json.Unmarshal([]byte(`{}`), &obj); err != nil {
		t.Fatal(err)
	}
	flat := flattenJSON(obj, "")
	if _, ok := flat["(root)"]; !ok {
		t.Error("expected (root) key for empty object")
	}
}

// ─── stableStringify ─────────────────────────────────────────────────────────

func TestStableStringifyNil(t *testing.T) {
	if got := stableStringify(nil); got != "null" {
		t.Errorf("expected null, got %s", got)
	}
}

func TestStableStringifyObjectKeyOrder(t *testing.T) {
	var a, b any
	if err := json.Unmarshal([]byte(`{"z":1,"a":2}`), &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"a":2,"z":1}`), &b); err != nil {
		t.Fatal(err)
	}
	if stableStringify(a) != stableStringify(b) {
		t.Error("stableStringify should produce same output regardless of key order")
	}
}

// ─── JSONDiff ─────────────────────────────────────────────────────────────────

func TestJSONDiffIdentical(t *testing.T) {
	result, err := JSONDiff(`{"a":1}`, `{"a":1}`)
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Total != 0 {
		t.Errorf("expected 0 differences, got %d", result.Summary.Total)
	}
}

func TestJSONDiffAdded(t *testing.T) {
	result, err := JSONDiff(`{"a":1}`, `{"a":1,"b":2}`)
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Added != 1 {
		t.Errorf("expected 1 added, got %d", result.Summary.Added)
	}
	if result.Diffs[0].Path != "b" || result.Diffs[0].Type != "added" {
		t.Errorf("unexpected diff entry: %+v", result.Diffs[0])
	}
}

func TestJSONDiffRemoved(t *testing.T) {
	result, err := JSONDiff(`{"a":1,"b":2}`, `{"a":1}`)
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Removed != 1 {
		t.Errorf("expected 1 removed, got %d", result.Summary.Removed)
	}
}

func TestJSONDiffChanged(t *testing.T) {
	result, err := JSONDiff(`{"a":"old"}`, `{"a":"new"}`)
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Changed != 1 {
		t.Errorf("expected 1 changed, got %d", result.Summary.Changed)
	}
}

func TestJSONDiffNestedPath(t *testing.T) {
	result, err := JSONDiff(`{"user":{"name":"Alice"}}`, `{"user":{"name":"Bob"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Changed != 1 {
		t.Errorf("expected 1 changed, got %d", result.Summary.Changed)
	}
	if result.Diffs[0].Path != "user.name" {
		t.Errorf("expected path=user.name, got %s", result.Diffs[0].Path)
	}
}

func TestJSONDiffInvalidLeftJSON(t *testing.T) {
	_, err := JSONDiff(`{invalid}`, `{}`)
	if err == nil {
		t.Error("expected error for invalid left JSON")
	}
}

func TestJSONDiffInvalidRightJSON(t *testing.T) {
	_, err := JSONDiff(`{}`, `{invalid}`)
	if err == nil {
		t.Error("expected error for invalid right JSON")
	}
}

func TestJSONDiffKeyOrder(t *testing.T) {
	// Keys with different insertion order should produce no differences.
	result, err := JSONDiff(`{"z":1,"a":2}`, `{"a":2,"z":1}`)
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Total != 0 {
		t.Errorf("expected 0 differences for same JSON with different key order, got %d",
			result.Summary.Total)
	}
}

// ─── HTTP handlers ────────────────────────────────────────────────────────────

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", resp["status"])
	}
}

func TestHandleTextDiff(t *testing.T) {
	body := `{"left":"hello\nworld","right":"hello\ngolang"}`
	req := httptest.NewRequest(http.MethodPost, "/api/diff/text", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleTextDiff(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp TextDiffResult
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Summary.Changed != 1 {
		t.Errorf("expected 1 changed, got %d", resp.Summary.Changed)
	}
}

func TestHandleJSONDiff(t *testing.T) {
	body := `{"left":"{\"a\":1}","right":"{\"a\":2}"}`
	req := httptest.NewRequest(http.MethodPost, "/api/diff/json", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleJSONDiff(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp JSONDiffResult
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Summary.Changed != 1 {
		t.Errorf("expected 1 changed, got %d", resp.Summary.Changed)
	}
}

func TestHandleTextDiffBadJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/diff/text", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleTextDiff(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleJSONDiffBadJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/diff/json", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleJSONDiff(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleJSONDiffInvalidPayloadJSON(t *testing.T) {
	// Valid outer JSON but invalid inner left/right JSON value.
	body := `{"left":"{bad}","right":"{}"}`
	req := httptest.NewRequest(http.MethodPost, "/api/diff/json", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleJSONDiff(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCORSOptions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	handler := corsMiddleware(mux)

	req := httptest.NewRequest(http.MethodOptions, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS header Access-Control-Allow-Origin: *")
	}
}
