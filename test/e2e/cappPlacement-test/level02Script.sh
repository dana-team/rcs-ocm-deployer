#!/bin/bash

placement=$(kubectl get placements -A --no-headers | awk {'print $2'})
clusterset=`kubectl get placement $placement  -n default -o jsonpath='{.spec.clusterSets[0]}'`
echo $clusterset
clusters=`kubectl get managedclusters -l cluster.open-cluster-management.io/clusterset=$clusterset -o jsonpath='{range .items[*]}{.metadata.name}{" "}{end}'`
clusters_strings=$(echo $clusters | tr ' ' '|')
# | sed 's/ /| /g'
fixed_clusters_strings+=$clusters_strings

kubectl assert exist-enhanced capp --field-selector metadata.name=capp-with-placement,status.applicationLinks.site=~$fixed_clusters_strings -A
