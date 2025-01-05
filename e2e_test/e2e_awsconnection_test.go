package e2e_test

import (
	"DemoServer_ConnectionManager/data"
	"DemoServer_ConnectionManager/helper"
	"DemoServer_ConnectionManager/utilities"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	AWSConnectionLoadTestIterations = 100
	AWSConnectionLoadTestThreads    = 5
	updateAWSConnectionTestLimit    = 10
	addAWSConnectionPath            = "/v1/connectionmgmt/connection/aws"
	getConnectionsPath              = "/v1/connectionmgmt/connections"
	getAWSConnectionsPath           = "/v1/connectionmgmt/connections/aws"
	testAWSConnectionsPath          = "/v1/connectionmgmt/connection/aws/test"
	credsAWSConnectionsPath         = "/v1/connectionmgmt/connection/aws/creds"
	deleteAWSConnectionPath         = "/v1/connectionmgmt/connection/aws"
	updateAWSConnectionsPath        = "/v1/connectionmgmt/connection/aws"
)

func (s *EndToEndSuite) funcAddAWSConnection_Load(threadID int, rounds int) {

	strThreadID := strUnderscore + strconv.Itoa(threadID) + strUnderscore

	dummyAWSConnectionJsonPath := "../testdata/aws_connection.json"

	dummy := s.funcLoadDummyAWSConnection(dummyAWSConnectionJsonPath)
	ip, port := GetIPAndPort()

	for i := 0; i < rounds; i++ {

		suffix := strThreadID + strconv.Itoa(i)

		s.funcAddAWSConnection(dummy, suffix, ip, port)

		if i%1000 == 0 {
			fmt.Printf("funcAddAWSConnection_Load - ThreadID: %d, Counter : %d\n", threadID, i)
		}
	}

	fmt.Printf("funcAddAWSConnection_Load - ThreadID: %d DONE\n", threadID)
}

