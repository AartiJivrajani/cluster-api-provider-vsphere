/*
Copyright 2021 The Kubernetes Authors.

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

package builder

import (
	goctx "context"
	"crypto/tls"
	"fmt"
	"net"
	"path"
	"sync"
	"testing"
	"time"

	vmwarecontext "sigs.k8s.io/cluster-api-provider-vsphere/pkg/context/vmware"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vmwarev1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/vmware/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/test/remote"

	// nolint
	. "github.com/onsi/ginkgo"
	// nolint
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
)

var (
	boolTrue  = true
	separator = "\n---\n"
)

func init() {
	/*klog.InitFlags(nil)
	klog.SetOutput(GinkgoWriter)
	logf.SetLogger(klogr.New())*/
}

// TestSuite is used for unit and integration testing builder.
type TestSuite struct {
	goctx.Context
	addToManagerFn        manager.AddToManagerFunc
	certDir               string
	integrationTestClient client.Client
	config                *rest.Config
	done                  chan struct{}
	envTest               envtest.Environment
	flags                 TestFlags
	manager               manager.Manager
	newReconcilerFn       NewReconcilerFunc
	webhookName           string
	managerRunning        bool
	managerRunningMutex   sync.Mutex
	webhookYaml           []byte
}

func (s *TestSuite) isWebhookTest() bool {
	return s.webhookName != ""
}

func (s *TestSuite) GetEnvTestConfg() *rest.Config {
	return s.config
}

type Reconciler interface {
	ReconcileNormal(ctx *vmwarecontext.GuestClusterContext) (reconcile.Result, error)
}

// NewReconcilerFunc is a base type for functions that return a reconciler
type NewReconcilerFunc func() Reconciler

// NewTestSuiteForController returns a new test suite used for unit and
// integration testing controllers created using the "pkg/builder"
// package.
func NewTestSuiteForController(
	addToManagerFn manager.AddToManagerFunc,
	newReconcilerFn NewReconcilerFunc) *TestSuite {

	testSuite := &TestSuite{
		Context: goctx.Background(),
	}
	testSuite.init(addToManagerFn, newReconcilerFn)

	if testSuite.flags.UnitTestsEnabled {
		if newReconcilerFn == nil {
			panic("newReconcilerFn is nil")
		}
	}

	return testSuite
}

func (s *TestSuite) init(addToManagerFn manager.AddToManagerFunc, newReconcilerFn NewReconcilerFunc, additionalAPIServerFlags ...string) {
	// Initialize the test flags.
	s.flags = GetTestFlags()

	s.newReconcilerFn = newReconcilerFn
}

// Register should be invoked by the function to which *testing.T is passed.
//
// Use runUnitTestsFn to pass a function that will be invoked if unit testing
// is enabled with Describe("Unit tests", runUnitTestsFn).
//
// Use runIntegrationTestsFn to pass a function that will be invoked if
// integration testing is enabled with
// Describe("Unit tests", runIntegrationTestsFn).
func (s *TestSuite) Register(t *testing.T, name string, runIntegrationTestsFn, runUnitTestsFn func()) {
	RegisterFailHandler(Fail)

	if runIntegrationTestsFn == nil {
		s.flags.IntegrationTestsEnabled = false
	}
	if runUnitTestsFn == nil {
		s.flags.UnitTestsEnabled = false
	}

	if s.flags.IntegrationTestsEnabled {
		Describe("Integration tests", runIntegrationTestsFn)
	}
	if s.flags.UnitTestsEnabled {
		Describe("Unit tests", runUnitTestsFn)
	}

	if s.flags.IntegrationTestsEnabled {
		SetDefaultEventuallyTimeout(time.Second * 30)
		RunSpecsWithDefaultAndCustomReporters(t, name, []Reporter{printer.NewlineReporter{}})
	} else if s.flags.UnitTestsEnabled {
		RunSpecs(t, name)
	}
}

// NewUnitTestContextForController returns a new unit test context for this
// suite's reconciler.
//
// Returns nil if unit testing is disabled.
func (s *TestSuite) NewUnitTestContextForController(initObjects ...runtime.Object) *UnitTestContextForController {
	return s.NewUnitTestContextForControllerWithVSphereCluster(nil, false, initObjects...)
}

// NewUnitTestContextForControllerWithPrototypeCluster returns a new unit test
// context with a prototype cluster for this suite's reconciler. This prototype cluster
// helps controllers that do not wish to invoke the full TanzuKubernetesCluster
// spec reconciliation.
//
// Returns nil if unit testing is disabled.
func (s *TestSuite) NewUnitTestContextForControllerWithPrototypeCluster(initObjects ...runtime.Object) *UnitTestContextForController {
	return s.NewUnitTestContextForControllerWithVSphereCluster(nil, true, initObjects...)
}

