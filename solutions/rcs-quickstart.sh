#!/bin/bash

git clone https://github.com/open-cluster-management-io/OCM
./OCM/solutions/setup-dev-environment/local-up.sh
rm -rf OCM/

mkdir bin

kubectl config use-context kind-hub
git clone https://github.com/dana-team/container-app-operator capp
make -C capp install
kubectl apply -f hack/placement.yml

kubectl config use-context kind-cluster1
make -C capp/ prereq
make -C capp deploy IMG=danateam/container-app-operator:release-0.1.3
kubectl config use-context kind-cluster2
make -C capp/ prereq
make -C capp deploy IMG=danateam/container-app-operator:release-0.1.3

rm -rf capp