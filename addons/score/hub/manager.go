package hub

import (
	"context"
	"embed"
	"fmt"
	"os"

	"github.com/go-logr/logr"

	"github.com/openshift/library-go/pkg/assets"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"github.com/spf13/cobra"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/component-base/version"
	"open-cluster-management.io/addon-framework/pkg/addonfactory"
	"open-cluster-management.io/addon-framework/pkg/addonmanager"
	frameworkagent "open-cluster-management.io/addon-framework/pkg/agent"
	"open-cluster-management.io/addon-framework/pkg/utils"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	addonv1alpha1client "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

const (
	addonImageDefault          = "ghcr.io/dana-team/rcs-ocm-deployer:main"
	AddonName                  = "rcs-score"
	templatePath               = "manifests/templates"
	agentInstallationNamespace = "open-cluster-management-agent-addon"
)

//go:embed manifests
//go:embed manifests/templates
var fs embed.FS

var agentPermissionFiles = []string{
	// role with RBAC rules to access resources on hub
	"manifests/permission/role.yaml",
	// rolebinding to bind the above role to a certain user group
	"manifests/permission/rolebinding.yaml",
}

// NewManagerCommand creates a new Cobra command to start the manager for a specific component.
// The anonymous function runController takes care of the logic of registering the agent addon and
// registering the supported configurations types to the addon.
func NewManagerCommand(componentName string, logger logr.Logger) *cobra.Command {
	runController := func(ctx context.Context, controllerContext *controllercmd.ControllerContext) error {
		addonClient, err := addonv1alpha1client.NewForConfig(controllerContext.KubeConfig)
		if err != nil {
			logger.Error(err, "failed creating an addon client")
			return err
		}

		mgr, err := addonmanager.New(controllerContext.KubeConfig)
		if err != nil {
			logger.Error(err, "failed creating an addon manager")
			return err
		}

		registrationOption := newRegistrationOption(
			controllerContext.KubeConfig,
			controllerContext.EventRecorder,
			utilrand.String(5))

		// Set agent install namespace from addon deployment config if it exists
		registrationOption.AgentInstallNamespace = utils.AgentInstallNamespaceFromDeploymentConfigFunc(
			utils.NewAddOnDeploymentConfigGetter(addonClient),
		)

		agentAddon, err := addonfactory.NewAgentAddonFactory(AddonName, fs, templatePath).
			// register the supported configuration types
			WithConfigGVRs(utils.AddOnDeploymentConfigGVR).
			WithGetValuesFuncs(
				getValueForAgentTemplate,
				addonfactory.GetValuesFromAddonAnnotation,
				// get the AddOnDeploymentConfig object and transform it to Values object
				addonfactory.GetAddOnDeploymentConfigValues(
					utils.NewAddOnDeploymentConfigGetter(addonClient),
					addonfactory.ToAddOnDeploymentConfigValues,
				),
			).WithAgentRegistrationOption(registrationOption).
			BuildTemplateAgentAddon()

		if err != nil {
			logger.Error(err, "failed building agent")
			return err
		}

		err = mgr.AddAgent(agentAddon)
		if err != nil {
			logger.Error(err, "failed adding agent to addon manager")
			return err
		}

		err = mgr.Start(ctx)
		if err != nil {
			logger.Error(err, "failed to start addon manager")
		}
		<-ctx.Done()

		return nil
	}

	cmdConfig := controllercmd.NewControllerCommandConfig(componentName, version.Get(), runController)

	cmd := cmdConfig.NewCommandWithContext(context.TODO())
	cmd.Use = "manager"
	cmd.Short = fmt.Sprintf("Start the %s's manager", componentName)

	// add disable leader election flag
	flags := cmd.Flags()
	flags.BoolVar(&cmdConfig.DisableLeaderElection, "disable-leader-election", true, "Disable leader election for the agent.")

	return cmd
}

// newRegistrationOption returns a RegistrationOption object which defines how the agent is registered to the hub cluster.
// It defines what CSR with what subject/signer should be created, how the CSR is approved and the RBAC setting of agent on the hub.
func newRegistrationOption(kubeConfig *rest.Config, recorder events.Recorder, agentName string) *frameworkagent.RegistrationOption {
	return &frameworkagent.RegistrationOption{
		CSRConfigurations: frameworkagent.KubeClientSignerConfigurations(AddonName, agentName),
		CSRApproveCheck:   utils.DefaultCSRApprover(agentName),
		PermissionConfig: func(cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn) error {
			kubeClient, err := kubernetes.NewForConfig(kubeConfig)
			if err != nil {
				return fmt.Errorf("failed creating client: %v", err.Error())
			}

			for _, file := range agentPermissionFiles {
				if err := applyAgentPermissionManifestFromFile(file, cluster.Name, addon.Name, kubeClient, recorder); err != nil {
					return fmt.Errorf("failed applying agent permission manifest: %v", err.Error())
				}
			}

			return nil
		},
	}
}

// applyAgentPermissionManifestFromFile applies the agent permission manifest from the specified file.
// It generates necessary configuration data based on the clusterName and addonName.
// The function reads the manifest file, substitutes placeholders with the generated configuration, and applies the manifests.
func applyAgentPermissionManifestFromFile(file, clusterName, addonName string, kubeClient *kubernetes.Clientset, recorder events.Recorder) error {
	groups := frameworkagent.DefaultGroups(clusterName, addonName)
	config := struct {
		ClusterName            string
		Group                  string
		RoleAndRolebindingName string
	}{
		ClusterName:            clusterName,
		Group:                  groups[0],
		RoleAndRolebindingName: fmt.Sprintf("open-cluster-management:%s:%s:agent", addonName, clusterName),
	}

	results := resourceapply.ApplyDirectly(context.Background(),
		resourceapply.NewKubeClientHolder(kubeClient),
		recorder,
		resourceapply.NewResourceCache(),
		func(name string) ([]byte, error) {
			template, err := fs.ReadFile(file)
			if err != nil {
				return nil, err
			}
			return assets.MustCreateAssetFromTemplate(name, template, config).Data, nil
		},
		file,
	)

	for _, result := range results {
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}

// getValueForAgentTemplate takes care of substituting the values in the addon template with given values.
func getValueForAgentTemplate(cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn) (addonfactory.Values, error) {
	installNamespace := addon.Spec.InstallNamespace
	if len(installNamespace) == 0 {
		installNamespace = agentInstallationNamespace
	}

	addonImage := os.Getenv("ADDON_IMAGE")
	if len(addonImage) == 0 {
		addonImage = addonImageDefault
	}

	manifestConfig := struct {
		KubeConfigSecret        string
		ClusterName             string
		AddonName               string
		AddonInstallNamespace   string
		Image                   string
		SpokeRolebindingName    string
		AgentServiceAccountName string
	}{
		KubeConfigSecret:        fmt.Sprintf("%s-hub-kubeconfig", addon.Name),
		AddonInstallNamespace:   installNamespace,
		ClusterName:             cluster.Name,
		AddonName:               fmt.Sprintf("%s-agent", addon.Name),
		Image:                   addonImage,
		SpokeRolebindingName:    addon.Name,
		AgentServiceAccountName: fmt.Sprintf("%s-agent-sa", addon.Name),
	}

	return addonfactory.StructToValues(manifestConfig), nil
}
