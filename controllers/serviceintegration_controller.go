// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aiven/aiven-go-client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// ServiceIntegrationReconciler reconciles a ServiceIntegration object
type ServiceIntegrationReconciler struct {
	Controller
}

type ServiceIntegrationHandler struct{}

// +kubebuilder:rbac:groups=aiven.io,resources=serviceintegrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=serviceintegrations/status,verbs=get;update;patch

func (r *ServiceIntegrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, ServiceIntegrationHandler{}, &v1alpha1.ServiceIntegration{})
}

func (r *ServiceIntegrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ServiceIntegration{}).
		Complete(r)
}

func (h ServiceIntegrationHandler) createOrUpdate(avn *aiven.Client, i client.Object, refs []client.Object) error {
	si, err := h.convert(i)
	if err != nil {
		return err
	}

	userConfig, err := si.GetUserConfig()
	if err != nil {
		return err
	}

	var reason string
	var integration *aiven.ServiceIntegration
	if si.Status.ID == "" {
		userConfigMap, err := UserConfigurationToAPIV2(userConfig, []string{"create", "update"})
		if err != nil {
			return err
		}

		integration, err = avn.ServiceIntegrations.Create(
			si.Spec.Project,
			aiven.CreateServiceIntegrationRequest{
				DestinationEndpointID: anyOptional(si.Spec.DestinationEndpointID),
				DestinationService:    anyOptional(si.Spec.DestinationServiceName),
				DestinationProject:    anyOptional(si.Spec.DestinationProjectName),
				IntegrationType:       si.Spec.IntegrationType,
				SourceEndpointID:      anyOptional(si.Spec.SourceEndpointID),
				SourceService:         anyOptional(si.Spec.SourceServiceName),
				SourceProject:         anyOptional(si.Spec.SourceProjectName),
				UserConfig:            userConfigMap,
			},
		)
		if err != nil {
			return fmt.Errorf("cannot createOrUpdate service integration: %w", err)
		}

		reason = "Created"
	} else {
		userConfigMap, err := UserConfigurationToAPIV2(userConfig, []string{"update"})
		if err != nil {
			return err
		}

		integration, err = avn.ServiceIntegrations.Update(
			si.Spec.Project,
			si.Status.ID,
			aiven.UpdateServiceIntegrationRequest{
				UserConfig: userConfigMap,
			},
		)
		reason = "Updated"
		if err != nil {
			if strings.Contains(err.Error(), "user config not changed") {
				return nil
			}
			return err
		}
	}

	si.Status.ID = integration.ServiceIntegrationID

	meta.SetStatusCondition(&si.Status.Conditions,
		getInitializedCondition(reason,
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&si.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason,
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&si.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(si.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h ServiceIntegrationHandler) delete(avn *aiven.Client, i client.Object) (bool, error) {
	si, err := h.convert(i)
	if err != nil {
		return false, err
	}

	err = avn.ServiceIntegrations.Delete(si.Spec.Project, si.Status.ID)
	if err != nil && !aiven.IsNotFound(err) {
		return false, fmt.Errorf("aiven client delete service ingtegration error: %w", err)
	}

	return true, nil
}

func (h ServiceIntegrationHandler) get(_ *aiven.Client, i client.Object) (*corev1.Secret, error) {
	si, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	meta.SetStatusCondition(&si.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&si.ObjectMeta, instanceIsRunningAnnotation, "true")

	return nil, nil
}

func (h ServiceIntegrationHandler) checkPreconditions(avn *aiven.Client, i client.Object) (bool, error) {
	si, err := h.convert(i)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&si.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	// todo: validate SourceEndpointID, DestinationEndpointID when ServiceIntegrationEndpoint kind released

	if si.Spec.SourceServiceName != "" {
		project := si.Spec.SourceProjectName
		if project == "" {
			project = si.Spec.Project
		}
		running, err := checkServiceIsRunning(avn, project, si.Spec.SourceServiceName)
		if !running || err != nil {
			return false, err
		}
	}

	if si.Spec.DestinationServiceName != "" {
		project := si.Spec.DestinationProjectName
		if project == "" {
			project = si.Spec.Project
		}
		running, err := checkServiceIsRunning(avn, project, si.Spec.DestinationServiceName)
		if !running || err != nil {
			return false, err
		}
	}

	return true, nil
}

func (h ServiceIntegrationHandler) convert(i client.Object) (*v1alpha1.ServiceIntegration, error) {
	si, ok := i.(*v1alpha1.ServiceIntegration)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ServiceIntegration")
	}

	return si, nil
}
