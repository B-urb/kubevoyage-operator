/*
Copyright 2024.

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
	"github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	v1 "k8s.io/api/networking/v1"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IngressHandlingReconciler reconciles a IngressHandling object
type IngressHandlingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=networking.burban.me,resources=ingresshandlings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.burban.me,resources=ingresshandlings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=networking.burban.me,resources=ingresshandlings/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch
//+kubebuilder:rbac:groups=traefik.containo.us,resources=middlewares,verbs=get;list;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IngressHandling object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *IngressHandlingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var ingress v1.Ingress
	if err := r.Get(ctx, req.NamespacedName, &ingress); err != nil {
		slog.Error("unable to fetch Ingress", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if _, exists := ingress.Annotations["kubevoyage-auth"]; exists {

		forwardAuth := &v1alpha1.ForwardAuth{
			Address:             "https://voyage.burban.me/api/authenticate?redirect=https://dev.burban.me",
			TrustForwardHeader:  true,
			AuthResponseHeaders: []string{"Location", "X-Auth-Site", "X-Auth-Token", "Cookie"},
		}
		middleware := &v1alpha1.Middleware{
			Spec: v1alpha1.MiddlewareSpec{
				ForwardAuth: forwardAuth,
			},
		}

		if err := r.Get(ctx, client.ObjectKeyFromObject(middleware), middleware); err != nil {
			if client.IgnoreNotFound(err) == nil {
				if err := r.Create(ctx, middleware); err != nil {
					slog.Error("failed to create Middleware", err)
					return ctrl.Result{}, err
				}
			} else {
				slog.Error("failed to fetch Middleware", err)
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *IngressHandlingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Ingress{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
			ingress, ok := object.(*v1.Ingress)
			if !ok {
				return false
			}
			annotation, exists := ingress.Annotations["kubevoyage-auth"]
			return exists && annotation == "true"
		}))).
		Complete(r)
}
