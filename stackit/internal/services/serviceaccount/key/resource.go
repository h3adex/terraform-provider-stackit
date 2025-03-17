package key

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/features"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
	"time"
)

// resourceBetaCheckDone is used to prevent multiple checks for beta resources.
// This is a workaround for the lack of a global state in the provider and
// needs to exist because the Configure method is called twice.
var resourceBetaCheckDone bool

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &serviceAccountKeyResource{}
	_ resource.ResourceWithConfigure = &serviceAccountKeyResource{}
)

// Model represents the schema for the service account key resource in Terraform.
type Model struct {
	Id                  types.String `tfsdk:"id"`
	KeyId               types.String `tfsdk:"key_id"`
	ServiceAccountEmail types.String `tfsdk:"service_account_email"`
	ProjectId           types.String `tfsdk:"project_id"`
	RotateWhenChanged   types.Map    `tfsdk:"rotate_when_changed"`
	TtlDays             types.Int64  `tfsdk:"ttl_days"`
	CreatedAt           types.String `tfsdk:"created_at"`
	ValidUntil          types.String `tfsdk:"valid_until"`
	Audience            types.String `tfsdk:"audience"`
	Issuer              types.String `tfsdk:"issuer"`
	Kid                 types.String `tfsdk:"kid"`
	PrivateKey          types.String `tfsdk:"private_key"`
	Subject             types.String `tfsdk:"subject"`
	KeyAlgorithm        types.String `tfsdk:"key_algorithm"`
	KeyOrigin           types.String `tfsdk:"key_origin"`
	KeyType             types.String `tfsdk:"key_type"`
	PublicKey           types.String `tfsdk:"public_key"`
	RawResponse         types.String `tfsdk:"raw_response"`
}

// NewServiceAccountKeyResource is a helper function to create a new service account key resource instance.
func NewServiceAccountKeyResource() resource.Resource {
	return &serviceAccountKeyResource{}
}

// serviceAccountKeyResource implements the resource interface for service account key.
type serviceAccountKeyResource struct {
	client *serviceaccount.APIClient
}

// Configure sets up the API client for the service account resource.
func (r *serviceAccountKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent potential panics if the provider is not properly configured.
	if req.ProviderData == nil {
		return
	}

	// Validate provider data type before proceeding.
	providerData, ok := req.ProviderData.(core.ProviderData)
	if !ok {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Expected configure type stackit.ProviderData, got %T", req.ProviderData))
		return
	}

	if !resourceBetaCheckDone {
		features.CheckBetaResourcesEnabled(ctx, &providerData, &resp.Diagnostics, "stackit_service_account_key", "resource")
		if resp.Diagnostics.HasError() {
			return
		}
		resourceBetaCheckDone = true
	}

	// Initialize the API client with the appropriate authentication and endpoint settings.
	var apiClient *serviceaccount.APIClient
	var err error
	if providerData.ServiceAccountCustomEndpoint != "" {
		apiClient, err = serviceaccount.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
			config.WithEndpoint(providerData.ServiceAccountCustomEndpoint),
		)
	} else {
		apiClient, err = serviceaccount.NewAPIClient(
			config.WithCustomAuth(providerData.RoundTripper),
		)
	}

	// Handle API client initialization errors.
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error configuring API client", fmt.Sprintf("Configuring client: %v. This is an error related to the provider configuration, not to the resource configuration", err))
		return
	}

	// Store the initialized client.
	r.client = apiClient
	tflog.Info(ctx, "Service Account client configured")
}

// Metadata sets the resource type name for the service account resource.
func (r *serviceAccountKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_key"
}

