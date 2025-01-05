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
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
)

type KeyAWSConnectionRecord struct{}
type KeyAWSConnectionPatchParamsRecord struct{}

type AWSConnectionHandler struct {
	l                      *slog.Logger
	cfg                    *configuration.Config
	pd                     *datalayer.PostgresDataSource
	vh                     *secretsmanager.VaultHandler
	connections_list_limit int
}

func NewAWSConnectionHandler(cfg *configuration.Config, l *slog.Logger, pd *datalayer.PostgresDataSource, vh *secretsmanager.VaultHandler) (*AWSConnectionHandler, error) {
	var c AWSConnectionHandler

	c.cfg = cfg
	c.l = l
	c.pd = pd
	c.connections_list_limit = cfg.Server.ListLimit
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

	tr := otel.Tracer(h.cfg.Server.PrefixMain)
	ctx, span := tr.Start(r.Context(), utilities.GetFunctionName())
	defer span.End()

	// Add trace context to the logger
	traceLogger := h.l.With(
		slog.String("trace_id", span.SpanContext().TraceID().String()),
		slog.String("span_id", span.SpanContext().SpanID().String()),
	)

	requestid, cl := helper.PrepareContext(r, &w, traceLogger)

	helper.LogInfo(cl, helper.InfoHandlingRequest, helper.ErrNone, span)

	vars := r.URL.Query()

	limit, skip := h.connections_list_limit, 0

	limit_str := vars.Get("limit")
	if limit_str != "" {
		limit, _ = strconv.Atoi(limit_str)
	}

	skip_str := vars.Get("skip")
	if skip_str != "" {
		skip, _ = strconv.Atoi(skip_str)
	}

	if limit == -1 || limit > h.cfg.DataLayer.MaxResults {
		limit = h.cfg.DataLayer.MaxResults
	}

	var response data.AWSConnectionsResponse

	var conns []data.AWSConnection

	result := h.pd.RODB().
		Preload("Connection"). // Preloads the Connection struct
		Limit(limit).
		Offset(skip).
		Order("connections.name"). // Orders by the name in the Connection table
		Joins("left join connections on connections.id = aws_connections.connection_id").
		Find(&conns) // Finds all AWSConnection entries

	if result.Error != nil {
		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreRetrievalFailed,
			result.Error,
			requestid,
			r,
			&w,
			span)
		return
	}

	response.Total = len(conns)
	response.Skip = skip
	response.Limit = limit
	if response.Total == 0 {
		response.AWSConnections = ([]data.AWSConnectionResponseWrapper{})
	} else {
		for _, value := range conns {
			err := h.vh.GetAWSSecretsEngine(&value, ctx)

			if err != nil {
				helper.LogError(cl, helper.ErrorVaultLoadFailed, err, span)

				helper.ReturnError(
					cl,
					http.StatusInternalServerError,
					helper.ErrorVaultLoadFailed,
					err,
					requestid,
					r,
					&w,
					span)
				return
			}

			var oRespConn data.AWSConnectionResponseWrapper
			_ = utilities.CopyMatchingFields(value, &oRespConn)
			response.AWSConnections = append(response.AWSConnections, oRespConn)
		}
	}

	err := json.NewEncoder(w).Encode(response)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err, span)
	}
}

