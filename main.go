package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"gokuberntes/common"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
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

type RestClientApp struct {
	ClientSet *kubernetes.Clientset
	Config    *restclient.Config
	Namespace string
	PodName   string
}

func NewRestClientApp(clientSet *kubernetes.Clientset, config *restclient.Config, nameSpace, podName string) *RestClientApp {
	return &RestClientApp{ClientSet: clientSet, Config: config, Namespace: nameSpace, PodName: podName}
}

func ExecCommandInPodContainer(clientset *kubernetes.Clientset, config *restclient.Config, namespace string,
	podName string, command []string) ([]string, error) {
	//var stdout,stderr bytes.Buffer
	reader, writer := io.Pipe()
	var (
		cmdOutput []string
	)

	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println("line", line)
			cmdOutput = append(cmdOutput, line)
		}
	}()
	stdin := reader
	stdout := writer
	stderr := writer
	tty := false

	//构建请求
	req := clientset.CoreV1().RESTClient().Post()
	req.Resource("pods").
		Name(podName). //podName
		Namespace(namespace).
		SubResource("exec")

	req.VersionedParams(&v1.PodExecOptions{
		Command: command,
		//Container: containerName,
		//Command: []string{"sh", "hello.sh"},
		Stdin:  stdin != nil,
		Stdout: stdout != nil,
		Stderr: stderr != nil,
		TTY:    tty,
	}, scheme.ParameterCodec)

	// 创建执行器
	//fmt.Println("url",req.URL())
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		fmt.Println("NewSPDYExecutor", err)
		return nil, err
	}
	//screen := struct {
	//	io.Reader
	//	io.Writer
	//}{os.Stdin, os.Stdout}
	// 执行命令
	err = exec.Stream(remotecommand.StreamOptions{
		//Stdin: os.Stdin,
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
	if err != nil {
		fmt.Print("stream", err)
		return nil, err
	}
	//res := stdout.String()

	return cmdOutput, nil
	//fmt.Println("stdout.String()",stdout.String())
	////fmt.Println("stderr.String()",stderr.String())
	//arrRes := strings.Split(res," ")
	//fmt.Println(arrRes[0])
}

func (rca *RestClientApp) GetClusterInfo(cmd []string) ([]string, error) {
	//fmt.Println("rca",rca.ClientSet,rca.Config,rca.Namespace,rca.PodName,cmd)
	result, err := ExecCommandInPodContainer(rca.ClientSet, rca.Config, rca.Namespace, rca.PodName, cmd)
	fmt.Println("result", result)
	if err != nil {
		fmt.Println("Error" + err.Error())
		return nil, err
	}
	return result, nil
}

func main() {
	// 读取配置文件
	config := getConfig()

	//config := loadConfig(kbconfig)

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

	//getPods(clientset)
	nq := common.NewNamespaceQuery([]string{"klish"})
	podList, err := GetPodList(clientset, nq)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(podList)

	//app := NewRestClientApp(clientset,config,"klish","ce1")
	//cmd := []string{"ip","neigh"}
	//sRes,err := app.GetClusterInfo(cmd)
	//for _,v := range sRes {
	//	fmt.Println(v)
	//}
	//fmt.Println(sRes)

	result := [][]string{}
	for _, pod := range podList.Pods {
		//fmt.Println(pod)
		app := NewRestClientApp(clientset, config, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
		//cmd := []string{"ip","neigh"}
		//cmd := []string{"route","-n"}
		cmd := []string{"ip", "link"}
		sRes, err := app.GetClusterInfo(cmd)
		fmt.Println(sRes)
		if err != nil {
			fmt.Println("err", err)
		}
		result = append(result, sRes)
	}

	for _, v := range result {
		fmt.Println(v)
	}

	//fmt.Println(strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err)
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

//func loadConfig(kubeconfig string) *restclient.Config {
//	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
//	if err != nil {
//		panic(err.Error())
//	}
//	return config
//}

// GetPodList returns a list of all Pods in the cluster.
func GetPodList(clientset *kubernetes.Clientset, nsQuery *common.NamespaceQuery) (*common.PodList, error) {
	log.Print("Getting list of all pods in the cluster")

	channels := &common.ResourceChannels{
		PodList: common.GetPodListChannelWithOptions(clientset, nsQuery, metaV1.ListOptions{}, 1),
	}

	return GetPodListFromChannels(channels)
}

// GetPodListFromChannels returns a list of all Pods in the cluster
// reading required resource list once from the channels.
func GetPodListFromChannels(channels *common.ResourceChannels) (*common.PodList, error) {
	pods := <-channels.PodList.List
	err := <-channels.PodList.Error
	if err != nil {
		return nil, err
	}

	podList := ToPodList(pods.Items)
	podList.Status = common.GetStatus(pods)
	return &podList, nil
}
func toPod(pod *v1.Pod) common.Pod {
	podDetail := common.Pod{
		ObjectMeta:      common.NewObjectMeta(pod.ObjectMeta),
		TypeMeta:        common.NewTypeMeta(common.ResourceKindPod),
		Status:          common.GetPodStatus(*pod),
		RestartCount:    common.GetRestartCount(*pod),
		NodeName:        pod.Spec.NodeName,
		ContainerImages: common.GetContainerImages(&pod.Spec),
	}

	return podDetail
}
func ToPodList(pods []v1.Pod) common.PodList {
	podList := common.PodList{
		Pods: make([]common.Pod, 0),
	}

	podList.ListMeta = common.ListMeta{TotalItems: len(pods)}

	for _, pod := range pods {
		podDetail := toPod(&pod)
		podList.Pods = append(podList.Pods, podDetail)
	}

	return podList
}

//func getPods(clientset *kubernetes.Clientset){
//	//得到所有pod
//	//pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metaV1.ListOptions{})
//	//if err != nil {
//	//	panic(err.Error())
//	//}
//	//namespace := "klish"
//	var nsQuery *common.NamespaceQuery =  &common.NamespaceQuery{[]string{"klish"}}
//	//var nsQuery *NamespaceQuery =  &NamespaceQuery{[]string{}}
//	list, err := clientset.CoreV1().Pods(nsQuery.ToRequestParam()).List(context.TODO(), metaV1.ListOptions{})
//	if err != nil {
//		fmt.Println(err)
//	}
//	var filteredItems []v1.Pod
//	for _, item := range list.Items {
//		if nsQuery.Matches(item.ObjectMeta.Namespace) {
//			filteredItems = append(filteredItems, item)
//		}
//	}
//	list.Items = filteredItems
//}

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
