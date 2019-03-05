package utils

import (
	"testing"

	"github.com/blang/semver"
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

func TestBlangSemver(t *testing.T) {

	// v, _ := semver.Parse("0.0.1-alpha.preview.222+123.github")
	// log.Println("***", v.Pre, v.Build)

	v, _ := semver.Parse("0.0.1-SNAPSHOT")
	assert.Equal(t, v.Pre[0].VersionStr, "SNAPSHOT", "")
	v.LTE(v)

}