func (h AWSConnectionHandler) MiddlewareValidateAWSConnectionsGet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		tr := otel.Tracer(h.cfg.Server.PrefixMain)
		_, span := tr.Start(r.Context(), utilities.GetFunctionName())
		defer span.End()

		// Add trace context to the logger
		traceLogger := h.l.With(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)

		requestid, cl := helper.PrepareContext(r, &rw, traceLogger)

		vars := r.URL.Query()

		limit_str := vars.Get("limit")
		if limit_str != "" {
			limit, err := strconv.Atoi(limit_str)
			if err != nil {
				helper.ReturnError(
					cl,
					http.StatusBadRequest,
					helper.ErrorInvalidValueForLimit,
					err,
					requestid,
					r,
					&rw,
					span)
				return
			}

			if limit <= 0 {
				helper.ReturnError(
					cl,
					http.StatusBadRequest,
					helper.ErrorLimitMustBeGtZero,
					helper.ErrorDictionary[helper.ErrorLimitMustBeGtZero].Error(),
					requestid,
					r,
					&rw,
					span)
				return
			}
		}

		skip_str := vars.Get("skip")
		if skip_str != "" {
			skip, err := strconv.Atoi(skip_str)
			if err != nil {
				helper.ReturnError(
					cl,
					http.StatusBadRequest,
					helper.ErrorInvalidValueForSkip,
					err,
					requestid,
					r,
					&rw,
					span)
				return
			}

			if skip < 0 {
				helper.ReturnError(
					cl,
					http.StatusBadRequest,
					helper.ErrorSkipMustBeGtZero,
					helper.ErrorDictionary[helper.ErrorSkipMustBeGtZero].Error(),
					requestid,
					r,
					&rw,
					span)
				return
			}
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

	tr := otel.Tracer(h.cfg.Server.PrefixMain)
	ctx, span := tr.Start(r.Context(), utilities.GetFunctionName())
	defer span.End()

	// Add trace context to the logger
	traceLogger := h.l.With(
		slog.String("trace_id", span.SpanContext().TraceID().String()),
		slog.String("span_id", span.SpanContext().SpanID().String()),
	)

	requestid, cl := helper.PrepareContext(r, &w, traceLogger)

	helper.LogInfo(cl, helper.InfoHandlingRequest, helper.ErrNone, span)

	vars := mux.Vars(r)
	connectionid := vars["connectionid"]
	var connection data.AWSConnection

	result := h.pd.RODB().Preload("Connection").First(&connection, "id = ?", connectionid)

	if result.Error != nil {
		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreRetrievalFailed,
			result.Error,
			requestid,
			r,
			&w,
			span)
		return
	}

	if result.RowsAffected == 0 {
		helper.ReturnError(
			cl,
			http.StatusNotFound,
			helper.ErrorResourceNotFound,
			helper.ErrorDictionary[helper.ErrorResourceNotFound].Error(),
			requestid,
			r,
			&w,
			span)
		return
	}

	err := h.vh.GetAWSSecretsEngine(&connection, ctx)

	if err != nil {
		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorVaultLoadFailed,
			err,
			requestid,
			r,
			&w,
			span)
		return
	}

	var oRespConn data.AWSConnectionResponseWrapper
	_ = utilities.CopyMatchingFields(connection, &oRespConn)

	err = json.NewEncoder(w).Encode(oRespConn)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err, span)
	}
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

	tr := otel.Tracer(h.cfg.Server.PrefixMain)
	ctx, span := tr.Start(r.Context(), utilities.GetFunctionName())
	defer span.End()

	// Add trace context to the logger
	traceLogger := h.l.With(
		slog.String("trace_id", span.SpanContext().TraceID().String()),
		slog.String("span_id", span.SpanContext().SpanID().String()),
	)

	requestid, cl := helper.PrepareContext(r, &w, traceLogger)

	helper.LogInfo(cl, helper.InfoHandlingRequest, helper.ErrNone, span)

	vars := mux.Vars(r)
	connectionid := vars["connectionid"]
	var connection data.AWSConnection

	result := h.pd.RODB().Preload("Connection").First(&connection, "id = ?", connectionid)

	if result.Error != nil {
		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreRetrievalFailed,
			result.Error,
			requestid,
			r,
			&w,
			span)
		return
	}

	if result.RowsAffected == 0 {

		helper.ReturnError(
			cl,
			http.StatusNotFound,
			helper.ErrorResourceNotFound,
			helper.ErrorDictionary[helper.ErrorResourceNotFound].Error(),
			requestid,
			r,
			&w,
			span)
		return
	}

	if connection.Connection.TestSuccessful != 1 {
		helper.LogDebug(cl, helper.ErrorConnectionNotTestedSuccessfully, helper.ErrNone, span)

		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorConnectionNotTestedSuccessfully,
			helper.ErrorDictionary[helper.ErrorConnectionNotTestedSuccessfully].Error(),
			requestid,
			r,
			&w,
			span)

		return
	}

	response, err := h.vh.GenerateCredsAWSSecretsEngine(connection.VaultPath, ctx)

	if err != nil {
		helper.LogDebug(cl, helper.DebugAWSConnectionTestFailed, err, span)
	} else {
		response.ConnectionID = connectionid
	}

	err = json.NewEncoder(w).Encode(&response)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err, span)
	}
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

	tr := otel.Tracer(h.cfg.Server.PrefixMain)
	ctx, span := tr.Start(r.Context(), utilities.GetFunctionName())
	defer span.End()

	// Add trace context to the logger
	traceLogger := h.l.With(
		slog.String("trace_id", span.SpanContext().TraceID().String()),
		slog.String("span_id", span.SpanContext().SpanID().String()),
	)

	requestid, cl := helper.PrepareContext(r, &w, traceLogger)

	helper.LogInfo(cl, helper.InfoHandlingRequest, helper.ErrNone, span)

	vars := mux.Vars(r)
	connectionid := vars["connectionid"]
	var connection data.AWSConnection

	var response data.TestAWSConnectionResponse

	result := h.pd.RODB().Preload("Connection").First(&connection, "id = ?", connectionid)

	if result.Error != nil {
		helper.LogError(cl, helper.ErrorDatastoreRetrievalFailed, result.Error, span)

		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreRetrievalFailed,
			result.Error,
			requestid,
			r,
			&w,
			span)
		return
	}

	if result.RowsAffected == 0 {
		helper.ReturnError(
			cl,
			http.StatusNotFound,
			helper.ErrorResourceNotFound,
			helper.ErrorDictionary[helper.ErrorResourceNotFound].Error(),
			requestid,
			r,
			&w,
			span)
		return
	}

	err := h.vh.TestAWSSecretsEngine(connection.VaultPath, ctx)

	if err != nil {
		helper.LogDebug(cl, helper.DebugAWSConnectionTestFailed, err, span)
		connection.Connection.SetTestFailed(err.Error())
	} else {
		connection.Connection.SetTestPassed()
	}

	result = h.pd.RWDB().Save(&connection.Connection)

	if result.Error != nil {
		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreSaveFailed,
			result.Error,
			requestid,
			r,
			&w,
			span)
		return
	}

	if result.RowsAffected != 1 {
		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreSaveFailed,
			helper.ErrorDictionary[helper.ErrorDatastoreSaveFailed].Error(),
			requestid,
			r,
			&w,
			span)
		return
	}

	response.ID = connection.ID.String()
	response.TestStatus = connection.Connection.TestError
	response.TestStatusCode = connection.Connection.TestSuccessful

	err = json.NewEncoder(w).Encode(&response)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err, span)
	}
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

	tr := otel.Tracer(h.cfg.Server.PrefixMain)
	ctx, span := tr.Start(r.Context(), utilities.GetFunctionName())
	defer span.End()

	// Add trace context to the logger
	traceLogger := h.l.With(
		slog.String("trace_id", span.SpanContext().TraceID().String()),
		slog.String("span_id", span.SpanContext().SpanID().String()),
	)

	requestid, cl := helper.PrepareContext(r, &w, traceLogger)

	helper.LogInfo(cl, helper.InfoHandlingRequest, helper.ErrNone, span)

	vars := mux.Vars(r)
	connectionid := vars["connectionid"]

	p := r.Context().Value(KeyAWSConnectionPatchParamsRecord{}).(data.AWSConnectionPatchWrapper)

	var connection data.AWSConnection

	result := h.pd.RODB().Preload("Connection").First(&connection, "id = ?", connectionid)

	if result.Error != nil {
		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreRetrievalFailed,
			result.Error,
			requestid,
			r,
			&w,
			span)
		return
	}

	if result.RowsAffected == 0 {
		helper.ReturnError(
			cl,
			http.StatusNotFound,
			helper.ErrorResourceNotFound,
			helper.ErrorDictionary[helper.ErrorResourceNotFound].Error(),
			requestid,
			r,
			&w,
			span)
		return
	}

	if strings.ToLower(connection.CredentialType) == "iam_user" {
		if len(p.PolicyARNs) == 0 {
			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorInvalidPolicyARNs,
				helper.ErrorDictionary[helper.ErrorInvalidPolicyARNs].Error(),
				requestid,
				r,
				&w,
				span)
			return
		}
	} else if strings.ToLower(connection.CredentialType) == "session_token" {
		if connection.DefaultLeaseTTL != "" {
			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorAWSConnectionInvalidValueForDefaultLeaseTTL,
				helper.ErrorDictionary[helper.ErrorAWSConnectionInvalidValueForDefaultLeaseTTL].Error(),
				requestid,
				r,
				&w,
				span)
			return
		}

		if connection.MaxLeaseTTL != "" {
			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorAWSConnectionInvalidValueForMaxLeaseTTL,
				helper.ErrorDictionary[helper.ErrorAWSConnectionInvalidValueForMaxLeaseTTL].Error(),
				requestid,
				r,
				&w,
				span)
			return
		}
	}

	err := h.vh.GetAWSSecretsEngine(&connection, ctx)

	if err != nil {
		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorVaultLoadFailed,
			err,
			requestid,
			r,
			&w,
			span)
		return
	}

	_ = utilities.CopyMatchingFields(p.Connection, &connection.Connection)
	_ = utilities.CopyMatchingFields(p, &connection)

	connection.Connection.ResetTestStatus()

	err = h.updateAWSConnection(&connection, ctx)

	if err != nil {
		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreSaveFailed,
			err,
			requestid,
			r,
			&w,
			span)
		return
	}

	result = h.pd.RODB().Preload("Connection").First(&connection, "id = ?", connectionid)

	if result.Error != nil {
		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreRetrievalFailed,
			result.Error,
			requestid,
			r,
			&w,
			span)
		return
	}

	if result.RowsAffected == 0 {
		helper.ReturnError(
			cl,
			http.StatusNotFound,
			helper.ErrorResourceNotFound,
			helper.ErrorDictionary[helper.ErrorResourceNotFound].Error(),
			requestid,
			r,
			&w,
			span)
		return
	}

	err = h.vh.GetAWSSecretsEngine(&connection, ctx)

	if err != nil {
		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorVaultLoadFailed,
			err,
			requestid,
			r,
			&w,
			span)
		return
	}

	var oRespConn data.AWSConnectionResponseWrapper
	_ = utilities.CopyMatchingFields(connection, &oRespConn)

	err = json.NewEncoder(w).Encode(oRespConn)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err, span)
	}
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

	tr := otel.Tracer(h.cfg.Server.PrefixMain)
	ctx, span := tr.Start(r.Context(), utilities.GetFunctionName())
	defer span.End()

	// Add trace context to the logger
	traceLogger := h.l.With(
		slog.String("trace_id", span.SpanContext().TraceID().String()),
		slog.String("span_id", span.SpanContext().SpanID().String()),
	)

	requestid, cl := helper.PrepareContext(r, &w, traceLogger)

	helper.LogInfo(cl, helper.InfoHandlingRequest, helper.ErrNone, span)

	vars := mux.Vars(r)
	connectionid := vars["connectionid"]

	var connection data.AWSConnection
	var err error

	connection.ID, err = uuid.Parse(connectionid)

	if err != nil {
		helper.ReturnError(
			cl,
			http.StatusBadRequest,
			helper.ErrorConnectionIDInvalid,
			err,
			requestid,
			r,
			&w,
			span)
		return
	}

	result := h.pd.RODB().Preload("Connection").First(&connection, "id = ?", connectionid)

	if result.Error != nil {
		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreRetrievalFailed,
			result.Error,
			requestid,
			r,
			&w,
			span)
		return
	}

	if result.RowsAffected == 0 {
		helper.ReturnError(
			cl,
			http.StatusNotFound,
			helper.ErrorResourceNotFound,
			helper.ErrorDictionary[helper.ErrorResourceNotFound].Error(),
			requestid,
			r,
			&w,
			span)
		return
	}

	err = h.deleteAWSConnection(&connection, ctx)

	if err != nil {
		helper.LogDebug(cl, helper.ErrorDatastoreDeleteFailed, err, span)

		helper.ReturnError(
			cl,
			http.StatusBadRequest,
			helper.ErrorDatastoreDeleteFailed,
			err,
			requestid,
			r,
			&w,
			span)
		return
	}

	var response data.DeleteAWSConnectionResponse
	response.StatusCode = http.StatusNoContent
	response.Status = http.StatusText(response.StatusCode)

	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err, span)
	}
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
	if err = tx.Exec("DELETE FROM aws_connections WHERE id = ?", c.ID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete aws_connection: %w", err)
	}

	// Delete from connections
	if err := tx.Exec("DELETE FROM connections WHERE id = ?", c.ConnectionID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete connection: %w", err)
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

	tr := otel.Tracer(h.cfg.Server.PrefixMain)
	ctx, span := tr.Start(r.Context(), utilities.GetFunctionName())
	defer span.End()

	// Add trace context to the logger
	traceLogger := h.l.With(
		slog.String("trace_id", span.SpanContext().TraceID().String()),
		slog.String("span_id", span.SpanContext().SpanID().String()),
	)

	requestid, cl := helper.PrepareContext(r, &w, traceLogger)

	helper.LogInfo(cl, helper.InfoHandlingRequest, helper.ErrNone, span)

	c := r.Context().Value(KeyAWSConnectionRecord{}).(*data.AWSConnection)

	c.Connection.ConnectionType = data.AWSConnectionType

	if strings.ToLower(c.CredentialType) == "iam_user" {
		if len(c.PolicyARNs) == 0 {
			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorInvalidPolicyARNs,
				helper.ErrorDictionary[helper.ErrorInvalidPolicyARNs].Error(),
				requestid,
				r,
				&w,
				span)
			return
		}
	} else if strings.ToLower(c.CredentialType) == "session_token" {
		if c.DefaultLeaseTTL != "" {
			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorAWSConnectionInvalidValueForDefaultLeaseTTL,
				helper.ErrorDictionary[helper.ErrorAWSConnectionInvalidValueForDefaultLeaseTTL].Error(),
				requestid,
				r,
				&w,
				span)
			return
		}

		if c.MaxLeaseTTL != "" {
			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorAWSConnectionInvalidValueForMaxLeaseTTL,
				helper.ErrorDictionary[helper.ErrorAWSConnectionInvalidValueForMaxLeaseTTL].Error(),
				requestid,
				r,
				&w,
				span)
			return
		}
	}

	// Begin a transaction
	tx := h.pd.RWDB().Begin()

	// Check if the transaction started successfully
	if tx.Error != nil {
		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreSaveFailed,
			tx.Error,
			requestid,
			r,
			&w,
			span)
		return
	}

	result := tx.Create(&c.Connection)

	if result.Error != nil {
		tx.Rollback()

		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreSaveFailed,
			result.Error,
			requestid,
			r,
			&w,
			span)
		return
	}

	result = tx.Create(&c)

	if result.Error != nil {
		tx.Rollback()

		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreSaveFailed,
			result.Error,
			requestid,
			r,
			&w,
			span)
		return
	}

	if result.RowsAffected != 1 {
		tx.Rollback()

		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreSaveFailed,
			helper.ErrorDictionary[helper.ErrorDatastoreSaveFailed].Error(),
			requestid,
			r,
			&w,
			span)
		return
	}

	err := h.vh.AddAWSSecretsEngine(c, ctx)
	if err != nil {
		tx.Rollback()

		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorVaultAWSEngineFailed,
			err,
			requestid,
			r,
			&w,
			span)
		return
	} else {
		err = tx.Commit().Error

		if err != nil {
			helper.ReturnError(
				cl,
				http.StatusInternalServerError,
				helper.ErrorDatastoreSaveFailed,
				err,
				requestid,
				r,
				&w,
				span)
			return
		}
	}

	var c_wrapper data.AWSConnectionResponseWrapper

	_ = utilities.CopyMatchingFields(c, &c_wrapper)

	err = json.NewEncoder(w).Encode(c_wrapper)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err, span)
	}

	c = nil
}

