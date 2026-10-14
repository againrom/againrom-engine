package game

import (
	"os"
	"testing"
)

// sessionCorpusAuditVariable opens the session corpus audits, the milestone-2
// family among them. scripts/check-milestone2-acceptance.sh sets it; the
// default run, which every chain leg shares, skips them.
const sessionCorpusAuditVariable = "AGAINROM_SESSION_CORPUS_AUDIT"

// sessionCorpusAudit skips a session corpus audit unless its variable is set.
// The reason names no AGAINROM_ variable: the release gate's runtime census
// counts such a skip as a release test. internal/gatedtests lists these tests
// apart from that population.
func sessionCorpusAudit(t *testing.T) {
	t.Helper()
	if os.Getenv(sessionCorpusAuditVariable) == "" {
		t.Skip("session corpus audit: run by scripts/check-milestone2-acceptance.sh")
	}
}
