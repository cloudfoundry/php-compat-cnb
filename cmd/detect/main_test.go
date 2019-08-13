package main

import (
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	. "github.com/onsi/gomega"
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
	var factory *test.DetectFactory

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewDetectFactory(t)
	})

	when("a COMPOSER_PATH is set", func(){
		it.Before(func() {
			os.Setenv("COMPOSER_PATH", "some/composer/path")
		})

		it.After(func(){
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
			err := helper.WriteFile(filepath.Join(factory.Detect.Application.Root, ".bp-config", "options.json"), 0x644, "{}")
			Expect(err).ToNot(HaveOccurred())
		})

		it.After(func() {
			err := os.RemoveAll(filepath.Join(factory.Detect.Application.Root, ".bp-config"))
			Expect(err).ToNot(HaveOccurred())
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


	when("a COMPOSER_PATH is not set and", func(){
		when(".bp-config does not exist", func() {
			it("fails detect", func() {
				code, err := runDetect(factory.Detect)
				Expect(err).ToNot(HaveOccurred())

				Expect(code).To(Equal(detect.FailStatusCode))
			})
		})
	})


}
