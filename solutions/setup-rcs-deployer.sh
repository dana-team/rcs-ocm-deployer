#!/bin/bash

hub=${CLUSTER1:-hub}
c1=${CLUSTER1:-cluster1}
c2=${CLUSTER2:-cluster2}

hubctx="kind-${hub}"
c1ctx="kind-${c1}"
c2ctx="kind-${c2}"

cappimage="ghcr.io/dana-team/rcs-ocm-deployer:main"
clusterset="test-clusterset"
ns="test"

# Create ManagedClusterSet and Palcement on Hub
kubectl config use-context "${hubctx}"
clusteradm create clusterset "${clusterset}"
clusteradm clusterset set "${clusterset}" --clusters "${c1}"
clusteradm clusterset set "${clusterset}" --clusters "${c2}"
kubectl create ns "${ns}"
clusteradm clusterset bind "${clusterset}" --namespace "${ns}"
cat <<EOF | kubectl apply -f -
apiVersion: cluster.open-cluster-management.io/v1beta1
kind: Placement
metadata:
  name: test-placement
  namespace: "${ns}"
spec:
  clusterSets:
  - "${clusterset}"
EOF

# Install cert-manager on Hub and install Capp CRD
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml
git clone https://github.com/dana-team/container-app-operator
make -C container-app-operator install

# Set up Managed Clusters by installing container-app-operator and its prerequisites on them
kubectl config use-context "${c1ctx}"
make -C container-app-operator prereq
make -C container-app-operator deploy IMG="${cappimage}"
kubectl config use-context "${c2ctx}"
make -C container-app-operator prereq
make -c container-app-operator deploy IMG="${cappimage}"
rm -rf container-app-operator/

# Create RCSConfig Object on Hub
kubectl config use-context "${hubctx}"
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