# rcs-ocm-deployer
The `rcs-ocm-deployer` is an operator designed to deploy `knative service` workloads created on the hub to the managed cluster with the lowest load average on specific namespace.

## Open Cluster Management and Knative
This project uses the `Placement` and `ManifestWork` APIs of the Open Cluster Management (OCM) project. For more information please refer to the [OCM documentation](https://open-cluster-management.io/concepts/).

In order, please refer to the [Knative Serving documentation](https://knative.dev/docs/serving/) for more information about the `knative service` object.

## Description
### The Annotations
Two annotations are required in the `knative service` yaml:  

- `dana.io/ocm-placement: <PLACEMENT>` - The name of the placement to decide from which manage cluster to deploy on.
- `dana.io/ocm-managed-cluster-namespace: <MANAGED_CLUSTER_NS>` - The namespace on the managed cluster where `knative service` needs to be deployed.

### The controllers

1. `service placement controller`: The controller extracts the placement from the `knative serivce` annotation and adds an annotation containing the cluster where to deploy, in accordance to the `placementDesicion`.

2. `service namespace controller`: The controller extracts the namespace name from the `knative serivce` annotaion and creates a `ManifestWork` in the desired `ManagedCluster` namespace, which contains the namespace with the desired name to be deployed, and adds an annotation to the service when the namespace is created.

3. `service controller`: The controller extracts the namespace name and the desired cluster from the `knative serivce` annotations and creates a `ManifestWork` in the desired `ManagedCluster` namespace; the agent on the `ManagedCluster` takes care of deploying the `knative serivce` on the cluster itself.

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KinD](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Setting Up a Hub Cluster and Managed Clusters locally
The [OCM project](https://github.com/open-cluster-management-io/OCM) contains a script that allows spinning up a `KinD`-based environment, containing of a Hub Cluster and 2 Managed Clusters.

You can clone the repository and set up the environment, provided that you have `KinD` and `docker` installed on your machine. 

This script would create 3 clusters with the following Kubernetes contextes: 

- `kind-hub`
- `kind-cluster1`
- `kind-cluster2`

To switch between contexes, use `kubectl config set-context <CONTEXT_NAME>`

```
$ git clone https://github.com/open-cluster-management-io/OCM
$ bash ./OCM/solutions/setup-dev-environment/local-up.sh
```

### Installing Knative
Follow the instructions of the Knative website to [install the Knative CLI](https://knative.dev/docs/getting-started/quickstart-install/#install-the-knative-cli), `kn`. Install Knative on the Managed Clusters:

```
$ kubectl config use-context kind-cluster1
$ kn quickstart kind --name cluster1

Running Knative Quickstart using Kind
✅ Checking dependencies...
    Kind version is: 0.17.0

Knative Cluster kind-cluster1 already installed.
Delete and recreate [y/N]: N

    Installation skipped
🍿 Installing Knative Serving v1.8.0 ...
    CRDs installed...
    Core installed...
    Finished installing Knative Serving
🕸️ Installing Kourier networking layer v1.8.0 ...
    Kourier installed...
    Ingress patched...
    Finished installing Kourier Networking layer
🕸 Configuring Kourier for Kind...
    Kourier service installed...
    Domain DNS set up...
    Finished configuring Kourier
🔥 Installing Knative Eventing v1.8.0 ...
    CRDs installed...
    Core installed...
    In-memory channel installed...
    Mt-channel broker installed...
    Example broker installed...
    Finished installing Knative Eventing
```
```
$ kubectl config use-context kind-cluster2
$ kn quickstart kind --name cluster2

Running Knative Quickstart using Kind
✅ Checking dependencies...
    Kind version is: 0.17.0

Knative Cluster kind-cluster2 already installed.
Delete and recreate [y/N]: N

    Installation skipped
🍿 Installing Knative Serving v1.8.0 ...
    CRDs installed...
    Core installed...
    Finished installing Knative Serving
🕸️ Installing Kourier networking layer v1.8.0 ...
    Kourier installed...
    Ingress patched...
    Finished installing Kourier Networking layer
🕸 Configuring Kourier for Kind...
    Kourier service installed...
    Domain DNS set up...
    Finished configuring Kourier
🔥 Installing Knative Eventing v1.8.0 ...
    CRDs installed...
    Core installed...
    In-memory channel installed...
    Mt-channel broker installed...
    Example broker installed...
    Finished installing Knative Eventing
```
### Deploying the `rcs-ocm-deployer`
On the hub cluster, install just the Knative Service CRD:

```
$ kubectl apply -f hack/crds/knative-serving-crds.yaml
```

Use the `Makefile` in order to deploy controllers onto the Hub cluster. You can either push the image to a remote registry, using `docker-push`, or work locally without pusing the image.

When working locally:

```
$ kubectl config use-context kind-hub
```
```
$ make docker-build IMG=rcs-ocm-deployer:v0.1
$ kind load docker-image rcs-ocm-deployer:v0.1
$ make deploy IMG=rcs-ocm-deployer:v0.1
```

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

