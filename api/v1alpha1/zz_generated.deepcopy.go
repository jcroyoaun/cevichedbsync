//go:build !ignore_autogenerated

/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CredentialReference) DeepCopyInto(out *CredentialReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CredentialReference.
func (in *CredentialReference) DeepCopy() *CredentialReference {
	if in == nil {
		return nil
	}
	out := new(CredentialReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatabaseServiceReference) DeepCopyInto(out *DatabaseServiceReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatabaseServiceReference.
func (in *DatabaseServiceReference) DeepCopy() *DatabaseServiceReference {
	if in == nil {
		return nil
	}
	out := new(DatabaseServiceReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresSync) DeepCopyInto(out *PostgresSync) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresSync.
func (in *PostgresSync) DeepCopy() *PostgresSync {
	if in == nil {
		return nil
	}
	out := new(PostgresSync)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PostgresSync) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresSyncList) DeepCopyInto(out *PostgresSyncList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PostgresSync, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresSyncList.
func (in *PostgresSyncList) DeepCopy() *PostgresSyncList {
	if in == nil {
		return nil
	}
	out := new(PostgresSyncList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PostgresSyncList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresSyncSpec) DeepCopyInto(out *PostgresSyncSpec) {
	*out = *in
	out.StatefulSetRef = in.StatefulSetRef
	out.DatabaseService = in.DatabaseService
	out.GitCredentials = in.GitCredentials
	out.DatabaseCredentials = in.DatabaseCredentials
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresSyncSpec.
func (in *PostgresSyncSpec) DeepCopy() *PostgresSyncSpec {
	if in == nil {
		return nil
	}
	out := new(PostgresSyncSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresSyncStatus) DeepCopyInto(out *PostgresSyncStatus) {
	*out = *in
	in.LastSyncTime.DeepCopyInto(&out.LastSyncTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresSyncStatus.
func (in *PostgresSyncStatus) DeepCopy() *PostgresSyncStatus {
	if in == nil {
		return nil
	}
	out := new(PostgresSyncStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StatefulSetReference) DeepCopyInto(out *StatefulSetReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StatefulSetReference.
func (in *StatefulSetReference) DeepCopy() *StatefulSetReference {
	if in == nil {
		return nil
	}
	out := new(StatefulSetReference)
	in.DeepCopyInto(out)
	return out
}