func (s *EndToEndSuite) funcUpdateAWSConnection_Load() {
	c := http.Client{}
	updateCount := 0

	ip, port := GetIPAndPort()

	for ok := true; ok; {

		r, err := c.Get(prefixHTTP + ip + ":" + port + getAWSConnectionsPath + "?skip=" + strconv.Itoa(updateCount) + "&limit=" + strconv.Itoa(updateAWSConnectionTestLimit))

		if err != nil {
			s.Require().Truef(false, "Get request received error: %s\n", err.Error())
		} else {
			if r == nil {
				s.Require().Truef(false, "No error but resonse object is nil.\n")
			}
		}

		defer func() { _ = r.Body.Close() }()

		s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
		requestid := r.Header.Get("X-Request-Id")
		s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

		b, _ := io.ReadAll(r.Body)

		var rc data.AWSConnectionsResponse

		err = json.Unmarshal(b, &rc)
		if err != nil {
			s.True(false, "Error unmarshalling response into JSON:", err)
		}

		if len(rc.AWSConnections) == 0 {
			break
		}

		suffix := strPatched

		for _, jc := range rc.AWSConnections {

			var obj data.AWSConnectionPatchWrapper
			var connection data.ConnectionPatchWrapper

			nameWithSuffix := (jc.Connection.Name + suffix)
			connection.Name = &nameWithSuffix

			descriptionWithSuffix := jc.Connection.Description + suffix
			connection.Description = &descriptionWithSuffix

			obj.Connection = &connection

			accesskeyWithSuffix := jc.AccessKey + suffix
			obj.AccessKey = &accesskeyWithSuffix

			obj.DefaultRegion = &strDefaultRegion

			secretaccesskeyWithSuffix := "Dummy Secret Key Value_" + suffix
			obj.SecretAccessKey = &secretaccesskeyWithSuffix

			obj.DefaultLeaseTTL = &strDefaultLeaseTTL
			obj.MaxLeaseTTL = &strMaxLeaseTTL
			obj.CredentialType = &jc.CredentialType
			obj.PolicyARNs = append(jc.PolicyARNs, "arn:aws:iam::aws:policy/AmazonS3FullAccess")

			jsonData, err := json.Marshal(obj)
			if err != nil {
				s.True(false, "Error marshalling JSON:", err)
			}

			req, err := http.NewRequest("PATCH", prefixHTTP+ip+":"+port+updateAWSConnectionsPath+"/"+strings.ToLower(jc.ID.String()), bytes.NewBuffer(jsonData))
			if err != nil {
				s.True(false, "UPDATE request creation failed")
			}

			req.Header.Set("Content-Type", "application/json")

			r, err := c.Do(req)

			if err != nil {
				s.True(false, "UPDATE request received error: %s\n", err.Error())
			} else {
				if r == nil {
					s.True(false, "No error but resonse object is nil.\n")
				}
			}

			defer func() { _ = r.Body.Close() }()

			s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
			requestid := r.Header.Get("X-Request-Id")
			s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

			b, err := io.ReadAll(r.Body)

			if err != nil {
				s.True(false, "Error getting response.\n")
			}

			_ = r.Body.Close()

			var rc data.AWSConnection

			err = json.Unmarshal(b, &rc)
			if err != nil {
				s.True(false, "Error unmarshalling response into JSON:", err)
			}

			s.Equal(strings.ToLower(jc.ID.String()), strings.ToLower(rc.ID.String()), "ConnectionID not matching. Expected: %s, Actual: %s", strings.ToLower(jc.ID.String()), strings.ToLower(rc.ID.String()))
			s.Equal(*obj.Connection.Name, rc.Connection.Name, "Unexpected title. Expected: %s, Actual: %s", *obj.Connection.Name, rc.Connection.Name)
			s.Equal(*obj.Connection.Description, rc.Connection.Description, "Unexpected Description. Expected %s, Actual: %s", *obj.Connection.Description, rc.Connection.Description)
			s.Equal(rc.Connection.ConnectionType, data.AWSConnectionType, "Unexpected connectiontype")
			s.Equal(rc.Connection.TestSuccessful, 0, "Unexpected TestSuccessful state. Expected: %d, Actual: %d", 0, rc.Connection.TestSuccessful)
			s.Equal(rc.Connection.TestError, "", "Unexpected TestError state. Expected: %s, Actual: %s", "EmptyString", rc.Connection.TestError)
			s.Equal(*obj.AccessKey, rc.AccessKey, "Unexpected URL. Expected %s, Actual: %s", *obj.AccessKey, rc.AccessKey)
			s.Equal(strDefaultRegion, rc.DefaultRegion, "Unexpected DefaultRegion. Expected %s, Actual: %s", strDefaultRegion, rc.DefaultRegion)
			s.Equal(*obj.DefaultLeaseTTL, rc.DefaultLeaseTTL, "Unexpected DefaultLeaseTTL. Expected %s, Actual: %s", *obj.DefaultLeaseTTL, rc.DefaultLeaseTTL)
			s.Equal(*obj.MaxLeaseTTL, rc.MaxLeaseTTL, "Unexpected MaxLeaseTTL. Expected %s, Actual: %s", *obj.MaxLeaseTTL, rc.MaxLeaseTTL)
			s.Equal(jc.RoleName, rc.RoleName, "Unexpected RoleName. Expected %s, Actual: %s", jc.RoleName, rc.RoleName)
			s.Equal(jc.CredentialType, rc.CredentialType, "Unexpected CredentialType. Expected %s, Actual: %s", jc.CredentialType, rc.CredentialType)
			s.Equal(strings.Join(obj.PolicyARNs, "_"), strings.Join(rc.PolicyARNs, "_"), "Unexpected PolicyARNs. Expected %s, Actual: %s", strings.Join(obj.PolicyARNs, "_"), strings.Join(rc.PolicyARNs, "_"))

			updateCount++

			if updateCount%1000 == 0 {
				fmt.Printf("funcUpdateConnection_Load - Counter : %d\n", updateCount)
			}
		}
	}

}

func (s *EndToEndSuite) funcDeleteAWSConnections_All(limit ...int) {

	limitValue := math.MaxInt64

	if len(limit) > 0 {
		limitValue = limit[0]
	}

	c := http.Client{}

	ip, port := GetIPAndPort()

	var rc data.AWSConnectionsResponse

	for ok := true; ok; {

		r, err := c.Get(prefixHTTP + ip + ":" + port + getAWSConnectionsPath + "?limit=" + strconv.Itoa(limitValue))

		if err != nil {
			strResponse := ""

			if r != nil {
				b, _ := io.ReadAll(r.Body)
				strResponse = string(b)
			}

			s.True(false, "Get request received error: %s, Response: %s", err.Error(), strResponse)
		} else {
			if r == nil {
				s.True(false, "No error but response object is nil.")

			}
		}

		defer func() { _ = r.Body.Close() }()

		s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %d", http.StatusOK, r.StatusCode)
		requestid := r.Header.Get("X-Request-Id")
		s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

		b, _ := io.ReadAll(r.Body)
		err = json.Unmarshal(b, &rc)
		if err != nil {
			s.True(false, "Error unmarshalling response into JSON:", err)
		}

		if len(rc.AWSConnections) == 0 {
			break
		}

		for i, jc := range rc.AWSConnections {
			req, err := http.NewRequest("DELETE", prefixHTTP+ip+":"+port+deleteAWSConnectionPath+"/"+strings.ToLower(jc.ID.String()), nil)
			if err != nil {
				s.True(false, "Delete request creation failed")
			}

			r, err := c.Do(req)

			if err != nil {
				s.True(false, "DELETE request received error: %s\n", err.Error())
			} else {
				if r == nil {
					s.True(false, "No error but resonse object is nil.\n")
				}
			}

			s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
			requestid := r.Header.Get("X-Request-Id")
			s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

			defer func() { _ = r.Body.Close() }()

			b, err := io.ReadAll(r.Body)

			if err != nil {
				s.True(false, "Error getting response.\n")
			}

			_ = r.Body.Close()

			diff := JSONCompare(`{"status": "No Content", "statusCode": 204}`, string(b))
			s.Equal("", diff, "JSON Response comparison failed. Expected no differences. Found: %s", diff)

			if i%1000 == 0 {
				fmt.Printf("funcDeleteConnection_Load - Counter : %d\n", i)
			}
		}
	}
}

