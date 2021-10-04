package forward

import (
	"context"
	"fmt"
	"net/http"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type Spec struct {
	LocalPort int

	RemoteHost string
	RemotePort int

	Timeout time.Duration
}

func (s *Spec) String() string {
	return fmt.Sprintf("%d:%d", s.LocalPort, s.RemotePort)
}

type Forwarder struct {
	Namespace string
	Client    *kubernetes.Clientset
	Config    *rest.Config

	genericclioptions.IOStreams

	pod string
}

func (f *Forwarder) Forward(ctx context.Context, spec Spec) error {
	err := f.createPod(ctx, spec)
	if err != nil {
		return err
	}

	defer f.deletePod(ctx)

	dialer, err := f.dialer()
	if err != nil {
		return err
	}

	fw, err := portforward.New(dialer, []string{spec.String()}, make(chan struct{}), make(chan struct{}), f.Out, f.ErrOut)
	if err != nil {
		return err
	}

	if err = fw.ForwardPorts(); err != nil {
		return err
	}

	return nil
}

func (f *Forwarder) dialer() (httpstream.Dialer, error) {
	transport, upgrader, err := spdy.RoundTripperFor(f.Config)
	if err != nil {
		return nil, err
	}

	return spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", f.Client.RESTClient().Post().Prefix("api/v1").Resource("pods").Namespace(f.Namespace).Name(f.pod).SubResource("portforward").URL()), nil
}

func (f *Forwarder) createPod(ctx context.Context, spec Spec) error {
	pod, err := f.Client.CoreV1().Pods(f.Namespace).Create(ctx, Pod(spec), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	f.pod = pod.Name

	if err = waitPodRunning(ctx, f.Client, pod); err != nil {
		return err
	}

	return nil
}

func (f *Forwarder) deletePod(ctx context.Context) error {
	return f.Client.CoreV1().Pods(f.Namespace).Delete(ctx, f.pod, metav1.DeleteOptions{})
}

func waitPodRunning(ctx context.Context, client kubernetes.Interface, pod *v1.Pod) error {
	return wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		p, err := client.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		switch p.Status.Phase {
		case v1.PodRunning:
			return true, nil
		case v1.PodFailed, v1.PodSucceeded:
			return false, fmt.Errorf("pod phase is %s", p.Status.Phase)
		}
		return false, nil
	})
}
