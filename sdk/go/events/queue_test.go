package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueuePersistsAndAcknowledges(t *testing.T) {
	dir := t.TempDir()
	queue, err := OpenQueue(dir)
	require.NoError(t, err)

	require.NoError(t, queue.Push(Event{Seq: 2, Type: "b"}))
	require.NoError(t, queue.Push(Event{Seq: 1, Type: "a"}))
	require.NoError(t, queue.Push(Event{Seq: 2, Type: "b"}))

	reopened, err := OpenQueue(dir)
	require.NoError(t, err)
	pending := reopened.Pending()
	require.Len(t, pending, 2)
	assert.Equal(t, int64(1), pending[0].Seq)
	assert.Equal(t, int64(2), pending[1].Seq)

	require.NoError(t, reopened.Ack(1))
	assert.Equal(t, int64(1), reopened.LastSeq())
	assert.Len(t, reopened.Pending(), 1)

	again, err := OpenQueue(dir)
	require.NoError(t, err)
	assert.Equal(t, int64(1), again.LastSeq())
	require.Len(t, again.Pending(), 1)
	assert.Equal(t, int64(2), again.Pending()[0].Seq)

	require.NoError(t, again.Ack(2))
	assert.Equal(t, int64(2), again.LastSeq())
	assert.Empty(t, again.Pending())
}
