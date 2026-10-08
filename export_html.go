package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// buildGroups groups sessions by project (merging related projects via the
// user's groups.json) and strips noise-only prompts, ready for -html
// rendering.
func buildGroups(sessions []Session, home string, projectFilter string, resetGroups bool) map[string][]Session {
	var names []string
	seen := map[string]bool{}
	for _, s := range sessions {
		if !seen[s.Project] {
			seen[s.Project] = true
			names = append(names, s.Project)
		}
	}
	assignment := resolveGroups(home, names, resetGroups)

	out := map[string][]Session{}
	for _, s := range sessions {
		group := assignment[s.Project]
		if group == "" {
			group = s.Project
		}
		if projectFilter != "" && !strings.Contains(strings.ToLower(group), strings.ToLower(projectFilter)) {
			continue
		}
		out[group] = append(out[group], s)
	}
	return out
}

func promptCount(sessions []Session) int {
	n := 0
	for _, s := range sessions {
		for _, m := range s.Messages {
			if m.Role == "user" && !isNoise(m.Content) {
				n++
			}
		}
	}
	return n
}

func latestTime(sessions []Session) time.Time {
	var latest time.Time
	for _, s := range sessions {
		if s.LastTime.After(latest) {
			latest = s.LastTime
		}
	}
	return latest
}

const reportCSS = `
* { box-sizing: border-box; margin: 0; padding: 0; }
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  background: #0f1117; color: #e2e8f0; padding: 2rem 1.5rem; line-height: 1.5;
  max-width: 920px; margin: 0 auto;
}
h1 { font-size: 1.4rem; font-weight: 600; margin-bottom: 0.4rem; color: #f8fafc; }
.subtitle { color: #64748b; font-size: 0.85rem; margin-bottom: 1.25rem; }
.controls { display: flex; align-items: center; gap: 1rem; margin-bottom: 1.5rem; font-size: 0.8rem; color: #64748b; }
.controls select { background: #1e2330; color: #e2e8f0; border: 1px solid #2d3748; border-radius: 4px; padding: 0.2rem 0.5rem; font-size: 0.8rem; cursor: pointer; }
.project-block { margin-bottom: 0.75rem; }
details { border-radius: 8px; overflow: hidden; }
details[data-level="project"] > summary {
  background: #1e2330; padding: 0.7rem 1rem; cursor: pointer; list-style: none;
  display: flex; align-items: center; gap: 0.6rem; font-weight: 600; font-size: 0.95rem;
  color: #93c5fd; border: 1px solid #2d3748; border-radius: 8px; user-select: none;
}
details[data-level="project"][open] > summary { border-radius: 8px 8px 0 0; }
details[data-level="project"] > summary::-webkit-details-marker { display: none; }
details[data-level="project"] > summary::before {
  content: "▶"; font-size: 0.6rem; color: #475569; transition: transform 0.15s; flex-shrink: 0;
}
details[data-level="project"][open] > summary::before { transform: rotate(90deg); }
.project-name { flex: 1; min-width: 0; display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.project-summary-text { font-size: 0.75rem; color: #94a3b8; font-style: italic; font-weight: 400; width: 100%; margin-top: 0.1rem; }
.project-meta { font-size: 0.75rem; color: #475569; white-space: nowrap; margin-left: auto; }
.summarize-btn {
  font-size: 0.65rem; background: transparent; color: #818cf8; border: 1px solid #3730a3;
  border-radius: 4px; padding: 0.15rem 0.45rem; cursor: pointer; white-space: nowrap; flex-shrink: 0;
}
.summarize-btn:hover:not(:disabled) { background: #1e1b4b; color: #a5b4fc; }
.summarize-btn:disabled { opacity: 0.3; cursor: default; }
.project-content { border: 1px solid #2d3748; border-top: none; border-radius: 0 0 8px 8px; padding: 0.6rem; background: #161b27; display: flex; flex-direction: column; gap: 0.4rem; }
details[data-level="session"] > summary {
  background: #1a2035; padding: 0.55rem 0.75rem; cursor: pointer; list-style: none;
  display: flex; align-items: flex-start; gap: 0.5rem; border-radius: 6px; border: 1px solid #2a3347; user-select: none;
}
details[data-level="session"] > summary::-webkit-details-marker { display: none; }
details[data-level="session"] > summary::before {
  content: "▶"; font-size: 0.55rem; color: #475569; margin-top: 0.35rem; transition: transform 0.15s; flex-shrink: 0;
}
details[data-level="session"][open] > summary::before { transform: rotate(90deg); }
.session-header { display: flex; flex-direction: column; gap: 0.15rem; flex: 1; min-width: 0; }
.session-top { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.session-date { font-size: 0.8rem; font-weight: 600; color: #a5b4fc; white-space: nowrap; }
.agent-tag {
  font-size: 0.6rem; text-transform: uppercase; letter-spacing: 0.03em; font-weight: 600;
  padding: 0.05rem 0.35rem; border-radius: 3px; background: #1a2840; color: #60a5fa; border: 1px solid #1e3a5f;
}
.session-title { font-size: 0.78rem; color: #94a3b8; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 28rem; }
.session-summary { font-size: 0.78rem; color: #94a3b8; font-style: italic; }
.session-count { font-size: 0.7rem; color: #475569; }
.prompt-list { list-style: none; padding: 0.4rem 0.75rem 0.4rem 1.6rem; display: flex; flex-direction: column; gap: 0.25rem; }
.prompt-item { display: flex; gap: 0.75rem; align-items: baseline; }
.prompt-time { font-size: 0.68rem; color: #475569; white-space: nowrap; flex-shrink: 0; font-variant-numeric: tabular-nums; }
.prompt-text { font-size: 0.8rem; color: #cbd5e1; word-break: break-word; }
`

