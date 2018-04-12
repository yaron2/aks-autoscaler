package main

import (
	"flag"
	"os"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const defaultNamespace string = "default"

type AzureAutoScaler struct {
	MaxNodes        int
	MinNodes        int
	Executor        ScaleCommandExecutor
	Started         bool
	ScaleOperations []ScaleOperation
}

func NewAzureAutoScaler(executor ScaleCommandExecutor, maxNodes int, minNodes int) *AzureAutoScaler {
	if maxNodes == 0 {
		maxNodes = 100
	}

	if minNodes == 0 {
		minNodes = 2
	}

	return &AzureAutoScaler{MaxNodes: maxNodes, MinNodes: minNodes, Executor: executor}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func createKubeClient() *kubernetes.Clientset {
	var kubeconfig *string
	if home := homeDir(); home != "" {
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

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		panic(err.Error())
	}

	return clientset
}

func (a AzureAutoScaler) Start() {
	a.Executor.Login()

	kubeClient := createKubeClient()
	a.watchDeploymentsStatus(kubeClient)
}

func (a *AzureAutoScaler) getDeploymentStatus(kubeClient *kubernetes.Clientset) {
	allNodes, _ := kubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	allPods, _ := kubeClient.CoreV1().Pods(defaultNamespace).List(metav1.ListOptions{})
	allDeps, _ := kubeClient.ExtensionsV1beta1().Deployments(defaultNamespace).List(metav1.ListOptions{})

	nodeCount := len(allNodes.Items)
	var unschedulablePodsDetected bool

	var neededAgents int32
	scaledDeploymentIds := []types.UID{}

	for _, pod := range allPods.Items {
		if string(pod.Status.Phase) == "Pending" {
			for _, condition := range pod.Status.Conditions {
				if condition.Reason == "Unschedulable" {
					unschedulablePodsDetected = true
				}
			}
		}
	}

	for _, dep := range allDeps.Items {
		unavailableReplicas := dep.Status.UnavailableReplicas
		if unavailableReplicas > 0 && a.IsScaleAllowed(dep.UID) {
			neededAgents++
			scaledDeploymentIds = append(scaledDeploymentIds, dep.UID)
		} else if unavailableReplicas == 0 && !a.IsScaleAllowed(dep.UID) {
			a.removeScaleOperation(dep.UID)
		}
	}

	if neededAgents > 0 && unschedulablePodsDetected {
		if int(neededAgents) > a.MaxNodes && nodeCount < a.MaxNodes {
			a.ScaleUp(scaledDeploymentIds, int32(a.MaxNodes))
		} else {
			a.ScaleUp(scaledDeploymentIds, int32(nodeCount)+neededAgents)
		}
	} else {
		var emptyNodes int
		pods, err := kubeClient.CoreV1().Pods(defaultNamespace).List(metav1.ListOptions{})

		if err != nil {
			panic(err.Error())
		}

		if len(pods.Items) > 0 {
			for _, node := range allNodes.Items {
				empty := true
				nodeName := node.Name

				for _, pod := range pods.Items {
					podNodeName := pod.Spec.NodeName

					if podNodeName == nodeName {
						empty = false
					}
				}

				if empty {
					emptyNodes++
				}
			}
		} else {
			emptyNodes = nodeCount
		}

		if emptyNodes > 0 {
			if nodeCount-emptyNodes <= a.MinNodes {
				a.ScaleDown(int32(a.MinNodes))
			} else {
				a.ScaleDown(int32(nodeCount - emptyNodes))
			}
		}
	}
}

func (a *AzureAutoScaler) removeScaleDownOperations() {
	for i := 0; i < len(a.ScaleOperations); i++ {
		scaleOp := a.ScaleOperations[i]

		if scaleOp.ScaleDirection == "down" {
			a.ScaleOperations = append(a.ScaleOperations[:i], a.ScaleOperations[i+1:]...)
		}
	}
}

func (a *AzureAutoScaler) removeScaleOperation(ID types.UID) {
	for i := 0; i < len(a.ScaleOperations); i++ {
		scaleOp := a.ScaleOperations[i]

		if scaleOp.DeploymentID == ID {
			a.ScaleOperations = append(a.ScaleOperations[:i], a.ScaleOperations[i+1:]...)
		}
	}
}

func (a *AzureAutoScaler) IsScaleAllowed(deploymentID types.UID) bool {
	for _, scaleOp := range a.ScaleOperations {
		if scaleOp.DeploymentID == deploymentID {
			return false
		}
	}

	return true
}

func (a *AzureAutoScaler) ScaleDown(agents int32) {
	// TODO: Act on return value from Scale and rollback if needed

	for _, scaleOp := range a.ScaleOperations {
		if scaleOp.ScaleDirection == "down" {
			return
		}
	}

	scaleOp := NewScaleOperation("", "down")
	a.ScaleOperations = append(a.ScaleOperations, scaleOp)

	a.Executor.Scale(agents)
	a.removeScaleDownOperations()
}

func (a *AzureAutoScaler) ScaleUp(deploymentIDs []types.UID, agents int32) {
	// TODO: Act on return value from Scale and rollback if needed

	for _, dep := range deploymentIDs {
		scaleOp := NewScaleOperation(dep, "up")
		a.ScaleOperations = append(a.ScaleOperations, scaleOp)
	}

	a.Executor.Scale(agents)

	for _, dep := range deploymentIDs {
		a.removeScaleOperation(dep)
	}
}

func (a *AzureAutoScaler) watchDeploymentsStatus(kubeClient *kubernetes.Clientset) {
	if !a.Started {
		a.Started = true
		t := time.NewTicker(10 * time.Second)
		q := make(chan struct{})

		func() {
			for {
				select {
				case <-t.C:
					a.getDeploymentStatus(kubeClient)
				case <-q:
					t.Stop()
					return
				}
			}
		}()
	}
}
