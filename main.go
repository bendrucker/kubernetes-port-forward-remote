package main

import (
	"os"

	"github.com/bendrucker/kubernetes-port-forward-remote/cmd"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func main() {
	cmd.Execute(genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})
}
