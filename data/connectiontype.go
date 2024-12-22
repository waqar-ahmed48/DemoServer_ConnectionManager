package data

import (
	"DemoServer_ConnectionManager/helper"
	"bytes"
	"encoding/json"
	"strings"
)

type ConnectionTypeEnum int

const (
	NoConnectionType ConnectionTypeEnum = iota
	AWSConnectionType
)

func (o ConnectionTypeEnum) String() string {
	return operation_toString[o]
}

var operation_toString = map[ConnectionTypeEnum]string{
	NoConnectionType:  strings.ToLower("NoConnectionType"),
	AWSConnectionType: strings.ToLower("AWSConnectionType"),
}

var operation_toID = map[string]ConnectionTypeEnum{
	strings.ToLower(""):                  NoConnectionType,
	strings.ToLower("NoConnectionType"):  NoConnectionType,
	strings.ToLower("AWSConnectionType"): AWSConnectionType,
}

// MarshalJSON marshals the enum as a quoted json string
func (o ConnectionTypeEnum) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(strings.ToLower(operation_toString[o]))
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted json string to the enum value
func (o *ConnectionTypeEnum) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then it will be set to the zero value, 'Created' in this case.
	_, found := operation_toID[strings.ToLower(j)]

	if !found {
		return helper.ErrNotFound
	}

	*o = operation_toID[strings.ToLower(j)]

	return nil
}
