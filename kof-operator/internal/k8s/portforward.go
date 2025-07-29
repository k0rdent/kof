package k8s

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForwarder struct {
	wg         *sync.WaitGroup
	restConfig *rest.Config
	pod        *corev1.Pod
	localPort  int
	podPort    int
	streams    genericclioptions.IOStreams
	stopCh     chan struct{}
	readyCh    chan struct{}
}

func NewPortForwarder(restConfig *rest.Config, pod *corev1.Pod, podPort, localPort int) *PortForwarder {
	return &PortForwarder{
		wg:         &sync.WaitGroup{},
		restConfig: restConfig,
		pod:        pod,
		podPort:    podPort,
		localPort:  localPort,
		stopCh:     make(chan struct{}),
		readyCh:    make(chan struct{}),
	}
}

func (pf *PortForwarder) Run() error {
	pf.wg.Add(1)
	var startErr error

	go func() {
		defer pf.wg.Done()

		path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", pf.pod.Namespace, pf.pod.Name)
		u, err := url.Parse(pf.restConfig.Host)
		if err != nil {
			startErr = fmt.Errorf("failed to parse host: %v", err)
			return
		}

		hostIP := u.Host

		transport, upgrader, err := spdy.RoundTripperFor(pf.restConfig)
		if err != nil {
			startErr = fmt.Errorf("failed to create round tripper: %v", err)
			return
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
			[]string{fmt.Sprintf("%d:%d", pf.localPort, pf.podPort)},
			pf.stopCh,
			pf.readyCh,
			pf.streams.Out,
			pf.streams.ErrOut,
		)
		if err != nil {
			startErr = fmt.Errorf("failed to create port forward: %v", err)
			return
		}

		if err = fw.ForwardPorts(); err != nil {
			startErr = fmt.Errorf("failed to forward ports: %v", err)
		}
	}()

	select {
	case <-pf.readyCh:
		return startErr
	case <-time.After(30 * time.Second):
		return fmt.Errorf("port-forward startup timeout")
	}
}

func (pf *PortForwarder) Close() {
	close(pf.stopCh)
	pf.wg.Wait()
}

func (p *PortForwarder) DoRequest(endpoint string) ([]byte, error) {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/%s", p.localPort, endpoint))
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	return body, err
}
