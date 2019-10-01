package integration

import (
	"path/filepath"

	"github.com/cloudfoundry/dagger"
)

var (
	phpCompatURI, phpDistURI, composerURI, httpdURI, nginxURI, phpWebURI string
)

// PreparePhpBps builds the current buildpacks
func PreparePhpBps() error {
	bpRoot, err := dagger.FindBPRoot()
	if err != nil {
		return err
	}

	phpDistURI, err = dagger.GetLatestBuildpack("php-dist-cnb")
	if err != nil {
		return err
	}

	composerURI, err = dagger.GetLatestBuildpack("php-composer-cnb")
	if err != nil {
		return err
	}

	httpdURI, err = dagger.GetLatestBuildpack("httpd-cnb")
	if err != nil {
		return err
	}

	nginxURI, err = dagger.GetLatestBuildpack("nginx-cnb")
	if err != nil {
		return err
	}

	phpWebURI, err = dagger.GetLatestBuildpack("php-web-cnb")
	if err != nil {
		return err
	}

	phpCompatURI, err = dagger.PackageBuildpack(bpRoot)
	if err != nil {
		return err
	}

	return nil
}

// CleanUpBps removes the packaged buildpacks
func CleanUpBps() {
	for _, bp := range []string{phpCompatURI, phpDistURI, httpdURI, nginxURI, phpWebURI} {
		dagger.DeleteBuildpack(bp)
	}
}

// MakeBuildEnv creates a build environment map
func MakeBuildEnv(debug bool) map[string]string {
	env := make(map[string]string)
	if debug {
		env["BP_DEBUG"] = "true"
	}

	return env
}

func PreparePhpApp(appName string, buildpacks []string, debug bool) (*dagger.App, error) {
	app, err := dagger.PackBuildWithEnv(filepath.Join("testdata", appName), MakeBuildEnv(debug), buildpacks...)
	if err != nil {
		return &dagger.App{}, err
	}

	app.SetHealthCheck("", "3s", "1s")
	app.Env["PORT"] = "8080"

	return app, nil
}

func PushSimpleApp(name string, buildpacks []string, script bool) (*dagger.App, error) {
	app, err := PreparePhpApp(name, buildpacks, false)
	if err != nil {
		return app, err
	}

	if script {
		app.SetHealthCheck("true", "3s", "1s")
	}

	err = app.Start()
	if err != nil {
		return app, err
	}

	return app, nil
}
