package main

import (
	"github.com/buildpack/libbuildpack/buildplan"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDetect(t *testing.T) {
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var factory *test.DetectFactory

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewDetectFactory(t)
	})

	when("a COMPOSER_PATH is set", func() {
		it.Before(func() {
			os.Setenv("COMPOSER_PATH", "some/composer/path")
		})

		it.After(func() {
			os.Unsetenv("COMPOSER_PATH")
		})

		when(".extensions is present", func() {
			it.Before(func() {
				err := helper.WriteFile(filepath.Join(factory.Detect.Application.Root, ".extensions", "options.json"), 0x644, "{}")
				Expect(err).ToNot(HaveOccurred())
			})

			it.After(func() {
				err := os.RemoveAll(filepath.Join(factory.Detect.Application.Root, ".extensions"))
				Expect(err).ToNot(HaveOccurred())
			})

			it("fails detect", func() {
				code, err := runDetect(factory.Detect)
				Expect(err).ToNot(HaveOccurred())

				Expect(code).To(Equal(detect.FailStatusCode))
			})
		})

		when(".extensions is not present", func() {
			it("passes detect", func() {
				code, err := runDetect(factory.Detect)
				Expect(err).ToNot(HaveOccurred())

				Expect(code).To(Equal(detect.PassStatusCode))
			})
		})
	})

	when(".bp-config exists", func() {
		it.Before(func() {
			err := helper.WriteFile(filepath.Join(factory.Detect.Application.Root, ".bp-config", "options.json"), 0644, "{}")
			Expect(err).ToNot(HaveOccurred())
		})

		it.After(func() {
			err := os.RemoveAll(filepath.Join(factory.Detect.Application.Root, ".bp-config"))
			Expect(err).ToNot(HaveOccurred())
		})

		when(".extensions is present", func() {
			it.Before(func() {
				err := helper.WriteFile(filepath.Join(factory.Detect.Application.Root, ".extensions", "options.json"), 0644, "{}")
				Expect(err).ToNot(HaveOccurred())
			})

			it.After(func() {
				err := os.RemoveAll(filepath.Join(factory.Detect.Application.Root, ".extensions"))
				Expect(err).ToNot(HaveOccurred())
			})
			it("fails detect", func() {
				code, err := runDetect(factory.Detect)
				Expect(err).ToNot(HaveOccurred())

				Expect(code).To(Equal(detect.FailStatusCode))
			})
		})

		when(".extensions is not present", func() {
			it("passes detect", func() {
				code, err := runDetect(factory.Detect)
				Expect(err).ToNot(HaveOccurred())

				Expect(code).To(Equal(detect.PassStatusCode))
			})
		})
	})

	when("a COMPOSER_PATH is not set and", func() {
		when(".bp-config does not exist", func() {
			it("fails detect", func() {
				code, err := runDetect(factory.Detect)
				Expect(err).ToNot(HaveOccurred())

				Expect(code).To(Equal(detect.PassStatusCode))
			})
		})
	})

	when("WEBDIR is not set", func() {
		when("htdocs folder does not exist", func() {
			it("provides and requires only itself", func() {
				code, err := runDetect(factory.Detect)
				Expect(err).ToNot(HaveOccurred())
				Expect(code).To(Equal(detect.PassStatusCode))

				Expect(factory.Plans.Plan).To(Equal(
					buildplan.Plan{
						Provides: []buildplan.Provided{{Name: "php-compat"}},
						Requires: []buildplan.Required{{Name: "php-compat"}},
					},
				))
			})
		})
		when("htdocs folder exists", func() {
			it.Before(func() {
				err := helper.WriteFile(filepath.Join(factory.Detect.Application.Root, "htdocs/index.php"), 0644, "")
				Expect(err).ToNot(HaveOccurred())
			})
			it("additionally provides httpd", func() {
				code, err := runDetect(factory.Detect)
				Expect(err).ToNot(HaveOccurred())
				Expect(code).To(Equal(detect.PassStatusCode))

				Expect(factory.Plans.Plan).To(Equal(
					buildplan.Plan{
						Provides: []buildplan.Provided{{Name: "php-compat"}},
						Requires: []buildplan.Required{{Name: "php-compat"},
							{Name: "httpd", Metadata: map[string]interface{}{"launch": true}}},
					},
				))
			})
		})
	})

	when("WEBDIR is set", func() {
		
		it("additionally provides httpd", func() {
			err := helper.WriteFile(filepath.Join(factory.Detect.Application.Root, ".bp-config/options.json"), 0644, `{"WEBDIR": "public"}`)
			Expect(err).ToNot(HaveOccurred())

			err = helper.WriteFile(filepath.Join(factory.Detect.Application.Root, "public/index.php"), 0644, "")
			Expect(err).ToNot(HaveOccurred())

			code, err := runDetect(factory.Detect)
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(detect.PassStatusCode))

			Expect(factory.Plans.Plan).To(Equal(
				buildplan.Plan{
					Provides: []buildplan.Provided{{Name: "php-compat"}},
					Requires: []buildplan.Required{{Name: "php-compat"},
						{Name: "httpd", Metadata: map[string]interface{}{"launch": true}}},
				},
			))
		})
	})
}
