/*
Copyright 2022 The Kubernetes Authors.

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

package helpers

import (
	"context"
	"fmt"
	"net"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	toxiproxyapi "github.com/Shopify/toxiproxy/v2/client"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/test/framework/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	Secret         corev1.Secret
	ClusterProxy   framework.ClusterProxy
	ToxiProxy      *toxiproxyapi.Proxy
	ConfigPath     string
}

func SetupForToxiProxyTestingBootstrapCluster(bootstrapClusterProxy framework.ClusterProxy, clusterName string) *ToxiProxyContext {
	// Read/parse the actual kubeconfig for the cluster
	kubeConfig := NewKubeconfig()
	unproxiedKubeconfigPath := bootstrapClusterProxy.GetKubeconfigPath()
	err := kubeConfig.Load(unproxiedKubeconfigPath)
	Expect(err).To(BeNil())

	// Get the cluster's server url from the kubeconfig
	server, err := kubeConfig.GetCurrentServer()
	Expect(err).To(BeNil())

	// Decompose server url into protocol, address and port
	protocol, address, port, _ := parseUrl(server)

	// Format into the needed addresses/URL form
	actualBootstrapClusterAddress := fmt.Sprintf("%v:%v", address, port)

	// Create the toxiProxy for this test
	toxiProxyClient := toxiproxyapi.NewClient("127.0.0.1:8474")
	toxiProxyName := fmt.Sprintf("deploy_app_toxi_test_%v_bootstrap", clusterName)
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
	toxiProxyKubeconfigFileName := fmt.Sprintf("toxiProxy_%v_%v%v", baseWithoutExtension, clusterName, extension)
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

func TearDownToxiProxyBootstrap(toxiProxyContext *ToxiProxyContext) {
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

func (tp *ToxiProxyContext) Disable() {
	tp.ToxiProxy.Disable()
}

func (tp *ToxiProxyContext) Enable() {
	tp.ToxiProxy.Enable()
}

func SetupForToxiProxyTestingACS(ctx context.Context, clusterName string, clusterProxy framework.ClusterProxy, e2eConfig *clusterctl.E2EConfig, configPath string) *ToxiProxyContext {
	// Get the cloud-config secret that CAPC will use to access CloudStack
	fdEndpointSecretObjectKey := client.ObjectKey{
		Namespace: e2eConfig.GetVariable("CLOUDSTACK_FD1_SECRET_NAMESPACE"),
		Name:      e2eConfig.GetVariable("CLOUDSTACK_FD1_SECRET_NAME"),
	}
	fdEndpointSecret := corev1.Secret{}
	err := clusterProxy.GetClient().Get(ctx, fdEndpointSecretObjectKey, &fdEndpointSecret)
	Expect(err).To(BeNil())

	// Extract and parse the URL for CloudStack from the secret
	cloudstackUrl := string(fdEndpointSecret.Data["api-url"])
	protocol, address, port, path := parseUrl(cloudstackUrl)
	upstreamAddress := fmt.Sprintf("%v:%v", address, port)

	// Create the CloudStack toxiProxy for this test
	toxiProxyClient := toxiproxyapi.NewClient("127.0.0.1:8474")
	toxiProxyName := fmt.Sprintf("%v_cloudstack", clusterName)

	// Formulate the proxy listen address.
	// CAPC can't route to the actual host's localhost.  We have to use a real host IP address for the proxy listen address.
	hostIP := getOutboundIP()
	proxyAddress := fmt.Sprintf("%v:0", hostIP)
	proxy, err := toxiProxyClient.CreateProxy(toxiProxyName, proxyAddress, upstreamAddress)
	Expect(err).To(BeNil())

	// Retrieve the actual listen address (having the toxiproxy-assigned port #).
	toxiProxyUrl := fmt.Sprintf("%v://%v%v", protocol, proxy.Listen, path)

	// Create a new cloud-config secret using the proxy listen address
	toxiProxyFdEndpointSecret := corev1.Secret{}
	toxiProxyFdEndpointSecret.Type = fdEndpointSecret.Type
	toxiProxyFdEndpointSecret.Namespace = fdEndpointSecret.Namespace
	toxiProxyFdEndpointSecret.Name = fdEndpointSecret.Name + "-toxiproxy"
	toxiProxyFdEndpointSecret.Data = make(map[string][]byte)
	toxiProxyFdEndpointSecret.Data["api-key"] = fdEndpointSecret.Data["api-key"]
	toxiProxyFdEndpointSecret.Data["secret-key"] = fdEndpointSecret.Data["secret-key"]
	toxiProxyFdEndpointSecret.Data["verify-ssl"] = fdEndpointSecret.Data["verify-ssl"]
	toxiProxyFdEndpointSecret.Data["api-url"] = []byte(toxiProxyUrl)

	err = clusterProxy.GetClient().Create(ctx, &toxiProxyFdEndpointSecret)
	Expect(err).To(BeNil())

	// Override the test config to use this alternate cloud-config secret
	e2eConfig.Variables["CLOUDSTACK_FD1_SECRET_NAME"] = toxiProxyFdEndpointSecret.Name

	// Overriding e2e config file into a new temp copy, so as not to inadvertently override the other e2e tests.
	newConfigFilePath := fmt.Sprintf("/tmp/%v.yaml", toxiProxyName)
	editConfigFile(newConfigFilePath, configPath, "CLOUDSTACK_FD1_SECRET_NAME", toxiProxyFdEndpointSecret.Name)

	// Return a context
	return &ToxiProxyContext{
		Secret:     toxiProxyFdEndpointSecret,
		ToxiProxy:  proxy,
		ConfigPath: newConfigFilePath,
	}
}

func TearDownToxiProxyACS(ctx context.Context, clusterProxy framework.ClusterProxy, toxiProxyContext *ToxiProxyContext) {
	// Tear down the proxy
	err := toxiProxyContext.ToxiProxy.Delete()
	Expect(err).To(BeNil())

	// Delete the secret
	err = clusterProxy.GetClient().Delete(ctx, &toxiProxyContext.Secret)
	Expect(err).To(BeNil())

	// Delete the overridden e2e config
	err = os.Remove(toxiProxyContext.ConfigPath)
	Expect(err).To(BeNil())

}

func parseUrl(url string) (string, string, int, string) {
	serverRegex := regexp.MustCompilePOSIX("(https?)://([0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+):([0-9]+)?(.*)")

	urlComponents := serverRegex.FindStringSubmatch(url)
	Expect(len(urlComponents)).To(BeNumerically(">=", 4))
	protocol := urlComponents[1]
	address := urlComponents[2]
	port, err := strconv.Atoi(urlComponents[3])
	Expect(err).To(BeNil())
	path := urlComponents[4]
	return protocol, address, port, path
}

func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80") // 8.8.8.8:80 is arbitrary.  Any IP will do, reachable or not.
	Expect(err).To(BeNil())

	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func editConfigFile(destFilename string, sourceFilename string, key string, newValue string) {
	// For config files with key: value on each line.

	dat, err := os.ReadFile(sourceFilename)
	Expect(err).To(BeNil())

	lines := strings.Split(string(dat), "\n")

	keyFound := false
	for index, line := range lines {
		if strings.HasPrefix(line, "CLOUDSTACK_FD1_SECRET_NAME:") {
			keyFound = true
			lines[index] = fmt.Sprintf("%v: %v", key, newValue)
			break
		}
	}
	Expect(keyFound).To(BeTrue())

	dat = []byte(strings.Join(lines[:], "\n"))
	err = os.WriteFile(destFilename, dat, 0600)
	Expect(err).To(BeNil())
}
