/*
 * A lightweight Kubernetes controller for automating PersistentVolumeClaim (PVC) snapshots.
 * It allows users to create snapshots on-demand via custom resource policies or periodically
 * according to a schedule.
 *
 * Copyright (C) 2025 Abdul Saqib
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SnapshotPolicy is the Schema for the snapshotpolicies API
type SnapshotPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SnapshotPolicySpec   `json:"spec,omitempty"`
	Status SnapshotPolicyStatus `json:"status,omitempty"`
}

// SnapshotPolicySpec defines the desired state of SnapshotPolicy
type SnapshotPolicySpec struct {
	// PVC to snapshot
	PVCName string `json:"pvcName"`

	// SnapshotClassName is the name of the CSI SnapshotClass to use
	SnapshotClassName string `json:"snapshotClassName"`

	// Schedule in cron format. Optional; if empty, snapshot is created immediately
	Schedule string `json:"schedule,omitempty"`

	// Retention specifies the number of snapshots to keep. Optional
	Retention *int `json:"retention,omitempty"`
}

// SnapshotPolicyStatus defines the observed state of SnapshotPolicy
type SnapshotPolicyStatus struct {
	// LastSnapshotTime indicates when the last snapshot was created
	LastSnapshotTime *metav1.Time `json:"lastSnapshotTime,omitempty"`

	// Phase indicates the current state of snapshot operation: Pending, InProgress, Succeeded, Failed
	Phase string `json:"phase,omitempty"`

	// Message provides additional info about errors or status
	Message string `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SnapshotPolicyList contains a list of SnapshotPolicy
type SnapshotPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SnapshotPolicy `json:"items"`
}