// NewUnitTestContextForControllerWithVSphereCluster returns a new unit test context for this
// suite's reconciler initialized with the given vspherecluster.
//
// Returns nil if unit testing is disabled.
func (s *TestSuite) NewUnitTestContextForControllerWithVSphereCluster(vsphereCluster *vmwarev1.VSphereCluster, prototypeCluster bool, initObjects ...runtime.Object) *UnitTestContextForController {
	if s.flags.UnitTestsEnabled {
		ctx := NewUnitTestContextForController(s.newReconcilerFn, vsphereCluster, prototypeCluster, initObjects, nil)
		//SetTKRRefForVersion(ctx.Cluster, FakeDistributionVersion)
		reconcileNormalAndExpectSuccess(ctx)
		// Update the TanzuKubernetesCluster and its status in the fake client.
		Expect(ctx.Client.Update(ctx, ctx.VSphereCluster)).To(Succeed())
		Expect(ctx.Client.Status().Update(ctx, ctx.VSphereCluster)).To(Succeed())

		return ctx
	}
	return nil
}

func reconcileNormalAndExpectSuccess(ctx *UnitTestContextForController) {
	// Manually invoke the reconciliation. This is poor design, but in order
	// to support unit testing with a minimum set of dependencies that does
	// not include the Kubernetes envtest package, this is required.
	//
	Expect(ctx.ReconcileNormal()).ShouldNot(HaveOccurred())
}

// BeforeSuite should be invoked by ginkgo.BeforeSuite.
func (s *TestSuite) BeforeSuite() {
	if s.flags.IntegrationTestsEnabled {
		//s.beforeSuiteForIntegrationTesting()
	}
}

// AfterSuite should be invoked by ginkgo.AfterSuite.
func (s *TestSuite) AfterSuite() {
	if s.flags.IntegrationTestsEnabled {
		//s.afterSuiteForIntegrationTesting()
	}
}

// Create a new Manager with default values
func (s *TestSuite) createManager() {
	var err error

	s.done = make(chan struct{})
	s.manager, err = manager.New(manager.Options{
		KubeConfig:   s.config,
		AddToManager: s.addToManagerFn,
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(s.manager).ToNot(BeNil())
	s.integrationTestClient = s.manager.GetClient()
}

func (s *TestSuite) initializeManager() {
	// If one or more webhooks are being tested then go ahead and configure the
	// webhook server.
	if s.isWebhookTest() {
		By("configuring webhook server", func() {
			s.manager.GetWebhookServer().Host = "127.0.0.1"
			s.manager.GetWebhookServer().Port = randomTCPPort()
			s.manager.GetWebhookServer().CertDir = s.certDir
		})
	}
}

// Set a flag to indicate that the manager is running or not
func (s *TestSuite) setManagerRunning(isRunning bool) {
	s.managerRunningMutex.Lock()
	s.managerRunning = isRunning
	s.managerRunningMutex.Unlock()
}

// Returns true if the manager is running, false otherwise
func (s *TestSuite) getManagerRunning() bool {
	var result bool
	s.managerRunningMutex.Lock()
	result = s.managerRunning
	s.managerRunningMutex.Unlock()
	return result
}

// Starts the manager and sets managerRunning
func (s *TestSuite) startManager() {
	go func() {
		defer GinkgoRecover()

		s.setManagerRunning(true)
		ctx := goctx.TODO()
		Expect(s.manager.Start(ctx)).ToNot(HaveOccurred())
		s.setManagerRunning(false)
	}()
}

// Blocks until the manager has stopped running
// Removes state applied in postConfigureManager()
func (s *TestSuite) stopManager() {
	close(s.done)
	Eventually(func() bool { //nolint:gocritic
		return s.getManagerRunning()
	}).Should((BeFalse()))
	if s.webhookYaml != nil {
		Eventually(func() error {
			return remote.DeleteYAML(s, s.integrationTestClient, s.webhookYaml)
		}).Should(Succeed())
	}
}

// Applies configuration to the Manager after it has started
func (s *TestSuite) postConfigureManager() {
	// If there's a configured certificate directory then it means one or more
	// webhooks are being tested. Go ahead and install the webhooks and wait
	// for the webhook server to come online.
	if s.isWebhookTest() {
		By("installing the webhook(s)", func() {
			// ASSERT that the file for validating webhook file exists.
			validatingWebhookFile := path.Join(s.flags.RootDir, "config", "webhook", "manifests.yaml")
			Expect(validatingWebhookFile).Should(BeAnExistingFile())

			// ASSERT that eventually the webhook config gets successfully
			// applied to the API server.
			Eventually(func() error {
				return remote.ApplyYAML(s, s.integrationTestClient, s.webhookYaml)
			}).Should(Succeed())
		})

		// It can take a few seconds for the webhook server to come online.
		// This step blocks until the webserver can be successfully accessed.
		By("waiting for the webhook server to come online", func() {
			addr := fmt.Sprintf("%s:%d",
				s.manager.GetWebhookServer().Host,
				s.manager.GetWebhookServer().Port)
			dialer := &net.Dialer{Timeout: time.Second}
			//nolint:gosec
			tlsConfig := &tls.Config{InsecureSkipVerify: true}
			Eventually(func() error {
				conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
				if err != nil {
					return err
				}
				conn.Close()
				return nil
			}).Should(Succeed())
		})
	}
}

func (s *TestSuite) afterSuiteForIntegrationTesting() {
	By("tearing down the test environment", func() {
		close(s.done)
		Expect(s.envTest.Stop()).To(Succeed())
	})
}
