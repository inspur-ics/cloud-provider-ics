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

	icfg "github.com/inspur-ics/cloud-provider-ics/pkg/common/config"
	cm "github.com/inspur-ics/cloud-provider-ics/pkg/common/credentialmanager"
	k8s "github.com/inspur-ics/cloud-provider-ics/pkg/common/kubernetes"
	icsgo "github.com/inspur-ics/ics-go-sdk"
)

// NewConnectionManager returns a new ConnectionManager object
// This function also initializes the Default/Global lister for secrets. In other words,
// If a single global secret is used for all VCs, the informMgr param will be used to
// obtain those secrets
func NewConnectionManager(cfg *icfg.Config, informMgr *k8s.InformerManager, client clientset.Interface) *ConnectionManager {
	connMgr := &ConnectionManager{
		client:             client,
		ICSInstanceMap: generateInstanceMap(cfg),
		credentialManagers: make(map[string]*cm.CredentialManager),
		informerManagers:   make(map[string]*k8s.InformerManager),
	}

	if informMgr != nil {
		klog.V(2).Info("Initializing with K8s SecretLister")
		credMgr := cm.NewCredentialManager(cfg.Global.SecretName, cfg.Global.SecretNamespace, "", informMgr.GetSecretLister())
		connMgr.credentialManagers[icfg.DefaultCredentialManager] = credMgr
		connMgr.informerManagers[icfg.DefaultCredentialManager] = informMgr

		return connMgr
	}

	if cfg.Global.SecretsDirectory != "" {
		klog.V(2).Info("Initializing for generic CO with secrets")
		credMgr, _ := connMgr.createManagersPerTenant("", "", cfg.Global.SecretsDirectory, nil)
		connMgr.credentialManagers[icfg.DefaultCredentialManager] = credMgr

		return connMgr
	}

	klog.V(2).Info("Initializing generic CO")
	credMgr := cm.NewCredentialManager("", "", "", nil)
	connMgr.credentialManagers[icfg.DefaultCredentialManager] = credMgr

	return connMgr
}

// generateInstanceMap creates a map of iCenter connection objects that can be
// use to create a connection to a iCenter using vclib package
func generateInstanceMap(cfg *icfg.Config) map[string]*ICSInstance {
	icsInstanceMap := make(map[string]*ICSInstance)
	for _, icsConfig := range cfg.ICSCenter {
		icsConn := icsgo.ICSConnection{
			Username:          icsConfig.User,
			Password:          icsConfig.Password,
			Hostname:          icsConfig.ICenterIP,
			Insecure:          icsConfig.InsecureFlag,
			Port:              icsConfig.ICenterPort,
		}
		icsIns := ICSInstance{
			Conn: &icsConn,
			Cfg:  icsConfig,
		}
		icsInstanceMap[icsConfig.TenantRef] = &icsIns
	}

	return icsInstanceMap
}

// InitializeSecretLister initializes the individual secret listers that are NOT
// handled through the Default/Global lister tied to the default service account.
func (connMgr *ConnectionManager) InitializeSecretLister() {
	// For each vsi that has a Secret set createManagersPerTenant
	for _, instance := range connMgr.ICSInstanceMap {
		klog.V(3).Infof("Checking icsServer=%s SecretRef=%s", instance.Cfg.ICenterIP, instance.Cfg.SecretRef)
		if strings.EqualFold(instance.Cfg.SecretRef, icfg.DefaultCredentialManager) {
			klog.V(3).Infof("Skipping. iCenter %s is configured using global service account/secret.", instance.Cfg.ICenterIP)
			continue
		}

		klog.V(3).Infof("Adding credMgr/informMgr for icsServer=%s", instance.Cfg.ICenterIP)
		credsMgr, informMgr := connMgr.createManagersPerTenant(instance.Cfg.SecretName,
			instance.Cfg.SecretNamespace, "", connMgr.client)
		connMgr.credentialManagers[instance.Cfg.SecretRef] = credsMgr
		connMgr.informerManagers[instance.Cfg.SecretRef] = informMgr
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
func (connMgr *ConnectionManager) Connect(ctx context.Context, icsInstance *ICSInstance) error {
	connMgr.Lock()
	defer connMgr.Unlock()
	err := icsInstance.Conn.Connect(ctx)
	if err == nil {
		return nil
	}

	if err != nil || connMgr.credentialManagers == nil {
		klog.Errorf("Cannot connect to iCenter with err: %v", err)
		return err
	}

	klog.V(2).Infof("Invalid credentials. Fetching credentials from secrets. icsServer=%s credentialHolder=%s",
		icsInstance.Cfg.ICenterIP, icsInstance.Cfg.SecretRef)

	credMgr := connMgr.credentialManagers[icsInstance.Cfg.SecretRef]
	if credMgr == nil {
		klog.Errorf("Unable to find credential manager for icsServer=%s credentialHolder=%s", icsInstance.Cfg.ICenterIP, icsInstance.Cfg.SecretRef)
		return ErrUnableToFindCredentialManager
	}
	credentials, err := credMgr.GetCredential(icsInstance.Cfg.ICenterIP)
	if err != nil {
		klog.Error("Failed to get credentials from Secret Credential Manager with err:", err)
		return err
	}
	icsInstance.Conn.UpdateCredentials(credentials.User, credentials.Password)
	return icsInstance.Conn.Connect(ctx)
}

// Logout closes existing connections to remote iCenter endpoints.
func (connMgr *ConnectionManager) Logout() {
	for _, icsIns := range connMgr.ICSInstanceMap {
		connMgr.Lock()
		c := icsIns.Conn.Client
		connMgr.Unlock()
		if c != nil {
			icsIns.Conn.Logout(context.TODO())
		}
	}
}

// Verify validates the configuration by attempting to connect to the
// configured, remote iCenter endpoints.
func (connMgr *ConnectionManager) Verify() error {
	for _, icsInstance := range connMgr.ICSInstanceMap {
		err := connMgr.Connect(context.Background(), icsInstance)
		if err == nil {
			klog.V(3).Infof("iCenter connect %s succeeded.", icsInstance.Cfg.ICenterIP)
		} else {
			klog.Errorf("iCenter %s failed. Err: %q", icsInstance.Cfg.ICenterIP, err)
			return err
		}
	}
	return nil
}

// VerifyWithContext is the same as Verify but allows a Go Context
// to control the lifecycle of the connection event.
func (connMgr *ConnectionManager) VerifyWithContext(ctx context.Context) error {
	for _, icsInstance := range connMgr.ICSInstanceMap {
		err := connMgr.Connect(ctx, icsInstance)
		if err == nil {
			klog.V(3).Infof("iCenter connect %s succeeded.", icsInstance.Cfg.ICenterIP)
		} else {
			klog.Errorf("iCenter %s failed. Err: %q", icsInstance.Cfg.ICenterIP, err)
			return err
		}
	}
	return nil
}

// APIVersion returns the version of the iCenter API
func (connMgr *ConnectionManager) APIVersion(icsInstance *ICSInstance) (string, error) {
	if err := connMgr.Connect(context.Background(), icsInstance); err != nil {
		return "", err
	}
	//FIXME TODO.
	return "5.6", nil
}
