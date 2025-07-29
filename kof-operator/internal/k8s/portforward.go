package k8s

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForwardAPodRequest struct {
	WaitGroup  *sync.WaitGroup
	RestConfig *rest.Config
	Pod        *v1.Pod
	LocalPort  int
	PodPort    int
	Streams    genericclioptions.IOStreams
	StopCh     <-chan struct{}
	ReadyCh    chan struct{}
	ErrorCh    chan error
}

func PortForwardAPod(req PortForwardAPodRequest) {
	defer req.WaitGroup.Done()

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", req.Pod.Namespace, req.Pod.Name)
	hostIP := strings.TrimLeft(req.RestConfig.Host, "htps:/")

	transport, upgrader, err := spdy.RoundTripperFor(req.RestConfig)
	if err != nil {
		req.ErrorCh <- err
	}

	dialer := spdy.NewDialer(
		upgrader,
		&http.Client{
			Transport: transport,
		},
		http.MethodPost,
		&url.URL{
			Scheme: "https",
			Path:   path,
			Host:   hostIP,
		},
	)
	fw, err := portforward.New(
		dialer,
		[]string{fmt.Sprintf("%d:%d", req.LocalPort, req.PodPort)},
		req.StopCh,
		req.ReadyCh,
		req.Streams.Out,
		req.Streams.ErrOut,
	)
	if err != nil {
		req.ErrorCh <- err
	}

	err = fw.ForwardPorts()
	if err != nil {
		req.ErrorCh <- err
	}
}
