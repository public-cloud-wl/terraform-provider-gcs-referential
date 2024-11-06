package provider

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/public-cloud-wl/terraform-provider-gcsreferential/internal/provider/connector"
	cidrCalculator "github.com/public-cloud-wl/tools/cidrCalculator"
)

type networkRequestResource struct {
	providerData GCSReferentialProviderModel
}

// Metadata implements resource.Resource.
func (r *networkRequestResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_request"
}

type networkRequestResourceModel struct {
	PrefixLength types.Int64  `tfsdk:"prefix_length"`
	BaseCidr     types.String `tfsdk:"base_cidr"`
	Netmask      types.String `tfsdk:"netmask"`
	Id           types.String `tfsdk:"id"`
}

type NetworkConfig struct {
	Subnets map[string]string `json:"subnets"`
}

func NewNetworkRequestResource() resource.Resource {
	return &networkRequestResource{}
}

func (r *networkRequestResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "network_request",
		Attributes: map[string]schema.Attribute{
			"prefix_length": schema.Int64Attribute{
				MarkdownDescription: "The prefix of the requested network for example with 24 a /24 subnet will be booked by the network_request",
				Required:            true,
			},
			"base_cidr": schema.StringAttribute{
				MarkdownDescription: "The supernet where to do the network_request, for example 10.0.0.0/8",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"netmask": schema.StringAttribute{
				MarkdownDescription: "The reserved netmask as full cidr, for example 10.12.13.0/24",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The id associate to your network_request",
				Required:            true,
			},
		},
	}
}

func (r *networkRequestResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	providerData, ok := req.ProviderData.(GCSReferentialProviderModel)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider data ", "")
	}
	r.providerData = providerData
}

func getNetworkConnector(ctx context.Context, data *networkRequestResourceModel, p GCSReferentialProviderModel, networkConfig *NetworkConfig) connector.GcpConnectorNetwork {
	gcpConnector := connector.NewNetwork(p.ReferentialBucket.ValueString(), data.BaseCidr.ValueString())
	// do a read in a fake resource to get the generation updated
	var networkconfig NetworkConfig
	err := gcpConnector.Read(ctx, &networkconfig)
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("Error on reading network_request file for cidr : %s", data.BaseCidr.ValueString()))
	}
	return gcpConnector
}

func readRemoteNetwork(ctx context.Context, data *networkRequestResourceModel, p GCSReferentialProviderModel, networkConfig *NetworkConfig, existingLock ...uuid.UUID) error {
	gcpConnector := getNetworkConnector(ctx, data, p, networkConfig)
	lockId, err := gcpConnector.WaitForlock(ctx, Timeout, existingLock...)
	if len(existingLock) <= 0 && err == nil {
		defer gcpConnector.Unlock(ctx, lockId)
	} else {
		if lockId != existingLock[0] {
			defer gcpConnector.Unlock(ctx, lockId)
		}
	}
	if err == nil {
		return gcpConnector.Read(ctx, &networkConfig)
	} else {
		return err
	}
}

func writeRemoteNetwork(ctx context.Context, data *networkRequestResourceModel, p GCSReferentialProviderModel, networkConfig *NetworkConfig, existingLock ...uuid.UUID) error {
	gcpConnector := getNetworkConnector(ctx, data, p, networkConfig)
	lockId, err := gcpConnector.WaitForlock(ctx, Timeout, existingLock...)
	if len(existingLock) <= 0 && err == nil {
		defer gcpConnector.Unlock(ctx, lockId)
	} else {
		if lockId != existingLock[0] {
			defer gcpConnector.Unlock(ctx, lockId)
		}
	}
	if err == nil {
		return gcpConnector.Write(ctx, networkConfig)
	} else {
		return err
	}
}

