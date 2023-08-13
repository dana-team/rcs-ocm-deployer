# Capp status sync add-on

By applying this add-on to your `OCM` Hub Cluster, the `Capp` status will automatically be synced back from Managed/Spoke Clusters to the Hub cluster.

## The components

- `agent` - deployed on the Managed/Spoke clusters; responsible for syncing the `Capp` status between the Managed Cluster and the Hub Cluster.

- `manager` - deployed on the Hub Cluster; responsible for deploying the `agent` on the Managed/Spoke clusters.

## Getting Started

1. Deploy the add-on the Hub Cluster.

    ```bash
    $ make deploy-addon IMG=ghcr.io/dana-team/rcs-ocm-deployer:<release>
    $ kubectl -n open-cluster-management get deploy

    NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
    capp-status-sync-addon   1/1     1            1           14s
    ```

2. The controller will automatically install the add-on `agent` on all Managed/Spoke Clusters. Validate the add-on agent is installed on a Managed/Spoke` cluster:

    ```bash
    $ kubectl -n open-cluster-management-agent-addon get deploy

    NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
    capp-status-sync-addon-agent    1/1     1            1           2m24s
    ```

3. You can also validate and check the status of the add-on on the Hub cluster:

    ```bash
    # replace "cluster1" with your managed cluster name
    $ kubectl -n cluster1 get managedclusteraddon
    
    NAME                                AVAILABLE   DEGRADED   PROGRESSING
    capp-status-sync-addon      True                   
    ```

4. Undeploy the add-on:

    ```bash
    $ make undeploy-addon                
    ```
