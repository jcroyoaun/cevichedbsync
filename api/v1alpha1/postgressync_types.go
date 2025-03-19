package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PostgresSyncSpec defines the desired state of PostgresSync
type PostgresSyncSpec struct {
	// StatefulSetRef points to the StatefulSet that this sync watches
	StatefulSetRef StatefulSetReference `json:"statefulSetRef"`

	// DatabaseService specifies the service and namespace to connect to the database
	DatabaseService DatabaseServiceReference `json:"databaseService"`

	// RepositoryURL is the Git repository URL where dumps will be stored
	RepositoryURL string `json:"repositoryURL"`

	// databaseDumpPath specifies the path within the Git repository where dumps should be stored
	// +optional
	DatabaseDumpPath string `json:"databaseDumpPath,omitempty"`

	// GitCredentials contains authentication information for Git
	GitCredentials CredentialReference `json:"gitCredentials"`

	// DatabaseCredentials contains authentication information for the database
	DatabaseCredentials CredentialReference `json:"databaseCredentials"`

	// DumpOnWebhook triggers a database dump when set to true
	// +optional
	DumpOnWebhook bool `json:"dumpOnWebhook,omitempty"`
}

// DatabaseServiceReference defines the service and namespace for database connection
type DatabaseServiceReference struct {
	// Name is the service name
	Name string `json:"name"`

	// Namespace is the namespace of the service
	// If empty, the PostgresSync namespace will be used
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// StatefulSetReference identifies the StatefulSet being watched
type StatefulSetReference struct {
	// Name is the name of the StatefulSet to watch
	Name string `json:"name"`
}

// CredentialReference identifies where credentials are stored
type CredentialReference struct {
	// SecretName is the name of the Secret containing credentials
	SecretName string `json:"secretName"`
}

// PostgresSyncStatus defines the observed state of PostgresSync
type PostgresSyncStatus struct {
	// Phase shows the current phase of the PostgresSync operation
	Phase string `json:"phase,omitempty"`

	// Message contains a human-readable message explaining the current status
	Message string `json:"message,omitempty"`

	// LastSyncTime is the timestamp of the last successful dump
	// +optional
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message"
//+kubebuilder:printcolumn:name="Last Sync",type="date",JSONPath=".status.lastSyncTime"

// PostgresSync is the Schema for the postgressyncs API
type PostgresSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresSyncSpec   `json:"spec,omitempty"`
	Status PostgresSyncStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PostgresSyncList contains a list of PostgresSync
type PostgresSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresSync `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresSync{}, &PostgresSyncList{})
}