func (h AWSConnectionHandler) MiddlewareValidateAWSConnection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		tr := otel.Tracer(h.cfg.Server.PrefixMain)
		_, span := tr.Start(r.Context(), utilities.GetFunctionName())
		defer span.End()

		// Add trace context to the logger
		traceLogger := h.l.With(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)

		requestid, cl := helper.PrepareContext(r, &rw, traceLogger)

		vars := mux.Vars(r)
		connectionid := vars["connectionid"]

		if len(connectionid) == 0 {
			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorConnectionIDInvalid,
				helper.ErrorDictionary[helper.ErrorConnectionIDInvalid].Error(),
				requestid,
				r,
				&rw,
				span)
			return
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(rw, r)
	})
}

func (h AWSConnectionHandler) MiddlewareValidateAWSConnectionPost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		tr := otel.Tracer(h.cfg.Server.PrefixMain)
		ctx, span := tr.Start(r.Context(), utilities.GetFunctionName())
		defer span.End()

		// Add trace context to the logger
		traceLogger := h.l.With(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)

		requestid, cl := helper.PrepareContext(r, &rw, traceLogger)

		c := data.NewAWSConnection(h.cfg)

		err := c.FromJSON(r.Body)
		if err != nil {
			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorInvalidJSONSchemaForParameter,
				err,
				requestid,
				r,
				&rw,
				span)
			return

		}

		err = c.Validate()
		if err != nil {
			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorInvalidJSONSchemaForParameter,
				err,
				requestid,
				r,
				&rw,
				span)
			return

		}

		if c.Connection.ConnectionType != data.NoConnectionType {
			if c.Connection.ConnectionType != data.AWSConnectionType {
				helper.ReturnError(
					cl,
					http.StatusBadRequest,
					helper.ErrorInvalidConnectionType,
					helper.ErrorDictionary[helper.ErrorInvalidConnectionType].Error(),
					requestid,
					r,
					&rw,
					span)
				return

			}
		}

		// add the connection to the context
		//ctx := context.WithValue(r.Context(), KeyAWSConnectionRecord{}, c)
		ctx = context.WithValue(ctx, KeyAWSConnectionRecord{}, c)
		r = r.WithContext(ctx)

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(rw, r)
	})
}

func (h AWSConnectionHandler) MiddlewareValidateAWSConnectionUpdate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		tr := otel.Tracer(h.cfg.Server.PrefixMain)
		ctx, span := tr.Start(r.Context(), utilities.GetFunctionName())
		defer span.End()

		// Add trace context to the logger
		traceLogger := h.l.With(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)

		requestid, cl := helper.PrepareContext(r, &rw, traceLogger)

		vars := mux.Vars(r)
		connectionid := vars["connectionid"]
		var p data.AWSConnectionPatchWrapper

		if len(connectionid) == 0 {
			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorConnectionIDInvalid,
				helper.ErrorDictionary[helper.ErrorConnectionIDInvalid].Error(),
				requestid,
				r,
				&rw,
				span)
			return
		}

		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorInvalidJSONSchemaForParameter,
				err,
				requestid,
				r,
				&rw,
				span)
			return

		}

		err = validator.New().Struct(p)
		if err != nil {
			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorInvalidJSONSchemaForParameter,
				err,
				requestid,
				r,
				&rw,
				span)
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
