package spoke

import (
	"context"
	"time"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"open-cluster-management.io/api/client/cluster/listers/cluster/v1alpha1"

	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	clientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterinformers1 "open-cluster-management.io/api/client/cluster/informers/externalversions/cluster/v1alpha1"
	apiv1alpha2 "open-cluster-management.io/api/cluster/v1alpha1"
)

// AgentController defines the controller to run the agent.
type AgentController struct {
	spokeKubeClient           kubernetes.Interface
	hubKubeClient             clientset.Interface
	AddOnPlacementScoreLister v1alpha1.AddOnPlacementScoreLister
	clusterName               string
	addonName                 string
	addonNamespace            string
	recorder                  events.Recorder
	nodeInformer              corev1informers.NodeInformer
	podInformer               corev1informers.PodInformer
	Logger                    logr.Logger
}

// newAgentController returns a new instance of agentController.
func newAgentController(spokeKubeClient kubernetes.Interface, hubKubeClient clientset.Interface, addOnPlacementScoreInformer clusterinformers1.AddOnPlacementScoreInformer, clusterName string, addonName string, addonNamespace string, recorder events.Recorder, nodeInformer corev1informers.NodeInformer, podInformer corev1informers.PodInformer, logger logr.Logger) factory.Controller {
	c := &AgentController{
		spokeKubeClient:           spokeKubeClient,
		hubKubeClient:             hubKubeClient,
		clusterName:               clusterName,
		addonName:                 addonName,
		addonNamespace:            addonNamespace,
		AddOnPlacementScoreLister: addOnPlacementScoreInformer.Lister(),
		recorder:                  recorder,
		podInformer:               podInformer,
		nodeInformer:              nodeInformer,
		Logger:                    logger,
	}
	return factory.New().WithInformersQueueKeyFunc(
		func(obj runtime.Object) string {
			key, _ := cache.MetaNamespaceKeyFunc(obj)
			return key
		}, addOnPlacementScoreInformer.Informer()).
		WithBareInformers(podInformer.Informer(), nodeInformer.Informer()).
		WithSync(c.sync).ResyncEvery(time.Second*60).ToController("score-agent-controller", recorder)
}

// sync takes care of the reconciliation logic of the AddOnPlacementScore object and updates
// the scores of based on the calculation on the agent.
func (c *AgentController) sync(ctx context.Context, syncCtx factory.SyncContext) error {
	resourceScore := NewResourceScore(c.nodeInformer, c.podInformer, c.Logger)

	cpuScore, memScore, err := resourceScore.calculateScore()
	if err != nil {
		return err
	}

	items := []apiv1alpha2.AddOnPlacementScoreItem{
		{
			Name:  "cpuAvailable",
			Value: int32(cpuScore),
		},
		{
			Name:  "memAvailable",
			Value: int32(memScore),
		},
	}

	addonPlacementScore, err := c.AddOnPlacementScoreLister.AddOnPlacementScores(c.clusterName).Get(AddOnPlacementScoresName)
	switch {
	case errors.IsNotFound(err):
		addonPlacementScore = &apiv1alpha2.AddOnPlacementScore{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: c.clusterName,
				Name:      AddOnPlacementScoresName,
			},
			Status: apiv1alpha2.AddOnPlacementScoreStatus{
				Scores: items,
			},
		}
		_, err = c.hubKubeClient.ClusterV1alpha1().AddOnPlacementScores(c.clusterName).Create(ctx, addonPlacementScore, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		return nil
	case err != nil:
		return err
	}

	addonPlacementScore.Status.Scores = items
	_, err = c.hubKubeClient.ClusterV1alpha1().AddOnPlacementScores(c.clusterName).UpdateStatus(ctx, addonPlacementScore, metav1.UpdateOptions{})
	return err
}
