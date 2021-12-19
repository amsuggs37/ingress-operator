/*
Copyright 2021.

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
	"os"

	solov1alpha1 "github.com/amsuggs37/ingress-operator.git/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// IngressOperatorReconciler reconciles a IngressOperator object
type IngressOperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	opNamespace string
	opName      string

	ingExternalIps bool
	ingClass       string
}

var (
	opNamespaceVar = "OPERATOR_NAMESPACE"
	opNameVar      = "OPERATOR_NAME"

	ingExternalIpsVar = "INGRESS_EXTERNAL_IPS"
	ingClassVar       = "INGRESS_CLASS_NAME"
)

func (r *IngressOperatorReconciler) initController() {
	r.opNamespace = os.Getenv(opNamespaceVar)
	r.opName = os.Getenv(opNameVar)
}

//+kubebuilder:rbac:groups=solo.solo.com,resources=ingressoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=solo.solo.com,resources=ingressoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=solo.solo.com,resources=ingressoperators/finalizers,verbs=update
//+kubebuilder:rbac:groups=,resources=services,verbs=get;list;update
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IngressOperator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *IngressOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cLog := log.FromContext(ctx)

	// load the ingress operator
	opr := &solov1alpha1.IngressOperator{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: r.opNamespace,
		Name:      r.opName,
	}, opr)
	// if there was an error loading, log and return
	if err != nil {
		cLog.Error(err, "Failed to retrieve ingress operator for reconcile!", "opNamespace:", r.opNamespace, "opName:", r.opName)
		return ctrl.Result{}, err
	}

	// load all the service request
	svc := &corev1.Service{}
	err = r.Get(ctx, req.NamespacedName, svc)
	// if there was an error loading, log and return
	if err != nil {
		cLog.Error(err, "Failed to retrieve service request!", "namespace:", req.Namespace, "name:", req.Name)
		return ctrl.Result{}, err
	}

	if len(svc.Spec.ExternalIPs) == 0 || r.ingExternalIps {
		ing := &networkingv1.Ingress{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Ingress",
				APIVersion: "networking.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      svc.Name + "-ing",
				Namespace: svc.Namespace,
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: &r.ingClass,
				DefaultBackend: &networkingv1.IngressBackend{
					Service: &networkingv1.IngressServiceBackend{
						Name: svc.Name,
						Port: networkingv1.ServiceBackendPort{
							Number: svc.Spec.Ports[0].Port, // TODO is there a better way to handle this? How to know which port to ingress?
						},
					},
				},
				// TODO implement TLS.
				// TODO do I need rules?
			},
		}
		err = r.Create(ctx, ing)
		if err != nil {
			cLog.Error(err, "Failed to create ingress for service request!", "namespace:", req.Namespace, "name:", req.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// For(&solov1alpha1.IngressOperator{}).
		For(&corev1.Service{}).
		Complete(r)
}
