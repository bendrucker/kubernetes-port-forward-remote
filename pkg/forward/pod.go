package forward

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Pod(spec Spec) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "port-forward-remote-",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "socat",
					Image: "alpine/socat",
					Command: command(spec.Timeout, []string{
						"socat",
						fmt.Sprintf("tcp-listen:%d,fork,reuseaddr", spec.RemotePort),
						fmt.Sprintf("tcp-connect:%s:%d", spec.RemoteHost, spec.RemotePort),
					}),
					Ports: []v1.ContainerPort{
						{
							Name:          "forwarded",
							ContainerPort: int32(spec.RemotePort),
						},
					},
				},
			},
		},
	}
}

func command(timeout time.Duration, args []string) []string {
	if timeout == 0 {
		return args
	}

	return append([]string{
		"timeout",
		fmt.Sprintf("%fs", timeout.Seconds()),
	}, args...)
}
