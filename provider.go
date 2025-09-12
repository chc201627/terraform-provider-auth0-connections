package main

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces
var _ provider.Provider = &Auth0ConnectionsProvider{}

// Auth0ConnectionsProvider defines the provider implementation.
type Auth0ConnectionsProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// Auth0ConnectionsProviderModel describes the provider data model.
type Auth0ConnectionsProviderModel struct {
	Domain       types.String `tfsdk:"domain"`
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

// Auth0Client represents the Auth0 API client
type Auth0Client struct {
	Domain       string
	ClientId     string
	ClientSecret string
	HTTPClient   *http.Client
}

func (p *Auth0ConnectionsProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "auth0-connections"
	resp.Version = p.version
}

func (p *Auth0ConnectionsProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "Auth0 domain (e.g., your-tenant.auth0.com)",
				Required:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "Auth0 Management API client ID",
				Required:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "Auth0 Management API client secret",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *Auth0ConnectionsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config Auth0ConnectionsProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Validate required attributes
	if config.Domain.IsUnknown() || config.Domain.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Auth0 Domain",
			"The provider cannot create the Auth0 API client as there is a missing or empty value for the Auth0 domain. "+
				"Set the domain value in the configuration and ensure the value is not empty.",
		)
		return
	}

	if config.ClientId.IsUnknown() || config.ClientId.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Auth0 Client ID",
			"The provider cannot create the Auth0 API client as there is a missing or empty value for the Auth0 client ID. "+
				"Set the client_id value in the configuration and ensure the value is not empty.",
		)
		return
	}

	if config.ClientSecret.IsUnknown() || config.ClientSecret.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Auth0 Client Secret",
			"The provider cannot create the Auth0 API client as there is a missing or empty value for the Auth0 client secret. "+
				"Set the client_secret value in the configuration and ensure the value is not empty.",
		)
		return
	}

	// Create Auth0 client
	client := &Auth0Client{
		Domain:       config.Domain.ValueString(),
		ClientId:     config.ClientId.ValueString(),
		ClientSecret: config.ClientSecret.ValueString(),
		HTTPClient:   &http.Client{},
	}

	// Make the client available to data sources and resources
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *Auth0ConnectionsProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// No resources for now, only data sources
	}
}

func (p *Auth0ConnectionsProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewConnectionsDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &Auth0ConnectionsProvider{
			version: version,
		}
	}
}
