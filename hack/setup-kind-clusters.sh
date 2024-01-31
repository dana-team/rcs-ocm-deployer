#!/bin/bash

cd $(dirname ${BASH_SOURCE})

set -e

initialize_kind() {
    if test -s "$1"; then
        kind="$1"
    else
        kind="/usr/local/bin/kind"
    fi
}

initialize_kind "$1"

hub=${CLUSTER1:-hub}
c1=${CLUSTER1:-cluster1}
c2=${CLUSTER2:-cluster2}

hubctx="kind-${hub}"
c1ctx="kind-${c1}"
c2ctx="kind-${c2}"

"${kind}" create cluster --name "${hub}"
"${kind}" create cluster --name "${c1}"
"${kind}" create cluster --name "${c2}"