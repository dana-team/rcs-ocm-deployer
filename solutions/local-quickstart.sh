#!/bin/bash
# Create 1 Hub KinD cluster and 2 Managed KinD clusters using a script from the OCM repository
bash ./solutions/setup-kind-clusters.sh
bash ./solutions/ocm-join-clusters.sh $1
bash ./solutions/setup-rcs-deployer.sh $1