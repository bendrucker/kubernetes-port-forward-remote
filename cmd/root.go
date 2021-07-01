package cmd

import (
	"strconv"

	"github.com/bendrucker/kubernetes-port-forward-remote/pkg/forward"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var overrides clientcmd.ConfigOverrides

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kubernetes-port-forward-remote",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Args: cobra.ExactArgs(2),
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

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	clientcmd.BindOverrideFlags(&overrides, rootCmd.PersistentFlags(), clientcmd.RecommendedConfigOverrideFlags(""))
}
