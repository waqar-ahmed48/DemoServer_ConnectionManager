package handlers

import (
	"DemoServer_ConnectionManager/configuration"
	"DemoServer_ConnectionManager/data"
	"DemoServer_ConnectionManager/datalayer"
	"DemoServer_ConnectionManager/helper"
	"DemoServer_ConnectionManager/secretsmanager"
	"DemoServer_ConnectionManager/utilities"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type KeyAWSConnectionRecord struct{}
type KeyAWSConnectionPatchParamsRecord struct{}

type AWSConnectionHandler struct {
	l          *slog.Logger
	cfg        *configuration.Config
	pd         *datalayer.PostgresDataSource
	vh         *secretsmanager.VaultHandler
	list_limit int
}

func NewAWSConnectionHandler(cfg *configuration.Config, l *slog.Logger, pd *datalayer.PostgresDataSource, vh *secretsmanager.VaultHandler) (*AWSConnectionHandler, error) {
	var c AWSConnectionHandler

	c.cfg = cfg
	c.l = l
	c.pd = pd
	c.list_limit = cfg.Server.ListLimit
	c.vh = vh

	return &c, nil
}

func (h *AWSConnectionHandler) GetAWSConnections(w http.ResponseWriter, r *http.Request) {

	// swagger:operation GET /connections/aws AWSConnection GetAWSConnections
	// List AWS Connections
	//
	// Endpoint: GET - /v1/connectionmgmt/connections/aws
	//
	// Description: Returns list of AWSConnection resources. Each AWSConnection resource
	// contains underlying generic Connection resource as well as AWSConnection
	// specific attributes.
	//
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: limit
	//   in: query
	//   description: maximum number of results to return.
	//   required: false
	//   type: integer
	//   format: int32
	// - name: skip
	//   in: query
	//   description: number of results to be skipped from beginning of list
	//   required: false
	//   type: integer
	//   format: int32
	// responses:
	//   '200':
	//     description: List of AWSConnection resources
	//     schema:
	//       type: array
	//       items:
	//         "$ref": "#/definitions/AWSConnection"
	//   '400':
	//     description: Issues with parameters or their value
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"
	//   '500':
	//     description: Internal server error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"
	//   default:
	//     description: unexpected error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	ctx, span, requestid, cl := utilities.SetupTraceAndLogger(r, w, h.l, utilities.GetFunctionName(), h.cfg.Server.PrefixMain)
	defer span.End()

	vars := r.URL.Query()
	limit := utilities.ParseQueryParam(vars, "limit", h.list_limit, h.cfg.DataLayer.MaxResults)
	skip := utilities.ParseQueryParam(vars, "skip", 0, math.MaxInt32)

	connections, err := h.fetchAWSConnections(limit, skip)
	if err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorDatastoreRetrievalFailed, err, requestid, r, &w, span)
		return
	}

	response, err := h.buildAWSConnectionsResponse(ctx, connections, limit, skip)
	if err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorVaultLoadFailed, err, requestid, r, &w, span)
		return
	}

	utilities.WriteResponse(w, cl, response, span)
}

func (h *AWSConnectionHandler) fetchAWSConnections(limit, skip int) ([]data.AWSConnection, error) {
	var connections []data.AWSConnection

	result := h.pd.RODB().
		Preload("Connection").
		Limit(limit).
		Offset(skip).
		Order("connections.name").
		Joins("LEFT JOIN connections ON connections.id = aws_connections.connection_id").
		Find(&connections)

	if result.Error != nil {
		return nil, result.Error
	}
	return connections, nil
}

