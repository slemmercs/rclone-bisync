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
	targetDir, err := filepath.Abs(filepath.Join(baseDir, relPath))
	if err != nil {
		return baseDir, "", false
	}
	// First: lexical check
	rel, err := filepath.Rel(baseDir, targetDir)
	if err != nil || strings.HasPrefix(rel, "..") {
		return baseDir, targetDir, false
	}

	// Second: symlink realpath confinement check
	realBase, err := filepath.EvalSymlinks(baseDir)
	if err == nil {
		fi, lstatErr := os.Lstat(targetDir)
		if lstatErr == nil && fi.Mode()&os.ModeSymlink != 0 {
			linkDest, err := os.Readlink(targetDir)
			if err == nil {
				if !filepath.IsAbs(linkDest) {
					linkDest = filepath.Join(filepath.Dir(targetDir), linkDest)
				}
				realDest, err := filepath.EvalSymlinks(linkDest)
				if err != nil {
					realDest, _ = filepath.Abs(linkDest)
				}
				realRel, err := filepath.Rel(realBase, realDest)
				if err != nil || strings.HasPrefix(realRel, "..") {
					return baseDir, targetDir, false
				}
			}
		}

		realTarget, err := filepath.EvalSymlinks(targetDir)
		if err == nil {
			realRel, err := filepath.Rel(realBase, realTarget)
			if err != nil || strings.HasPrefix(realRel, "..") {
				return baseDir, targetDir, false
			}
		} else if os.IsNotExist(err) {
			// Target does not exist yet (e.g. file to be created or copied)
			// Verify closest existing parent directory
			parentDir := filepath.Dir(targetDir)
			for {
				if realParent, pErr := filepath.EvalSymlinks(parentDir); pErr == nil {
					realRel, rErr := filepath.Rel(realBase, realParent)
					if rErr != nil || strings.HasPrefix(realRel, "..") {
						return baseDir, targetDir, false
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

	return baseDir, targetDir, true
}

func isLocalPathJailed(rawBaseDir, relPath string) (string, string, bool) {
	return IsLocalPathJailed(rawBaseDir, relPath)
}
