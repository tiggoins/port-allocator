package controller

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"github.com/tiggoins/port-allocator/store"
)

// EventType type of event associated with an informer
type EventType string

const (
	// DeleteEvent event associated when an object is removed from an informer
	DeleteEvent EventType = "DELETE"
)

// Event holds the context of an event.
type Event struct {
	Type      EventType
	Namespace string
	Ports     []int
}

// Informer defines the required SharedIndexInformers that interact with the API server.
type Controller struct {
	informer cache.SharedInformer
	queue    workqueue.RateLimitingInterface
}

func NewController(kubeClient *kubernetes.Clientset) *Controller {
	lw := cache.NewListWatchFromClient(kubeClient.RESTClient(), "services", metav1.NamespaceAll, fields.Everything())
	informer := cache.NewSharedInformer(lw, &corev1.Service{}, time.Second*5)

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	return &Controller{
		informer: informer,
		queue:    queue,
	}
}

func (controller *Controller) Add(event Event) {
	controller.queue.AddRateLimited(event)
}

func (controller *Controller) WatchDeleteEvent(stopCh chan struct{}) {
	go controller.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, controller.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
	}

	controller.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			var event Event
			if service, ok := obj.(*corev1.Service); ok {
				if service.Spec.Type == corev1.ServiceTypeNodePort {
					for ports := range service.Spec.Ports {
						event.Ports = append(event.Ports, ports)
					}
					event.Namespace = service.Namespace
				}
			}
			event.Type = DeleteEvent
			controller.queue.AddRateLimited(event)
		},
	})

}

// Run 运行控制器
func (controller *Controller) Run(stopCh <-chan struct{}) {
	defer controller.queue.ShutDown()

	klog.Info("Controller started")

	// 开启工作协程
	for i := 0; i < 2; i++ {
		go wait.Until(controller.worker, time.Second, stopCh)
	}

	<-stopCh
	klog.Info("Controller stopped")
}

// worker 工作者函数，用于处理 DeltaFIFO 中的事件
func (controller *Controller) worker() {
	for {
		key, shutdown := controller.queue.Get()
		if shutdown {
			klog.Info("Error getting item from FIFO")
			return
		}

		event, ok := key.(Event)
		if !ok {
			klog.Warningln("get a key from workqueue but not Event type.ignore")
			continue
		}

		

		controller.queue.Done(key)
	}
}