func (s *EndToEndSuite) funcGetConnectionCount() int {
	c := http.Client{}

	ip, port := GetIPAndPort()

	r, err := c.Get(prefixHTTP + ip + ":" + port + getConnectionsPath)

	if err != nil {
		fmt.Printf("Get request received error: %s\n", err.Error())
		s.True(false)
	} else {
		if r == nil {
			fmt.Printf("No error but resonse object is nil.\n")
			s.True(false)
		}
	}

	defer func() { _ = r.Body.Close() }()

	s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %d", http.StatusOK, r.StatusCode)
	requestid := r.Header.Get("X-Request-Id")
	s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

	b, _ := io.ReadAll(r.Body)

	var rc data.ConnectionsResponse

	err = json.Unmarshal(b, &rc)
	if err != nil {
		s.True(false, "Error unmarshalling response into JSON:", err)
	}

	return rc.Total
}

func (s *EndToEndSuite) funcGetAWSConnectionCount() int {
	c := http.Client{}

	ip, port := GetIPAndPort()

	r, err := c.Get(prefixHTTP + ip + ":" + port + getAWSConnectionsPath)

	if err != nil {
		fmt.Printf("Get request received error: %s\n", err.Error())
		s.True(false)
	} else {
		if r == nil {
			fmt.Printf("No error but resonse object is nil.\n")
			s.True(false)
		}
	}

	defer func() { _ = r.Body.Close() }()

	s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %d", http.StatusOK, r.StatusCode)
	requestid := r.Header.Get("X-Request-Id")
	s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

	b, _ := io.ReadAll(r.Body)

	var rc data.AWSConnectionsResponse

	err = json.Unmarshal(b, &rc)
	if err != nil {
		s.True(false, "Error unmarshalling response into JSON:", err)
	}

	return rc.Total
}

func (s *EndToEndSuite) TestPositive_AWSConnection_Load() {
	s.funcDeleteAWSConnections_All()

	cStartCount := s.funcGetConnectionCount()
	awsStartCount := s.funcGetAWSConnectionCount()
	utilities.CallMultiThreadedFunc(s.funcAddAWSConnection_Load, AWSConnectionLoadTestIterations, AWSConnectionLoadTestThreads)

	s.funcUpdateAWSConnection_Load()

	s.funcDeleteAWSConnections_All()

	cEndCount := s.funcGetConnectionCount()
	awsEndCount := s.funcGetAWSConnectionCount()

	s.Equal(cStartCount, cEndCount, "Start and end count of connections should match. Expected: %d, Actual: %d", cStartCount, cEndCount)
	s.Equal(awsStartCount, awsEndCount, "Start and end count of AWS Connections should match. Expected: %d, Actual: %d", awsStartCount, awsEndCount)
}

func (s *EndToEndSuite) funcTestAWSConnection(connectionid string) {
	c := http.Client{}

	ip, port := GetIPAndPort()

	r, err := c.Get(prefixHTTP + ip + ":" + port + testAWSConnectionsPath + "/" + connectionid)

	if err != nil {
		fmt.Printf("Get request received error: %s\n", err.Error())
		s.True(false)
	} else {
		if r == nil {
			fmt.Printf("No error but resonse object is nil.\n")
			s.True(false)
		}
	}

	defer func() { _ = r.Body.Close() }()

	s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
	requestid := r.Header.Get("X-Request-Id")
	s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

	b, _ := io.ReadAll(r.Body)

	var rc data.TestAWSConnectionResponse

	err = json.Unmarshal(b, &rc)
	if err != nil {
		s.True(false, "Error unmarshalling response into JSON:", err)
	}

	s.NotEmpty(rc.ID, "ID empty")
	s.Equal(rc.TestStatus, "", "Test Status comparison failed. Expected Empty String, Received: %s", rc.TestStatus)
	s.Equal(rc.TestStatusCode, 1, "TestStatusCode comparison failed. Expected: %d, Received: %d", 1, rc.TestStatusCode)
}

