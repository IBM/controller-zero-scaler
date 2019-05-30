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
	//"context"
	//"fmt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sync"
	"time"
)

type Manager struct {
	lock sync.Mutex

	//Map of controller deployments to scalers
	scalers map[types.NamespacedName]*AutoScaler
}

func NewManager() *Manager {
	return &Manager{scalers: make(map[types.NamespacedName]*AutoScaler)}
}

func (s *Manager) CreateRegistration(controller types.NamespacedName, d time.Duration, watchedKinds []schema.GroupVersionKind, ownedKinds []schema.GroupVersionKind, cfg *rest.Config) {
	as := NewAutoScaler(controller, cfg, d, watchedKinds, ownedKinds)
	s.scalers[controller] = as
	as.Start()
}

func (s *Manager) Cancel(controller types.NamespacedName) {
	if as, exists := s.scalers[controller]; exists {
		as.Cancel()
		delete(s.scalers, controller)
	}
}
