package controllers

import (
	"context"

	rcsv1alpha1 "github.com/dana-team/rcs-ocm-deployer/api/v1alpha1"
	utils "github.com/dana-team/rcs-ocm-deployer/internals/utils"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	v1 "open-cluster-management.io/api/work/v1"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ServiceNamespaceReconciler reconciles a ServiceNamespace object
type ServiceNamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	// NamespaceManifestWorkPrefix prefix of the manifest work creating a namespace on the managed cluster
	NamespaceManifestWorkPrefix  = "mw-create-"
	FinalizerCleanupManifestWork = "dana.io/cleanup-ocm-manifestwork"
)

func (r *ServiceNamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	capp := rcsv1alpha1.Capp{}
	if err := r.Client.Get(ctx, req.NamespacedName, &capp); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	l.Info("Start reconciling Service in ServiceNamespace controller")
	if capp.ObjectMeta.DeletionTimestamp != nil {
		// remove finalizer from Workflow but do not 'commit' yet
		if controllerutil.ContainsFinalizer(&capp, FinalizerCleanupManifestWork) {
			if err := r.finalizeService(ctx, NamespaceManifestWorkPrefix+"-"+capp.Namespace+"-"+capp.Name, capp.Status.ApplicationLinks.Site, l); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&capp, FinalizerCleanupManifestWork)
			if err := r.Client.Update(ctx, &capp); err != nil {
				return ctrl.Result{}, nil
			}
		}
	}

	if err := r.ensureFinalizer(ctx, capp); err != nil {
		return ctrl.Result{}, err
	}
	// verify the ManagedCluster actually exists
	if err := r.verifyManagedClusterExistence(ctx, l, capp.Status.ApplicationLinks.Site); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.ensureNamespaceExistence(ctx, capp); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.SyncManifestWork(capp, ctx, l); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

var CappPredicateFuncs = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		newCapp := e.ObjectNew.(*rcsv1alpha1.Capp)
		return utils.ContainesPlacementAnnotation(*newCapp)

	},
	CreateFunc: func(e event.CreateEvent) bool {
		capp := e.Object.(*rcsv1alpha1.Capp)
		return utils.ContainesPlacementAnnotation(*capp)
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		capp := e.Object.(*rcsv1alpha1.Capp)
		return utils.ContainesPlacementAnnotation(*capp)
	},
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceNamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rcsv1alpha1.Capp{}).
		WithEventFilter(CappPredicateFuncs).
		Complete(r)
}

// ensureNamespaceExistence gets context and service
// The function ensures the manifest work containing the namespace exists, if it doesn't it creates it
// Returns an error if occured
func (r *ServiceNamespaceReconciler) ensureNamespaceExistence(ctx context.Context, capp rcsv1alpha1.Capp) error {
	if utils.ContainsValidOCMNamespaceAnnotation(capp) {
		return nil
	}
	managedClusterName := capp.Status.ApplicationLinks.Site
	mwName := NamespaceManifestWorkPrefix + capp.Namespace + "-" + capp.Name
	ok, err := r.checkManifestWorkExistence(ctx, mwName, managedClusterName)
	if err != nil {
		return err
	}
	if !ok {
		ns := utils.GenerateNamespace(capp.Namespace)
		w := utils.GenerateManifestWorkGeneric(mwName, managedClusterName, &ns)
		if err := r.Client.Create(ctx, w); err != nil {
			return err
		}
	}
	if err := r.addNamespaceCreatedAnnotation(ctx, capp); err != nil {
		return err
	}
	return nil
}

// checkManifestWorkExistence gets context, manifest work name and manifest work namespace
// The function returns true whether the manifest work already exist and false otherwise
func (r *ServiceNamespaceReconciler) checkManifestWorkExistence(ctx context.Context, mwName string, mwNamespace string) (bool, error) {
	mw := v1.ManifestWork{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: mwName, Namespace: mwNamespace}, &mw); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

// addNamespaceCreatedAnnotation get context and service
// The function add annotation to the service that namespace created
// Return error if occured
func (r *ServiceNamespaceReconciler) addNamespaceCreatedAnnotation(ctx context.Context, service rcsv1alpha1.Capp) error {
	cappAnno := service.GetAnnotations()
	if cappAnno == nil {
		cappAnno = make(map[string]string)
	}
	cappAnno["AnnotationNamespaceCreated"] = "true"
	service.SetAnnotations(cappAnno)
	if err := r.Client.Update(ctx, &service); err != nil {
		return err
	}
	return nil
}

func (r *ServiceNamespaceReconciler) removeCreatedAnnotation(ctx context.Context, service rcsv1alpha1.Capp) error {
	cappAnno := service.GetAnnotations()
	delete(cappAnno, "AnnotationNamespaceCreated")
	service.SetAnnotations(cappAnno)
	if err := r.Client.Update(ctx, &service); err != nil {
		return err
	}
	return nil
}

