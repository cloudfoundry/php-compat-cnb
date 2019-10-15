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

func TestIntegration(t *testing.T) {
	RegisterTestingT(t)

	err := PreparePhpBps()
	Expect(err).ToNot(HaveOccurred())
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
	CleanUpBps()
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	var (
		app    *dagger.App
		err    error
	)

	it.After(func() {
		if app != nil {
			app.Destroy()
		}
	})

	when("deploying the simple_app fixture", func() {
		it("serves a simple php page with custom httpd config", func() {
			app, err = PushSimpleApp("simple_app_httpd", []string{httpdURI, phpCompatURI}, false)
			Expect(err).To(HaveOccurred())

			// because it fails, the error contains the build logs, not app.BuildLogs()
			Expect(err.Error()).To(ContainSubstring("Found 1 HTTPD configuration files under `.bp-config/httpd`. Customizing HTTPD configuration in this manner is no longer supported. Please migrate your configuration, see the Migration guide for more details."))
		})

		it("serves a simple php page with custom nginx config", func() {
			app, err = PushSimpleApp("simple_app_nginx", []string{httpdURI, phpCompatURI}, false)
			Expect(err).To(HaveOccurred())

			// because it fails, the error contains the build logs, not app.BuildLogs()
			Expect(err.Error()).To(ContainSubstring("Found 1 Nginx configuration files under `.bp-config/nginx`. Customizing Nginx configuration in this manner is no longer supported. Please migrate your configuration, see the Migration guide for more details."))
		})

		it("serves a simple php page which requires files to be moved into WEBDIR", func() {
			app, err = PushSimpleApp("simple_app_moves_files", []string{httpdURI, phpCompatURI}, false)
			Expect(err).To(HaveOccurred())

			// because it fails, the error contains the build logs, not app.BuildLogs()
			Expect(err.Error()).To(ContainSubstring("WEBDIR doesn't exist, we no longer move files into WEBDIR. Please create WEBDIR and push your app again."))
		})

		it("serves a cake php app with remote dependencies", func() {
			app, err = PushSimpleApp("cake_remote_deps", []string{httpdURI, phpCompatURI, phpDistURI, composerURI, phpWebURI}, false)
			Expect(err).ToNot(HaveOccurred())

			body, _, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("Your version of PHP is 5.6.0 or higher"))
			Expect(body).To(ContainSubstring("Your version of PHP has the mbstring extension loaded"))
			Expect(body).To(ContainSubstring("Your version of PHP has the openssl extension loaded"))
			Expect(body).To(ContainSubstring("Your version of PHP has the intl extension loaded"))
			Expect(body).To(ContainSubstring("Your tmp directory is writable"))
			Expect(body).To(ContainSubstring("Your logs directory is writable"))
			Expect(body).To(ContainSubstring("CakePHP is able to connect to the database"))
		})

		it("deploying a basic PHP app with custom conf files in php.ini.d dir in app root", func() {
			app, err = PushSimpleApp("php_with_php_ini_d", []string{httpdURI, phpCompatURI, phpDistURI, phpWebURI}, false)
			Expect(err).ToNot(HaveOccurred())

			body, _, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("lkjoienfOIENFlnflkdnfiwpenLKDNFoi"))
		})

		it("deploying a basic PHP app with custom conf files in fpm.d dir in app root", func() {
			app, err = PushSimpleApp("php_with_fpm_d", []string{httpdURI, phpCompatURI, phpDistURI, phpWebURI}, false)
			Expect(err).ToNot(HaveOccurred())

			body, _, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("TEST_WEBDIR == htdocs"))
			Expect(body).To(ContainSubstring("TEST_HOME_PATH == /workspace/test/path"))
		})

		it("deploying a basic PHP app that loads all prepackaged extensions", func() {
			app, err = PushSimpleApp("php_all_modules", []string{httpdURI, phpCompatURI, phpDistURI, phpWebURI}, false)
			Expect(err).ToNot(HaveOccurred())

			body, _, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(MatchRegexp("(?i)module_(Zend[+ ])?%s", "sqlsrv"))
			Expect(body).To(MatchRegexp("(?i)module_(Zend[+ ])?%s", "pdo_sqlsrv"))
			Expect(body).To(MatchRegexp("(?i)module_(Zend[+ ])?%s", "maxminddb"))
			Expect(body).To(MatchRegexp("(?i)module_(Zend[+ ])?%s", "ioncube"))
		})

		it("deploying a basic PHP Symfony app", func() {
			app, err = PushSimpleApp("symfony_service", []string{httpdURI, phpCompatURI, phpDistURI, composerURI, phpWebURI}, false)
			Expect(err).ToNot(HaveOccurred())

			body, _, err := app.HTTPGet("/lucky/number")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("Lucky number: 42"))
		})
	})
}
