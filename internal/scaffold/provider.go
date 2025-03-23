package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Move SchemaCache and all provider-related functions here
// (fetchProviderSchema, initSchemaCache, cleanupSchemaCache)

func fetchProviderSchema(provider, version, resource string) (*ProviderSchema, error) {
	cache, err := initSchemaCache()
	if err != nil {
		return nil, err
	}

	if cache.Schema != nil {
		return cache.Schema, nil
	}

	// Create provider.tf in cache directory
	providerConfig := fmt.Sprintf(`
terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "%s"
    }
  }
}

provider "azurerm" {
  features {}
}`, version)

	providerPath := filepath.Join(cache.CachePath, "provider.tf")
	if err := os.WriteFile(providerPath, []byte(providerConfig), 0644); err != nil {
		return nil, fmt.Errorf("failed to write provider.tf: %w", err)
	}

	cmd := exec.Command("terraform", "init")
	cmd.Dir = cache.CachePath
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("terraform init failed: %s: %w", string(out), err)
	}

	cmd = exec.Command("terraform", "providers", "schema", "-json")
	cmd.Dir = cache.CachePath
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("terraform providers schema failed: %w", err)
	}

	var schema ProviderSchema
	if err := json.Unmarshal(out, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	// Store schema in cache
	cache.Schema = &schema

	return &schema, nil
}
