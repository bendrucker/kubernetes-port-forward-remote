package cmd

import (
	"strconv"

	"github.com/bendrucker/kubernetes-port-forward-remote/pkg/forward"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewForwardCommand() *cobra.Command {
	overrides := clientcmd.ConfigOverrides{}

	cmd := &cobra.Command{
		Use:   "kubectl port-forward-remote",
		Short: "Forward from a local port to a remote host via a Kubernetes pod",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			kc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				clientcmd.NewDefaultClientConfigLoadingRules(),
				&overrides,
			)

			config, err := kc.ClientConfig()
			if err != nil {
				return err
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return err
			}

			port, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}

			spec := forward.Spec{
				LocalPort:  0,
				RemoteHost: args[0],
				RemotePort: port,
			}

			ns, _, _ := kc.Namespace()
			forwarder := forward.Forwarder{
				Namespace: ns,
				Client:    clientset,
				Config:    config,

				Stdout: cmd.OutOrStdout(),
				Stderr: cmd.ErrOrStderr(),
			}

			return forwarder.Forward(cmd.Context(), spec)
		},
	}

	clientcmd.BindOverrideFlags(&overrides, cmd.PersistentFlags(), clientcmd.RecommendedConfigOverrideFlags(""))

	return cmd
}

func Execute() {
	cmd := NewForwardCommand()
	cobra.CheckErr(cmd.Execute())
}