func (s *EndToEndSuite) funcCredsAWSConnection_Negative(connectionid string) {
	c := http.Client{}

	ip, port := GetIPAndPort()

	r, err := c.Get(prefixHTTP + ip + ":" + port + credsAWSConnectionsPath + "/" + connectionid)

	if err != nil {
		fmt.Printf("Get request received error: %s\n", err.Error())
		s.True(false)
	} else {
		if r == nil {
			fmt.Printf("No error but resonse object is nil.\n")
			s.True(false)
		}
	}

	defer func() { _ = r.Body.Close() }()

	s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
	requestid := r.Header.Get("X-Request-Id")
	s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

	b, _ := io.ReadAll(r.Body)

	var rc helper.ErrorResponse

	err = json.Unmarshal(b, &rc)
	if err != nil {
		s.True(false, "Error unmarshalling response into JSON:", err)
	}

	s.NotEmpty(rc.Timestamp, "Timestamp empty")
	s.Equal(rc.Status, 500, "Status. Expected: %d, Received: %d", 500, rc.Status)
	s.Equal(rc.ErrorCode, "ConnectionManager_Err_000032", "Unexpected error code. Expected: %s, Received: %s", "ConnectionManager_Err_000032", rc.ErrorCode)
	s.NotEmpty(rc.ErrorDescription, "ErrorDescription empty")
	s.NotEmpty(rc.Endpoint, "Endpoint empty")
	s.NotEmpty(rc.Method, "Method empty")
	s.NotEmpty(rc.RequestID, "RequestID empty")
}

func (s *EndToEndSuite) funcCredsAWSConnection(connectionid string) {
	c := http.Client{}

	ip, port := GetIPAndPort()

	r, err := c.Get(prefixHTTP + ip + ":" + port + credsAWSConnectionsPath + "/" + connectionid)

	if err != nil {
		fmt.Printf("Get request received error: %s\n", err.Error())
		s.True(false)
	} else {
		if r == nil {
			fmt.Printf("No error but response object is nil.\n")
			s.True(false)
		}
	}

	defer func() { _ = r.Body.Close() }()

	s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
	requestid := r.Header.Get("X-Request-Id")
	s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

	b, _ := io.ReadAll(r.Body)

	var rc data.CredsAWSConnectionResponse

	err = json.Unmarshal(b, &rc)
	if err != nil {
		s.True(false, "Error unmarshalling response into JSON:", err)
	}

	s.NotEmpty(rc.ConnectionID, "ID empty")
	s.Equal(rc.ConnectionID, connectionid, "Unexpected ConnectionID in response. Expected: %s, Received: %s", connectionid, rc.ConnectionID)
	s.NotEmpty(rc.LeaseID, "LeaseID empty")
	s.NotEmpty(rc.Data.AccessKey, "AccessKey empty")
	s.NotEmpty(rc.Data.SecretKey, "AccessKey empty")
	s.NotEmpty(rc.Data.SessionToken, "SessionToken is not empty. It was supposed to be empty")
}

func (s *EndToEndSuite) funcGetAWSConnection_Nth(skip int) *data.AWSConnectionResponseWrapper {
	c := http.Client{}

	ip, port := GetIPAndPort()

	r, err := c.Get(prefixHTTP + ip + ":" + port + getAWSConnectionsPath + "?skip=" + strconv.Itoa(skip))

	if err != nil {
		fmt.Printf("Get request received error: %s\n", err.Error())
		s.True(false)
	} else {
		if r == nil {
			fmt.Printf("No error but resonse object is nil.\n")
			s.True(false)
		}
	}

	defer func() { _ = r.Body.Close() }()

	s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
	requestid := r.Header.Get("X-Request-Id")
	s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

	b, _ := io.ReadAll(r.Body)

	var rc data.AWSConnectionsResponse

	err = json.Unmarshal(b, &rc)
	if err != nil {
		s.True(false, "Error unmarshalling response into JSON:", err)
	}

	if len(rc.AWSConnections) == 0 {
		return nil
	} else {
		return &rc.AWSConnections[0]
	}
}

