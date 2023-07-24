resource=$1

placement=$(kubectl get placements -A --no-headers | awk {'print $2'})
placementNS=$(kubectl get placements -A --no-headers | awk {'print $1'})
clusterset=$(kubectl get placement $placement -n $placementNS -o jsonpath='{.spec.clusterSets[0]}')
cluster=$(kubectl get managedclusters -l cluster.open-cluster-management.io/clusterset=$clusterset  --no-headers | awk {'print $1'})

if [ $resource = "placement" ]
then
    echo $placement
elif [ $resource = "cluster" ]
then
    echo $cluster
fi