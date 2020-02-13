package config

import (
	"fmt"
	"strings"
	"time"

	runtime "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/lloydmeta/tasques/client"
)

type TasquesWorker struct {
	ID              WorkerId      `json:"id" mapstructure:"id"`
	ClaimAmount     uint          `json:"claim_amount" mapstructure:"claim_amount"`
	BlockFor        time.Duration `json:"block_for" mapstructure:"block_for"`
	LoopStopTimeout time.Duration `json:"loop_exit_wait" mapstructure:"loop_exit_wait"`
	Server          TasquesServer `json:"server" mapstructure:"server"`
}

type WorkerId string

type TasquesServer struct {
	Address  string         `json:"address" mapstructure:"address"`
	BasePath *string        `json:"base_path,omitempty" mapstructure:"base_path"`
	Schemes  *[]string      `json:"schemes,omitempty" mapstructure:"schemes"`
	Auth     *BasicAuthUser `json:"auth,omitempty" mapstructure:"auth"`
}

type BasicAuthUser struct {
	Name     string `json:"name" mapstructure:"name"`
	Password string `json:"password" mapstructure:"password"`
}

func (t *TasquesServer) BuildClient() *client.Tasques {
	tasquesConfig := client.DefaultTransportConfig()
	tasquesConfig.Host = t.Address
	if t.BasePath != nil {
		tasquesConfig.BasePath = *t.BasePath
	}
	if t.Schemes != nil {
		tasquesConfig.Schemes = *t.Schemes
	}
	if t.Auth == nil {
		return client.NewHTTPClientWithConfig(strfmt.Default, tasquesConfig)
	} else {
		transport := runtime.New(tasquesConfig.Host, tasquesConfig.BasePath, tasquesConfig.Schemes)
		transport.DefaultAuthentication = runtime.BasicAuth(t.Auth.Name, t.Auth.Password)
		return client.New(transport, strfmt.Default)
	}
}

func (w *WorkerId) String() string {
	return string(*w)
}

func (w *WorkerId) Set(s string) error {
	if len(strings.TrimSpace(s)) == 0 {
		return fmt.Errorf("worker Id cannot be empty")
	} else {
		*w = WorkerId(s)
		return nil
	}
}

func (w *WorkerId) Type() string {
	return "WorkerId"
}
