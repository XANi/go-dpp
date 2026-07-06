package overlord

import (
	"os/exec"
	"testing"

	"github.com/XANi/go-dpp/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// makeOriginRepo creates a local git repo with a single commit on master,
// usable as a pull URL for repo.New.
func makeOriginRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-b", "master", dir},
		{"-C", dir, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "init"},
	} {
		out, err := exec.Command("git", args...).CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}
	return dir
}

// testConfig returns a config with one defined repo ("good") and one that is
// listed in use_repos but has no definition ("missing"). The undefined repo
// comes first, so tests prove processing continues past it.
func testConfig(t *testing.T, killOnBadConfig bool) *config.Config {
	t.Helper()
	if _, err := exec.LookPath("puppet"); err != nil {
		t.Skip("puppet binary not available: " + err.Error())
	}
	dir := t.TempDir()
	return &config.Config{
		Logger:       zaptest.NewLogger(t).Sugar(),
		WorkDir:      dir,
		RepoDir:      dir + "/repos",
		ManifestFrom: "good",
		UseRepos:     []string{"missing", "good"},
		Repo: map[string]config.Repository{
			"good": {PullUrl: makeOriginRepo(t)},
		},
		KillOnBadConfig: killOnBadConfig,
	}
}

// Regression test: a repo listed in use_repos but missing from the repo map
// should be skipped when KillOnBadConfig is false, while defined repos are
// still set up and the daemon keeps running.
func TestNewSkipsUndefinedRepo(t *testing.T) {
	cfg := testConfig(t, false)
	o, err := New(cfg)
	require.NoError(t, err, "undefined repo should be skipped, not fail overlord startup")
	require.NotNil(t, o)
	assert.Contains(t, o.repos, "good")
	assert.NotContains(t, o.repos, "missing")
}

// With KillOnBadConfig set, an undefined repo should abort startup instead.
func TestNewUndefinedRepoKillOnBadConfig(t *testing.T) {
	cfg := testConfig(t, true)
	assert.Panics(t, func() { _, _ = New(cfg) },
		"undefined repo with kill_on_bad_config should abort startup")
}
