module github.com/aws/cluster-api-provider-cloudstack-staging/test/e2e

go 1.16

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	sigs.k8s.io/cluster-api v1.0.2
	sigs.k8s.io/cluster-api/test v1.0.2
)

replace sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v1.0.2
