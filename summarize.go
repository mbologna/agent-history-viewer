package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// summaryCachePath is where AI-generated session/project summaries are
// cached so re-running -html doesn't re-spend tokens on unchanged sessions.
func summaryCachePath(home string) string {
	return filepath.Join(home, ".config", "agent-history-viewer", "summary_cache.json")
}

type summaryCache struct {
	path string
	data map[string]string
}

func loadSummaryCache(home string) *summaryCache {
	path := summaryCachePath(home)
	c := &summaryCache{path: path, data: map[string]string{}}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &c.data)
	}
	return c
}

func (c *summaryCache) save() {
	_ = os.MkdirAll(filepath.Dir(c.path), 0o755)
	data, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(c.path, data, 0o644)
}

// summarizeWithClaude shells out to the `claude` CLI in print mode to turn a
// prompt into a short summary. It is intentionally synchronous and used only
// on demand (TUI keypress or -html's local server), never during bulk scans.
func summarizeWithClaude(prompt string) string {
	cmd := exec.Command("claude", "-p", prompt)
	out, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return "(claude CLI failed: " + strings.TrimSpace(string(out)) + ")"
		}
		return "(claude CLI not found — install it to enable summaries)"
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "(no output)"
	}
	return s
}

// summarizeSession produces a 1-2 sentence summary of a single session's
// user prompts, describing what was worked on.
func summarizeSession(s Session, cache *summaryCache) string {
	key := "session:" + s.ID
	if v, ok := cache.data[key]; ok {
		return v
	}
	var lines []string
	for _, m := range s.Messages {
		if m.Role != "user" || isNoise(m.Content) {
			continue
		}
		lines = append(lines, "- "+firstLine(m.Content))
	}
	if len(lines) == 0 {
		return "(nothing but noise prompts)"
	}
	summary := summarizeWithClaude(
		"In 1-2 sentences, summarize what you were working on based on these prompts. " +
			"Be specific about the task. Do not say 'the user' — write as if describing your own session:\n\n" +
			strings.Join(lines, "\n"))
	cache.data[key] = summary
	return summary
}

// summarizeProject produces a 2-3 sentence summary of a project group across
// all of its sessions, sampling up to 5 prompts per session (60 total).
func summarizeProject(groupName string, sessions []Session, cache *summaryCache) string {
	key := "project:" + groupName
	if v, ok := cache.data[key]; ok {
		return v
	}
	sorted := append([]Session(nil), sessions...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].StartTime.Before(sorted[j].StartTime) })

	var lines []string
	for _, s := range sorted {
		n := 0
		for _, m := range s.Messages {
			if m.Role != "user" || isNoise(m.Content) {
				continue
			}
			lines = append(lines, "- "+firstLine(m.Content))
			n++
			if n >= 5 {
				break
			}
		}
		if len(lines) >= 60 {
			break
		}
	}
	if len(lines) == 0 {
		return "(nothing but noise prompts)"
	}
	if len(lines) > 60 {
		lines = lines[:60]
	}
	summary := summarizeWithClaude(
		"Based on these prompts from multiple sessions, write 2-3 sentences describing " +
			"what the project '" + groupName + "' is about and what kinds of things were worked on. " +
			"Be concrete, not generic:\n\n" + strings.Join(lines, "\n"))
	cache.data[key] = summary
	return summary
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	const max = 300
	if len(s) > max {
		s = s[:max] + "..."
	}
	return s
}
