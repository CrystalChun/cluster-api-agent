//go:build !ignore_autogenerated

/*
Copyright 2020.

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

package common

import ()

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ValidationResult) DeepCopyInto(out *ValidationResult) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ValidationResult.
func (in *ValidationResult) DeepCopy() *ValidationResult {
	if in == nil {
		return nil
	}
	out := new(ValidationResult)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in ValidationResults) DeepCopyInto(out *ValidationResults) {
	{
		in := &in
		*out = make(ValidationResults, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ValidationResults.
func (in ValidationResults) DeepCopy() ValidationResults {
	if in == nil {
		return nil
	}
	out := new(ValidationResults)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in ValidationsStatus) DeepCopyInto(out *ValidationsStatus) {
	{
		in := &in
		*out = make(ValidationsStatus, len(*in))
		for key, val := range *in {
			var outVal []ValidationResult
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make(ValidationResults, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ValidationsStatus.
func (in ValidationsStatus) DeepCopy() ValidationsStatus {
	if in == nil {
		return nil
	}
	out := new(ValidationsStatus)
	in.DeepCopyInto(out)
	return *out
}
