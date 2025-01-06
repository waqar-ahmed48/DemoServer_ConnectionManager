package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"DemoServer_ConnectionManager/configuration"
	"DemoServer_ConnectionManager/data"
	"DemoServer_ConnectionManager/datalayer"
	"DemoServer_ConnectionManager/helper"

	"github.com/gorilla/mux"
)

type KeyConnectionRecord struct{}

type ConnectionHandler struct {
	l                      *slog.Logger
	cfg                    *configuration.Config
	pd                     *datalayer.PostgresDataSource
	connections_list_limit int
}

func NewConnectionsHandler(cfg *configuration.Config, l *slog.Logger, pd *datalayer.PostgresDataSource) (*ConnectionHandler, error) {
	var c ConnectionHandler

	c.cfg = cfg
	c.l = l
	c.pd = pd
	c.connections_list_limit = cfg.Server.ListLimit

	return &c, nil
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

	_, span, requestid, cl := h.setupTraceAndLogger(r, w)
	defer span.End()

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

	var response data.ConnectionsResponse

	result := h.pd.RODB().Limit(limit).Offset(skip).Find(&response.Connections)

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

	response.Total = len(response.Connections)
	response.Skip = skip
	response.Limit = limit

	err := json.NewEncoder(w).Encode(response)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err, span)
	}
}

func (h ConnectionHandler) MiddlewareValidateConnectionsGet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		_, span, requestid, cl := h.setupTraceAndLogger(r, w)
		defer span.End()

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
					&rw, span)
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

		r = r.WithContext(ctx)

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(rw, r)
	})
}

func (h ConnectionHandler) MiddlewareValidateConnectionLink(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		_, span, requestid, cl := h.setupTraceAndLogger(r, w)
		defer span.End()

		vars := mux.Vars(r)
		connectionid := vars["connectionid"]
		applicationid := vars["applicationid"]

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

		if len(applicationid) == 0 {
			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorApplicationIDInvalid,
				helper.ErrorDictionary[helper.ErrorApplicationIDInvalid].Error(),
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

func (h ConnectionHandler) MiddlewareValidateConnectionUnlink(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		_, span, requestid, cl := h.setupTraceAndLogger(r, w)
		defer span.End()

		vars := mux.Vars(r)
		connectionid := vars["connectionid"]
		applicationid := vars["applicationid"]

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

		if len(applicationid) == 0 {
			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorApplicationIDInvalid,
				helper.ErrorDictionary[helper.ErrorApplicationIDInvalid].Error(),
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

	_, span, requestid, cl := h.setupTraceAndLogger(r, w)
	defer span.End()

	var connection data.Connection

	vars := mux.Vars(r)
	connectionid := vars["connectionid"]
	applicationid := vars["applicationid"]

	result := h.pd.RODB().First(&connection, "id = ?", connectionid)

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
		helper.LogDebug(cl, helper.ErrorResourceNotFound, helper.ErrNone, span)

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

	result = h.pd.RWDB().Save(&connection)

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

	_, span, requestid, cl := h.setupTraceAndLogger(r, w)
	defer span.End()

	var connection data.Connection

	vars := mux.Vars(r)
	connectionid := vars["connectionid"]
	applicationid := vars["applicationid"]

	result := h.pd.RODB().First(&connection, "id = ?", connectionid)

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

	result = h.pd.RWDB().Save(&connection)

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
}