func (h *AWSConnectionHandler) buildAWSConnectionsResponse(ctx context.Context, connections []data.AWSConnection, limit, skip int) (data.AWSConnectionsResponse, error) {
	response := data.AWSConnectionsResponse{
		Total: len(connections),
		Skip:  skip,
		Limit: limit,
	}

	if len(connections) == 0 {
		response.AWSConnections = []data.AWSConnectionResponseWrapper{}
		return response, nil
	}

	for _, conn := range connections {
		if err := h.vh.GetAWSSecretsEngine(&conn, ctx); err != nil {
			return response, err
		}

		var wrappedConn data.AWSConnectionResponseWrapper
		if err := utilities.CopyMatchingFields(conn, &wrappedConn); err != nil {
			return response, err
		}
		response.AWSConnections = append(response.AWSConnections, wrappedConn)
	}

	return response, nil
}

func (h AWSConnectionHandler) MiddlewareValidateAWSConnectionsGet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		_, span, requestid, cl := utilities.SetupTraceAndLogger(r, rw, h.l, utilities.GetFunctionName(), h.cfg.Server.PrefixMain)
		defer span.End()

		vars := r.URL.Query()

		// Validate limit parameter
		if err := utilities.ValidateQueryParam(vars.Get("limit"), 1, true, cl, r, rw, span, requestid, helper.ErrorInvalidValueForLimit); err != nil {
			return
		}

		// Validate skip parameter
		if err := utilities.ValidateQueryParam(vars.Get("skip"), 0, false, cl, r, rw, span, requestid, helper.ErrorInvalidValueForSkip); err != nil {
			return
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(rw, r)
	})
}

// GetAWSConnection returns AWSConnection resource based on connectionid parameter
func (h *AWSConnectionHandler) GetAWSConnection(w http.ResponseWriter, r *http.Request) {

	// swagger:operation GET /connection AWSConnection GetAWSConnection
	// Retrieve AWS Connection
	//
	// Endpoint: GET - /v1/connectionmgmt/connection/aws/{connectionid}
	//
	// Description: Returns AWSConnection resource based on connectionid.
	//
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: connectionid
	//   in: query
	//   description: id for AWSConnection resource to be retrieved. expected to be in uuid format i.e. XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
	//   required: true
	//   type: string
	// responses:
	//   '200':
	//     description: AWSConnection resource
	//     schema:
	//         "$ref": "#/definitions/AWSConnection"
	//   '404':
	//     description: Resource not found. Resources are filtered based on connectiontype = AWSConnectionType. If connectionid of Non-AWSConnection is provided ResourceNotFound error is returned.
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"
	//   '500':
	//     description: Internal server error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"
	//   default:
	//     description: unexpected error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	ctx, span, requestID, cl := utilities.SetupTraceAndLogger(r, w, h.l, utilities.GetFunctionName(), h.cfg.Server.PrefixMain)
	defer span.End()

	connectionID := mux.Vars(r)["connectionid"]
	connection, err := h.fetchAWSConnection(connectionID)
	if err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorDatastoreRetrievalFailed, err, requestID, r, &w, span)
		return
	}

	if err := h.vh.GetAWSSecretsEngine(connection, ctx); err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorVaultLoadFailed, err, requestID, r, &w, span)
		return
	}

	var response data.AWSConnectionResponseWrapper
	_ = utilities.CopyMatchingFields(connection, &response)

	utilities.WriteResponse(w, cl, response, span)
}

