#!/bin/bash

bash $(shell pwd)/solutions/ocm-join-clusters.sh $1
bash $(shell pwd)/solutions/setup-rcs-deployer.sh $1