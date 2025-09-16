package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var _ resource.Resource = &ApplicationConnectionsResource{}
var _ resource.ResourceWithImportState = &ApplicationConnectionsResource{}

// ApplicationConnectionsResource defines the resource implementation.
type ApplicationConnectionsResource struct {
	client *Auth0Client
}

// ApplicationConnectionsResourceModel describes the resource data model.
type ApplicationConnectionsResourceModel struct {
	Id                    types.String `tfsdk:"id"`
	ApplicationId         types.String `tfsdk:"application_id"`
	EnabledConnectionIds  types.List   `tfsdk:"enabled_connection_ids"`
	ManagedConnectionIds  types.List   `tfsdk:"managed_connection_ids"`
}

// Auth0 Connection Client data structure
type Auth0ConnectionClient struct {
	ConnectionId    string   `json:"connection_id"`
	EnabledClients  []string `json:"enabled_clients"`
}

func NewApplicationConnectionsResource() resource.Resource {
	return &ApplicationConnectionsResource{}
}

func (r *ApplicationConnectionsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_connections"
}

func (r *ApplicationConnectionsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Auth0 connection associations for a specific application. This resource ensures that the application is enabled for specified connections and disabled for all others, while preserving other applications' access.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
				MarkdownDescription: "The Auth0 application (client) ID to manage connections for",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled_connection_ids": schema.ListAttribute{
				MarkdownDescription: "List of connection IDs that should be enabled for this application",
				ElementType:         types.StringType,
				Required:            true,
			},
			"managed_connection_ids": schema.ListAttribute{
				MarkdownDescription: "List of all connection IDs that were managed by this resource (read-only)",
				ElementType:         types.StringType,
				Computed:            true,
			},
		},
	}
}

