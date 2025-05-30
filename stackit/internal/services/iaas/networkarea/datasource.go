package networkarea

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	iaasUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/iaas/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &networkAreaDataSource{}
)

// NewNetworkDataSource is a helper function to simplify the provider implementation.
func NewNetworkAreaDataSource() datasource.DataSource {
	return &networkAreaDataSource{}
}

// networkDataSource is the data source implementation.
type networkAreaDataSource struct {
	client *iaas.APIClient
}

// Metadata returns the data source type name.
func (d *networkAreaDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_area"
}

func (d *networkAreaDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	providerData, ok := conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := iaasUtils.ConfigureClient(ctx, &providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "IaaS client configured")
}

// Schema defines the schema for the data source.
func (d *networkAreaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	description := "Network area datasource schema. Must have a `region` specified in the provider configuration."
	resp.Schema = schema.Schema{
		Description:         description,
		MarkdownDescription: description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform's internal resource ID. It is structured as \"`organization_id`,`network_area_id`\".",
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "STACKIT organization ID to which the network area is associated.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"network_area_id": schema.StringAttribute{
				Description: "The network area ID.",
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the network area.",
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(63),
				},
			},
			"project_count": schema.Int64Attribute{
				Description: "The amount of projects currently referencing this area.",
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"default_nameservers": schema.ListAttribute{
				Description: "List of DNS Servers/Nameservers.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"network_ranges": schema.ListNestedAttribute{
				Description: "List of Network ranges.",
				Computed:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(64),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"network_range_id": schema.StringAttribute{
							Computed: true,
							Validators: []validator.String{
								validate.UUID(),
								validate.NoSeparator(),
							},
						},
						"prefix": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
			"transfer_network": schema.StringAttribute{
				Description: "Classless Inter-Domain Routing (CIDR).",
				Computed:    true,
			},
			"default_prefix_length": schema.Int64Attribute{
				Description: "The default prefix length for networks in the network area.",
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(24),
					int64validator.AtMost(29),
				},
			},
			"max_prefix_length": schema.Int64Attribute{
				Description: "The maximal prefix length for networks in the network area.",
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(24),
					int64validator.AtMost(29),
				},
			},
			"min_prefix_length": schema.Int64Attribute{
				Description: "The minimal prefix length for networks in the network area.",
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(22),
					int64validator.AtMost(29),
				},
			},
			"labels": schema.MapAttribute{
				Description: "Labels are key-value string pairs which can be attached to a resource container",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *networkAreaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	organizationId := model.OrganizationId.ValueString()
	networkAreaId := model.NetworkAreaId.ValueString()
	ctx = tflog.SetField(ctx, "organization_id", organizationId)
	ctx = tflog.SetField(ctx, "network_area_id", networkAreaId)

	networkAreaResp, err := d.client.GetNetworkArea(ctx, organizationId, networkAreaId).Execute()
	if err != nil {
		utils.LogError(
			ctx,
			&resp.Diagnostics,
			err,
			"Reading network area",
			fmt.Sprintf("Network area with ID %q does not exist in organization %q.", networkAreaId, organizationId),
			map[int]string{
				http.StatusForbidden: fmt.Sprintf("Organization with ID %q not found or forbidden access", organizationId),
			},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	networkAreaRanges := networkAreaResp.Ipv4.NetworkRanges

	err = mapFields(ctx, networkAreaResp, networkAreaRanges, &model)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading network area", fmt.Sprintf("Processing API payload: %v", err))
		return
	}
	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Network area read")
}
