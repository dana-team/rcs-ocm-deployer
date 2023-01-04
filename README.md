# rcs-ocm-deployer
RCS OCM Deployer is an operator designed to deploy knative services created on the hub to the managed cluster with the lowest load average on specific namespace.

## Description
RCS OCM Deployer depends on two annotations in the knative service's yaml:
dana.io/ocm-placement: "placement" - The name of the placement to decide from which manage cluster to delpoy on
dana.io/ocm-managed-cluster-namespace: "managed-cluster-namespace" - The namespace on the managed cluster to deploy the service to

contains 3 controllers:
service placement controller: the controller extracts the placement from the serivce's annotations and adds an annotation containing the cluster to deploy to accoding the placementDesicion.
service namespace controller: the controller extracts the namespace name from the service's annotaion and creates a manifestWork in the desired managedCluster namespace containing the namespace with the desired name to be deployed and adds an annotation to the service when the namespace has been created.
service controller: the controller extracts the namespace name and the desired cluster from the service's annotations and creates a manifestWork in the desired managedCluster namespace deploying the Service to the managed cluster

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

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

