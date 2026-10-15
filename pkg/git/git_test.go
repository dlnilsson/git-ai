package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func runGit(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestHasStaged(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	runGit(t, "init", "-q")

	staged, err := HasStaged()
	if err != nil || staged {
		t.Fatalf("empty repo: staged=%v err=%v, want false nil", staged, err)
	}

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	staged, err = HasStaged()
	if err != nil || staged {
		t.Fatalf("unstaged file: staged=%v err=%v, want false nil", staged, err)
	}

	runGit(t, "add", "a.txt")
	staged, err = HasStaged()
	if err != nil || !staged {
		t.Fatalf("staged file: staged=%v err=%v, want true nil", staged, err)
	}

	runGit(t, "commit", "-q", "-m", "init")
	staged, err = HasStaged()
	if err != nil || staged {
		t.Fatalf("after commit: staged=%v err=%v, want false nil", staged, err)
	}
}

func TestHasStagedNotGitDir(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(t.TempDir()))
	if _, err := HasStaged(); err != ErrNotGitDir {
		t.Fatalf("err=%v, want ErrNotGitDir", err)
	}
}
