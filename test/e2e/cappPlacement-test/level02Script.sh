#!/bin/bash

clusterset=`kubectl get placement placement-1st -n default -o jsonpath='{.spec.clusterSets[0]}'`
echo $clusterset
clusters=`kubectl get managedclusters -l cluster.open-cluster-management.io/clusterset=$clusterset -o jsonpath='{range .items[*]}{.metadata.name}{" "}{end}'`
clusters_strings=$(echo $clusters | tr ' ' '|')
# | sed 's/ /| /g'
fixed_clusters_strings+=$clusters_strings

kubectl assert exist-enhanced capp --field-selector metadata.name=capp-with-placement,status.applicationLinks.site=~$fixed_clusters_strings -A
