// Package helper contains all utility methods and types for Microservice.
package helper

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

var (
	// ErrNone is used where there is yet we have to provide error type to report.
	ErrNone = errors.New("no error")

	//ErrNotFound is used when a lookup operation didnt find any resource.
	ErrNotFound = errors.New("not found")

	//ErrNotImplemented is used for operations not implemented yet.
	ErrNotImplemented = errors.New("not implemented")

	//ErrOperationNotSupported operation not supported
	ErrOperationNotSupported = errors.New("operation not supported")

	//ErrActionNotSupported action not supported
	ErrActionNotSupported = errors.New("action not supported")

	//ErrOperationFailed operation failed
	ErrOperationFailed = errors.New("operation failed")

	//ErrAWSConnectionNotInitialized AWSConnection not initialized
	ErrAWSConnectionNotInitialized = errors.New("AWSConnection not initialized")

	//ErrVaultUnsealedButInStandby vault Instance is in standby mode
	ErrVaultUnsealedButInStandby = errors.New("vault Instance is in standby mode, it wont serve requests")

	//ErrVaultSealedOrInErrorState vault is sealed or in an error state
	ErrVaultSealedOrInErrorState = errors.New("vault is sealed or in an error state")

	//ErrVaultNotInitialized Vault is not initialized
	ErrVaultNotInitialized = errors.New("vault is not initialized")

	//ErrVaultPingUnexpectedResponseCode Vault returned unexpected response code for health check
	ErrVaultPingUnexpectedResponseCode = errors.New("vault returned unexpected response code for health check")

	//ErrVaultAuthenticationFailed approle authentication with Vault failed.
	ErrVaultAuthenticationFailed = errors.New("approle authentication with Vault failed")

	//ErrVaultFailToEnableAWSSecretsEngine failed to enable Vault's AWS secrets engine
	ErrVaultFailToEnableAWSSecretsEngine = errors.New("failed to enable Vault's AWS secrets engine")

	//ErrVaultFailToConfigureAWSSecretsEngine failed to enable Vault's AWS secrets engine
	ErrVaultFailToConfigureAWSSecretsEngine = errors.New("failed to configure Vault's AWS secrets engine")

	//ErrAWSConnectionTestFailed AWS Connection Test Failed
	ErrAWSConnectionTestFailed = errors.New("AWS Connection Test Failed")

	//ErrVaultFailToDisableAWSSecretsEngine failed to enable Vault's AWS secrets engine
	ErrVaultFailToDisableAWSSecretsEngine = errors.New("failed to disable Vault's AWS secrets engine")
)

// ErrorTypeEnum is the type enum log dictionary for microservice.
type ErrorTypeEnum int

