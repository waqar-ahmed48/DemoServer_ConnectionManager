package handlers

import (
	"DemoServer_ConnectionManager/configuration"
	"DemoServer_ConnectionManager/data"
	"DemoServer_ConnectionManager/datalayer"
	"DemoServer_ConnectionManager/helper"
	"DemoServer_ConnectionManager/secretsmanager"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Response schema for DELETE - DeleteAWSConnection
// swagger:model
type DeleteAWSConnectionResponse struct {
	// Descriptive human readable HTTP status of delete operation.
	// in: status
	Status string `json:"status"`

	// HTTP status code for delete operation.
	// in: statusCode
	StatusCode int `json:"statusCode"`
}

// Response schema for GET - TestAWSConnection
// swagger:model
type TestAWSConnectionResponse struct {
	// connectionid for AWSConnection which was tested.
	// in: id
	ID string `json:"id"`

	// test status descriptive human readable message.
	// in: test_status
	TestStatus string `json:"testStatus"`

	// test_status_code. 1 = connectivity test successful. 0 = connectivity test failed.
	// in: test_status_code
	TestStatusCode int `json:"testStatusCode"`
}

type KeyAWSConnectionRecord struct{}
type KeyAWSConnectionPatchParamsRecord struct{}

type AWSConnectionsResponse struct {
	Skip           int                  `json:"skip"`
	Limit          int                  `json:"limit"`
	Total          int                  `json:"total"`
	AWSConnections []data.AWSConnection `json:"awsconnections"`
}

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
	// GET - AWSConnections
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

	var response AWSConnectionsResponse

	result := h.pd.RODB().Table("aws_connections").
		Select("aws_connections.*, connections.name as connection_name").
		Joins("left join connections on connections.id = aws_connections.id").
		Order("connections.name").
		Scan(&response.AWSConnections)

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

	response.Total = len(response.AWSConnections)
	response.Skip = skip
	response.Limit = limit
	if response.Total == 0 {
		response.AWSConnections = ([]data.AWSConnection{})
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
	// GET - AWSConnection
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

	result := h.pd.RODB().First(&connection, "id = ?", connectionid)

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

	err := json.NewEncoder(w).Encode(connection)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err)
	}
}

