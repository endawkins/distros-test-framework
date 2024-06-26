package template

import (
	"fmt"
	"strings"

	"github.com/rancher/distros-test-framework/pkg/assert"
	"github.com/rancher/distros-test-framework/pkg/testcase"
	"github.com/rancher/distros-test-framework/shared"
)

// upgradeVersion upgrades the product version
func upgradeVersion(template TestTemplate, version string) error {
	err := testcase.TestUpgradeClusterManually(version)
	if err != nil {
		return err
	}

	updateExpectedValue(template)

	return nil
}

// updateExpectedValue updates the expected values getting the values from flag ExpectedValueUpgrade
func updateExpectedValue(template TestTemplate) {
	for i := range template.TestCombination.Run {
		template.TestCombination.Run[i].ExpectedValue = template.TestCombination.Run[i].ExpectedValueUpgrade
	}
}

// executeTestCombination get a template and pass it to `processTestCombination`
//
// to execute test combination on group of IPs
func executeTestCombination(template TestTemplate) error {
	currentVersion, err := currentProductVersion()
	if err != nil {
		return shared.ReturnLogError("failed to get current version: %w", err)
	}

	ips := shared.FetchNodeExternalIPs()
	processErr := processTestCombination(ips, currentVersion, &template)
	if processErr != nil {
		return shared.ReturnLogError("failed to process test combination: %w", processErr)
	}

	if template.TestConfig != nil {
		testCaseWrapper(template)
	}

	return nil
}

// AddTestCases returns the test case based on the name to be used as customflag.
func AddTestCases(names []string) ([]testCase, error) {
	var testCases []testCase

	tcs := map[string]testCase{
		"TestDaemonset":                    testcase.TestDaemonset,
		"TestIngress":                      testcase.TestIngress,
		"TestDnsAccess":                    testcase.TestDnsAccess,
		"TestServiceClusterIP":             testcase.TestServiceClusterIp,
		"TestServiceNodePort":              testcase.TestServiceNodePort,
		"TestLocalPathProvisionerStorage":  testcase.TestLocalPathProvisionerStorage,
		"TestServiceLoadBalancer":          testcase.TestServiceLoadBalancer,
		"TestInternodeConnectivityMixedOS": testcase.TestInternodeConnectivityMixedOS,
		"TestSonobuoyMixedOS": func(applyWorkload, deleteWorkload bool) {
			testcase.TestSonobuoyMixedOS(deleteWorkload)
		},
		"TestSelinuxEnabled": func(applyWorkload, deleteWorkload bool) {
			testcase.TestSelinux()
		},
		"TestSelinux": func(applyWorkload, deleteWorkload bool) {
			testcase.TestSelinux()
		},
		"TestSelinuxSpcT": func(applyWorkload, deleteWorkload bool) {
			testcase.TestSelinuxSpcT()
		},
		"TestUninstallPolicy": func(applyWorkload, deleteWorkload bool) {
			testcase.TestUninstallPolicy()
		},
		"TestSelinuxContext": func(applyWorkload, deleteWorkload bool) {
			testcase.TestSelinuxContext()
		},
		"TestIngressRoute": func(applyWorkload, deleteWorkload bool) {
			testcase.TestIngressRoute(applyWorkload, deleteWorkload, "traefik.io/v1alpha1")
		},
		"TestCertRotate": func(applyWorkload, deleteWorkload bool) {
			testcase.TestCertRotate()
		},
	}

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			testCases = append(testCases, func(applyWorkload, deleteWorkload bool) {})
		} else if test, ok := tcs[name]; ok {
			testCases = append(testCases, test)
		} else {
			return nil, shared.ReturnLogError("invalid test case name")
		}
	}

	return testCases, nil
}

func currentProductVersion() (string, error) {
	product, err := shared.Product()
	if err != nil {
		return "", shared.ReturnLogError("failed to get product: %w", err)
	}

	version, err := shared.ProductVersion(product)
	if err != nil {

		return "", shared.ReturnLogError("failed to get product version: %w", err)
	}

	return version, nil
}

func ComponentsBumpResults() {
	product, err := shared.Product()
	if err != nil {
		return
	}

	v, err := shared.ProductVersion(product)
	if err != nil {
		return
	}

	var components []string
	for _, result := range assert.Results {
		if product == "rke2" {
			components = []string{"flannel", "calico", "ingressController", "coredns", "metricsServer", "etcd",
				"containerd", "runc"}
		} else {
			components = []string{"flannel", "coredns", "metricsServer", "etcd", "cniPlugins", "traefik", "local-path",
				"containerd", "klipper", "runc"}
		}
		for _, component := range components {
			if strings.Contains(result.Command, component) {
				fmt.Printf("\n---------------------\nResults from %s on version: %s\n``` \n%v\n ```\n---------------------"+
					"\n\n\n", component, v, result)
			}
		}
		fmt.Printf("\n---------------------\nResults from %s\n``` \n%v\n ```\n---------------------\n\n\n",
			result.Command, result)
	}
}
