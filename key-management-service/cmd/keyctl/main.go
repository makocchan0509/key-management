// Package main はCLIツールのエントリポイント。
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

const version = "1.0.0"

var (
	apiURL  string
	output  string
	timeout time.Duration
)

// HTTPクライアント
var httpClient *http.Client

func main() {
	rootCmd := &cobra.Command{
		Use:   "keyctl",
		Short: "Key Management Service CLI",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if apiURL == "" {
				apiURL = os.Getenv("KEYCTL_API_URL")
			}
			httpClient = &http.Client{Timeout: timeout}
		},
	}

	// グローバルフラグ
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "API endpoint URL (or set KEYCTL_API_URL)")
	rootCmd.PersistentFlags().StringVar(&output, "output", "text", "Output format: text, json")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 30*time.Second, "Request timeout")

	// サブコマンド登録
	rootCmd.AddCommand(createCmd())
	rootCmd.AddCommand(getCmd())
	rootCmd.AddCommand(rotateCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(disableCmd())
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// versionCmd はバージョン情報を表示する。
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("keyctl version %s\n", version)
		},
	}
}

// createCmd は鍵の生成コマンド。
func createCmd() *cobra.Command {
	var tenantID string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new key for a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" {
				return fmt.Errorf("--tenant is required")
			}
			if apiURL == "" {
				return fmt.Errorf("--api-url is required (or set KEYCTL_API_URL)")
			}

			url := fmt.Sprintf("%s/v1/tenants/%s/keys", apiURL, tenantID)
			resp, err := httpClient.Post(url, "application/json", nil)
			if err != nil {
				return fmt.Errorf("API request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading response: %w", err)
			}

			if resp.StatusCode != http.StatusCreated {
				return handleErrorResponse(resp.StatusCode, body)
			}

			if output == "json" {
				fmt.Println(string(body))
			} else {
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					return fmt.Errorf("parsing response: %w", err)
				}
				fmt.Printf("Created key for tenant %q (generation: %.0f)\n", tenantID, result["generation"])
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tenantID, "tenant", "", "Tenant ID (required)")
	cmd.MarkFlagRequired("tenant")
	return cmd
}

// getCmd は鍵の取得コマンド。
func getCmd() *cobra.Command {
	var tenantID string
	var generation uint
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a key for a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" {
				return fmt.Errorf("--tenant is required")
			}
			if apiURL == "" {
				return fmt.Errorf("--api-url is required (or set KEYCTL_API_URL)")
			}

			var url string
			if generation > 0 {
				url = fmt.Sprintf("%s/v1/tenants/%s/keys/%d", apiURL, tenantID, generation)
			} else {
				url = fmt.Sprintf("%s/v1/tenants/%s/keys/current", apiURL, tenantID)
			}

			resp, err := httpClient.Get(url)
			if err != nil {
				return fmt.Errorf("API request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading response: %w", err)
			}

			if resp.StatusCode != http.StatusOK {
				return handleErrorResponse(resp.StatusCode, body)
			}

			if output == "json" {
				fmt.Println(string(body))
			} else {
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					return fmt.Errorf("parsing response: %w", err)
				}
				fmt.Println(result["key"])
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tenantID, "tenant", "", "Tenant ID (required)")
	cmd.Flags().UintVar(&generation, "generation", 0, "Key generation (optional, defaults to current)")
	cmd.MarkFlagRequired("tenant")
	return cmd
}

// rotateCmd は鍵のローテーションコマンド。
func rotateCmd() *cobra.Command {
	var tenantID string
	cmd := &cobra.Command{
		Use:   "rotate",
		Short: "Rotate key for a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" {
				return fmt.Errorf("--tenant is required")
			}
			if apiURL == "" {
				return fmt.Errorf("--api-url is required (or set KEYCTL_API_URL)")
			}

			url := fmt.Sprintf("%s/v1/tenants/%s/keys/rotate", apiURL, tenantID)
			resp, err := httpClient.Post(url, "application/json", nil)
			if err != nil {
				return fmt.Errorf("API request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading response: %w", err)
			}

			if resp.StatusCode != http.StatusCreated {
				return handleErrorResponse(resp.StatusCode, body)
			}

			if output == "json" {
				fmt.Println(string(body))
			} else {
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					return fmt.Errorf("parsing response: %w", err)
				}
				fmt.Printf("Rotated key for tenant %q (new generation: %.0f)\n", tenantID, result["generation"])
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tenantID, "tenant", "", "Tenant ID (required)")
	cmd.MarkFlagRequired("tenant")
	return cmd
}

// listCmd は鍵一覧の取得コマンド。
func listCmd() *cobra.Command {
	var tenantID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all keys for a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" {
				return fmt.Errorf("--tenant is required")
			}
			if apiURL == "" {
				return fmt.Errorf("--api-url is required (or set KEYCTL_API_URL)")
			}

			url := fmt.Sprintf("%s/v1/tenants/%s/keys", apiURL, tenantID)
			resp, err := httpClient.Get(url)
			if err != nil {
				return fmt.Errorf("API request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading response: %w", err)
			}

			if resp.StatusCode != http.StatusOK {
				return handleErrorResponse(resp.StatusCode, body)
			}

			if output == "json" {
				fmt.Println(string(body))
			} else {
				var result struct {
					Keys []struct {
						Generation uint   `json:"generation"`
						Status     string `json:"status"`
						CreatedAt  string `json:"created_at"`
					} `json:"keys"`
				}
				if err := json.Unmarshal(body, &result); err != nil {
					return fmt.Errorf("parsing response: %w", err)
				}

				fmt.Printf("%-12s %-10s %s\n", "GENERATION", "STATUS", "CREATED_AT")
				for _, k := range result.Keys {
					fmt.Printf("%-12d %-10s %s\n", k.Generation, k.Status, k.CreatedAt)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tenantID, "tenant", "", "Tenant ID (required)")
	cmd.MarkFlagRequired("tenant")
	return cmd
}

// disableCmd は鍵の無効化コマンド。
func disableCmd() *cobra.Command {
	var tenantID string
	var generation uint
	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable a key for a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tenantID == "" {
				return fmt.Errorf("--tenant is required")
			}
			if generation == 0 {
				return fmt.Errorf("--generation is required")
			}
			if apiURL == "" {
				return fmt.Errorf("--api-url is required (or set KEYCTL_API_URL)")
			}

			url := fmt.Sprintf("%s/v1/tenants/%s/keys/%d", apiURL, tenantID, generation)
			req, err := http.NewRequest(http.MethodDelete, url, nil)
			if err != nil {
				return fmt.Errorf("creating request: %w", err)
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				return fmt.Errorf("API request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading response: %w", err)
			}

			if resp.StatusCode != http.StatusAccepted {
				return handleErrorResponse(resp.StatusCode, body)
			}

			if output == "json" {
				fmt.Println("{}")
			} else {
				fmt.Printf("Disabled key for tenant %q (generation: %d)\n", tenantID, generation)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tenantID, "tenant", "", "Tenant ID (required)")
	cmd.Flags().UintVar(&generation, "generation", 0, "Key generation (required)")
	cmd.MarkFlagRequired("tenant")
	cmd.MarkFlagRequired("generation")
	return cmd
}

func handleErrorResponse(statusCode int, body []byte) error {
	var errResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&errResp); err == nil && errResp.Message != "" {
		return fmt.Errorf("Error: %s", errResp.Message)
	}
	return fmt.Errorf("Error: server returned status %d", statusCode)
}
