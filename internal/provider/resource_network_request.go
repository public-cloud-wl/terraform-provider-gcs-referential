package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/public-cloud-wl/terraform-provider-gcsreferential/internal/provider/connector"
	"github.com/public-cloud-wl/tools/utils"
)

type networkRequestResource struct{}

func NewNetworkRequestResource() resource.Resource {
	return &networkRequestResource{}
}

func (r *networkRequestResource) Schema(ctx context.Context) (resource.Schema, diag.Diagnostics) {
	return resource.Schema{
		Attributes: map[string]resource.Attribute{
			"prefix_length": {
				Type:     types.Int64Type,
				Required: true,
			},
			"base_cidr": {
				Type:     types.StringType,
				Required: true,
				ForceNew: true,
			},
			"netmask_id": {
				Type:     types.StringType,
				Required: true,
			},
			"netmask": {
				Type:     types.StringType,
				Computed: true,
			},
		},
	}, nil
}

func (r *networkRequestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var diags diag.Diagnostics
	// Call the existing inner function for create logic
	err := innerResourceServerCreate(ctx, req, resp)
	if err != nil {
		resp.Diagnostics.AddError("Error creating resource", err.Error())
		return
	}
	resp.Diagnostics = diags
}

func (r *networkRequestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var diags diag.Diagnostics
	// Call the existing inner function for read logic
	idContent := strings.Split(req.State.ID, ":")
	reservatorBucket := idContent[0]
	baseCidr := idContent[1]
	netmaskId := idContent[2]
	gcpConnector := connector.NewNetwork(reservatorBucket, baseCidr)
	networkConfig, err := gcpConnector.ReadRemote(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading resource", err.Error())
		return
	}
	subnet, contains := networkConfig.Subnets[netmaskId]
	if !contains {
		resp.Diagnostics.AddError("Netmask not found", fmt.Sprintf("Netmask with id %s does not exist!", netmaskId))
		return
	}
	prefixLength := strings.Split(subnet, "/")[1]
	resp.State.SetAttribute("base_cidr", baseCidr)
	resp.State.SetAttribute("netmask_id", netmaskId)
	resp.State.SetAttribute("prefix_length", prefixLength)
	resp.State.SetAttribute("netmask", subnet)
}

func (r *networkRequestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var diags diag.Diagnostics
	// Call the existing inner function for update logic
	err := innerResourceServerUpdate(ctx, req, resp)
	if err != nil {
		resp.Diagnostics.AddError("Error updating resource", err.Error())
		return
	}
	resp.Diagnostics = diags
}

func (r *networkRequestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var diags diag.Diagnostics
	// Call the existing inner function for delete logic
	err := innerResourceServerDelete(ctx, req, resp)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting resource", err.Error())
		return
	}
	resp.Diagnostics = diags
}

func importState(ctx context.Context, data *schema.ResourceData, i interface{}) ([]*schema.ResourceData, error) {
	idContent := strings.Split(data.Id(), ":")
	reservatorBucket := idContent[0]
	baseCidr := idContent[1]
	netmaskId := idContent[2]
	gcpConnector := connector.NewNetwork(reservatorBucket, baseCidr)
	networkConfig, err := gcpConnector.ReadRemote(ctx)
	if err != nil {
		return nil, err
	}
	subnet, contains := networkConfig.Subnets[netmaskId]
	if !contains {
		return nil, errors.New(fmt.Sprintf("Netmask with id %s does not exist!", netmaskId))
	}
	prefixLength := strings.Split(subnet, "/")[1]
	data.Set("base_cidr", baseCidr)
	data.Set("netmask_id", netmaskId)
	data.Set("prefix_length", prefixLength)
	return []*schema.ResourceData{data}, nil
}

func readRemote(ctx context.Context, data *schema.ResourceData, m interface{}) (*connector.NetworkConfig, *connector.GcpConnector, error) {
	cidrProviderBucket := m.(string)
	gcpConnector := connector.NewNetwork(cidrProviderBucket, data.Get("base_cidr").(string))
	networkConfig, err := gcpConnector.ReadRemote(ctx)
	if err != nil {
		if err.Error() == "storage: object doesn't exist" {
			err = nil
			networkConfig = &connector.NetworkConfig{Subnets: make(map[string]string)}
		} else {
			return nil, nil, err
		}
	}
	//netmaskId := data.Get("netmask_id").(string)
	//data.SetId("")
	//if netmask, contains := networkConfig.Subnets[netmaskId]; contains {
	//	data.SetId(netmaskId)
	//	err = data.Set("netmask", netmask)
	//	if err != nil {
	//		return nil, nil, err
	//	}
	//}
	return networkConfig, &gcpConnector, nil
}

