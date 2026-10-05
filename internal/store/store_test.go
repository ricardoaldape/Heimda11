package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ricardoaldape/Heimda11/internal/core"
)

func TestFileStorePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	first, err := NewFileStore(path)
	if err != nil { t.Fatal(err) }
	agent := core.Agent{ID:"agt_test",Name:"Test",Role:"worker",HumanOwner:"owner",Status:core.AgentActive,CreatedAt:time.Now().UTC()}
	if err := first.UpsertAgent(agent); err != nil { t.Fatal(err) }

	second, err := NewFileStore(path)
	if err != nil { t.Fatal(err) }
	got, err := second.Agent(agent.ID)
	if err != nil { t.Fatal(err) }
	if got.Name != agent.Name { t.Fatalf("got %q", got.Name) }
}
