/*
Copyright (c) 2015 VMware, Inc. All Rights Reserved.

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
	icsgo "github.com/inspur-ics/ics-go-sdk"
	"github.com/inspur-ics/ics-go-sdk/client"
	"k8s.io/klog"
)

// Common contains the fields and functions common to all objects.
type Common struct {
	Con *icsgo.ICSConnection
}

func NewCommon(con *icsgo.ICSConnection) Common {
	return Common{Con: con}
}

func (c Common) Client() *client.Client {
	client, err := c.Con.GetClient()
	if err != nil {
		klog.Errorf("Cannot get  ics token. error: %q", err)
	}
	return client
}
