package sub

import (
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v2/database/model"
	naiveplugin "github.com/mhsanaei/3x-ui/v2/plugins/naive"
)

func TestGenNaiveLinkHealthyState(t *testing.T) {
	s := &SubService{remarkModel: "-ieo"}
	inbound := &model.Inbound{Remark: "reality-main"}
	state := &naiveplugin.NaivePluginState{
		Enabled: true,
		State:   naiveplugin.NaivePluginStateHealthy,
	}
	cfg := &naiveplugin.NaiveRuntimeConfig{
		Domain:   "naive.example.com",
		Port:     443,
		Username: "user",
		Password: "pass",
	}

	link := s.genNaiveLink(inbound, "test@example.com", state, cfg)
	if !strings.HasPrefix(link, "naive+https://user:pass@naive.example.com:443") {
		t.Fatalf("unexpected naive link: %s", link)
	}
}

func TestGenNaiveLinkSkipsWhenDegraded(t *testing.T) {
	s := &SubService{}
	link := s.genNaiveLink(
		&model.Inbound{},
		"a@b",
		&naiveplugin.NaivePluginState{Enabled: true, State: naiveplugin.NaivePluginStateDegraded},
		&naiveplugin.NaiveRuntimeConfig{Domain: "naive.example.com", Port: 443, Username: "u", Password: "p"},
	)
	if link != "" {
		t.Fatalf("expected empty link for degraded state, got: %s", link)
	}
}

func TestSubJsonNaiveConfigNotGeneratedWithoutInbound(t *testing.T) {
	svc := &SubJsonService{
		SubService: &SubService{},
	}
	cfg := svc.genNaiveConfig(nil, "user@example.com")
	if cfg != nil {
		t.Fatalf("expected nil naive json config when inbound is nil")
	}
}
