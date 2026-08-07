package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExampleWorkerFlow(t *testing.T) {
	processed, states, err := runExample()
	require.NoError(t, err)
	assert.Equal(t, 2, processed)
	assert.Equal(t, map[string]string{"task-1": "done", "task-2": "failed"}, states)
}
