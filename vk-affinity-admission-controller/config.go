/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	certutil "k8s.io/client-go/util/cert"

	"github.com/golang/glog"
)

type certKey struct {
	// CertFile is the PEM-encoded certificate, and possibly the complete certificate chain
	CertFile []byte
	// KeyFile is the PEM-encoded private key for the certificate specified by CertFile
	KeyFile []byte
	// CACertFile is an optional file containing the certificate chain for certKey.CertFile
	CACertFile string
	// CertDirectory is a directory that will contain the certificates.  If the cert and key aren't specifically set
	// this will be used to derive a match with the "pair-name"
	CertDirectory string
	// PairName is the name which will be used with CertDirectory to make a cert and key names
	// It becomes CertDirectory/PairName.crt and CertDirectory/PairName.key
	PairName string
}

// get a clientset with in-cluster config.
func getClient() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatal(err)
	}
	return clientset
}

// retrieve the CA cert that will signed the cert used by the
// "ValidatingAdmissionWebhook or MutationAdmissionWebhook" plugin admission controller.
func getAPIServerCert(clientset *kubernetes.Clientset) []byte {
	c, err := clientset.CoreV1().ConfigMaps("kube-system").Get("extension-apiserver-authentication", metav1.GetOptions{})
	if err != nil {
		glog.Fatal(err)
	}

	pem, ok := c.Data["requestheader-client-ca-file"]
	if !ok {
		glog.Fatalf(fmt.Sprintf("cannot find the ca.crt in the configmap, configMap.Data is %#v", c.Data))
	}
	glog.Info("client-ca-file=", pem)
	return []byte(pem)
}

func configTLS(clientset *kubernetes.Clientset, ck *certKey) *tls.Config {
	cert := getAPIServerCert(clientset)
	var err error
	apiserverCA := x509.NewCertPool()
	apiserverCA.AppendCertsFromPEM(cert)

	certPath := path.Join(ck.CertDirectory, ck.PairName+".crt")
	keyPath := path.Join(ck.CertDirectory, ck.PairName+".key")

	ck.CertFile, err = ioutil.ReadFile(certPath)
	if err != nil {
		glog.Fatalf("Cannot read cert from %s", certPath)
	}
	ck.KeyFile, err = ioutil.ReadFile(keyPath)
	if err != nil {
		glog.Fatalf("Cannot read key from %s", keyPath)
	}

	_, err = certutil.CanReadCertAndKey(certPath, keyPath)
	if err != nil {
		glog.Fatal("Cannot verify server certificate and key")
	}

	sCert, err := tls.X509KeyPair(ck.CertFile, ck.KeyFile)
	if err != nil {
		glog.Fatal(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{sCert},
		ClientCAs:    apiserverCA,
		// ClientAuth:   tls.RequireAndVerifyClientCert,
	}
}
