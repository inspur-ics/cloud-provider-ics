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

package connectionmanager

import (
	"context"
	"strings"

	clientset "k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"

	vcfg "github.com/inspur-ics/cloud-provider-ics/pkg/common/config"
	cm "github.com/inspur-ics/cloud-provider-ics/pkg/common/credentialmanager"
	k8s "github.com/inspur-ics/cloud-provider-ics/pkg/common/kubernetes"
	icslib "github.com/inspur-ics/cloud-provider-ics/pkg/common/icslib"
)

// NewConnectionManager returns a new ConnectionManager object
// This function also initializes the Default/Global lister for secrets. In other words,
// If a single global secret is used for all VCs, the informMgr param will be used to
// obtain those secrets
func NewConnectionManager(cfg *vcfg.Config, informMgr *k8s.InformerManager, client clientset.Interface) *ConnectionManager {
	connMgr := &ConnectionManager{
		client:             client,
		IcsInstanceMap: generateInstanceMap(cfg),
		credentialManagers: make(map[string]*cm.CredentialManager),
		informerManagers:   make(map[string]*k8s.InformerManager),
	}

	if informMgr != nil {
		klog.V(2).Info("Initializing with K8s SecretLister")
		credMgr := cm.NewCredentialManager(cfg.Global.SecretName, cfg.Global.SecretNamespace, "", informMgr.GetSecretLister())
		connMgr.credentialManagers[vcfg.DefaultCredentialManager] = credMgr
		connMgr.informerManagers[vcfg.DefaultCredentialManager] = informMgr

		return connMgr
	}

	if cfg.Global.SecretsDirectory != "" {
		klog.V(2).Info("Initializing for generic CO with secrets")
		credMgr, _ := connMgr.createManagersPerTenant("", "", cfg.Global.SecretsDirectory, nil)
		connMgr.credentialManagers[vcfg.DefaultCredentialManager] = credMgr

		return connMgr
	}

	klog.V(2).Info("Initializing generic CO")
	credMgr := cm.NewCredentialManager("", "", "", nil)
	connMgr.credentialManagers[vcfg.DefaultCredentialManager] = credMgr

	return connMgr
}

// generateInstanceMap creates a map of iCenter connection objects that can be
// use to create a connection to a iCenter using icslib package
func generateInstanceMap(cfg *vcfg.Config) map[string]*ICsInstance {
	icsInstanceMap := make(map[string]*ICsInstance)
//ics
	for _, vcConfig := range cfg.VirtualCenter {
		icsConn := icslib.ICsConnection{
			Username:          vcConfig.User,
			Password:          vcConfig.Password,
			Hostname:          vcConfig.VCenterIP,
			RoundTripperCount: vcConfig.RoundTripperCount,
			Port:              vcConfig.VCenterPort,
			Thumbprint:        vcConfig.Thumbprint,
		}
		icsIns := ICsInstance{
			Conn: &icsConn,
			Cfg:  vcConfig,
		}
		icsInstanceMap[vcConfig.TenantRef] = &icsIns
	}

	return icsInstanceMap
}

// InitializeSecretLister initializes the individual secret listers that are NOT
// handled through the Default/Global lister tied to the default service account.
func (connMgr *ConnectionManager) InitializeSecretLister() {
	// For each vsi that has a Secret set createManagersPerTenant
	for _, vInstance := range connMgr.IcsInstanceMap {
		klog.V(3).Infof("Checking vcServer=%s SecretRef=%s", vInstance.Cfg.VCenterIP, vInstance.Cfg.SecretRef)
		if strings.EqualFold(vInstance.Cfg.SecretRef, vcfg.DefaultCredentialManager) {
			klog.V(3).Infof("Skipping. iCenter %s is configured using global service account/secret.", vInstance.Cfg.VCenterIP)
			continue
		}

		klog.V(3).Infof("Adding credMgr/informMgr for vcServer=%s", vInstance.Cfg.VCenterIP)
		credsMgr, informMgr := connMgr.createManagersPerTenant(vInstance.Cfg.SecretName,
			vInstance.Cfg.SecretNamespace, "", connMgr.client)
		connMgr.credentialManagers[vInstance.Cfg.SecretRef] = credsMgr
		connMgr.informerManagers[vInstance.Cfg.SecretRef] = informMgr
	}
}

