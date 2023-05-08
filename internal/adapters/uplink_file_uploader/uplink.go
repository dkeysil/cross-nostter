package uplink

import (
	"context"
	"fmt"
	"io"
	"strings"

	"storj.io/common/uuid"
	"storj.io/uplink"
	"storj.io/uplink/edge"
)

type Config struct {
	AccessGrant string `envconfig:"ACCESS_GRANT"`
	BucketName  string `envconfig:"BUCKET_NAME" default:"dev-cross-nostter"`
}

type UplinkFileUploader struct {
	project    *uplink.Project
	access     *uplink.Access
	bucketName string
}

func NewUplinkFileUploader(ctx context.Context, cfg *Config) (*UplinkFileUploader, error) {
	// Request access grant to the satellite with the API key and passphrase.
	access, err := uplink.ParseAccess(cfg.AccessGrant)
	if err != nil {
		return nil, fmt.Errorf("could not request access grant: %v", err)
	}

	// Open up the Project we will be working with.
	project, err := uplink.OpenProject(ctx, access)
	if err != nil {
		return nil, fmt.Errorf("could not open project: %v", err)
	}
	defer project.Close()

	// Ensure the desired Bucket within the Project is created.
	_, err = project.EnsureBucket(ctx, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("could not ensure bucket: %v", err)
	}

	return &UplinkFileUploader{
		access:     access,
		project:    project,
		bucketName: cfg.BucketName,
	}, nil
}

func (u *UplinkFileUploader) UploadFile(ctx context.Context, fileName string, file io.Reader) (url string, err error) {
	id, err := uuid.New()
	if err != nil {
		return "", fmt.Errorf("could not generate file name: %v", err)
	}

	splittedFileName := strings.Split(fileName, ".")
	if len(splittedFileName) < 2 {
		return "", fmt.Errorf("invalid file name: %s", fileName)
	}
	fileName = fmt.Sprintf("%s.%s", id.String(), splittedFileName[len(splittedFileName)-1])

	upload, err := u.project.UploadObject(ctx, u.bucketName, fileName, nil)
	if err != nil {
		return "", fmt.Errorf("could not initiate upload: %v", err)
	}

	_, err = io.Copy(upload, file)
	if err != nil {
		_ = upload.Abort()
		return "", fmt.Errorf("could not upload data: %v", err)
	}

	err = upload.Commit()
	if err != nil {
		return "", fmt.Errorf("could not commit uploaded object: %v", err)
	}

	c := &edge.Config{
		AuthServiceAddress: "auth.storjshare.io:7777",
	}
	access, err := u.access.Share(
		uplink.ReadOnlyPermission(),
		uplink.SharePrefix{Bucket: u.bucketName, Prefix: fileName},
	)
	if err != nil {
		return "", fmt.Errorf("could not share access: %v", err)
	}

	creds, err := c.RegisterAccess(ctx, access, &edge.RegisterAccessOptions{Public: true})
	if err != nil {
		return "", fmt.Errorf("could not register access: %v", err)
	}

	url, err = edge.JoinShareURL(
		"https://link.storjshare.io",
		creds.AccessKeyID,
		u.bucketName,
		fileName,
		&edge.ShareURLOptions{Raw: true},
	)

	return
}
