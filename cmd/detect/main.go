package main

import (
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	"github.com/cloudfoundry/php-dist-cnb/php"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/httpd-cnb/httpd"
	"github.com/cloudfoundry/nginx-cnb/nginx"

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
		webServer := httpd.Dependency
		if options.PHP.WebServer != "" {
			webServer = options.PHP.WebServer
		}

		webServerVersion := "*"
		if webServer == httpd.Dependency {
			webServerVersion = options.HTTPD.Version
		} else if webServer == nginx.Dependency {
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
			Name:    php.Dependency,
			Version: options.PHP.Version,
			Metadata: buildplan.Metadata{
				"launch":                    true,
				buildpackplan.VersionSource: php.BuildpackYAMLSource,
			},
		})
	}

	return context.Pass(plan)
}
