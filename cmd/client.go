package cmd

import (
	"github.com/nycon/qemu-cli/internal/api"
	"github.com/nycon/qemu-cli/internal/config"
)

// clientFromActive returns an API client, honoring --insecure for this invocation.
func clientFromActive() (*api.Client, error) {
	_, ctx, err := config.Active()
	if err != nil {
		return nil, err
	}
	insecure := ctx.Insecure || insecureFlag
	return api.New(ctx.ServerURL, ctx.Token, insecure), nil
}
