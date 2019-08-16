package compat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const Layer = "php-compat"

type Contributor struct {
	appRoot string
	log logger.Logger
}

func NewContributor(context build.Build) (Contributor, bool, error) {
	wantDependency := context.Plans.Has(Layer)
	if !wantDependency {
		return Contributor{}, false, nil
	}

	return Contributor{
		appRoot: context.Application.Root,
	}, true, nil
}

func (c Contributor) Contribute() error {
	options, err := LoadOptionsJSON(c.appRoot)
	if err != nil {
		return err
	}

	if strings.ToLower(options.Composer.Version) == "latest" {
		options.Composer.Version = ""
		c.log.BodyWarning("Specifying a version of 'latest' is no longer supported. The default version of the php-composer-cnb will be used instead.")
	}

	// migrate COMPOSER_PATH to buildpack.yml
	options.Composer.Path = os.Getenv("COMPOSER_PATH")

	//migrate PHP/ZEND_EXTENSIONS
	err = c.MigrateExtensions(options)
	if err != nil {
		return err
	}

	err = WriteOptionsToBuildpackYAML(c.appRoot, options)
	if err != nil {
		return err
	}

	return nil
}

func (c Contributor) MigrateExtensions(options Options) error {
	buf := bytes.Buffer{}

	for _, phpExt := range options.PHP.Extensions {
		buf.WriteString(fmt.Sprintf("extension=%s.so\n", phpExt))
	}

	for _, zendExt := range options.PHP.ZendExtensions {
		buf.WriteString(fmt.Sprintf("zend_extension=%s.so\n", zendExt))
	}

	err := helper.WriteFile(filepath.Join(c.appRoot, ".php.ini.d", "compat-extensions.ini"), 0655, buf.String())

	return err
}

type Options struct {
	HTTPD HTTPDOptions `yaml:"httpd"`
	PHP PHPOptions `yaml:"php"`
	Nginx NginxOptions `yaml:"nginx"`
	Composer ComposerOptions `yaml:"composer"`
}

type PHPOptions struct {
	WebServer string `json:"WEB_SERVER" yaml:"webserver"`
	Version string `json:"PHP_VERSION" yaml:"version"`
	AdminEmail string `json:"ADMIN_EMAIL" yaml:"serveradmin"`
	AppStartCommand string `json:"APP_START_CMD" yaml:"script"`
	WebDir string `json:"WEBDIR" yaml:"webdirectory"`
	LibDir string `json:"LIBDIR" yaml:"libdirectory"`
	Extensions []string `json:"PHP_EXTENSIONS" yaml:"-"`
	ZendExtensions []string `json:"ZEND_EXTENSIONS" yaml:"-"`
}

type HTTPDOptions struct {
	Version string `json:"HTTPD_VERSION" yaml:version`
}

type NginxOptions struct{
	Version string `json:"NGINX_VERSION" yaml:"version"`
}

type ComposerOptions struct{
	Version string `json:"COMPOSER_VERSION" yaml:"version"`
	Path string `yaml:"json_path"`
}

// LoadOptionsJSON loads the options.json file from disk
func LoadOptionsJSON(appRoot string) (Options, error) {
	configFile := filepath.Join(appRoot, ".bp-config", "options.json")

	phpOptions := PHPOptions{}
	httpdOptions := HTTPDOptions{}
	nginxOptions := NginxOptions{}
	composerOptions := ComposerOptions{}

	if exists, err := helper.FileExists(configFile); err != nil {
		return Options{}, err
	} else if exists {
		file, err := os.Open(configFile)
		if err != nil {
			return Options{}, err
		}
		defer file.Close()

		contents, err := ioutil.ReadAll(file)
		if err != nil {
			return Options{}, err
		}

		// We marshal four times, one for each options type
		//   this is intentional.
		err = json.Unmarshal(contents, &phpOptions)
		if err != nil {
			return Options{}, err
		}
		setPhpDefaultVersions(&phpOptions)

		err = json.Unmarshal(contents, &httpdOptions)
		if err != nil {
			return Options{}, err
		}

		err = json.Unmarshal(contents, &nginxOptions)
		if err != nil {
			return Options{}, err
		}

		err = json.Unmarshal(contents, &composerOptions)
		if err != nil {
			return Options{}, err
		}
	}
	return Options{ PHP: phpOptions, HTTPD: httpdOptions, Nginx: nginxOptions, Composer: composerOptions}, nil
}

func setPhpDefaultVersions(phpOptions *PHPOptions) {
	if phpOptions.Version == "{PHP_DEFAULT}" {
		phpOptions.Version = ""
	}
	if phpOptions.Version == "{PHP_71_LATEST}" {
		phpOptions.Version = "7.1.*"
	}
	if phpOptions.Version == "{PHP_72_LATEST}" {
		phpOptions.Version = "7.2.*"
	}
	if phpOptions.Version == "{PHP_73_LATEST}" {
		phpOptions.Version = "7.3.*"
	}
}

func WriteOptionsToBuildpackYAML(appRoot string, options Options) error {
	configFile := filepath.Join(appRoot, "buildpack.yml")

	if exists, err := helper.FileExists(configFile); err != nil {
		return err
	} else if exists {
		return errors.New("you cannot have both `.bp-config/options.json` and `buildpack.yml`")
	}

	optionsBytes, err := yaml.Marshal(options)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(appRoot, "buildpack.yml"), optionsBytes, 0655)
	if err != nil {
		return err
	}

	return nil
}