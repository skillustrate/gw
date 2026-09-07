package vendorcheck

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

var forbiddenVendorKeywords = []string{
	"claude",
	"anthropic",
	"openai",
	"codex",
	"mcp",
	"gemini",
}

// CheckNoVendorImports scans all Go files under root and returns any forbidden vendor import paths.
func CheckNoVendorImports(root string) ([]string, error) {
	var violations []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "vendor" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		inImportBlock := false

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			if strings.HasPrefix(line, "import (") {
				inImportBlock = true
				continue
			}
			if inImportBlock && line == ")" {
				inImportBlock = false
				continue
			}

			var importPath string
			if inImportBlock {
				importPath = strings.Trim(line, "\"`\t ")
			} else if strings.HasPrefix(line, "import ") {
				importPath = strings.TrimPrefix(line, "import ")
				importPath = strings.Trim(importPath, "\"`\t ")
			}

			if importPath != "" {
				lower := strings.ToLower(importPath)
				for _, kw := range forbiddenVendorKeywords {
					if strings.Contains(lower, kw) {
						violations = append(violations, path+": imports "+importPath)
						break
					}
				}
			}
		}

		return scanner.Err()
	})

	return violations, err
}