package icslib

// StoragePod extends the govmomi StoragePod object
type StoragePod struct {
	Datacenter *Datacenter
	//*object.StoragePod
	Datastores []*Datastore
}

// StoragePodInfo is a structure to store the StoragePod and it's Info.
type StoragePodInfo struct {
	*StoragePod
	//Summary        *types.StoragePodSummary
	//Config         *types.StorageDrsConfigInfo
	DatastoreInfos []*DatastoreInfo
}