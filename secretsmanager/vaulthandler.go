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
	"DemoServer_ConnectionManager/data"
	"DemoServer_ConnectionManager/helper"

	_ "github.com/lib/pq"
)

type VaultHandler struct {
	c            *configuration.Config
	l            *slog.Logger
	hc           *http.Client
	vaultAddress string
}

type vaultAWSConfig struct {
	Data struct {
		AccessKey       string   `json:"access_key"`
		Role            string   `json:"role"`
		Region          string   `json:"region"`
		DefaultLeaseTTL int      `json:"default_lease_ttl"`
		MaxLeaseTTL     int      `json:"max_lease_ttl"`
		CredentialType  string   `json:"credential_type"`
		PolicyARNs      []string `json:"policy_arns"`
	} `json:"data"`
}

type vaultAWSConfigRolesList struct {
	Data struct {
		Keys []string `json:"keys"`
	} `json:"data"`
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

func (vh *VaultHandler) getAWSSecretsEngineConfig(token string, path string, r *vaultAWSConfig) error {
	// Prepare the request URL
	url := fmt.Sprintf("%s/v1/%s/config/root", vh.vaultAddress, path)

	// Create the request with appropriate headers
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// Add the Vault token in the Authorization header
	req.Header.Add("X-Vault-Token", token)

	// Send the request
	resp, err := vh.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check if the response status code is OK (200)
	if resp.StatusCode != http.StatusOK {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, r)
	if err != nil {
		return err
	}

	return err
}

func (vh *VaultHandler) getAWSSecretsEngineLease(token string, path string, r *vaultAWSConfig) error {
	// Prepare the request URL
	url := fmt.Sprintf("%s/v1/sys/mounts/%s/tune", vh.vaultAddress, path)

	// Create the request with appropriate headers
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// Add the Vault token in the Authorization header
	req.Header.Add("X-Vault-Token", token)

	// Send the request
	resp, err := vh.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check if the response status code is OK (200)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to get AWS config, status: %s, response: %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, r)
	if err != nil {
		return err
	}

	return err
}

func (vh *VaultHandler) getAWSSecretsEngineRole(token string, path string, r *vaultAWSConfig) error {
	// Prepare the request URL
	url := fmt.Sprintf("%s/v1/%s/roles/%s", vh.vaultAddress, path, r.Data.Role)

	// Create the request with appropriate headers
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// Add the Vault token in the Authorization header
	req.Header.Add("X-Vault-Token", token)

	// Send the request
	resp, err := vh.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check if the response status code is OK (200)
	if resp.StatusCode != http.StatusOK {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, r)
	if err != nil {
		return err
	}

	return err
}

func (vh *VaultHandler) getAWSSecretsEngineRoleName(token string, path string, r *vaultAWSConfig) error {
	// Prepare the request URL
	url := fmt.Sprintf("%s/v1/%s/roles", vh.vaultAddress, path)

	// Create the request with appropriate headers
	req, err := http.NewRequest("LIST", url, nil)
	if err != nil {
		return err
	}

	// Add the Vault token in the Authorization header
	req.Header.Add("X-Vault-Token", token)

	// Send the request
	resp, err := vh.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check if the response status code is OK (200)
	if resp.StatusCode != http.StatusOK {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var rl vaultAWSConfigRolesList

	err = json.Unmarshal(body, &rl)
	if err != nil {
		return err
	}

	r.Data.Role = rl.Data.Keys[0]
	return err
}

func (vh *VaultHandler) GetAWSSecretsEngine(c *data.AWSConnection) error {
	// Parse the response body into the struct
	var awsConfig vaultAWSConfig

	token, err := vh.GetToken()
	if err != nil {
		return err
	}

	err = vh.getAWSSecretsEngineConfig(token, c.VaultPath, &awsConfig)
	if err != nil {
		return err
	}

	err = vh.getAWSSecretsEngineLease(token, c.VaultPath, &awsConfig)
	if err != nil {
		return err
	}

	err = vh.getAWSSecretsEngineRoleName(token, c.VaultPath, &awsConfig)
	if err != nil {
		return err
	}

	err = vh.getAWSSecretsEngineRole(token, c.VaultPath, &awsConfig)
	if err != nil {
		return err
	}

	c.AccessKey = awsConfig.Data.AccessKey
	c.DefaultLeaseTTL = strconv.Itoa(awsConfig.Data.DefaultLeaseTTL) + "s"
	c.DefaultRegion = awsConfig.Data.Region
	c.MaxLeaseTTL = strconv.Itoa(awsConfig.Data.MaxLeaseTTL) + "s"
	c.PolicyARNs = awsConfig.Data.PolicyARNs
	c.RoleName = awsConfig.Data.Role
	c.CredentialType = awsConfig.Data.CredentialType

	return nil
}

func (vh *VaultHandler) AddAWSSecretsEngine(c *data.AWSConnection) error {
	token, err := vh.GetToken()
	if err != nil {
		return err
	}

	err = vh.enableAWSSecretsEngine(token, c.VaultPath)
	if err != nil {
		return err
	}

	err = vh.configureAWSSecretsEngine(token, c.VaultPath, c.DefaultLeaseTTL, c.MaxLeaseTTL)
	if err != nil {
		return err
	}

	err = vh.configureAWSRootCredentials(token, c.VaultPath, c.AccessKey, c.SecretAccessKey, c.DefaultRegion)
	if err != nil {
		return err
	}

	err = vh.configureAWSIAMRole(token, c.VaultPath, c.RoleName, c.PolicyARNs, c.CredentialType)
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

func (vh *VaultHandler) configureAWSIAMRole(token string, path string, roleName string, policyARNs []string, credentialType string) error {
	url := fmt.Sprintf("%s/v1/%s/roles/%s", vh.vaultAddress, path, roleName)

	data := map[string]interface{}{
		//"policy_arns":     "[" + strings.Join(policyARNs, ", ") + "]",
		"policy_arns":     policyARNs,
		"credential_type": credentialType,
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
