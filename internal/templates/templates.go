package templates

// AppSettingsData represents the data needed for app settings templates
type AppSettingsData struct {
	ComponentName string
	StackName     string
}

// PolicyData represents the data needed for policy templates
type PolicyData struct {
	ComponentName string
	StackName     string
}

// EnvironmentConfigData represents the data needed for environment config template
type EnvironmentConfigData struct {
	EnvironmentName   string
	EnvironmentPrefix string
	StackName         string
}

// EnvironmentData represents the data needed for environment configuration templates
type EnvironmentData struct {
	SubscriptionName  string
	EnvironmentName   string
	EnvironmentPrefix string
	RemoteStateName   string
	ResourceGroup     string
}

// RemoteStateData represents the data needed for remote state configuration
type RemoteStateData struct {
	Name           string
	ResourceGroup  string
	StorageAccount string
	ContainerName  string
}

// RootData represents the data needed for root configuration templates
type RootData struct {
	ProjectName string
	RemoteState RemoteStateData
}
