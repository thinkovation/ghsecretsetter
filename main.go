package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v60/github"
	"golang.org/x/crypto/nacl/box"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Owner   string            `yaml:"owner"`
	Repo    string            `yaml:"repo"`
	Secret  string            `yaml:"secret,omitempty"`  // Keep for backward compatibility
	Value   string            `yaml:"value,omitempty"`   // Keep for backward compatibility
	Secrets map[string]string `yaml:"secrets,omitempty"` // New: multiple secrets
	Token   string            `yaml:"token,omitempty"`
}

func main() {
	// CLI flags
	configPath := flag.String("config", "", "Path to optional YAML config file")
	owner := flag.String("owner", "", "GitHub repository owner")
	repo := flag.String("repo", "", "GitHub repository name")
	secret := flag.String("secret", "", "Name of the secret to set (for single secret)")
	value := flag.String("value", "", "Value of the secret (for single secret)")
	token := flag.String("token", "", "GitHub personal access token (optional, can use GITHUB_TOKEN env var)")

	flag.Parse()

	cfg := Config{}

	// Load YAML config if provided
	if *configPath != "" {
		fileCfg, err := loadConfig(*configPath)
		if err != nil {
			log.Fatalf("Failed to load config file: %v", err)
		}
		cfg = *fileCfg
	}

	// Override config values with flags if provided
	if *owner != "" {
		cfg.Owner = *owner
	}
	if *repo != "" {
		cfg.Repo = *repo
	}
	if *token != "" {
		cfg.Token = *token
	}

	// Handle single secret from CLI flags
	if *secret != "" && *value != "" {
		if cfg.Secrets == nil {
			cfg.Secrets = make(map[string]string)
		}
		cfg.Secrets[*secret] = *value
	} else if *secret != "" || *value != "" {
		log.Fatal("Both -secret and -value must be provided when using CLI flags")
	}

	// Handle backward compatibility with single secret in config
	if cfg.Secret != "" && cfg.Value != "" {
		if cfg.Secrets == nil {
			cfg.Secrets = make(map[string]string)
		}
		cfg.Secrets[cfg.Secret] = cfg.Value
	}

	// Validate required fields
	if cfg.Owner == "" || cfg.Repo == "" {
		log.Fatal("Missing required values: owner, repo")
	}
	if len(cfg.Secrets) == 0 {
		log.Fatal("No secrets to set. Use -secret/-value flags or 'secrets' in config file")
	}

	// Determine token source
	finalToken := cfg.Token
	if finalToken == "" {
		finalToken = os.Getenv("GITHUB_TOKEN")
	}
	if finalToken == "" {
		log.Fatal("GitHub token not provided via flag, config, or GITHUB_TOKEN env var")
	}

	// GitHub client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: finalToken})
	client := github.NewClient(oauth2.NewClient(ctx, ts))

	// Fetch public key once
	key, _, err := client.Actions.GetRepoPublicKey(ctx, cfg.Owner, cfg.Repo)
	if err != nil {
		log.Fatalf("Failed to get public key: %v", err)
	}

	// Set all secrets
	successCount := 0
	for secretName, secretValue := range cfg.Secrets {
		// Resolve secret value (handle file() syntax)
		resolvedValue, err := resolveSecretValue(secretValue)
		if err != nil {
			log.Printf("Failed to resolve secret '%s': %v", secretName, err)
			continue
		}

		// Encrypt the secret
		encrypted, err := encryptSecret(key.GetKey(), resolvedValue)
		if err != nil {
			log.Printf("Failed to encrypt secret '%s': %v", secretName, err)
			continue
		}

		// Set the secret
		secretReq := &github.EncryptedSecret{
			Name:           secretName,
			KeyID:          key.GetKeyID(),
			EncryptedValue: encrypted,
		}
		_, err = client.Actions.CreateOrUpdateRepoSecret(ctx, cfg.Owner, cfg.Repo, secretReq)
		if err != nil {
			log.Printf("Failed to set secret '%s': %v", secretName, err)
			continue
		}

		fmt.Printf("Secret '%s' set successfully\n", secretName)
		successCount++
	}

	if successCount == len(cfg.Secrets) {
		fmt.Printf("All %d secrets set successfully!\n", successCount)
	} else {
		fmt.Printf("Warning: %d of %d secrets set successfully\n", successCount, len(cfg.Secrets))
	}
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func resolveSecretValue(value string) (string, error) {
	// Check if value uses file() syntax
	if strings.HasPrefix(value, "file(") && strings.HasSuffix(value, ")") {
		// Extract file path from file(path)
		filePath := value[5 : len(value)-1] // Remove "file(" and ")"
		if filePath == "" {
			return "", fmt.Errorf("empty file path in file() syntax")
		}
		
		// Read file contents
		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read file '%s': %w", filePath, err)
		}
		
		// Return content as string, trimming trailing newline if present
		result := string(content)
		if strings.HasSuffix(result, "\n") {
			result = strings.TrimSuffix(result, "\n")
		}
		return result, nil
	}
	
	// Return value as-is if not file syntax
	return value, nil
}

func encryptSecret(publicKeyBase64, secretValue string) (string, error) {
	// Decode the repository's public key
	recipientKey, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %w", err)
	}
	if len(recipientKey) != 32 {
		return "", fmt.Errorf("public key must be 32 bytes, got %d", len(recipientKey))
	}

	var publicKey [32]byte
	copy(publicKey[:], recipientKey)

	// Use libsodium sealed box format (anonymous encryption)
	// This is what GitHub expects - no ephemeral keys needed
	sealed, err := box.SealAnonymous(nil, []byte(secretValue), &publicKey, rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to seal secret: %w", err)
	}

	return base64.StdEncoding.EncodeToString(sealed), nil
}
