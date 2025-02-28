// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var clickhouselog = logf.Log.WithName("clickhouse-resource")

func (r *Clickhouse) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-clickhouse,mutating=true,failurePolicy=fail,groups=aiven.io,resources=clickhouses,verbs=create;update,versions=v1alpha1,name=mclickhouse.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &Clickhouse{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Clickhouse) Default() {
	clickhouselog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-clickhouse,mutating=false,failurePolicy=fail,groups=aiven.io,resources=clickhouses,versions=v1alpha1,name=vclickhouse.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Clickhouse{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Clickhouse) ValidateCreate() error {
	clickhouselog.Info("validate create", "name", r.Name)

	return r.Spec.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Clickhouse) ValidateUpdate(old runtime.Object) error {
	clickhouselog.Info("validate update", "name", r.Name)

	if r.Spec.Project != old.(*Clickhouse).Spec.Project {
		return errors.New("cannot update a Clickhouse service, project field is immutable and cannot be updated")
	}

	if r.Spec.ConnInfoSecretTarget.Name != old.(*Clickhouse).Spec.ConnInfoSecretTarget.Name {
		return errors.New("cannot update a Clickhouse service, connInfoSecretTarget.name field is immutable and cannot be updated")
	}

	return r.Spec.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Clickhouse) ValidateDelete() error {
	clickhouselog.Info("validate delete", "name", r.Name)

	if r.Spec.TerminationProtection != nil && *r.Spec.TerminationProtection {
		return errors.New("cannot delete Clickhouse service, termination protection is on")
	}

	return nil
}