func (r *networkRequestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data networkRequestResourceModel
	var networkConfig NetworkConfig
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	gcpConnector := getNetworkConnector(ctx, &data, r.providerData, &networkConfig)
	lockId, err := gcpConnector.WaitForlock(ctx, Timeout)
	if err != nil {
		resp.Diagnostics.AddError("Cannot put lock to create the network_request :", err.Error())
		return
	}
	defer gcpConnector.Unlock(ctx, lockId)
	err = readRemoteNetwork(ctx, &data, r.providerData, &networkConfig, lockId)
	if err != nil {
		// file does not exist so create empty network config
		networkConfig = NetworkConfig{
			Subnets: make(map[string]string),
		}
	}
	if _, contains := networkConfig.Subnets[data.Id.ValueString()]; contains {
		resp.Diagnostics.AddError("network_request already exist with this id : %s, check your config or consider to import", data.Id.ValueString())
		return
	}

	cidrCalc, err := cidrCalculator.New(&networkConfig.Subnets, int8(data.PrefixLength.ValueInt64()), gcpConnector.BaseCidrRange)
	if err != nil {
		resp.Diagnostics.AddError("Fail to get the subnet calculator for the network_request : %s", err.Error())
		return
	}
	netmask, err := cidrCalc.GetNextNetmask()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Cannot find any subnet in %s withcidr %d available", gcpConnector.BaseCidrRange, data.PrefixLength.ValueInt64()), err.Error())
		return
	}
	networkConfig.Subnets[data.Id.ValueString()] = netmask
	err = writeRemoteNetwork(ctx, &data, r.providerData, &networkConfig, lockId)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Cannot Write %s in %s", gcpConnector.BaseCidrRange, r.providerData.ReferentialBucket.ValueString()), err.Error())
		return
	}
	data.Netmask = types.StringValue(netmask)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *networkRequestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data networkRequestResourceModel
	var networkConfig NetworkConfig
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	gcpConnector := getNetworkConnector(ctx, &data, r.providerData, &networkConfig)
	lockId, err := gcpConnector.WaitForlock(ctx, Timeout)
	if err != nil {
		resp.Diagnostics.AddError("Cannot put lock to create the network_request :", err.Error())
		return
	}
	defer gcpConnector.Unlock(ctx, lockId)
	err = readRemoteNetwork(ctx, &data, r.providerData, &networkConfig, lockId)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Cannot Read %s in %s", gcpConnector.BaseCidrRange, r.providerData.ReferentialBucket.ValueString()), err.Error())
		return
	}
	reservedSubnet, contains := networkConfig.Subnets[data.Id.ValueString()]
	if !contains {
		resp.Diagnostics.AddError("network_request cannot be find with this id : %s", data.Id.ValueString())
		return
	}
	data.Netmask = types.StringValue(reservedSubnet)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *networkRequestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data networkRequestResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *networkRequestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data networkRequestResourceModel
	var networkConfig NetworkConfig
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	gcpConnector := getNetworkConnector(ctx, &data, r.providerData, &networkConfig)
	lockId, err := gcpConnector.WaitForlock(ctx, Timeout)
	if err != nil {
		resp.Diagnostics.AddError("Cannot put lock to create the network_request :", err.Error())
		return
	}
	defer gcpConnector.Unlock(ctx, lockId)
	err = readRemoteNetwork(ctx, &data, r.providerData, &networkConfig, lockId)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Cannot Read %s in %s", gcpConnector.BaseCidrRange, r.providerData.ReferentialBucket.ValueString()), err.Error())
		return
	}
	_, contains := networkConfig.Subnets[data.Id.ValueString()]
	if !contains {
		resp.Diagnostics.AddError("network_request cannot be find with this id : %s", data.Id.ValueString())
		return
	}
	delete(networkConfig.Subnets, data.Id.ValueString())
	err = writeRemoteNetwork(ctx, &data, r.providerData, &networkConfig, lockId)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Cannot Write %s in %s", gcpConnector.BaseCidrRange, r.providerData.ReferentialBucket.ValueString()), err.Error())
		return
	}
}

func (r *networkRequestResource) importState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
