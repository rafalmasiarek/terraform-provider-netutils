package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rafalmasiarek/terraform-provider-netutils/internal/diagnostics/tcp"
)

type tcpCheckModel struct {
	Host           types.String `tfsdk:"host"`
	Port           types.Int64  `tfsdk:"port"`
	TimeoutSeconds types.Int64  `tfsdk:"timeout_seconds"`
	Success        types.Bool   `tfsdk:"success"`
	Error          types.String `tfsdk:"error"`
	DurationMS     types.Int64  `tfsdk:"duration_ms"`
}

type tcpCheckDataSource struct{}

func NewTCPCheckDataSource() datasource.DataSource {
	return &tcpCheckDataSource{}
}

func (d *tcpCheckDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tcp_check"
}

func (d *tcpCheckDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attempts a TCP connection to host:port. Connection failure is reported via success=false and error, never as a plan-time error - an unreachable endpoint is a meaningful diagnostic result, not an exceptional one.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{Required: true},
			"port": schema.Int64Attribute{Required: true},
			"timeout_seconds": schema.Int64Attribute{
				Optional:    true,
				Description: "Defaults to 5.",
			},
			"success":     schema.BoolAttribute{Computed: true},
			"error":       schema.StringAttribute{Computed: true, Description: "Empty when success is true."},
			"duration_ms": schema.Int64Attribute{Computed: true},
		},
	}
}

func (d *tcpCheckDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config tcpCheckModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout := 5 * time.Second
	if !config.TimeoutSeconds.IsNull() {
		timeout = time.Duration(config.TimeoutSeconds.ValueInt64()) * time.Second
	}

	result := tcp.Check(ctx, config.Host.ValueString(), int(config.Port.ValueInt64()), timeout)

	config.Success = types.BoolValue(result.Success)
	config.Error = types.StringValue(result.Error)
	config.DurationMS = types.Int64Value(result.DurationMS)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
