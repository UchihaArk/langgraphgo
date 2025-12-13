package postgres

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v3"
	"github.com/smallnest/langgraphgo/graph"
	"github.com/stretchr/testify/assert"
)

func TestPostgresCheckpointStore_Save(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	cp := &graph.Checkpoint{
		ID:        "cp-1",
		NodeName:  "node-a",
		State:     map[string]any{"foo": "bar"},
		Timestamp: time.Now(),
		Version:   1,
		Metadata: map[string]any{
			"execution_id": "exec-1",
		},
	}

	stateJSON, _ := json.Marshal(cp.State)
	metadataJSON, _ := json.Marshal(cp.Metadata)

	// Expect INSERT
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO checkpoints")).
		WithArgs(
			cp.ID,
			"exec-1",
			cp.NodeName,
			stateJSON,
			metadataJSON,
			cp.Timestamp,
			cp.Version,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = store.Save(context.Background(), cp)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCheckpointStore_Load(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	store := NewPostgresCheckpointStoreWithPool(mock, "checkpoints")

	cpID := "cp-1"
	timestamp := time.Now()
	state := map[string]any{"foo": "bar"}
	metadata := map[string]any{"execution_id": "exec-1"}

	stateJSON, _ := json.Marshal(state)
	metadataJSON, _ := json.Marshal(metadata)

	rows := pgxmock.NewRows([]string{"id", "node_name", "state", "metadata", "timestamp", "version"}).
		AddRow(cpID, "node-a", stateJSON, metadataJSON, timestamp, 1)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, node_name, state, metadata, timestamp, version FROM checkpoints WHERE id = $1")).
		WithArgs(cpID).
		WillReturnRows(rows)

	loaded, err := store.Load(context.Background(), cpID)
	assert.NoError(t, err)
	assert.Equal(t, cpID, loaded.ID)
	assert.Equal(t, "node-a", loaded.NodeName)
	assert.Equal(t, 1, loaded.Version)

	// Check state
	loadedState, ok := loaded.State.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "bar", loadedState["foo"])

	assert.NoError(t, mock.ExpectationsWereMet())
}
