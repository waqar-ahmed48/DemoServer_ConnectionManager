package handlers

import (
	"fmt"
	"log/slog"
	"math"
	"net/http"

	"DemoServer_ConnectionManager/configuration"
	"DemoServer_ConnectionManager/data"
	"DemoServer_ConnectionManager/datalayer"
	"DemoServer_ConnectionManager/helper"
	"DemoServer_ConnectionManager/utilities"

	"github.com/gorilla/mux"
)

type KeyConnectionRecord struct{}

type ConnectionHandler struct {
	l          *slog.Logger
	cfg        *configuration.Config
	pd         *datalayer.PostgresDataSource
	list_limit int
}

func NewConnectionsHandler(cfg *configuration.Config, l *slog.Logger, pd *datalayer.PostgresDataSource) (*ConnectionHandler, error) {
	var c ConnectionHandler

	c.cfg = cfg
	c.l = l
	c.pd = pd
	c.list_limit = cfg.Server.ListLimit

	return &c, nil
}

func (h *ConnectionHandler) fetchConnections(limit, skip int) ([]data.Connection, error) {
	var connections []data.Connection

	result := h.pd.RODB().
		Limit(limit).
		Offset(skip).
		Order("name").
		Find(&connections)

	if result.Error != nil {
		return nil, result.Error
	}
	return connections, nil
}

func (h *ConnectionHandler) getConnection(connectionid string) (*data.Connection, int, helper.ErrorTypeEnum, error) {
	var connection data.Connection

	result := h.pd.RODB().First(&connection, "id = ?", connectionid)

	if result.Error != nil {
		return nil, http.StatusInternalServerError, helper.ErrorDatastoreRetrievalFailed, result.Error
	}

	if result.RowsAffected == 0 {
		return nil, http.StatusNotFound, helper.ErrorResourceNotFound, fmt.Errorf("%s", helper.ErrorDictionary[helper.ErrorResourceNotFound].Error())
	}

	return &connection, http.StatusOK, helper.ErrorNone, nil
}

func (h *ConnectionHandler) GetConnections(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /connections Connection GetConnections
	// List Connections
	//
	// Endpoint: GET - /v1/connectionmgmt/connections
	//
	// Description: Returns list of generic connections resources. It is useful to list all connections
	// currently in ConnectionManager. Generic Connection resource does not have specific details
	// and attributes of specialized connection types. It only tracks general information about
	// connection including its type.
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

	_, span, requestid, cl := utilities.SetupTraceAndLogger(r, w, h.l, utilities.GetFunctionName(), h.cfg.Server.PrefixMain)
	defer span.End()

	vars := r.URL.Query()
	limit := utilities.ParseQueryParam(vars, "limit", h.list_limit, h.cfg.DataLayer.MaxResults)
	skip := utilities.ParseQueryParam(vars, "skip", 0, math.MaxInt32)

	connections, err := h.fetchConnections(limit, skip)
	if err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorDatastoreRetrievalFailed, err, requestid, r, &w, span)
		return
	}

	response, err := h.buildConnectionsResponse(connections, limit, skip)
	if err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorVaultLoadFailed, err, requestid, r, &w, span)
		return
	}

	utilities.WriteResponse(w, cl, response, span)
}

func (h *ConnectionHandler) buildConnectionsResponse(connections []data.Connection, limit, skip int) (data.ConnectionsResponse, error) {
	response := data.ConnectionsResponse{
		Total: len(connections),
		Skip:  skip,
		Limit: limit,
	}

	if len(connections) == 0 {
		response.Connections = []data.Connection{}
		return response, nil
	}

	for _, conn := range connections {
		var wrappedConn data.Connection
		if err := utilities.CopyMatchingFields(conn, &wrappedConn); err != nil {
			return response, err
		}
		response.Connections = append(response.Connections, wrappedConn)
	}

	return response, nil
}

func (h ConnectionHandler) MiddlewareValidateConnectionsGet(next http.Handler) http.Handler {
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

func (h ConnectionHandler) MiddlewareValidateConnectionLink(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		_, span, _, cl := utilities.SetupTraceAndLogger(r, rw, h.l, utilities.GetFunctionName(), h.cfg.Server.PrefixMain)
		defer span.End()

		if _, found := utilities.ValidateQueryStringParam("connectionid", r, cl, rw, span); !found {
			return
		}

		if _, found := utilities.ValidateQueryStringParam("applicationid", r, cl, rw, span); !found {
			return
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(rw, r)
	})
}

func (h ConnectionHandler) MiddlewareValidateConnectionUnlink(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		_, span, _, cl := utilities.SetupTraceAndLogger(r, rw, h.l, utilities.GetFunctionName(), h.cfg.Server.PrefixMain)
		defer span.End()

		if _, found := utilities.ValidateQueryStringParam("connectionid", r, cl, rw, span); !found {
			return
		}

		if _, found := utilities.ValidateQueryStringParam("applicationid", r, cl, rw, span); !found {
			return
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(rw, r)
	})
}

func (h *ConnectionHandler) LinkConnection(w http.ResponseWriter, r *http.Request) {

	// swagger:operation POST /connection/link LinkConnection
	// Link application to connection
	//
	// Endpoint: POST - /v1/connectionmgmt/connection/link
	//
	// Description: Link application to connection.
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
	//     description: Connection linked successfully.
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
	applicationid := vars["applicationid"]

	connection, httpStatusCode, helpError, err := h.getConnection(connectionid)
	if err != nil {
		helper.ReturnError(cl, httpStatusCode, helpError, err, requestid, r, &w, span)
		return
	}

	for _, item := range connection.Applications {
		if item == applicationid {
			// The applicationid already exists.
			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorApplicationAlreadyLinked,
				helper.ErrorDictionary[helper.ErrorApplicationAlreadyLinked].Error(),
				requestid,
				r,
				&w,
				span)
			return
		}
	}
	// The string does not exist, append it.
	connection.Applications = append(connection.Applications, applicationid)

	if err := utilities.UpdateObject(h.pd.RWDB(), connection, ctx, h.cfg.Server.PrefixMain); err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorDatastoreRetrievalFailed, err, requestid, r, &w, span)
	}
}

func (h *ConnectionHandler) UnlinkConnection(w http.ResponseWriter, r *http.Request) {

	// swagger:operation POST /connection/link LinkConnection
	// Link application to connection
	//
	// Endpoint: POST - /v1/connectionmgmt/connection/link
	//
	// Description: Link application to connection.
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
	//     description: Connection linked successfully.
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
	applicationid := vars["applicationid"]

	connection, httpStatusCode, helpError, err := h.getConnection(connectionid)
	if err != nil {
		helper.ReturnError(cl, httpStatusCode, helpError, err, requestid, r, &w, span)
		return
	}

	found := false
	apps := []string{}

	for _, item := range connection.Applications {
		if item == applicationid {
			found = true
			continue // Skip the string to be removed
		}
		apps = append(apps, item)
	}

	if !found {
		helper.ReturnError(
			cl,
			http.StatusNotFound,
			helper.ErrorLinkNotFound,
			helper.ErrorDictionary[helper.ErrorLinkNotFound].Error(),
			requestid,
			r,
			&w,
			span)
		return
	}

	connection.Applications = apps

	if err := utilities.UpdateObject(h.pd.RWDB(), &connection, ctx, h.cfg.Server.PrefixMain); err != nil {
		helper.ReturnError(cl, http.StatusInternalServerError, helper.ErrorDatastoreRetrievalFailed, err, requestid, r, &w, span)
	}
}
