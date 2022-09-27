package helpers

import (
	"context"
	"fmt"
	toxiproxyapi "github.com/Shopify/toxiproxy/v2/client"
	. "github.com/onsi/gomega"
	"math/rand"
	"os"
	"path"
	"regexp"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/exec"
	"strconv"
	"strings"
)

func ToxiProxyServerExec(ctx context.Context) error {
	execArgs := []string{"run", "-d", "--name=capc-e2e-toxiproxy", "--net=host", "--rm", "ghcr.io/shopify/toxiproxy"}
	runCmd := exec.NewCommand(
		exec.WithCommand("docker"),
		exec.WithArgs(execArgs...),
	)
	_, stderr, err := runCmd.Run(ctx)
	if err != nil {
		fmt.Println(string(stderr))
	}
	return err
}

func ToxiProxyServerKill(ctx context.Context) error {
	execArgs := []string{"stop", "capc-e2e-toxiproxy"}
	runCmd := exec.NewCommand(
		exec.WithCommand("docker"),
		exec.WithArgs(execArgs...),
	)
	_, _, err := runCmd.Run(ctx)
	return err
}

type ToxiProxyContext struct {
	KubeconfigPath string
	ClusterProxy   framework.ClusterProxy
	ToxiProxy      *toxiproxyapi.Proxy
}

func SetupForToxiProxyTesting(bootstrapClusterProxy framework.ClusterProxy) *ToxiProxyContext {
	// Read/parse the actual kubeconfig for the cluster
	kubeConfig := NewKubeconfig()
	unproxiedKubeconfigPath := bootstrapClusterProxy.GetKubeconfigPath()
	err := kubeConfig.Load(unproxiedKubeconfigPath)
	Expect(err).To(BeNil())

	// Get the cluster's server url from the kubeconfig
	server, err := kubeConfig.GetCurrentServer()
	Expect(err).To(BeNil())

	// Decompose server url into protocol, address and port
	serverRegex := regexp.MustCompilePOSIX("(https?)://([0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+):([0-9]*)")
	urlComponents := serverRegex.FindStringSubmatch(server)
	Expect(len(urlComponents)).To(Equal(4))
	protocol := urlComponents[1]
	address := urlComponents[2]
	port, err := strconv.Atoi(urlComponents[3])
	Expect(err).To(BeNil())

	// Format into the needed addresses/URL form
	actualBootstrapClusterAddress := fmt.Sprintf("%v:%v", address, port)

	// Create the toxiProxy for this test
	toxiProxyClient := toxiproxyapi.NewClient("127.0.0.1:8474")
	randomTestId := rand.Intn(65535)
	toxiProxyName := fmt.Sprintf("deploy_app_toxi_test_%#x", randomTestId)
	proxy, err := toxiProxyClient.CreateProxy(toxiProxyName, "127.0.0.1:0", actualBootstrapClusterAddress)
	Expect(err).To(BeNil())

	// Get the actual listen address (having the toxiproxy-assigned port #).
	toxiProxyServerUrl := fmt.Sprintf("%v://%v", protocol, proxy.Listen)

	// Modify the kubeconfig to use the toxiproxy's server url
	err = kubeConfig.SetCurrentServer(toxiProxyServerUrl)
	Expect(err).To(BeNil())

	// Write the modified kubeconfig using a new name.
	extension := path.Ext(unproxiedKubeconfigPath)
	baseWithoutExtension := strings.TrimSuffix(path.Base(unproxiedKubeconfigPath), extension)
	toxiProxyKubeconfigFileName := fmt.Sprintf("toxiProxy_%v_%#x%v", baseWithoutExtension, randomTestId, extension)
	toxiProxyKubeconfigPath := path.Join("/tmp", toxiProxyKubeconfigFileName)
	err = kubeConfig.Save(toxiProxyKubeconfigPath)
	Expect(err).To(BeNil())

	// Create a new ClusterProxy using the new kubeconfig
	toxiproxyBootstrapClusterProxy := framework.NewClusterProxy(
		"toxiproxy-bootstrap",
		toxiProxyKubeconfigPath,
		bootstrapClusterProxy.GetScheme(),
		framework.WithMachineLogCollector(framework.DockerLogCollector{}),
	)

	return &ToxiProxyContext{
		KubeconfigPath: toxiProxyKubeconfigPath,
		ClusterProxy:   toxiproxyBootstrapClusterProxy,
		ToxiProxy:      proxy,
	}
}

func TearDownToxiProxy(toxiProxyContext *ToxiProxyContext) {
	// Tear down the proxy
	err := toxiProxyContext.ToxiProxy.Delete()
	Expect(err).To(BeNil())

	// Delete the kubeconfig pointing to the proxy
	err = os.Remove(toxiProxyContext.KubeconfigPath)
	Expect(err).To(BeNil())
}

func (tp *ToxiProxyContext) RemoveToxic(toxicName string) {
	err := tp.ToxiProxy.RemoveToxic(toxicName)
	Expect(err).To(BeNil())
}

func (tp *ToxiProxyContext) AddLatencyToxic(latencyMs int, jitterMs int, toxicity float32, upstream bool) string {
	stream := "downstream"
	if upstream == true {
		stream = "upstream"
	}
	toxicName := fmt.Sprintf("latency_%v", stream)

	_, err := tp.ToxiProxy.AddToxic(toxicName, "latency", stream, toxicity, toxiproxyapi.Attributes{
		"latency": latencyMs,
		"jitter":  jitterMs,
	})
	Expect(err).To(BeNil())

	return toxicName
}
