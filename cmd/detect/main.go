package main

import (
	"fmt"
	"path/filepath"

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
	extensionsExists, err := helper.FileExists(filepath.Join(context.Application.Root, ".extensions"))
	if err != nil {
		return context.Fail(), err
	}

	if extensionsExists {
		context.Logger.BodyError("Use of .extensions folder has been removed. Please remove this folder from your application.")
		return context.Fail(), nil
	}

	options, err := compat.LoadOptionsJSON(context.Application.Root)
	if err != nil {
		return context.Fail(), err
	}

	// we fake provides for httpd.Dependency because nothing provides httpd when using Nginx
	// but php-web-cnb will require HTTPD because that is the default and it doesn't know
	// at detect time that it should be using Nginx. It is OK though because it will know
	// to use Nginx at build time
	plan := buildplan.Plan{
		Provides: []buildplan.Provided{{Name: compat.Layer}},
		Requires: []buildplan.Required{{Name: compat.Layer}},
	}

	webDirExists, err := helper.FileExists(filepath.Join(context.Application.Root, options.PHP.WebDir))
	if err != nil {
		return context.Fail(), err
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
		plan.Provides = append(plan.Provides, buildplan.Provided{Name: httpd.Dependency})
		plan.Requires = append(plan.Requires, buildplan.Required{
			Name:     webServer,
			Version:  webServerVersion,
			Metadata: buildplan.Metadata{"launch": true},
		})
	}

	return context.Pass(plan)
}
