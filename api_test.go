package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestFetchIssues_CapsAtSonarQube10000ResultLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, _ := strconv.Atoi(r.URL.Query().Get("p"))
		ps, _ := strconv.Atoi(r.URL.Query().Get("ps"))
		if p*ps > 10000 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(Response{
				Errors: []struct {
					Msg string `json:"msg"`
				}{{Msg: "Can return only the first 10000 results"}},
			})
			return
		}

		batch := make([]Issue, ps)
		for i := range batch {
			batch[i] = Issue{Key: strconv.Itoa((p-1)*ps + i), Severity: "LOW"}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Response{
			Paging: Paging{Total: 12000},
			Issues: batch,
		})
	}))
	defer srv.Close()

	originalURL := SONAR_URL
	SONAR_URL = srv.URL
	defer func() { SONAR_URL = originalURL }()

	issues, err := fetchIssues("huge-proj", "token", []string{"SECURITY"}, []string{"LOW"}, false)
	if err != nil {
		t.Fatalf("expected fetchIssues to stop at 10000 limit without error, got: %v", err)
	}
	if len(issues) != 10000 {
		t.Errorf("expected 10000 issues, got %d", len(issues))
	}
}

func TestIsValidURL(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"http://localhost:9000", true},
		{"https://sonar.example.com/", true},
		{"ftp://sonar.example.com", false},
		{"http://", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isValidURL(tc.input); got != tc.want {
			t.Errorf("isValidURL(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestFetchIssues_MapsHighestMatchingImpactSeverityAndPreservesLegacyFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Response{
			Paging: Paging{Total: 2},
			Issues: []Issue{
				{
					Key:      "issue-multi-impact",
					Severity: "MAJOR",
					Impacts: []struct {
						SoftwareQuality string `json:"softwareQuality"`
						Severity        string `json:"severity"`
					}{
						{SoftwareQuality: "MAINTAINABILITY", Severity: "LOW"},
						{SoftwareQuality: "SECURITY", Severity: "BLOCKER"},
					},
				},
				{
					Key:      "issue-legacy-no-impacts",
					Severity: "HIGH",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	originalURL := SONAR_URL
	SONAR_URL = srv.URL
	defer func() { SONAR_URL = originalURL }()

	issues, err := fetchIssues("test-proj", "token-123", []string{"MAINTAINABILITY", "SECURITY"}, []string{"BLOCKER", "HIGH", "LOW"}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Severity != "BLOCKER" {
		t.Errorf("expected highest matching impact severity BLOCKER, got %s", issues[0].Severity)
	}
	if issues[0].SoftwareQuality != "SECURITY" {
		t.Errorf("expected highest matching impact software quality SECURITY, got %s", issues[0].SoftwareQuality)
	}
	if issues[1].Severity != "HIGH" {
		t.Errorf("expected legacy issue without impacts to preserve HIGH, got %s", issues[1].Severity)
	}
}

func TestFetchIssuesWithOptions_PassesBranchAndPullRequestAndReportsTruncation(t *testing.T) {
	var gotBranch, gotPR string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBranch = r.URL.Query().Get("branch")
		gotPR = r.URL.Query().Get("pullRequest")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Response{
			Paging: Paging{Total: 12500},
			Issues: []Issue{},
		})
	}))
	defer srv.Close()

	res, err := fetchIssuesWithOptions(FetchOptions{
		BaseURL:           srv.URL,
		ProjectKey:        "my-proj",
		Token:             "tok",
		SoftwareQualities: []string{"SECURITY"},
		Severities:        []string{"HIGH"},
		IsNewCodePeriod:   true,
		Branch:            "feature/login",
		PullRequest:       "42",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBranch != "feature/login" {
		t.Errorf("expected branch query param 'feature/login', got %q", gotBranch)
	}
	if gotPR != "42" {
		t.Errorf("expected pullRequest query param '42', got %q", gotPR)
	}
	if !res.Truncated || res.Total != 12500 {
		t.Errorf("expected Truncated=true and Total=12500, got Truncated=%v Total=%d", res.Truncated, res.Total)
	}
}

func TestFetchRuleDetails_ParsesDescriptionSectionsAndLegacyDesc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/rules/show" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		key := r.URL.Query().Get("key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"rule": {
				"key": "` + key + `",
				"name": "Functions should not have too many parameters",
				"severity": "MAJOR",
				"type": "CODE_SMELL",
				"descriptionSections": [
					{"key": "root_cause", "content": "<p>Too many parameters make functions hard to read.</p>"},
					{"key": "how_to_fix", "content": "<p>Group related parameters into a struct.</p>"}
				]
			}
		}`))
	}))
	defer srv.Close()

	detail, err := fetchRuleDetails(srv.URL, "token-123", "go:S107")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Key != "go:S107" || detail.Name != "Functions should not have too many parameters" {
		t.Errorf("unexpected rule metadata: %+v", detail)
	}
	if !strings.Contains(detail.Description, "Too many parameters make functions hard to read.") ||
		!strings.Contains(detail.Description, "Group related parameters into a struct.") ||
		strings.Contains(detail.Description, "<p>") {
		t.Errorf("expected HTML-stripped description with both sections, got: %q", detail.Description)
	}
}

func TestIssue_SeverityMapping(t *testing.T) {
	issue := Issue{
		Key:       "test-key-123",
		Rule:      "java:S1234",
		Severity:  "HIGH",
		Component: "test-project:src/Main.java",
		Line:      42,
		Message:   "Test issue message",
		Status:    "OPEN",
	}

	if issue.Severity != "HIGH" {
		t.Errorf("Expected Severity HIGH, got %s", issue.Severity)
	}
	if issue.Line != 42 {
		t.Errorf("Expected Line 42, got %d", issue.Line)
	}
}

func TestIssue_Impacts(t *testing.T) {
	issue := Issue{
		Key:       "test-key-456",
		Rule:      "java:S5678",
		Severity:  "MEDIUM",
		Component: "test-project:src/Test.java",
		Impacts: []struct {
			SoftwareQuality string `json:"softwareQuality"`
			Severity        string `json:"severity"`
		}{
			{SoftwareQuality: "MAINTAINABILITY", Severity: "MEDIUM"},
			{SoftwareQuality: "RELIABILITY", Severity: "LOW"},
		},
	}

	if len(issue.Impacts) != 2 {
		t.Errorf("Expected 2 impacts, got %d", len(issue.Impacts))
	}

	if issue.Impacts[0].SoftwareQuality != "MAINTAINABILITY" {
		t.Errorf("Expected first impact MAINTAINABILITY, got %s", issue.Impacts[0].SoftwareQuality)
	}
}
