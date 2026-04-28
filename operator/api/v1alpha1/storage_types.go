package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=LOCAL;S3;GOOGLE_DRIVE;NAS;AZURE_BLOB;FTP;SFTP;RCLONE
type StorageType string

const (
	StorageTypeLocal       StorageType = "LOCAL"
	StorageTypeS3          StorageType = "S3"
	StorageTypeGoogleDrive StorageType = "GOOGLE_DRIVE"
	StorageTypeNAS         StorageType = "NAS"
	StorageTypeAzureBlob   StorageType = "AZURE_BLOB"
	StorageTypeFTP         StorageType = "FTP"
	StorageTypeSFTP        StorageType = "SFTP"
	StorageTypeRclone      StorageType = "RCLONE"
)

type StorageSpec struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	Type StorageType `json:"type"`

	S3          *S3StorageSpec          `json:"s3,omitempty"`
	SFTP        *SFTPStorageSpec        `json:"sftp,omitempty"`
	AzureBlob   *AzureBlobStorageSpec   `json:"azureBlob,omitempty"`
	Local       *LocalStorageSpec       `json:"local,omitempty"`
	FTP         *FTPStorageSpec         `json:"ftp,omitempty"`
	Rclone      *RcloneStorageSpec      `json:"rclone,omitempty"`
	NAS         *NASStorageSpec         `json:"nas,omitempty"`
	GoogleDrive *GoogleDriveStorageSpec `json:"googleDrive,omitempty"`
}

type S3StorageSpec struct {
	Bucket             string       `json:"bucket"`
	Region             string       `json:"region"`
	Endpoint           string       `json:"endpoint,omitempty"`
	Prefix             string       `json:"prefix,omitempty"`
	AccessKeySecretRef SecretKeyRef `json:"accessKeySecretRef"`
	SecretKeySecretRef SecretKeyRef `json:"secretKeySecretRef"`

	// +kubebuilder:default=false
	IsUseVirtualHostedStyle bool `json:"isUseVirtualHostedStyle,omitempty"`
	// +kubebuilder:default=false
	IsSkipTLSVerify bool `json:"isSkipTLSVerify,omitempty"`
	// +kubebuilder:validation:Enum="";STANDARD;STANDARD_IA;ONEZONE_IA;INTELLIGENT_TIERING;REDUCED_REDUNDANCY;GLACIER_IR
	StorageClass string `json:"storageClass,omitempty"`
}

type SFTPStorageSpec struct {
	Host string `json:"host"`
	// +kubebuilder:default=22
	Port                int           `json:"port,omitempty"`
	Username            string        `json:"username"`
	PasswordSecretRef   *SecretKeyRef `json:"passwordSecretRef,omitempty"`
	PrivateKeySecretRef *SecretKeyRef `json:"privateKeySecretRef,omitempty"`
	Path                string        `json:"path,omitempty"`
	// +kubebuilder:default=false
	IsSkipHostKeyVerify bool `json:"isSkipHostKeyVerify,omitempty"`
}

type AzureBlobStorageSpec struct {
	// +kubebuilder:validation:Enum=CONNECTION_STRING;ACCOUNT_KEY
	AuthMethod                string        `json:"authMethod"`
	ConnectionStringSecretRef *SecretKeyRef `json:"connectionStringSecretRef,omitempty"`
	AccountName               string        `json:"accountName,omitempty"`
	AccountKeySecretRef       *SecretKeyRef `json:"accountKeySecretRef,omitempty"`
	ContainerName             string        `json:"containerName"`
	Endpoint                  string        `json:"endpoint,omitempty"`
	Prefix                    string        `json:"prefix,omitempty"`
}

type LocalStorageSpec struct {
	Path string `json:"path"`
}

type FTPStorageSpec struct {
	Host string `json:"host"`
	// +kubebuilder:default=21
	Port              int          `json:"port,omitempty"`
	Username          string       `json:"username"`
	PasswordSecretRef SecretKeyRef `json:"passwordSecretRef"`
	Path              string       `json:"path,omitempty"`
	// +kubebuilder:default=false
	IsUseSSL bool `json:"isUseSsl,omitempty"`
	// +kubebuilder:default=false
	IsSkipTLSVerify bool `json:"isSkipTlsVerify,omitempty"`
}

type RcloneStorageSpec struct {
	ConfigContentSecretRef SecretKeyRef `json:"configContentSecretRef"`
	RemotePath             string       `json:"remotePath,omitempty"`
}

type NASStorageSpec struct {
	Host string `json:"host"`
	// +kubebuilder:default=445
	Port              int          `json:"port,omitempty"`
	Share             string       `json:"share"`
	Username          string       `json:"username"`
	PasswordSecretRef SecretKeyRef `json:"passwordSecretRef"`
	// +kubebuilder:default=false
	IsUseSSL bool   `json:"isUseSsl,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Path     string `json:"path,omitempty"`
}

type GoogleDriveStorageSpec struct {
	ClientIDSecretRef     SecretKeyRef `json:"clientIdSecretRef"`
	ClientSecretSecretRef SecretKeyRef `json:"clientSecretSecretRef"`
	TokenJSONSecretRef    SecretKeyRef `json:"tokenJsonSecretRef"`
}

type StorageStatus struct {
	// ID assigned by the databasus API.
	ID string `json:"id,omitempty"`
	// ObservedGeneration is the most recent generation observed.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Storage is the Schema for the storages API.
type Storage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageSpec   `json:"spec"`
	Status StorageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StorageList contains a list of Storage.
type StorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Storage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Storage{}, &StorageList{})
}
