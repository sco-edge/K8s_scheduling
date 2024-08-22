package scheduler

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/internal/queue"
	"k8s.io/kubernetes/pkg/scheduler/profile"
)

type Cache interface {
	ListNodes() ([]*v1.Node, error)
}

type Scheduler struct {
	Cache     Cache
	Extenders []framework.Extender
	Framework framework.Framework
	NextPod   func() *v1.Pod
	PodQueue  queue.SchedulingQueue
	Profiles  profile.Map
}

func (sched *Scheduler) SchedulePod(ctx context.Context, pod *v1.Pod) (result ScheduleResult, err error) {
	start := time.Now()
	klog.V(3).InfoS("Attempting to schedule pod", "pod", klog.KObj(pod))

	feasibleNodes, err := sched.findNodesThatFit(ctx, pod)
	if err != nil {
		return ScheduleResult{}, err
	}

	if len(feasibleNodes) == 0 {
		return ScheduleResult{}, fmt.Errorf("no nodes available to schedule pod %s/%s", pod.Namespace, pod.Name)
	}

	host, err := sched.selectHost(ctx, pod, feasibleNodes)
	if err != nil {
		return ScheduleResult{}, err
	}

	err = sched.bind(ctx, pod, host)
	if err != nil {
		return ScheduleResult{}, err
	}

	duration := time.Since(start)
	klog.V(2).InfoS("Successfully bound pod to node", "pod", klog.KObj(pod), "node", host, "duration", duration)

	return ScheduleResult{
		SuggestedHost:  host,
		EvaluatedNodes: len(feasibleNodes),
		FeasibleNodes:  len(feasibleNodes),
	}, nil
}

func (sched *Scheduler) findNodesThatFit(ctx context.Context, pod *v1.Pod) ([]string, error) {
	nodes, err := sched.Cache.ListNodes()
	if err != nil {
		return nil, err
	}

	feasibleNodes := []string{}
	for _, node := range nodes {
		fits, err := sched.podFitsOnNode(pod, node)
		if err != nil {
			return nil, err
		}
		if fits {
			feasibleNodes = append(feasibleNodes, node.Name)
		}
	}
	return feasibleNodes, nil
}

func (sched *Scheduler) podFitsOnNode(pod *v1.Pod, node *v1.Node) (bool, error) {
	if node.Status.Allocatable.Cpu().Cmp(pod.Spec.Containers[0].Resources.Requests.Cpu()) < 0 {
		return false, nil
	}

	return true, nil
}

func (sched *Scheduler) selectHost(ctx context.Context, pod *v1.Pod, nodes []string) (string, error) {
	if len(nodes) == 0 {
		return "", fmt.Errorf("no feasible nodes available")
	}
	return nodes[0], nil
}

func (sched *Scheduler) bind(ctx context.Context, pod *v1.Pod, host string) error {
	binding := &v1.Binding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pod.Namespace,
			Name:      pod.Name,
			UID:       pod.UID,
		},
		Target: v1.ObjectReference{
			Kind: "Node",
			Name: host,
		},
	}

	err := sched.Framework.Handle().ClientSet().CoreV1().Pods(pod.Namespace).Bind(ctx, binding, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to bind pod: %v", err)
	}
	return nil
}

func (sched *Scheduler) NextPod() *v1.Pod {
	pod, err := sched.PodQueue.Pop()
	if err != nil {
		klog.ErrorS(err, "Failed to pop pod from scheduling queue")
		return nil
	}
	return pod
}

type ScheduleResult struct {
	SuggestedHost  string
	EvaluatedNodes int
	FeasibleNodes  int
}

func main() {
	ctx := context.Background()

	sched := &Scheduler{
		Cache:     NewCache(),
		Extenders: nil,
		Framework: NewFramework(),
		PodQueue:  queue.NewSchedulingQueue(),
		Profiles:  profile.NewMap(),
	}

	for {
		pod := sched.NextPod()
		if pod == nil {
			time.Sleep(1 * time.Second)
			continue
		}

		_, err := sched.SchedulePod(ctx, pod)
		if err != nil {
			klog.ErrorS(err, "Failed to schedule pod", "pod", klog.KObj(pod))
		}
	}
}
