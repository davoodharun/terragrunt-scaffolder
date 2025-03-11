package azure

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// CreateContainer creates a new container in the specified storage account
func CreateContainer(storageAccountName, containerName string) error {
	// Get the storage account key from environment variable
	storageAccountKey := os.Getenv("AZURE_STORAGE_KEY")
	if storageAccountKey == "" {
		return fmt.Errorf("AZURE_STORAGE_KEY environment variable is not set")
	}

	// Create a credential object using the storage account key
	cred, err := azblob.NewSharedKeyCredential(storageAccountName, storageAccountKey)
	if err != nil {
		return fmt.Errorf("failed to create shared key credential: %w", err)
	}

	// Create a service client
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", storageAccountName)
	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}

	// Create the container
	_, err = client.CreateContainer(context.Background(), containerName, nil)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	return nil
}
