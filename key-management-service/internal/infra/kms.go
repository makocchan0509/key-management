package infra

import (
	"context"
	"fmt"
	"os"

	kms "cloud.google.com/go/kms/apiv1"
	kmspb "cloud.google.com/go/kms/apiv1/kmspb"
)

// KMSClient はCloud KMSクライアントをラップする。
type KMSClient struct {
	client  *kms.KeyManagementClient
	keyName string
}

// NewKMSClient は環境変数KMS_KEY_NAMEからキー名を取得してKMSClientを生成する。
func NewKMSClient(ctx context.Context) (*KMSClient, error) {
	keyName := os.Getenv("KMS_KEY_NAME")
	if keyName == "" {
		return nil, fmt.Errorf("KMS_KEY_NAME environment variable is required")
	}

	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating KMS client: %w", err)
	}

	return &KMSClient{
		client:  client,
		keyName: keyName,
	}, nil
}

// Encrypt は平文をCloud KMSで暗号化する。
func (c *KMSClient) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	req := &kmspb.EncryptRequest{
		Name:      c.keyName,
		Plaintext: plaintext,
	}
	resp, err := c.client.Encrypt(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("encrypting: %w", err)
	}
	return resp.Ciphertext, nil
}

// Decrypt は暗号文をCloud KMSで復号する。
func (c *KMSClient) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	req := &kmspb.DecryptRequest{
		Name:       c.keyName,
		Ciphertext: ciphertext,
	}
	resp, err := c.client.Decrypt(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}
	return resp.Plaintext, nil
}

// Close はKMSクライアントを閉じる。
func (c *KMSClient) Close() error {
	return c.client.Close()
}