func (h *AWSConnectionHandler) GenerateCredsAWSConnection(w http.ResponseWriter, r *http.Request) {

	// swagger:operation GET /creds AWSConnection Generate Ephemeral Credentials
	// Generate AWS Creds
	//
	// Endpoint: GET - /v1/connectionmgmt/connection/aws/creds/{connectionid}
	//
	// Description: Generate dynamic credentials using specified AWSConnection.
	//
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: connectionid
	//   in: query
	//   description: id for AWSConnection resource to be used for dynamic credentials generation. expected to be in uuid format i.e. XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
	//   required: true
	//   type: string
	// responses:
	//   '200':
	//     description: Credentials generated successfully.
	//     schema:
	//         "$ref": "#/definitions/CredsAWSConnectionResponse"
	//   '404':
	//     description: Resource not found. Resources are filtered based on connectiontype = AWSConnectionType. If connectionid of Non-AWSConnection is provided ResourceNotFound error is returned.
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"
	//   '500':
	//     description: Internal server error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"
	//   default:
	//     description: unexpected error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	ctx, span, requestID, cl := utilities.SetupTraceAndLogger(r, w, h.l, utilities.GetFunctionName(), h.cfg.Server.PrefixMain)
	defer span.End()

	connectionID := mux.Vars(r)["connectionid"]
	connection, err := h.fetchAWSConnection(connectionID)
	if err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorDatastoreRetrievalFailed, err, requestID, r, &w, span)
		return
	}

	if err := h.vh.GetAWSSecretsEngine(connection, ctx); err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorVaultLoadFailed, err, requestID, r, &w, span)
		return
	}

	var response data.AWSConnectionResponseWrapper
	_ = utilities.CopyMatchingFields(connection, &response)

	utilities.WriteResponse(w, cl, response, span)
}

func (h *AWSConnectionHandler) fetchAWSConnection(connectionID string) (*data.AWSConnection, error) {
	var connection data.AWSConnection
	result := h.pd.RODB().Preload("Connection").First(&connection, "id = ?", connectionID)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, helper.ErrorDictionary[helper.ErrorResourceNotFound].Error()
	}
	return &connection, nil
}

func (h *AWSConnectionHandler) TestAWSConnection(w http.ResponseWriter, r *http.Request) {

	// swagger:operation GET /Test AWSConnection TestAWSConnection
	// Test AWS Connection
	//
	// Endpoint: GET - /v1/connectionmgmt/connection/aws/test/{connectionid}
	//
	// Description: Test connectivity of specified AWSConnection resource.
	//
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: connectionid
	//   in: query
	//   description: id for AWSConnection resource to be retrieved. expected to be in uuid format i.e. XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
	//   required: true
	//   type: string
	// responses:
	//   '200':
	//     description: Connectivity test status
	//     schema:
	//         "$ref": "#/definitions/TestAWSConnectionResponse"
	//   '404':
	//     description: Resource not found. Resources are filtered based on connectiontype = AWSConnectionType. If connectionid of Non-AWSConnection is provided ResourceNotFound error is returned.
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"
	//   '500':
	//     description: Internal server error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"
	//   default:
	//     description: unexpected error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	ctx, span, requestID, cl := utilities.SetupTraceAndLogger(r, w, h.l, utilities.GetFunctionName(), h.cfg.Server.PrefixMain)
	defer span.End()

	connectionID := mux.Vars(r)["connectionid"]
	connection, err := h.fetchAWSConnection(connectionID)
	if err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorDatastoreRetrievalFailed, err, requestID, r, &w, span)
		return
	}

	var response data.TestAWSConnectionResponse
	if err := h.vh.TestAWSSecretsEngine(connection.VaultPath, ctx); err != nil {
		helper.LogDebug(cl, helper.DebugAWSConnectionTestFailed, err, span)
		connection.Connection.SetTestFailed(err.Error())
	} else {
		connection.Connection.SetTestPassed()
	}

	if err := utilities.UpdateObject(h.pd.RWDB(), &connection.Connection, ctx, h.cfg.Server.PrefixMain); err != nil {
		helper.LogError(cl, helper.ErrorDatastoreSaveFailed, err, span)
	}

	response.ID = connection.ID.String()
	response.TestStatus = connection.Connection.TestError
	response.TestStatusCode = connection.Connection.TestSuccessful

	utilities.WriteResponse(w, cl, response, span)
}

