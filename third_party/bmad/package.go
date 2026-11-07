package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

type CatalogSkill struct {
	Name            string `json:"name"`
	IsDeprecated    bool   `json:"isDeprecated"`
	ApfSupported    *bool  `json:"apf_supported"`
	RequiresSandbox *bool  `json:"requires_sandbox,omitempty"`
}

func loadExcludedSkills(catalogPath string) (map[string]bool, error) {
	excluded := make(map[string]bool)
	if catalogPath == "" {
		return excluded, nil
	}

	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog file %s: %w", catalogPath, err)
	}

	var skills []CatalogSkill
	if err := json.Unmarshal(data, &skills); err != nil {
		return nil, fmt.Errorf("failed to parse catalog JSON from %s: %w", catalogPath, err)
	}

	for _, skill := range skills {
		if skill.IsDeprecated {
			excluded[skill.Name] = true
		}
		if skill.ApfSupported != nil && !*skill.ApfSupported {
			excluded[skill.Name] = true
		}
	}
	return excluded, nil
}

func loadDeprecatedSkills(catalogPath string) (map[string]bool, error) {
	return loadExcludedSkills(catalogPath)
}

func main() {
	installDirFlag := flag.String("install-dir", "", "installation directory")
	catalogFileFlag := flag.String("catalog-file", "", "path to catalog JSON file")

	flag.Parse()

	var installDir string

	if *installDirFlag != "" {
		installDir = *installDirFlag
	} else {
		// Fallback to positional arguments
		args := flag.Args()
		if len(args) == 1 {
			installDir = args[0]
		} else {
			fmt.Fprintf(os.Stderr, "Usage:\n")
			fmt.Fprintf(os.Stderr, "  Using flags:\n")
			fmt.Fprintf(os.Stderr, "    go run package.go --install-dir <path> [--catalog-file <path>]\n")
			fmt.Fprintf(os.Stderr, "  Using positional arguments:\n")
			fmt.Fprintf(os.Stderr, "    go run package.go <install-dir>\n")
			os.Exit(1)
		}
	}

	if installDir == "" {
		fmt.Fprintf(os.Stderr, "Error: install-dir cannot be empty\n")
		os.Exit(1)
	}

	fmt.Printf("Running package script with install-dir=%q, catalog-file=%q\n", installDir, *catalogFileFlag)

	if err := installLocal(installDir, *catalogFileFlag); err != nil {
		log.Fatalf("Error during installation/packaging: %v", err)
	}
	fmt.Println("Installation/packaging completed successfully!")
}

func installLocal(installDir, catalogFile string) error {
	// 1. Get the directory where this code file itself is located.
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("unable to determine code file location via runtime.Caller")
	}
	sourceDir := filepath.Dir(filename)

	excludedSkills, err := loadExcludedSkills(catalogFile)
	if err != nil {
		return err
	}

	// a) If they exist, delete folders _bmad and skills (directly inside installDir)
	for _, folder := range []string{"_bmad", "skills"} {
		targetPath := filepath.Join(installDir, folder)
		if _, err := os.Stat(targetPath); err == nil {
			fmt.Printf("Deleting existing folder: %s\n", targetPath)
			if err := os.RemoveAll(targetPath); err != nil {
				return fmt.Errorf("failed to delete folder %s: %w", targetPath, err)
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to check existence of folder %s: %w", targetPath, err)
		}
	}

	// b) Copy _bmad directory
	sourceBmad := filepath.Join(sourceDir, "_bmad")
	targetBmad := filepath.Join(installDir, "_bmad")
	fmt.Printf("Copying fresh copy from %s to %s...\n", sourceBmad, targetBmad)
	if err := copyDir(sourceBmad, targetBmad); err != nil {
		return fmt.Errorf("failed to copy folder from %s to %s: %w", sourceBmad, targetBmad, err)
	}

	// c) Copy skills directory, filtering out deprecated and non-APF supported skills
	sourceSkills := filepath.Join(sourceDir, "skills")
	targetSkills := filepath.Join(installDir, "skills")
	fmt.Printf("Copying fresh copy from %s to %s (filtering %d excluded skills)...\n", sourceSkills, targetSkills, len(excludedSkills))
	if err := os.MkdirAll(targetSkills, 0755); err != nil {
		return fmt.Errorf("failed to create target skills folder %s: %w", targetSkills, err)
	}

	entries, err := os.ReadDir(sourceSkills)
	if err != nil {
		return fmt.Errorf("failed to read source skills directory %s: %w", sourceSkills, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(sourceSkills, entry.Name())
		dstPath := filepath.Join(targetSkills, entry.Name())

		if entry.IsDir() {
			if excludedSkills[entry.Name()] {
				fmt.Printf("Skipping excluded skill (deprecated or not APF supported): %s\n", entry.Name())
				continue
			}
			if err := copyDir(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy skill from %s to %s: %w", srcPath, dstPath, err)
			}
		} else {
			entryInfo, err := entry.Info()
			if err != nil {
				return err
			}
			if err := copyFile(srcPath, dstPath, entryInfo.Mode()); err != nil {
				return fmt.Errorf("failed to copy file %s to %s: %w", srcPath, dstPath, err)
			}
		}
	}

	return nil
}

// copyDir recursively copies a directory tree, attempting to preserve permissions and symlinks.
func copyDir(src string, dst string) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode().Perm()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Handle symlinks
			if entry.Type()&os.ModeSymlink != 0 {
				linkTarget, err := os.Readlink(srcPath)
				if err != nil {
					return err
				}
				if err := os.Symlink(linkTarget, dstPath); err != nil {
					return err
				}
			} else {
				entryInfo, err := entry.Info()
				if err != nil {
					return err
				}
				if err := copyFile(srcPath, dstPath, entryInfo.Mode()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// copyFile copies a single file and its permissions.
func copyFile(src string, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
