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
)

// Event holds the context of an event.
type Event struct {
	Namespace string
	Ports     []int32
}

// Informer defines the required SharedIndexInformers that interact with the API server.
type Queue struct {
	informer  cache.SharedInformer
	workqueue workqueue.RateLimitingInterface
	stopCh    chan struct{}
}

func NewQueue(kubeClient *kubernetes.Clientset) *Queue {
	lw := cache.NewListWatchFromClient(kubeClient.RESTClient(), "services", metav1.NamespaceAll, fields.Everything())
	informer := cache.NewSharedInformer(lw, &corev1.Service{}, time.Second*5)

	rq := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	stopCh := make(chan struct{})
	queue := &Queue{informer: informer, workqueue: rq, stopCh: stopCh}
	
	go queue.watchDeleteEvent(queue.stopCh)

	return queue
}

func (queue *Queue) watchDeleteEvent(stopCh chan struct{}) {
	go queue.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, queue.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
	}

	queue.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			var event Event
			if service, ok := obj.(*corev1.Service); ok {
				if service.Spec.Type == corev1.ServiceTypeNodePort {
					for _, ports := range service.Spec.Ports {
						event.Ports = append(event.Ports, ports.NodePort)
					}
					event.Namespace = service.Namespace
				}
			}
			queue.workqueue.AddRateLimited(event)
		},
	})
}

// run 运行控制器,从workqueue从取出数据交给worker处理
func (queue *Queue) run(stopCh <-chan struct{}) {
	defer queue.workqueue.ShutDown()

	klog.Info("start controller to watch the delete event of service.")
	// 开启工作协程
	for i := 0; i < 2; i++ {
		go wait.Until(queue.worker, time.Second, stopCh)
	}

	<-stopCh
	klog.Info("Controller stopped")
}

// worker 工作者函数，用于处理 DeltaFIFO 中的事件
func (queue *Queue) worker() {
	for {
		key, shutdown := queue.workqueue.Get()
		if shutdown {
			klog.Info("Error getting item from FIFO")
			return
		}

		event, ok := key.(Event)
		if !ok {
			klog.Warningln("get a key from workqueue but not Event type.ignore")
			continue
		}

		queue.workqueue.Done(key)
	}
}
