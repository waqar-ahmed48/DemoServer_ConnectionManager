package data

import (
	"DemoServer_ConnectionManager/configuration"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-playground/validator"
	"github.com/google/uuid"
)

// AWSConnectionPostWrapper represents AWSConnection attributes for POST request body schema.
// swagger:model
type AWSConnectionPostWrapper struct {
	Connection ConnectionPostWrapper `json:"connection" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	// AccessKey for AWS Account
	// required: true
	AccessKey string `json:"accesskey" validate:"required" gorm:"-"`

	// SecretAccessKey for AWS Account
	// required: true
	SecretAccessKey string `json:"secretaccesskey" validate:"required" gorm:"-"`

	// DefaultRegion for AWS Account
	// required: false
	DefaultRegion string `json:"default_region" gorm:"-"`

	// DefaultRegion for AWS Account
	// required: false
	DefaultLeaseTTL string `json:"default_lease_ttl" gorm:"-"`

	// DefaultRegion for AWS Account
	// required: false
	MaxLeaseTTL string `json:"max_lease_ttl" gorm:"-"`

	// RoleName RoleName for AWS Account
	// required: true
	RoleName string `json:"role_name" validate:"required" gorm:"-"`

	// CredentialType CredentialType for AWS Account Role
	// required: true
	CredentialType string `json:"credential_type" validate:"required,oneof=iam_user" gorm:"-"`

	// PolicyARNs PolicyARNs for AWS Account
	// required: true
	PolicyARNs []string `json:"policy_arns" validate:"required" gorm:"-"`
}

// AWSConnectionPatchWrapper represents AWSConnection attributes for PATCH request body schema.
// swagger:model
type AWSConnectionPatchWrapper struct {
	Connection ConnectionPatchWrapper `json:"connection" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	// AccessKey for AWS Account
	// required: true
	AccessKey string `json:"accesskey" validate:"required" gorm:"-"`

	// SecretAccessKey for AWS Account
	// required: true
	SecretAccessKey string `json:"secretaccesskey" validate:"required" gorm:"-"`

	// DefaultRegion for AWS Account
	// required: false
	DefaultRegion string `json:"default_region" gorm:"-"`

	// DefaultRegion for AWS Account
	// required: false
	DefaultLeaseTTL string `json:"default_lease_ttl" gorm:"-"`

	// DefaultRegion for AWS Account
	// required: false
	MaxLeaseTTL string `json:"max_lease_ttl" gorm:"-"`

	// CredentialType CredentialType for AWS Account Role
	// required: true
	CredentialType string `json:"credential_type" validate:"required,oneof=iam_user" gorm:"-"`

	// PolicyARNs PolicyARNs for AWS Account
	// required: true
	PolicyARNs []string `json:"policy_arns" validate:"required" gorm:"-"`
}

// AWSConnection represents AWSConnection resource serialized by Microservice endpoints
// swagger:model
type AWSConnection struct {
	ID           uuid.UUID  `json:"id" gorm:"primaryKey"`
	CreatedAt    time.Time  `json:"createdat" gorm:"autoCreateTime;index;not null"`
	UpdatedAt    time.Time  `json:"updatedat" gorm:"autoUpdateTime;index"`
	ConnectionID uuid.UUID  `json:"connectionid" gorm:"not null;index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Connection   Connection `json:"connection" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	// VaultPath for AWS Account
	// required: true
	VaultPath string `json:"vaultpath" validate:"required" gorm:"not null"`

	// AccessKey for AWS Account
	// required: true
	AccessKey string `json:"accesskey" validate:"required" gorm:"-"`

	// SecretAccessKey for AWS Account
	// required: true
	SecretAccessKey string `json:"secretaccesskey" validate:"required" gorm:"-"`

	// DefaultRegion for AWS Account
	// required: false
	DefaultRegion string `json:"default_region" gorm:"-"`

	// DefaultRegion for AWS Account
	// required: false
	DefaultLeaseTTL string `json:"default_lease_ttl" gorm:"-"`

	// DefaultRegion for AWS Account
	// required: false
	MaxLeaseTTL string `json:"max_lease_ttl" gorm:"-"`

	// RoleName RoleName for AWS Account
	// required: true
	RoleName string `json:"role_name" validate:"required" gorm:"-"`

	// CredentialType CredentialType for AWS Account Role
	// required: true
	CredentialType string `json:"credential_type" validate:"required,oneof=iam_user" gorm:"-"`

	// PolicyARNs PolicyARNs for AWS Account
	// required: true
	PolicyARNs []string `json:"policy_arns" validate:"required" gorm:"-"`
}

