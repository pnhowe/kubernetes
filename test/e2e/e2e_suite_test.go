/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"t3kton.com/pkg/contractor"
	"t3kton.com/test/utils"
)

var (
	// Optional Environment Variables:
	// - CERT_MANAGER_INSTALL_SKIP=true: Skips CertManager installation during test setup.
	// These variables are useful if CertManager is already installed, avoiding
	// re-installation and conflicts.
	skipCertManagerInstall = os.Getenv("CERT_MANAGER_INSTALL_SKIP") == "true"
	// isCertManagerAlreadyInstalled will be set true when CertManager CRDs be found on the cluster
	isCertManagerAlreadyInstalled = false

	// projectImage is the name of the image which will be build and loaded
	// with the code source changes to be tested.
	projectImage = "t3kton-operator:e2e"

	// The tests need a running contractor with the demo data loaded, the demo stack in the docker repository
	// (docker-compose up -d) is what these tests are written for, it has the Test plugin, which only simulates builds, and
	// the foundations test01 - test04 each with a structure.
	// - E2E_CONTRACTOR_LOCAL_URL: how the tests get to contractor, default http://localhost:8080
	// - E2E_CONTRACTOR_URL: how the operator, in the cluster, gets to contractor, default is the host's address on the
	//   kind docker network, on port 8080
	// - E2E_CONTRACTOR_USERNAME, E2E_CONTRACTOR_PASSWORD: default root/root
	contractorLocalURL   = envDefault("E2E_CONTRACTOR_LOCAL_URL", "http://localhost:8080")
	contractorClusterURL string
	contractorUsername   = envDefault("E2E_CONTRACTOR_USERNAME", "root")
	contractorPassword   = envDefault("E2E_CONTRACTOR_PASSWORD", "root")

	// the structures the tests work on, test01 is used for the enforce tests, and test02 for the follow tests
	enforceFixture = &fixture{Foundation: "test01"}
	followFixture  = &fixture{Foundation: "test02"}
)

func envDefault(name string, defaultValue string) string {
	if value, ok := os.LookupEnv(name); ok && value != "" {
		return value
	}
	return defaultValue
}

// kindGatewayURL is the address the host (and so the contractor demo stack's published port) can be reached
// at from inside the kind cluster
func kindGatewayURL() (string, error) {
	cmd := exec.Command("docker", "network", "inspect", "kind", "-f", "{{range .IPAM.Config}}{{.Gateway}} {{end}}")
	output, err := utils.Run(cmd)
	if err != nil {
		return "", err
	}
	for _, gateway := range strings.Fields(output) {
		if !strings.Contains(gateway, ":") { // skip the IPv6 one
			return "http://" + gateway + ":8080", nil
		}
	}
	return "", fmt.Errorf("no IPv4 gateway on the kind docker network, got '%s'", output)
}

// TestE2E runs the end-to-end (e2e) test suite for the project. These tests execute in an isolated,
// temporary environment to validate project changes with the the purposed to be used in CI jobs.
// The default setup requires Kind, builds/loads the Manager Docker image locally, installs
// CertManager, and needs the contractor demo stack running (see above).
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting kubernetes integration test suite\n")
	RunSpecs(t, "e2e suite")
}

var _ = BeforeSuite(func() {
	ctx := context.Background()

	By("connecting to contractor")
	Expect(contractor.SetupFactory(contractorLocalURL, contractorUsername, contractorPassword, "")).To(Succeed())
	_, err := contractor.GetClient(ctx)
	Expect(err).NotTo(HaveOccurred(), "Unable to connect to contractor at %s, start the demo stack from the docker "+
		"repository (docker-compose up -d), or set E2E_CONTRACTOR_LOCAL_URL", contractorLocalURL)

	contractorClusterURL = os.Getenv("E2E_CONTRACTOR_URL")
	if contractorClusterURL == "" {
		contractorClusterURL, err = kindGatewayURL()
		Expect(err).NotTo(HaveOccurred(), "Unable to find how the cluster can get to contractor, set E2E_CONTRACTOR_URL")
	}
	_, _ = fmt.Fprintf(GinkgoWriter, "the operator will use contractor at %s\n", contractorClusterURL)

	By("finding the demo structures")
	Expect(enforceFixture.find(ctx)).To(Succeed())
	Expect(followFixture.find(ctx)).To(Succeed())

	By("building the manager(Operator) image")
	cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectImage))
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the manager(Operator) image")

	// TODO(user): If you want to change the e2e test vendor from Kind, ensure the image is
	// built and available before running the tests. Also, remove the following block.
	By("loading the manager(Operator) image on Kind")
	err = utils.LoadImageToKindClusterWithName(projectImage)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the manager(Operator) image into Kind")

	// The tests-e2e are intended to run on a temporary cluster that is created and destroyed for testing.
	// To prevent errors when tests run in environments with CertManager already installed,
	// we check for its presence before execution.
	// Setup CertManager before the suite if not skipped and if not already installed
	if !skipCertManagerInstall {
		By("checking if cert manager is installed already")
		isCertManagerAlreadyInstalled = utils.IsCertManagerCRDsInstalled()
		if !isCertManagerAlreadyInstalled {
			_, _ = fmt.Fprintf(GinkgoWriter, "Installing CertManager...\n")
			Expect(utils.InstallCertManager()).To(Succeed(), "Failed to install CertManager")
		} else {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: CertManager is already installed. Skipping installation...\n")
		}
	}
})

var _ = AfterSuite(func() {
	// Teardown CertManager after the suite if not skipped and if it was not already installed
	if !skipCertManagerInstall && !isCertManagerAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "Uninstalling CertManager...\n")
		utils.UninstallCertManager()
	}
})
