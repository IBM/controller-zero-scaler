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
	//"k8s.io/api/autoscaling/v1"
	//v1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"testing"
)

func TestScale(t *testing.T) {
	cfg := config.GetConfigOrDie()
	cl := kubernetes.NewForConfigOrDie(cfg)

	deployment, err := cl.Apps().Deployments("default").Get("nginx-deployment", metav1.GetOptions{})
	t.Log(err)
	zero := int32(0)
	deployment.Spec.Replicas = &zero

	deployment, err = cl.Apps().Deployments("default").Update(deployment)
	t.Log(err)
}
