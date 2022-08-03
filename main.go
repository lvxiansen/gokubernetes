package main

import (
	"context"
	"flag"
	"fmt"
	"gokuberntes/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"path/filepath"
	"time"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

func main() {
	// 读取配置文件
	config := getConfig()

	// 创建kubernetes客户端
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// 服务器版本
	//ServerVersion, _ := clientset.ServerVersion()

	//输出pods
	//getPods(clientset)

	// 错误处理 Examples for error handling:
	//handleError(clientset)

	getPods(clientset)

	// namespace列表 default kube-node-lease kube-publi kube-system meshnet
	//namespaces := getNamespace(clientset)
	//fmt.Println("------------namespaces:",namespaces)
	//namespace := "klish"
	//pod := "ce1"
	//temp, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), pod, metaV1.GetOptions{})
	//fmt.Printf("%+v",temp)

	// deployments列表 calico-kube-controllers coredns
	//getDeployments(clientset,namespaces)

	//service列表 kubernetes default map  kube-dns kube-system
	//getService(clientset,namespaces)

}

func getConfig() *restclient.Config {
	// 设置配置文件目录
	var kubeconfig *string
	if home, _ := os.Getwd(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, "kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	return config
}

type Time struct {
	time.Time `protobuf:"-"`
}

// ToRequestParam returns K8s API namespace query for list of objects from this namespaces.
// This is an optimization to query for single namespace if one was selected and for all
// namespaces otherwise.
func (n common.NamespaceQuery) ToRequestParam() string {
	if len(n.namespaces) == 1 {
		return n.namespaces[0]
	}
	return ""
}

// Matches returns true when the given namespace matches this query.
func (n *NamespaceQuery) Matches(namespace string) bool {
	if len(n.namespaces) == 0 {
		return true
	}

	for _, queryNamespace := range n.namespaces {
		if namespace == queryNamespace {
			return true
		}
	}
	return false
}

func getPods(clientset *kubernetes.Clientset) {
	//得到所有pod
	//pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metaV1.ListOptions{})
	//if err != nil {
	//	panic(err.Error())
	//}
	//namespace := "klish"
	var nsQuery *NamespaceQuery = &NamespaceQuery{[]string{"klish"}}
	//var nsQuery *NamespaceQuery =  &NamespaceQuery{[]string{}}
	list, err := clientset.CoreV1().Pods(nsQuery.ToRequestParam()).List(context.TODO(), metaV1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}
	var filteredItems []v1.Pod
	for _, item := range list.Items {
		if nsQuery.Matches(item.ObjectMeta.Namespace) {
			filteredItems = append(filteredItems, item)
		}
	}
	list.Items = filteredItems
	temp, err := clientset.CoreV1().Events(nsQuery.ToRequestParam()).List(context.TODO(), metaV1.ListOptions{})
	//fmt.Println(temp)
	//fmt.Println(klishPods)
	//fmt.Println()a
	//fmt.Printf("------------There are %d pods in the cluster\n", len(pods.Items))

	type ResultPods struct {
		Name              string            `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
		Namespace         string            `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
		UID               types.UID         `json:"uid,omitempty" protobuf:"bytes,5,opt,name=uid,casttype=k8s.io/kubernetes/pkg/types.UID"`
		CreationTimestamp Time              `json:"creationTimestamp,omitempty" protobuf:"bytes,8,opt,name=creationTimestamp"`
		Labels            map[string]string `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`
	}
}

func GetPodListFromChannels(podList *v1.PodList) *v1.PodList {
	pods := podList.Items

}

func handleError(clientset *kubernetes.Clientset) {
	// - Use helper functions like e.g. errors.IsNotFound()
	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	namespace := "default"
	pod := "example-xxxxx"
	_, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), pod, metaV1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("------------Pod %s in namespace %s not found\n", pod, namespace)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting pod %s in namespace %s: %v\n",
			pod, namespace, statusError.ErrStatus.Message)
	} else if err != nil {
		panic(err.Error())
	} else {
		fmt.Printf("Found pod %s in namespace %s\n", pod, namespace)
	}
}

func getNamespace(clientset *kubernetes.Clientset) []string {
	// 相当于命令kubectl get nodes -o yaml
	namespaceClient := clientset.CoreV1().Namespaces()
	namespaceResult, err := namespaceClient.List(context.TODO(), metaV1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	now := time.Now()
	namespaces := []string{}
	for _, namespace := range namespaceResult.Items {
		namespaces = append(namespaces, namespace.Name)
		fmt.Println(namespace.Name, now.Sub(namespace.CreationTimestamp.Time))
	}
	return namespaces
}

func getDeployments(clientset *kubernetes.Clientset, namespaces []string) {
	fmt.Println("------------deployments:")
	for _, namespace := range namespaces {
		deploymentClient := clientset.AppsV1().Deployments(namespace)
		depoymentResult, err := deploymentClient.List(context.TODO(), metaV1.ListOptions{})
		if err != nil {
			log.Println(err)
		} else {
			for _, deployment := range depoymentResult.Items {
				fmt.Println(deployment.Name, deployment.Namespace, deployment.CreationTimestamp)
			}

		}
	}
}

func getService(clientset *kubernetes.Clientset, namespaces []string) {
	fmt.Println("------------services:")
	for _, namespace := range namespaces {
		serviceClient := clientset.CoreV1().Services(namespace)
		serviceResult, err := serviceClient.List(context.TODO(), metaV1.ListOptions{})
		if err != nil {
			log.Println(err)
		} else {
			for _, service := range serviceResult.Items {
				fmt.Println(service.Name, service.Namespace, service.Labels, service.Spec.Selector, service.Spec.Type, service.Spec.ClusterIP, service.Spec.Ports, service.CreationTimestamp)
			}
		}
	}
}
