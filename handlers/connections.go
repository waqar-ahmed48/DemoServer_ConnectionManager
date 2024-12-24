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
	// GET - Connections
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

	requestid, cl := helper.PrepareContext(r, &w, h.l)

	helper.LogInfo(cl, helper.InfoHandlingRequest, helper.ErrNone)

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
		helper.LogError(cl, helper.ErrorDatastoreRetrievalFailed, result.Error)

		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreRetrievalFailed,
			requestid,
			r,
			&w)
		return
	}

	response.Total = len(response.Connections)
	response.Skip = skip
	response.Limit = limit

	err := json.NewEncoder(w).Encode(response)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err)
	}
}

func (h ConnectionHandler) MiddlewareValidateConnectionsGet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		requestid, cl := helper.PrepareContext(r, &rw, h.l)

		vars := r.URL.Query()

		limit_str := vars.Get("limit")
		if limit_str != "" {
			limit, err := strconv.Atoi(limit_str)
			if err != nil {
				helper.LogDebug(cl, helper.ErrorInvalidValueForLimit, err)

				helper.ReturnError(
					cl,
					http.StatusBadRequest,
					helper.ErrorInvalidValueForLimit,
					requestid,
					r,
					&rw)
				return
			}

			if limit <= 0 {
				helper.LogDebug(cl, helper.ErrorLimitMustBeGtZero, helper.ErrNone)

				helper.ReturnError(
					cl,
					http.StatusBadRequest,
					helper.ErrorLimitMustBeGtZero,
					requestid,
					r,
					&rw)
				return
			}
		}

		skip_str := vars.Get("skip")
		if skip_str != "" {
			skip, err := strconv.Atoi(skip_str)
			if err != nil {
				helper.LogDebug(cl, helper.ErrorInvalidValueForSkip, err)

				helper.ReturnError(
					cl,
					http.StatusBadRequest,
					helper.ErrorInvalidValueForSkip,
					requestid,
					r,
					&rw)
				return
			}

			if skip < 0 {
				helper.LogDebug(cl, helper.ErrorSkipMustBeGtZero, helper.ErrNone)

				helper.ReturnError(
					cl,
					http.StatusBadRequest,
					helper.ErrorSkipMustBeGtZero,
					requestid,
					r,
					&rw)
				return
			}
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(rw, r)
	})
}
