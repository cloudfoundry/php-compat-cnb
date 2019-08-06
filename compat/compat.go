package compat

import (
	"github.com/cloudfoundry/libcfbuildpack/build"
)

const Layer = "php-compat"

type Contributor struct {
}

func NewContributor(context build.Build) (Contributor, bool, error) {
	_, wantDependency := context.BuildPlan[Layer]
	if !wantDependency {
		return Contributor{}, false, nil
	}

	return Contributor{}, true, nil
}

func (c Contributor) Contribute() error {
	return nil
}
