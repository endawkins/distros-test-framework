package clusterrestore

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/rancher/distros-test-framework/config"
	"github.com/rancher/distros-test-framework/pkg/customflag"
	"github.com/rancher/distros-test-framework/pkg/qase"
	"github.com/rancher/distros-test-framework/shared"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	qaseReport = os.Getenv("REPORT_TO_QASE")
	flags      *customflag.FlagConfig
	kubeconfig string
	cfg        *config.Product
	cluster    *shared.Cluster
)

func TestMain(m *testing.M) {
	var err error
	flags = &customflag.ServiceFlag
	flag.Var(&flags.Destroy, "destroy", "Destroy cluster after test")
	flag.StringVar(&flags.S3Flags.Bucket, "s3Bucket", "distros_qa", "s3 bucket to store snapshots")
	flag.StringVar(&flags.S3Flags.Folder, "s3Folder", "snapshots", "s3 folder to store snapshots")
	flag.Parse()

	_, err = config.AddEnv()
	if err != nil {
		shared.LogLevel("error", "error adding env vars: %w\n", err)
		os.Exit(1)
	}

	kubeconfig = os.Getenv("KUBE_CONFIG")
	if kubeconfig == "" {
		// gets a cluster from terraform.
		cluster = shared.ClusterConfig()
	} else {
		// gets a cluster from kubeconfig.
		cluster = shared.KubeConfigCluster(kubeconfig)
	}

	// k8sClient, err = k8s.AddClient()
	// if err != nil {
	// 	shared.LogLevel("error", "error adding k8s client: %w\n", err)
	// 	os.Exit(1)
	// }

	os.Exit(m.Run())
}

func TestClusterResetRestoreSuite(t *testing.T) {
	RegisterFailHandler(FailWithReport)
	RunSpecs(t, "Cluster Reset Restore Test Suite")
}

var _ = AfterSuite(func() {
	if customflag.ServiceFlag.Destroy {
		status, err := shared.DestroyCluster()
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal("cluster destroyed"))
	}
})

var _ = ReportAfterSuite("Cluster Reset Restore Test Suite", func(report Report) {
	// AddClient Qase reporting capabilities.
	if strings.ToLower(qaseReport) == "true" {
		qaseClient, err := qase.AddQase()
		Expect(err).ToNot(HaveOccurred(), "error adding qase")

		qaseClient.ReportTestResults(qaseClient.Ctx, &report, cfg.InstallVersion)
	} else {
		shared.LogLevel("info", "Qase reporting is not enabled")
	}
})

func FailWithReport(message string, callerSkip ...int) {
	Fail(message, callerSkip[0]+1)
}
