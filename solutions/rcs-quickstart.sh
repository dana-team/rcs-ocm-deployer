#!/bin/bash
# Create 1 Hub KinD cluster and 2 Managed KinD clusters using a script from the OCM repository
git clone https://github.com/open-cluster-management-io/OCM
bash ./OCM/solutions/setup-dev-environment/local-up.sh
rm -rf OCM/

# Create ManagedClusterSet and Palcement on Hub
kubectl config use-context kind-hub
clusteradm create clusterset test-clusterset
clusteradm clusterset set test-clusterset --clusters cluster1
clusteradm clusterset set test-clusterset --clusters cluster2
kubectl create ns test
clusteradm clusterset bind test-clusterset --namespace test
cat <<EOF | kubectl apply -f -
apiVersion: cluster.open-cluster-management.io/v1beta1
kind: Placement
metadata:
  name: test-placement
  namespace: test
spec:
  clusterSets:
  - test-clusterset
EOF

# Install cert-manager on Hub and install Capp CRD
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml
git clone https://github.com/dana-team/container-app-operator
make -C container-app-operator install

# Set up Managed Clusters by installing container-app-operator and its prerequisites on them
kubectl config use-context kind-cluster1
make -C container-app-operator prereq
make -C container-app-operator deploy IMG=$1
kubectl config use-context kind-cluster2
make -C container-app-operator prereq
make -c container-app-operator deploy IMG=$1
rm -rf container-app-operator/

# Create RCSConfig Object on Hub
kubectl config use-context kind-hub
make install
kubectl create ns rcs-deployer-system
cat <<EOF | kubectl apply -f -
apiVersion: rcsd.dana.io/v1alpha1
kind: RCSConfig
metadata:
  name: rcs-config
  namespace: rcs-deployer-system
spec:
  placements:
  - test-placement
  placementsNamespace: test
EOF
