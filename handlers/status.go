package handlers

import (
	"DemoServer_ConnectionManager/datalayer"
	"DemoServer_ConnectionManager/helper"
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
	l  *slog.Logger
	pd *datalayer.PostgresDataSource
}

func NewStatusHandler(l *slog.Logger, pd *datalayer.PostgresDataSource) *StatusHandler {
	return &StatusHandler{l, pd}
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

	_, cl := helper.PrepareContext(r, &w, eh.l)

	helper.LogInfo(cl, helper.InfoHandlingRequest, helper.ErrNone)

	var response StatusResponse
	response.Status = "DOWN"
	response.Timestamp = time.Now().String()

	err := eh.pd.Ping()

	if err != nil {
		response.Status = helper.ErrorDictionary[helper.InfoDemoServerConnectionManagerStatusDOWN].Description
		response.StatusCode = helper.ErrorDictionary[helper.InfoDemoServerConnectionManagerStatusDOWN].Code

		helper.LogError(cl, helper.ErrorDatastoreNotAvailable, err)
	} else {
		helper.LogDebug(cl, helper.DebugDatastoreConnectionUP, helper.ErrNone)
		response.Status = helper.ErrorDictionary[helper.InfoDemoServerConnectionManagerStatusUP].Description
		response.StatusCode = helper.ErrorDictionary[helper.InfoDemoServerConnectionManagerStatusUP].Code
	}

	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err)
	}
}
