module github.com/aws/cluster-api-provider-cloudstack-staging/test/e2e

go 1.16

require (
	github.com/apache/cloudstack-go/v2 v2.12.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/onsi/ginkgo/v2 v2.1.4
	github.com/onsi/gomega v1.19.0
	gopkg.in/ini.v1 v1.63.2
	k8s.io/api v0.23.0
	k8s.io/apimachinery v0.23.0
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	sigs.k8s.io/cluster-api v1.1.0
	sigs.k8s.io/cluster-api/test v1.1.0
	sigs.k8s.io/controller-runtime v0.11.0
)

replace sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v1.1.0
