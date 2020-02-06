package plugin

import (
	"fmt"
	"sort"
	"time"

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

type StructuredPod struct {
	namespace string
	restarts  int32
	name      string
	age       string
	start     string
}

type sortablePods []StructuredPod

func (s sortablePods) Len() int { return len(s) }

func (s sortablePods) Less(i, j int) bool {
	v := viper.GetViper()
	sortBy := v.Get("sort-by")

	switch sortBy {
	case "restarts":
		return s[i].restarts < s[j].restarts
	case "age":
		return s[i].age < s[j].age
	case "start":
		return s[i].start < s[j].start
	}

	return s[i].namespace < s[j].namespace
}

func (s sortablePods) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

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
	if err != nil || len(podFind.Items) == 0 {
		fmt.Println("Failed to get pods data: check your parameters, set a context or verify API server.")
		return nil
	}

	// is there a more correct way to
	// grab flags anywhere inside the code?
	v := viper.GetViper()
	listContainers := v.GetBool("containers")
	listThreshold := v.GetInt32("threshold")
	listSortBy := v.Get("sort-by")

	tbl.AddRow("NAMESPACE", "RESTARTS", "NAME", "AGE", "START")

	var allRestarts int32 = 0
	pods := podFind.Items

	allStructuredPods := []StructuredPod{}
	for _, pod := range pods {
		// restarts in the API are all int32
		var totalRestarts int32 = 0

		// just so we can have pretty printing of ages
		startTimePretty := "0"
		startTime := time.Since(pod.Status.StartTime.Time)
		startSeconds := startTime.Seconds()
		startMinutes := startTime.Minutes()
		startHours := startTime.Hours()
		startDays := startTime.Hours() / 24
		if startSeconds < 180 {
			startTimePretty = fmt.Sprintf("%.0fs", startSeconds)
		} else if startMinutes < 120 {
			startTimePretty = fmt.Sprintf("%.0fm", startMinutes)
		} else if startHours < 72 {
			startTimePretty = fmt.Sprintf("%.0fh", startHours)
		} else {
			startTimePretty = fmt.Sprintf("%.0fd", startDays)
		}

		for _, containerStatuses := range pod.Status.ContainerStatuses {
			containersCount := containerStatuses.RestartCount
			if containersCount != 0 {
				if listContainers {
					var thisPod = StructuredPod{
						pod.GetNamespace(),
						containersCount,
						pod.GetName() + "/" + containerStatuses.Name,
						startTimePretty,
						pod.Status.StartTime.String()}
					allStructuredPods = append(allStructuredPods, thisPod)
				}
				totalRestarts += containersCount
			}
		}

		for _, initContainerStatuses := range pod.Status.InitContainerStatuses {
			initContainersCount := initContainerStatuses.RestartCount
			if initContainersCount != 0 {
				if listContainers {
					var thisPod = StructuredPod{
						pod.GetNamespace(),
						initContainersCount,
						pod.GetName() + "/" + initContainerStatuses.Name,
						startTimePretty,
						pod.Status.StartTime.String()}
					allStructuredPods = append(allStructuredPods, thisPod)
				}
				totalRestarts += initContainersCount
			}
		}

		if totalRestarts != 0 {
			if listThreshold != 0 {
				if totalRestarts > listThreshold {
					var thisPod = StructuredPod{
						pod.GetNamespace(),
						totalRestarts,
						pod.GetName(),
						startTimePretty,
						pod.Status.StartTime.String()}
					allStructuredPods = append(allStructuredPods, thisPod)
				}
			} else {
				if !listContainers {
					var thisPod = StructuredPod{
						pod.GetNamespace(),
						totalRestarts,
						pod.GetName(),
						startTimePretty,
						pod.Status.StartTime.String()}
					allStructuredPods = append(allStructuredPods, thisPod)
				}
			}
			allRestarts += totalRestarts
		}
	}

	if listSortBy != "" {
		sort.Sort(sortablePods(allStructuredPods))
	}

	for _, pod := range allStructuredPods {
		tbl.AddRow(
			pod.namespace,
			pod.restarts,
			pod.name,
			pod.age,
			pod.start)
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
