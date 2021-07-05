package main

import (
	"context"
	"github.com/k8s-autoops/autoops"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/clientset/versioned"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func exit(err *error) {
	if *err != nil {
		log.Println("exited with error:", (*err).Error())
		os.Exit(1)
	} else {
		log.Println("exited")
	}
}

func main() {
	var err error
	defer exit(&err)

	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	var client *kubernetes.Clientset
	if client, err = autoops.InClusterClient(); err != nil {
		return
	}

	mClient := versioned.New(client.RESTClient())

	s := &http.Server{
		Addr: ":443",
		Handler: autoops.NewMutatingAdmissionHTTPHandler(
			func(ctx context.Context, request *admissionv1.AdmissionRequest, patches *[]map[string]interface{}) (deny string, err error) {
				var buf []byte
				if buf, err = request.Object.MarshalJSON(); err != nil {
					return
				}
				var pod corev1.Pod
				if err = json.Unmarshal(buf, &pod); err != nil {
					return
				}
				log.Println("Try to Create Pod:", request.Name, "in", request.Namespace)
				var replicaSetName string
				for _, ref := range pod.OwnerReferences {
					if ref.Kind == "ReplicaSet" {
						replicaSetName = ref.Name
					}
				}
				if replicaSetName == "" {
					return
				}
				log.Println("Found ReplicaSet:", replicaSetName)
				var replicaSet *v1.ReplicaSet
				if replicaSet, err = client.AppsV1().ReplicaSets(request.Namespace).Get(ctx, replicaSetName, metav1.GetOptions{}); err != nil {
					return
				}
				var deploymentName string
				for _, ref := range replicaSet.OwnerReferences {
					if ref.Kind == "Deployment" {
						deploymentName = ref.Name
					}
				}
				if deploymentName == "" {
					return
				}
				log.Println("Found Deployment:", deploymentName)
				var deployment *v1.Deployment
				if deployment, err = client.AppsV1().Deployments(request.Namespace).Get(ctx, deploymentName, metav1.GetOptions{}); err != nil {
					return
				}
				labels := deployment.Spec.Template.Labels
				var pods *corev1.PodList
				if pods, err = client.CoreV1().Pods(request.Namespace).List(ctx, metav1.ListOptions{
					LabelSelector: Labels2Selector(labels),
					FieldSelector: "status.phase=Running",
				}); err != nil {
					return
				}
				log.Println("Known Pods:", len(pods.Items))

				var (
					maxCPU = map[string]*resource.Quantity{}
					maxMEM = map[string]*resource.Quantity{}
				)

				for _, knownPod := range pods.Items {
					var podMetrics *metricsv1beta1.PodMetrics
					if podMetrics, err = mClient.MetricsV1beta1().PodMetricses(knownPod.Namespace).Get(context.Background(), knownPod.Name, metav1.GetOptions{}); err != nil {
						return
					}
					for _, containerMetrics := range podMetrics.Containers {
						if cpu := containerMetrics.Usage.Cpu(); cpu != nil {
							if maxCPU[containerMetrics.Name] == nil {
								maxCPU[containerMetrics.Name] = cpu
							} else {
								if cpu.Cmp(*maxCPU[containerMetrics.Name]) == 1 {
									maxCPU[containerMetrics.Name] = cpu
								}
							}
						}
						if mem := containerMetrics.Usage.Memory(); mem != nil {
							if maxMEM[containerMetrics.Name] == nil {
								maxMEM[containerMetrics.Name] = mem
							} else {
								if mem.Cmp(*maxMEM[containerMetrics.Name]) == 1 {
									maxMEM[containerMetrics.Name] = mem
								}
							}
						}
					}
				}
				for k, v := range maxCPU {
					log.Println("Known Max CPU", k, "=", v.String())
				}
				for k, v := range maxMEM {
					log.Println("Known Max MEM", k, "=", v.String())
				}
				for i, container := range pod.Spec.Containers {
					if len(container.Resources.Requests) == 0 && len(container.Resources.Limits) == 0 {
						*patches = append(*patches, map[string]interface{}{
							"op":    "replace",
							"path":  "/spec/containers/" + strconv.Itoa(i) + "/resources",
							"value": map[string]interface{}{},
						})
					}
					if len(container.Resources.Requests) == 0 {
						*patches = append(*patches, map[string]interface{}{
							"op":    "replace",
							"path":  "/spec/containers/" + strconv.Itoa(i) + "/resources/requests",
							"value": map[string]interface{}{},
						})
					}
					if len(container.Resources.Limits) == 0 {
						*patches = append(*patches, map[string]interface{}{
							"op":    "replace",
							"path":  "/spec/containers/" + strconv.Itoa(i) + "/resources/limits",
							"value": map[string]interface{}{},
						})
					}
					{
						if cpuUsage := maxCPU[container.Name]; cpuUsage != nil {
							cpuReq := container.Resources.Requests.Cpu()
							if cpuReq == nil || cpuUsage.Cmp(*cpuReq) == 1 {
								*patches = append(*patches, map[string]interface{}{
									"op":    "replace",
									"path":  "/spec/containers/" + strconv.Itoa(i) + "/resources/requests/cpu",
									"value": cpuUsage.String(),
								})
								log.Println("CPU Requests Updated")
							}

							cpuLim := container.Resources.Limits.Cpu()
							if cpuLim != nil && cpuUsage.Cmp(*cpuLim) == 1 {
								newCPULim := cpuUsage.DeepCopy()
								newCPULim.Add(newCPULim)
								*patches = append(*patches, map[string]interface{}{
									"op":    "replace",
									"path":  "/spec/containers/" + strconv.Itoa(i) + "/resources/limits/cpu",
									"value": newCPULim.String(),
								})
								log.Println("CPU Limits Updated")
							}
						}
					}
					{
						if memUsage := maxMEM[container.Name]; memUsage != nil {
							memReq := container.Resources.Requests.Memory()
							if memReq == nil || memUsage.Cmp(*memReq) == 1 {
								*patches = append(*patches, map[string]interface{}{
									"op":    "replace",
									"path":  "/spec/containers/" + strconv.Itoa(i) + "/resources/requests/memory",
									"value": memUsage.String(),
								})
								log.Println("MEM Requests Updated")
							}

							memLim := container.Resources.Limits.Memory()
							if memLim != nil && memUsage.Cmp(*memLim) == 1 {
								newMEMLim := memUsage.DeepCopy()
								newMEMLim.Add(newMEMLim)
								*patches = append(*patches, map[string]interface{}{
									"op":    "replace",
									"path":  "/spec/containers/" + strconv.Itoa(i) + "/resources/limits/memory",
									"value": newMEMLim.String(),
								})
								log.Println("MEM Limits Updated")
							}
						}
					}
				}
				return
			},
		),
	}

	if err = autoops.RunAdmissionServer(s); err != nil {
		return
	}
}

func Labels2Selector(labels map[string]string) string {
	var items []string
	for k, v := range labels {
		items = append(items, k+"="+v)
	}
	return strings.Join(items, ",")
}
