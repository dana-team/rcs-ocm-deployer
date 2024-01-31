#!/bin/bash

set -e

clusteradm=""

initialize_clusteradm() {
    if [ -x "$1" ]; then
        clusteradm="$1"
    else
        clusteradm="/usr/local/bin/clusteradm"
    fi
}

initialize_clusteradm "$1"

hub=${CLUSTER1:-hub}
c1=${CLUSTER1:-cluster1}
c2=${CLUSTER2:-cluster2}

hubctx="kind-${hub}"
c1ctx="kind-${c1}"
c2ctx="kind-${c2}"

echo "Initialize the ocm hub cluster\n"
${clusteradm} init --wait --context ${hubctx}
joincmd=$(${clusteradm} get token --context ${hubctx} | grep clusteradm)

echo "Join cluster1 to hub\n"
$(echo ${joincmd} --force-internal-endpoint-lookup --wait --context ${c1ctx} | sed "s/<cluster_name>/$c1/g")

echo "Join cluster2 to hub\n"
$(echo ${joincmd} --force-internal-endpoint-lookup --wait --context ${c2ctx} | sed "s/<cluster_name>/$c2/g")

echo "Accept join of cluster1 and cluster2"
${clusteradm} accept --context ${hubctx} --clusters ${c1},${c2} --wait

kubectl get managedclusters --all-namespaces --context ${hubctx}