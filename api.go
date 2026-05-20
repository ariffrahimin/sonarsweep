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
	Key          string `json:"key"`
	Rule         string `json:"rule"`
	Severity     string `json:"severity"`
	Component    string `json:"component"`
	Line         int    `json:"line"`
	Message      string `json:"message"`
	Status       string `json:"status"`
	Effort       string `json:"effort"`
	Author       string `json:"author"`
	CreationDate string `json:"creationDate"`
	Impacts      []struct {
		SoftwareQuality string `json:"softwareQuality"`
		Severity        string `json:"severity"`
	} `json:"impacts"`
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
	var allIssues []Issue
	client := &http.Client{Timeout: 15 * time.Second}

	p := 1
	ps := 500

	for {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/issues/search", SONAR_URL), nil)
		if err != nil {
			return nil, err
		}

		q := req.URL.Query()
		q.Add("componentKeys", projectKey)
		q.Add("statuses", "OPEN,CONFIRMED")
	if len(severities) == 0 {
		severities = []string{"BLOCKER", "HIGH", "MEDIUM", "LOW"}
	}
	q.Add("impactSeverities", strings.Join(severities, ","))
		q.Add("impactSoftwareQualities", strings.Join(softwareQualities, ","))
		if isNewCodePeriod {
			q.Add("inNewCodePeriod", "true")
		}
		q.Add("p", strconv.Itoa(p))
		q.Add("ps", strconv.Itoa(ps))
		req.URL.RawQuery = q.Encode()

		req.SetBasicAuth(token, "")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("network or connection issue: %w", err)
		}
		defer resp.Body.Close()

		var data Response
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		if resp.StatusCode != 200 {
			errMsg := resp.Status
			if len(data.Errors) > 0 {
				errMsg = data.Errors[0].Msg
			}
			if resp.StatusCode == 401 {
				errMsg += "\nTip: This might be an invalid token or lack of permissions."
			}
			return nil, fmt.Errorf("failed to fetch issues (Status Code: %d)\nDetails: %s", resp.StatusCode, errMsg)
		}

		allIssues = append(allIssues, data.Issues...)

		if len(data.Issues) == 0 || len(allIssues) >= data.Paging.Total {
			break
		}
		p++
	}

	// Map modern impacts to the severity field, overriding legacy severities
	for i := range allIssues {
		impactSeverity := "LOW" // Fallback
		for _, impact := range allIssues[i].Impacts {
			for _, sq := range softwareQualities {
				if impact.SoftwareQuality == sq {
					impactSeverity = impact.Severity
					goto FoundImpact
				}
			}
		}
	FoundImpact:
		allIssues[i].Severity = impactSeverity
	}

	return allIssues, nil
}
