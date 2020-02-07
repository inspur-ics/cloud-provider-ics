/*
Copyright 2018 The Kubernetes Authors.

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

package icslib

import (
	"github.com/inspur-ics/ics-go-sdk/client/types"
)

// Datastore extends the govmomi Datastore object
type Datastore struct {
	Common
	*types.Storage
	Datacenter *Datacenter
}

// DatastoreInfo is a structure to store the Datastore and it's Info.
type DatastoreInfo struct {
	*Datastore
}