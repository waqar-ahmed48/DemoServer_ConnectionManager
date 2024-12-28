package handlers

import (
	"DemoServer_ConnectionManager/configuration"
	"DemoServer_ConnectionManager/datalayer"
	"DemoServer_ConnectionManager/helper"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
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
	tr := otel.Tracer(eh.cfg.Server.PrefixMain)
	ctx, span := tr.Start(context.Background(), "GetStatus")
	defer span.End()

	// Add trace context to the logger
	traceLogger := eh.l.With(
		slog.String("trace_id", span.SpanContext().TraceID().String()),
		slog.String("span_id", span.SpanContext().SpanID().String()),
	)

	_, cl := helper.PrepareContext(r, &w, traceLogger)

	helper.LogInfo(cl, helper.InfoHandlingRequest, helper.ErrNone)

	var response StatusResponse
	response.Status = "DOWN"
	response.Timestamp = time.Now().String()

	err := eh.pd.Ping(ctx)

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
