package util

import (
	"encoding/json"
	"fmt"
)

func AsJsonString(o interface{}) string {
	b, err := json.Marshal(o)
	if err != nil {
		return fmt.Sprintf("error marshaling %T: %v", o, err)
	}
	return string(b)
}
