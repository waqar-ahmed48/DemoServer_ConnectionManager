package secretsmanager

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"DemoServer_ConnectionManager/configuration"
	"DemoServer_ConnectionManager/helper"

	_ "github.com/lib/pq"
)

type VaultHandler struct {
	c            *configuration.Config
	l            *slog.Logger
	hc           *http.Client
	vaultAddress string
}

func (vh *VaultHandler) GetToken() (string, error) {

	// Create the authentication payload
	authData := map[string]string{
		"role_id":   vh.c.Vault.RoleID,
		"secret_id": vh.c.Vault.SecretID,
	}
	authDataJSON, err := json.Marshal(authData)
	if err != nil {
		return "", err
	}

	// Construct the authentication request
	url := fmt.Sprintf("%s/v1/%s", vh.vaultAddress, "auth/approle/login")
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(authDataJSON))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute the HTTP request
	resp, err := vh.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to enable secrets engine: %s", string(body))
		//return "", helper.ErrVaultAuthenticationFailed
	}

	// Parse the response
	var respData struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	return respData.Auth.ClientToken, nil
}

func NewVaultHandler(c *configuration.Config, l *slog.Logger) (*VaultHandler, error) {

	var vaultAddress string

	if c.Vault.HTTPS {
		vaultAddress += "https://"
	} else {
		vaultAddress += "http://"
	}
	vaultAddress += c.Vault.Host

	if c.Vault.Port != -1 {
		vaultAddress += ":" + strconv.Itoa(c.Vault.Port)
	}

	// Create a custom transport with TLS verification disabled
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: c.Vault.TLSSkipVerify}, // Set TLS verification according to requested configuration
	}

	// Create an HTTP client with the custom transport
	hc := &http.Client{
		Transport: transport,
	}

	vh := &VaultHandler{c, l, hc, vaultAddress}

	err := vh.Ping()
	if err != nil {
		helper.LogError(vh.l, helper.ErrorVaultNotAvailable, err)
		return nil, err
	}

	return vh, nil
}

func (vh *VaultHandler) AddAWSSecretsEngine(path string, accessKey string, secretAccessKey string, defaultTTL string, maxTTL string, defaultRegion string, roleName string, policyARN string) error {
	token, err := vh.GetToken()
	if err != nil {
		return err
	}

	err = vh.enableAWSSecretsEngine(token, path)
	if err != nil {
		return err
	}

	err = vh.configureAWSSecretsEngine(token, path, defaultTTL, maxTTL)
	if err != nil {
		return err
	}

	err = vh.configureAWSRootCredentials(token, path, accessKey, secretAccessKey, defaultRegion)
	if err != nil {
		return err
	}

	err = vh.configureAWSIAMRole(token, path, roleName, policyARN)
	if err != nil {
		return err
	}

	return nil
}

func (vh *VaultHandler) enableAWSSecretsEngine(token string, path string) error {

	url := fmt.Sprintf("%s/v1/sys/mounts/%s", vh.vaultAddress, path)
	data := map[string]interface{}{
		"type": "aws",
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("X-Vault-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := vh.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return helper.ErrVaultFailToEnableAWSSecretsEngine
	}

	return nil
}

func (vh *VaultHandler) configureAWSRootCredentials(token string, path string, accessKey string, secretKey string, defaultRegion string) error {
	url := fmt.Sprintf("%s/v1/%s/config/root", vh.vaultAddress, path)
	data := map[string]interface{}{
		"access_key": accessKey,
		"secret_key": secretKey,
		"region":     defaultRegion,
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("X-Vault-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := vh.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return helper.ErrVaultFailToConfigureAWSSecretsEngine
	}

	return nil
}

func (vh *VaultHandler) configureAWSSecretsEngine(token string, path string, defaultTTL string, maxTTL string) error {

	url := fmt.Sprintf("%s/v1/sys/mounts/%s/tune", vh.vaultAddress, path)
	data := map[string]interface{}{
		"default_lease_ttl": defaultTTL,
		"max_lease_ttl":     maxTTL,
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("X-Vault-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := vh.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return helper.ErrVaultFailToConfigureAWSSecretsEngine
	}

	return nil
}

func (vh *VaultHandler) configureAWSIAMRole(token string, path string, roleName string, PolicyARNs string) error {
	url := fmt.Sprintf("%s/v1/%s/roles/%s", vh.vaultAddress, path, roleName)

	/*
		// Define IAM role configuration
		roleConfig := RoleConfig{
			RoleName: "my-iam-role",
			Policy:   "arn:aws:iam::aws:policy/AdministratorAccess",
			AssumeRolePolicy: `{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Principal": { "AWS": "arn:aws:iam::123456789012:root" },
						"Action": "sts:AssumeRole"
					}
				]
			}`,
		}

		// Marshal the role configuration to JSON
		body, err := json.Marshal(roleConfig)
		if err != nil {
			log.Fatalf("Failed to marshal role config: %v", err)
		}*/

	data := map[string]interface{}{
		"RoleName":        roleName,
		"Policy":          PolicyARNs,
		"credential_type": "iam_user",
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("X-Vault-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := vh.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to configure IAM role: %s", string(body))
	}

	return nil
}

func (vh *VaultHandler) Ping() error {

	// Ping the Vault server by checking its health
	healthCheckURL := fmt.Sprintf("%s/v1/sys/health", vh.vaultAddress)

	resp, err := vh.hc.Get(healthCheckURL)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// Check the HTTP status code
	switch resp.StatusCode {
	case 200:
		return nil
	case 429:
		return helper.ErrVaultUnsealedButInStandby
	case 500:
		return helper.ErrVaultSealedOrInErrorState
	case 501:
		return helper.ErrVaultNotInitialized
	default:
		return helper.ErrVaultPingUnexpectedResponseCode
	}
}
