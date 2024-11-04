package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &GCSReferentialProvider{}

/*var _ provider.ProviderWithFunctions = &GCSReferentialProvider{} */

const ProviderName = "gcsreferential"
const NumberOfRetry = int(10)

type GCSReferentialProvider struct {
	version string
}

// New function to create the provider
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GCSReferentialProvider{
			version: version,
		}
	}
}

type GCSReferentialProviderModel struct {
	ReferentialBucket types.String `tfsdk:"referential_bucket"`
}

func (p *GCSReferentialProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = ProviderName
	resp.Version = p.version
}

// Define the Provider schema
func (p *GCSReferentialProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"referential_bucket": schema.StringAttribute{
				MarkdownDescription: "The GCS bucket name where the information from this provider will be stocked",
				Required:            true,
			},
		},
	}
}

// Configure function for the provider
func (p *GCSReferentialProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data GCSReferentialProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.ReferentialBucket.ValueString() == "" {
		resp.Diagnostics.AddError("The provide must be set with referential_bucket argument", "")
	}
	resp.DataSourceData = data
	resp.ResourceData = data
}

func (p *GCSReferentialProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		/*NewNetworkRequestResource,*/
		NewIdPoolResource,
		NewIdRequestResource,
	}

}

// DataSources implements provider.Provider.
func (p *GCSReferentialProvider) DataSources(context.Context) []func() datasource.DataSource {
	/*return []func() datasource.DataSource{}*/
	return nil
}