// AWSConnectionResponseWrapper represents limited information AWSConnection resource returned by Post, Get and List endpoints
// swagger:model
type AWSConnectionResponseWrapper struct {
	ID           uuid.UUID  `json:"id" gorm:"primaryKey"`
	CreatedAt    time.Time  `json:"createdat" gorm:"autoCreateTime;index;not null"`
	UpdatedAt    time.Time  `json:"updatedat" gorm:"autoUpdateTime;index"`
	ConnectionID uuid.UUID  `json:"connectionid" gorm:"not null;index"`
	Connection   Connection `json:"connection" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	// AccessKey for AWS Account
	// required: true
	AccessKey string `json:"accesskey" validate:"required" gorm:"-"`

	// DefaultRegion for AWS Account
	// required: false
	DefaultRegion string `json:"default_region" gorm:"-"`

	// DefaultRegion for AWS Account
	// required: false
	DefaultLeaseTTL string `json:"default_lease_ttl" gorm:"-"`

	// DefaultRegion for AWS Account
	// required: false
	MaxLeaseTTL string `json:"max_lease_ttl" gorm:"-"`

	// RoleName RoleName for AWS Account
	// required: true
	RoleName string `json:"role_name" validate:"required" gorm:"-"`

	// CredentialType CredentialType for AWS Account Role
	// required: true
	CredentialType string `json:"credential_type" validate:"required,oneof=iam_user" gorm:"-"`

	// PolicyARNs PolicyARNs for AWS Account
	// required: true
	PolicyARNs []string `json:"policy_arns" validate:"required" gorm:"-"`
}

// DeleteAWSConnectionResponse represents Response schema for DELETE - DeleteAWSConnection
// swagger:model
type DeleteAWSConnectionResponse struct {
	// Descriptive human readable HTTP status of delete operation.
	// in: status
	Status string `json:"status"`

	// HTTP status code for delete operation.
	// in: statusCode
	StatusCode int `json:"statusCode"`
}

// TestAWSConnectionResponse Response schema for GET - TestAWSConnection
// swagger:model
type TestAWSConnectionResponse struct {
	// connectionid for AWSConnection which was tested.
	// in: id
	ID string `json:"id"`

	// test status descriptive human readable message.
	// in: test_status
	TestStatus string `json:"testStatus"`

	// test_status_code. 1 = connectivity test successful. 0 = connectivity test failed.
	// in: test_status_code
	TestStatusCode int `json:"testStatusCode"`
}

// AWSConnectionsResponse represents AWS Connection attributes which are returned in response of GET on connections/aws endpoint.
// swagger:model
type AWSConnectionsResponse struct {
	// Number of skipped resources
	// required: true
	Skip int `json:"skip"`

	// Limit applied on resources returned
	// required: true
	Limit int `json:"limit"`

	// Total number of resources returned
	// required: true
	Total int `json:"total"`

	// Connection resource objects
	// required: true
	AWSConnections []AWSConnectionResponseWrapper `json:"awsconnections"`
}

type Connections []*AWSConnection

func NewAWSConnection(cfg *configuration.Config) *AWSConnection {
	var c AWSConnection

	c.ID = uuid.New()
	c.Connection.ID = uuid.New()
	c.ConnectionID = c.Connection.ID
	c.Connection.ConnectionType = NoConnectionType
	c.VaultPath = cfg.Vault.PathPrefix + "/aws_" + c.ID.String()

	return &c
}

func InitAWSConnection(id string, cfg *configuration.Config) *AWSConnection {
	var c AWSConnection

	c.ID, _ = uuid.Parse(id)

	return &c
}

func (c *AWSConnection) GetNewID() {
	c.ID = uuid.New()
}

func (c *AWSConnection) FromJSON(r io.Reader) error {
	e := json.NewDecoder(r)
	err := e.Decode(c)

	return err
}

func (c *AWSConnection) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

func (c *AWSConnection) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(c)
}

func (c *AWSConnection) Initialize() *http.Client {
	bool_insecureallowed := true
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: bool_insecureallowed}}
	return &http.Client{Transport: tr}
}
