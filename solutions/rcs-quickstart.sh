#!/bin/bash
# Step 1: Create 1 hub kind cluster and 2 managed Kind Clusters using a script from the OCM repository
git clone https://github.com/open-cluster-management-io/OCM
./OCM/solutions/setup-dev-environment/local-up.sh
rm -rf OCM/

# Step 2: Install capp CRD on hub cluster 
kubectl config use-context kind-hub
git clone https://github.com/dana-team/container-app-operator capp
make -C capp install

# Step 3: Install and deploy capp and prerequisites on managed clusters
kubectl config use-context kind-cluster1
make -C capp/ prereq
make -C capp deploy IMG=$1
kubectl config use-context kind-cluster2
make -C capp/ prereq
make -C capp deploy IMG=$1

# Step 4: Cleanup
rm -rf capp