const (
	//ErrorNone represents no error.
	ErrorNone ErrorTypeEnum = iota

	//ErrorConnectionIDInvalid represents invalid connectionid.
	ErrorConnectionIDInvalid

	//ErrorResourceNotFound represents resource not found.
	ErrorResourceNotFound

	//ErrorInvalidValueForLimit represents invalid value for limit.
	ErrorInvalidValueForLimit

	//ErrorLimitMustBeGtZero represents limit must be great than zero.
	ErrorLimitMustBeGtZero

	//ErrorInvalidValueForSkip represents invalid value for skip.
	ErrorInvalidValueForSkip

	//ErrorSkipMustBeGtZero represents skip must be greater than zero.
	ErrorSkipMustBeGtZero

	//ErrorDatastoreRetrievalFailed represents datastore retrieval failed.
	ErrorDatastoreRetrievalFailed

	//ErrorDatalayerConversionFailed represents data layer conversion failed.
	ErrorDatalayerConversionFailed

	//ErrorDatastoreSaveFailed represents datastore save failed.
	ErrorDatastoreSaveFailed

	//ErrorInvalidJSONSchemaForParameter represents invalid json schema for parmeter.
	ErrorInvalidJSONSchemaForParameter

	//ErrorInvalidConnectionType represents error message for invalid connection type.
	ErrorInvalidConnectionType

	//ErrorDatastoreDeleteFailed represents error message for datastore delete failed.
	ErrorDatastoreDeleteFailed

	//ErrorConnectionTypeUpdateNotAllowed represents error message for connection type update not allowed.
	ErrorConnectionTypeUpdateNotAllowed

	//ErrorAWSConnectionPatchInvalidValueForTitle represents error message for invalid value for Title.
	ErrorAWSConnectionPatchInvalidValueForTitle

	//ErrorAWSConnectionPatchInvalidValueForDescription represents error message for invalid value for Description.
	ErrorAWSConnectionPatchInvalidValueForDescription

	//ErrorAWSConnectionPatchInvalidValueForURL represents error message for invalid value for URL.
	ErrorAWSConnectionPatchInvalidValueForAccessKey

	//ErrorAWSConnectionPatchInvalidValueForUsername represents error message for invalid value for Username.
	ErrorAWSConnectionPatchInvalidValueForSecretAccessKey

	//ErrorAWSConnectionPatchInvalidValueForPassword represents error message for invalid value for Password.
	ErrorAWSConnectionPatchInvalidValueForRegion

	//ErrorAWSConnectionPatchInvalidValueForProjectID represents error message for invalid value for ProjectID.
	ErrorAWSConnectionPatchInvalidValueForDefaultLeaseTTL

	//ErrorAWSConnectionPatchInvalidValueForIssueTypeID represents error message for invalid value for IssueTypeID.
	ErrorAWSConnectionPatchInvalidValueForMaxLeaseTTL

	//ErrorDatastoreNotAvailable represents error message for datastore not available.
	ErrorDatastoreNotAvailable

	//ErrorJSONEncodingFailed represents error message for json encoding failed.
	ErrorJSONEncodingFailed

	//ErrorHTTPServerShutdownFailed represents error message for HTTP server shutdown failed.
	ErrorHTTPServerShutdownFailed

	//ErrorAWSConnectionPatchInvalidValueForConnectionType represents error message for invalid value for connectiontype.
	ErrorAWSConnectionPatchInvalidValueForConnectionType

	//ErrorDatastoreConnectionCloseFailed represents failure to close datastore connection.
	ErrorDatastoreConnectionCloseFailed

	//ErrorDatastoreFailedToCreateDB represents failure to create database in datastore.
	ErrorDatastoreFailedToCreateDB

	//InfoHandlingRequest represents info message for handling request.
	InfoHandlingRequest

	//InfoDemoServerConnectionManagerStatusUP represents info message for connection manager status down.
	InfoDemoServerConnectionManagerStatusUP

	//InfoDemoServerConnectionManagerStatusDOWN represents info message for connection manager status down.
	InfoDemoServerConnectionManagerStatusDOWN

	//DebugAWSConnectionTestFailed represents debug message for AWS connection test failed.
	DebugAWSConnectionTestFailed

	//DebugDatastoreConnectionUP represents debug message for datastore connection up.
	DebugDatastoreConnectionUP

	//ErrorVaultNotAvailable represents error message for Vault not available.
	ErrorVaultNotAvailable

	//ErrorVaultAuthenticationFailed represents error message for client failed to authenticate with Vault.
	ErrorVaultAuthenticationFailed

	//ErrorVaultTLSConfigurationFailed represents error message for client failed to configure TLS for connection.
	ErrorVaultTLSConfigurationFailed

	//ErrorVaultAWSEngineFailed represents error message for request to Vault to enable new AWS Engine failed.
	ErrorVaultAWSEngineFailed

	//ErrorVaultLoadFailed represents load from vault failed.
	ErrorVaultLoadFailed

	//ErrorVaultDeleteFailed represents delete from vault failed.
	ErrorVaultDeleteFailed
)

// Error represent the details of error occurred.
type Error struct {
	Code        string `json:"errorCode"`
	Description string `json:"errorDescription"`
	Help        string `json:"errorHelp"`
}

