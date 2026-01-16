//go:build multus

package versionbump

import (
	"fmt"

	"github.com/rancher/distros-test-framework/pkg/assert"
	. "github.com/rancher/distros-test-framework/pkg/customflag"
	. "github.com/rancher/distros-test-framework/pkg/template"
	"github.com/rancher/distros-test-framework/pkg/testcase"

	. "github.com/onsi/ginkgo/v2"
)

const (
	thickMultusCmd = "sudo /var/lib/rancher/rke2/bin/crictl -r unix:///run/k3s/containerd/containerd.sock images | grep hardened | grep 'thick', "
	whereaboutsCmd = "sudo /var/lib/rancher/rke2/data/*/bin/crictl -r /var/run/k3s/containerd/containerd.sock images | grep 'whereabouts' "
)

var _ = Describe("Multus + canal Version bump:", func() {
	It("Start Up with no issues", func() {
		testcase.TestBuildCluster(cluster)
	})

	It("Validate Node", func() {
		testcase.TestNodeStatus(
			cluster,
			assert.NodeAssertReadyStatus(),
			nil)
	})

	It("Validate Pod", func() {
		testcase.TestPodStatus(
			cluster,
			assert.PodAssertRestart(),
			assert.PodAssertReady())
	})

	It("Validate Thick Multus", func() {
		testcase.TestThickMultusWhereabouts(cluster)
	})

	// It("Test Bump version", func() {
	// 	Template(TestTemplate{
	// 		TestCombination: &RunCmd{
	// 			Run: []TestMapConfig{
	// 				{
	// 					Cmd: "kubectl get node -o yaml : | grep multus-cni -A1, " +
	// 						"kubectl -n kube-system get pods -l k8s-app=canal -o jsonpath=\"{..image}\" : " +
	// 						"| awk '{for(i=1;i<=NF;i++) if($i ~ /calico/) print $i}', " +
	// 						" kubectl -n kube-system get pods -l k8s-app=canal -o jsonpath=\"{..image}\" : " +
	// 						"| awk '{for(i=1;i<=NF;i++) if($i ~ /flannel/) print $i}' , " +
	// 						"kubectl get pods -n kube-system : | grep multus | awk '{print $1} {print $3}' ",
	// 					ExpectedValue:        TestMap.ExpectedValue,
	// 					ExpectedValueUpgrade: TestMap.ExpectedValueUpgrade,
	// 				},
	// 			},
	// 		},
	// 		InstallMode: ServiceFlag.InstallMode.String(),
	// 	})
	// })

	It("Test thick multus and whereabouts bump", func() {
		Template(TestTemplate{
			TestCombination: &RunCmd{
				Run: []TestMapConfig{
					{
						Cmd:                  thickMultusCmd + whereaboutsCmd,
						ExpectedValue:        TestMap.ExpectedChartsValue,
						ExpectedValueUpgrade: TestMap.ExpectedChartsValueUpgrade,
					},
				},
			},
			InstallMode: ServiceFlag.InstallMode.String(),
		})
	})

	It("Verifies dns access", func() {
		testcase.TestDNSAccess(true, true)
	})

	It("Verifies ClusterIP Service", func() {
		testcase.TestServiceClusterIP(true, true)
	})

	It("Verifies NodePort Service", func() {
		testcase.TestServiceNodePort(true, true)
	})

	It("Verifies Ingress", func() {
		testcase.TestIngress(true, true)
	})
})

var _ = AfterEach(func() {
	if CurrentSpecReport().Failed() {
		fmt.Printf("\nFAILED! %s\n\n", CurrentSpecReport().FullText())
	} else {
		fmt.Printf("\nPASSED! %s\n\n", CurrentSpecReport().FullText())
	}
})
