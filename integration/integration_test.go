/*
 * Copyright 2018-2019 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package integration

import (
	"testing"

	"github.com/cloudfoundry/dagger"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	app *dagger.App
	err error
)

func TestIntegration(t *testing.T) {
	RegisterTestingT(t)

	var err error
	err = PreparePhpBps()
	Expect(err).ToNot(HaveOccurred())
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
	CleanUpBps()
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion
		app    *dagger.App
		err    error
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect
	})

	it.After(func() {
		if app != nil {
			app.Destroy()
		}
	})

	when("deploying the simple_app fixture", func() {
		it("serves a simple php page with custom httpd config", func() {
			app, err = PushSimpleApp("simple_app_httpd", []string{phpCompatURI}, false)
			Expect(err).To(HaveOccurred())

			// because it fails, the error contains the build logs, not app.BuildLogs()
			Expect(err.Error()).To(ContainSubstring("Found 1 HTTPD configuration files under `.bp-config/httpd`. Customizing HTTPD configuration in this manner is no longer supported. Please migrate your configuration, see the Migration guide for more details."))
		})

		it("serves a simple php page with custom nginx config", func() {
			app, err = PushSimpleApp("simple_app_nginx", []string{phpCompatURI}, false)
			Expect(err).To(HaveOccurred())

			// because it fails, the error contains the build logs, not app.BuildLogs()
			Expect(err.Error()).To(ContainSubstring("Found 1 Nginx configuration files under `.bp-config/nginx`. Customizing Nginx configuration in this manner is no longer supported. Please migrate your configuration, see the Migration guide for more details."))
		})
	})
}
