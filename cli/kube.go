package cli

// temporary file to explore K8s

import (
	"context"
	"fmt"
	"path/filepath"

	//"k8s.io/apimachinery/pkg/api/errors"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/czankel/cne/errdefs"
	"github.com/czankel/cne/kube"
	/*
		"github.com/czankel/cne/container"
		"github.com/czankel/cne/project"
		"github.com/czankel/cne/runtime"
	*/
	"github.com/spf13/cobra"
)

var kubeCmd = &cobra.Command{
	Use: "kube",
}

var kubeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Kubernetes cluster status",
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

	// FIXME: support specific K8s cluster...
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	// FIXME
	namespace := "" // all

	ctx := context.Background()
	ctx = kube.WithNamespace(ctx, namespace)

	pods, err := kube.GetPods(ctx, clientset)
	if err != nil {
		return err
	}
	fmt.Printf("There are %d pods in the cluster\n", len(pods))
	printList(pods, false)

	/*
		// Examples for error handling:
		// - Use helper functions like e.g. errors.IsNotFound()
		// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
		pods, err = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		fmt.Printf("POD \"%s\"\n", pods)
		printValue("", "", "", pods)
		/*
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
	*/
	return nil
}

func init() {
	rootCmd.AddCommand(kubeCmd)
	kubeCmd.AddCommand(kubeStatusCmd)
}
