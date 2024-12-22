package e2e_test

import (
	"fmt"
	"io"
	"net/http"
)

const (
	getStatusRequestIDTestIterations = 1000
	getStatusPath                    = "/v1/connectionmgmt/status"
)

func (s *EndToEndSuite) TestPositive_GetStatus_HappyPath() {
	c := http.Client{}

	ip, port := GetIPAndPort()

	r, err := c.Get(prefixHTTP + ip + ":" + port + getStatusPath)

	if err != nil {
		fmt.Printf("Get request received error: %s\n", err.Error())
		s.True(false)
	} else {
		if r == nil {
			fmt.Printf("No error but resonse object is nil.\n")
			s.True(false)
		}
	}

	s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)
	requestid := r.Header.Get("X-Request-Id")
	s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

	b, _ := io.ReadAll(r.Body)

	_ = r.Body.Close()

	diff := JSONCompare(`{"status": "UP", "statusCode": "ConnectionManager_Info_000002"}`, string(b))
	s.Equal("", diff, "JSON Response comparison failed. Expected no differences. Found: %s", diff)
}

func (s *EndToEndSuite) TestPositive_GetStatus_UniqueRequestID() {
	c := http.Client{}

	ip, port := GetIPAndPort()

	requestidMap := map[string]bool{}

	for i := 0; i < getStatusRequestIDTestIterations; i++ {
		r, err := c.Get(prefixHTTP + ip + ":" + port + getStatusPath)

		if err != nil {
			fmt.Printf("Get request received error: %s\n", err.Error())
			s.True(false)
		} else {
			if r == nil {
				fmt.Printf("No error but resonse object is nil.\n")
				s.True(false)
			}
		}

		requestid := r.Header.Get("X-Request-Id")

		s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)

		s.NotEqual(requestid, "", "X-Request-ID Header not returned by endpoint. X-Request-ID received: %s", requestid)

		_, found := requestidMap[requestid]
		s.True(!found, "Duplicate X-Request-ID found. Iteration: %d, RequestID: %s", i, requestid)

		requestidMap[requestid] = true

		_ = r.Body.Close()
	}
}

func (s *EndToEndSuite) TestNegative_PostgresDown_GetStatus_ErrorPath() {
	c := http.Client{}

	ip, port := GetIPAndPort()

	r, err := c.Get(prefixHTTP + ip + ":" + port + getStatusPath)

	if err != nil {
		fmt.Printf("Get request received error: %s\n", err.Error())
		s.True(false)
	} else {
		if r == nil {
			fmt.Printf("No error but resonse object is nil.\n")
			s.True(false)
		}
	}

	s.Equal(http.StatusOK, r.StatusCode, "HTTP Status Code comparison failed. Expected %d, Received: %", http.StatusOK, r.StatusCode)

	b, _ := io.ReadAll(r.Body)
	_ = r.Body.Close()

	diff := JSONCompare(`{"status": "DOWN", "statusCode": "ConnectionManager_Info_000003"}`, string(b))
	s.Equal(diff, "", "JSON Response comparison failed. Expected no differences. Found: %", diff)
}
