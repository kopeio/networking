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
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/addon"
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/addon/pkg/status"
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/declarative"
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/declarative/pkg/manifest"

	addonsv1alpha1 "kope.io/networking/operator/api/v1alpha1"
)

var _ reconcile.Reconciler = &NetworkingReconciler{}

// NetworkingReconciler reconciles a Networking object
type NetworkingReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	RBACMode string

	declarative.Reconciler
}

//+kubebuilder:rbac:groups=addons.kope.io,resources=networkings,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=addons.kope.io,resources=networkings/status,verbs=get;update;patch

// Generated with: rbac-gen --yaml channels/packages/networking/1.0.20210815/manifest.yaml --format kubebuilder --supervisory --limit-namespaces --limit-resource-names

//+kubebuilder:rbac:groups=apps,namespace=kopeio-networking-system,resources=daemonsets,verbs=create;get;list;watch
//+kubebuilder:rbac:groups=apps,namespace=kopeio-networking-system,resources=daemonsets,resourceNames=kopeio-networking-agent,verbs=delete;patch;update

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	addon.Init()

	labels := map[string]string{
		"k8s-app": "networking",
	}

	rbacModeTransform := func(ctx context.Context, object declarative.DeclarativeObject, objects *manifest.Objects) error {
		switch r.RBACMode {
		case "reconcile", "":
			return nil

		case "ignore":
			var keepItems []*manifest.Object
			for _, obj := range objects.Items {
				keep := true
				if obj.GroupKind().Group == "rbac.authorization.k8s.io" {
					switch obj.GroupKind().Kind {
					case "ClusterRole", "ClusterRoleBinding", "Role", "RoleBinding":
						keep = false
					}
				}
				if keep {
					keepItems = append(keepItems, obj)
				}
			}
			objects.Items = keepItems
			return nil

		default:
			return fmt.Errorf("unknown rbac mode %q", r.RBACMode)
		}
	}

	watchLabels := declarative.SourceLabel(mgr.GetScheme())

	if err := r.Reconciler.Init(mgr, &addonsv1alpha1.Networking{},
		declarative.WithObjectTransform(rbacModeTransform),
		declarative.WithObjectTransform(declarative.AddLabels(labels)),
		declarative.WithOwner(declarative.SourceAsOwner),
		declarative.WithLabels(watchLabels),
		declarative.WithStatus(status.NewBasic(mgr.GetClient())),
		// TODO: add an application to your manifest:  declarative.WithObjectTransform(addon.TransformApplicationFromStatus),
		// TODO: add an application to your manifest:  declarative.WithManagedApplication(watchLabels),
		declarative.WithObjectTransform(addon.ApplyPatches),
	); err != nil {
		return err
	}

	c, err := controller.New("networking-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Networking
	err = c.Watch(&source.Kind{Type: &addonsv1alpha1.Networking{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to deployed objects
	_, err = declarative.WatchChildren(declarative.WatchChildrenOptions{
		Manager:                 mgr,
		Controller:              c,
		Reconciler:              r,
		LabelMaker:              watchLabels,
		ScopeWatchesToNamespace: true,
	})
	if err != nil {
		return err
	}

	return nil
}