func (s *EndToEndSuite) funcGetConnection_Nth(skip int) *data.Connection {
	c := http.Client{}

	ip, port := GetIPAndPort()

	r, err := c.Get(prefixHTTP + ip + ":" + port + getConnectionsPath + "?skip=" + strconv.Itoa(skip))

	if err != nil {
		fmt.Printf("Get request received error: %s\n", err.Error())
		s.True(false)
	} else {
		if r == nil {
			fmt.Printf("No error but resonse object is nil.\n")
			s.True(false)
		}
	}

	defer func() { _ = r.Body.Close() }()

	s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
	requestid := r.Header.Get("X-Request-Id")
	s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

	b, _ := io.ReadAll(r.Body)

	var rc data.ConnectionsResponse

	err = json.Unmarshal(b, &rc)
	if err != nil {
		s.True(false, "Error unmarshalling response into JSON:", err)
	}

	if rc.Total == 0 {
		return nil
	} else {
		return &rc.Connections[0]
	}
}

func (s *EndToEndSuite) funcLoadDummyAWSConnection(filePath ...string) data.AWSConnectionPostWrapper {

	filePathValue := "../testdata/aws_connection.json"

	if len(filePath) > 0 {
		filePathValue = filePath[0]
	}

	var obj data.AWSConnectionPostWrapper

	fileContent, err := os.ReadFile(filePathValue)
	if err != nil {
		s.True(false, "Couldnt load json file: "+filePathValue)
	}

	err = json.Unmarshal(fileContent, &obj)
	if err != nil {
		s.True(false, "Error unmarshalling filecontent into JSON:", err)
	}

	return obj
}

func (s *EndToEndSuite) funcAddAWSConnection(dummy data.AWSConnectionPostWrapper, suffix string, ip string, port string) string {
	c := http.Client{}

	var jc data.AWSConnectionPostWrapper

	jc.Connection.Name = dummy.Connection.Name + suffix
	jc.Connection.Description = dummy.Connection.Description + suffix
	jc.AccessKey = dummy.AccessKey + suffix
	jc.SecretAccessKey = dummy.SecretAccessKey + suffix
	jc.DefaultRegion = dummy.DefaultRegion + suffix
	jc.DefaultLeaseTTL = dummy.DefaultLeaseTTL
	jc.MaxLeaseTTL = dummy.MaxLeaseTTL
	jc.RoleName = dummy.RoleName + suffix
	jc.CredentialType = dummy.CredentialType
	jc.PolicyARNs = dummy.PolicyARNs

	jsonData, err := json.Marshal(jc)
	if err != nil {
		s.True(false, "Error marshalling JSON:", err)
	}

	r, err := c.Post(prefixHTTP+ip+":"+port+addAWSConnectionPath, "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		fmt.Printf("Get request received error: %s\n", err.Error())
		s.True(false)
	} else {
		if r == nil {
			fmt.Printf("No error but resonse object is nil.\n")
			s.True(false)
		}
	}

	defer func() { _ = r.Body.Close() }()

	s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %d", http.StatusOK, r.StatusCode)
	requestid := r.Header.Get("X-Request-Id")
	s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

	b, _ := io.ReadAll(r.Body)

	var rc data.AWSConnectionResponseWrapper

	err = json.Unmarshal(b, &rc)
	if err != nil {
		s.True(false, "Error unmarshalling response into JSON:", err)
	}

	s.NotEmpty(rc.ID.String(), "ID empty")
	s.NotEmpty(rc.CreatedAt, "createdat empty")
	s.Equal(rc.UpdatedAt, rc.CreatedAt, "Unexpected updatedat. Should be same as CreatedAt")
	s.NotEmpty(rc.ID.String(), "ConnectionID empty")
	s.Equal(rc.ConnectionID.String(), rc.Connection.ID.String(), "ConnectionID should be same as Connection.ID")
	s.Equal(rc.Connection.Name, jc.Connection.Name, "Unexpected Name")
	s.Equal(rc.Connection.Description, jc.Connection.Description, "Unexpected title")
	s.Equal(rc.Connection.ConnectionType, data.AWSConnectionType, "Unexpected connectiontype")
	s.Equal(rc.Connection.TestSuccessful, 0, "Unexpected testsuccessful")
	s.Equal(rc.Connection.TestError, "", "Unexpected testerror")
	s.Equal(rc.Connection.TestedOn, "", "Unexpected testedon")
	s.Equal(rc.Connection.LastSuccessfulTest, "", "Unexpected lastsuccessfultest")

	s.Equal(rc.AccessKey, jc.AccessKey, "Unexpected URL")
	s.Equal(rc.DefaultRegion, jc.DefaultRegion, "Unexpected DefaultRegion")
	s.Equal(rc.DefaultLeaseTTL, jc.DefaultLeaseTTL, "Unexpected DefaultLeaseTTL")
	s.Equal(rc.MaxLeaseTTL, jc.MaxLeaseTTL, "Unexpected MaxLeaseTTL")
	s.Equal(rc.RoleName, jc.RoleName, "Unexpected RoleName")
	s.Equal(rc.CredentialType, jc.CredentialType, "Unexpected CredentialType")
	s.Equal(strings.Join(rc.PolicyARNs, "_"), strings.Join(jc.PolicyARNs, "_"), "Unexpected PolicyARNs")

	return rc.ID.String()
}