// ErrorDictionary represents log dictionary for microservice.
var ErrorDictionary = map[ErrorTypeEnum]Error{
	InfoHandlingRequest:                       {"ConnectionManager_Info_000001", "Handling Request", ""},
	InfoDemoServerConnectionManagerStatusUP:   {"ConnectionManager_Info_000002", "UP", ""},
	InfoDemoServerConnectionManagerStatusDOWN: {"ConnectionManager_Info_000003", "DOWN", ""},

	DebugAWSConnectionTestFailed: {"ConnectionManager_Debug_000001", "AWSConnection Test Failed", ""},
	DebugDatastoreConnectionUP:   {"ConnectionManager_Debug_000002", "Datastore connection UP", ""},

	ErrorNone:                                             {"ConnectionManager_Err_000000", "No error", ""},
	ErrorConnectionIDInvalid:                              {"ConnectionManager_Err_000001", "ConnectionID is Invalid", ""},
	ErrorResourceNotFound:                                 {"ConnectionManager_Err_000002", "Resource not found", ""},
	ErrorInvalidValueForLimit:                             {"ConnectionManager_Err_000003", "Invalid value for Limit parameter", ""},
	ErrorLimitMustBeGtZero:                                {"ConnectionManager_Err_000004", "Limit is expected to be greater than or equal to 0 when present", ""},
	ErrorInvalidValueForSkip:                              {"ConnectionManager_Err_000005", "Invalid value for Skip parameter", ""},
	ErrorSkipMustBeGtZero:                                 {"ConnectionManager_Err_000006", "Skip is expected to be greater than or equal to 0 when present", ""},
	ErrorDatastoreRetrievalFailed:                         {"ConnectionManager_Err_000007", "Failed to retrieve from datastore", ""},
	ErrorDatalayerConversionFailed:                        {"ConnectionManager_Err_000008", "Failed to convert datastore document to object", ""},
	ErrorDatastoreSaveFailed:                              {"ConnectionManager_Err_000009", "Failed to save resource in datastore", ""},
	ErrorInvalidJSONSchemaForParameter:                    {"ConnectionManager_Err_000010", "Invalid JSON Schema for parameter passed", ""},
	ErrorInvalidConnectionType:                            {"ConnectionManager_Err_000011", "Invalid connection type", ""},
	ErrorDatastoreDeleteFailed:                            {"ConnectionManager_Err_000012", "Failed to delete resource from datastore", ""},
	ErrorConnectionTypeUpdateNotAllowed:                   {"ConnectionManager_Err_000013", "ConnectionType attribute can not be patched", ""},
	ErrorAWSConnectionPatchInvalidValueForTitle:           {"ConnectionManager_Err_000014", "Invalid value for Title. string expected", ""},
	ErrorAWSConnectionPatchInvalidValueForDescription:     {"ConnectionManager_Err_000015", "Invalid value for description. string expected", ""},
	ErrorAWSConnectionPatchInvalidValueForAccessKey:       {"ConnectionManager_Err_000016", "Invalid value for AccessKey. string expected", ""},
	ErrorAWSConnectionPatchInvalidValueForSecretAccessKey: {"ConnectionManager_Err_000017", "Invalid value for SecretAccessKey. string expected", ""},
	ErrorAWSConnectionPatchInvalidValueForRegion:          {"ConnectionManager_Err_000018", "Invalid value for region. string expected", ""},
	ErrorAWSConnectionPatchInvalidValueForDefaultLeaseTTL: {"ConnectionManager_Err_000021", "Invalid value for DefaultLeaseTTL. int expected", ""},
	ErrorAWSConnectionPatchInvalidValueForMaxLeaseTTL:     {"ConnectionManager_Err_000022", "Invalid value for MaxLeaseTTL. int expected", ""},
	ErrorDatastoreNotAvailable:                            {"ConnectionManager_Err_000023", "Datastore connection down", ""},
	ErrorJSONEncodingFailed:                               {"ConnectionManager_Err_000024", "JSON Ecoding Failed", ""},
	ErrorHTTPServerShutdownFailed:                         {"ConnectionManager_Err_000025", "HTTP Server Shutdown failed", ""},
	ErrorAWSConnectionPatchInvalidValueForConnectionType:  {"ConnectionManager_Err_000026", "Invalid value for connectiontype. string expected", ""},
	ErrorDatastoreConnectionCloseFailed:                   {"ConnectionManager_Err_000027", "Failed to close datastore connection", ""},
	ErrorDatastoreFailedToCreateDB:                        {"ConnectionManager_Err_000028", "Failed to create database in datastore", ""},
	ErrorVaultNotAvailable:                                {"ConnectionManager_Err_000029", "Vault connection down", ""},
	ErrorVaultAuthenticationFailed:                        {"ConnectionManager_Err_000030", "Vault authentication failed", ""},
	ErrorVaultTLSConfigurationFailed:                      {"ConnectionManager_Err_000031", "Vault TLS Configuration failed", ""},
}

