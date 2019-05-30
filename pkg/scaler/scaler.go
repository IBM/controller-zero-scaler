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

package scaler

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"log"
	"reflect"
	"strings"
	"time"
)

type AutoScaler struct {
	cl                       *kubernetes.Clientset
	factory                  dynamicinformer.DynamicSharedInformerFactory
	controller               types.NamespacedName
	timeoutDuration          time.Duration
	stopC                    chan struct{}
	watchedGVKs              []schema.GroupVersionKind
	ownedGVKs                []schema.GroupVersionKind
	lastNotification         time.Time
	events                   chan interface{}
	observedResourceVersions map[types.NamespacedName]string
}

func NewAutoScaler(controller types.NamespacedName, cfg *rest.Config, d time.Duration, watchedKinds []schema.GroupVersionKind, ownedKinds []schema.GroupVersionKind) *AutoScaler {
	kcl := kubernetes.NewForConfigOrDie(cfg)
	dcl := dynamic.NewForConfigOrDie(cfg)
	s := &AutoScaler{
		cl:              kcl,
		controller:      controller,
		factory:         dynamicinformer.NewDynamicSharedInformerFactory(dcl, 0),
		timeoutDuration: d,
		stopC:           make(chan struct{}),
		watchedGVKs:     watchedKinds,
		ownedGVKs:       ownedKinds,
		events:          make(chan interface{}),
	}

	s.factory.Start(s.stopC)
	return s
}

func (s *AutoScaler) Start() {
	//Start the informers
	go func() {
		for _, watched := range s.watchedGVKs {
			plural := strings.ToLower(watched.Kind) + "s"
			gvr := schema.GroupVersionResource{Group: watched.Group, Version: watched.Version, Resource: plural}

			informerLister := s.factory.ForResource(gvr)
			informer := informerLister.Informer()
			informer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
				AddFunc: func(added interface{}) {
					log.Println("Add")
					s.lastNotification = time.Now()
					s.events <- new(struct{})
				},
				UpdateFunc: func(old, updated interface{}) {
					_old := old.(*unstructured.Unstructured)
					_updated := updated.(*unstructured.Unstructured)

					// Compare the labels, annotations, and generation to determine if an
					// update has been made
					if !reflect.DeepEqual(_old.GetAnnotations(), _updated.GetAnnotations()) ||
						!reflect.DeepEqual(_old.GetLabels(), _updated.GetLabels()) ||
						_old.GetGeneration() != _updated.GetGeneration() {
						log.Println("Update")

						s.lastNotification = time.Now()
						s.events <- new(struct{})
					}

				},
				DeleteFunc: func(deleted interface{}) {
					s.lastNotification = time.Now()
					s.events <- new(struct{})
				},
			})
		}

		s.factory.Start(s.stopC)
	}()

	//Watch for events from the informer or timeout
	go func() {
		for {
			select {
			case <-s.events:
				err := s.scale_up()
				if err != nil {
					fmt.Println("Could not scale up", err)
				}
			case <-time.After(s.timeoutDuration):
				err := s.scale_down()
				if err != nil {
					fmt.Println("Could not scale down", err)
				}
			case <-s.stopC:
				//This Scaler was canceled
				return
			}
		}
	}()
}

func (s *AutoScaler) Cancel() {
	close(s.stopC)
}

func (s *AutoScaler) scale_down() error {
	ss, err := s.cl.Apps().StatefulSets(s.controller.Namespace).Get(s.controller.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	desiredReplicas := int32(0)
	if *ss.Spec.Replicas != desiredReplicas {
		fmt.Println("Scaling down")
		ss.Spec.Replicas = &desiredReplicas

		ss, err = s.cl.Apps().StatefulSets(s.controller.Namespace).Update(ss)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *AutoScaler) scale_up() error {
	ss, err := s.cl.Apps().StatefulSets(s.controller.Namespace).Get(s.controller.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	desiredReplicas := int32(1)

	if *ss.Spec.Replicas != desiredReplicas {
		fmt.Println("Scaling up")
		ss.Spec.Replicas = &desiredReplicas

		ss, err = s.cl.Apps().StatefulSets(s.controller.Namespace).Update(ss)
		if err != nil {
			return err
		}
	}

	return nil
}
