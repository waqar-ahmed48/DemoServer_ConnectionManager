package data

import (
	"DemoServer_ConnectionManager/helper"
	"bytes"
	"encoding/json"
	"strings"
)

type ActionStatusTypeEnum int

const (
	NoStatus ActionStatusTypeEnum = iota
	Successful
	Failed
)

func (o ActionStatusTypeEnum) String() string {
	return actionstatus_toString[o]
}

var actionstatus_toString = map[ActionStatusTypeEnum]string{
	NoStatus:   strings.ToLower(""),
	Successful: strings.ToLower("Successful"),
	Failed:     strings.ToLower("Failed"),
}

var actionstatus_toID = map[string]ActionStatusTypeEnum{
	strings.ToLower(""):           NoStatus,
	strings.ToLower("Successful"): Successful,
	strings.ToLower("Failed"):     Failed,
}

// MarshalJSON marshals the enum as a quoted json string
func (o ActionStatusTypeEnum) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(strings.ToLower(actionstatus_toString[o]))
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted json string to the enum value
func (o *ActionStatusTypeEnum) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then it will be set to the zero value, 'Created' in this case.
	_, found := actionstatus_toID[strings.ToLower(j)]

	if !found {
		return helper.ErrNotFound
	}

	*o = actionstatus_toID[strings.ToLower(j)]

	return nil
}
