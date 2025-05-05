package api

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"

	paramscmd "github.com/optiflow-os/tracelens-cli/cmd/params"
	"github.com/optiflow-os/tracelens-cli/pkg/api"
	"github.com/optiflow-os/tracelens-cli/pkg/log"

	tz "github.com/gandarez/go-olson-timezone"
)

// NewClient 根据传入的参数初始化一个带有所有选项的API客户端
func NewClient(ctx context.Context, params paramscmd.API) (*api.Client, error) {
	withAuth, err := api.WithAuth(api.BasicAuth{
		Secret: params.Key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set up auth option on api client: %w", err)
	}

	return newClient(ctx, params, withAuth)
}

// NewClientWithoutAuth 根据传入的参数初始化一个带有所有选项的API客户端，
// 但禁用了身份验证
func NewClientWithoutAuth(ctx context.Context, params paramscmd.API) (*api.Client, error) {
	return newClient(ctx, params)
}

// newClient 包含客户端初始化的逻辑，除了身份验证初始化
func newClient(ctx context.Context, params paramscmd.API, opts ...api.Option) (*api.Client, error) {
	opts = append(opts, api.WithTimeout(params.Timeout))
	opts = append(opts, api.WithHostname(strings.TrimSpace(params.Hostname)))

	logger := log.Extract(ctx)

	tz, err := timezone()
	if err != nil {
		logger.Debugf("failed to detect local timezone: %s", err)
	} else {
		opts = append(opts, api.WithTimezone(strings.TrimSpace(tz)))
	}

	if params.DisableSSLVerify {
		opts = append(opts, api.WithDisableSSLVerify())
	}

	if !params.DisableSSLVerify && params.SSLCertFilepath != "" {
		withSSLCert, err := api.WithSSLCertFile(params.SSLCertFilepath)
		if err != nil {
			return nil, fmt.Errorf("failed to set up ssl cert file option on api client: %s", err)
		}

		opts = append(opts, withSSLCert)
	} else if !params.DisableSSLVerify {
		opts = append(opts, api.WithSSLCertPool(api.CACerts(ctx)))
	}

	if params.ProxyURL != "" {
		withProxy, err := api.WithProxy(params.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("failed to set up proxy option on api client: %w", err)
		}

		opts = append(opts, withProxy)

		if strings.Contains(params.ProxyURL, `\\`) {
			withNTLMRetry, err := api.WithNTLMRequestRetry(ctx, params.ProxyURL)
			if err != nil {
				return nil, fmt.Errorf("failed to set up ntlm request retry option on api client: %w", err)
			}

			opts = append(opts, withNTLMRetry)
		}
	}

	opts = append(opts, api.WithUserAgent(ctx, params.Plugin))

	return api.NewClient(params.URL, opts...), nil
}

// timezone 获取本地时区名称
func timezone() (name string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panicked: failed to get timezone: %v. Stack: %s", r, string(debug.Stack()))
		}
	}()

	name, err = tz.Name()

	return name, err
}
