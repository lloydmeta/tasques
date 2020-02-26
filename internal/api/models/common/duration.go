package common

import (
	"encoding/json"
	"fmt"
	"time"
)

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) (err error) {
	if b[0] == '"' {
		sd := string(b[1 : len(b)-1])
		duration, parseErr := time.ParseDuration(sd)
		if parseErr == nil {
			*d = Duration(duration)
		} else {
			err = parseErr
		}
		return
	}
	var id int64
	id, err = json.Number(string(b)).Int64()
	if err != nil {
		*d = Duration(time.Duration(id))
	}
	return
}
func (d Duration) MarshalJSON() (b []byte, err error) {
	return []byte(fmt.Sprintf(`"%s"`, time.Duration(d).String())), nil
}
