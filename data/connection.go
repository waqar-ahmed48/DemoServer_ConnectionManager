package data

import (
	"time"

	"github.com/google/uuid"
)

// Connection represents generic Connection attributes which are allowed in POST request.
//
// swagger:model
type ConnectionPostWrapper struct {
	// User friendly name for Connection
	// required: true
	Name string `json:"name" validate:"required" gorm:"index;not null;unique"`

	// Description of Connection
	// required: false
	Description string `json:"description" gorm:"index"`

	// Type of connection.
	// required: true
	ConnectionType ConnectionTypeEnum `json:"connectiontype" gorm:"index;not null"`

	// Latest connectivity test result. 0 = Failed. 1 = Successful
	// required: false
	TestSuccessful int `json:"testsuccessful"`

	// Descriptive error for latest connectivity test
	// required: false
	TestError string `json:"testerror"`

	// Date and time of latest connectivity test whether it was successful or not
	// required: false
	TestedOn string `json:"testedon"`

	// Date and time of latest successful connectivity test
	// required: false
	LastSuccessfulTest string `json:"lastsuccessfultest"`
}

// Connection represents generic Connection attributes which are allowed in PATCH request.
//
// swagger:model
type ConnectionPatchWrapper struct {
	// User friendly name for Connection
	// required: true
	Name string `json:"name" validate:"required" gorm:"index;not null;unique"`

	// Description of Connection
	// required: false
	Description string `json:"description" gorm:"index"`
}

// Connection represents generic Connection resource returned by Microservice endpoints
// Different types of connections (for example: AWSConnection) contains an object of
// Connection inside.
//
// swagger:model
type Connection struct {
	ID        uuid.UUID `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdat" gorm:"autoCreateTime;index;not null"`
	UpdatedAt time.Time `json:"updatedat" gorm:"autoUpdateTime;index"`

	// User friendly name for Connection
	// required: true
	Name string `json:"name" validate:"required" gorm:"index;not null;unique"`

	// Description of Connection
	// required: false
	Description string `json:"description" gorm:"index"`

	// Type of connection.
	// required: true
	ConnectionType ConnectionTypeEnum `json:"connectiontype" gorm:"index;not null"`

	// Latest connectivity test result. 0 = Failed. 1 = Successful
	// required: false
	TestSuccessful int `json:"testsuccessful"`

	// Descriptive error for latest connectivity test
	// required: false
	TestError string `json:"testerror"`

	// Date and time of latest connectivity test whether it was successful or not
	// required: false
	TestedOn string `json:"testedon"`

	// Date and time of latest successful connectivity test
	// required: false
	LastSuccessfulTest string `json:"lastsuccessfultest"`
}

// ConnectionsResponse represents generic Connection attributes which are returned in response of GET on connections endpoint.
//
// swagger:model
type ConnectionsResponse struct {
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
	Connections []Connection `json:"connections"`
}

func (c *Connection) SetTestFailed(e string) {
	c.TestSuccessful = 0
	c.TestedOn = time.Now().UTC().String()
	c.TestError = e
}

func (c *Connection) SetTestPassed() {
	c.TestSuccessful = 1
	c.TestedOn = time.Now().UTC().String()
	c.LastSuccessfulTest = time.Now().UTC().String()
	c.TestError = ""
}

func (c *Connection) ResetTestStatus() {
	c.TestSuccessful = 0
	c.TestError = ""
}
