/*
Copyright 2025.

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

package controller

import (
	"context"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	sshcontrollercomv1alpha1 "ssh-monitor-controller/api/v1alpha1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// SshCommandAuditReconciler reconciles a SshCommandAudit object
type SshCommandAuditReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Log           logr.Logger
	EventRecorder record.EventRecorder
}

// +kubebuilder:rbac:groups=ssh.controller.com,resources=sshcommandaudits,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ssh.controller.com,resources=sshcommandaudits/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ssh.controller.com,resources=sshcommandaudits/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SshCommandAudit object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *SshCommandAuditReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Reconciling SshCommandAudit")
	var cr sshcontrollercomv1alpha1.SshCommandAudit
	r.Log.Info("Reconciling", "request.Namespace", req.Namespace, "request.Name", req.Name)
	err := r.Get(ctx, req.NamespacedName, &cr)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info("CR resource not found; may have been deleted.", "name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	cr.Status.TotalCommands = len(cr.Spec.SshCommandEntity)
	err = r.Status().Update(ctx, &cr)
	if err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, err
		}
		r.EventRecorder.Eventf(&cr, corev1.EventTypeWarning, "UpdateStatusFailed", "Failed to update this status,error is %v", err)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *SshCommandAuditReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sshcontrollercomv1alpha1.SshCommandAudit{}).
		Named("sshcommandaudit").WithOptions(controller.Options{
		MaxConcurrentReconciles: 3,
	}).Complete(r)
}
