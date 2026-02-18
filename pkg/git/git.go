package git

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

var ErrNotGitDir = errors.New("not a git directory")

// maxDiffBytes is the maximum number of bytes we read from git diff --staged.
// Diffs larger than this are truncated with a notice so the prompt stays within
// model context limits.
const maxDiffBytes = 512 * 1024

func DiffStaged() (string, error) {
	check := exec.Command("git", "rev-parse", "--git-dir")
	check.Stderr = io.Discard
	if err := check.Run(); err != nil {
		return "", ErrNotGitDir
	}
	cmd := exec.Command("git", "diff", "--staged")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read staged diff (git diff --staged): %w", err)
	}
	if len(out) > maxDiffBytes {
		stat := exec.Command("git", "diff", "--staged", "--stat")
		stat.Stderr = os.Stderr
		statOut, statErr := stat.Output()
		if statErr != nil {
			return "", fmt.Errorf("failed to read staged diff stat: %w", statErr)
		}
		return "[diff too large; showing --stat summary only]\n" + string(statOut), nil
	}
	return string(out), nil
}
