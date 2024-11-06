package provider

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	IdPoolTools "github.com/public-cloud-wl/tools/idPoolTools"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IdRequestResource{}
var _ resource.ResourceWithImportState = &IdRequestResource{}

const IdRequestResourceName = "id_request"

func NewIdRequestResource() resource.Resource {
	return &IdRequestResource{}
}

type IdRequestResource struct {
	providerData GCSReferentialProviderModel
}

type IdRequestResourceModel struct {
	Id          types.String `tfsdk:"id"`
	Pool        types.String `tfsdk:"pool"`
	RequestedId types.Int64  `tfsdk:"requested_id"`
}

func (r *IdRequestResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + IdRequestResourceName
}

func (r *IdRequestResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource allow you to request and id from an id_pool",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The terraform id of the resource",
				Optional:            false,
				Required:            true,
			},
			"pool": schema.StringAttribute{
				MarkdownDescription: "The name of the pool, to make the id_request on. If you change it, the id_request will be destroyed and recreate",
				Optional:            false,
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"requested_id": schema.Int64Attribute{
				MarkdownDescription: "The requested id from the pool, a free one that will be reserved for this resource",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *IdRequestResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IdRequestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IdRequestResourceModel
	var err error
	var poolModel IdPoolResourceModel
	var pool IdPoolTools.IDPool
	var lockId uuid.UUID
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	poolModel.Name = data.Pool
	gcpConnector := getPoolConnector(ctx, &poolModel, r.providerData, &pool)
	lockId, err = gcpConnector.WaitForlock(ctx, Timeout)
	if err != nil {
		resp.Diagnostics.AddError("Cannot put lock to create the id_request :", err.Error())
		return
	}
	defer gcpConnector.Unlock(ctx, lockId)
	err = readRemoteIdPool(ctx, &poolModel, r.providerData, &pool, lockId)
	if err != nil {
		resp.Diagnostics.AddError("Cannot find pool to make the id_request on", err.Error())
		return
	}
	_, ok := pool.Members[data.Id.ValueString()]
	if ok {
		resp.Diagnostics.AddError("The if of your id_request is already present in the pool, be sure you did not make any mistake, or consider to import", err.Error())
		return
	}
	generatedId := pool.AllocateID(data.Id.ValueString())
	if generatedId == IdPoolTools.NoID {
		resp.Diagnostics.AddError("There is no more id available in the pool", err.Error())
		return
	}
	data.RequestedId = types.Int64Value(int64(generatedId))
	err = writeRemoteIdPool(ctx, &poolModel, r.providerData, &pool, lockId)
	if err != nil {
		resp.Diagnostics.AddError("Cannot update pool on the referential_bucket", err.Error())
		return
	}
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IdRequestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IdRequestResourceModel
	var err error
	var poolModel IdPoolResourceModel
	var pool IdPoolTools.IDPool
	var lockId uuid.UUID
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	poolModel.Name = data.Pool
	err = readRemoteIdPool(ctx, &poolModel, r.providerData, &pool, lockId)
	if err != nil {
		resp.Diagnostics.AddError("Cannot find pool to make the id_request on", err.Error())
		return
	}
	value, ok := pool.Members[data.Id.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Cannot find your id_request on the pool", err.Error())
		return
	}
	data.RequestedId = types.Int64Value(int64(value))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (r *IdRequestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IdRequestResourceModel
	//var err error
	//var poolModel IdPoolResourceModel
	//var pool IdPoolTools.IDPool
	//var lockId uuid.UUID
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IdRequestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IdRequestResourceModel
	var err error
	var poolModel IdPoolResourceModel
	var pool IdPoolTools.IDPool
	var lockId uuid.UUID
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	poolModel.Name = data.Pool
	gcpConnector := getPoolConnector(ctx, &poolModel, r.providerData, &pool)
	lockId, err = gcpConnector.WaitForlock(ctx, Timeout)
	if err != nil {
		resp.Diagnostics.AddError("Cannot put lock to create the id_request :", err.Error())
		return
	}
	defer gcpConnector.Unlock(ctx, lockId)

	localerr := readRemoteIdPool(ctx, &poolModel, r.providerData, &pool, lockId)
	if localerr != nil {
		resp.Diagnostics.AddError("Cannot get id_pool from id_request.pool on the referential_bucket", err.Error())
		return
	}
	value, ok := pool.Members[data.Id.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Cannot find your id_request in the referential_bucket", err.Error())
		return
	}
	pool.Release(value)
	poolJson, _ := json.Marshal(pool)
	tflog.Debug(ctx, string(poolJson))
	err = writeRemoteIdPool(ctx, &poolModel, r.providerData, &pool, lockId)

	if err != nil {
		resp.Diagnostics.AddError("Cannot update pool on the referential_bucket", err.Error())
		return
	}
}

func (r *IdRequestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
