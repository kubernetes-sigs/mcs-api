# Multi-cluster Service APIs

This repository hosts the Multi-Cluster Service APIs. Providers can import packages in this repo to ensure their multi-cluster service controller implementations will be compatible with MCS data planes.

This repo contains the initial implementation according to [KEP-1645][kep] and will
be used for iterative development as we work to meet our Alpha -> Beta
[graduation requirements][grad-reqs].

[kep]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api
[grad-reqs]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#alpha---beta-graduation

## Try it out

Prerequisites:
- [kind](http://kind.sigs.k8s.io)
- This git repository should be checked out within a `GOPATH`, i.e. in `$GOPATH/src/sigs.k8s.io/mcs-api`

To see the API in action, run `make demo` to build and run a local demo against
a pair of kind clusters. Alternatively, you can take a self guided tour. Use:

- `./scripts/up.sh` to create a pair of clusters with mutually connected networks
  and install the `mcs-api-controller`.

  _This will use a pre-existing controller image if available, it's recommended
  to run `make docker-build` first._
- `./demo/demo.sh` to run the same demo as above against your newly created
  clusters (must run `./scripts/up.sh` first).
- `./scripts/down.sh` to tear down your clusters.

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](https://kubernetes.slack.com/messages/sig-multicluster)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-multicluster)

[Our meeting schedule is here]( https://github.com/kubernetes/community/tree/master/sig-multicluster#meetings)


## Technical Leads

- @pmorie
- @jeremyot

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
