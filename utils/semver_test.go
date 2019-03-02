package utils

import (
	"log"
	"testing"

	"github.com/blang/semver"
)

func TestBlangSemver(t *testing.T) {

	v, _ := semver.Parse("0.0.1-alpha.preview.222+123.github")
	log.Println("***", v)

}
