package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// groupsConfigPath returns the path to the user-editable file that maps a
// project name to the group it should be displayed under in the -html
// report. Related projects (e.g. "myapp-frontend" and "myapp-backend") are
// auto-grouped under their common prefix on first run; the file can then be
// hand-edited to merge or split groups, and is only ever added to, not
// overwritten, on subsequent runs.
func groupsConfigPath(home string) string {
	return filepath.Join(home, ".config", "agent-history-viewer", "groups.json")
}

const groupsConfigInfoKey = "_info"
const groupsConfigInfoText = "Maps project name -> group display name. Edit to merge related " +
	"projects, then re-run. Delete this file (or pass -reset-groups) to regenerate from scratch."

func loadGroupsConfig(path string) map[string]string {
	out := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return out
	}
	for k, v := range raw {
		if strings.HasPrefix(k, "_") {
			continue
		}
		out[k] = v
	}
	return out
}

func saveGroupsConfig(path string, assignment map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out := map[string]string{groupsConfigInfoKey: groupsConfigInfoText}
	for k, v := range assignment {
		out[k] = v
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// autoGroup assigns each project name to a group, merging a name into an
// existing shorter group when it extends that group's name with a
// "-", "_" or "." separator (e.g. "myapp-frontend" -> group "myapp", if
// "myapp" is itself one of the names). Names that don't extend anything
// become their own group.
func autoGroup(names []string) map[string]string {
	seen := map[string]bool{}
	var uniq []string
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			uniq = append(uniq, n)
		}
	}
	sort.Slice(uniq, func(i, j int) bool {
		if len(uniq[i]) != len(uniq[j]) {
			return len(uniq[i]) < len(uniq[j])
		}
		return uniq[i] < uniq[j]
	})

	groups := map[string][]string{}
	assignment := map[string]string{}

	for _, name := range uniq {
		var matched string
		// Prefer the longest existing group that name extends.
		var candidates []string
		for g := range groups {
			candidates = append(candidates, g)
		}
		sort.Slice(candidates, func(i, j int) bool { return len(candidates[i]) > len(candidates[j]) })
		for _, g := range candidates {
			if name == g {
				matched = g
				break
			}
			if strings.HasPrefix(name, g+"-") || strings.HasPrefix(name, g+"_") || strings.HasPrefix(name, g+".") {
				matched = g
				break
			}
		}
		if matched != "" {
			assignment[name] = matched
			groups[matched] = append(groups[matched], name)
		} else {
			groups[name] = []string{name}
			assignment[name] = name
		}
	}
	return assignment
}

// resolveGroups loads the user's groups config (creating it on first run),
// merges in any newly-seen project names, and returns the full project ->
// group assignment to use for this run.
func resolveGroups(home string, names []string, forceReset bool) map[string]string {
	path := groupsConfigPath(home)
	auto := autoGroup(names)

	if forceReset {
		_ = saveGroupsConfig(path, auto)
		return auto
	}
	if _, err := os.Stat(path); err != nil {
		_ = saveGroupsConfig(path, auto)
		return auto
	}

	user := loadGroupsConfig(path)
	merged := map[string]string{}
	for k, v := range auto {
		merged[k] = v
	}
	newKeys := map[string]string{}
	for k, v := range auto {
		if _, ok := user[k]; !ok {
			newKeys[k] = v
		}
	}
	for k, v := range user {
		if _, ok := auto[k]; ok {
			merged[k] = v
		}
	}
	if len(newKeys) > 0 {
		combined := map[string]string{}
		for k, v := range user {
			combined[k] = v
		}
		for k, v := range newKeys {
			combined[k] = v
		}
		_ = saveGroupsConfig(path, combined)
	}
	return merged
}
