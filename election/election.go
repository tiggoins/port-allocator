package election

import (
	"context"
	"log"
	"os"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	"github.com/tiggoins/port-allocator/k8s"
)

func Election(client *kubernetes.Clientset) {
	callbacks := leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			klog.V(2).InfoS("I am the new leader now.")
		},
		OnStoppedLeading: func() {
			klog.V(2).InfoS("I am not the leader anymore.")
		},
		OnNewLeader: func(identity string) {
			klog.InfoS("New leader elected", "identity", identity)
		},
	}

	broadcaster := record.NewBroadcaster()
	hostname, _ := os.Hostname()

	recorder := broadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{
		Component: "range-based-port-allocator",
		Host:      hostname,
	})

	// 创建 LeaderElection 配置
	leaderElectionConfig := leaderelection.LeaderElectionConfig{
		Lock: &resourcelock.LeaseLock{
			LeaseMeta: metav1.ObjectMeta{
				Namespace: k8s.PodDetails.Namespace,
				Name:      "range-based-port-allocator",
			},
			Client: client.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{
				Identity:      k8s.PodDetails.Name,
				EventRecorder: recorder,
			},
		},
		ReleaseOnCancel: true,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks:       callbacks,
	}

	// 创建 LeaderElector
	leaderElector, err := leaderelection.NewLeaderElector(leaderElectionConfig)
	if err != nil {
		log.Fatalf("Error creating leader elector: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 开始 LeaderElection
	go func() {
		leaderElector.Run(ctx)
	}()
}
