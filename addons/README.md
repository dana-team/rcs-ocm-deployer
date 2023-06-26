# Capp Status Sync Add-on
By applying this add-on to your OCM hub cluster, the Capp status will automatically be sync back from `managed` (`spoke`) clusters to `hub` cluster.

There are two components:

- The agent that lives on the `managed` (`spoke`)  clusters, which responsible to update the Capp statuses on the `hub` cluster .
- The manager that lives on the `hub` cluster that is responsible for deploying and lifecycle of the agent to the `managed` (`spoke`)  clusters.

# Prerequisite

Set up an Open Cluster Management environment. See: https://open-cluster-management.io/getting-started/quick-start/ for more details

# Get started

Deploy the add-on the OCM `Hub` cluster:

```
$ make deploy-addon
$ kubectl -n open-cluster-management get deploy
NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
capp-status-sync-addon   1/1     1            1           14s
```

The controller will automatically install the add-on agent to all `managed` (`spoke`) clusters.

Validate the add-on agent is installed on a `managed` (`spoke`) cluster:

```
$ kubectl -n open-cluster-management-agent-addon get deploy
NAME                                    READY   UP-TO-DATE   AVAILABLE   AGE
capp-status-sync-addon-agent    1/1     1            1           2m24s
```

You can also validate and check the status of the add-on on the `Hub` cluster:

```
$ kubectl -n cluster1 get managedclusteraddon # replace "cluster1" with your managed cluster name
NAME                                AVAILABLE   DEGRADED   PROGRESSING
capp-status-sync-addon      True                   
```

Undeploy the add-on the OCM `Hub` cluster:
```
$ make undeploy-addon                
```