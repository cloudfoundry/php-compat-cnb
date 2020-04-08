package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/php-compat-cnb/compat"
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
	optionsExists, err := helper.FileExists(filepath.Join(context.Application.Root, ".bp-config", "options.json"))
	if err != nil {
		return context.Fail(), err
	}

	bpYAMLExists, err := helper.FileExists(filepath.Join(context.Application.Root, "buildpack.yml"))
	if err != nil {
		return context.Fail(), err
	}

	if bpYAMLExists && !optionsExists {
		return context.Fail(), nil
	}

	options, err := compat.LoadOptionsJSON(context.Application.Root)
	if err != nil {
		return context.Fail(), err
	}

	plan := buildplan.Plan{
		Provides: []buildplan.Provided{{Name: compat.Layer}},
		Requires: []buildplan.Required{{Name: compat.Layer}},
	}

	webDirExists := false
	if options.PHP.WebDir != "" {
		webDirExists, err = helper.FileExists(filepath.Join(context.Application.Root, options.PHP.WebDir))
		if err != nil {
			return context.Fail(), err
		}
	} else {
		webDirExists, err = helper.FileExists(filepath.Join(context.Application.Root, "htdocs"))
		if err != nil {
			return context.Fail(), err
		}
	}

	if webDirExists {
		webServer := "httpd"
		if options.PHP.WebServer != "" {
			webServer = options.PHP.WebServer
		}

		webServerVersion := "*"
		if webServer == "httpd" {
			webServerVersion = options.HTTPD.Version
		} else if webServer == "nginx" {
			webServerVersion = options.Nginx.Version
		}

		if webServer != "php-server" {
			plan.Requires = append(plan.Requires, buildplan.Required{
				Name:     webServer,
				Version:  webServerVersion,
				Metadata: buildplan.Metadata{"launch": true},
			})
		}
	}

	if options.PHP.Version != "" {
		plan.Requires = append(plan.Requires, buildplan.Required{
			Name:    "php",
			Version: options.PHP.Version,
			Metadata: buildplan.Metadata{
				"launch":                    true,
				buildpackplan.VersionSource: "buildpack.yml",
			},
		})
	}

	return context.Pass(plan)
}