func (s *EndToEndSuite) func_VerifyAWSConnection_Nth(i int, suffix string) {
	suffix = suffix + strconv.Itoa(i)
	c := s.funcGetAWSConnection_Nth(i)
	if !strings.Contains(c.Connection.Name, suffix) {
		s.True(false, "Mismatched Name. Expected %s, Actual: %s ", suffix, c.Connection.Name)
	}
}

func (s *EndToEndSuite) func_VerifyConnection_Nth(i int, suffix string) {
	suffix = suffix + strconv.Itoa(i)
	c := s.funcGetConnection_Nth(i)
	if c != nil {
		if !strings.Contains(c.Name, suffix) {
			s.True(false, "Mismatched Name. Expected %s, Actual: %s ", suffix, c.Name)
		}
	} else {
		s.Truef(false, "Resource not found. Null resource returned")
	}
}

func (s *EndToEndSuite) func_VerifyConnection_BeyondLimit(i int) {
	c := s.funcGetConnection_Nth(i)
	if c != nil {
		s.Truef(false, "Resource found. It was expected to be nil")
	}
}

func (s *EndToEndSuite) func_VerifyAWSConnection_BeyondLimit(i int) {
	c := s.funcGetAWSConnection_Nth(i)
	if c != nil {
		s.Truef(false, "Resource found. It was expected to be nil")
	}
}

func (s *EndToEndSuite) TestPositive_Functional_AWSConnectionsGet_Skip() {
	threadID := 1
	strThreadID := strUnderscore + strconv.Itoa(threadID) + strUnderscore

	s.funcDeleteAWSConnections_All()

	dummy := s.funcLoadDummyAWSConnection("../testdata/aws_connection.json")
	ip, port := GetIPAndPort()

	for i := 0; i < 5; i++ {

		suffix := strThreadID + strconv.Itoa(i)

		s.funcAddAWSConnection(dummy, suffix, ip, port)
	}

	s.func_VerifyAWSConnection_Nth(0, strThreadID)
	s.func_VerifyAWSConnection_Nth(2, strThreadID)
	s.func_VerifyAWSConnection_Nth(4, strThreadID)
	s.func_VerifyAWSConnection_BeyondLimit(5)

	s.funcDeleteAWSConnections_All()
}

