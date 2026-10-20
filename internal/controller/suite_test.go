//go:build !windows

// envtest is unsupported on Windows with controller-runtime v0.25.x
// (signal_windows.go defines signalProcess instead of signalProcessImpl,
// so sigs.k8s.io/controller-runtime/pkg/envtest does not compile there).
// Run these integration tests on Linux/macOS or in CI.

package controller

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	databasesv1alpha1 "github.com/vaibhavmahiwal/mini_everest/api/v1alpha1"
)

var (
	testEnv    *envtest.Environment
	testCfg    *rest.Config
	testClient client.Client
	testScheme = runtime.NewScheme()
)

func TestMain(m *testing.M) {
	log.SetLogger(zap.New(zap.WriteTo(os.Stdout), zap.UseDevMode(true)))

	if err := clientgoscheme.AddToScheme(testScheme); err != nil {
		panic(err)
	}
	if err := databasesv1alpha1.AddToScheme(testScheme); err != nil {
		panic(err)
	}

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	testCfg, err = testEnv.Start()
	if err != nil {
		panic(err)
	}
	defer func() {
		if stopErr := testEnv.Stop(); stopErr != nil {
			panic(stopErr)
		}
	}()

	testClient, err = client.New(testCfg, client.Options{Scheme: testScheme})
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

// newReconciler returns a reconciler wired to the envtest API server.
func newReconciler() *DatabaseClusterReconciler {
	return &DatabaseClusterReconciler{
		Client: testClient,
		Scheme: testScheme,
	}
}
