/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Workflow annotation that dictates which OCM Placement this Workflow should use to determine the managed cluster.
const AnnotationKeyOCMPlacement = "dana.io/ocm-placement"

// ServicePlacementReconciler reconciles a ServicePlacement object
type ServicePlacementReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// ServicePlacementPredicateFunctions defines which service this controller evaluate the placement decision
var ServicePlacementPredicateFunctions = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		newService := e.ObjectNew.(*knativev1.Service)
		return ContainsValidOCMPlacementAnnotation(*newService)

	},
	CreateFunc: func(e event.CreateEvent) bool {
		service := e.Object.(*knativev1.Service)
		return ContainsValidOCMPlacementAnnotation(*service)
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		return false
	},
}

//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=placementdecisions,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=placements,verbs=get;list;watch

func (r *ServicePlacementReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	service := knativev1.Service{}
	if err := r.Client.Get(ctx, req.NamespacedName, &service); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if service.ObjectMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	return r.pickDecision(service, l, ctx)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServicePlacementReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&knativev1.Service{}).
		WithEventFilter(ServicePlacementPredicateFunctions).
		Complete(r)
}

func (r *ServicePlacementReconciler) pickDecision(service knativev1.Service, log logr.Logger, ctx context.Context) (ctrl.Result, error) {
	placementRef := service.Annotations[AnnotationKeyOCMPlacement]
	placement := clusterv1beta1.Placement{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: placementRef, Namespace: service.Namespace}, &placement); err != nil {
		return ctrl.Result{}, err
	}
	placementDecisions, err := r.getPlacementDecisionList(service, log, ctx, placementRef)
	if len(placementDecisions.Items) == 0 {
		log.Info("unable to find any PlacementDecision, try again after 10 seconds")
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	//TODO: consider force placement reeval

	managedClusterName := getDecisionClusterName(placementDecisions, log)
	if managedClusterName == "" {
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}
	log.Info("updating Service with annotation ManagedCluster: " + managedClusterName)
	if err := r.updateServiceAnnotations(service, managedClusterName, ctx); err != nil {
		log.Error(err, "unable to update Knative Service")
		return ctrl.Result{}, err
	}
	log.Info("done reconciling Workflow for Placement evaluation")
	return ctrl.Result{}, nil
}

func (r *ServicePlacementReconciler) getPlacementDecisionList(service knativev1.Service, log logr.Logger, ctx context.Context, placementRef string) (*clusterv1beta1.PlacementDecisionList, error) {

	listopts := &client.ListOptions{}
	// query all placementdecisions of the placement
	requirement, err := labels.NewRequirement(clusterv1beta1.PlacementLabel, selection.Equals, []string{placementRef})
	if err != nil {
		log.Error(err, "unable to create new PlacementDecision label requirement")
		return nil, err
	}
	labelSelector := labels.NewSelector().Add(*requirement)
	listopts.LabelSelector = labelSelector
	listopts.Namespace = service.Namespace
	placementDecisions := &clusterv1beta1.PlacementDecisionList{}
	if err = r.Client.List(ctx, placementDecisions, listopts); err != nil {
		log.Error(err, "unable to list PlacementDecisions")
		return nil, err
	}
	return placementDecisions, nil
}

func getDecisionClusterName(placementDecisions *clusterv1beta1.PlacementDecisionList, log logr.Logger) string {
	// TODO only handle one PlacementDecision target for now
	pd := placementDecisions.Items[0]
	if len(pd.Status.Decisions) == 0 {
		log.Info("unable to find any Decisions from PlacementDecision, try again after 10 seconds")
		return ""
	}

	// TODO only using the first decision
	managedClusterName := pd.Status.Decisions[0].ClusterName
	if len(managedClusterName) == 0 {
		log.Info("unable to find a valid ManagedCluster from PlacementDecision, try again after 10 seconds")
		return ""
	}
	return managedClusterName
}

func (r *ServicePlacementReconciler) updateServiceAnnotations(service knativev1.Service, managedClusterName string, ctx context.Context) error {
	service.Annotations[AnnotationKeyOCMPlacement] = ""
	service.Annotations[AnnotationKeyOCMManagedCluster] = managedClusterName
	return r.Client.Update(ctx, &service)
}
