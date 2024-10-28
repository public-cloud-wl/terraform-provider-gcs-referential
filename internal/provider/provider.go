package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Define the Provider struct
type GCSReferentialProvider struct{}

// Define the Provider schema
func (p *GCSReferentialProvider) Schema(ctx context.Context) (types.Schema, diag.Diagnostics) {
	return types.Schema{
		Attributes: map[string]types.Attribute{
			"reservator_bucket": {
				Type:     types.StringType,
				Required: true,
			},
		},
	}, nil
}

// Configure function for the provider
func (p *GCSReferentialProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var diags diag.Diagnostics

	cidrReservatorBucket := req.Config.GetAttribute("reservator_bucket").(string)
	if cidrReservatorBucket == "" {
		diags = append(diags, diag.NewErrorDiagnostic("reservator_bucket is not set!"))
		resp.Diagnostics = diags
		return
	}

	// Store the bucket value in the response
	resp.Data = cidrReservatorBucket
}

// New function to create the provider
func New() func() provider.Provider {
	return func() provider.Provider {
		return &GCSReferentialProvider{}
	}
}

// Define the resource
func resourceServer() resource.Resource {
	// Implement your resource logic here
	return nil // Replace with actual resource implementation
}