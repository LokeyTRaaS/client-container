package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	lokeyv1 "github.com/lokey/client-container/api/v1"
	"github.com/lokey/client-container/pkg/operator/reconciler"
)

// LokeyClientReconciler reconciles a LokeyClient object
type LokeyClientReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=lokey.io,resources=lokeyclients,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=lokey.io,resources=lokeyclients/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=lokey.io,resources=lokeyclients/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;patch
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *LokeyClientReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var lokeyClient lokeyv1.LokeyClient
	if err := r.Get(ctx, req.NamespacedName, &lokeyClient); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Update status
	lokeyClient.Status.LastSyncTime = metav1.Now()

	// Helper to check if a bool pointer is true
	enableDaemonset := lokeyClient.Spec.EnableDaemonset != nil && *lokeyClient.Spec.EnableDaemonset
	enableSidecar := lokeyClient.Spec.EnableSidecar != nil && *lokeyClient.Spec.EnableSidecar

	// Handle sidecar injection
	if enableSidecar {
		// Count and inject sidecars into matching pods
		podList := &corev1.PodList{}
		if err := r.List(ctx, podList); err == nil {
			injectedCount := 0
			for i := range podList.Items {
				pod := &podList.Items[i]

				// Skip if already injected
				if pod.Annotations != nil && pod.Annotations["lokey.io/injected"] == "true" {
					injectedCount++
					continue
				}

				// Get namespace labels
				var nsLabels map[string]string
				ns := &corev1.Namespace{}
				if err := r.Get(ctx, client.ObjectKey{Name: pod.Namespace}, ns); err == nil {
					nsLabels = ns.Labels
				}

				// Check if pod should be injected
				if reconciler.ShouldInjectSidecar(pod, nsLabels, lokeyClient.Spec.NamespaceSelector, lokeyClient.Spec.PodSelector) {
					enableSeccomp := lokeyClient.Spec.EnableSeccomp != nil && *lokeyClient.Spec.EnableSeccomp
					sidecarSpec := &reconciler.SidecarSpec{
						VirtIOURL:         lokeyClient.Spec.VirtIOURL,
						DevicePaths:       lokeyClient.Spec.DevicePaths,
						ChunkSize:         lokeyClient.Spec.ChunkSize,
						ReconnectInterval: lokeyClient.Spec.ReconnectInterval,
						Image:             lokeyClient.Spec.Image,
						InitImage:         lokeyClient.Spec.InitImage,
						Resources:         lokeyClient.Spec.Resources,
						EnableSeccomp:     enableSeccomp,
					}

					// Create a copy for patching
					podCopy := pod.DeepCopy()
					if err := reconciler.InjectSidecar(podCopy, sidecarSpec); err != nil {
						logger.Error(err, "failed to inject sidecar", "pod", pod.Name, "namespace", pod.Namespace)
						continue
					}

					// Patch the pod
					if err := r.Patch(ctx, podCopy, client.MergeFrom(pod)); err != nil {
						logger.Error(err, "failed to patch pod with sidecar", "pod", pod.Name, "namespace", pod.Namespace)
						continue
					}

					logger.Info("injected sidecar into pod", "pod", pod.Name, "namespace", pod.Namespace)
					injectedCount++
				}
			}
			lokeyClient.Status.InjectedPods = injectedCount
		} else {
			logger.Error(err, "failed to list pods for sidecar injection")
		}
	} else {
		// Sidecar injection disabled, reset count
		lokeyClient.Status.InjectedPods = 0
	}

	// Handle daemonset
	daemonsetName := fmt.Sprintf("lokey-client-%s", lokeyClient.Name)
	if enableDaemonset {
		enableSeccomp := lokeyClient.Spec.EnableSeccomp != nil && *lokeyClient.Spec.EnableSeccomp
		daemonsetSpec := &reconciler.DaemonsetSpec{
			VirtIOURL:         lokeyClient.Spec.VirtIOURL,
			DevicePaths:       lokeyClient.Spec.DevicePaths,
			ChunkSize:         lokeyClient.Spec.ChunkSize,
			ReconnectInterval: lokeyClient.Spec.ReconnectInterval,
			Image:             lokeyClient.Spec.Image,
			Resources:         lokeyClient.Spec.Resources,
			EnableSeccomp:     enableSeccomp,
		}

		daemonset := reconciler.CreateDaemonset(daemonsetName, "lokey-system", daemonsetSpec)

		// Apply daemonset
		if err := ctrl.SetControllerReference(&lokeyClient, daemonset, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		var existingDaemonset appsv1.DaemonSet
		if err := r.Get(ctx, client.ObjectKeyFromObject(daemonset), &existingDaemonset); err != nil {
			if client.IgnoreNotFound(err) == nil {
				// Create
				if err := r.Create(ctx, daemonset); err != nil {
					logger.Error(err, "failed to create daemonset")
					return ctrl.Result{}, err
				}
				logger.Info("created daemonset", "name", daemonsetName)
				lokeyClient.Status.DaemonsetStatus = "Created"
			} else {
				return ctrl.Result{}, err
			}
		} else {
			// Update
			existingDaemonset.Spec = daemonset.Spec
			if err := r.Update(ctx, &existingDaemonset); err != nil {
				logger.Error(err, "failed to update daemonset")
				return ctrl.Result{}, err
			}
			logger.Info("updated daemonset", "name", daemonsetName)
			lokeyClient.Status.DaemonsetStatus = "Running"
		}
	} else {
		// Daemonset disabled, delete if exists
		daemonset := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      daemonsetName,
				Namespace: "lokey-system",
			},
		}
		if err := r.Get(ctx, client.ObjectKeyFromObject(daemonset), daemonset); err == nil {
			if err := r.Delete(ctx, daemonset); err != nil {
				logger.Error(err, "failed to delete daemonset")
				return ctrl.Result{}, err
			}
			logger.Info("deleted daemonset", "name", daemonsetName)
		}
		lokeyClient.Status.DaemonsetStatus = "Disabled"
	}

	// Update status
	if err := r.Status().Update(ctx, &lokeyClient); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LokeyClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&lokeyv1.LokeyClient{}).
		Owns(&appsv1.DaemonSet{}).
		Complete(r)
}
