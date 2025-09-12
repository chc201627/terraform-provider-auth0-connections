package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces
var _ datasource.DataSource = &ConnectionsDataSource{}

// ConnectionsDataSource defines the data source implementation.
type ConnectionsDataSource struct {
	client *Auth0Client
}

// ConnectionsDataSourceModel describes the data source data model.
type ConnectionsDataSourceModel struct {
	Id            types.String      `tfsdk:"id"`
	Connections   []ConnectionModel `tfsdk:"connections"`
	ConnectionIds types.List        `tfsdk:"connection_ids"`
	ConnectionMap types.Map         `tfsdk:"connection_map"`
}

// ConnectionModel represents a single Auth0 connection
type ConnectionModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Strategy    types.String `tfsdk:"strategy"`
	DisplayName types.String `tfsdk:"display_name"`
	Enabled     types.Bool   `tfsdk:"enabled"`
}

// Auth0 API response structure
type Auth0ConnectionsResponse struct {
	Connections []Auth0Connection `json:"connections"`
	Total       int               `json:"total"`
	Start       int               `json:"start"`
	Limit       int               `json:"limit"`
	Length      int               `json:"length"`
}

type Auth0Connection struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Strategy    string `json:"strategy"`
	DisplayName string `json:"display_name"`
	Enabled     bool   `json:"enabled"`
}

func NewConnectionsDataSource() datasource.DataSource {
	return &ConnectionsDataSource{}
}

func (d *ConnectionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connections"
}

func (d *ConnectionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all Auth0 connections from the Management API",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the data source",
				Computed:            true,
			},
			"connections": schema.ListNestedAttribute{
				MarkdownDescription: "List of all Auth0 connections",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Connection ID",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Connection name",
							Computed:            true,
						},
						"strategy": schema.StringAttribute{
							MarkdownDescription: "Connection strategy (e.g., auth0, google-oauth2, etc.)",
							Computed:            true,
						},
						"display_name": schema.StringAttribute{
							MarkdownDescription: "Connection display name",
							Computed:            true,
						},
						"enabled": schema.BoolAttribute{
							MarkdownDescription: "Whether the connection is enabled",
							Computed:            true,
						},
					},
				},
			},
			"connection_ids": schema.ListAttribute{
				MarkdownDescription: "List of connection IDs",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"connection_map": schema.MapAttribute{
				MarkdownDescription: "Map of connection names to IDs",
				ElementType:         types.StringType,
				Computed:            true,
			},
		},
	}
}

func (d *ConnectionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Auth0Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Auth0Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *ConnectionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConnectionsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get access token
	accessToken, err := d.getAccessToken(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get Auth0 access token",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}

	// Fetch connections from Auth0 API
	connections, err := d.fetchConnections(ctx, accessToken)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to fetch Auth0 connections",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}

	// Convert to Terraform model
	var connectionModels []ConnectionModel
	var connectionIds []string
	connectionMap := make(map[string]string)

	for _, conn := range connections {
		connectionModels = append(connectionModels, ConnectionModel{
			Id:          types.StringValue(conn.Id),
			Name:        types.StringValue(conn.Name),
			Strategy:    types.StringValue(conn.Strategy),
			DisplayName: types.StringValue(conn.DisplayName),
			Enabled:     types.BoolValue(conn.Enabled),
		})
		connectionIds = append(connectionIds, conn.Id)
		connectionMap[conn.Name] = conn.Id
	}

	// Set the data
	data.Id = types.StringValue("auth0-connections")
	data.Connections = connectionModels

	// Convert slices to Terraform types
	connectionIdsList, diags := types.ListValueFrom(ctx, types.StringType, connectionIds)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ConnectionIds = connectionIdsList

	connectionMapValue, diags := types.MapValueFrom(ctx, types.StringType, connectionMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ConnectionMap = connectionMapValue

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *ConnectionsDataSource) getAccessToken(ctx context.Context) (string, error) {
	// Auth0 Management API token endpoint
	tokenURL := fmt.Sprintf("https://%s/oauth/token", d.client.Domain)

	// Prepare the request body
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", d.client.ClientId)
	data.Set("client_secret", d.client.ClientSecret)
	data.Set("audience", fmt.Sprintf("https://%s/api/v2/", d.client.Domain))

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Make the request
	resp, err := d.client.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	return tokenResp.AccessToken, nil
}

func (d *ConnectionsDataSource) fetchConnections(ctx context.Context, accessToken string) ([]Auth0Connection, error) {
	// Auth0 Management API connections endpoint
	connectionsURL := fmt.Sprintf("https://%s/api/v2/connections", d.client.Domain)

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "GET", connectionsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create connections request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := d.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make connections request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("connections request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response - Auth0 API returns an array directly
	var connections []Auth0Connection
	if err := json.NewDecoder(resp.Body).Decode(&connections); err != nil {
		return nil, fmt.Errorf("failed to decode connections response: %w", err)
	}

	return connections, nil
}
