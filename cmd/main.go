package main

import (
	"flag"
	"path/filepath"
	"time"

	"github.com/abdul-saqib/snapit/controllers"
	"github.com/abdul-saqib/snapit/pkg/signals"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	clientset "github.com/abdul-saqib/snapit/pkg/generated/clientset/versioned"
	informers "github.com/abdul-saqib/snapit/pkg/generated/informers/externalversions"
	snapshotclient "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
)

func main() {
	klog.InitFlags(nil)

	var configPath string
	flag.StringVar(&configPath, "configpath", "", "Path to kubeconfig")

	var masterURL string
	flag.StringVar(&masterURL, "master", "", "API server address")

	flag.Parse()

	cfg, err := readConfig(configPath, masterURL)
	if err != nil {
		klog.Fatalf("error reading config: %v", err)
	}

	kubeclient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("error creating kube client: %v", err)
	}

	snapClientSet, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("error creating snapclient: %v", err)
	}

	snapShotClient, err := snapshotclient.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("error creating snapShotclient: %v", err)
	}

	ctx := signals.SetupSignalHandler()
	factory := informers.NewSharedInformerFactory(snapClientSet, time.Second*30)

	ctrl := controllers.NewController(kubeclient, snapClientSet, snapShotClient, factory)
	if err := ctrl.Run(ctx, 3); err != nil {
		klog.Error(err, "error running controller")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
}

func readConfig(configPath, masterURL string) (*rest.Config, error) {
	if configPath != "" {
		klog.Infof("using kube config: %s", configPath)
		return clientcmd.BuildConfigFromFlags(masterURL, filepath.Clean(configPath))
	}
	klog.Info("using cluster config")
	return rest.InClusterConfig()
}
