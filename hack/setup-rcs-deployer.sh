#!/bin/bash

## Initialize variables
kind=""
clusteradm=""
capprelease=""
addonsrelease=""

initialize_kind() {
    if test -s "$1"; then
        kind="$1"
    else
        kind="/usr/local/bin/kind"
    fi
}

initialize_clusteradm() {
    if test -s "$1"; then
        clusteradm="$1"
    else
        clusteradm="/usr/local/bin/clusteradm"
    fi
}

initialize_capp_release() {
    if test -s "$1"; then
        capprelease="$1"
    else
        capprelease="main"
    fi
}

initialize_addons_release() {
    if test -s "$1"; then
        addonsrelease="$1"
    else
        addonsrelease="main"
    fi
}

initialize_kind "$1"
initialize_clusteradm "$2"
initialize_capp_release "$3"
initialize_addons_release "$4"

hub=${CLUSTER1:-hub}
c1=${CLUSTER1:-cluster1}
c2=${CLUSTER2:-cluster2}

hubctx="kind-${hub}"
c1ctx="kind-${c1}"
c2ctx="kind-${c2}"

clusterset="test-clusterset"
ns="test"

cappimage="ghcr.io/dana-team/container-app-operator:${capprelease}"
rcsaddonsimage="ghcr.io/dana-team/rcs-ocm-addons:${addonsrelease}"

capprepo=https://github.com/dana-team/container-app-operator
addonsrepo=https://github.com/dana-team/rcs-ocm-addons
certmanagerurl=https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml

# Create ManagedClusterSet and Placement on Hub
kubectl config use-context "${hubctx}"
"${clusteradm}" create clusterset "${clusterset}"
"${clusteradm}" clusterset set "${clusterset}" --clusters "${c1}"
"${clusteradm}" clusterset set "${clusterset}" --clusters "${c2}"
kubectl create ns "${ns}"
"${clusteradm}" clusterset bind "${clusterset}" --namespace "${ns}"

cat << EOF | kubectl apply -f -
apiVersion: cluster.open-cluster-management.io/v1beta1
kind: Placement
metadata:
  name: test-placement
  namespace: "${ns}"
spec:
  clusterSets:
    - "${clusterset}"
  numberOfClusters: 1
  prioritizerPolicy:
    mode: Exact
    configurations:
      - scoreCoordinate:
          type: AddOn
          addOn:
            resourceName: rcs-score
            scoreName: cpuAvailable
        weight: 1
EOF
cat << EOF | kubectl apply -f -
apiVersion: cluster.open-cluster-management.io/v1beta1
kind: Placement
metadata:
  name: all-clusters
  namespace: "${ns}"
spec:
  clusterSets:
    - "${clusterset}"
EOF

# Install cert-manager on Hub and install Capp CRD
kubectl apply -f ${certmanagerurl}
git clone ${capprepo}
make -C container-app-operator install

# Set up Managed Clusters by installing container-app-operator and its prerequisites on them
kubectl config use-context "${c1ctx}"
make -C container-app-operator prereq
make -C container-app-operator deploy IMG="${cappimage}"
kubectl wait --for=condition=ready pods -l control-plane=controller-manager -n capp-operator-system
make -C container-app-operator create-cappConfig

kubectl config use-context "${c2ctx}"
make -C container-app-operator prereq
make -C container-app-operator deploy IMG="${cappimage}"
kubectl wait --for=condition=ready pods -l control-plane=controller-manager -n capp-operator-system
make -C container-app-operator create-cappConfig

rm -rf container-app-operator/

# Create RCSConfig Object on Hub
kubectl config use-context "${hubctx}"
make install
kubectl create ns rcs-deployer-system
kubectl apply -f hack/manifests/rcsconfig.yaml

## Deploy add-ons on placement and create configuration for them
git clone ${addonsrepo}
make -C rcs-ocm-addons deploy-addons IMG="${rcsaddonsimage}"
rm -rf rcs-ocm-addons

cat <<EOF | kubectl apply -f -
apiVersion: addon.open-cluster-management.io/v1alpha1
kind: AddOnDeploymentConfig
metadata:
  name: rcs-score-deploy-config
  namespace: open-cluster-management-hub
spec:
  agentInstallNamespace: open-cluster-management-agent-addon
  customizedVariables:
  - name: MAX_CPU_COUNT
    value: "4"
  - name: MIN_CPU_COUNT
    value: "0"
  - name: MAX_MEMORY_BYTES
    value: "104857"
  - name: MIN_MEMORY_BYTES
    value: "0"
EOF

kubectl patch clustermanagementaddon rcs-score --type merge -p \
'{"spec":{"installStrategy":{"type":"Placements","placements":[{"name":"all-clusters","namespace":"'"${ns}"'","configs":[{"group":"addon.open-cluster-management.io","resource":"addondeploymentconfigs","name":"rcs-score-deploy-config","namespace":"open-cluster-management-hub"}]}]}}}'
kubectl patch clustermanagementaddon capp-status-addon --type merge -p \
'{"spec":{"installStrategy":{"type":"Placements","placements":[{"name":"all-clusters","namespace":"'"${ns}"'"}]}}}'
