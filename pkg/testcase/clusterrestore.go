package testcase

import (
	"fmt"
	"os"
	"strings"

	"github.com/rancher/distros-test-framework/pkg/aws"
	"github.com/rancher/distros-test-framework/pkg/customflag"
	"github.com/rancher/distros-test-framework/shared"

	. "github.com/onsi/gomega"
)

// var s3Config shared.S3Config
// var awsConfig shared.AwsConfig

// func setConfigs(cluster *shared.Cluster, flags *customflag.FlagConfig) {

// 	s3Config = shared.S3Config{
// 		Region: os.Getenv("region"),
// 		Bucket: flags.S3Flags.Bucket,
// 		Folder: flags.S3Flags.Folder,
// 	}
// 	awsConfig = shared.AwsConfig{
// 		AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
// 		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
// 	}

// }

func TestClusterRestoreS3(
	cluster *shared.Cluster,
	applyWorkload bool,
	flags *customflag.FlagConfig,
) {
	product := cluster.Config.Product
	_, version, err := shared.Product()
	Expect(err).NotTo(HaveOccurred())
	versionCleanUp := strings.TrimPrefix(version, "rke2 version ")
	endChar := strings.Index(versionCleanUp, "(")
	versionClean := versionCleanUp[:endChar]

	var workloadErr error
	if applyWorkload {
		workloadErr = shared.ManageWorkload("apply", product+"-extra-metadata.yaml")
		Expect(workloadErr).NotTo(HaveOccurred(), "configmap failed to create")
	}

	shared.LogLevel("info", "%s-extra-metadata configmap successfully added", product)

	testTakeS3Snapshot(
		cluster,
		flags,
		true,
	)

	testS3SnapshotSave(
		cluster,
		flags,
	)

	onDemandPath, onDemandPathErr := shared.FetchSnapshotOnDemandPath(cluster.Config.Product, cluster.ServerIPs[0])
	Expect(onDemandPathErr).NotTo(HaveOccurred())

	clusterToken, clusterTokenErr := shared.FetchToken(cluster.Config.Product, cluster.ServerIPs[0])
	Expect(clusterTokenErr).NotTo(HaveOccurred())

	resourceName := os.Getenv("resource_name")
	ec2, err := aws.AddClient(cluster)
	Expect(err).NotTo(HaveOccurred(), "error adding aws nodes: %s", err)

	stopServerInstances(cluster, ec2)

	stopAgentInstance(cluster, ec2)

	// oldLeadServerIP := cluster.ServerIPs[0]

	// create new server.
	var serverName []string

	serverName = append(serverName, fmt.Sprintf("%s-server-fresh", resourceName))

	externalServerIP, _, _, createErr :=
		ec2.CreateInstances(serverName...)
	Expect(createErr).NotTo(HaveOccurred(), createErr)

	shared.LogLevel("info", "Created server public ip: %s",
		externalServerIP[0])

	newServerIP := externalServerIP

	shared.LogLevel("info", "overriding previous cluster data with new cluster")

	shared.LogLevel("info", "installing %s on server: %s", product, newServerIP)

	installProduct(
		cluster,
		newServerIP[0],
		versionClean,
	)

	shared.LogLevel("info", "running cluster reset on server %s\n", newServerIP)
	testRestoreS3Snapshot(
		cluster,
		onDemandPath,
		clusterToken,
		newServerIP[0],
		flags,
	)

	enableAndStartService(
		cluster,
		newServerIP[0],
	)

	newKubeConfig, newKubeConfigErr := shared.UpdateKubeConfig(newServerIP[0],
		resourceName, product)
	Expect(newKubeConfigErr).NotTo(HaveOccurred())
	shared.LogLevel("info", "kubeconfig updated to %s\n", newKubeConfig)

}

func testS3SnapshotSave(cluster *shared.Cluster, flags *customflag.FlagConfig) {

	s3, err := aws.AddClient(cluster)
	Expect(err).NotTo(HaveOccurred(), "error creating s3 client: %s", err)

	s3.GetObjects(flags)
}

func testTakeS3Snapshot(
	cluster *shared.Cluster,
	flags *customflag.FlagConfig,
	applyWorkload bool,
) {
	productLocationCmd, findErr := shared.FindPath(cluster.Config.Product, cluster.ServerIPs[0])
	Expect(findErr).NotTo(HaveOccurred())

	takeSnapshotCmd := fmt.Sprintf("sudo %s etcd-snapshot save --s3 --s3-bucket=%s "+
		"--s3-folder=%s --s3-region=%s --s3-access-key=%s --s3-secret-key=%s",
		productLocationCmd, flags.S3Flags.Bucket, flags.S3Flags.Folder, cluster.Aws.Region,
		cluster.Aws.AccessKeyID, cluster.Aws.SecretAccessKey)

	takeSnapshotRes, takeSnapshotErr := shared.RunCommandOnNode(takeSnapshotCmd, cluster.ServerIPs[0])
	Expect(takeSnapshotErr).NotTo(HaveOccurred())
	Expect(takeSnapshotRes).To(ContainSubstring("Snapshot on-demand"))

	var workloadErr error
	if applyWorkload {
		workloadErr = shared.ManageWorkload("apply", "daemonset.yaml")
		Expect(workloadErr).NotTo(HaveOccurred(), "Daemonset manifest not deployed")
	}
}

