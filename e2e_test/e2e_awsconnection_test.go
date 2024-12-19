package e2e_test

const (
	addAWSConnectionTestIterations = 200
	updateAWSConnectionTestLimit   = 10
	addAWSConnectionPath           = "/v1/connectionmgmt/connection/aws"
	getAWSConnectionsPath          = "/v1/connectionmgmt/connections/aws"
	deleteAWSConnectionPath        = "/v1/connectionmgmt/connection/aws"
	updateAWSConnectionsPath       = "/v1/connectionmgmt/connection/aws"
)

func (s *EndToEndSuite) funcAddAWSConnection_Load(rounds int) {
	/*c := http.Client{}

	ip, port := GetIPAndPort()

	var obj data.AWSConnectionPostWrapper

	filePath := "../testdata/connection_jsd_gemalto.json"

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		s.True(false, "Couldnt load json file: "+filePath)
	}

	err = json.Unmarshal(fileContent, &obj)
	if err != nil {
		s.True(false, "Error unmarshalling filecontent into JSON:", err)
	}

	for i := 0; i < rounds; i++ {

		var jc data.AWSConnectionPostWrapper

		str_i := strconv.Itoa(i + 1)
		jc.Name = obj.Name + strHyphen + str_i
		jc.Description = obj.Description + strHyphen + str_i
		jc.URL = obj.URL + "/" + str_i
		jc.Username = obj.Username + strHyphen + str_i
		jc.Password = obj.Password + strHyphen + str_i
		jc.Max_Issue_Description = i + 1
		jc.ProjectId = jc.Max_Issue_Description * 10
		jc.IssueTypeId = jc.ProjectId * 10

		jsonData, err := json.Marshal(jc)
		if err != nil {
			s.True(false, "Error marshalling JSON:", err)
		}

		r, err := c.Post(prefixHTTP+ip+":"+port+addJiraConnectionPath, "application/json", bytes.NewBuffer(jsonData))

		if err != nil {
			fmt.Printf("Get request received error: %s\n", err.Error())
			s.True(false)
		} else {
			if r == nil {
				fmt.Printf("No error but resonse object is nil.\n")
				s.True(false)
			}
		}

		s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %d", http.StatusOK, r.StatusCode)
		requestid := r.Header.Get("X-Request-Id")
		s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

		b, _ := io.ReadAll(r.Body)

		_ = r.Body.Close()

		var rc data.JiraConnection

		err = json.Unmarshal(b, &rc)
		if err != nil {
			s.True(false, "Error unmarshalling response into JSON:", err)
		}

		s.NotEmpty(rc.ID.String(), "Connection ID empty")
		s.Equal(jc.Name, rc.Name, "Unexpected title")
		s.Equal(jc.Description, rc.Description, "Unexpected title")
		s.Equal(rc.ConnectionType, data.JiraConnectionType, "Unexpected connectiontype")
		s.Equal(rc.TestSuccessful, 0, "Unexpected testsuccessful")
		s.Equal(rc.TestError, "", "Unexpected testerror")
		s.Equal(rc.TestedOn, "", "Unexpected testedon")
		s.Equal(rc.LastSuccessfulTest, "", "Unexpected lastsuccessfultest")
		s.NotEmpty(rc.CreatedAt, "createdat empty")
		s.Equal(rc.UpdatedAt, rc.CreatedAt, "Unexpected updatedat. Should be same as CreatedAt")
		s.Equal(jc.URL, rc.URL, "Unexpected URL")
		s.Equal(jc.Username, rc.Username, "Unexpected Username")
		s.Equal(jc.Password, rc.Password, "Unexpected Password")
		s.Equal(jc.Max_Issue_Description, rc.Max_Issue_Description, "Unexpected max_issue_description")
		s.Equal(jc.InsecureAllowed, rc.InsecureAllowed, "Unexpected insecureallowed")
		s.Equal(jc.ProjectId, rc.ProjectId, "Unexpected projectid")
		s.Equal(jc.IssueTypeId, rc.IssueTypeId, "Unexpected issuetypeid")
	}
	*/
}

