package connector

import (
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"io"
	"strings"
	"time"
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

func (gcp *GcpConnector) ReadRemote(ctx context.Context) (*NetworkConfig, error) {
	// Creates a client.
	networkConfig := NetworkConfig{}
	client, err := storage.NewClient(ctx)
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
	client, err := storage.NewClient(ctx)
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

//func (gcp GcpConnector) lockCidrProviderJson(bucket *storage.BucketHandle, bucketFile string, ctx context.Context) error {
//	writer := bucket.Object(fmt.Sprintf("%s.lock", bucketFile)).If(storage.Conditions{GenerationMatch: 0}).NewWriter(ctx)
//	defer writer.Close()
//	return gcp.recursiveTryLock(writer, 0)
//}

func (gcp *GcpConnector) RecursiveRetryReadWrite(ctx context.Context, retryCount int8) error {
	if retryCount > 4 {
		return errors.New("Failed to write file after 4 retries!!!")
	}
	networkConfig, err := gcp.ReadRemote(ctx)
	if err != nil {
		return err
	}
	sleepDuration, _ := time.ParseDuration(fmt.Sprintf("%ds", 2*retryCount))
	time.Sleep(sleepDuration)
	err = gcp.WriteRemote(networkConfig, ctx)
	if err != nil {
		return gcp.RecursiveRetryReadWrite(ctx, retryCount+1)
	}
	return nil
}

//func (gcp GcpConnector) deleteLock(bucket *storage.BucketHandle, bucketFile string, ctx context.Context) {
//	bucket.Object(fmt.Sprintf("%s.lock", bucketFile)).Delete(ctx)
//}

func readNetsegmentJson(ctx context.Context, cidrProviderBucket string, netsegmentName string) (NetworkConfig, error) {
	return NetworkConfig{}, nil
	//return readRemote(cidrProviderBucket, fmt.Sprintf("gcp-cidr-provider/%s.json", netsegmentName), ctx)
}

// TODO: implement!
func uploadNewNetsegmentJson() {}
