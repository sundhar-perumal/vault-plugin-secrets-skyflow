package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/sundhar-perumal/vault-plugin-secrets-skyflow/backend"
	"github.com/hashicorp/vault/sdk/logical"
)

func main() {
	credPath := "sandbox_poc.json"

	// Check if file exists
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		log.Fatalf("Credentials file not found: %s", credPath)
	}

	// Read credentials file
	credJSON, err := os.ReadFile(credPath)
	if err != nil {
		log.Fatalf("Failed to read credentials file: %v", err)
	}

	// Create backend
	ctx := context.Background()
	storage := &logical.InmemStorage{}

	config := &logical.BackendConfig{
		Logger:      nil,
		System:      &logical.StaticSystemView{},
		StorageView: storage,
	}

	b, err := backend.Factory(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create backend: %v", err)
	}

	// Configure backend with credentials JSON
	configReq := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "config",
		Storage:   storage,
		Data: map[string]interface{}{
			"credentials_json":     string(credJSON),
			"validate_credentials": false,
		},
		Connection: &logical.Connection{RemoteAddr: "127.0.0.1"},
	}

	resp, err := b.HandleRequest(ctx, configReq)
	if err != nil || (resp != nil && resp.IsError()) {
		log.Fatalf("Failed to configure backend: %v %v", err, resp)
	}
	fmt.Println("✓ Backend configured")

	// Create a role with Skyflow role IDs (required)
	roleReq := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "roles/example-role",
		Storage:   storage,
		Data: map[string]interface{}{
			"role_ids":    []string{"your-skyflow-role-id"}, // Replace with actual Skyflow role ID
			"description": "Example role for Sandbox credentials",
		},
		Connection: &logical.Connection{RemoteAddr: "127.0.0.1"},
	}

	resp, err = b.HandleRequest(ctx, roleReq)
	if err != nil || (resp != nil && resp.IsError()) {
		log.Fatalf("Failed to create role: %v %v", err, resp)
	}
	fmt.Println("✓ Role created")

	// Fetch token
	credsReq := &logical.Request{
		Operation:  logical.ReadOperation,
		Path:       "creds/example-role",
		Storage:    storage,
		Connection: &logical.Connection{RemoteAddr: "127.0.0.1"},
	}

	resp, err = b.HandleRequest(ctx, credsReq)
	if err != nil {
		log.Fatalf("Failed to fetch token: %v", err)
	}
	if resp != nil && resp.IsError() {
		log.Fatalf("Token generation error: %s", resp.Error().Error())
	}

	fmt.Println("=== Token Generated Successfully ===")
	accessToken := resp.Data["access_token"].(string)
	if len(accessToken) > 50 {
		fmt.Printf("Access Token: %s...\n", accessToken[:50])
	} else {
		fmt.Printf("Access Token: %s\n", accessToken)
	}
	fmt.Printf("Token Type: %s\n", resp.Data["token_type"])
}
