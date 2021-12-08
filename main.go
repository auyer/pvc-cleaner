package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Conf struct {
	PrefixFilter string `yaml:"prefixFilter"`
	Namespace    string `yaml:"namespace"`
	SufixFilter  string `yaml:"sufixFilter"`
}

func readConf(filename string) (*Conf, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := &Conf{}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %v", filename, err.Error())
	}

	return c, nil
}

func main() {
	var kubeconfig *string
	var configFile *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	configFile = flag.String("c", "./config.yaml", "config file")
	flag.Parse()

	var config *rest.Config
	var err error
	if *kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			log.Println(err, " attempting InClusterConfig")
			config, err = rest.InClusterConfig()
			if err != nil {
				panic(err.Error())
			}
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	conf, err := readConf(*configFile)
	if err != nil {
		log.Println("Failed to read config file, loading defaults. ", err.Error())
	}

	ctx := context.Background()
	ns := conf.Namespace
	if conf.Namespace == "" {
		ns = v1.NamespaceAll
	}
	pvcClient := clientset.CoreV1().PersistentVolumeClaims(ns)
	podsClient := clientset.CoreV1().Pods(ns)
	pvcList, err := pvcClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	if conf.PrefixFilter != "" {
		pvcList.Items = filterFunc(pvcList.Items, prefixFilterFunc(conf.PrefixFilter))
	}

	if conf.SufixFilter != "" {
		pvcList.Items = filterFunc(pvcList.Items, sufixFilterFunc(conf.SufixFilter))
	}
	podList, err := podsClient.List(ctx, metav1.ListOptions{})
	var podVolumes []v1.Volume
	canTrustPodlist := true
	if err != nil {
		log.Println("Failed to get Pods. wont be able to delete unused PVCs.")
		canTrustPodlist = false
	} else {
		for _, pod := range podList.Items {
			for _, volume := range pod.Spec.Volumes {
				if volume.PersistentVolumeClaim != nil {
					podVolumes = append(podVolumes, volume)
				}
			}
		}
	}

	for _, item := range pvcList.Items {
		if item.Status.Phase != v1.ClaimBound || (canTrustPodlist && !isVolumeUsed(item, &podVolumes)) {
			log.Println("Deleting: ", item.Name)
			err = pvcClient.Delete(ctx, item.Name, metav1.DeleteOptions{})

			if err != nil {
				log.Println("Failed to delete: ", err.Error())
			}
		}
	}
}

func isVolumeUsed(volumeClaim v1.PersistentVolumeClaim, volumes *[]v1.Volume) bool {
	for _, volume := range *volumes {
		if volume.PersistentVolumeClaim.ClaimName == volumeClaim.Name {
			return true
		}
	}
	return false
}

// filterFunc accespts different filters
func filterFunc(list []v1.PersistentVolumeClaim, f func(string) bool) []v1.PersistentVolumeClaim {
	newList := []v1.PersistentVolumeClaim{}
	for _, i := range list {
		if f(i.Name) {
			newList = append(newList, i)
		}
	}
	return newList
}

// prefixFilterFunc filters by prefix
func prefixFilterFunc(prefix string) func(string) bool {
	return func(field string) bool {
		return strings.HasPrefix(field, prefix)
	}
}

// sufixFilterFunc filters by prefix
func sufixFilterFunc(sufix string) func(string) bool {
	return func(field string) bool {
		return strings.HasSuffix(field, sufix)
	}
}
