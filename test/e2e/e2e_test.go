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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"t3kton.com/test/utils"
)

// namespace where the project is deployed in
const namespace = "t3kton-contractor"

// namePrefix is added to the name of everything by config/default/kustomization.yaml
const namePrefix = "t3kton-contractor-"

// serviceAccountName created for the project
const serviceAccountName = namePrefix + "controller-manager"

// deploymentName is the name of the operator's deployment
const deploymentName = namePrefix + "controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = namePrefix + "controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = namePrefix + "metrics-binding"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("creating the contractor-credentials Secret")
		cmd = exec.Command("kubectl", "create", "secret", "generic", "contractor-credentials", "-n", namespace,
			"--from-literal=host="+contractorClusterURL,
			"--from-literal=username="+contractorUsername,
			"--from-literal=password="+contractorPassword)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create the contractor-credentials Secret")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		// make deploy changes the image in config/manager/kustomization.yaml, put it back when we are done
		projectDir, err := utils.GetProjectDir()
		Expect(err).NotTo(HaveOccurred())
		kustomizationFile := filepath.Join(projectDir, "config", "manager", "kustomization.yaml")
		originalKustomization, err := os.ReadFile(kustomizationFile)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { _ = os.WriteFile(kustomizationFile, originalKustomization, 0o644) })

		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")

		By("making the operator resync quickly, so the tests don't need to wait long")
		setOperatorArgs("--resync-interval=10s")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace, "--ignore-not-found")
		_, _ = utils.Run(cmd)

		By("removing the metrics ClusterRoleBinding")
		cmd = exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName, "--ignore-not-found")
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy", "ignore-not-found=true")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall", "ignore-not-found=true")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace, "--ignore-not-found")
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				// Validate the pod's status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole="+namePrefix+"metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("waiting for the metrics endpoint to be ready")
			verifyMetricsEndpointReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpoints", metricsServiceName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("8443"), "Metrics endpoint is not ready")
			}
			Eventually(verifyMetricsEndpointReady).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("controller-runtime.metrics\tServing metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted).Should(Succeed())

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": ["curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics"],
							"securityContext": {
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccount": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			metricsOutput := getMetricsOutput()
			// controller_runtime_reconcile_total only shows up after the first reconcile, this is there from startup
			Expect(metricsOutput).To(ContainSubstring(
				"certwatcher_read_certificate_total",
			))
		})

		It("should provisioned cert-manager", func() {
			By("validating that cert-manager has the certificate Secret")
			verifyCertManager := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "secrets", "webhook-server-cert", "-n", namespace)
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
			}
			Eventually(verifyCertManager).Should(Succeed())
		})

		It("should have CA injection for validating webhooks", func() {
			By("checking CA injection for validating webhooks")
			verifyCAInjection := func(g Gomega) {
				cmd := exec.Command("kubectl", "get",
					"validatingwebhookconfigurations.admissionregistration.k8s.io",
					namePrefix+"validating-webhook-configuration",
					"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
				vwhOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(vwhOutput)).To(BeNumerically(">", 10))
			}
			Eventually(verifyCAInjection).Should(Succeed())
		})

		It("should have CA injection for mutating webhooks", func() {
			By("checking CA injection for mutating webhooks")
			verifyCAInjection := func(g Gomega) {
				cmd := exec.Command("kubectl", "get",
					"mutatingwebhookconfigurations.admissionregistration.k8s.io",
					namePrefix+"mutating-webhook-configuration",
					"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
				mwhOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(mwhOutput)).To(BeNumerically(">", 10))
			}
			Eventually(verifyCAInjection).Should(Succeed())
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks

	})

	Context("Structure, with the operator enforcing the spec", func() {
		const name = "e2e-enforce"

		It("should refuse a Structure that does not exist in contractor", func() {
			err := applyStructure("e2e-missing", "id: 999999")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to get structure"))
		})

		It("should adopt the Structure, filling in the state, blueprint and config values from contractor", func() {
			Expect(enforceFixture.ensureFoundationBuilt(context.Background())).To(Succeed())

			Expect(applyStructure(name, fmt.Sprintf("id: %d", enforceFixture.StructureID))).To(Succeed())

			Expect(structureField(name, "{.spec.state}")).To(Equal("planned"))
			Expect(structureField(name, "{.spec.blueprint}")).NotTo(BeEmpty())

			Eventually(func(g Gomega) {
				g.Expect(structureField(name, "{.status.state}")).To(Equal("planned"))
				g.Expect(structureField(name, ready)).To(Equal("True"))
				g.Expect(structureField(name, readyReason)).To(Equal("Reconciled"))
			}).Should(Succeed())
		})

		It("should refuse to change the ID", func() {
			_, err := kubectl("patch", "structure", name, "-n", namespace, "--type=merge",
				"-p", fmt.Sprintf(`{"spec":{"id":%d}}`, enforceFixture.StructureID+1000))
			Expect(err).To(HaveOccurred())
		})

		It("should build the Structure when the state is changed to built", func() {
			_, err := kubectl("patch", "structure", name, "-n", namespace, "--type=merge", "-p", `{"spec":{"state":"built"}}`)
			Expect(err).NotTo(HaveOccurred())

			By("seeing the job in the status")
			Eventually(func(g Gomega) {
				g.Expect(structureField(name, "{.status.job.script}")).To(Equal("create"))
			}, 2*time.Minute, 500*time.Millisecond).Should(Succeed())
			Expect(structureField(name, ready)).To(Equal("False"))

			By("waiting for it to be built")
			Eventually(func(g Gomega) {
				g.Expect(structureField(name, "{.status.state}")).To(Equal("built"))
				g.Expect(structureField(name, ready)).To(Equal("True"))
			}, 5*time.Minute).Should(Succeed())

			state, hasJob, err := enforceFixture.state(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(state).To(Equal("built"))
			Expect(hasJob).To(BeFalse())
		})

		It("should refuse to delete a built Structure", func() {
			_, err := kubectl("delete", "structure", name, "-n", namespace)
			Expect(err).To(HaveOccurred())
		})

		It("should write config values to contractor", func() {
			_, err := kubectl("patch", "structure", name, "-n", namespace, "--type=merge",
				"-p", `{"spec":{"configValues":{"e2e":"from-k8s","e2e_number":42}}}`)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func(g Gomega) {
				values, err := enforceFixture.configValues(context.Background())
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(values).To(HaveKeyWithValue("e2e", "from-k8s"))
				g.Expect(values).To(HaveKeyWithValue("e2e_number", BeNumerically("==", 42)))
			}).Should(Succeed())
		})

		It("should put back config values that are changed in contractor", func() {
			values, err := enforceFixture.configValues(context.Background())
			Expect(err).NotTo(HaveOccurred())
			values["e2e"] = "changed-in-contractor"
			Expect(enforceFixture.setConfigValues(context.Background(), values)).To(Succeed())

			// the resync is every 10 seconds
			Eventually(func(g Gomega) {
				values, err := enforceFixture.configValues(context.Background())
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(values).To(HaveKeyWithValue("e2e", "from-k8s"))
			}).Should(Succeed())
		})

		It("should destroy the Structure when the state is changed to planned", func() {
			_, err := kubectl("patch", "structure", name, "-n", namespace, "--type=merge", "-p", `{"spec":{"state":"planned"}}`)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func(g Gomega) {
				g.Expect(structureField(name, "{.status.state}")).To(Equal("planned"))
				g.Expect(structureField(name, ready)).To(Equal("True"))
			}, 5*time.Minute).Should(Succeed())
		})

		It("should leave the Structure alone in contractor when the resource is deleted", func() {
			_, err := kubectl("delete", "structure", name, "-n", namespace)
			Expect(err).NotTo(HaveOccurred())

			state, _, err := enforceFixture.state(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(state).To(Equal("planned"))

			By("putting the demo data back the way it was")
			values, err := enforceFixture.configValues(context.Background())
			Expect(err).NotTo(HaveOccurred())
			delete(values, "e2e")
			delete(values, "e2e_number")
			Expect(enforceFixture.setConfigValues(context.Background(), values)).To(Succeed())
		})
	})

	Context("Structure, with the operator following contractor", func() {
		const name = "e2e-follow"

		It("should start following", func() {
			setOperatorArgs("--resync-interval=10s", "--sync-mode=follow")
		})

		It("should update the spec when contractor is changed", func() {
			Expect(applyStructure(name, fmt.Sprintf("id: %d", followFixture.StructureID))).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(structureField(name, ready)).To(Equal("True"))
			}).Should(Succeed())

			values, err := followFixture.configValues(context.Background())
			Expect(err).NotTo(HaveOccurred())
			values["e2e_followed"] = "from-contractor"
			Expect(followFixture.setConfigValues(context.Background(), values)).To(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(structureField(name, "{.spec.configValues.e2e_followed}")).To(Equal("from-contractor"))
			}).Should(Succeed())
		})

		It("should not change contractor to match the spec", func() {
			_, err := kubectl("patch", "structure", name, "-n", namespace, "--type=merge",
				"-p", `{"spec":{"configValues":{"e2e_followed":"from-k8s"}}}`)
			Expect(err).NotTo(HaveOccurred())

			// the spec is put back to match contractor, and contractor still has its value
			Eventually(func(g Gomega) {
				g.Expect(structureField(name, "{.spec.configValues.e2e_followed}")).To(Equal("from-contractor"))
			}).Should(Succeed())
			values, err := followFixture.configValues(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(HaveKeyWithValue("e2e_followed", "from-contractor"))
		})

		It("should clean up", func() {
			_, err := kubectl("delete", "structure", name, "-n", namespace)
			Expect(err).NotTo(HaveOccurred())

			By("putting the demo data back the way it was")
			values, err := followFixture.configValues(context.Background())
			Expect(err).NotTo(HaveOccurred())
			delete(values, "e2e_followed")
			Expect(followFixture.setConfigValues(context.Background(), values)).To(Succeed())
		})
	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() string {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	metricsOutput, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
	Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
	return metricsOutput
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}

// JSONPaths of the Ready condition
const (
	ready       = `{.status.conditions[?(@.type=="Ready")].status}`
	readyReason = `{.status.conditions[?(@.type=="Ready")].reason}`
)

// kubectl runs kubectl, the output (which has the error message for the failures) is in the error
func kubectl(args ...string) (string, error) {
	return utils.Run(exec.Command("kubectl", args...))
}

// applyStructure creates or updates a Structure, spec is the YAML of the spec's fields, ie: "id: 3"
func applyStructure(name string, spec string) error {
	manifest := fmt.Sprintf(`apiVersion: contractor.t3kton.com/v1
kind: Structure
metadata:
  name: %s
  namespace: %s
spec:
  %s
`, name, namespace, spec)

	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)
	_, err := utils.Run(cmd)
	return err
}

// structureField returns the field (a JSONPath) of the Structure, or "" if it is not there (yet)
func structureField(name string, jsonPath string) string {
	output, err := kubectl("get", "structure", name, "-n", namespace, "-o", "jsonpath="+jsonPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(output)
}

// setOperatorArgs adds command line arguments to the operator, and waits for it to restart with them
func setOperatorArgs(args ...string) {
	patch := make([]map[string]string, 0, len(args))
	for _, arg := range args {
		patch = append(patch, map[string]string{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": arg})
	}
	patchJSON, err := json.Marshal(patch)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	_, err = kubectl("patch", "deployment", deploymentName, "-n", namespace, "--type=json", "-p", string(patchJSON))
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to add the operator arguments")

	_, err = kubectl("rollout", "status", "deployment/"+deploymentName, "-n", namespace, "--timeout=180s")
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "The operator did not restart")
}
