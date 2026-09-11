package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHealth(t *testing.T) {
    r := newRouter()
    req := httptest.NewRequest(http.MethodGet, "/health", nil)
    rr := httptest.NewRecorder()
    r.ServeHTTP(rr, req)
    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
}

func TestEventIngestionCreatesEvaluationAndAudit(t *testing.T) {
    r := newRouter()
    body := map[string]any{
        "event_id": "evt-001",
        "source_system": "Agent-OS",
        "event_type": "model.action.requested",
        "requires_human_review": false,
    }
    b, _ := json.Marshal(body)
    req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rr := httptest.NewRecorder()
    r.ServeHTTP(rr, req)
    if rr.Code != http.StatusAccepted {
        t.Fatalf("expected 202, got %d body=%s", rr.Code, rr.Body.String())
    }
    var got map[string]any
    if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
        t.Fatal(err)
    }
    if got["decision"] != "ALLOW" {
        t.Fatalf("expected ALLOW, got %v", got["decision"])
    }
    if got["audit_hash"] == "" || got["audit_hash"] == nil {
        t.Fatalf("expected audit hash, got %v", got["audit_hash"])
    }
}

func TestHumanReviewRequestQueuesReview(t *testing.T) {
    r := newRouter()
    body := map[string]any{
        "event_id": "evt-002",
        "source_system": "Model-Gateway",
        "event_type": "model.action.requested",
        "requires_human_review": true,
    }
    b, _ := json.Marshal(body)
    req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rr := httptest.NewRecorder()
    r.ServeHTTP(rr, req)
    if rr.Code != http.StatusAccepted {
        t.Fatalf("expected 202, got %d body=%s", rr.Code, rr.Body.String())
    }
    var got map[string]any
    _ = json.Unmarshal(rr.Body.Bytes(), &got)
    if got["decision"] != "REVIEW" {
        t.Fatalf("expected REVIEW, got %v", got["decision"])
    }

    req2 := httptest.NewRequest(http.MethodGet, "/v1/reviews", nil)
    rr2 := httptest.NewRecorder()
    r.ServeHTTP(rr2, req2)
    if rr2.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr2.Code)
    }
    var reviews []map[string]any
    _ = json.Unmarshal(rr2.Body.Bytes(), &reviews)
    if len(reviews) != 1 || reviews[0]["event_id"] != "evt-002" {
        t.Fatalf("unexpected review queue: %v", reviews)
    }
}

func TestRejectsInvalidEvent(t *testing.T) {
    r := newRouter()
    req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewBufferString(`{"event_id":""}`))
    req.Header.Set("Content-Type", "application/json")
    rr := httptest.NewRecorder()
    r.ServeHTTP(rr, req)
    if rr.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
    }
}
