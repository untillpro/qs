package utils

import (
	"testing"

	coreos "github.com/coreos/go-semver/semver"
	"github.com/stretchr/testify/assert"
)

func TestCoreOsSemver(t *testing.T) {
	// Trim PreRelease and bump
	{
		sv1 := *coreos.New("1.2.3-SNAPSHOT")
		assert.Equal(t, sv1.PreRelease, coreos.PreRelease("SNAPSHOT"))
		assert.Equal(t, "1.2.3-SNAPSHOT", sv1.String())

		sv1Release := sv1
		sv1Release.PreRelease = ""

		sv1Next := sv1
		sv1Next.Minor++
		assert.Equal(t, "1.3.3-SNAPSHOT", sv1Next.String())
	}

	// Metadata?
	{
		sv := *coreos.New("1.2.3-SNAPSHOT+argo")
		sv.PreRelease = ""
		assert.Equal(t, "1.2.3+argo", sv.String())
	}

}
