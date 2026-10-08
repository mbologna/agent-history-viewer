package main

import "strings"

// noiseExact is the set of short acknowledgement-only prompts that carry no
// information about what was actually worked on.
var noiseExact = map[string]bool{
	"continue": true, "yes": true, "no": true, "y": true, "n": true,
	"ok": true, "retry": true, "sure": true, "done": true, "go": true,
	"yep": true, "nope": true, "yup": true, "nah": true, "k": true,
	"proceed": true, "next": true, "stop": true, "wait": true,
}

var noisePrefixes = []string{
	"/rate-limit", "/clear", "/help", "/cost", "/quit", "/exit", "/reset", "[Pasted",
}

// isNoise reports whether a prompt is too short or generic to be worth
// surfacing in a grouped history report (acknowledgements, slash commands,
// pasted-content placeholders).
func isNoise(display string) bool {
	s := strings.TrimSpace(display)
	if len(s) <= 3 {
		return true
	}
	if noiseExact[strings.ToLower(s)] {
		return true
	}
	for _, p := range noisePrefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
