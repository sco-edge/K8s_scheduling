// incomplete
package scheduler

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/internal/queue"
	"k8s.io/kubernetes/pkg/scheduler/profile"
)

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

	return true, nil
}

func (sched *Scheduler) selectHost(ctx context.Context, pod *v1.Pod, nodes []string) (string, error) {
	if len(nodes) == 0 {
		return "", fmt.Errorf("no feasible nodes available")
	}
	return nodes[0], nil
}

func (sched *Scheduler) bind(ctx context.Context, pod *v1.Pod, host string) error {
	return nil
}

func main() {
	//incomplete
}