// ErrorResponse represents information returned by Microservice endpoints in case that was an error
// in normal execution flow.
// swagger:model
type ErrorResponse struct {
	// Date and time when this error occurred
	//
	// required: true
	Timestamp string `json:"timestamp"`

	// HTTP status code
	//
	// required: true
	Status int `json:"status"`

	// Microservice specific error code
	//
	// required: true
	ErrorCode string `json:"errorCode"`

	// Microservice specific error code's description
	//
	// required: true
	ErrorDescription string `json:"errorDescription"`

	// Any additional contextual message for error that Microservice may want to provide
	//
	// required: false
	ErrorAdditionalInfo string `json:"errorAdditionalInfo"`

	// Link to documentation for errorcode for more details
	//
	// required: false
	ErrorHelp string `json:"errorHelp"`

	// Microservice endpoint that was called
	//
	// required: true
	Endpoint string `json:"endpoint"`

	// HTTP method (GET, POST,...) for request
	//
	// required: true
	Method string `json:"method"`

	// ID to track API call
	//
	// required: true
	RequestID string `json:"requestID"`
}

// GetErrorResponse prepares error response to be returned to caller.
func GetErrorResponse(status int, err ErrorTypeEnum, r *http.Request, requestid string) ErrorResponse {
	return ErrorResponse{
		Timestamp:           time.Now().String(),
		Status:              status,
		ErrorCode:           ErrorDictionary[err].Code,
		ErrorDescription:    ErrorDictionary[err].Description,
		ErrorAdditionalInfo: "",
		ErrorHelp:           ErrorDictionary[err].Help,
		Endpoint:            r.URL.EscapedPath(),
		Method:              r.Method,
		RequestID:           requestid,
	}
}

// GetErrorResponseWithAdditionalInfo prepares error response with additional original error contextual message to be returned to caller.
func GetErrorResponseWithAdditionalInfo(status int, err ErrorTypeEnum, r *http.Request, requestid string, e error) ErrorResponse {
	return ErrorResponse{
		Timestamp:           time.Now().String(),
		Status:              status,
		ErrorCode:           ErrorDictionary[err].Code,
		ErrorDescription:    ErrorDictionary[err].Description,
		ErrorAdditionalInfo: e.Error(),
		ErrorHelp:           ErrorDictionary[err].Help,
		Endpoint:            r.URL.EscapedPath(),
		Method:              r.Method,
		RequestID:           requestid,
	}
}

// LogDebug logs debug structure log message.
func LogDebug(cl *slog.Logger, err ErrorTypeEnum, e error) {
	cl.Debug(ErrorDictionary[err].Description,
		slog.String("code", ErrorDictionary[err].Code),
		slog.String("description", ErrorDictionary[err].Description),
		slog.String("originalError", e.Error()))
}

// LogError logs error structure log message.
func LogError(cl *slog.Logger, err ErrorTypeEnum, e error) {
	cl.Error(ErrorDictionary[err].Description,
		slog.String("code", ErrorDictionary[err].Code),
		slog.String("description", ErrorDictionary[err].Description),
		slog.String("originalError", e.Error()))
}

// LogInfo logs info structure log message.
func LogInfo(cl *slog.Logger, err ErrorTypeEnum, e error) {
	cl.Info(ErrorDictionary[err].Description,
		slog.String("code", ErrorDictionary[err].Code),
		slog.String("description", ErrorDictionary[err].Description),
		slog.String("originalError", e.Error()))
}

// ReturnError prepares error json to be returned to caller.
func ReturnError(cl *slog.Logger, status int, err ErrorTypeEnum, requestid string, r *http.Request, rw *http.ResponseWriter) {
	errorResponse := GetErrorResponse(
		status,
		err,
		r,
		requestid)

	http.Error(*rw, "", status)

	e := json.NewEncoder(*rw).Encode(errorResponse)

	if e != nil {
		LogError(cl, ErrorJSONEncodingFailed, e)
	}
}

// ReturnErrorWithAdditionalInfo prepares error json to be returned to caller with additional context.
func ReturnErrorWithAdditionalInfo(cl *slog.Logger, status int, err ErrorTypeEnum, requestid string, r *http.Request, rw *http.ResponseWriter, e error) {
	errorResponse := GetErrorResponseWithAdditionalInfo(
		status,
		err,
		r,
		requestid,
		e)

	http.Error(*rw, "", http.StatusBadRequest)

	e = json.NewEncoder(*rw).Encode(errorResponse)

	if e != nil {
		LogError(cl, ErrorJSONEncodingFailed, e)
	}
}
