package watcher

import (
	"log"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	//corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func init() {
	prometheus.MustRegister(vkContainerCounter)
	prometheus.MustRegister(containerCounter)
}

var (
	vkContainerCounter = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "running_containers_vk",
			Help: "Number of Watched Containers in ACI",
		},
	)
	containerCounter = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "running_containers_nodes",
			Help: "Number of Watched Containers on traditional nodes",
		},
	)
)

type Watcher interface {
	Run() error
}

type watcher struct {
	namespace string
	podLabel  string
	k8sClient *kubernetes.Clientset
	nodeName  string
}

func (w *watcher) Run() error {
	tickChan := time.NewTicker(time.Second * 30).C
	for {
		<-tickChan
		w.sendUpdatedMetrics()
	}
}

func (w *watcher) sendUpdatedMetrics() {
	log.Println("attempting to get pod counts")
	listOpts := metav1.ListOptions{
		LabelSelector: w.podLabel,
	}
	pods, err := w.k8sClient.CoreV1().Pods(w.namespace).List(listOpts)
	if err != nil {
		log.Printf("got an error trying to fetch pods: %s", err)
		return
	}
	var vkp int
	var nvkp int
	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" {
			if pod.Spec.NodeName == w.nodeName {
				vkp++
			} else {
				nvkp++
			}
		}
	}
	log.Printf("Latest metrics: vpk(%v), nvkp(%v)", vkp, nvkp)
	vkContainerCounter.Set(float64(vkp))
	containerCounter.Set(float64(nvkp))
}

func New(podLabel, namespace, nodeName, kubeConfig string) (Watcher, error) {
	var config *rest.Config
	// Check if the kubeConfig file exists.
	if _, err := os.Stat(kubeConfig); !os.IsNotExist(err) {
		// Get the kubeconfig from the filepath.
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			return nil, err
		}
	} else {
		// Set to in-cluster config.
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &watcher{
			k8sClient: clientset,
			namespace: namespace,
			podLabel:  podLabel,
			nodeName:  nodeName,
		},
		nil
}
