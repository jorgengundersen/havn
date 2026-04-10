package doctor

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormatHuman returns the default human-readable output.
func FormatHuman(r Report) string {
	var b strings.Builder

	currentSection := ""
	for _, c := range r.Checks {
		section := tierSection(c)
		if section != currentSection {
			if currentSection != "" {
				b.WriteString("\n")
			}
			b.WriteString(section)
			b.WriteString("\n")
			currentSection = section
		}

		fmt.Fprintf(&b, "  [%s]  %s\n", c.Status, c.Message)
		if c.Recommendation != "" {
			fmt.Fprintf(&b, "         -> %s\n", c.Recommendation)
		}
	}

	b.WriteString("\n")
	b.WriteString(formatSummaryLine(r.Summary))
	b.WriteString("\n")

	return b.String()
}

// FormatVerbose returns verbose output with detail lines.
func FormatVerbose(r Report) string {
	var b strings.Builder

	currentSection := ""
	for _, c := range r.Checks {
		section := tierSection(c)
		if section != currentSection {
			if currentSection != "" {
				b.WriteString("\n")
			}
			b.WriteString(section)
			b.WriteString("\n")
			currentSection = section
		}

		fmt.Fprintf(&b, "  [%s]  %s\n", c.Status, c.Message)
		if c.Detail != "" {
			fmt.Fprintf(&b, "          %s\n", c.Detail)
		}
		if c.Recommendation != "" {
			fmt.Fprintf(&b, "         -> %s\n", c.Recommendation)
		}
	}

	b.WriteString("\n")
	b.WriteString(formatSummaryLine(r.Summary))
	b.WriteString("\n")

	return b.String()
}

func tierSection(c ReportCheck) string {
	if c.Tier == "host" {
		return "Host"
	}
	if c.Container != "" {
		return fmt.Sprintf("Container: %s", c.Container)
	}
	return "Container"
}

func formatSummaryLine(s Summary) string {
	warnWord := "warnings"
	if s.Warnings == 1 {
		warnWord = "warning"
	}
	errWord := "errors"
	if s.Errors == 1 {
		errWord = "error"
	}
	return fmt.Sprintf("%d %s, %d %s", s.Warnings, warnWord, s.Errors, errWord)
}

// jsonReport is the JSON output structure matching the spec schema.
type jsonReport struct {
	Status  string      `json:"status"`
	Summary jsonSummary `json:"summary"`
	Checks  []jsonCheck `json:"checks"`
}

type jsonSummary struct {
	Passed   int `json:"passed"`
	Warnings int `json:"warnings"`
	Errors   int `json:"errors"`
}

type jsonCheck struct {
	Tier           string `json:"tier"`
	Container      string `json:"container,omitempty"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	Message        string `json:"message"`
	Detail         string `json:"detail,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

// FormatJSON returns the JSON output matching the spec schema.
func FormatJSON(r Report) string {
	jr := jsonReport{
		Status: string(r.Status),
		Summary: jsonSummary{
			Passed:   r.Summary.Passed,
			Warnings: r.Summary.Warnings,
			Errors:   r.Summary.Errors,
		},
	}

	for _, c := range r.Checks {
		jr.Checks = append(jr.Checks, jsonCheck{
			Tier:           c.Tier,
			Container:      c.Container,
			Name:           c.Name,
			Status:         string(c.Status),
			Message:        c.Message,
			Detail:         c.Detail,
			Recommendation: c.Recommendation,
		})
	}

	data, _ := json.MarshalIndent(jr, "", "  ")
	return string(data)
}
