package e2e_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/suite"
)

const (
	prefixHTTP         = "http://"
	strPatched         = "_Patched"
	strHyphen          = "-"
	strUnderscore      = "_"
	strDefaultLeaseTTL = "99s"
	strMaxLeaseTTL     = "101s"
	strDefaultRegion   = "us-west-2"
)

type EndToEndSuite struct {
	suite.Suite
}

func TestEndtoEndSuite(t *testing.T) {
	suite.Run(t, new(EndToEndSuite))
}

func GetIPAndPort() (string, string) {
	ip := os.Getenv("DEMOSERVER_CONNECTIONMANAGER_SERVICE_IP")
	port := os.Getenv("DEMOSERVER_CONNECTIONMANAGER_SERVICE_PORT")

	if ip == "" {
		ip = "127.0.0.1"

		fmt.Printf("Environment varaiable DEMOSERVER_CONNECTIONMANAGER_SERVICE_IP not set. Setting it to default: %s\n", ip)
	}

	if port == "" {
		port = "5678"

		fmt.Printf("Environment varaiable DEMOSERVER_CONNECTIONMANAGER_SERVICE_PORT not set. Setting it to default: %s\n", port)
	}

	//fmt.Printf("ip: %s, port: %s\n", ip, port)

	return ip, port
}

// transformJSON transforms any Go string that looks like JSON into
// a generic data structure that represents that JSON input.
// We use an AcyclicTransformer to avoid having the transformer
// apply on outputs of itself (lest we get stuck in infinite recursion).
func TransformJSON(s string) interface{} {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s // use unparseable input as the output
	}
	return v
}

func jsonCompareSkipTimeStamp(p cmp.Path) bool {
	vx := p.Last().String()

	return vx == `["timestamp"]`
}

func JSONCompare(a string, b string) string {
	transformJSON := cmpopts.AcyclicTransformer("TransformJSON", TransformJSON)

	opt := cmp.FilterPath(jsonCompareSkipTimeStamp, cmp.Ignore())

	diff := cmp.Diff(a, b, transformJSON, opt)

	return diff
}
