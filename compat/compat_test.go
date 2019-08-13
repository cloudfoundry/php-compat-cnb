package compat

import (
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDetect(t *testing.T) {
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var factory *test.BuildFactory
	var appRoot string

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewBuildFactory(t)
		factory.AddBuildPlan(Layer, buildplan.Dependency{})
		appRoot = factory.Build.Application.Root
	})

	when("an options.json exists", func() {
		it.Before(func() {
			json := `{
				"WEB_SERVER": "httpd",
				"HTTPD_VERSION": "2.4.39",
				"PHP_VERSION": "7.3.10",
				"NGINX_VERSION": "1.14.3",
				"COMPOSER_VERSION": "1.9.0"}`
			err := writeOptionsJSON(appRoot, json)
			Expect(err).ToNot(HaveOccurred())
		})

		it.After(func() {
			os.RemoveAll(filepath.Join(appRoot, ".bp-config"))
		})

		when("and we're loading options", func() {
			it("can load PHP options", func() {
				options, err := LoadOptionsJSON(appRoot)
				Expect(err).ToNot(HaveOccurred())
				Expect(options.PHP.WebServer).To(Equal("httpd"))
				Expect(options.PHP.Version).To(Equal("7.3.10"))
			})
			it("can load HTTPD options", func() {
				options, err := LoadOptionsJSON(appRoot)
				Expect(err).ToNot(HaveOccurred())
				Expect(options.HTTPD.Version).To(Equal("2.4.39"))
			})
			it("can load Nginx options", func() {
				options, err := LoadOptionsJSON(appRoot)
				Expect(err).ToNot(HaveOccurred())
				Expect(options.Nginx.Version).To(Equal("1.14.3"))
			})
			it("can load Composer options", func() {
				options, err := LoadOptionsJSON(appRoot)
				Expect(err).ToNot(HaveOccurred())
				Expect(options.Composer.Version).To(Equal("1.9.0"))
			})
		})
	})

	when("options.json exists and there are specific version requirements", func() {
		it("loads PHP_DEFAULT", func() {
			json := `{"PHP_VERSION": "{PHP_DEFAULT}"}`
			err := writeOptionsJSON(appRoot, json)
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(filepath.Join(appRoot, ".bp-config"))

			options, err := LoadOptionsJSON(appRoot)
			Expect(err).ToNot(HaveOccurred())
			Expect(options.PHP.Version).To(BeEmpty())
		})
		it("loads PHP_71_LATEST", func() {
			json := `{"PHP_VERSION": "{PHP_71_LATEST}"}`
			err := writeOptionsJSON(appRoot, json)
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(filepath.Join(appRoot, ".bp-config"))

			options, err := LoadOptionsJSON(appRoot)
			Expect(err).ToNot(HaveOccurred())
			Expect(options.PHP.Version).To(Equal("7.1.*"))
		})
		it("loads PHP_72_LATEST", func() {
			json := `{"PHP_VERSION": "{PHP_72_LATEST}"}`
			err := writeOptionsJSON(appRoot, json)
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(filepath.Join(appRoot, ".bp-config"))

			options, err := LoadOptionsJSON(appRoot)
			Expect(err).ToNot(HaveOccurred())
			Expect(options.PHP.Version).To(Equal("7.2.*"))
		})
		it("loads PHP_73_LATEST", func() {
			json := `{"PHP_VERSION": "{PHP_73_LATEST}"}`
			err := writeOptionsJSON(appRoot, json)
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(filepath.Join(appRoot, ".bp-config"))

			options, err := LoadOptionsJSON(appRoot)
			Expect(err).ToNot(HaveOccurred())
			Expect(options.PHP.Version).To(Equal("7.3.*"))
		})

	})

	when("options need to be written", func() {
		it("produces buildpack.yml", func() {
			options := Options{
				HTTPD:    HTTPDOptions{
					Version: "2.3.49",
				},
				PHP:      PHPOptions{
					Version: "7.3.10",
					WebServer: "standalone",
				},
				Nginx:    NginxOptions{
					Version: "1.14.9",
				},
				Composer: ComposerOptions{
					Version: "1.8.9",
				},
			}
			err := WriteOptionsToBuildpackYAML(appRoot, options)
			Expect(err).ToNot(HaveOccurred())

			exists, err := helper.FileExists(filepath.Join(appRoot, "buildpack.yml"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			buildpackYMLOutput, err :=ioutil.ReadFile(filepath.Join(appRoot, "buildpack.yml"))
			Expect(err).ToNot(HaveOccurred())

			actualOptions := Options{}
			err = yaml.Unmarshal(buildpackYMLOutput, &actualOptions)
			Expect(err).ToNot(HaveOccurred())

			Expect(options).To(Equal(actualOptions))
		})
	})

	when("extensions need to be migrated", func() {
		it("migrates PHP_EXTENSIONS", func() {
			c, _, err := NewContributor(factory.Build)
			Expect(err).ToNot(HaveOccurred())
			options := Options{
				PHP:      PHPOptions{
					Extensions: []string{"ext1", "ext2"},
				},
			}

			err = c.MigrateExtensions(options)
			Expect(err).ToNot(HaveOccurred())

			extensionOutput, err := ioutil.ReadFile(filepath.Join(appRoot, ".php.ini.d", "compat-extensions.ini"))
			Expect(err).ToNot(HaveOccurred())

			Expect(string(extensionOutput)).To(ContainSubstring("extension=ext1.so"))
			Expect(string(extensionOutput)).To(ContainSubstring("extension=ext2.so"))
		})

		it("migrates ZEND_EXTENSIONS", func() {
			c, _, err := NewContributor(factory.Build)
			Expect(err).ToNot(HaveOccurred())
			options := Options{
				PHP:      PHPOptions{
					ZendExtensions: []string{"zext1", "zext2"},
				},
			}

			err = c.MigrateExtensions(options)
			Expect(err).ToNot(HaveOccurred())

			extensionOutput, err :=ioutil.ReadFile(filepath.Join(appRoot, ".php.ini.d", "compat-extensions.ini"))
			Expect(err).ToNot(HaveOccurred())

			Expect(string(extensionOutput)).To(ContainSubstring("zend_extension=zext1.so"))
			Expect(string(extensionOutput)).To(ContainSubstring("zend_extension=zext2.so"))
		})
	})
}

func writeOptionsJSON(appRoot, jsonBody string ) error{
	optionsJson := filepath.Join(appRoot, ".bp-config", "options.json")
	err := helper.WriteFile(optionsJson, 0655, "%s", jsonBody)
	if err != nil{
		return err
	}
	return nil
}
