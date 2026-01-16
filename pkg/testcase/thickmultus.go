package testcase

import (
	"github.com/rancher/distros-test-framework/shared"

	. "github.com/onsi/gomega"
)

func TestThickMultusWhereabouts(cluster *shared.Cluster) {
	manifestErr := shared.ManageManifestYaml("cp", "rke2-multus-config.yaml")
	Expect(manifestErr).NotTo(HaveOccurred(), "multus config failed to copy")

	ms := shared.NewManageService(5, 5)

	restartServer(cluster, ms)
}
