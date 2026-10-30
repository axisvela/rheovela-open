package conformance

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"testing"

	schema "github.com/axisvela/rheovela-open/v3/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type invalidExpectation struct {
	Code            string `json:"code"`
	Line            int    `json:"line"`
	MessageContains string `json:"message_contains"`
}

var codePattern = regexp.MustCompile(`^RHEOIR[0-9]{3}$`)

func TestManifestAndCorpus(t *testing.T) {
	m, err := LoadManifest(RHEOIRv1)
	require.NoError(t, err)
	require.Len(t, m.Cases.Valid, 7)
	require.Len(t, m.Cases.Invalid, 22)

	referenced := map[string]bool{}
	for _, c := range m.Cases.Valid {
		referenced[RHEOIRv1Root+"/"+c.Source] = true
		referenced[RHEOIRv1Root+"/"+c.Expected] = true

		src, err := fs.ReadFile(RHEOIRv1, RHEOIRv1Root+"/"+c.Source)
		require.NoError(t, err, c.Name)
		assert.NotEmpty(t, src, c.Name)

		raw, err := fs.ReadFile(RHEOIRv1, RHEOIRv1Root+"/"+c.Expected)
		require.NoError(t, err, c.Name)
		var def schema.WorkflowDefinition
		require.NoError(t, json.Unmarshal(raw, &def), c.Name)
		assert.NotEmpty(t, def.WorkflowID, c.Name)
		assert.Equal(t, "rheo-ir", def.Source, c.Name)
		assert.NotEmpty(t, def.Stages, c.Name)
		assert.NotNil(t, def.Transitions, c.Name)
		for _, stage := range def.Stages {
			assert.NotEmpty(t, stage.ID, c.Name)
			assert.NotNil(t, stage.Inputs, "%s stage %s inputs", c.Name, stage.ID)
			assert.NotNil(t, stage.Outputs, "%s stage %s outputs", c.Name, stage.ID)
		}
	}

	seenCodes := map[string]int{}
	for _, c := range m.Cases.Invalid {
		referenced[RHEOIRv1Root+"/"+c.Source] = true
		referenced[RHEOIRv1Root+"/"+c.Expected] = true

		src, err := fs.ReadFile(RHEOIRv1, RHEOIRv1Root+"/"+c.Source)
		require.NoError(t, err, c.Name)
		assert.NotEmpty(t, src, c.Name)

		raw, err := fs.ReadFile(RHEOIRv1, RHEOIRv1Root+"/"+c.Expected)
		require.NoError(t, err, c.Name)
		var want invalidExpectation
		require.NoError(t, json.Unmarshal(raw, &want), c.Name)
		assert.Regexp(t, codePattern, want.Code, c.Name)
		assert.Greater(t, want.Line, 0, c.Name)
		assert.NotEmpty(t, want.MessageContains, c.Name)
		seenCodes[want.Code]++
	}

	for i := 1; i <= 22; i++ {
		code := fmt.Sprintf("RHEOIR%03d", i)
		assert.Equal(t, 1, seenCodes[code], "code %s coverage", code)
	}

	for _, pattern := range []string{"valid/*.rheo", "valid/*.json", "invalid/*.rheo", "invalid/*.json"} {
		files, err := fs.Glob(RHEOIRv1, RHEOIRv1Root+"/"+pattern)
		require.NoError(t, err)
		require.NotEmpty(t, files, pattern)
		for _, file := range files {
			assert.True(t, referenced[file], "unreferenced fixture %s", file)
		}
	}
}
