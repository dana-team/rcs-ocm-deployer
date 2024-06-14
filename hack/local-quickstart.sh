#!/bin/bash

bash ./hack/setup-kind-clusters.sh $1
bash ./hack/ocm-join-clusters.sh $2
bash ./hack/setup-rcs-deployer.sh $1 $2 $3 $4 $5