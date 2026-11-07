package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDeprecatedSkills(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bmad-catalog-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test 1: Empty path returns empty map
	emptyMap, err := loadDeprecatedSkills("")
	if err != nil {
		t.Fatalf("Expected nil error for empty catalog path, got: %v", err)
	}
	if len(emptyMap) != 0 {
		t.Fatalf("Expected empty map for empty catalog path, got: %v", emptyMap)
	}

	// Test 2: Valid catalog file parsing
	catalogContent := `[
		{"name": "skill-a", "isDeprecated": false, "apf_supported": true, "requires_sandbox": false},
		{"name": "skill-b", "isDeprecated": true, "apf_supported": true, "requires_sandbox": false},
		{"name": "skill-c", "isDeprecated": true, "apf_supported": false, "requires_sandbox": false},
		{"name": "skill-d", "isDeprecated": false, "apf_supported": false, "requires_sandbox": false},
		{"name": "skill-e", "isDeprecated": false, "apf_supported": true, "requires_sandbox": true}
	]`
	catalogPath := filepath.Join(tempDir, "test_catalog.json")
	if err := os.WriteFile(catalogPath, []byte(catalogContent), 0644); err != nil {
		t.Fatalf("Failed to write catalog file: %v", err)
	}

	deprecatedMap, err := loadDeprecatedSkills(catalogPath)
	if err != nil {
		t.Fatalf("loadDeprecatedSkills failed: %v", err)
	}
	if len(deprecatedMap) != 3 {
		t.Errorf("Expected 3 excluded skills (skill-b, skill-c, skill-d), got %d", len(deprecatedMap))
	}
	if !deprecatedMap["skill-b"] {
		t.Errorf("Expected skill-b to be marked excluded (deprecated)")
	}
	if !deprecatedMap["skill-c"] {
		t.Errorf("Expected skill-c to be marked excluded (deprecated + apf unsupported)")
	}
	if !deprecatedMap["skill-d"] {
		t.Errorf("Expected skill-d to be marked excluded (apf unsupported)")
	}
	if deprecatedMap["skill-a"] {
		t.Errorf("Expected skill-a to NOT be marked excluded")
	}
	if deprecatedMap["skill-e"] {
		t.Errorf("Expected skill-e to NOT be marked excluded")
	}

	// Test 3: Non-existent file returns error
	_, err = loadDeprecatedSkills(filepath.Join(tempDir, "non_existent.json"))
	if err == nil {
		t.Errorf("Expected error for non-existent catalog file")
	}

	// Test 4: Invalid JSON returns error
	invalidJsonPath := filepath.Join(tempDir, "invalid.json")
	if err := os.WriteFile(invalidJsonPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid json file: %v", err)
	}
	_, err = loadDeprecatedSkills(invalidJsonPath)
	if err == nil {
		t.Errorf("Expected error for invalid JSON catalog file")
	}
}

func TestCopyDirAndPreserveContent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bmad-copy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	srcDir := filepath.Join(tempDir, "src")
	dstDir := filepath.Join(tempDir, "dst")

	if err := os.MkdirAll(filepath.Join(srcDir, "sub"), 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}

	rawSkillContent := `---
name: sample-skill
---
# Full Markdown Header
### Step 1: Original Step 1
# Comment in markdown or toml
Do not change anything here.`

	rawTomlContent := `# Top level comment
[agent]
name = "Sample" # Inline comment
# Another comment
`

	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte(rawSkillContent), 0644); err != nil {
		t.Fatalf("Failed to write test SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "sub", "customize.toml"), []byte(rawTomlContent), 0644); err != nil {
		t.Fatalf("Failed to write test customize.toml: %v", err)
	}

	if err := copyDir(srcDir, dstDir); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	// Verify SKILL.md content is strictly identical
	copiedSkillBytes, err := os.ReadFile(filepath.Join(dstDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("Failed to read copied SKILL.md: %v", err)
	}
	if string(copiedSkillBytes) != rawSkillContent {
		t.Errorf("Expected copied SKILL.md to be byte-for-byte identical.\nExpected:\n%s\nGot:\n%s", rawSkillContent, string(copiedSkillBytes))
	}

	// Verify customize.toml content is strictly identical (comments not pruned)
	copiedTomlBytes, err := os.ReadFile(filepath.Join(dstDir, "sub", "customize.toml"))
	if err != nil {
		t.Fatalf("Failed to read copied customize.toml: %v", err)
	}
	if string(copiedTomlBytes) != rawTomlContent {
		t.Errorf("Expected copied customize.toml to be byte-for-byte identical.\nExpected:\n%s\nGot:\n%s", rawTomlContent, string(copiedTomlBytes))
	}
}
