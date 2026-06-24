package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// RenderDiff produces the colored diff text for base..target, optionally
// restricted to a single file. When base is empty the empty tree is used (root
// commit).
func (r *Repo) RenderDiff(base, target, file string, width int) (string, error) {
	if base == "" {
		base = emptyTree
	}
	args := []string{"diff", "--find-renames", base, target}
	if file != "" {
		args = append(args, "--", file)
	}
	return r.renderDiffArgs(args, width, false)
}

// RenderWorking renders unstaged changes (working tree vs index).
func (r *Repo) RenderWorking(file string, width int) (string, error) {
	args := []string{"diff", "--find-renames"}
	if file != "" {
		args = append(args, "--", file)
	}
	return r.renderDiffArgs(args, width, false)
}

// RenderStaged renders staged changes (index vs HEAD).
func (r *Repo) RenderStaged(file string, width int) (string, error) {
	args := []string{"diff", "--cached", "--find-renames"}
	if file != "" {
		args = append(args, "--", file)
	}
	return r.renderDiffArgs(args, width, false)
}

// RenderUntracked renders an untracked file as additions, via `git diff
// --no-index`. That command exits 1 when the files differ, which is expected.
func (r *Repo) RenderUntracked(file string, width int) (string, error) {
	args := []string{"diff", "--no-index", "--", os.DevNull, file}
	return r.renderDiffArgs(args, width, true)
}

// renderDiffArgs runs the given `git <args>` and returns colored diff text,
// piped through delta when enabled. allowExit1 tolerates git's exit code 1
// (used by --no-index, which returns 1 when files differ).
func (r *Repo) renderDiffArgs(gitArgs []string, width int, allowExit1 bool) (string, error) {
	if r.useDelta {
		return r.renderWithDelta(gitArgs, width, allowExit1)
	}
	colorArgs := append([]string{"-c", "color.ui=always"}, gitArgs...)
	return r.runGitAllow(colorArgs, allowExit1)
}

// runGitAllow is like runGit but optionally tolerates exit code 1.
func (r *Repo) runGitAllow(args []string, allowExit1 bool) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.root
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil && !(allowExit1 && exitCode(err) == 1) {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return out.String(), nil
}

// renderWithDelta runs `git <gitArgs> | delta` and returns delta's output.
func (r *Repo) renderWithDelta(gitArgs []string, width int, allowExit1 bool) (string, error) {
	gitCmd := exec.Command("git", gitArgs...)
	gitCmd.Dir = r.root
	gitOut, err := gitCmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	var gitErr bytes.Buffer
	gitCmd.Stderr = &gitErr

	deltaArgs := []string{"--paging=never"}
	if width > 0 {
		deltaArgs = append(deltaArgs, "--width", strconv.Itoa(width))
	}
	deltaCmd := exec.Command("delta", deltaArgs...)
	deltaCmd.Stdin = gitOut
	var out, deltaErr bytes.Buffer
	deltaCmd.Stdout = &out
	deltaCmd.Stderr = &deltaErr

	if err := gitCmd.Start(); err != nil {
		return "", err
	}
	if err := deltaCmd.Start(); err != nil {
		return "", err
	}
	gitWaitErr := gitCmd.Wait()
	deltaWaitErr := deltaCmd.Wait()

	if gitWaitErr != nil && !(allowExit1 && exitCode(gitWaitErr) == 1) {
		msg := strings.TrimSpace(gitErr.String())
		if msg == "" {
			msg = gitWaitErr.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(gitArgs, " "), msg)
	}
	if deltaWaitErr != nil {
		msg := strings.TrimSpace(deltaErr.String())
		if msg == "" {
			msg = deltaWaitErr.Error()
		}
		return "", fmt.Errorf("delta: %s", msg)
	}
	return out.String(), nil
}

func exitCode(err error) int {
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	return -1
}
