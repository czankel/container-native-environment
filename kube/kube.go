package kube

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	// "k8s.io/client-go/tools/clientcmd"
	// "k8s.io/client-go/util/homedir"
)

const ContextNamespace = "CNE_NS"

func WithNamespace(ctx context.Context, ns string) context.Context {
	return context.WithValue(ctx, ContextNamespace, ns)
}

type Pod struct {
	Name string
}

func GetPods(ctx context.Context, cs *kubernetes.Clientset) ([]Pod, error) {

	ns := ctx.Value(ContextNamespace).(string)
	kPods, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pods := make([]Pod, len(kPods.Items))
	fmt.Println("GPU")
	for i, p := range kPods.Items {
		pods[i] = Pod{
			Name: p.Name,
		}
		res := p.Spec.Containers[0].Resources
		name := p.Spec.Containers[0].Name
		limits := res.Limits
		requests := res.Requests
		gpu, ok := limits["nvidia.com/gpu"]
		if ok {
			fmt.Printf("%s - GPU Limits: %v Requests: %v\n", name, gpu, requests)
		}
	}

	//cs.CoreV1().Pods(ns).Update(ctx, name,
	return pods, nil
}
