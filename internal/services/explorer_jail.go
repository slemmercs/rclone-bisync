package services

import (
	"os"
	"path/filepath"
	"strings"
)

// IsLocalPathJailed verifies that relPath resolved against rawBaseDir is strictly contained inside rawBaseDir.
// It resolves real paths on disk using filepath.EvalSymlinks to prevent symlink jail escapes.
// It returns baseAbs, targetAbs, and true if safe, or false if path traversal is detected.
func IsLocalPathJailed(rawBaseDir, relPath string) (string, string, bool) {
	if strings.HasPrefix(rawBaseDir, "~") {
		home, _ := os.UserHomeDir()
		rawBaseDir = filepath.Join(home, strings.TrimPrefix(rawBaseDir, "~"))
	}
	baseDir, err := filepath.Abs(rawBaseDir)
	if err != nil {
		return "", "", false
	}
	cleanBase := filepath.Clean(baseDir)
	basePrefix := cleanBase + string(filepath.Separator)

	targetDir, err := filepath.Abs(filepath.Join(cleanBase, relPath))
	if err != nil {
		return cleanBase, "", false
	}
	cleanTarget := filepath.Clean(targetDir)

	// First: lexical confinement check
	if cleanTarget != cleanBase && !strings.HasPrefix(cleanTarget, basePrefix) {
		return cleanBase, "", false
	}
	rel, err := filepath.Rel(cleanBase, cleanTarget)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return cleanBase, "", false
	}
	safeTarget := filepath.Join(cleanBase, rel)
	if safeTarget != cleanBase && !strings.HasPrefix(safeTarget, basePrefix) {
		return cleanBase, "", false
	}

	// Second: symlink realpath confinement check
	realBase, err := filepath.EvalSymlinks(cleanBase)
	if err == nil {
		realBasePrefix := realBase + string(filepath.Separator)
		fi, lstatErr := os.Lstat(safeTarget)
		if lstatErr == nil && fi.Mode()&os.ModeSymlink != 0 {
			linkDest, err := os.Readlink(safeTarget)
			if err == nil {
				if !filepath.IsAbs(linkDest) {
					linkDest = filepath.Join(filepath.Dir(safeTarget), linkDest)
				}
				realDest, err := filepath.EvalSymlinks(linkDest)
				if err != nil {
					realDest, _ = filepath.Abs(linkDest)
				}
				realRel, err := filepath.Rel(realBase, realDest)
				if err != nil || strings.HasPrefix(realRel, "..") {
					return cleanBase, "", false
				}
			}
		}

		realTarget, err := filepath.EvalSymlinks(safeTarget)
		if err == nil {
			if realTarget != realBase && !strings.HasPrefix(realTarget, realBasePrefix) {
				return cleanBase, "", false
			}
			realRel, err := filepath.Rel(realBase, realTarget)
			if err != nil || strings.HasPrefix(realRel, "..") {
				return cleanBase, "", false
			}
		} else if os.IsNotExist(err) {
			// Target does not exist yet (e.g. file to be created or copied)
			// Verify closest existing parent directory
			parentDir := filepath.Dir(safeTarget)
			for {
				if realParent, pErr := filepath.EvalSymlinks(parentDir); pErr == nil {
					if realParent != realBase && !strings.HasPrefix(realParent, realBasePrefix) {
						return cleanBase, "", false
					}
					realRel, rErr := filepath.Rel(realBase, realParent)
					if rErr != nil || strings.HasPrefix(realRel, "..") {
						return cleanBase, "", false
					}
					break
				}
				nextParent := filepath.Dir(parentDir)
				if nextParent == parentDir {
					break
				}
				parentDir = nextParent
			}
		}
	}

	return cleanBase, safeTarget, true
}

func isLocalPathJailed(rawBaseDir, relPath string) (string, string, bool) {
	return IsLocalPathJailed(rawBaseDir, relPath)
}
