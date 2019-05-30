/*

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
	"encoding/json"
	"fmt"
	"hash/adler32"
	"strings"
	"time"

	"github.com/ibm/controller-zero-scaler/pkg/scaler"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller")

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDeployment{Client: mgr.GetClient(), scheme: mgr.GetScheme(), ScaleManager: scaler.NewManager(), checksums: make(map[types.NamespacedName]uint32)}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("controller-zero-scaler-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileDeployment{}

type ReconcileDeployment struct {
	client.Client
	scheme       *runtime.Scheme
	ScaleManager *scaler.Manager
	checksums    map[types.NamespacedName]uint32
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets/status,verbs=get;update;patch
func (r *ReconcileDeployment) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	cfg := config.GetConfigOrDie()
	instance := &appsv1.StatefulSet{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	annotations := instance.GetAnnotations()
	if _, exists := annotations[ANNOTATION_TIMEOUT]; exists {
		//When the timeout exists, is enabled for scale to zero

		b := []byte(annotations[ANNOTATION_TIMEOUT] + annotations[ANNOTATION_WATCHED_KINDS] + annotations[ANNOTATION_OWNED_KINDS])
		sum := adler32.Checksum(b)

		if currentSum, exists := r.checksums[request.NamespacedName]; !exists {
			//Create registration
			d, watchedKinds, ownedKinds, err := parseAnnotations(annotations)

			fmt.Println("Annotations", watchedKinds, ownedKinds, err)

			r.ScaleManager.CreateRegistration(request.NamespacedName, d, watchedKinds, ownedKinds, cfg)

			r.checksums[request.NamespacedName] = sum
		} else if exists && currentSum != sum {
			//Update registration

			r.ScaleManager.Cancel(request.NamespacedName)

			d, watchedKinds, ownedKinds, err := parseAnnotations(annotations)

			fmt.Println("Annotations", watchedKinds, ownedKinds, err)

			r.ScaleManager.CreateRegistration(request.NamespacedName, d, watchedKinds, ownedKinds, cfg)

			r.checksums[request.NamespacedName] = sum
		} else {
			// Current registration is valid
		}
	}

	return reconcile.Result{}, nil
}

func parseAnnotations(annotations map[string]string) (time.Duration, []schema.GroupVersionKind, []schema.GroupVersionKind, error) {
	var d time.Duration
	if _, exists := annotations[ANNOTATION_TIMEOUT]; exists {
		var err error
		d, err = time.ParseDuration(annotations[ANNOTATION_TIMEOUT])
		if err != nil {
			return time.Second, nil, nil, err
		}
	}

	var watchedGVKs []schema.GroupVersionKind
	if _, exists := annotations[ANNOTATION_WATCHED_KINDS]; exists {
		var watchedKinds []interface{}
		err := json.Unmarshal([]byte(annotations[ANNOTATION_WATCHED_KINDS]), &watchedKinds)
		if err != nil {
			return time.Second, nil, nil, err
		}
		for _, watchedKind := range watchedKinds {
			_watchedKind := watchedKind.(map[string]interface{})
			group := strings.Split(_watchedKind["apiVersion"].(string), "/")[0]
			version := strings.Split(_watchedKind["apiVersion"].(string), "/")[1]
			kind := _watchedKind["Kind"].(string)
			gvk := schema.GroupVersionKind{Group: group, Version: version, Kind: kind}
			watchedGVKs = append(watchedGVKs, gvk)
		}
	}

	var ownedGVKs []schema.GroupVersionKind
	if _, exists := annotations[ANNOTATION_OWNED_KINDS]; exists {
		var ownedKinds []interface{}
		err := json.Unmarshal([]byte(annotations[ANNOTATION_OWNED_KINDS]), &ownedKinds)
		if err != nil {
			return time.Second, nil, nil, err
		}

		for _, ownedKind := range ownedKinds {
			_ownedKind := ownedKind.(map[string]interface{})
			group := strings.Split(_ownedKind["apiVersion"].(string), "/")[0]
			version := strings.Split(_ownedKind["apiVersion"].(string), "/")[1]
			kind := _ownedKind["Kind"].(string)
			gvk := schema.GroupVersionKind{Group: group, Version: version, Kind: kind}
			ownedGVKs = append(ownedGVKs, gvk)
		}
	}

	return d, watchedGVKs, ownedGVKs, nil
}