func (h *AWSConnectionHandler) TestAWSConnection(w http.ResponseWriter, r *http.Request) {

	// swagger:operation GET /Test AWSConnection TestAWSConnection
	// GET - TestAWSConnection
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

	var response TestAWSConnectionResponse

	result := h.pd.RODB().First(&connection, "id = ?", connectionid)

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

	err := connection.Test()

	if err != nil {
		helper.LogDebug(cl, helper.DebugAWSConnectionTestFailed, err)
	}

	result = h.pd.RWDB().Save(&connection)

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
	// PATCH - AWSConnection
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
	//     "$ref": "#/definitions/AWSConnectionWrapper"
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

	result := h.pd.RODB().First(&connection, "id = ?", connectionid)

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

	if p.Name != "" {
		connection.Connection.Name = p.Name
		connection.Connection.TestSuccessful = 0
	}

	if p.Description != "" {
		connection.Connection.Description = p.Description
		connection.Connection.TestSuccessful = 0
	}

	/*
		updated := false

		for k, v := range params {
			if !strings.EqualFold(k, "connectiontype") {
				if strings.EqualFold(k, "Name") {
					if reflect.TypeOf(v).Kind() != reflect.String {
						helper.LogDebug(cl, helper.ErrorAWSConnectionPatchInvalidValueForTitle, helper.ErrNone)

						helper.ReturnError(
							cl,
							http.StatusInternalServerError,
							helper.ErrorAWSConnectionPatchInvalidValueForTitle,
							requestid,
							r,
							&w)
						return
					} else {
						connection.Name = v.(string)
						updated = true
					}
				} else if strings.EqualFold(k, "Description") {
					if reflect.TypeOf(v).Kind() != reflect.String {
						helper.LogDebug(cl, helper.ErrorAWSConnectionPatchInvalidValueForDescription, helper.ErrNone)

						helper.ReturnError(
							cl,
							http.StatusInternalServerError,
							helper.ErrorAWSConnectionPatchInvalidValueForDescription,
							requestid,
							r,
							&w)
						return
					} else {
						connection.Description = v.(string)
						updated = true
					}
				} else if strings.EqualFold(k, "URL") {
					if reflect.TypeOf(v).Kind() != reflect.String {
						helper.LogDebug(cl, helper.ErrorAWSConnectionPatchInvalidValueForURL, helper.ErrNone)

						helper.ReturnError(
							cl,
							http.StatusInternalServerError,
							helper.ErrorAWSConnectionPatchInvalidValueForURL,
							requestid,
							r,
							&w)
						return
					} else {
						connection.URL = v.(string)
						updated = true
					}
				} else if strings.EqualFold(k, "Username") {
					if reflect.TypeOf(v).Kind() != reflect.String {
						helper.LogDebug(cl, helper.ErrorAWSConnectionPatchInvalidValueForUsername, helper.ErrNone)

						helper.ReturnError(
							cl,
							http.StatusInternalServerError,
							helper.ErrorAWSConnectionPatchInvalidValueForUsername,
							requestid,
							r,
							&w)
						return
					} else {
						connection.Username = v.(string)
						updated = true
					}
				} else if strings.EqualFold(k, "Password") {
					if reflect.TypeOf(v).Kind() != reflect.String {
						helper.LogDebug(cl, helper.ErrorAWSConnectionPatchInvalidValueForPassword, helper.ErrNone)

						helper.ReturnError(
							cl,
							http.StatusInternalServerError,
							helper.ErrorAWSConnectionPatchInvalidValueForPassword,
							requestid,
							r,
							&w)
						return
					} else {
						connection.Password = v.(string)
						updated = true
					}
				} else if strings.EqualFold(k, "Max_Issue_Description") {
					if reflect.TypeOf(v).Kind() != reflect.Float64 {
						helper.LogDebug(cl, helper.ErrorAWSConnectionPatchInvalidValueForMaxIssueDescription, helper.ErrNone)

						helper.ReturnError(
							cl,
							http.StatusInternalServerError,
							helper.ErrorAWSConnectionPatchInvalidValueForMaxIssueDescription,
							requestid,
							r,
							&w)
						return
					} else {
						connection.Max_Issue_Description = int(v.(float64))
						updated = true
					}
				} else if strings.EqualFold(k, "InsecureAllowed") {
					if reflect.TypeOf(v).Kind() != reflect.Float64 {
						helper.LogDebug(cl, helper.ErrorAWSConnectionPatchInvalidValueForInsecureAllowed, helper.ErrNone)

						helper.ReturnError(
							cl,
							http.StatusInternalServerError,
							helper.ErrorAWSConnectionPatchInvalidValueForInsecureAllowed,
							requestid,
							r,
							&w)
						return
					} else {
						connection.InsecureAllowed = int(v.(float64))
						updated = true
					}
				} else if strings.EqualFold(k, "ProjectId") {
					if reflect.TypeOf(v).Kind() != reflect.Float64 {
						helper.LogDebug(cl, helper.ErrorAWSConnectionPatchInvalidValueForProjectID, helper.ErrNone)

						helper.ReturnError(
							cl,
							http.StatusInternalServerError,
							helper.ErrorAWSConnectionPatchInvalidValueForProjectID,
							requestid,
							r,
							&w)
						return
					} else {
						connection.ProjectId = int(v.(float64))
						updated = true
					}
				} else if strings.EqualFold(k, "IssueTypeId") {
					if reflect.TypeOf(v).Kind() != reflect.Float64 {
						helper.LogDebug(cl, helper.ErrorAWSConnectionPatchInvalidValueForIssueTypeID, helper.ErrNone)

						helper.ReturnError(
							cl,
							http.StatusInternalServerError,
							helper.ErrorAWSConnectionPatchInvalidValueForIssueTypeID,
							requestid,
							r,
							&w)
						return
					} else {
						connection.IssueTypeId = int(v.(float64))
						updated = true
					}
				}
			} else {
				if reflect.TypeOf(v).Kind() != reflect.String {
					helper.LogDebug(cl, helper.ErrorAWSConnectionPatchInvalidValueForConnectionType, helper.ErrNone)

					helper.ReturnError(
						cl,
						http.StatusBadRequest,
						helper.ErrorAWSConnectionPatchInvalidValueForConnectionType,
						requestid,
						r,
						&w)
					return
				} else {
					ctype := v.(string)

					if strings.ToLower(ctype) != data.NoConnectionType.String() && ctype != "" {
						if strings.ToLower(ctype) != data.AWSConnectionType.String() {
							helper.LogDebug(cl, helper.ErrorConnectionTypeUpdateNotAllowed, helper.ErrNone)

							helper.ReturnError(
								cl,
								http.StatusBadRequest,
								helper.ErrorConnectionTypeUpdateNotAllowed,
								requestid,
								r,
								&w)
						}
					}
				}
			}
		}

		if updated {
			connection.TestSuccessful = 0
		}
	*/
	result = h.pd.RWDB().Save(&connection)

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

	err := json.NewEncoder(w).Encode(connection)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err)
	}
}

// DeleteAWSConnection deletes a AWSConnection from datastore
func (h *AWSConnectionHandler) DeleteAWSConnection(w http.ResponseWriter, r *http.Request) {

	// swagger:operation DELETE /aws AWSConnection DeleteAWSConnection
	// DELETE - AWSConnection
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
	//         "$ref": "#/definitions/AWSAWSConnectionResponse"
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

	result := h.pd.RWDB().Delete(&connection)

	if result.Error != nil {
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

	if result.RowsAffected != 1 {
		helper.LogDebug(cl, helper.ErrorResourceNotFound, helper.ErrNone)

		helper.ReturnError(
			cl,
			http.StatusInternalServerError,
			helper.ErrorResourceNotFound,
			requestid,
			r,
			&w)
		return
	}

	var response DeleteAWSConnectionResponse
	response.StatusCode = http.StatusNoContent
	response.Status = http.StatusText(response.StatusCode)

	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err)
	}
}

func (h *AWSConnectionHandler) AddAWSConnection(w http.ResponseWriter, r *http.Request) {

	// swagger:operation POST /aws AWSConnection AddAWSConnection
	// POST - AddAWSConnection
	//
	// Endpoint: GET - /v1/connectionmgmt/connection/aws
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
	//     "$ref": "#/definitions/AWSConnectionWrapper"
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

	err := h.vh.AddAWSSecretsEngine(c.VaultPath, c.AccessKey, c.SecretAccessKey, c.DefaultLeaseTTL, c.MaxLeaseTTL, c.DefaultRegion, c.RoleName, c.PolicyARNs)
	if err != nil {
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
	}

	c.Connection.ConnectionType = data.AWSConnectionType

	result := h.pd.RWDB().Create(&c)

	if result.Error != nil {
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

	err = json.NewEncoder(w).Encode(c)

	if err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err)
	}
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
