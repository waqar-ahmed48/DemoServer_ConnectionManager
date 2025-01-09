package handlers

import (
	"DemoServer_ConnectionManager/configuration"
	"DemoServer_ConnectionManager/datalayer"
	"DemoServer_ConnectionManager/helper"
	"DemoServer_ConnectionManager/utilities"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Response schema for ConnectionManager Status GET
// swagger:model
type StatusResponse struct {
	// UP = UP, DOWN = DOWN
	// in: status
	Status string `json:"status"`
	// Down = ConnectionManager_Info_000003. UP = ConnectionManager_Info_000002
	// in: statusCode
	StatusCode string `json:"statusCode"`
	// date & time stamp for status check
	// in: timestamp
	Timestamp string `json:"timestamp"`
}

type StatusHandler struct {
	l   *slog.Logger
	pd  *datalayer.PostgresDataSource
	cfg *configuration.Config
}

func NewStatusHandler(l *slog.Logger, pd *datalayer.PostgresDataSource, cfg *configuration.Config) *StatusHandler {
	return &StatusHandler{l, pd, cfg}
}

func (eh *StatusHandler) GetStatus(w http.ResponseWriter, r *http.Request) {

	// swagger:operation GET /status Status GetStatus
	// GET - Status
	//
	// Endpoint: GET - /v1/connectionmgmt/status
	//
	//
	// Description: Returns status of ConnectionManager Instance
	//
	// ---
	// produces:
	// - application/json
	// responses:
	//   '200':
	//     description: StatusReponse
	//     schema:
	//         "$ref": "#/definitions/StatusResponse"
	//   default:
	//     description: unexpected error
	//     schema:
	//       "$ref": "#/definitions/ErrorResponse"

	// Start a trace
	ctx, span, _, cl := utilities.SetupTraceAndLogger(r, w, eh.l, utilities.GetFunctionName(), eh.cfg.Server.PrefixMain)
	defer span.End()

	var response StatusResponse
	response.Status = "DOWN"
	response.Timestamp = time.Now().String()

	err := eh.pd.Ping(ctx)

	if err != nil {
		response.Status = helper.ErrorDictionary[helper.InfoDemoServerConnectionManagerStatusDOWN].Description
		response.StatusCode = helper.ErrorDictionary[helper.InfoDemoServerConnectionManagerStatusDOWN].Code

		helper.LogError(cl, helper.ErrorDatastoreNotAvailable, err, span)
	} else {
		helper.LogDebug(cl, helper.DebugDatastoreConnectionUP, helper.ErrNone, span)
		response.Status = helper.ErrorDictionary[helper.InfoDemoServerConnectionManagerStatusUP].Description
		response.StatusCode = helper.ErrorDictionary[helper.InfoDemoServerConnectionManagerStatusUP].Code
	}

	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err, span)
	}
}
