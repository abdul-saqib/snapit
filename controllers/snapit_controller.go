package controllers

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/google/uuid"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	clientset "github.com/abdul-saqib/snapit/pkg/generated/clientset/versioned"
	informers "github.com/abdul-saqib/snapit/pkg/generated/informers/externalversions"
	listers "github.com/abdul-saqib/snapit/pkg/generated/listers/snapitcontroller/v1alpha1"
	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	snapshotclient "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
)

type Controller struct {
	kubeclient     kubernetes.Interface
	clientset      clientset.Interface
	snapshotClient snapshotclient.Interface
	policyLister   listers.SnapshotPolicyLister
	policyInformer cache.SharedIndexInformer
	queue          workqueue.TypedRateLimitingInterface[string]
}

func NewController(kubeclient kubernetes.Interface, clientset clientset.Interface, snapshotClient snapshotclient.Interface, factory informers.SharedInformerFactory) *Controller {
	informer := factory.Snapitcontroller().V1alpha1().SnapshotPolicies()
	workerqueue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())

	ctrl := &Controller{
		kubeclient:     kubeclient,
		clientset:      clientset,
		snapshotClient: snapshotClient,
		policyLister:   informer.Lister(),
		policyInformer: informer.Informer(),
		queue:          workerqueue,
	}

	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			key, _ := cache.MetaNamespaceKeyFunc(obj)
			ctrl.queue.Add(key)
		},
		UpdateFunc: func(_, newObj any) {
			key, _ := cache.MetaNamespaceKeyFunc(newObj)
			ctrl.queue.Add(key)
		},
	})

	return ctrl
}

func (c *Controller) Run(ctx context.Context, workers int) error {
	klog.Info("Starting SnapshotPolicy Controller...")

	defer runtime.HandleCrash()
	defer c.queue.ShutDown()

	go c.policyInformer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), c.policyInformer.HasSynced) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}
	klog.Info("starting worker")
	<-ctx.Done()
	klog.Info("shutting down worker")

	return nil
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextItem(ctx) {
	}
}

func (c *Controller) processNextItem(ctx context.Context) bool {
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(key)

	if err := c.syncHandler(ctx, key); err != nil {
		klog.Errorf("error syncing %s: %v", key, err)
		c.queue.AddRateLimited(key)
	}
	return true
}

func (c *Controller) syncHandler(ctx context.Context, key string) error {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	policy, err := c.policyLister.SnapshotPolicies(ns).Get(name)
	if err != nil {
		return err
	}
	now := time.Now()

	pvc, err := c.kubeclient.CoreV1().PersistentVolumeClaims(ns).Get(ctx, policy.Spec.PVCName, metav1.GetOptions{})
	if err != nil {
		// PVC does not exist
		klog.Errorf("PVC %s/%s not found: %v", ns, policy.Spec.PVCName, err)
		return nil
	}

	// check if PVC is bound
	if pvc.Status.Phase != corev1.ClaimBound {
		klog.Infof("PVC %s/%s is not bound yet (status: %s), skipping snapshot", ns, policy.Spec.PVCName, pvc.Status.Phase)
		return nil
	}

	if policy.Spec.Schedule == "" {
		// immediate snapshot
		if policy.Status.LastProcessedRequestID != "" {
			// already processed - go idle silently
			return nil
		}
	} else {
		// periodic snapshot
		interval, err := time.ParseDuration(policy.Spec.Schedule)
		if err != nil {
			return fmt.Errorf("invalid schedule for %s/%s: %v", ns, name, err)
		}

		if policy.Status.LastSnapshotTime != nil && now.Sub(policy.Status.LastSnapshotTime.Time) < interval {
			return nil
		}
	}

	// ----------------- Create snapshot -----------------
	requestID := uuid.NewString()
	snapName := fmt.Sprintf("%s-%s", name, requestID[:8])

	vs := &snapshotv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      snapName,
			Namespace: ns,
			Labels: map[string]string{
				"snapitpolicy": name,
			},
		},
		Spec: snapshotv1.VolumeSnapshotSpec{
			Source: snapshotv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: &policy.Spec.PVCName,
			},
			VolumeSnapshotClassName: &policy.Spec.SnapshotClassName,
		},
	}

	if _, err := c.snapshotClient.SnapshotV1().VolumeSnapshots(ns).Create(ctx, vs, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create snapshot: %v", err)
	}

	// ----------------- enforce retention -----------------
	if policy.Spec.Retention != nil {
		_ = c.enforceRetention(ctx, ns, name, *policy.Spec.Retention)
	}

	klog.Infof("Successfully created snapshot %s for %s/%s", snapName, ns, name)

	// ----------------- update status -----------------
	newStatus := policy.DeepCopy()
	newStatus.Status.LastSnapshotTime = &metav1.Time{Time: now}
	newStatus.Status.LastProcessedRequestID = requestID
	newStatus.Status.Phase = "SnapshotCreated"
	newStatus.Status.Message = fmt.Sprintf("Snapshot %s created", snapName)

	if !reflect.DeepEqual(policy.Status, newStatus.Status) {
		_, err := c.clientset.SnapitcontrollerV1alpha1().
			SnapshotPolicies(ns).
			UpdateStatus(ctx, newStatus, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update status: %v", err)
		}
	}

	return nil
}

func (c *Controller) enforceRetention(ctx context.Context, namespace, policyName string, retention int) error {
	snaps, err := c.snapshotClient.SnapshotV1().VolumeSnapshots(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	var owned []snapshotv1.VolumeSnapshot
	for _, snap := range snaps.Items {
		if snap.Labels["snapitpolicy"] == policyName {
			owned = append(owned, snap)
		}
	}

	if len(owned) <= retention {
		return nil
	}

	sort.Slice(owned, func(i, j int) bool {
		return owned[i].CreationTimestamp.Before(&owned[j].CreationTimestamp)
	})

	// delete the oldest snapshot
	toDelete := owned[0 : len(owned)-retention]
	for _, snap := range toDelete {
		err := c.snapshotClient.SnapshotV1().VolumeSnapshots(namespace).Delete(ctx, snap.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}