func (h *AWSConnectionHandler) UpdateAWSConnection(w http.ResponseWriter, r *http.Request) {

	// swagger:operation PATCH /aws AWSConnection UpdateAWSConnection
	// Update AWS Connection
	//
	// Endpoint: PATCH - /v1/connectionmgmt/connection/aws/{connectionid}
	//
	// Description: Update attributes of AWSConnection resource. Update operation resets Tested status of AWSConnection.
	//
	// ---
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: connectionid
	//   in: query
	//   description: id for AWSConnection resource to be retrieved. expected to be in uuid format i.e. XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
	//   required: true
	//   type: string
	// - in: body
	//   name: Body
	//   description: JSON string defining AWSConnection resource. Change of connectiontype and ID attributes is not allowed.
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/AWSConnectionPatchWrapper"
	// responses:
	//   '200':
	//     description: AWSConnection resource after updates.
	//     schema:
	//         "$ref": "#/definitions/AWSConnection"
	//   '400':
	//     description: Bad request or parameters
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"
	//   default:
	//     description: unexpected error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	ctx, span, requestid, cl := utilities.SetupTraceAndLogger(r, w, h.l, utilities.GetFunctionName(), h.cfg.Server.PrefixMain)
	defer span.End()

	connectionID := mux.Vars(r)["connectionid"]
	p := r.Context().Value(KeyAWSConnectionPatchParamsRecord{}).(data.AWSConnectionPatchWrapper)

	connection, err := h.getAWSConnection(connectionID, cl, requestid, r, &w, span)
	if err != nil {
		return
	}

	if err := h.validateAWSConnectionUpdate(&connection, &p, cl, requestid, r, &w, span); err != nil {
		return
	}

	if err := h.vh.GetAWSSecretsEngine(&connection, ctx); err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorVaultLoadFailed, err, requestid, r, &w, span)
		return
	}

	if p.Connection != nil {
		if err := utilities.CopyMatchingFields(p.Connection, &connection.Connection); err != nil {
			helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorJSONDecodingFailed, err, requestid, r, &w, span)
			return
		}
	}

	if err := utilities.CopyMatchingFields(p, &connection); err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorJSONDecodingFailed, err, requestid, r, &w, span)
		return
	}

	connection.Connection.ResetTestStatus()

	if err := h.updateAWSConnection(&connection, ctx); err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorDatastoreSaveFailed, err, requestid, r, &w, span)
		return
	}

	response, err := h.prepareAWSConnectionResponse(connection, cl, requestid, r, &w, span)
	if err != nil {
		return
	}

	utilities.WriteResponse(w, cl, response, span)
}

func (h *AWSConnectionHandler) getAWSConnection(connectionID string, cl *slog.Logger, requestID string, r *http.Request, w *http.ResponseWriter, span trace.Span) (data.AWSConnection, error) {
	var connection data.AWSConnection
	result := h.pd.RODB().Preload("Connection").First(&connection, "id = ?", connectionID)
	if result.Error != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorDatastoreRetrievalFailed, result.Error, requestID, r, w, span)
		return data.AWSConnection{}, result.Error
	}
	if result.RowsAffected == 0 {
		helper.ReturnError(cl, http.StatusNotFound, helper.ErrorResourceNotFound, helper.ErrorDictionary[helper.ErrorResourceNotFound].Error(), requestID, r, w, span)
		return data.AWSConnection{}, fmt.Errorf("resource not found")
	}
	return connection, nil
}

func (h *AWSConnectionHandler) validateAWSConnection(c *data.AWSConnection, cl *slog.Logger, requestid string, r *http.Request, w http.ResponseWriter, span trace.Span) error {
	switch strings.ToLower(c.CredentialType) {
	case "iam_user":
		if len(c.PolicyARNs) == 0 {
			helper.ReturnError(cl, http.StatusBadRequest, helper.ErrorInvalidPolicyARNs, helper.ErrorDictionary[helper.ErrorInvalidPolicyARNs].Error(), requestid, r, &w, span)
			return fmt.Errorf("invalid policy ARNs")
		}

		if c.DefaultLeaseTTL != "" {
			helper.ReturnError(cl, http.StatusBadRequest, helper.ErrorAWSConnectionInvalidValueForDefaultLeaseTTL, helper.ErrorDictionary[helper.ErrorInvalidPolicyARNs].Error(), requestid, r, &w, span)
			return fmt.Errorf("invalid default lease ttl")
		}

		if c.MaxLeaseTTL != "" {
			helper.ReturnError(cl, http.StatusBadRequest, helper.ErrorAWSConnectionInvalidValueForMaxLeaseTTL, helper.ErrorDictionary[helper.ErrorInvalidPolicyARNs].Error(), requestid, r, &w, span)
			return fmt.Errorf("invalid max lease ttl")
		}
	}
	return nil
}