/*
	func (s *EndToEndSuite) TestPositive_Functional_AWSConnectionTest() {
		threadID := 1
		strThreadID := strUnderscore + strconv.Itoa(threadID) + strUnderscore

		s.funcDeleteAWSConnections_All()

		dummy := s.funcLoadDummyAWSConnection("../testdata/aws_connection.json")
		ip, port := GetIPAndPort()
		suffix := strThreadID + strconv.Itoa(1)

		connectionid := s.funcAddAWSConnection(dummy, suffix, ip, port)

		s.funcTestAWSConnection(connectionid)

		s.funcDeleteAWSConnections_All()
	}

	func (s *EndToEndSuite) TestPositive_Functional_AWSConnectionCreds() {
		threadID := 1
		strThreadID := strUnderscore + strconv.Itoa(threadID) + strUnderscore

		s.funcDeleteAWSConnections_All()

		dummy := s.funcLoadDummyAWSConnection("../testdata/aws_connection.json")
		ip, port := GetIPAndPort()
		suffix := strThreadID + strconv.Itoa(1)

		connectionid := s.funcAddAWSConnection(dummy, suffix, ip, port)

		s.funcTestAWSConnection(connectionid)

		s.funcCredsAWSConnection(connectionid)

		s.funcDeleteAWSConnections_All()
	}

	func (s *EndToEndSuite) TestNegative_Functional_AWSConnectionCreds() {
		threadID := 1
		strThreadID := strUnderscore + strconv.Itoa(threadID) + strUnderscore

		s.funcDeleteAWSConnections_All()

		dummy := s.funcLoadDummyAWSConnection("../testdata/aws_connection.json")
		ip, port := GetIPAndPort()
		suffix := strThreadID + strconv.Itoa(1)

		connectionid := s.funcAddAWSConnection(dummy, suffix, ip, port)

		s.funcCredsAWSConnection_Negative(connectionid)

		s.funcDeleteAWSConnections_All()
	}
*/
func (s *EndToEndSuite) TestPositive_Functional_AWSConnectionsGet_Limit() {
	limit := 5
	total := limit * 2
	threadID := 1
	strThreadID := strUnderscore + strconv.Itoa(threadID) + strUnderscore

	s.funcDeleteAWSConnections_All()

	dummy := s.funcLoadDummyAWSConnection("../testdata/aws_connection.json")
	ip, port := GetIPAndPort()

	for i := 0; i < total; i++ {

		suffix := strThreadID + strconv.Itoa(i)

		s.funcAddAWSConnection(dummy, suffix, ip, port)
	}

	c := http.Client{}

	r, err := c.Get(prefixHTTP + ip + ":" + port + getAWSConnectionsPath + "?limit=" + strconv.Itoa(limit))

	if err != nil {
		fmt.Printf("Get request received error: %s\n", err.Error())
		s.True(false)
	} else {
		if r == nil {
			fmt.Printf("No error but resonse object is nil.\n")
			s.True(false)
		}
	}

	defer func() { _ = r.Body.Close() }()

	s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
	requestid := r.Header.Get("X-Request-Id")
	s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

	b, _ := io.ReadAll(r.Body)

	var rc data.AWSConnectionsResponse

	err = json.Unmarshal(b, &rc)
	if err != nil {
		s.True(false, "Error unmarshalling response into JSON:", err)
	}

	if rc.Total != limit {
		s.True(false, "Incorrect number of AWSConnections Returned. Expected: %d, Actual: %d", limit, rc.Total)
	} else {
		for i := 0; i < limit; i++ {

			suffix := strThreadID + strconv.Itoa(i)

			if !strings.Contains(rc.AWSConnections[i].Connection.Name, suffix) {
				s.True(false, "Unexpected AWS Connection name. Expected: %s, Actual: %s", suffix, rc.AWSConnections[i].Connection.Name)
			}
		}
	}

	s.funcDeleteAWSConnections_All()
}

func (s *EndToEndSuite) TestPositive_Functional_AWSConnectionsGet_SkipAndLimit() {
	skip := 3
	limit := 5
	total := limit * 2
	threadID := 1
	strThreadID := strUnderscore + strconv.Itoa(threadID) + strUnderscore

	s.funcDeleteAWSConnections_All()

	dummy := s.funcLoadDummyAWSConnection("../testdata/aws_connection.json")
	ip, port := GetIPAndPort()

	for i := 0; i < total; i++ {

		suffix := strThreadID + strconv.Itoa(i)

		s.funcAddAWSConnection(dummy, suffix, ip, port)
	}

	c := http.Client{}

	r, err := c.Get(prefixHTTP + ip + ":" + port + getAWSConnectionsPath + "?skip=" + strconv.Itoa(skip) + "&limit=" + strconv.Itoa(limit))

	if err != nil {
		fmt.Printf("Get request received error: %s\n", err.Error())
		s.True(false)
	} else {
		if r == nil {
			fmt.Printf("No error but resonse object is nil.\n")
			s.True(false)
		}
	}

	defer func() { _ = r.Body.Close() }()

	s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
	requestid := r.Header.Get("X-Request-Id")
	s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

	b, _ := io.ReadAll(r.Body)

	var rc data.AWSConnectionsResponse

	err = json.Unmarshal(b, &rc)
	if err != nil {
		s.True(false, "Error unmarshalling response into JSON:", err)
	}

	if len(rc.AWSConnections) != limit {
		s.True(false, "Incorrect number of AWSConnections Returned. Expected: %d, Actual: %d", limit, len(rc.AWSConnections))
	} else {
		for i := 0; i < limit; i++ {

			suffix := strThreadID + strconv.Itoa(i+skip)

			if !strings.Contains(rc.AWSConnections[i].Connection.Name, suffix) {
				s.True(false, "Unexpected AWS Connection name. Expected: %s, Actual: %s", suffix, rc.AWSConnections[i].Connection.Name)
			}
		}
	}

	s.funcDeleteAWSConnections_All()
}

