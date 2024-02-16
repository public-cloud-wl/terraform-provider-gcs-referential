package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

type GcpConnector struct {
	BucketName    string
	BaseCidrRange string
	FileName      string
	generation    int64
}

type NetworkConfig struct {
	Subnets map[string]string `json:"subnets"`
}

func New(bucketName string, baseCidr string) GcpConnector {
	fileName := fmt.Sprintf("cidr-reservation/baseCidr-%s.json", strings.Replace(strings.Replace(baseCidr, ".", "-", -1), "/", "-", -1))
	return GcpConnector{bucketName, baseCidr, fileName, -1}
}

func getStorageClient(ctx context.Context) (*storage.Client, error) {
	access_token := os.Getenv("GOOGLE_OAUTH_ACCESS_TOKEN")
	if access_token != "" {
		var tokenSource oauth2.TokenSource
		var credOptions []option.ClientOption
		tokenSource = oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: access_token,
		})
		credOptions = append(credOptions, option.WithTokenSource(tokenSource))
		return storage.NewClient(ctx, credOptions...)
	} else {
		return storage.NewClient(ctx)
	}
}

func (gcp *GcpConnector) ReadRemote(ctx context.Context) (*NetworkConfig, error) {
	// Creates a client.
	networkConfig := NetworkConfig{}
	client, err := getStorageClient(ctx)
	if err != nil {
		return &networkConfig, err
	}
	defer client.Close()

	// Creates a Bucket instance.
	bucket := client.Bucket(gcp.BucketName)
	if err != nil {
		return nil, err
	}
	objectHandle := bucket.Object(gcp.FileName)
	attrs, err := objectHandle.Attrs(ctx)
	if err == nil {
		gcp.generation = attrs.Generation
	}
	rc, err := objectHandle.NewReader(ctx)
	if err != nil {
		return &networkConfig, err
	}
	defer rc.Close()
	slurp, err := io.ReadAll(rc)
	if err != nil {
		return &networkConfig, err
	}
	if err := json.Unmarshal(slurp, &networkConfig); err != nil {
		return &networkConfig, err
	}
	return &networkConfig, nil
}

func (gcp *GcpConnector) WriteRemote(networkConfig *NetworkConfig, ctx context.Context) error {
	// Creates a client.
	client, err := getStorageClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	// Creates a Bucket instance.
	bucket := client.Bucket(gcp.BucketName)
	var writer *storage.Writer
	if gcp.generation == -1 {
		writer = bucket.Object(gcp.FileName).If(storage.Conditions{DoesNotExist: true}).NewWriter(ctx)
	} else {
		writer = bucket.Object(gcp.FileName).If(storage.Conditions{GenerationMatch: gcp.generation}).NewWriter(ctx)
	}
	marshalled, err := json.Marshal(networkConfig)
	if err != nil {
		return err
	}
	_, _ = writer.Write(marshalled)
	if err := writer.Close(); err != nil {
		tflog.Error(ctx, "Failed to write file to GCP", map[string]interface{}{"error": err, "generation": gcp.generation})
		return err
	}
	return nil
}

func readNetsegmentJson(ctx context.Context, cidrProviderBucket string, netsegmentName string) (NetworkConfig, error) {
	return NetworkConfig{}, nil
	//return readRemote(cidrProviderBucket, fmt.Sprintf("gcp-cidr-provider/%s.json", netsegmentName), ctx)
}

// TODO: implement!
func uploadNewNetsegmentJson() {}
