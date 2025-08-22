package host

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/common"
)

// During factory reconfig, the reconciled status is expected to be updated to
// false to unblock the configuration as the day 1 configuration.
// updateStatusForFactoryInstall updates the reconciler status based
// on the factory install config map.
func (r *HostReconciler) updateStatusForFactoryInstall(ctx context.Context, obj client.Object) []reconcile.Request {
	IgnoreReconcile := []reconcile.Request{}
	namespace := r.GetNamespace()
	if _, ok := common.FactoryReconfigAllowed(namespace, obj); !ok {
		return IgnoreReconcile
	}

	log := logHost.WithName("enrollment")
	log.Info("starting config update")

	hosts := &starlingxv1.HostList{}
	opts := client.ListOptions{Namespace: namespace}

	if err := r.Client.List(context.TODO(), hosts, &opts); err != nil {
		log.Error(err, "failed to retrieve the resources")
		return IgnoreReconcile
	}

	if len(hosts.Items) == 0 {
		log.Info("not hosts found")
		return IgnoreReconcile
	}

	for _, host := range hosts.Items {
		host.Status.Reconciled = false
		// set defaults back to nil to make sure that the host will be re-inventoried after
		// enrollment.
		host.Status.Defaults = nil
		log.Info(fmt.Sprintf("updating status of %s for factory install", host.Name))
		if err := r.Client.Status().Update(context.TODO(), &host); err != nil {
			log.Error(err, "failed to update host status")
			return IgnoreReconcile
		}
	}

	namespacedName := types.NamespacedName{Namespace: namespace, Name: "hosts"}
	return []reconcile.Request{{NamespacedName: namespacedName}}
}

// setFactoryReconfigAsFinalized updates the factory config map, marking the factory config as
// finalized.
func (r *HostReconciler) setFactoryReconfigAsFinalized() error {
	configMap := &v1.ConfigMap{}
	configMapName := client.ObjectKey{
		Namespace: r.GetNamespace(),
		Name:      common.FactoryInstallConfigMapName,
	}
	if err := r.Client.Get(context.TODO(), configMapName, configMap); err != nil {
		return err
	}

	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	configMap.Data[common.FactoryConfigFinalized] = "true"
	return r.Client.Update(context.TODO(), configMap)
}