func reportJS(apiBase string) string {
	return fmt.Sprintf(`
const API_BASE = %s;
const sortSel = document.getElementById('sort-select');
function sortProjects() {
  const container = document.getElementById('projects');
  const blocks = [...container.querySelectorAll('.project-block')];
  const key = sortSel.value;
  blocks.sort((a, b) => key === 'prompts'
    ? parseInt(b.dataset.prompts) - parseInt(a.dataset.prompts)
    : key === 'sessions'
    ? parseInt(b.dataset.sessions) - parseInt(a.dataset.sessions)
    : parseInt(b.dataset.latest) - parseInt(a.dataset.latest));
  blocks.forEach(b => container.appendChild(b));
}
sortSel.addEventListener('change', sortProjects);
sortProjects();

if (!API_BASE) {
  document.querySelectorAll('.summarize-btn').forEach(btn => {
    btn.disabled = true;
    btn.title = 'Run with -serve to enable live summaries';
  });
}

async function doSummarize(btn, url, onResult) {
  btn.disabled = true;
  const orig = btn.textContent;
  btn.textContent = '…';
  try {
    const r = await fetch(url);
    if (!r.ok) throw new Error(r.status);
    const data = await r.json();
    onResult(data.summary);
    btn.remove();
  } catch (e) {
    btn.textContent = orig;
    btn.disabled = false;
  }
}

document.querySelectorAll('.summarize-btn[data-session]').forEach(btn => {
  btn.addEventListener('click', e => {
    e.stopPropagation();
    const id = encodeURIComponent(btn.dataset.session);
    doSummarize(btn, API_BASE + '/api/summarize/session?id=' + id, summary => {
      const header = btn.closest('.session-header');
      let el = header.querySelector('.session-summary');
      if (!el) {
        el = document.createElement('span');
        el.className = 'session-summary';
        header.querySelector('.session-count').before(el);
      }
      el.textContent = summary;
    });
  });
});

document.querySelectorAll('.summarize-btn[data-project]').forEach(btn => {
  btn.addEventListener('click', e => {
    e.stopPropagation();
    const name = encodeURIComponent(btn.dataset.project);
    doSummarize(btn, API_BASE + '/api/summarize/project?name=' + name, summary => {
      const nameEl = btn.closest('summary').querySelector('.project-name');
      let el = nameEl.querySelector('.project-summary-text');
      if (!el) {
        el = document.createElement('span');
        el.className = 'project-summary-text';
        nameEl.appendChild(el);
      }
      el.textContent = summary;
    });
  });
});
`, apiBase)
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

// renderHTMLReport renders the full grouped project -> session -> prompt
// report. apiPort is 0 when summaries are not being served live.
func renderHTMLReport(grouped map[string][]Session, apiPort int) string {
	totalGroups := len(grouped)
	totalSessions := 0
	totalPrompts := 0
	for _, sessions := range grouped {
		totalSessions += len(sessions)
		totalPrompts += promptCount(sessions)
	}

	groupNames := make([]string, 0, len(grouped))
	for g := range grouped {
		groupNames = append(groupNames, g)
	}
	sort.Slice(groupNames, func(i, j int) bool {
		return promptCount(grouped[groupNames[i]]) > promptCount(grouped[groupNames[j]])
	})

	apiBase := "null"
	serveNote := ""
	if apiPort != 0 {
		apiBase = fmt.Sprintf("%q", fmt.Sprintf("http://localhost:%d", apiPort))
		serveNote = fmt.Sprintf(` &mdash; <span style="color:#6366f1">server on :%d</span>`, apiPort)
	}

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<title>Agent History Viewer Report</title>\n")
	b.WriteString("<style>" + reportCSS + "</style>\n</head>\n<body>\n")
	b.WriteString("<h1>Agent Session History</h1>\n")
	fmt.Fprintf(&b, `<p class="subtitle">Generated %s%s &mdash; %d projects &middot; %d sessions &middot; %d prompts</p>`+"\n",
		time.Now().Format("2006-01-02 15:04"), serveNote, totalGroups, totalSessions, totalPrompts)
	b.WriteString(`<div class="controls"><label>Sort by: <select id="sort-select">` +
		`<option value="prompts">Prompts</option><option value="recent">Most recent</option>` +
		`<option value="sessions">Sessions</option></select></label></div>` + "\n")
	b.WriteString(`<div id="projects">` + "\n")

	for _, group := range groupNames {
		sessions := grouped[group]
		pCount := promptCount(sessions)
		sCount := len(sessions)
		latest := latestTime(sessions)

		fmt.Fprintf(&b, `<div class="project-block" data-prompts="%d" data-latest="%d" data-sessions="%d">`+"\n",
			pCount, latest.Unix(), sCount)
		b.WriteString(`<details data-level="project"><summary>`)
		fmt.Fprintf(&b, `<div class="project-name">%s</div>`, escapeHTML(group))
		fmt.Fprintf(&b, `<button class="summarize-btn" data-project="%s">Summarize</button>`, escapeHTML(group))
		fmt.Fprintf(&b, `<span class="project-meta">%d session%s &middot; %d prompts &middot; last %s</span>`,
			sCount, plural(sCount), pCount, latest.Format("2006-01-02 15:04"))
		b.WriteString(`</summary><div class="project-content">` + "\n")

		sorted := append([]Session(nil), sessions...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].StartTime.Before(sorted[j].StartTime) })

		for _, s := range sorted {
			var prompts []Message
			for _, m := range s.Messages {
				if m.Role == "user" && !isNoise(m.Content) {
					prompts = append(prompts, m)
				}
			}
			if len(prompts) == 0 {
				continue
			}
			dateRange := s.StartTime.Format("2006-01-02 15:04")
			if s.StartTime.Format("2006-01-02") != s.LastTime.Format("2006-01-02") {
				dateRange += " -> " + s.LastTime.Format("2006-01-02 15:04")
			}

			b.WriteString(`<details data-level="session"><summary><div class="session-header">`)
			b.WriteString(`<div class="session-top">`)
			fmt.Fprintf(&b, `<span class="session-date">%s</span>`, escapeHTML(dateRange))
			fmt.Fprintf(&b, `<span class="agent-tag">%s</span>`, escapeHTML(s.Agent))
			if s.Title != "" {
				fmt.Fprintf(&b, `<span class="session-title">%s</span>`, escapeHTML(s.Title))
			}
			fmt.Fprintf(&b, `<button class="summarize-btn" data-session="%s">Summarize</button>`, escapeHTML(s.ID))
			b.WriteString(`</div>`)
			fmt.Fprintf(&b, `<span class="session-count">%d prompt%s</span>`, len(prompts), plural(len(prompts)))
			b.WriteString(`</div></summary><ul class="prompt-list">` + "\n")

			for _, m := range prompts {
				cls := "prompt-text"
				if strings.HasPrefix(strings.TrimSpace(m.Content), "/") {
					cls = "prompt-text cmd"
				}
				fmt.Fprintf(&b, `<li class="prompt-item"><span class="prompt-time">%s</span><span class="%s">%s</span></li>`+"\n",
					m.Time.Format("15:04"), cls, escapeHTML(firstLine(m.Content)))
			}
			b.WriteString(`</ul></details>` + "\n")
		}

		b.WriteString(`</div></details></div>` + "\n")
	}

	b.WriteString(`</div>` + "\n")
	fmt.Fprintf(&b, "<script>%s</script>\n", reportJS(apiBase))
	b.WriteString("</body></html>\n")
	return b.String()
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// findFreePort asks the OS for an unused TCP port on localhost.
func findFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// serveReport starts a local HTTP server that serves the static report and
// answers on-demand summarize requests, backed by the shared summary cache.
func serveReport(port int, grouped map[string][]Session, cache *summaryCache, htmlBytes []byte) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(htmlBytes)
	})
	mux.HandleFunc("/api/summarize/session", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		var found *Session
		for _, sessions := range grouped {
			for i := range sessions {
				if sessions[i].ID == id {
					found = &sessions[i]
					break
				}
			}
			if found != nil {
				break
			}
		}
		summary := "(session not found)"
		if found != nil {
			summary = summarizeSession(*found, cache)
			cache.save()
		}
		writeJSON(w, map[string]string{"summary": summary})
	})
	mux.HandleFunc("/api/summarize/project", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		summary := "(project not found)"
		if sessions, ok := grouped[name]; ok {
			summary = summarizeProject(name, sessions, cache)
			cache.save()
		}
		writeJSON(w, map[string]string{"summary": summary})
	})

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	fmt.Printf("Server running at http://localhost:%d/\n", port)
	fmt.Println("Press Ctrl+C to stop.")
	return http.ListenAndServe(addr, mux)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// runHTMLExport is the entry point for `-html`: it discovers sessions across
