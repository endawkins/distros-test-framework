package testcase

import (
	"fmt"
	"os"

	"github.com/rancher/distros-test-framework/pkg/aws"
	"github.com/rancher/distros-test-framework/shared"

	. "github.com/onsi/gomega"
)

func TestClusterResetRestoreS3Snapshot(
	cluster *shared.Cluster,
	applyWorkload,
	deleteWorkload bool,
) {
	var workloadErr error
	if applyWorkload {
		workloadErr = shared.ManageWorkload("apply", cluster.Config.Product+"-extra-metadata.yaml")
		Expect(workloadErr).NotTo(HaveOccurred(), "configmap failed to create")
	}

	shared.LogLevel("info", "%s-extra-metadata configmap successfully added", cluster.Config.Product)

	s3Bucket := os.Getenv("S3_BUCKET")
	s3Folder := os.Getenv("S3_FOLDER")
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	s3Region := cluster.AwsEc2.Region

	takeS3Snapshot(
		cluster,
		s3Bucket,
		s3Folder,
		s3Region,
		accessKeyID,
		secretAccessKey,
		true,
		false,
	)

	onDemandPathCmd := fmt.Sprintf("sudo ls /var/lib/rancher/%s/server/db/snapshots", cluster.Config.Product)
	onDemandPath, _ := shared.RunCommandOnNode(onDemandPathCmd, cluster.ServerIPs[0])

	fmt.Println("\non-demand-path: ", onDemandPath)

	clusterTokenCmd := fmt.Sprintf("sudo cat /var/lib/rancher/%s/server/token", cluster.Config.Product)
	clusterToken, _ := shared.RunCommandOnNode(clusterTokenCmd, cluster.ServerIPs[0])

	fmt.Println("\ntoken: ", clusterToken)

	// stopInstances()
	// create fresh new VM and install K3s/RKE2 using RunCommandOnNode
	createNewServer(cluster)

	// how do I delete the instances, bring up a new instance and install K3s/RKE2 using what we currently have?
	shared.LogLevel("info", "running cluster reset on server %s\n", cluster.ServerIPs[0])
	restoreS3Snapshot(
		cluster,
		s3Bucket,
		s3Folder,
		s3Region,
		onDemandPath,
		accessKeyID,
		secretAccessKey,
		clusterToken,
	)

}

// perform snapshot and list snapshot commands -- deploy workloads after snapshot [apply workload]
func takeS3Snapshot(
	cluster *shared.Cluster,
	s3Bucket,
	s3Folder,
	s3Region,
	accessKeyID,
	secretAccessKey string,
	applyWorkload,
	deleteWorkload bool,
) {
	productLocationCmd, findErr := shared.FindPath(cluster.Config.Product, cluster.ServerIPs[0])
	Expect(findErr).NotTo(HaveOccurred())

	takeSnapshotCmd := fmt.Sprintf("sudo %s etcd-snapshot save --s3 --s3-bucket=%s "+
		"--s3-folder=%s --s3-region=%s --s3-access-key=%s --s3-secret-key=%s",
		productLocationCmd, s3Bucket, s3Folder, s3Region, accessKeyID, secretAccessKey)

	takeSnapshotRes, takeSnapshotErr := shared.RunCommandOnNode(takeSnapshotCmd, cluster.ServerIPs[0])
	Expect(takeSnapshotRes).To(ContainSubstring("Snapshot on-demand"))
	Expect(takeSnapshotErr).NotTo(HaveOccurred())

	var workloadErr error
	if applyWorkload {
		workloadErr = shared.ManageWorkload("apply", "daemonset.yaml")
		Expect(workloadErr).NotTo(HaveOccurred(), "Daemonset manifest not deployed")
	}

	// diff command -- comparison of outputs []

}

func stopInstances(
	cluster *shared.Cluster,
	a *aws.Client,
) {
	for i := 0; i < len(cluster.ServerIPs); i++ {
		a.StopInstance(cluster.ServerIPs[i])
	}

}

func createNewServer(cluster *shared.Cluster) (externalServerIP []string) {

	resourceName := os.Getenv("resource_name")
	awsDependencies, err := aws.AddAWSClient(cluster)
	Expect(err).NotTo(HaveOccurred(), "error adding aws nodes: %s", err)

	// create server names.
	var serverName []string

	serverName = append(serverName, fmt.Sprintf("%s-server-fresh", resourceName))

	externalServerIp, _, _, createErr :=
		awsDependencies.CreateInstances(serverName...)
	Expect(createErr).NotTo(HaveOccurred(), createErr)

	return externalServerIp
}

// func installProduct(cluster *share.cluster) {
// 	version := cluster.Config.Version
// 	if cluster.Config.Product == "k3s" {
// 		installCmd := fmt.Sprintf("curl -sfL https://get.k3s.io/ | sudo INSTALL_K3S_VERSION=%s INSTALL_K3S_SKIP_ENABLE=true sh -", version)
// 	} else {
// 		installCmd := fmt.Sprintf("curl -sfL https://get.rke2.io | sudo INSTALL_RKE2_VERSION=%s sh -", version)
// 	}

// 	installRes, installCmdErr := shared.RunCommandOnNode(installCmd, cluster.ServerIPs[0])
// }

func restoreS3Snapshot(
	cluster *shared.Cluster,
	s3Bucket,
	s3Folder,
	s3Region,
	onDemandPath,
	accessKeyID,
	secretAccessKey,
	token string,
) {
	// var path string
	productLocationCmd, findErr := shared.FindPath(cluster.Config.Product, cluster.ServerIPs[0])
	Expect(findErr).NotTo(HaveOccurred())
	resetCmd := fmt.Sprintf("sudo %s server --cluster-reset --etcd-s3 --cluster-reset-restore-path=%s"+
		"--etcd-s3-bucket=%s --etcd-s3-folder=%s --etcd-s3-region=%s --etcd-s3-access-key=%s"+
		"--etcd-s3-secret-key=%s --token=%s", productLocationCmd, onDemandPath, s3Bucket, s3Folder, s3Region, accessKeyID,
		secretAccessKey, token)
	resetRes, resetCmdErr := shared.RunCommandOnNode(resetCmd, cluster.ServerIPs[0])
	Expect(resetCmdErr).NotTo(HaveOccurred())
	Expect(resetRes).To(ContainSubstring("Managed etcd cluster"))
	Expect(resetRes).To(ContainSubstring("has been reset"))
}

// func deleteOldNodes() {

// }
