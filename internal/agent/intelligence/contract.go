package intelligence

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Contract struct {
	AgentName     string
	PromptName    string
	PromptVersion string
	SchemaName    string
	SchemaVersion string
	SystemPrompt  string
	OutputSchema  string
}

func LoadContract(promptDirectory, schemaDirectory, agentName, promptName, promptVersion, schemaName, schemaVersion string) (Contract, error) {
	read := func(directory, name string) (string, error) {
		value, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			return "", fmt.Errorf("read agent intelligence contract %s: %w", name, err)
		}
		return strings.TrimSpace(string(value)), nil
	}
	system, err := read(promptDirectory, promptName+"_system_v"+promptVersion+".txt")
	if err != nil {
		return Contract{}, err
	}
	schema, err := read(schemaDirectory, schemaName+"-v"+schemaVersion+".json")
	if err != nil {
		return Contract{}, err
	}
	return Contract{
		AgentName: agentName, PromptName: promptName, PromptVersion: promptVersion,
		SchemaName: schemaName, SchemaVersion: schemaVersion,
		SystemPrompt: system, OutputSchema: schema,
	}, nil
}
