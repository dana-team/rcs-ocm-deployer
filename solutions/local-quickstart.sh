#!/bin/bash


bash $(pwd)/solutions/setup-kind-clusters.sh
bash $(pwd)/solutions/ocm-join-clusters.sh $1
bash $(pwd)/solutions/setup-rcs-deployer.sh $1