func (h *AWSConnectionHandler) validateAWSConnectionUpdate(connection *data.AWSConnection, p *data.AWSConnectionPatchWrapper, cl *slog.Logger, requestID string, r *http.Request, w *http.ResponseWriter, span trace.Span) error {
	credentialType := strings.ToLower(connection.CredentialType)
	switch credentialType {
	case "iam_user":
		if len(p.PolicyARNs) == 0 {
			helper.ReturnError(cl, http.StatusBadRequest, helper.ErrorInvalidPolicyARNs, helper.ErrorDictionary[helper.ErrorInvalidPolicyARNs].Error(), requestID, r, w, span)
			return fmt.Errorf("invalid policy ARNs")
		}
	case "session_token":
		if connection.DefaultLeaseTTL != "" {
			helper.ReturnError(cl, http.StatusBadRequest, helper.ErrorAWSConnectionInvalidValueForDefaultLeaseTTL, helper.ErrorDictionary[helper.ErrorAWSConnectionInvalidValueForDefaultLeaseTTL].Error(), requestID, r, w, span)
			return fmt.Errorf("invalid default lease TTL")
		}
		if connection.MaxLeaseTTL != "" {
			helper.ReturnError(cl, http.StatusBadRequest, helper.ErrorAWSConnectionInvalidValueForMaxLeaseTTL, helper.ErrorDictionary[helper.ErrorAWSConnectionInvalidValueForMaxLeaseTTL].Error(), requestID, r, w, span)
			return fmt.Errorf("invalid max lease TTL")
		}
	}
	return nil
}

func (h *AWSConnectionHandler) prepareAWSConnectionResponse(connection data.AWSConnection, cl *slog.Logger, requestID string, r *http.Request, w *http.ResponseWriter, span trace.Span) (data.AWSConnectionResponseWrapper, error) {
	var response data.AWSConnectionResponseWrapper
	if err := utilities.CopyMatchingFields(connection.Connection, &response.Connection); err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorJSONDecodingFailed, err, requestID, r, w, span)
		return response, err
	}
	if err := utilities.CopyMatchingFields(connection, &response); err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorJSONDecodingFailed, err, requestID, r, w, span)
		return response, err
	}
	return response, nil
}

