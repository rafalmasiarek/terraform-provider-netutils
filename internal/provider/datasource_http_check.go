package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/diagnostics/httpcheck"
)

type httpCheckModel struct {
	URL            types.String `tfsdk:"url"`
	TimeoutSeconds types.Int64  `tfsdk:"timeout_seconds"`
	ExpectedStatus types.Int64  `tfsdk:"expected_status"`
	Success        types.Bool   `tfsdk:"success"`
	StatusCode     types.Int64  `tfsdk:"status_code"`
	Error          types.String `tfsdk:"error"`
	DurationMS     types.Int64  `tfsdk:"duration_ms"`
}

type httpCheckDataSource struct{}

func NewHTTPCheckDataSource() datasource.DataSource {
	return &httpCheckDataSource{}
}

func (d *httpCheckDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_http_check"
}

func (d *httpCheckDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Performs an HTTP GET against url. Connection failure is reported via success=false and error, never as a plan-time error.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{Required: true},
			"timeout_seconds": schema.Int64Attribute{
				Optional:    true,
				Description: "Defaults to 5.",
			},
			"expected_status": schema.Int64Attribute{
				Optional:    true,
				Description: "Exact status code required for success. Omit to accept any 2xx or 3xx response.",
			},
			"success":     schema.BoolAttribute{Computed: true},
			"status_code": schema.Int64Attribute{Computed: true, Description: "0 if the request never got a response."},
			"error":       schema.StringAttribute{Computed: true, Description: "Empty when success is true."},
			"duration_ms": schema.Int64Attribute{Computed: true},
		},
	}
}

func (d *httpCheckDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config httpCheckModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout := 5 * time.Second
	if !config.TimeoutSeconds.IsNull() {
		timeout = time.Duration(config.TimeoutSeconds.ValueInt64()) * time.Second
	}
	expectedStatus := 0
	if !config.ExpectedStatus.IsNull() {
		expectedStatus = int(config.ExpectedStatus.ValueInt64())
	}

	result := httpcheck.Check(ctx, config.URL.ValueString(), timeout, expectedStatus)

	config.Success = types.BoolValue(result.Success)
	config.StatusCode = types.Int64Value(int64(result.StatusCode))
	config.Error = types.StringValue(result.Error)
	config.DurationMS = types.Int64Value(result.DurationMS)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
