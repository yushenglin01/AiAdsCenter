package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PromptSet struct {
	Name           string
	Version        string
	SchemaName     string
	SchemaVersion  string
	OutputSchema   string
	BusinessSystem string
	BusinessUser   string
}

func LoadPrompts(promptDirectory, schemaDirectory string) (PromptSet, error) {
	read := func(directory, name string) (string, error) {
		value, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			return "", fmt.Errorf("read prompt %s: %w", name, err)
		}
		return strings.TrimSpace(string(value)), nil
	}
	system, err := read(promptDirectory, "business_agent_system_v1.1.0.txt")
	if err != nil {
		return PromptSet{}, err
	}
	user, err := read(promptDirectory, "business_agent_user_v1.1.0.txt")
	if err != nil {
		return PromptSet{}, err
	}
	schema, err := read(schemaDirectory, "business-agent-output-v1.0.0.json")
	if err != nil {
		return PromptSet{}, err
	}
	return PromptSet{Name: "business_agent", Version: "1.1.0", SchemaName: "business-agent-output", SchemaVersion: "1.0.0", OutputSchema: schema, BusinessSystem: system, BusinessUser: user}, nil
}
