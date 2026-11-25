package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

func main() {
	gvr := schema.GroupVersionResource{Group: "opmodel.dev", Version: "v1alpha1", Resource: "moduleinstances"}
	for _, host := range []string{"https://127.0.0.1:1", "https://no-such-host.invalid:6443", "https://10.255.255.1:6443"} {
		c, _ := dynamic.NewForConfig(&rest.Config{Host: host, Timeout: 3 * time.Second, TLSClientConfig: rest.TLSClientConfig{Insecure: true}})
		_, err := c.Resource(gvr).Namespace("default").Get(context.Background(), "x", metav1.GetOptions{})
		var ne net.Error
		fmt.Printf("%s\n  err=%v\n  type=%T apiStatus=%v timeout=%v unavailable=%v netErr=%v\n", host, err, err,
			func() bool { _, ok := err.(apierrors.APIStatus); return ok }(), apierrors.IsServerTimeout(err), apierrors.IsServiceUnavailable(err), errors.As(err, &ne))
	}
}
