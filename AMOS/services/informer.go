package services

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Informer(clientset *kubernetes.Clientset) {

	ctx := context.TODO()

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "amos",
			Namespace: "amos",
		},
	}
	// Error handling ignored as per original code style, ideally should log.
	_, _ = clientset.CoreV1().ServiceAccounts("amos").Create(ctx, sa, metav1.CreateOptions{})

	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "amos",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "services", "configmaps", "secrets"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
		},
	}
	_, _ = clientset.RbacV1().ClusterRoles().Create(ctx, role, metav1.CreateOptions{})

	roleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "amos",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "amos",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "amos",
				Namespace: "amos",
			},
		},
	}
	_, _ = clientset.RbacV1().ClusterRoleBindings().Create(ctx, roleBinding, metav1.CreateOptions{})

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "amos",
			Namespace: "amos",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "amos",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "amos",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "amos",
					Containers: []corev1.Container{
						{
							Name:  "amos",
							Image: "amos:latest",
						},
					},
				},
			},
		},
	}
	_, _ = clientset.AppsV1().Deployments("amos").Create(ctx, deployment, metav1.CreateOptions{})

}