func (s *EndToEndSuite) TestPositive_Functional_ConnectionsGet_Skip() {
	threadID := 1
	strThreadID := strUnderscore + strconv.Itoa(threadID) + strUnderscore

	s.funcDeleteAWSConnections_All()

	dummy := s.funcLoadDummyAWSConnection("../testdata/aws_connection.json")
	ip, port := GetIPAndPort()

	for i := 0; i < 5; i++ {

		suffix := strThreadID + strconv.Itoa(i)

		s.funcAddAWSConnection(dummy, suffix, ip, port)
	}

	s.func_VerifyConnection_Nth(0, strThreadID)
	s.func_VerifyConnection_Nth(2, strThreadID)
	s.func_VerifyConnection_Nth(4, strThreadID)
	s.func_VerifyConnection_BeyondLimit(5)

	s.funcDeleteAWSConnections_All()
}

func (s *EndToEndSuite) TestPositive_Functional_ConnectionsGet_Limit() {
	limit := 5
	total := limit * 2
	threadID := 1
	strThreadID := strUnderscore + strconv.Itoa(threadID) + strUnderscore

	s.funcDeleteAWSConnections_All()

	dummy := s.funcLoadDummyAWSConnection("../testdata/aws_connection.json")
	ip, port := GetIPAndPort()

	for i := 0; i < total; i++ {

		suffix := strThreadID + strconv.Itoa(i)

		s.funcAddAWSConnection(dummy, suffix, ip, port)
	}

	c := http.Client{}

	r, err := c.Get(prefixHTTP + ip + ":" + port + getConnectionsPath + "?limit=" + strconv.Itoa(limit))

	if err != nil {
		fmt.Printf("Get request received error: %s\n", err.Error())
		s.True(false)
	} else {
		if r == nil {
			fmt.Printf("No error but resonse object is nil.\n")
			s.True(false)
		}
	}

	defer func() { _ = r.Body.Close() }()

	s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
	requestid := r.Header.Get("X-Request-Id")
	s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

	b, _ := io.ReadAll(r.Body)

	var rc data.ConnectionsResponse

	err = json.Unmarshal(b, &rc)
	if err != nil {
		s.True(false, "Error unmarshalling response into JSON:", err)
	}

	if len(rc.Connections) != limit {
		s.True(false, "Incorrect number of Connections Returned. Expected: %d, Actual: %d", limit, rc.Total)
	} else {
		for i := 0; i < limit; i++ {

			suffix := strThreadID + strconv.Itoa(i)

			if !strings.Contains(rc.Connections[i].Name, suffix) {
				s.True(false, "Unexpected Connection name. Expected: %s, Actual: %s", suffix, rc.Connections[i].Name)
			}
		}
	}

	s.funcDeleteAWSConnections_All()
}

func (s *EndToEndSuite) TestPositive_Functional_ConnectionsGet_SkipAndLimit() {
	skip := 3
	limit := 5
	total := limit * 2
	threadID := 1
	strThreadID := strUnderscore + strconv.Itoa(threadID) + strUnderscore

	s.funcDeleteAWSConnections_All()

	dummy := s.funcLoadDummyAWSConnection("../testdata/aws_connection.json")
	ip, port := GetIPAndPort()

	for i := 0; i < total; i++ {

		suffix := strThreadID + strconv.Itoa(i)

		s.funcAddAWSConnection(dummy, suffix, ip, port)
	}

	c := http.Client{}

	r, err := c.Get(prefixHTTP + ip + ":" + port + getConnectionsPath + "?skip=" + strconv.Itoa(skip) + "&limit=" + strconv.Itoa(limit))

	if err != nil {
		fmt.Printf("Get request received error: %s\n", err.Error())
		s.True(false)
	} else {
		if r == nil {
			fmt.Printf("No error but resonse object is nil.\n")
			s.True(false)
		}
	}

	defer func() { _ = r.Body.Close() }()

	s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
	requestid := r.Header.Get("X-Request-Id")
	s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

	b, _ := io.ReadAll(r.Body)

	var rc data.ConnectionsResponse

	err = json.Unmarshal(b, &rc)
	if err != nil {
		s.True(false, "Error unmarshalling response into JSON:", err)
	}

	if rc.Total != limit {
		s.True(false, "Incorrect number of Connections Returned. Expected: %d, Actual: %d", limit, rc.Total)
	} else {
		for i := 0; i < limit; i++ {

			suffix := strThreadID + strconv.Itoa(i+skip)

			if !strings.Contains(rc.Connections[i].Name, suffix) {
				s.True(false, "Unexpected AWS Connection name. Expected: %s, Actual: %s", suffix, rc.Connections[i].Name)
			}
		}
	}

	s.funcDeleteAWSConnections_All(3)
}
