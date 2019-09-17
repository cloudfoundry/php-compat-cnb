package compat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"github.com/cloudfoundry/php-composer-cnb/composer"
	"gopkg.in/yaml.v2"
)

const Layer = "php-compat"

type Contributor struct {
	appRoot string
	log     logger.Logger
}

func NewContributor(context build.Build) (Contributor, bool, error) {
	wantDependency := context.Plans.Has(Layer)
	if !wantDependency {
		return Contributor{}, false, nil
	}

	return Contributor{
		appRoot: context.Application.Root,
		log:     context.Logger,
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

	composerLocation, _ :=	composer.FindComposer(c.appRoot, "")
	if composerLocation != "" {
		c.log.BodyWarning("Attention: some lesser used Composer configuration options have been removed.")
		c.log.BodyWarning("- The vendor directory is no longer migrated to LIBDIR. You may need to adjust your code to use a relative path to Composer dependencies.")
		c.log.BodyWarning("- The composer.json and composer.lock files are no longer moved to the root of your application. This is the behavior most people expect. If you need them in a specific location, put them there prior to pushing your code.")
		c.log.BodyWarning("- COMPOSER_BIN_DIR is no longer supported. Please create a Github issue if you have a use case which requires this option.")
		c.log.BodyWarning("- COMPOSER_CACHE_DIR is no longer supported. Please create a Github issue if you have a use case which requires this option.")
	}

	err = c.ErrorOnCustomServerConfig("HTTPD", "httpd", ".conf")
	if err != nil {
		return err
	}

	err = c.ErrorOnCustomServerConfig("Nginx", "nginx", ".conf")
	if err != nil {
		return err
	}

	// migrate php.ini and php-fpm snippets
	err = c.MigratePHPSnippets("PHP INI", "php.ini.d", ".php.ini.d", "ini")
	if err != nil {
		return err
	}

	err = c.MigratePHPSnippets("PHP-FPM", "fpm.d", ".php.fpm.d", "conf")
	if err != nil {
		return err
	}

	// migrate COMPOSER_PATH to buildpack.yml
	options.Composer.Path = os.Getenv("COMPOSER_PATH")

	// migrate PHP/ZEND_EXTENSIONS
	err = c.MigrateExtensions(options)
	if err != nil {
		return err
	}

	// move files if there is no WEBDIR
	err = c.MoveWebFilesToWebDir(options)
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

	return helper.WriteFile(filepath.Join(c.appRoot, ".php.ini.d", "compat-extensions.ini"), 0644, buf.String())
}

func (c Contributor) MigrateAdditionalCommands(options Options) error {
	buf := bytes.Buffer{}

	for _, command := range options.PHP.AdditionalPreprocessCommands {
		buf.WriteString(fmt.Sprintf("%s\n", command))
	}

	return helper.WriteFile(filepath.Join(c.appRoot, ".profile.d", "additional-cmds.sh"), 0644, buf.String())
}

func (c Contributor) MigratePHPSnippets(name string, oldSnippetFolder string, newSnippetFolder string, extension string) error {
	oldIniPath := filepath.Join(c.appRoot, ".bp-config", "php", oldSnippetFolder)
	exists, err := helper.FileExists(oldIniPath)
	if err != nil {
		return err
	}

	if exists {
		iniFiles, err := helper.FindFiles(oldIniPath, regexp.MustCompile(fmt.Sprintf(`^.*\.%s$`, extension)))
		if err != nil {
			return err
		}

		if len(iniFiles) > 0 {
			c.log.BodyWarning("Found %d %s snippets under `.bp-config/php/%s/`. This location has changed. Moving files to `%s/`", len(iniFiles), name, oldSnippetFolder, newSnippetFolder)
		}

		newIniFolder := filepath.Join(c.appRoot, newSnippetFolder)
		for _, file := range iniFiles {
			filename := filepath.Base(file)
			err := helper.CopyFile(file, filepath.Join(newIniFolder, filename))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c Contributor) ErrorOnCustomServerConfig(serverName string, folderName string, extension string) error {
	serverPath := filepath.Join(c.appRoot, ".bp-config", folderName)

	files := []string{}
	err := filepath.Walk(serverPath, func(path string, f os.FileInfo, err error) error {
		if filepath.Ext(path) == extension {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return err
	}

	if len(files) > 0 {
		c.log.BodyError("Found %d %s configuration files under `.bp-config/%s`. Customizing %s configuration in this manner is no longer supported. Please migrate your configuration, see the Migration guide for more details.", len(files), serverName, folderName, serverName)
		return errors.New("migration failure")
	}

	return nil
}

func (c Contributor) IsWebApp(options Options) (bool, error) {
	// TODO: define how to determine if this is a web app
	//   - can't use the build plan as php-web-cnb's detect won't pass
	//   - can't look at files because there's no way to tell
	//   - thinking we can look at app start command, make sure it's empty and then
	//      check that none of the default script names exist, then it would be a web app
}

func (c Contributor) MoveWebFilesToWebDir(options Options) error {
	isWebApp, err := c.IsWebApp(options)
	if err != nil {
		return err
	}

	if isWebApp {
		webDirPath := filepath.Join(c.appRoot, options.PHP.WebDir)
		webDirExists, err := helper.FileExists(webDirPath)
		if err != nil {
			return err
		}

		if !webDirExists {
			c.log.Body("WEBDIR doesn't exist, moving files into WEBDIR...")
			args := []string {
				"-vam",
				"--delete",
				"--remove-source-files",
				fmt.Sprintf("--exclude=%s", ".extensions"),
				fmt.Sprintf("--exclude=%s", ".bp-config"),
				fmt.Sprintf("--exclude=%s", options.PHP.LibDir),
				fmt.Sprintf("--exclude=%s", "manifest.yml"),
				fmt.Sprintf("--exclude=%s", ".profile.d"),
				fmt.Sprintf("--exclude=%s", ".profile"),
				fmt.Sprintf("--exclude=%s", options.PHP.WebDir),
				fmt.Sprintf("%s/", c.appRoot),
				webDirPath,
			}
			cmd := exec.Command("rsync", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				return err
			}

			// rsync leaves behind empty directories, remove those
			args = []string {
				c.appRoot,
				"-type",
				"d",
				"-empty",
				"-delete",
			}
			cmd = exec.Command("find", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				return err
			}
			c.log.Body("Moving files done.")
		}

	}

	return nil
}

type Options struct {
	HTTPD    HTTPDOptions    `yaml:"httpd"`
	PHP      PHPOptions      `yaml:"php"`
	Nginx    NginxOptions    `yaml:"nginx"`
	Composer ComposerOptions `yaml:"composer"`
}

type PHPOptions struct {
	WebServer                    string   `json:"WEB_SERVER" yaml:"webserver"`
	Version                      string   `json:"PHP_VERSION" yaml:"version"`
	AdminEmail                   string   `json:"ADMIN_EMAIL" yaml:"serveradmin"`
	AppStartCommand              string   `json:"APP_START_CMD" yaml:"script"`
	WebDir                       string   `json:"WEBDIR" yaml:"webdirectory"`
	LibDir                       string   `json:"LIBDIR" yaml:"libdirectory"`
	Extensions                   []string `json:"PHP_EXTENSIONS" yaml:"-"`
	ZendExtensions               []string `json:"ZEND_EXTENSIONS" yaml:"-"`
	AdditionalPreprocessCommands []string `json:"ADDITIONAL_PREPROCESS_CMDS" yaml:"-"`
}

type HTTPDOptions struct {
	Version string `json:"HTTPD_VERSION" yaml:version`
}

type NginxOptions struct {
	Version string `json:"NGINX_VERSION" yaml:"version"`
}

type ComposerOptions struct {
	Version string `json:"COMPOSER_VERSION" yaml:"version"`
	Path    string `yaml:"json_path"`
	GlobalOptions []string `json:"COMPOSER_INSTALL_GLOBAL" yaml:"install_global"`
	InstallOptions []string `json:"COMPOSER_INSTALL_OPTIONS" yaml:"install_options"`
	VendorDirectory string `json:"COMPOSER_VENDOR_DIR" yaml:"vendor_directory"`
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
	return Options{PHP: phpOptions, HTTPD: httpdOptions, Nginx: nginxOptions, Composer: composerOptions}, nil
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
