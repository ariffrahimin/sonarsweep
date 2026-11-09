package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var SONAR_URL string

type Issue struct {
	Key             string `json:"key"`
	Rule            string `json:"rule"`
	Severity        string `json:"severity"`
	SoftwareQuality string `json:"-"`
	Component       string `json:"component"`
	Line            int    `json:"line"`
	Message         string `json:"message"`
	Status          string `json:"status"`
	Effort          string `json:"effort"`
	Author          string `json:"author"`
	CreationDate    string `json:"creationDate"`
	Impacts         []struct {
		SoftwareQuality string `json:"softwareQuality"`
		Severity        string `json:"severity"`
	} `json:"impacts"`
}

type FetchOptions struct {
	BaseURL           string
	ProjectKey        string
	Token             string
	SoftwareQualities []string
	Severities        []string
	IsNewCodePeriod   bool
	Branch            string
	PullRequest       string
}

type FetchResult struct {
	Issues    []Issue
	Total     int
	Truncated bool
}

type RuleDetail struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Severity    string `json:"severity,omitempty"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description"`
}

func stripHTML(input string) string {
	var b strings.Builder
	inTag := false
	for _, r := range input {
		switch r {
		case '<':
			inTag = true
		case '>':
			if inTag {
				inTag = false
			}
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	replacer := strings.NewReplacer(
		"&lt;", "<",
		"&gt;", ">",
		"&amp;", "&",
		"&quot;", `"`,
		"&#39;", "'",
		"&nbsp;", " ",
	)
	return strings.TrimSpace(replacer.Replace(b.String()))
}

func fetchRuleDetails(baseURL, token, ruleKey string) (RuleDetail, error) {
	if baseURL == "" {
		baseURL = SONAR_URL
	}
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/rules/show", strings.TrimRight(baseURL, "/")), nil)
	if err != nil {
		return RuleDetail{}, err
	}
	q := req.URL.Query()
	q.Add("key", ruleKey)
	req.URL.RawQuery = q.Encode()
	req.SetBasicAuth(token, "")

	resp, err := client.Do(req)
	if err != nil {
		return RuleDetail{}, fmt.Errorf("network or connection issue: %w", err)
	}
	defer resp.Body.Close()

	var payload struct {
		Rule struct {
			Key                 string `json:"key"`
			Name                string `json:"name"`
			Severity            string `json:"severity"`
			Type                string `json:"type"`
			MdDesc              string `json:"mdDesc"`
			HtmlDesc            string `json:"htmlDesc"`
			DescriptionSections []struct {
				Key     string `json:"key"`
				Content string `json:"content"`
			} `json:"descriptionSections"`
		} `json:"rule"`
		Errors []struct {
			Msg string `json:"msg"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return RuleDetail{}, fmt.Errorf("failed to decode rule response: %w", err)
	}

	if resp.StatusCode != 200 {
		errMsg := resp.Status
		if len(payload.Errors) > 0 {
			errMsg = payload.Errors[0].Msg
		}
		return RuleDetail{}, fmt.Errorf("failed to fetch rule %q (Status Code: %d): %s", ruleKey, resp.StatusCode, errMsg)
	}

	var sections []string
	for _, sec := range payload.Rule.DescriptionSections {
		cleaned := stripHTML(sec.Content)
		if cleaned != "" {
			sections = append(sections, fmt.Sprintf("[%s]\n%s", sec.Key, cleaned))
		}
	}
	desc := strings.Join(sections, "\n\n")
	if desc == "" {
		if payload.Rule.MdDesc != "" {
			desc = stripHTML(payload.Rule.MdDesc)
		} else {
			desc = stripHTML(payload.Rule.HtmlDesc)
		}
	}

	return RuleDetail{
		Key:         payload.Rule.Key,
		Name:        payload.Rule.Name,
		Severity:    payload.Rule.Severity,
		Type:        payload.Rule.Type,
		Description: desc,
	}, nil
}

type Paging struct {
	Total int `json:"total"`
}

type Response struct {
	Issues []Issue `json:"issues"`
	Paging Paging  `json:"paging"`
	Errors []struct {
		Msg string `json:"msg"`
	} `json:"errors"`
}

func isValidURL(s string) bool {
	s = strings.TrimRight(s, "/")
	if s == "" {
		return false
	}

	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.Host == "" {
		return false
	}
	return true
}

func fetchIssues(projectKey, token string, softwareQualities, severities []string, isNewCodePeriod bool) ([]Issue, error) {
	res, err := fetchIssuesWithOptions(FetchOptions{
		BaseURL:           SONAR_URL,
		ProjectKey:        projectKey,
		Token:             token,
		SoftwareQualities: softwareQualities,
		Severities:        severities,
		IsNewCodePeriod:   isNewCodePeriod,
	})
	if err != nil {
		return nil, err
	}
	return res.Issues, nil
}

func fetchIssuesWithOptions(opts FetchOptions) (FetchResult, error) {
	var allIssues []Issue
	var total int
	client := &http.Client{Timeout: 15 * time.Second}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = SONAR_URL
	}

	severities := opts.Severities
	if len(severities) == 0 {
		severities = []string{"BLOCKER", "HIGH", "MEDIUM", "LOW"}
	}

	p := 1
	ps := 500

	for {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/issues/search", baseURL), nil)
		if err != nil {
			return FetchResult{}, err
		}

		q := req.URL.Query()
		q.Add("componentKeys", opts.ProjectKey)
		q.Add("statuses", "OPEN,CONFIRMED")
		q.Add("impactSeverities", strings.Join(severities, ","))
		q.Add("impactSoftwareQualities", strings.Join(opts.SoftwareQualities, ","))
		if opts.IsNewCodePeriod {
			q.Add("inNewCodePeriod", "true")
		}
		if opts.Branch != "" {
			q.Add("branch", opts.Branch)
		}
		if opts.PullRequest != "" {
			q.Add("pullRequest", opts.PullRequest)
		}
		q.Add("p", strconv.Itoa(p))
		q.Add("ps", strconv.Itoa(ps))
		req.URL.RawQuery = q.Encode()

		req.SetBasicAuth(opts.Token, "")

		resp, err := client.Do(req)
		if err != nil {
			return FetchResult{}, fmt.Errorf("network or connection issue: %w", err)
		}

		var data Response
		decodeErr := json.NewDecoder(resp.Body).Decode(&data)
		resp.Body.Close()
		if decodeErr != nil {
			return FetchResult{}, fmt.Errorf("failed to decode response: %w", decodeErr)
		}

		if resp.StatusCode != 200 {
			errMsg := resp.Status
			if len(data.Errors) > 0 {
				errMsg = data.Errors[0].Msg
			}
			if resp.StatusCode == 401 {
				errMsg += "\nTip: This might be an invalid token or lack of permissions."
			}
			return FetchResult{}, fmt.Errorf("failed to fetch issues (Status Code: %d)\nDetails: %s", resp.StatusCode, errMsg)
		}

		total = data.Paging.Total
		allIssues = append(allIssues, data.Issues...)

		if len(data.Issues) == 0 || len(allIssues) >= data.Paging.Total || (p+1)*ps > 10000 {
			break
		}
		p++
	}

	severityRank := map[string]int{
		"INFO":    1,
		"LOW":     2,
		"MEDIUM":  3,
		"HIGH":    4,
		"BLOCKER": 5,
	}

	// Map modern impacts to the severity and softwareQuality fields, selecting the highest matching impact
	for i := range allIssues {
		bestSeverity := ""
		bestQuality := ""
		bestRank := 0
		for _, impact := range allIssues[i].Impacts {
			for _, sq := range opts.SoftwareQualities {
				if impact.SoftwareQuality == sq {
					if rank := severityRank[impact.Severity]; rank > bestRank {
						bestRank = rank
						bestSeverity = impact.Severity
						bestQuality = impact.SoftwareQuality
					}
				}
			}
		}
		if bestSeverity != "" {
			allIssues[i].Severity = bestSeverity
			allIssues[i].SoftwareQuality = bestQuality
		} else if allIssues[i].Severity == "" {
			allIssues[i].Severity = "LOW"
		}
	}

	return FetchResult{
		Issues:    allIssues,
		Total:     total,
		Truncated: total > 10000,
	}, nil
}