// DeleteAWSConnection deletes a AWSConnection from datastore
func (h *AWSConnectionHandler) DeleteAWSConnection(w http.ResponseWriter, r *http.Request) {

	// swagger:operation DELETE /aws AWSConnection DeleteAWSConnection
	// Delete AWS Connection
	//
	// Endpoint: DELETE - /v1/connectionmgmt/connection/aws/{connectionid}
	//
	// Description: Returns AWSConnection resource based on connectionid.
	//
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: connectionid
	//   in: query
	//   description: id for AWSConnection resource to be retrieved. expected to be in uuid format i.e. XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
	//   required: true
	//   type: string
	// responses:
	//   '200':
	//     description: Resource successfully deleted.
	//     schema:
	//         "$ref": "#/definitions/DeleteAWSConnectionResponse"
	//   '404':
	//     description: Resource not found.
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"
	//   '500':
	//     description: Internal server error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"
	//   default:
	//     description: unexpected error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	ctx, span, requestid, cl := utilities.SetupTraceAndLogger(r, w, h.l, utilities.GetFunctionName(), h.cfg.Server.PrefixMain)
	defer span.End()

	vars := mux.Vars(r)
	connectionid := vars["connectionid"]

	if _, err := uuid.Parse(connectionid); err != nil {
		helper.ReturnError(cl, http.StatusBadRequest, helper.ErrorConnectionIDInvalid, err, requestid, r, &w, span)
		return
	}

	connection, err := h.getAWSConnection(connectionid, cl, requestid, r, &w, span)
	if err != nil {
		return
	}

	if err = h.deleteAWSConnection(&connection, ctx); err != nil {
		helper.ReturnError(cl, http.StatusBadRequest, helper.ErrorDatastoreDeleteFailed, err, requestid, r, &w, span)
		return
	}

	var response data.DeleteAWSConnectionResponse
	response.StatusCode = http.StatusNoContent
	response.Status = http.StatusText(response.StatusCode)

	utilities.WriteResponse(w, cl, response, span)
}

func (h *AWSConnectionHandler) deleteAWSConnection(c *data.AWSConnection, ctx context.Context) error {

	tr := otel.Tracer(h.cfg.Server.PrefixMain)
	ctx, span := tr.Start(ctx, utilities.GetFunctionName())
	defer span.End()

	// Begin a transaction
	tx := h.pd.RWDB().Begin()

	// Check if the transaction started successfully
	if tx.Error != nil {
		return tx.Error
	}

	err := h.vh.RemoveAWSSecretsEngine(c, ctx)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Delete from aws_connections
	if err = tx.Exec("DELETE FROM aws_connections WHERE id = ?", c.ID).Error; err != nil || tx.RowsAffected != 1 {
		tx.Rollback()

		if err != nil {
			return fmt.Errorf("failed to delete aws_connection: %w", err)
		} else {
			return fmt.Errorf("unexpected affected row count. %d", tx.RowsAffected)
		}
	}

	// Delete from connections
	if err := tx.Exec("DELETE FROM connections WHERE id = ?", c.ConnectionID).Error; err != nil || tx.RowsAffected != 1 {
		tx.Rollback()

		if err != nil {
			return fmt.Errorf("failed to delete aws_connection: %w", err)
		} else {
			return fmt.Errorf("unexpected affected row count. %d", tx.RowsAffected)
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (h *AWSConnectionHandler) updateAWSConnection(c *data.AWSConnection, ctx context.Context) error {

	tr := otel.Tracer(h.cfg.Server.PrefixMain)
	ctx, span := tr.Start(ctx, utilities.GetFunctionName())
	defer span.End()

	// Begin a transaction
	tx := h.pd.RWDB().Begin()

	// Check if the transaction started successfully
	if tx.Error != nil {
		return tx.Error
	}

	err := h.vh.UpdateAWSSecretsEngine(c, ctx)
	if err != nil {
		tx.Rollback()
		return err
	}

	result := tx.Save(&c.Connection)

	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}

	result = tx.Save(c)

	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (h *AWSConnectionHandler) AddAWSConnection(w http.ResponseWriter, r *http.Request) {

	// swagger:operation POST /aws AWSConnection AddAWSConnection
	// New AWS Connection
	//
	// Endpoint: POST - /v1/connectionmgmt/connection/aws
	//
	// Description: Create new AWSConnection resource.
	//
	// ---
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - in: body
	//   name: Body
	//   description: JSON string defining AWSConnection resource
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/AWSConnectionPostWrapper"
	// responses:
	//   '200':
	//     description: AWSConnection resource just created.
	//     schema:
	//         "$ref": "#/definitions/AWSConnection"
	//   '500':
	//     description: Internal server error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"
	//   default:
	//     description: unexpected error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	ctx, span, requestid, cl := utilities.SetupTraceAndLogger(r, w, h.l, utilities.GetFunctionName(), h.cfg.Server.PrefixMain)
	defer span.End()

	c := r.Context().Value(KeyAWSConnectionRecord{}).(*data.AWSConnection)
	c.Connection.ConnectionType = data.AWSConnectionType

	if err := h.validateAWSConnection(c, cl, requestid, r, w, span); err != nil {
		return
	}

	// Begin a transaction
	tx := h.pd.RWDB().Begin()

	// Check if the transaction started successfully
	if tx.Error != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorDatastoreSaveFailed, tx.Error, requestid, r, &w, span)
		return
	}

	if err := utilities.CreateObject(tx, &c.Connection, ctx, h.cfg.Server.PrefixMain); err != nil {
		tx.Rollback()
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorDatastoreSaveFailed, err, requestid, r, &w, span)
		return
	}

	if err := utilities.CreateObject(tx, &c, ctx, h.cfg.Server.PrefixMain); err != nil {
		tx.Rollback()
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorDatastoreSaveFailed, err, requestid, r, &w, span)
		return
	}

	if err := h.vh.AddAWSSecretsEngine(c, ctx); err != nil {
		tx.Rollback()
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorVaultAWSEngineFailed, err, requestid, r, &w, span)
		return
	}

	if err := tx.Commit().Error; err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorDatastoreSaveFailed, err, requestid, r, &w, span)
		return
	}

	var c_wrapper data.AWSConnectionResponseWrapper

	if err := utilities.CopyMatchingFields(c, &c_wrapper); err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorJSONDecodingFailed, err, requestid, r, &w, span)
		return
	}

	utilities.WriteResponse(w, cl, c_wrapper, span)
}

