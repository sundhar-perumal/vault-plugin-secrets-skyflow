// +build ignore

package main

import (
	"fmt"
	"log"

	"github.com/skyflowapi/skyflow-go/v2/serviceaccount"
	"github.com/skyflowapi/skyflow-go/v2/utils/common"
	"github.com/skyflowapi/skyflow-go/v2/utils/logger"
)

func main() {
	credPath := "insurance_read_poc.json"

	fmt.Println("=== Direct SDK Test ===")
	fmt.Printf("Credentials file: %s\n", credPath)

	opts := common.BearerTokenOptions{LogLevel: logger.DEBUG}

	fmt.Println("Calling GenerateBearerToken...")
	token, err := serviceaccount.GenerateBearerToken(credPath, opts)

	if err != nil {
		log.Fatalf("SDK Error: %v", err)
	}

	if token == nil {
		log.Fatal("Token is nil!")
	}

	fmt.Println("=== Token Generated Successfully ===")
	if len(token.AccessToken) > 50 {
		fmt.Printf("Access Token: %s...\n", token.AccessToken[:50])
	} else {
		fmt.Printf("Access Token: %s\n", token.AccessToken)
	}
	fmt.Printf("Token Type: %s\n", token.TokenType)
}

