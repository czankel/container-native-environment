package cli

// temporary file to explore K8s

import (
	"github.com/spf13/cobra"

	"context"
	"fmt"
	"path/filepath"

	//"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/czankel/cne/errdefs"
	/*
		"github.com/czankel/cne/container"
		"github.com/czankel/cne/project"
		"github.com/czankel/cne/runtime"
	*/)

var kubeCmd = &cobra.Command{
	Use: "kube",
}

var kubeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show some status",
	Args:  cobra.NoArgs,
	RunE:  kubeStatusRunE,
}

func kubeStatusRunE(cmd *cobra.Command, args []string) error {

	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		return errdefs.InvalidArgument("Missing $HOME/.kube/config")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
	//fmt.Printf("POD \"%s\" %d\n", pods, len(pods.Items))

	/*
		for p := range pods.Items {
			//runtimes, err := clientset.RuntimeV1().List(context.TODO(), metav1.ListOptions{})
			fmt.Printf("Pod %s\n", p.PodSpec())
		}
	*/

	nodes := clientset.CoreV1().Nodes() //.List(context.TODO(), metav1.ListOptions())
	fmt.Printf("nodes %s\n", nodes)

	/*
		// Examples for error handling:
		// - Use helper functions like e.g. errors.IsNotFound()
		// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
		namespace := "default"
		pods, err = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
			if errors.IsNotFound(err) {
				fmt.Printf("Pod %s in namespace %s not found\n", pod, namespace)
			} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
				fmt.Printf("Error getting pod %s in namespace %s: %v\n",
					pod, namespace, statusError.ErrStatus.Message)
			} else if err != nil {
				panic(err.Error())
			} else {
				fmt.Printf("Found pod %s in namespace %s\n", pod, namespace)
			}
		fmt.Printf("POD \"%s\" %d\n", pods, len(pods.Items))
		for i := range pods.Items {
			fmt.Printf("POD \"%s\"\n", i)
		}
	*/

	return nil
}

func init() {
	rootCmd.AddCommand(kubeCmd)
	kubeCmd.AddCommand(kubeStatusCmd)
}