func (connMgr *ConnectionManager) createManagersPerTenant(secretName string, secretNamespace string,
	secretsDirectory string, client clientset.Interface) (*cm.CredentialManager, *k8s.InformerManager) {

	var informMgr *k8s.InformerManager
	var lister listerv1.SecretLister
	if client != nil && secretsDirectory == "" {
		informMgr = k8s.NewInformer(client, true)
		lister = informMgr.GetSecretLister()
	}

	credMgr := cm.NewCredentialManager(secretName, secretNamespace, secretsDirectory, lister)

	if lister != nil {
		informMgr.Listen()
	}

	return credMgr, informMgr
}

// Connect connects to iCenter with existing credentials
// If credentials are invalid:
// 		1. It will fetch credentials from credentialManager
//      2. Update the credentials
//		3. Connects again to iCenter with fetched credentials
func (connMgr *ConnectionManager) Connect(ctx context.Context, vcInstance *ICsInstance) error {
	connMgr.Lock()
	defer connMgr.Unlock()

	err := vcInstance.Conn.Connect(ctx)
	if err == nil {
		return nil
	}
//ics
	if !icslib.IsInvalidCredentialsError(err) || connMgr.credentialManagers == nil {
		klog.Errorf("Cannot connect to iCenter with err: %v", err)
		return err
	}
//ics
	klog.V(2).Infof("Invalid credentials. Fetching credentials from secrets. vcServer=%s credentialHolder=%s",
		vcInstance.Cfg.VCenterIP, vcInstance.Cfg.SecretRef)

	credMgr := connMgr.credentialManagers[vcInstance.Cfg.SecretRef]
	if credMgr == nil {
		klog.Errorf("Unable to find credential manager for vcServer=%s credentialHolder=%s", vcInstance.Cfg.VCenterIP, vcInstance.Cfg.SecretRef)
		return ErrUnableToFindCredentialManager
	}
	credentials, err := credMgr.GetCredential(vcInstance.Cfg.VCenterIP)
	if err != nil {
		klog.Error("Failed to get credentials from Secret Credential Manager with err:", err)
		return err
	}
	vcInstance.Conn.UpdateCredentials(credentials.User, credentials.Password)
	return vcInstance.Conn.Connect(ctx)
}

// Logout closes existing connections to remote iCenter endpoints.
func (connMgr *ConnectionManager) Logout() {
//ics
	for _, icsIns := range connMgr.IcsInstanceMap {
		connMgr.Lock()
		c := icsIns.Conn.Client
		connMgr.Unlock()
//		if c != nil {
        if len(c) != 0{
			icsIns.Conn.Logout(context.TODO())
		}
    }
//ics
}

// Verify validates the configuration by attempting to connect to the
// configured, remote iCenter endpoints.
func (connMgr *ConnectionManager) Verify() error {
	for _, vcInstance := range connMgr.IcsInstanceMap {
		err := connMgr.Connect(context.Background(), vcInstance)
		if err == nil {
			klog.V(3).Infof("iCenter connect %s succeeded.", vcInstance.Cfg.VCenterIP)
		} else {
			klog.Errorf("iCenter %s failed. Err: %q", vcInstance.Cfg.VCenterIP, err)
			return err
		}
	}
	return nil
}

// VerifyWithContext is the same as Verify but allows a Go Context
// to control the lifecycle of the connection event.
func (connMgr *ConnectionManager) VerifyWithContext(ctx context.Context) error {
	for _, vcInstance := range connMgr.IcsInstanceMap {
		err := connMgr.Connect(ctx, vcInstance)
		if err == nil {
			klog.V(3).Infof("iCenter connect %s succeeded.", vcInstance.Cfg.VCenterIP)
		} else {
			klog.Errorf("iCenter %s failed. Err: %q", vcInstance.Cfg.VCenterIP, err)
			return err
		}
	}
	return nil
}

// APIVersion returns the version of the iCenter API
func (connMgr *ConnectionManager) APIVersion(vcInstance *ICsInstance) (string, error) {
	if err := connMgr.Connect(context.Background(), vcInstance); err != nil {
		return "", err
	}
//ics
//	return vcInstance.Conn.Client.ServiceContent.About.ApiVersion, nil
        return "v1", nil	
//ics
}
