// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KafkaACLReconciler reconciles a KafkaACL object
type KafkaACLReconciler struct {
	Controller
}

type KafkaACLHandler struct {
	Handlers
}

// +kubebuilder:rbac:groups=aiven.io,resources=kafkaacls,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=aiven.io,resources=kafkaacls/status,verbs=get

func (r *KafkaACLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("kafkaacl", req.NamespacedName)

	log.Info("reconciling aiven kafka acl")

	const finalizer = "kafka-acl-finalizer.aiven.io"
	acl := &k8soperatorv1alpha1.KafkaACL{}
	return r.reconcileInstance(&KafkaACLHandler{}, ctx, log, req, acl, finalizer)
}

func (r *KafkaACLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.KafkaACL{}).
		Complete(r)
}

func (h KafkaACLHandler) create(c *aiven.Client, _ logr.Logger, i client.Object) (client.Object, error) {
	acl, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	a, err := c.KafkaACLs.Create(
		acl.Spec.Project,
		acl.Spec.ServiceName,
		aiven.CreateKafkaACLRequest{
			Permission: acl.Spec.Permission,
			Topic:      acl.Spec.Topic,
			Username:   acl.Spec.Username,
		},
	)
	if err != nil && !aiven.IsAlreadyExists(err) {
		return nil, err
	}

	h.setStatus(acl, a)

	return acl, nil
}

func (h KafkaACLHandler) delete(c *aiven.Client, log logr.Logger, i client.Object) (bool, error) {
	acl, err := h.convert(i)
	if err != nil {
		return false, err
	}

	err = c.KafkaACLs.Delete(acl.Status.Project, acl.Status.ServiceName, acl.Status.ID)
	if err != nil && !aiven.IsNotFound(err) {
		log.Error(err, "Cannot delete Kafka ACL")
		return false, fmt.Errorf("aiven client delete Kafka ACL error: %w", err)
	}

	log.Info("successfully finalized kafka acl service on aiven side")

	return true, nil
}

func (h KafkaACLHandler) exists(c *aiven.Client, _ logr.Logger, i client.Object) (exists bool, error error) {
	acl, err := h.convert(i)
	if err != nil {
		return false, err
	}

	var aivenACL *aiven.KafkaACL
	if acl.Status.ID != "" {
		aivenACL, err = c.KafkaACLs.Get(acl.Spec.Project, acl.Spec.ServiceName, acl.Status.ID)
		if err != nil {
			return false, err
		}
	} else {
		list, err := c.KafkaACLs.List(acl.Spec.Project, acl.Spec.ServiceName)
		if err != nil {
			return false, err
		}

		for _, a := range list {
			if acl.Spec.Topic == a.Topic && acl.Spec.Username == a.Username && acl.Spec.Permission == a.Permission {
				aivenACL = a
			}
		}
	}

	return aivenACL != nil, nil
}

func (h KafkaACLHandler) update(_ *aiven.Client, _ logr.Logger, _ client.Object) (client.Object, error) {
	return nil, nil
}

func (h KafkaACLHandler) getSecret(_ *aiven.Client, _ logr.Logger, _ client.Object) (*corev1.Secret, error) {
	return nil, nil
}

func (h KafkaACLHandler) checkPreconditions(c *aiven.Client, _ logr.Logger, i client.Object) bool {
	acl, err := h.convert(i)
	if err != nil {
		return false
	}

	return checkServiceIsRunning(c, acl.Spec.Project, acl.Spec.ServiceName)
}

func (h KafkaACLHandler) isActive(_ *aiven.Client, _ logr.Logger, _ client.Object) (bool, error) {
	return true, nil
}

func (h KafkaACLHandler) convert(i client.Object) (*k8soperatorv1alpha1.KafkaACL, error) {
	acl, ok := i.(*k8soperatorv1alpha1.KafkaACL)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaACL")
	}

	return acl, nil
}

func (h KafkaACLHandler) setStatus(acl *k8soperatorv1alpha1.KafkaACL, a *aiven.KafkaACL) {
	acl.Status.Project = acl.Spec.Project
	acl.Status.ServiceName = acl.Spec.ServiceName
	acl.Status.Username = a.Username
	acl.Status.Permission = a.Permission
	acl.Status.Topic = a.Topic
	acl.Status.ID = a.ID
}

func (h KafkaACLHandler) getSecretReference(i client.Object) *k8soperatorv1alpha1.AuthSecretReference {
	acl, err := h.convert(i)
	if err != nil {
		return nil
	}

	return &acl.Spec.AuthSecretRef
}