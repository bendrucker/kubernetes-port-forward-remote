package cmd

import (
	"flag"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var cfgFile string

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
	Run: func(cmd *cobra.Command, args []string) {
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()

		// use the current context in kubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic(err.Error())
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		port, err := strconv.Atoi(args[1])
		if err != nil {
			panic(err.Error())
		}

		pod, err := clientset.CoreV1().Pods("default").Create(cmd.Context(), &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "port-forward-remote-",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "socat",
						Image: "alpine/socat",
						Args: []string{
							fmt.Sprintf("tcp-listen:%d,fork,reuseaddr", port),
							fmt.Sprintf("tcp-connect:%s:%d", args[0], port),
						},
						Ports: []v1.ContainerPort{
							{
								Name:          "forwarded",
								ContainerPort: int32(port),
							},
						},
					},
				},
			},
		}, metav1.CreateOptions{})

		if err != nil {
			panic(err.Error())
		}

		cmd.Println(pod.Name)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
}
