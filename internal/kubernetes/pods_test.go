package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExtractPodInfo(t *testing.T) {
	tests := []struct {
		name     string
		pod      corev1.Pod
		expected podInfo
	}{
		{
			name: "Running/Ready zero restarts",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "app-abc"},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{Type: "Ready", Status: "True"},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{RestartCount: 0},
					},
				},
			},
			expected: podInfo{Name: "app-abc", Phase: "Running", Ready: true, Restarts: 0},
		},
		{
			// Waiting reason overrides pod phase; CrashLoopBackOff shortened to CrashLoop.
			// No termination reason stored — that comes separately from lastState.
			name: "CrashLoopBackOff",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "app-crash"},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{Type: "Ready", Status: "False"},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{
							RestartCount: 5,
							State:        corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
						},
					},
				},
			},
			expected: podInfo{Name: "app-crash", Phase: "CrashLoop", Ready: false, Reason: "", Restarts: 5},
		},
		{
			// CrashLoopBackOff (waiting) + OOMKilled (lastState): phase = "CrashLoop",
			// reason = "OOMKilled" (the termination detail).
			name: "CrashLoopBackOff with OOMKilled last state",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "app-crash-oom"},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{Type: "Ready", Status: "False"},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{
							RestartCount:         5,
							State:                corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
							LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled"}},
						},
					},
				},
			},
			expected: podInfo{Name: "app-crash-oom", Phase: "CrashLoop", Ready: false, Reason: "OOMKilled", Restarts: 5},
		},
		{
			// OOMKilled in lastState only (no waiting state): pod phase kept, reason = "OOMKilled".
			name: "OOMKilled (last terminated)",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "app-oom"},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{Type: "Ready", Status: "False"},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{
							RestartCount:         3,
							LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled"}},
						},
					},
				},
			},
			expected: podInfo{Name: "app-oom", Phase: "Running", Ready: false, Reason: "OOMKilled", Restarts: 3},
		},
		{
			// "Completed" last-terminated reason is filtered: it means the container exited
			// with code 0 (normal), which is not a useful diagnostic reason.
			name: "Completed last state filtered",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "app-completed"},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{Type: "Ready", Status: "False"},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{
							RestartCount:         11,
							LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Completed"}},
						},
					},
				},
			},
			expected: podInfo{Name: "app-completed", Phase: "Running", Ready: false, Reason: "", Restarts: 11},
		},
		{
			// Waiting reason other than CrashLoopBackOff passes through unchanged and overrides phase.
			name: "ImagePullBackOff",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "app-img"},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					Conditions: []corev1.PodCondition{
						{Type: "Ready", Status: "False"},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{
							RestartCount: 0,
							State:        corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}},
						},
					},
				},
			},
			expected: podInfo{Name: "app-img", Phase: "ImagePullBackOff", Ready: false, Reason: "", Restarts: 0},
		},
		{
			name: "Pending no container statuses",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "app-pending"},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
				},
			},
			expected: podInfo{Name: "app-pending", Phase: "Pending", Ready: false, Restarts: 0},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractPodInfoFromPod(&tc.pod)
			assert.Equal(t, tc.expected, result)
		})
	}
}
