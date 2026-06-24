package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// RenderDiff produces the colored diff text for base..target, optionally
// restricted to a single file. When base is empty the empty tree is used (root
// commit). The result is rendered through delta when enabled, otherwise with
// git's own colored output. width is used to wrap delta's output to the pane.
func (r *Repo) RenderDiff(base, target, file string, width int) (string, error) {
	if base == "" {
		base = emptyTree
	}
	gitArgs := []string{"diff", "--find-renames", base, target}
	if file != "" {
		gitArgs = append(gitArgs, "--", file)
	}

	if r.useDelta {
		return r.renderWithDelta(gitArgs, width)
	}
	colorArgs := append([]string{"-c", "color.ui=always"}, gitArgs...)
	return r.runGit(colorArgs...)
}

// renderWithDelta runs `git <gitArgs> | delta` and returns delta's output.
func (r *Repo) renderWithDelta(gitArgs []string, width int) (string, error) {
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

	if gitWaitErr != nil {
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
