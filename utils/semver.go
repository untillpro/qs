package utils

import (
	"regexp"
	"strconv"
)

type Semver struct {
	Major    int
	Minor    int
	Patch    int
	SNAPSHOT bool
}

func FromString(ver string) *Semver {
	re := regexp.MustCompile(`(\d).*\.(\d).(\d)(-SNAPSHOT)?`)
	parts := re.FindStringSubmatch(ver)
	if len(parts) < 3 {
		return nil
	}

	var res Semver
	var err error

	res.Major, err = strconv.Atoi(parts[0])
	if err != nil {
		return nil
	}

	res.Minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return nil
	}

	res.Patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return nil
	}

	res.SNAPSHOT = len(parts) > 3

	return &res

}
