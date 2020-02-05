package plugin

import (
	"fmt"

	"github.com/gosuri/uitable"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type PodRestartsPlugin struct {
	config    *rest.Config
	Clientset *kubernetes.Clientset
	PodObject *v1.Pod
}

func NewPodRestartsPlugin(configFlags *genericclioptions.ConfigFlags) (*PodRestartsPlugin, error) {
	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, errors.New("Failed to read kubeconfig, exiting.")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.New("Failed to create API clientset")
	}

	return &PodRestartsPlugin{
		config:    config,
		Clientset: clientset,
	}, nil
}

func (pd *PodRestartsPlugin) findPodByPodName(namespace string) error {
	tbl := uitable.New()
	tbl.Separator = "    "

	// we will seek the whole cluster if namespace is not passed as a flag (it will be a "" string)
	podFind, err := pd.Clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{})

	v := viper.GetViper()
	listContainers := v.GetBool("containers")
	listThreshold := v.GetInt32("threshold")

	if err != nil || len(podFind.Items) == 0 {
		return errors.New("Failed to get pods data: check your parameters, set a context or verify API server.")
	}

	tbl.AddRow("NAMESPACE", "RESTARTS", "NAME", "LAST START")

	var allRestarts int32 = 0
	for _, pod := range podFind.Items {
		// RestartCount are all int32
		var totalRestarts int32 = 0

		for _, containerStatuses := range pod.Status.ContainerStatuses {
			containersCount := containerStatuses.RestartCount
			if containersCount != int32(0) {
				if listContainers {
					tbl.AddRow(pod.GetNamespace(), containersCount, pod.GetName()+"/"+containerStatuses.Name, pod.Status.StartTime)
				}
				totalRestarts += containersCount
			}
		}

		for _, initContainerStatuses := range pod.Status.InitContainerStatuses {
			initContainersCount := initContainerStatuses.RestartCount
			if initContainersCount != int32(0) {
				if listContainers {
					tbl.AddRow(pod.GetNamespace(), initContainersCount, pod.GetName()+"/"+initContainerStatuses.Name, pod.Status.StartTime)
				}
				totalRestarts += initContainersCount
			}
		}

		if totalRestarts != int32(0) {
			if listThreshold != int32(0) {
				if totalRestarts > listThreshold {
					tbl.AddRow(pod.GetNamespace(), totalRestarts, pod.GetName(), pod.Status.StartTime)
				}
			} else {
				if !listContainers {
					tbl.AddRow(pod.GetNamespace(), totalRestarts, pod.GetName(), pod.Status.StartTime)
				}
			}
			allRestarts += totalRestarts
		}
	}
	if allRestarts == 0 {
		fmt.Println("No restarts.")
	} else {
		fmt.Println(tbl)
	}

	return nil
}

func RunPlugin(configFlags *genericclioptions.ConfigFlags) error {
	pd, err := NewPodRestartsPlugin(configFlags)
	if err != nil {
		return err
	}

	if err := pd.findPodByPodName(*configFlags.Namespace); err != nil {
		return err
	}

	return nil
}
