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

	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
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

	response.Total = len(conns)
	response.Skip = skip
	response.Limit = limit
	if response.Total == 0 {
		response.AWSConnections = ([]data.AWSConnectionResponseWrapper{})
	} else {
		for _, value := range conns {
			err := h.vh.GetAWSSecretsEngine(&value)

			if err != nil {
				helper.LogError(cl, helper.ErrorVaultLoadFailed, err)

				helper.ReturnErrorWithAdditionalInfo(
					cl,
					http.StatusInternalServerError,
					helper.ErrorVaultLoadFailed,
					requestid,
					r,
					&w,
					err)
				return
			}

			var oRespConn data.AWSConnectionResponseWrapper
			_ = utilities.CopyMatchingFields(value, &oRespConn)
			response.AWSConnections = append(response.AWSConnections, oRespConn)
		}
	}

	err := json.NewEncoder(w).Encode(response)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err)
	}
}

func (h AWSConnectionHandler) MiddlewareValidateAWSConnectionsGet(next http.Handler) http.Handler {
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

	requestid, cl := helper.PrepareContext(r, &w, h.l)

	helper.LogInfo(cl, helper.InfoHandlingRequest, helper.ErrNone)

	vars := mux.Vars(r)
	connectionid := vars["connectionid"]
	var connection data.AWSConnection

	result := h.pd.RODB().Preload("Connection").First(&connection, "id = ?", connectionid)

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

	if result.RowsAffected == 0 {
		helper.LogDebug(cl, helper.ErrorResourceNotFound, helper.ErrNone)

		helper.ReturnError(
			cl,
			http.StatusNotFound,
			helper.ErrorResourceNotFound,
			requestid,
			r,
			&w)
		return
	}

	err := h.vh.GetAWSSecretsEngine(&connection)

	if err != nil {
		helper.LogError(cl, helper.ErrorVaultLoadFailed, err)

		helper.ReturnErrorWithAdditionalInfo(
			cl,
			http.StatusInternalServerError,
			helper.ErrorVaultLoadFailed,
			requestid,
			r,
			&w,
			err)
		return
	}

	var oRespConn data.AWSConnectionResponseWrapper
	_ = utilities.CopyMatchingFields(connection, &oRespConn)

	err = json.NewEncoder(w).Encode(oRespConn)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err)
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

	requestid, cl := helper.PrepareContext(r, &w, h.l)

	helper.LogInfo(cl, helper.InfoHandlingRequest, helper.ErrNone)

	vars := mux.Vars(r)
	connectionid := vars["connectionid"]
	var connection data.AWSConnection

	var response data.TestAWSConnectionResponse

	result := h.pd.RODB().Preload("Connection").First(&connection, "id = ?", connectionid)

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

	if result.RowsAffected == 0 {
		helper.LogDebug(cl, helper.ErrorResourceNotFound, helper.ErrNone)

		helper.ReturnError(
			cl,
			http.StatusNotFound,
			helper.ErrorResourceNotFound,
			requestid,
			r,
			&w)
		return
	}

	err := h.vh.TestAWSSecretsEngine(connection.VaultPath)

	if err != nil {
		helper.LogDebug(cl, helper.DebugAWSConnectionTestFailed, err)
		connection.Connection.SetTestFailed(err.Error())
	} else {
		connection.Connection.SetTestPassed()
	}

	result = h.pd.RWDB().Save(&connection.Connection)

	if result.Error != nil {
		helper.LogError(cl, helper.ErrorDatastoreSaveFailed, result.Error)

		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreSaveFailed,
			requestid,
			r,
			&w)
		return
	}

	if result.RowsAffected != 1 {
		helper.LogError(cl, helper.ErrorDatastoreSaveFailed, helper.ErrNone)

		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreSaveFailed,
			requestid,
			r,
			&w)
		return
	}

	response.ID = connection.ID.String()
	response.TestStatus = connection.Connection.TestError
	response.TestStatusCode = connection.Connection.TestSuccessful

	err = json.NewEncoder(w).Encode(&response)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err)
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

	requestid, cl := helper.PrepareContext(r, &w, h.l)

	helper.LogInfo(cl, helper.InfoHandlingRequest, helper.ErrNone)

	vars := mux.Vars(r)
	connectionid := vars["connectionid"]

	p := r.Context().Value(KeyAWSConnectionPatchParamsRecord{}).(data.AWSConnectionPatchWrapper)

	var connection data.AWSConnection

	result := h.pd.RODB().Preload("Connection").First(&connection, "id = ?", connectionid)

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

	if result.RowsAffected == 0 {
		helper.LogDebug(cl, helper.ErrorResourceNotFound, helper.ErrNone)

		helper.ReturnError(
			cl,
			http.StatusNotFound,
			helper.ErrorResourceNotFound,
			requestid,
			r,
			&w)
		return
	}

	err := h.vh.GetAWSSecretsEngine(&connection)

	if err != nil {
		helper.LogError(cl, helper.ErrorVaultLoadFailed, err)

		helper.ReturnErrorWithAdditionalInfo(
			cl,
			http.StatusInternalServerError,
			helper.ErrorVaultLoadFailed,
			requestid,
			r,
			&w,
			err)
		return
	}

	_ = utilities.CopyMatchingFields(p.Connection, &connection.Connection)
	_ = utilities.CopyMatchingFields(p, &connection)

	connection.Connection.ResetTestStatus()

	err = h.updateAWSConnection(&connection)

	if err != nil {
		helper.LogError(cl, helper.ErrorDatastoreSaveFailed, err)

		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreSaveFailed,
			requestid,
			r,
			&w)
		return
	}

	result = h.pd.RODB().Preload("Connection").First(&connection, "id = ?", connectionid)

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

	if result.RowsAffected == 0 {
		helper.LogDebug(cl, helper.ErrorResourceNotFound, helper.ErrNone)

		helper.ReturnError(
			cl,
			http.StatusNotFound,
			helper.ErrorResourceNotFound,
			requestid,
			r,
			&w)
		return
	}

	err = h.vh.GetAWSSecretsEngine(&connection)

	if err != nil {
		helper.LogError(cl, helper.ErrorVaultLoadFailed, err)

		helper.ReturnErrorWithAdditionalInfo(
			cl,
			http.StatusInternalServerError,
			helper.ErrorVaultLoadFailed,
			requestid,
			r,
			&w,
			err)
		return
	}

	var oRespConn data.AWSConnectionResponseWrapper
	_ = utilities.CopyMatchingFields(connection, &oRespConn)

	err = json.NewEncoder(w).Encode(oRespConn)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err)
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

	requestid, cl := helper.PrepareContext(r, &w, h.l)

	helper.LogInfo(cl, helper.InfoHandlingRequest, helper.ErrNone)

	vars := mux.Vars(r)
	connectionid := vars["connectionid"]

	var connection data.AWSConnection
	var err error

	connection.ID, err = uuid.Parse(connectionid)

	if err != nil {
		helper.LogDebug(cl, helper.ErrorConnectionIDInvalid, err)

		helper.ReturnError(
			cl,
			http.StatusBadRequest,
			helper.ErrorConnectionIDInvalid,
			requestid,
			r,
			&w)
		return
	}

	result := h.pd.RODB().Preload("Connection").First(&connection, "id = ?", connectionid)

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

	if result.RowsAffected == 0 {
		helper.LogDebug(cl, helper.ErrorResourceNotFound, helper.ErrNone)

		helper.ReturnError(
			cl,
			http.StatusNotFound,
			helper.ErrorResourceNotFound,
			requestid,
			r,
			&w)
		return
	}

	err = h.deleteAWSConnection(&connection)

	if err != nil {
		helper.LogDebug(cl, helper.ErrorDatastoreDeleteFailed, err)

		helper.ReturnError(
			cl,
			http.StatusBadRequest,
			helper.ErrorDatastoreDeleteFailed,
			requestid,
			r,
			&w)
		return
	}

	var response data.DeleteAWSConnectionResponse
	response.StatusCode = http.StatusNoContent
	response.Status = http.StatusText(response.StatusCode)

	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err)
	}
}