func (r *ApplicationConnectionsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Auth0Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Auth0Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ApplicationConnectionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ApplicationConnectionsResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get access token
	accessToken, err := r.getAccessToken(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get Auth0 access token",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}

	// Get all connections
	allConnections, err := r.fetchAllConnections(ctx, accessToken)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to fetch Auth0 connections",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}

	// Extract enabled connection IDs from plan
	var enabledConnectionIds []string
	resp.Diagnostics.Append(data.EnabledConnectionIds.ElementsAs(ctx, &enabledConnectionIds, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Apply the desired state
	managedConnections, err := r.applyConnectionState(ctx, accessToken, allConnections, data.ApplicationId.ValueString(), enabledConnectionIds)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to apply connection state",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}

	// Set computed values
	data.Id = types.StringValue(data.ApplicationId.ValueString())
	
	managedConnectionsList, diags := types.ListValueFrom(ctx, types.StringType, managedConnections)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ManagedConnectionIds = managedConnectionsList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationConnectionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ApplicationConnectionsResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get access token
	accessToken, err := r.getAccessToken(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get Auth0 access token",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}

	// Get current state of connections for this application
	currentState, err := r.getCurrentConnectionState(ctx, accessToken, data.ApplicationId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get current connection state",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}

	// Update managed connection IDs
	managedConnectionsList, diags := types.ListValueFrom(ctx, types.StringType, currentState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ManagedConnectionIds = managedConnectionsList

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationConnectionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ApplicationConnectionsResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get access token
	accessToken, err := r.getAccessToken(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get Auth0 access token",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}

	// Get all connections
	allConnections, err := r.fetchAllConnections(ctx, accessToken)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to fetch Auth0 connections",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}

	// Extract enabled connection IDs from plan
	var enabledConnectionIds []string
	resp.Diagnostics.Append(data.EnabledConnectionIds.ElementsAs(ctx, &enabledConnectionIds, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Apply the desired state
	managedConnections, err := r.applyConnectionState(ctx, accessToken, allConnections, data.ApplicationId.ValueString(), enabledConnectionIds)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to apply connection state",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}

	// Set computed values
	managedConnectionsList, diags := types.ListValueFrom(ctx, types.StringType, managedConnections)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ManagedConnectionIds = managedConnectionsList

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationConnectionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ApplicationConnectionsResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get access token
	accessToken, err := r.getAccessToken(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get Auth0 access token",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}

	// Get all connections
	allConnections, err := r.fetchAllConnections(ctx, accessToken)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to fetch Auth0 connections",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}

	// Disable this application from all connections (cleanup)
	_, err = r.applyConnectionState(ctx, accessToken, allConnections, data.ApplicationId.ValueString(), []string{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to cleanup connection state",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}
}

func (r *ApplicationConnectionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("application_id"), req, resp)
}

// Helper methods

func (r *ApplicationConnectionsResource) getAccessToken(ctx context.Context) (string, error) {
	// Reuse the same token logic from data source
	tokenURL := fmt.Sprintf("https://%s/oauth/token", r.client.Domain)

	data := fmt.Sprintf(
		"grant_type=client_credentials&client_id=%s&client_secret=%s&audience=https://%s/api/v2/",
		r.client.ClientId,
		r.client.ClientSecret,
		r.client.Domain,
	)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := r.client.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

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

func (r *ApplicationConnectionsResource) fetchAllConnections(ctx context.Context, accessToken string) ([]string, error) {
	connectionsURL := fmt.Sprintf("https://%s/api/v2/connections", r.client.Domain)

	req, err := http.NewRequestWithContext(ctx, "GET", connectionsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create connections request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make connections request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("connections request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var connections []Auth0Connection
	if err := json.NewDecoder(resp.Body).Decode(&connections); err != nil {
		return nil, fmt.Errorf("failed to decode connections response: %w", err)
	}

	var connectionIds []string
	for _, conn := range connections {
		connectionIds = append(connectionIds, conn.Id)
	}

	return connectionIds, nil
}

func (r *ApplicationConnectionsResource) getCurrentConnectionState(ctx context.Context, accessToken string, applicationId string) ([]string, error) {
	// Get all connections that currently have this application enabled
	connections, err := r.fetchAllConnections(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	var enabledConnections []string
	for _, connectionId := range connections {
		clients, err := r.getConnectionClients(ctx, accessToken, connectionId)
		if err != nil {
			continue // Skip if we can't get clients for this connection
		}

		for _, clientId := range clients {
			if clientId == applicationId {
				enabledConnections = append(enabledConnections, connectionId)
				break
			}
		}
	}

	return enabledConnections, nil
}

func (r *ApplicationConnectionsResource) getConnectionClients(ctx context.Context, accessToken string, connectionId string) ([]string, error) {
	url := fmt.Sprintf("https://%s/api/v2/connections/%s", r.client.Domain, connectionId)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make connection request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []string{}, nil // Return empty if connection doesn't exist or no access
	}

	var connection struct {
		EnabledClients []string `json:"enabled_clients"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&connection); err != nil {
		return nil, fmt.Errorf("failed to decode connection response: %w", err)
	}

	return connection.EnabledClients, nil
}

func (r *ApplicationConnectionsResource) applyConnectionState(ctx context.Context, accessToken string, allConnections []string, applicationId string, enabledConnectionIds []string) ([]string, error) {
	var managedConnections []string

	// Create a set of enabled connections for quick lookup
	enabledSet := make(map[string]bool)
	for _, connId := range enabledConnectionIds {
		enabledSet[connId] = true
	}

	// Process each connection
	for _, connectionId := range allConnections {
		// Get current enabled clients for this connection
		currentClients, err := r.getConnectionClients(ctx, accessToken, connectionId)
		if err != nil {
			continue // Skip if we can't access this connection
		}

		// Determine new client list
		var newClients []string
		
		// Add all clients except our application
		for _, clientId := range currentClients {
			if clientId != applicationId {
				newClients = append(newClients, clientId)
			}
		}

		// Add our application if it should be enabled for this connection
		if enabledSet[connectionId] {
			newClients = append(newClients, applicationId)
		}

		// Sort for consistent ordering
		sort.Strings(newClients)
		sort.Strings(currentClients)

		// Only update if the client list has changed
		if !stringSlicesEqual(currentClients, newClients) {
			err := r.updateConnectionClients(ctx, accessToken, connectionId, newClients)
			if err != nil {
				return nil, fmt.Errorf("failed to update connection %s: %w", connectionId, err)
			}
			managedConnections = append(managedConnections, connectionId)
		}
	}

	return managedConnections, nil
}

func (r *ApplicationConnectionsResource) updateConnectionClients(ctx context.Context, accessToken string, connectionId string, enabledClients []string) error {
	url := fmt.Sprintf("https://%s/api/v2/connections/%s", r.client.Domain, connectionId)

	payload := map[string]interface{}{
		"enabled_clients": enabledClients,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make update request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Helper function to compare string slices
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
