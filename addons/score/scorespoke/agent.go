package scorespoke

import (
	"context"
	"time"

	"github.com/go-logr/logr"

	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/spf13/cobra"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"open-cluster-management.io/addon-framework/pkg/lease"
	"open-cluster-management.io/addon-framework/pkg/version"
	clientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterinformers "open-cluster-management.io/api/client/cluster/informers/externalversions"
)

const AddOnPlacementScoresName = "resource-usage-score"

// NewAgentCommand creates a new Cobra command to start the agent for a specific component.
func NewAgentCommand(componentName, addonName string, logger logr.Logger) *cobra.Command {
	o := NewAgentOptions(addonName, logger)
	cmd := controllercmd.
		NewControllerCommandConfig(componentName, version.Get(), o.RunAgent).
		NewCommandWithContext(context.TODO())
	cmd.Use = "agent"
	cmd.Short = "Start the addon agent"

	o.AddFlags(cmd)
	return cmd
}

// AgentOptions defines the flags for workload agent.
type AgentOptions struct {
	Logger            logr.Logger
	HubKubeconfigFile string
	SpokeClusterName  string
	AddonName         string
	AddonNamespace    string
}

// NewAgentOptions returns a new instance of AgentOptions.
func NewAgentOptions(addonName string, logger logr.Logger) *AgentOptions {
	return &AgentOptions{
		AddonName: addonName,
		Logger:    logger,
	}
}

// AddFlags adds flags to the provided Cobra command.
func (o *AgentOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&o.HubKubeconfigFile, "hub-kubeconfig", o.HubKubeconfigFile, "Location of kubeconfig file to connect to hub cluster.")
	flags.StringVar(&o.SpokeClusterName, "cluster-name", o.SpokeClusterName, "Name of spoke cluster.")
	flags.StringVar(&o.AddonNamespace, "addon-namespace", o.AddonNamespace, "Installation namespace of addon.")
	flags.StringVar(&o.AddonName, "addon-name", o.AddonName, "name of the addon.")
}

// RunAgent starts the controllers on agent to process work from hub.
func (o *AgentOptions) RunAgent(ctx context.Context, controllerContext *controllercmd.ControllerContext) error {
	spokeKubeClient, err := kubernetes.NewForConfig(controllerContext.KubeConfig)
	if err != nil {
		o.Logger.Error(err, "failed creating client for spoke")
		return err
	}

	// build kubeinformerfactory of hub cluster
	hubRestConfig, err := clientcmd.BuildConfigFromFlags("" /* leave masterurl as empty */, o.HubKubeconfigFile)
	if err != nil {
		o.Logger.Error(err, "failed building hub rest config")
		return err
	}

	hubClusterClient, err := clientset.NewForConfig(hubRestConfig)
	if err != nil {
		o.Logger.Error(err, "failed creating client for hub")
		return err
	}

	spokeKubeInformerFactory := informers.NewSharedInformerFactory(spokeKubeClient, 10*time.Minute)

	clusterInformers := clusterinformers.NewSharedInformerFactoryWithOptions(hubClusterClient, 10*time.Minute, clusterinformers.WithNamespace(o.SpokeClusterName))

	agent := newAgentController(
		spokeKubeClient,
		hubClusterClient,
		clusterInformers.Cluster().V1alpha1().AddOnPlacementScores(),
		o.SpokeClusterName,
		o.AddonName,
		o.AddonNamespace,
		controllerContext.EventRecorder,
		spokeKubeInformerFactory.Core().V1().Nodes(),
		spokeKubeInformerFactory.Core().V1().Pods(),
		o.Logger,
	)

	leaseUpdater := lease.NewLeaseUpdater(spokeKubeClient, o.AddonName, o.AddonNamespace)

	o.Logger.Info("successfully created agent and lease, running agent")

	go clusterInformers.Start(ctx.Done())
	go spokeKubeInformerFactory.Start(ctx.Done())
	go agent.Run(ctx, 1)
	go leaseUpdater.Start(ctx)

	<-ctx.Done()
	return nil
}
