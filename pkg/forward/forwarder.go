package forward

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type Spec struct {
	LocalPort int

	RemoteHost string
	RemotePort int
}

func (s *Spec) String() string {
	return fmt.Sprintf("%d:%d", s.LocalPort, s.RemotePort)
}

type Forwarder struct {
	Namespace string
	Client    *kubernetes.Clientset
	Config    *rest.Config

	Stdout io.Writer
	Stderr io.Writer

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

	fw, err := portforward.New(dialer, []string{spec.String()}, make(chan struct{}), make(chan struct{}), f.Stdout, f.Stderr)
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

	return spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", f.Client.RESTClient().Post().Prefix("api/v1").Resource("pods").Namespace("default").Name(f.pod).SubResource("portforward").URL()), nil
}

func (f *Forwarder) createPod(ctx context.Context, spec Spec) error {
	pod, err := f.Client.CoreV1().Pods(f.Namespace).Create(ctx, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "port-forward-remote-",
		},
		Spec: podSpec(spec),
	}, metav1.CreateOptions{})
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

func podSpec(spec Spec) v1.PodSpec {
	return v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:  "socat",
				Image: "alpine/socat",
				Args: []string{
					fmt.Sprintf("tcp-listen:%d,fork,reuseaddr", spec.RemotePort),
					fmt.Sprintf("tcp-connect:%s:%d", spec.RemoteHost, spec.RemotePort),
				},
				Ports: []v1.ContainerPort{
					{
						Name:          "forwarded",
						ContainerPort: int32(spec.RemotePort),
					},
				},
			},
		},
	}
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