// Schema defines the resource schema for the service account access token.
func (r *serviceAccountKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	descriptions := map[string]string{
		"id":                    "The unique internal identifier for the Terraform resource, formatted as a combination of 'project_id,key_id'.",
		"project_id":            "The STACKIT project ID associated with the service account token.",
		"key_id":                "The unique identifier for the key associated with the service account.",
		"service_account_email": "The email address associated with the service account, used for account identification and communication.",
		"ttl_days":              "Specifies the key's validity duration in days. If left unspecified, the key is considered valid until it is deleted",
		"valid_until":           "The date and time until which the key remains valid. If left unspecified, the key is considered valid until it is deleted.",
		"created_at":            "The precise timestamp marking when the access token was created, provided in a formatted date-time string.",
		"rotate_when_changed":   "A map of arbitrary key/value pairs designed to force token recreation when they change, facilitating token rotation based on external factors such as a changing timestamp. Modifying this map triggers the creation of a new resource.",
		"audience":              "A list or string representing the intended audience for the token, indicating who can consume it.",
		"issuer":                "The entity or authority that issued the token, typically a URL or email address, indicating its source.",
		"kid":                   "The Key ID ('kid'), which aids in identifying the exact key used for signing the token.",
		"private_key":           "The private portion of the key pair, used to sign or encrypt data. For security reasons, handle with care.",
		"subject":               "The subject claim ('sub') identifies the principal that is the subject of the JWT.",
		"key_algorithm":         "The cryptographic algorithm used for the key, such as 'RSA_2048', specifying the bit size and type.",
		"key_origin":            "The way in which the key was provided or generated, with possible values including 'USER_PROVIDED' or 'GENERATED'.",
		"key_type":              "The type of key management, such as 'USER_MANAGED' or 'SYSTEM_MANAGED', indicating how keys are administered.",
		"public_key":            "The public portion of the key pair, which may be shared openly and used for verification or encryption.",
		"raw_response":          "The raw JSON representation of the API response, available for direct use.",
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: markdownDescription,
		Description:         "Schema for managing a STACKIT service account access token.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_id": schema.StringAttribute{
				Description: descriptions["key_id"],
				Computed:    true,
			},
			"service_account_email": schema.StringAttribute{
				Description: descriptions["service_account_email"],
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_key": schema.StringAttribute{
				Description: descriptions["public_key"],
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ttl_days": schema.Int64Attribute{
				Description: descriptions["ttl_days"],
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"rotate_when_changed": schema.MapAttribute{
				Description: descriptions["rotate_when_changed"],
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: descriptions["created_at"],
				Computed:    true,
			},
			"valid_until": schema.StringAttribute{
				Description: descriptions["valid_until"],
				Computed:    true,
			},
			"audience": schema.StringAttribute{
				Description: descriptions["audience"],
				Computed:    true,
			},
			"issuer": schema.StringAttribute{
				Description: descriptions["issuer"],
				Computed:    true,
			},
			"kid": schema.StringAttribute{
				Description: descriptions["kid"],
				Computed:    true,
			},
			"private_key": schema.StringAttribute{
				Description: descriptions["private_key"],
				Computed:    true,
				Sensitive:   true,
			},
			"subject": schema.StringAttribute{
				Description: descriptions["subject"],
				Computed:    true,
			},
			"key_algorithm": schema.StringAttribute{
				Description: descriptions["key_algorithm"],
				Computed:    true,
			},
			"key_origin": schema.StringAttribute{
				Description: descriptions["key_origin"],
				Computed:    true,
			},
			"key_type": schema.StringAttribute{
				Description: descriptions["key_type"],
				Computed:    true,
			},
			"raw_response": schema.StringAttribute{
				Description: descriptions["raw_response"],
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state for service accounts.
func (r *serviceAccountKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the planned values for the resource.
	var model Model
	diags := req.Plan.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set logging context with the project ID and service account email.
	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "service_account_email", serviceAccountEmail)

	if model.TtlDays.IsUnknown() {
		model.TtlDays = types.Int64Null()
	}

	// Generate the API request payload.
	payload, err := toCreatePayload(&model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating service account access token", fmt.Sprintf("Creating API payload: %v", err))
		return
	}

	// Initialize the API request with the required parameters.
	serviceAccountAccessTokenResp, err := r.client.CreateServiceAccountKey(ctx, projectId, serviceAccountEmail).CreateServiceAccountKeyPayload(*payload).Execute()

	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Failed to create service account access token", fmt.Sprintf("API call error: %v", err))
		return
	}

	// Map the response to the resource schema.
	err = mapCreateResponse(serviceAccountAccessTokenResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error creating service account access token", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set the state with fully populated data.
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Service account access token created")
}

// Read refreshes the Terraform state with the latest service account data.
func (r *serviceAccountKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve the current state of the resource.
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	keyId := model.KeyId.ValueString()

	saKeyResp, err := r.client.GetServiceAccountKey(ctx, projectId, serviceAccountEmail, keyId).Execute()
	if err != nil {
		oapiErr, ok := err.(*oapierror.GenericOpenAPIError) //nolint:errorlint //complaining that error.As should be used to catch wrapped errors, but this error should not be wrapped
		if ok && oapiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service account key", fmt.Sprintf("Calling API: %v", err))
		return
	}

	err = mapReadResponse(saKeyResp, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading service account key", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "key read")
}

// Update attempts to update the resource. In this case, service account token cannot be updated.
// Note: This method is intentionally left without update logic because changes
// to 'project_id', 'service_account_email' or 'ttl_days' require the resource to be entirely replaced.
// As a result, the Update function is redundant since any modifications will
// automatically trigger a resource recreation through Terraform's built-in
// lifecycle management.
func (r *serviceAccountKeyResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) { // nolint:gocritic // function signature required by Terraform
	// Service accounts cannot be updated, so we log an error.
	core.LogAndAddError(ctx, &resp.Diagnostics, "Error updating service account access token", "Service accounts can't be updated")
}

// Delete deletes the service account and removes it from the Terraform state on success.
func (r *serviceAccountKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { // nolint:gocritic // function signature required by Terraform
	// Retrieve current state of the resource.
	// Retrieve current state of the resource.
	var model Model
	diags := req.State.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectId := model.ProjectId.ValueString()
	serviceAccountEmail := model.ServiceAccountEmail.ValueString()
	keyId := model.KeyId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "service_account_email", serviceAccountEmail)
	ctx = tflog.SetField(ctx, "key_id", keyId)

	// Call API to delete the existing service account.
	err := r.client.DeleteServiceAccountKey(ctx, projectId, serviceAccountEmail, keyId).Execute()
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error deleting service account key", fmt.Sprintf("Calling API: %v", err))
		return
	}

	tflog.Info(ctx, "Service account key deleted")
}

func toCreatePayload(model *Model) (*serviceaccount.CreateServiceAccountKeyPayload, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}

	// Prepare the payload
	payload := &serviceaccount.CreateServiceAccountKeyPayload{}

	// Set ValidUntil based on TtlDays if specified
	if !model.TtlDays.IsNull() {
		validUntil, err := computeValidUntil(model.TtlDays.ValueInt64Pointer())
		if err != nil {
			return nil, err
		}
		payload.ValidUntil = &validUntil
	}

	// Set PublicKey if specified
	if model.PublicKey.String() != "" {
		payload.PublicKey = conversion.StringValueToPointer(model.PublicKey)
	}

	return payload, nil
}

// ComputeValidUntil calculates the timestamp for when the item will no longer be valid.
func computeValidUntil(ttlDays *int64) (time.Time, error) {
	if ttlDays == nil {
		return time.Time{}, fmt.Errorf("ttlDays is nil")
	}
	return time.Now().UTC().Add(time.Duration(*ttlDays) * 24 * time.Hour), nil
}

// Converts a pointer to a string to a types.String value.
func mapString(s *string) types.String {
	if s != nil {
		return types.StringValue(*s)
	}
	return types.StringNull()
}

// Converts a pointer to a time.Time to a types.String in RFC3339 format.
func mapTimestamp(t *time.Time) types.String {
	if t != nil {
		return types.StringValue(t.Format(time.RFC3339))
	}
	return types.StringNull()
}

// Maps common fields between a source structure and a model.
func mapCommonFields(
	id,
	keyAlgorithm,
	keyOrigin,
	keyType,
	publicKey,
	aud,
	kid,
	iss,
	sub *string,
	createdAt,
	validUntil *time.Time,
	model *Model,
) {
	if model.PublicKey.IsNull() {
		model.PublicKey = mapString(publicKey)
	}

	model.Id = types.StringValue(strings.Join([]string{model.ProjectId.ValueString(), *id}, core.Separator))
	model.KeyId = mapString(id)
	model.CreatedAt = mapTimestamp(createdAt)
	model.ValidUntil = mapTimestamp(validUntil)
	model.KeyAlgorithm = mapString(keyAlgorithm)
	model.KeyOrigin = mapString(keyOrigin)
	model.KeyType = mapString(keyType)
	model.Audience = mapString(aud)
	model.Issuer = mapString(iss)
	model.Kid = mapString(kid)
	model.Subject = mapString(sub)
}

// Maps response data from a read operation to the model.
func mapReadResponse(resp *serviceaccount.GetServiceAccountKeyResponse, model *Model) error {
	if resp == nil || model == nil {
		return fmt.Errorf("response or model input is nil")
	}
	if resp.Id == nil {
		return fmt.Errorf("service account key id not present")
	}

	mapCommonFields(
		resp.Id,
		resp.KeyAlgorithm,
		resp.KeyOrigin,
		resp.KeyType,
		resp.PublicKey,
		resp.Credentials.Aud,
		resp.Credentials.Kid,
		resp.Credentials.Iss,
		resp.Credentials.Sub,
		resp.CreatedAt,
		resp.ValidUntil,
		model,
	)
	return nil
}

// Maps response data from a create operation to the model.
func mapCreateResponse(resp *serviceaccount.CreateServiceAccountKeyResponse, model *Model) error {
	if resp == nil || model == nil {
		return fmt.Errorf("response or model input is nil")
	}
	if resp.Id == nil {
		return fmt.Errorf("service account key id not present")
	}

	if !model.PrivateKey.IsNull() {
		model.PrivateKey = mapString(resp.Credentials.PrivateKey)
	}

	mapCommonFields(
		resp.Id,
		resp.KeyAlgorithm,
		resp.KeyOrigin,
		resp.KeyType,
		resp.PublicKey,
		resp.Credentials.Aud,
		resp.Credentials.Kid,
		resp.Credentials.Iss,
		resp.Credentials.Sub,
		resp.CreatedAt,
		resp.ValidUntil,
		model,
	)

	jsonData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("error marshaling to JSON: %w", err)
	}
	model.RawResponse = types.StringValue(string(jsonData))

	return nil
}
