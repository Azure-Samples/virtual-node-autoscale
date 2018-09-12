package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jeremyrickard/prometheus-containercounter/pkg/watcher"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	namespace := os.Getenv("NAMESPACE")
	labelSelector := os.Getenv("LABEL_SELECTOR")
	specialNode := os.Getenv("SPECIAL_NODE_NAME")
	if specialNode == "" {
		specialNode = "virtual_kubelet"
	}
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		log.Fatalf("couldn't read home directory for kubeconf")
	}
	kubeConfig := filepath.Join(home, ".kube", "config")
	watcher, err := watcher.New(labelSelector, namespace, specialNode, kubeConfig)
	go watcher.Run()
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
