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