func (h *AWSConnectionHandler) deleteAWSConnection(c *data.AWSConnection) error {
	// Begin a transaction
	tx := h.pd.RWDB().Begin()

	// Check if the transaction started successfully
	if tx.Error != nil {
		return tx.Error
	}

	err := h.vh.RemoveAWSSecretsEngine(c)
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

func (h *AWSConnectionHandler) updateAWSConnection(c *data.AWSConnection) error {
	// Begin a transaction
	tx := h.pd.RWDB().Begin()

	// Check if the transaction started successfully
	if tx.Error != nil {
		return tx.Error
	}

	err := h.vh.UpdateAWSSecretsEngine(c)
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

	requestid, cl := helper.PrepareContext(r, &w, h.l)

	helper.LogInfo(cl, helper.InfoHandlingRequest, helper.ErrNone)

	c := r.Context().Value(KeyAWSConnectionRecord{}).(*data.AWSConnection)

	c.Connection.ConnectionType = data.AWSConnectionType

	// Begin a transaction
	tx := h.pd.RWDB().Begin()

	// Check if the transaction started successfully
	if tx.Error != nil {
		helper.LogError(cl, helper.ErrorDatastoreSaveFailed, tx.Error)

		helper.ReturnErrorWithAdditionalInfo(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreSaveFailed,
			requestid,
			r,
			&w,
			tx.Error)
		return
	}

	result := tx.Create(&c.Connection)

	if result.Error != nil {
		tx.Rollback()

		helper.LogError(cl, helper.ErrorDatastoreSaveFailed, result.Error)

		helper.ReturnErrorWithAdditionalInfo(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreSaveFailed,
			requestid,
			r,
			&w,
			result.Error)
		return
	}

	result = tx.Create(&c)

	if result.Error != nil {
		tx.Rollback()

		helper.LogError(cl, helper.ErrorDatastoreSaveFailed, result.Error)

		helper.ReturnErrorWithAdditionalInfo(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreSaveFailed,
			requestid,
			r,
			&w,
			result.Error)
		return
	}

	if result.RowsAffected != 1 {
		tx.Rollback()

		helper.LogError(cl, helper.ErrorDatastoreSaveFailed, helper.ErrNone)

		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorDatastoreSaveFailed,
			requestid,
			r,
			&w)
		return
	}

	err := h.vh.AddAWSSecretsEngine(c)
	if err != nil {
		tx.Rollback()

		helper.LogError(cl, helper.ErrorVaultAWSEngineFailed, err)

		helper.ReturnErrorWithAdditionalInfo(
			cl,
			http.StatusInternalServerError,
			helper.ErrorVaultAWSEngineFailed,
			requestid,
			r,
			&w,
			err)
		return
	} else {
		err = tx.Commit().Error

		if err != nil {
			helper.LogError(cl, helper.ErrorDatastoreSaveFailed, err)

			helper.ReturnErrorWithAdditionalInfo(
				cl,
				http.StatusInternalServerError,
				helper.ErrorDatastoreSaveFailed,
				requestid,
				r,
				&w,
				tx.Error)
			return
		}
	}

	var c_wrapper data.AWSConnectionResponseWrapper

	_ = utilities.CopyMatchingFields(c, &c_wrapper)

	err = json.NewEncoder(w).Encode(c_wrapper)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err)
	}

	c = nil
}

