package helpers

import (
	"context"
	"fmt"
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

func ToxiProxyCli(ctx context.Context, command string, args ...string) (string, error) {
	// Toxiproxy offers a nice API, but it hasn't published new go packages for it since 2019.
	// The most recently published package can't be installed because of dependent package
	// case incompatibility (logrus).  Sigh.
	// So, we're gonna just go with the cli in the docker image.

	execArgs := append([]string{"run", "--rm", "--net=host", "--entrypoint=/toxiproxy-cli", "-t", "ghcr.io/shopify/toxiproxy", command}, args...)
	runCmd := exec.NewCommand(
		exec.WithCommand("docker"),
		exec.WithArgs(execArgs...),
	)
	stdout, stderr, err := runCmd.Run(ctx)
	if err != nil {
		fmt.Printf("stdout:\n%v\n\nstderr:\n%v", string(stdout), string(stderr))
		return "", err
	}
	return string(stdout), nil
}

type ToxiProxy struct {
	context        context.Context
	KubeconfigPath string
	ClusterProxy   framework.ClusterProxy
	ProxyName      string
}

func SetupForToxiproxyTesting(ctx context.Context, bootstrapClusterProxy framework.ClusterProxy) *ToxiProxy {
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

	// Declare a port for the proxy to listen on
	// ToDo: make this only return an unused port
	var toxiProxyPort int
	if port < (65535 - 1000) {
		toxiProxyPort = port + rand.Intn(1000)
	} else {
		toxiProxyPort = port - rand.Intn(1000)
	}

	// Format into the needed addresses/URL form
	actualBootstrapClusterAddress := fmt.Sprintf("%v:%v", address, port)
	toxiproxyBootstrapClusterAddress := fmt.Sprintf("127.0.0.1:%v", toxiProxyPort)
	toxiProxyServerUrl := fmt.Sprintf("%v://%v", protocol, toxiproxyBootstrapClusterAddress)

	// Create the toxiProxy for this test
	randomTestId := rand.Intn(65535)
	toxiProxyName := fmt.Sprintf("deploy_app_toxi_test_%#x", randomTestId)
	output, err := ToxiProxyCli(ctx, "create",
		"--listen", toxiproxyBootstrapClusterAddress,
		"--upstream", actualBootstrapClusterAddress,
		toxiProxyName,
	)
	if err != nil {
		fmt.Println(output)
	}
	Expect(err).To(BeNil())

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
	toxiproxyBootstrapClusterProxy := framework.NewClusterProxy("toxiproxy-bootstrap", toxiProxyKubeconfigPath, bootstrapClusterProxy.GetScheme(), framework.WithMachineLogCollector(framework.DockerLogCollector{}))

	return &ToxiProxy{
		context:        ctx,
		KubeconfigPath: toxiProxyKubeconfigPath,
		ClusterProxy:   toxiproxyBootstrapClusterProxy,
		ProxyName:      toxiProxyName,
	}
}

func TearDownToxiProxy(ctx context.Context, proxy *ToxiProxy) {
	// Tear down the proxy
	output, err := ToxiProxyCli(ctx, "delete", proxy.ProxyName)
	if err != nil {
		fmt.Println(output)
	}
	Expect(err).To(BeNil())

	// Delete the kubeconfig pointing to the proxy
	err = os.Remove(proxy.KubeconfigPath)
	Expect(err).To(BeNil())

}

func (tp *ToxiProxy) addToxic(toxicType string, toxicity int, upstream bool, args ...string) string {
	var directionFlag = "--downstream"
	toxicName := fmt.Sprintf("%v_downstream", toxicType)
	if upstream {
		directionFlag = "--upstream"
		toxicName = fmt.Sprintf("%v_upstream", toxicType)
	}

	cliArgs := []string{
		"--type", toxicType,
		"--toxicName", toxicName,
		"--toxicity", fmt.Sprintf("%d", toxicity),
		directionFlag,
	}
	cliArgs = append(cliArgs, args...)
	cliArgs = append(cliArgs, tp.ProxyName)

	output, err := ToxiProxyCli(tp.context, "toxic add", cliArgs...)
	if err != nil {
		fmt.Println(output)
	}
	Expect(err).To(BeNil())
	return toxicName
}

func (tp *ToxiProxy) RemoveToxic(toxicName string) {
	output, err := ToxiProxyCli(tp.context, "toxic remove", "--toxicName", toxicName, tp.ProxyName)
	if err != nil {
		fmt.Println(output)
	}
	Expect(err).To(BeNil())
}

func (tp *ToxiProxy) AddLatencyToxic(latencyMs int, jitterMs int, toxicity int, upstream bool) string {
	return tp.addToxic("latency", toxicity, upstream,
		"-a", fmt.Sprintf("latency=%d", latencyMs),
		"-a", fmt.Sprintf("jitter=%d", jitterMs),
	)
}
