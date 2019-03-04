package utils

import (
	"log"
	"testing"

	"github.com/blang/semver"
	coreos "github.com/coreos/go-semver/semver"
	"github.com/stretchr/testify/assert"
)

func TestCoreOsSemver(t *testing.T) {
	{
		sv1 := coreos.New("1.2.3-SNAPSHOT")
		assert.Equal(t, sv1.PreRelease, coreos.PreRelease("SNAPSHOT"))
		log.Println("*** PreRelease", sv1.PreRelease, " Metadata", sv1.Metadata)
		sv2 := coreos.New("1.2.4-SNAPSHOT")
		log.Println("***sv2 ", sv2)
		assert.True(t, sv1.LessThan(*sv2))
		sv2.Patch += 1
		log.Println("***sv2 ", sv2)

		sv3 := coreos.New("1.2.4")
		assert.Equal(t, sv3.PreRelease, coreos.PreRelease(""))
	}

}

func TestBlangSemver(t *testing.T) {

	// v, _ := semver.Parse("0.0.1-alpha.preview.222+123.github")
	// log.Println("***", v.Pre, v.Build)

	v, _ := semver.Parse("0.0.1-SNAPSHOT")
	assert.Equal(t, v.Pre[0].VersionStr, "SNAPSHOT", "")
	v.LTE(v)

}
