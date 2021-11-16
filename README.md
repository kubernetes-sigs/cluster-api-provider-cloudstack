## CloudStack Cluster API Provider (CAPC)

A Kubernetes Cluster API Provider implementation for Apache CloudStack, suitable for integration and use by AWS EKS-Anywhere.

## Security

See [CONTRIBUTING](CONTRIBUTING.md#security-issue-notifications) for more information.

## License

This project is licensed under the Apache-2.0 License.

## Testing

To run a particular test. In this case TestCreateInstance2.

- export PROJECT_DIR=`pwd`

- copy cloud-config to project dir.

- `$ go test -v -run TestCreateInstance2 ./pkg/cloud`

## Dev w/Tilt

Install the prerequisites listed here: https://cluster-api.sigs.k8s.io/developer/tilt.html

Clone the capi repository at v1.0.0.

`git clone https://github.com/kubernetes-sigs/cluster-api.git`
`cd cluster-api`
`git checkout v1.0.0`

Setup a kind cluster using
[capi repository]/hack/kind-install-for-capd.sh

There is a tiltfile in the hack directory. Edit the relative repository to match the location of the capi repository.

Copy the tiltfile into the capi repo.

Run `tilt up` from the capi repository.
