package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadPromptsReadsVersionedContracts(t *testing.T) {
	promptDir := t.TempDir()
	schemaDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(promptDir, "business_agent_system_v1.2.0.txt"), []byte("system"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(promptDir, "business_agent_user_v1.2.0.txt"), []byte("user"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(schemaDir, "business-agent-output-v1.0.0.json"), []byte(`{"type":"object"}`), 0o600))

	prompts, err := LoadPrompts(promptDir, schemaDir)
	require.NoError(t, err)
	require.Equal(t, "business_agent", prompts.Name)
	require.Equal(t, "1.2.0", prompts.Version)
	require.Equal(t, "business-agent-output", prompts.SchemaName)
	require.Equal(t, "1.0.0", prompts.SchemaVersion)
	require.JSONEq(t, `{"type":"object"}`, prompts.OutputSchema)
}

func TestLoadPromptsFailsWhenVersionedSchemaIsMissing(t *testing.T) {
	promptDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(promptDir, "business_agent_system_v1.2.0.txt"), []byte("system"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(promptDir, "business_agent_user_v1.2.0.txt"), []byte("user"), 0o600))

	_, err := LoadPrompts(promptDir, t.TempDir())
	require.Error(t, err)
}