func (h AWSConnectionHandler) MiddlewareValidateAWSConnection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		_, span, _, cl := utilities.SetupTraceAndLogger(r, rw, h.l, utilities.GetFunctionName(), h.cfg.Server.PrefixMain)
		defer span.End()

		if _, found := utilities.ValidateQueryStringParam("connectionid", r, cl, rw, span); !found {
			return
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(rw, r)
	})
}

func (h AWSConnectionHandler) MiddlewareValidateAWSConnectionPost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		ctx, span, _, cl := utilities.SetupTraceAndLogger(r, rw, h.l, utilities.GetFunctionName(), h.cfg.Server.PrefixMain)
		defer span.End()

		payload, valid := utilities.DecodeAndValidate[data.ConnectionPostWrapper](r, cl, rw, span)
		if !valid {
			return
		}

		// Add application to context
		ctx = context.WithValue(ctx, KeyAWSConnectionRecord{}, payload)
		next.ServeHTTP(rw, r.WithContext(ctx))
	})
}

func (h AWSConnectionHandler) MiddlewareValidateAWSConnectionUpdate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		ctx, span, requestid, cl := utilities.SetupTraceAndLogger(r, rw, h.l, utilities.GetFunctionName(), h.cfg.Server.PrefixMain)
		defer span.End()

		if _, found := utilities.ValidateQueryStringParam("connectionid", r, cl, rw, span); !found {
			return
		}

		// Decode JSON into a map
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			helper.ReturnError(cl, http.StatusBadRequest, helper.ErrorInvalidJSONSchemaForParameter, err, requestid, r, &rw, span)
			return
		}

		var p data.AWSConnectionPatchWrapper

		// Validate and wrap the payload
		err = utilities.ValidateAndWrapPayload(payload, &p)
		if err != nil {
			helper.ReturnError(cl, http.StatusBadRequest, helper.ErrorInvalidJSONSchemaForParameter, err, requestid, r, &rw, span)
			return
		}

		// add the connection to the context
		//ctx := context.WithValue(r.Context(), KeyAWSConnectionPatchParamsRecord{}, p)
		ctx = context.WithValue(ctx, KeyAWSConnectionPatchParamsRecord{}, p)
		r = r.WithContext(ctx)

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(rw, r)
	})
}