func innerResourceServerCreate(ctx context.Context, data *schema.ResourceData, m interface{}) func() error {
	return func() error {
		networkConfig, gcpConnector, err := readRemote(ctx, data, m)
		if err != nil {
			return err
		}
		if data.Get("id") != nil {
			return nil
		}
		netmaskId := data.Get("netmask_id").(string)
		if _, contains := networkConfig.Subnets[netmaskId]; contains {
			return fmt.Errorf("The netmaskId %s already exists, but does not belong to your Terraform state!!!", netmaskId)
		}
		prefixLength := int8(data.Get("prefix_length").(int))
		if err != nil {
			return err
		}
		newCidrCalculator, err := cidrCalculator.New(&networkConfig.Subnets, prefixLength, gcpConnector.BaseCidrRange)
		if err != nil {
			return err
		}
		nextNetmask, err := newCidrCalculator.GetNextNetmask()
		if err != nil {
			return err
		}
		networkConfig.Subnets[netmaskId] = nextNetmask
		//err = gcpConnector.Recursiveutils.retryReadWrite
		err = gcpConnector.WriteRemote(networkConfig, ctx)
		if err != nil {
			return err
		}
		data.SetId(fmt.Sprintf("%s:%s:%s", gcpConnector.BucketName, gcpConnector.BaseCidrRange, netmaskId))
		err = data.Set("netmask", nextNetmask)
		if err != nil {
			return err
		}
		return nil
	}
}

// TODO: if netmask_id already exists, which does not belong to THIS state, throw error.
// TODO: Update of netmask_id should not enforce recreate.
func innerResourceServerUpdate(ctx context.Context, data *schema.ResourceData, m interface{}) func() error {
	return func() error {
		networkConfig, gcpConnector, err := readRemote(ctx, data, m)
		if err != nil {
			return err
		}
		id := data.Id()
		valuesFromId := strings.Split(id, ":")
		netmaskId := data.Get("netmask_id").(string)
		netmaskIdFromId := valuesFromId[2]
		currentSubnet := networkConfig.Subnets[netmaskIdFromId]
		if netmaskIdFromId != netmaskId {
			delete(networkConfig.Subnets, netmaskIdFromId)
			if _, contains := networkConfig.Subnets[netmaskId]; contains {
				return fmt.Errorf("The netmaskId %s already exists, but does not belong to your Terraform state!!!", netmaskId)
			}
			networkConfig.Subnets[netmaskId] = currentSubnet
		}
		currentPrefixLength, err := strconv.ParseInt(strings.Split(currentSubnet, "/")[1], 10, 8)
		if err != nil {
			return err
		}
		prefixLength := int8(data.Get("prefix_length").(int))
		baseCidrRangeFromId := valuesFromId[1]
		if (baseCidrRangeFromId != gcpConnector.BaseCidrRange) || (int8(currentPrefixLength) != prefixLength) {
			newCidrCalculator, err := cidrCalculator.New(&networkConfig.Subnets, int8(prefixLength), gcpConnector.BaseCidrRange)
			if err != nil {
				return err
			}
			nextNetmask, err := newCidrCalculator.GetNextNetmask()
			if err != nil {
				return err
			}
			networkConfig.Subnets[netmaskId] = nextNetmask
			err = data.Set("netmask", nextNetmask)
			if err != nil {
				return err
			}
		}
		err = gcpConnector.WriteRemote(networkConfig, ctx)
		if err != nil {
			return err
		}
		data.SetId(fmt.Sprintf("%s:%s:%s", gcpConnector.BucketName, gcpConnector.BaseCidrRange, netmaskId))
		return nil
	}
}

func innerResourceServerDelete(ctx context.Context, data *schema.ResourceData, m interface{}) func() error {
	return func() error {
		networkConfig, gcpConnector, err := readRemote(ctx, data, m)
		if err != nil {
			return err
		}
		netmaskId := data.Get("netmask_id").(string)
		delete(networkConfig.Subnets, netmaskId)
		err = gcpConnector.WriteRemote(networkConfig, ctx)
		if err != nil {
			return err
		}
		return nil
	}
}