// every supported agent, groups them by (user-editable) project grouping,
// writes the static report, and optionally serves it with live AI summaries.
func runHTMLExport(home, output, projectFilter, agentFilter string, resetGroups, serve bool, port int) {
	sessions := findSessions(home)
	if agentFilter != "" {
		sessions = filterSessionsCLI(sessions, agentFilter, "")
	}

	grouped := buildGroups(sessions, home, projectFilter, resetGroups)
	if len(grouped) == 0 {
		fmt.Println("No sessions found matching the given filters.")
		return
	}

	apiPort := 0
	if serve {
		if port != 0 {
			apiPort = port
		} else {
			p, err := findFreePort()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error finding a free port: %v\n", err)
				os.Exit(1)
			}
			apiPort = p
		}
	}

	html := renderHTMLReport(grouped, apiPort)

	if output == "" {
		output = filepath.Join(home, ".config", "agent-history-viewer", "report.html")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(output, []byte(html), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Report written to: %s\n", output)

	if !serve {
		fmt.Printf("Open with: open %s\n", output)
		fmt.Println("Tip: run with -serve to enable Summarize buttons")
		return
	}

	cache := loadSummaryCache(home)
	if err := serveReport(apiPort, grouped, cache, []byte(html)); err != nil {
		fmt.Fprintf(os.Stderr, "Error running server: %v\n", err)
		os.Exit(1)
	}
}
