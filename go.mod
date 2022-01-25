module github.com/aws/cluster-api-provider-cloudstack

go 1.16

require (
	github.com/apache/cloudstack-go/v2 v2.11.1-0.20211020121644-369057554f66
	github.com/go-logr/logr v0.4.0
	github.com/golang/mock v1.6.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.17.0
	github.com/pkg/errors v0.9.1
	gopkg.in/ini.v1 v1.63.2
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	k8s.io/utils v0.0.0-20210802155522-efc7438f0176
	sigs.k8s.io/cluster-api v0.3.23
	sigs.k8s.io/controller-runtime v0.10.0
)

replace github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt/v4 v4.0.0 // Indirect upgrade to address https://github.com/advisories/GHSA-w73w-5m7g-f7qc
