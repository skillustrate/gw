package vendorcheck

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// SkillFrontmatter represents parsed metadata from SKILL.md YAML header.
type SkillFrontmatter struct {
	Name                   string
	Description            string
	DisableModelInvocation bool
}

// LintSkill parses the frontmatter of the specified SKILL.md file and verifies compliance with §19:
// 1. Frontmatter exists and is properly bounded by '---'.
// 2. DisableModelInvocation is false (must be model-invoked).
// 3. Description is non-empty and under 400 characters.
// 4. Description mentions all 5 triggers: commit, push, sync, pull request/pr, merge.
func LintSkill(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open skill file %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var frontmatterLines []string
	dashCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			dashCount++
			if dashCount == 2 {
				break
			}
			continue
		}
		if dashCount == 1 {
			frontmatterLines = append(frontmatterLines, line)
		}
	}

	if dashCount < 2 {
		return []string{"SKILL.md is missing valid YAML frontmatter bounded by '---'"}, nil
	}

	fm := SkillFrontmatter{}
	for _, line := range frontmatterLines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])

		switch k {
		case "name":
			fm.Name = v
		case "description":
			fm.Description = v
		case "disable-model-invocation":
			b, _ := strconv.ParseBool(v)
			fm.DisableModelInvocation = b
		}
	}

	var violations []string

	if fm.DisableModelInvocation {
		violations = append(violations, "disable-model-invocation must be false (skill must be model-invoked, §19)")
	}

	if len(fm.Description) == 0 {
		violations = append(violations, "description must not be empty")
	} else if len(fm.Description) > 400 {
		violations = append(violations, fmt.Sprintf("description exceeds 400 chars (%d chars)", len(fm.Description)))
	}

	descLower := strings.ToLower(fm.Description)

	// Check 5 required triggers
	if !strings.Contains(descLower, "commit") {
		violations = append(violations, "description missing 'commit' trigger")
	}
	if !strings.Contains(descLower, "push") {
		violations = append(violations, "description missing 'push' trigger")
	}
	if !strings.Contains(descLower, "sync") {
		violations = append(violations, "description missing 'sync' trigger")
	}
	if !strings.Contains(descLower, "pull request") && !strings.Contains(descLower, "pr") {
		violations = append(violations, "description missing 'pull request' or 'pr' trigger")
	}
	if !strings.Contains(descLower, "merg") {
		violations = append(violations, "description missing 'merge' trigger")
	}

	return violations, nil
}