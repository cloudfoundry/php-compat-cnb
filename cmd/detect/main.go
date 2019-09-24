package main

import (
	"fmt"
	"path/filepath"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/php-compat-cnb/compat"

	"os"

	"github.com/cloudfoundry/libcfbuildpack/detect"
)

func main() {
	context, err := detect.DefaultDetect()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to create a default detection context: %s", err)
		os.Exit(100)
	}

	code, err := runDetect(context)
	if err != nil {
		context.Logger.BodyError(err.Error())
	}

	os.Exit(code)
}

func runDetect(context detect.Detect) (int, error) {
	extensionsExists, err := helper.FileExists(filepath.Join(context.Application.Root, ".extensions"))
	if err != nil {
		return context.Fail(), err
	}

	if extensionsExists {
		context.Logger.BodyError("Use of .extensions folder has been removed. Please remove this folder from your application.")
		return context.Fail(), nil
	}

	_, composerPathSet := os.LookupEnv("COMPOSER_PATH")
	bpConfigExists, err := helper.FileExists(filepath.Join(context.Application.Root, ".bp-config"))
	if err != nil {
		return context.Fail(), err
	}

	if !composerPathSet && !bpConfigExists {
		return context.Fail(), nil
	}

	options, err := compat.LoadOptionsJSON(context.Application.Root)
	if err != nil {
		return context.Fail(), err
	}
	err = compat.ErrorIfShouldHaveMovedWebFilesToWebDir(options, context)
	if err != nil {
		return context.Fail(), err
	}

	return context.Pass(buildplan.Plan{
		Provides: []buildplan.Provided{{Name: compat.Layer}},
		Requires: []buildplan.Required{{Name: compat.Layer}},
	})
}

