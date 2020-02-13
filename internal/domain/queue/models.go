package queue

import (
	"fmt"
	"strings"
)

// Name of a queue
type Name string

var invalidChars = `\/*?"<>| ,#:`

var illegalPrefixes = []string{
	"_",
	"-",
	"+",
}

var illegals = []string{
	".",
	"..",
}

// NameFromString takes a string and returns a Queue Name if valid, otherwise returns an InvalidName error.
//
// Mostly taken from https://github.com/elastic/elasticsearch/blob/a8b39ed842c7770bd9275958c9f747502fd9a3ea/server/src/test/java/org/elasticsearch/cluster/metadata/MetaDataCreateIndexServiceTests.java#L465-L497
func NameFromString(s string) (*Name, error) {
	var errs []error

	if len(s) == 0 {
		errs = append(errs, fmt.Errorf("empty string"))
	}
	if strings.ContainsAny(s, invalidChars) {
		errs = append(errs, fmt.Errorf("contains invalid chars [%v]", invalidChars))
	}
	for _, illegalPrefix := range illegalPrefixes {
		if strings.HasPrefix(s, illegalPrefix) {
			errs = append(errs, fmt.Errorf("starts with illegal char [%v]", illegalPrefix))
		}
	}
	for _, illegalStr := range illegals {
		if s == illegalStr {
			errs = append(errs, fmt.Errorf("equal to illegal string sequence [%v]", illegalStr))
		}
	}
	if s != strings.ToLower(s) {
		errs = append(errs, fmt.Errorf("not lower case [%v]", s))
	}
	if len(errs) == 0 {
		q := Name(s)
		return &q, nil
	} else {
		return nil, &InvalidName{
			Errors: errs,
		}
	}

}

type InvalidName struct {
	Errors []error
}

func (i *InvalidName) Error() string {
	return fmt.Sprintf("Illegal Queue name: [%v]", i.Errors)
}
