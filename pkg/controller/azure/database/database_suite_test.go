/*
Copyright 2018 The Conductor Authors.

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

package database

import (
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/upbound/conductor/pkg/apis/azure"
	databasev1alpha1 "github.com/upbound/conductor/pkg/apis/azure/database/v1alpha1"
	azurev1alpha1 "github.com/upbound/conductor/pkg/apis/azure/v1alpha1"
	"github.com/upbound/conductor/pkg/test"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	timeout       = 5 * time.Second
	namespace     = "test-db-namespace"
	instanceName  = "test-db-instance"
	secretName    = "test-secret"
	secretDataKey = "credentials"
	providerName  = "test-provider"
)

var (
	ctx             = context.TODO()
	cfg             *rest.Config
	expectedRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: instanceName, Namespace: namespace}}
)

func TestMain(m *testing.M) {
	azure.AddToScheme(scheme.Scheme)

	t := test.NewTestEnv(namespace, test.CRDs())
	cfg = t.Start()
	t.StopAndExit(m.Run())
}

// SetupTestReconcile returns a reconcile.Reconcile implementation that delegates to inner and
// writes the request to requests after Reconcile is finished.
func SetupTestReconcile(inner reconcile.Reconciler) (reconcile.Reconciler, chan reconcile.Request) {
	requests := make(chan reconcile.Request)
	fn := reconcile.Func(func(req reconcile.Request) (reconcile.Result, error) {
		result, err := inner.Reconcile(req)
		requests <- req
		return result, err
	})
	return fn, requests
}

// StartTestManager adds recFn
func StartTestManager(mgr manager.Manager, g *gomega.GomegaWithT) chan struct{} {
	stop := make(chan struct{})
	go func() {
		g.Expect(mgr.Start(stop)).NotTo(gomega.HaveOccurred())
	}()
	return stop
}

func testSecret(data []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			secretDataKey: data,
		},
	}
}

func testProvider(s *corev1.Secret) *azurev1alpha1.Provider {
	return &azurev1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      providerName,
			Namespace: s.Namespace,
		},
		Spec: azurev1alpha1.ProviderSpec{
			AuthSecret: corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				Key:                  secretDataKey,
			},
		},
	}
}

func testInstance(p *azurev1alpha1.Provider) *databasev1alpha1.MysqlServer {
	return &databasev1alpha1.MysqlServer{
		ObjectMeta: metav1.ObjectMeta{Name: instanceName, Namespace: namespace},
		Spec: databasev1alpha1.MysqlServerSpec{
			ProviderRef:    corev1.LocalObjectReference{Name: p.Name},
			AdminLoginName: "myadmin",
			PricingTier: databasev1alpha1.PricingTierSpec{
				Tier: "Basic", VCores: 1, Family: "Gen4",
			},
		},
	}
}
