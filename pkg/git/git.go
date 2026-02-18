package git

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
	"sort"
	"strings"
)

var ErrNotGitDir = errors.New("not a git directory")

// maxDiffBytes is the cap for the full diff (used by codex backend).
const maxDiffBytes = 512 * 1024

// maxChunkBytes is the per-directory cap used by the chunked diff path.
const maxChunkBytes = 100 * 1024

// DiffChunk holds the staged diff (or --stat fallback) for one directory.
type DiffChunk struct {
	Dir  string
	Diff string
}

// gitCmd returns an exec.Cmd for git with GIT_PAGER=cat set so that git never
// invokes a pager regardless of the user's config.
func gitCmd(args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Env = append(cmd.Environ(), "GIT_PAGER=cat")
	return cmd
}

func checkGitDir() error {
	check := gitCmd("rev-parse", "--git-dir")
	check.Stderr = io.Discard
	if err := check.Run(); err != nil {
		return ErrNotGitDir
	}
	return nil
}

// DiffStaged returns the full staged diff, falling back to --stat when the
// diff exceeds maxDiffBytes. Used by the codex backend.
func DiffStaged() (string, error) {
	if err := checkGitDir(); err != nil {
		return "", err
	}
	cmd := gitCmd("diff", "--staged")
	cmd.Stderr = io.Discard
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read staged diff (git diff --staged): %w", err)
	}
	if len(out) > maxDiffBytes {
		stat := gitCmd("diff", "--staged", "--stat")
		stat.Stderr = io.Discard
		statOut, statErr := stat.Output()
		if statErr != nil {
			return "", fmt.Errorf("failed to read staged diff stat: %w", statErr)
		}
		return "[diff too large; showing --stat summary only]\n" + string(statOut), nil
	}
	return string(out), nil
}

// DiffStagedChunks returns one DiffChunk per changed directory, each capped
// at maxChunkBytes (falls back to --stat for that directory if exceeded).
// Used by the claude backend to send one stream-json message per directory.
func DiffStagedChunks() ([]DiffChunk, error) {
	if err := checkGitDir(); err != nil {
		return nil, err
	}

	// Collect changed file paths.
	namesCmd := gitCmd("diff", "--staged", "--name-only")
	namesCmd.Stderr = io.Discard
	namesOut, err := namesCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list staged files: %w", err)
	}

	// Group files by their immediate parent directory.
	dirSet := map[string]struct{}{}
	for _, file := range strings.Split(strings.TrimSpace(string(namesOut)), "\n") {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		dirSet[path.Dir(file)] = struct{}{}
	}
	if len(dirSet) == 0 {
		return nil, nil
	}

	dirs := make([]string, 0, len(dirSet))
	for d := range dirSet {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)

	chunks := make([]DiffChunk, 0, len(dirs))
	for _, dir := range dirs {
		diffCmd := gitCmd("diff", "--staged", "--", dir)
		diffCmd.Stderr = io.Discard
		diffOut, diffErr := diffCmd.Output()
		if diffErr != nil {
			return nil, fmt.Errorf("failed to get diff for %s: %w", dir, diffErr)
		}
		content := string(diffOut)
		if len(diffOut) > maxChunkBytes {
			statCmd := gitCmd("diff", "--staged", "--stat", "--", dir)
			statCmd.Stderr = io.Discard
			statOut, statErr := statCmd.Output()
			if statErr != nil {
				return nil, fmt.Errorf("failed to get stat for %s: %w", dir, statErr)
			}
			content = "[diff too large; showing --stat only]\n" + string(statOut)
		}
		if strings.TrimSpace(content) != "" {
			chunks = append(chunks, DiffChunk{Dir: dir, Diff: content})
		}
	}
	return chunks, nil
}