func (s *EndToEndSuite) funcUpdateAWSConnection_Load() int {
	/*c := http.Client{}
	updateCount := 0

	ip, port := GetIPAndPort()

	for ok := true; ok; {

		r, err := c.Get(prefixHTTP + ip + ":" + port + getJiraConnectionsPath + "?skip=" + strconv.Itoa(updateCount) + "&limit=" + strconv.Itoa(updateJiraConnectionTestLimit))

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

		s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
		requestid := r.Header.Get("X-Request-Id")
		s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

		b, _ := io.ReadAll(r.Body)

		_ = r.Body.Close()

		var rc handlers.JiraConnectionsResponse

		err = json.Unmarshal(b, &rc)
		if err != nil {
			s.True(false, "Error unmarshalling response into JSON:", err)
		}

		if rc.Total == 0 {
			break
		}

		for i, jc := range rc.JiraConnections {

			var obj data.JiraConnectionPatchWrapper

			obj.Name = jc.Name + strPatched
			obj.Description = jc.Description + strPatched
			obj.URL = jc.URL + strPatched
			obj.Username = jc.Username + strPatched
			obj.Password = jc.Password + strPatched
			obj.Max_Issue_Description = i + 2
			obj.ProjectId = jc.Max_Issue_Description * 10
			obj.IssueTypeId = jc.ProjectId * 10

			jsonData, err := json.Marshal(obj)
			if err != nil {
				s.True(false, "Error marshalling JSON:", err)
			}

			req, err := http.NewRequest("PATCH", prefixHTTP+ip+":"+port+updateJiraConnectionsPath+"/"+strings.ToLower(jc.ID.String()), bytes.NewBuffer(jsonData))
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

			s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
			requestid := r.Header.Get("X-Request-Id")
			s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

			b, err := io.ReadAll(r.Body)

			if err != nil {
				s.True(false, "Error getting response.\n")
			}

			_ = r.Body.Close()

			var rc data.JiraConnection

			err = json.Unmarshal(b, &rc)
			if err != nil {
				s.True(false, "Error unmarshalling response into JSON:", err)
			}

			s.Equal(strings.ToLower(jc.ID.String()), strings.ToLower(rc.ID.String()), "ConnectionID not matching. Expected: %s, Actual: %s", strings.ToLower(jc.ID.String()), strings.ToLower(rc.ID.String()))
			s.Equal(obj.Name, rc.Name, "Unexpected title. Expected: %s, Actual: %s", jc.Name, rc.Name)
			s.Equal(obj.Description, rc.Description, "Unexpected Description. Expected %s, Actual: %s", jc.Description, rc.Description)
			s.Equal(rc.ConnectionType, data.JiraConnectionType, "Unexpected connectiontype")
			s.Equal(obj.URL, rc.URL, "Unexpected URL. Expected %s, Actual: %s", jc.URL, rc.URL)
			s.Equal(obj.Username, rc.Username, "Unexpected Username. Expected %s, Actual: %s", jc.Username, rc.Username)
			s.Equal(obj.Password, rc.Password, "Unexpected Password. Expected %s, Actual: %s", jc.Password, rc.Password)
			s.Equal(obj.Max_Issue_Description, rc.Max_Issue_Description, "Unexpected max_issue_description. Expected %d, Actual: %d", jc.Max_Issue_Description, rc.Max_Issue_Description)
			s.Equal(obj.InsecureAllowed, rc.InsecureAllowed, "Unexpected insecureallowed. Expected %d, Actual: %d", jc.InsecureAllowed, rc.InsecureAllowed)
			s.Equal(obj.ProjectId, rc.ProjectId, "Unexpected projectid. Expected %d, Actual: %d", jc.ProjectId, rc.ProjectId)
			s.Equal(obj.IssueTypeId, rc.IssueTypeId, "Unexpected issuetypeid. Expected %d, Actual: %d", jc.IssueTypeId, rc.IssueTypeId)

			updateCount++
		}
	}

	return updateCount*/
	return 0
}

func (s *EndToEndSuite) funcDeleteAWSConnection_Load() int {
	/*c := http.Client{}
	deleteCount := 0

	ip, port := GetIPAndPort()

	var rc handlers.JiraConnectionsResponse

	for ok := true; ok; {

		r, err := c.Get(prefixHTTP + ip + ":" + port + getJiraConnectionsPath + "?limit=" + strconv.Itoa(math.MaxInt64))

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

		s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
		requestid := r.Header.Get("X-Request-Id")
		s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

		b, _ := io.ReadAll(r.Body)

		_ = r.Body.Close()

		err = json.Unmarshal(b, &rc)
		if err != nil {
			s.True(false, "Error unmarshalling response into JSON:", err)
		}

		if rc.Total == 0 {
			break
		}

		for _, jc := range rc.JiraConnections {
			req, err := http.NewRequest("DELETE", prefixHTTP+ip+":"+port+deleteJiraConnectionPath+"/"+strings.ToLower(jc.ID.String()), nil)
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

			b, err := io.ReadAll(r.Body)

			if err != nil {
				s.True(false, "Error getting response.\n")
			}

			_ = r.Body.Close()

			diff := JSONCompare(`{"status": "No Content", "statusCode": 204}`, string(b))
			s.Equal("", diff, "JSON Response comparison failed. Expected no differences. Found: %s", diff)

			deleteCount++
		}
	}

	return deleteCount*/
	return 0
}

func (s *EndToEndSuite) TestPositive_AWSConnection_Load() {
	s.funcAddAWSConnection_Load(addAWSConnectionTestIterations)
	s.funcUpdateAWSConnection_Load()
	deleteCount := s.funcDeleteAWSConnection_Load()
	s.Equal(addAWSConnectionTestIterations, deleteCount, "Added and deleted awsconnection count does not match. Expected: %d, Actual: %d", addAWSConnectionTestIterations, deleteCount)
}
