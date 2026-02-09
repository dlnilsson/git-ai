package git

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

var ErrNotGitDir = errors.New("not a git directory")

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
	return string(out), nil
}
