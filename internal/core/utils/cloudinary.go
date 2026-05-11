package utils

import (
	"context"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type Cloudinary interface {
	UploadFile(filePath string) (string, error)
}

type cloudinaryUtil struct {
	client *cloudinary.Cloudinary
}

// NewCloudinaryUtil now accepts the pointer from your main.go
func NewCloudinaryUtil(cld *cloudinary.Cloudinary) Cloudinary {
	return &cloudinaryUtil{client: cld}
}

func (c *cloudinaryUtil) UploadFile(filePath string) (string, error) {
	ctx := context.Background()

	// Use c.client.Upload.Upload to avoid the 'undefined' error
	// in newer versions of the Cloudinary SDK.
	resp, err := c.client.Upload.Upload(ctx, filePath, uploader.UploadParams{
		Folder:       "contracts",
		ResourceType: "raw", // MUST be raw for PDF files
	})

	if err != nil {
		return "", err
	}

	// Important: Delete the temp file from your server after successful upload
	os.Remove(filePath)

	return resp.SecureURL, nil
}