func stopServerInstances(cluster *shared.Cluster, ec2 *aws.Client) {

	for i := 0; i < len(cluster.ServerIPs); i++ {
		serverInstanceIDs, serverInstanceIDsErr := ec2.GetInstanceIDByIP(cluster.ServerIPs[i])
		Expect(serverInstanceIDsErr).NotTo(HaveOccurred())
		fmt.Println(serverInstanceIDs)
		ec2.StopInstance(serverInstanceIDs)
		Expect(serverInstanceIDsErr).NotTo(HaveOccurred())
	}

}

func stopAgentInstance(cluster *shared.Cluster, ec2 *aws.Client) {

	for i := 0; i < len(cluster.AgentIPs); i++ {
		agentInstanceIDs, agentInstanceIDsErr := ec2.GetInstanceIDByIP(cluster.AgentIPs[i])
		Expect(agentInstanceIDsErr).NotTo(HaveOccurred())
		fmt.Println(agentInstanceIDs)
		ec2.StopInstance(agentInstanceIDs)
		Expect(agentInstanceIDsErr).NotTo(HaveOccurred())
	}

}

func installProduct(
	cluster *shared.Cluster,
	newClusterIP string,
	version string,
) {

	if cluster.Config.Product == "k3s" {
		installCmd := fmt.Sprintf("curl -sfL https://get.k3s.io/ | sudo INSTALL_K3S_VERSION=%s INSTALL_K3S_SKIP_ENABLE=true sh -", version)
		_, installCmdErr := shared.RunCommandOnNode(installCmd, newClusterIP)
		Expect(installCmdErr).NotTo(HaveOccurred())
	} else if cluster.Config.Product == "rke2" {
		installCmd := fmt.Sprintf("curl -sfL https://get.rke2.io | sudo INSTALL_RKE2_VERSION=%s sh -", version)
		_, installCmdErr := shared.RunCommandOnNode(installCmd, newClusterIP)
		Expect(installCmdErr).NotTo(HaveOccurred())
	} else {
		shared.LogLevel("error", "unsupported product")
	}
}

func testRestoreS3Snapshot(
	cluster *shared.Cluster,
	onDemandPath,
	token string,
	newClusterIP string,
	flags *customflag.FlagConfig,
) {
	fmt.Println("s3Bucket set to ", flags.S3Flags.Bucket)
	fmt.Println("s3Folder set to ", flags.S3Flags.Folder)
	fmt.Println("s3Region set to ", cluster.Aws.Region)
	// var path string
	productLocationCmd, findErr := shared.FindPath(cluster.Config.Product, newClusterIP)
	Expect(findErr).NotTo(HaveOccurred())
	resetCmd := fmt.Sprintf("sudo %s server --cluster-reset --etcd-s3 --cluster-reset-restore-path=%s"+
		" --etcd-s3-bucket=%s --etcd-s3-folder=%s --etcd-s3-region=%s --etcd-s3-access-key=%s"+
		" --etcd-s3-secret-key=%s --token=%s", productLocationCmd, onDemandPath, flags.S3Flags.Bucket,
		flags.S3Flags.Folder, cluster.Aws.Region, cluster.Aws.AccessKeyID, cluster.Aws.SecretAccessKey, token)
	resetCmdRes, resetCmdErr := shared.RunCommandOnNode(resetCmd, newClusterIP)
	Expect(resetCmdErr).To(HaveOccurred())
	Expect(resetCmdErr.Error).To(ContainSubstring("Managed etcd cluster"))
	Expect(resetCmdErr.Error).To(ContainSubstring("has been reset"))
	fmt.Println("Response: ", resetCmdRes)
	fmt.Println("Error: ", resetCmdErr)
}

func enableAndStartService(
	cluster *shared.Cluster,
	externalServerIP string,
) {
	_, enableServiceCmdErr := shared.ManageService(cluster.Config.Product, "enable", "server",
		[]string{externalServerIP})
	Expect(enableServiceCmdErr).NotTo(HaveOccurred())
	_, startServiceCmdErr := shared.ManageService(cluster.Config.Product, "start", "server",
		[]string{externalServerIP})
	Expect(startServiceCmdErr).NotTo(HaveOccurred())
	statusServiceRes, statusServiceCmdErr := shared.ManageService(cluster.Config.Product,
		"status", "server",
		[]string{externalServerIP})
	Expect(statusServiceCmdErr).NotTo(HaveOccurred())
	Expect(statusServiceRes).To(ContainSubstring("active"))
}

// func testValidateNodesAfterSnapshot() {

// }

// func testValidatePodsAfterSnapshot() {

// }