func (h AWSConnectionHandler) MiddlewareValidateAWSConnection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		requestid, cl := helper.PrepareContext(r, &rw, h.l)

		vars := mux.Vars(r)
		connectionid := vars["connectionid"]

		if len(connectionid) == 0 {
			helper.LogDebug(cl, helper.ErrorConnectionIDInvalid, helper.ErrNone)

			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorConnectionIDInvalid,
				requestid,
				r,
				&rw)
			return
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(rw, r)
	})
}

func (h AWSConnectionHandler) MiddlewareValidateAWSConnectionPost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		requestid, cl := helper.PrepareContext(r, &rw, h.l)

		c := data.NewAWSConnection(h.cfg)

		err := c.FromJSON(r.Body)
		if err != nil {
			helper.LogDebug(cl, helper.ErrorInvalidJSONSchemaForParameter, err)

			helper.ReturnErrorWithAdditionalInfo(
				cl,
				http.StatusBadRequest,
				helper.ErrorInvalidJSONSchemaForParameter,
				requestid,
				r,
				&rw,
				err)
			return

		}

		err = c.Validate()
		if err != nil {
			helper.LogDebug(cl, helper.ErrorInvalidJSONSchemaForParameter, err)

			helper.ReturnErrorWithAdditionalInfo(
				cl,
				http.StatusBadRequest,
				helper.ErrorInvalidJSONSchemaForParameter,
				requestid,
				r,
				&rw,
				err)
			return

		}

		if c.Connection.ConnectionType != data.NoConnectionType {
			if c.Connection.ConnectionType != data.AWSConnectionType {
				helper.LogDebug(cl, helper.ErrorInvalidConnectionType, helper.ErrNone)

				helper.ReturnError(
					cl,
					http.StatusBadRequest,
					helper.ErrorInvalidConnectionType,
					requestid,
					r,
					&rw)
				return

			}
		}

		// add the connection to the context
		ctx := context.WithValue(r.Context(), KeyAWSConnectionRecord{}, c)
		r = r.WithContext(ctx)

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(rw, r)
	})
}

func (h AWSConnectionHandler) MiddlewareValidateAWSConnectionUpdate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		requestid, cl := helper.PrepareContext(r, &rw, h.l)

		vars := mux.Vars(r)
		connectionid := vars["connectionid"]
		var p data.AWSConnectionPatchWrapper

		if len(connectionid) == 0 {
			helper.LogDebug(cl, helper.ErrorConnectionIDInvalid, helper.ErrNone)

			helper.ReturnError(
				cl,
				http.StatusBadRequest,
				helper.ErrorConnectionIDInvalid,
				requestid,
				r,
				&rw)
			return
		}

		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			helper.LogDebug(cl, helper.ErrorInvalidJSONSchemaForParameter, err)

			helper.ReturnErrorWithAdditionalInfo(
				cl,
				http.StatusBadRequest,
				helper.ErrorInvalidJSONSchemaForParameter,
				requestid,
				r,
				&rw,
				err)
			return

		}

		err = validator.New().Struct(p)
		if err != nil {
			helper.LogDebug(cl, helper.ErrorInvalidJSONSchemaForParameter, err)

			helper.ReturnErrorWithAdditionalInfo(
				cl,
				http.StatusBadRequest,
				helper.ErrorInvalidJSONSchemaForParameter,
				requestid,
				r,
				&rw,
				err)
			return

		}

		// add the connection to the context
		ctx := context.WithValue(r.Context(), KeyAWSConnectionPatchParamsRecord{}, p)
		r = r.WithContext(ctx)

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(rw, r)
	})
}
