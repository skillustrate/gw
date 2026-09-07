package vendorcheck

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillLint_RealSkillFile(t *testing.T) {
	skillPath := filepath.Join(findRepoRoot(t), "skill", "git-workflow-engine", "SKILL.md")
	violations, err := LintSkill(skillPath)
	if err != nil {
		t.Fatalf("unexpected error linting real skill file: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("real SKILL.md failed lint check:\n%v", violations)
	}
}

func TestSkillLint_NegativeCases(t *testing.T) {
	tempDir := t.TempDir()

	// Case 1: disable-model-invocation: true
	c1Path := filepath.Join(tempDir, "skill1.md")
	_ = os.WriteFile(c1Path, []byte("---\nname: test\ndisable-model-invocation: true\ndescription: commit push sync pr merge\n---\n"), 0644)
	v1, err := LintSkill(c1Path)
	if err != nil || len(v1) == 0 {
		t.Errorf("expected violation for disable-model-invocation=true, got %v", v1)
	}

	// Case 2: Missing 'merge' trigger
	c2Path := filepath.Join(tempDir, "skill2.md")
	_ = os.WriteFile(c2Path, []byte("---\nname: test\ndescription: commit push sync pr\n---\n"), 0644)
	v2, err := LintSkill(c2Path)
	if err != nil || len(v2) == 0 {
		t.Errorf("expected violation for missing merge trigger, got %v", v2)
	}

	// Case 3: Over 400 chars
	c3Path := filepath.Join(tempDir, "skill3.md")
	longDesc := "commit push sync pr merge " + string(make([]byte, 400))
	_ = os.WriteFile(c3Path, []byte("---\nname: test\ndescription: "+longDesc+"\n---\n"), 0644)
	v3, err := LintSkill(c3Path)
	if err != nil || len(v3) == 0 {
		t.Errorf("expected violation for description > 400 chars, got %v", v3)
	}
}