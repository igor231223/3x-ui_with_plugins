package naive

import "testing"

func TestNaivePreflightRejectsRealityNaivePortConflict(t *testing.T) {
	svc := NaivePluginService{}
	res := svc.ValidatePreflight(NaivePreflightRequest{
		RealityListen: "0.0.0.0",
		RealityPort:   443,
		NaiveListen:   "0.0.0.0",
		NaivePort:     443,
		Domain:        "naive.example.com",
		EnableNaive:   true,
	})
	if res.OK {
		t.Fatalf("expected conflict to be rejected")
	}
	if len(res.Errors) == 0 {
		t.Fatalf("expected error details for conflict")
	}
}

func TestNaivePreflightRequiresDomain(t *testing.T) {
	svc := NaivePluginService{}
	res := svc.ValidatePreflight(NaivePreflightRequest{
		RealityListen: "10.0.0.1",
		RealityPort:   443,
		NaiveListen:   "10.0.0.2",
		NaivePort:     443,
		Domain:        "",
		EnableNaive:   true,
	})
	if res.OK {
		t.Fatalf("expected empty domain to be rejected")
	}
}

func TestNaivePreflightPassesForTwoIpsOn443(t *testing.T) {
	svc := NaivePluginService{}
	res := svc.ValidatePreflight(NaivePreflightRequest{
		RealityListen: "10.0.0.1",
		RealityPort:   443,
		NaiveListen:   "10.0.0.2",
		NaivePort:     443,
		Domain:        "naive.example.com",
		EnableNaive:   true,
	})
	if !res.OK {
		t.Fatalf("expected two-ip 443 topology to pass, got errors: %v", res.Errors)
	}
}

func TestSetNaiveRuntimeConfigRejectsInvalidPort(t *testing.T) {
	svc := NaivePluginService{}
	_, err := svc.SetRuntimeConfig(&NaiveRuntimeConfig{
		Domain:   "naive.example.com",
		Port:     70000,
		Username: "u",
		Password: "p",
	})
	if err == nil {
		t.Fatalf("expected invalid port to fail")
	}
}

func TestRuntimeInstallFailsOnIncompleteConfig(t *testing.T) {
	svc := NaivePluginService{}
	_, err := svc.SetRuntimeConfig(&NaiveRuntimeConfig{
		Domain: "",
		Port:   443,
	})
	if err != nil {
		t.Fatalf("unexpected set runtime config error: %v", err)
	}
	_, err = svc.InstallRuntimeArtifacts()
	if err == nil {
		t.Fatalf("expected runtime install to fail for incomplete config")
	}
}