// finalizeService gets context, manifest work name, managed cluster name and logger
// The function checks whether the manifest work deploying the service exists
// If it does it deletes it
func (r *ServiceNamespaceReconciler) finalizeService(ctx context.Context, mwName string, managedClusterName string, log logr.Logger) error {
	// delete the ManifestWork associated with this service
	var work v1.ManifestWork
	if err := r.Get(ctx, types.NamespacedName{Name: mwName, Namespace: managedClusterName}, &work); err != nil {
		if errors.IsNotFound(err) {
			log.Info("already deleted ManifestWork, commit the Workflow finalizer removal")
			return nil
		} else {
			log.Error(err, "unable to fetch ManifestWork")
			return err
		}
	}
	if err := r.Delete(ctx, &work); err != nil {
		log.Error(err, "unable to delete ManifestWork")
		return err
	}
	return nil
}

// ensureFinalizer ensures the service has the finalizer
func (r *ServiceNamespaceReconciler) ensureFinalizer(ctx context.Context, service rcsv1alpha1.Capp) error {
	if !controllerutil.ContainsFinalizer(&service, FinalizerCleanupManifestWork) {
		controllerutil.AddFinalizer(&service, FinalizerCleanupManifestWork)
		if err := r.Client.Update(ctx, &service); err != nil {
			return err
		}
	}
	return nil
}

// verifyManagedClusterExistence get managed cluster name and checks whether it exists or not
func (r *ServiceNamespaceReconciler) verifyManagedClusterExistence(ctx context.Context, l logr.Logger, managedClusterName string) error {
	managedCluster := clusterv1.ManagedCluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: managedClusterName}, &managedCluster); err != nil {
		l.Error(err, "unable to fetch ManagedCluster")
		return err
	}
	return nil
}

func (r *ServiceNamespaceReconciler) GatherCappResources(capp rcsv1alpha1.Capp, ctx context.Context, l logr.Logger) (error, []v1.Manifest) {
	manifests := []v1.Manifest{}
	svc := utils.PrepareServiceForWorkPayload(capp)
	ns := utils.GenerateNamespace(capp.Namespace)
	manifests = append(manifests, v1.Manifest{RawExtension: runtime.RawExtension{Object: &svc}}, v1.Manifest{RawExtension: runtime.RawExtension{Object: &ns}})
	configMaps, secrets := r.GetResourceVolumesFromContainerSpec(capp, ctx, l)
	for _, resource := range configMaps {
		cm := &corev1.ConfigMap{}
		if err := r.Get(ctx, types.NamespacedName{Name: resource, Namespace: capp.Namespace}, cm); err != nil {
			l.Error(err, "unable to fetch configmap")
			return err, nil
		} else {
			manifests = append(manifests, v1.Manifest{RawExtension: runtime.RawExtension{Object: cm}})
		}
	}
	for _, resource := range secrets {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: resource, Namespace: capp.Namespace}, secret); err != nil {
			l.Error(err, "unable to fetch secret")
			return err, nil
		} else {
			manifests = append(manifests, v1.Manifest{RawExtension: runtime.RawExtension{Object: secret}})
		}
	}
	return nil, manifests
}

func (r *ServiceNamespaceReconciler) GetResourceVolumesFromContainerSpec(capp rcsv1alpha1.Capp, ctx context.Context, l logr.Logger) ([]string, []string) {
	var configMaps []string
	var secrets []string
	for _, containerSpec := range capp.Spec.ConfigurationSpec.Template.Spec.Containers {
		for _, resourceEnv := range containerSpec.EnvFrom {
			if resourceEnv.ConfigMapRef != nil {
				configMaps = append(configMaps, resourceEnv.ConfigMapRef.Name)
			}
			if resourceEnv.SecretRef != nil {
				secrets = append(secrets, resourceEnv.SecretRef.Name)
			}
		}
	}
	for _, volume := range capp.Spec.ConfigurationSpec.Template.Spec.Volumes {

		if volume.ConfigMap != nil {
			configMaps = append(configMaps, volume.ConfigMap.Name)
		}
		if volume.Secret != nil {
			secrets = append(secrets, volume.Secret.SecretName)
		}
	}

	return configMaps, secrets
}

// EnsureManifestWork checks whether the manifest work deploying the service exists in the managed cluster namespace
// If it does, it updates the service in the manifest work spec, if it doesn't, it creates it
func (r *ServiceNamespaceReconciler) SyncManifestWork(capp rcsv1alpha1.Capp, ctx context.Context, l logr.Logger) error {
	mwName := NamespaceManifestWorkPrefix + capp.Namespace + "-" + capp.Name
	managedClusterName := capp.Status.ApplicationLinks.Site

	var mw v1.ManifestWork
	err := r.Get(ctx, types.NamespacedName{Name: mwName, Namespace: managedClusterName}, &mw)
	if errors.IsNotFound(err) {
		return r.removeCreatedAnnotation(ctx, capp)
	}

	if err == nil {
		err, manifests := r.GatherCappResources(capp, ctx, l)
		if err != nil {
			//TODO handle secret not found
			return err
		}
		mw.Spec.Workload.Manifests = manifests
		if err = r.Client.Update(ctx, &mw); err != nil {
			l.Error(err, "unable to update ManifestWork")
			return err
		}
		l.Info("done reconciling Workflow")
		return nil
	}
	l.Error(err, "unable to fetch ManifestWork")
	return err
}
