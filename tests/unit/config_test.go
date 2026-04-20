package unit_test

import (
	"testing"

	"one-cli/internal/config"
)

func TestLoadDoesNotRequireLegacyAuthToken(t *testing.T) {
	t.Setenv("ONE_AI_AUTH_TOKEN", "")
	t.Setenv("ONE_AI_JOB_NO", "")

	t.Chdir(t.TempDir())

	_, err := config.Load()
	if err != nil {
		t.Fatalf("expected config.Load to succeed without legacy auth token, got %v", err)
	}
}

func TestBootstrapDotenvReturnsWhetherItLoaded(t *testing.T) {
	t.Chdir(t.TempDir())

	loaded, err := config.BootstrapDotenv()
	if err != nil {
		t.Fatalf("expected no error bootstrapping dotenv, got %v", err)
	}
	if loaded {
		t.Fatal("expected no dotenv file to be loaded in empty temp dir")
	}
}

func TestParseFromLookupUsesInjectedValues(t *testing.T) {
	env, err := config.ParseFromLookup(func(key string) (string, bool) {
		switch key {
		case "ONE_AI_JOB_NO":
			return "415327", true
		case "ONE_AI_AUTH_TOKEN":
			return "Basic token", true
		case "ONE_AI_LEAVE_LIST_URL":
			return "https://example.invalid/list", true
		default:
			return "", false
		}
	})
	if err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}
	if env.JobNo != "415327" {
		t.Fatalf("unexpected job no: %q", env.JobNo)
	}
	if env.One.AuthToken != "Basic token" {
		t.Fatalf("unexpected auth token: %q", env.One.AuthToken)
	}
	if env.Endpoints.LeaveListURL != "https://example.invalid/list" {
		t.Fatalf("unexpected leave list url: %q", env.Endpoints.LeaveListURL)
	}
}

func TestValidateOneAuthRequiresAuthToken(t *testing.T) {
	env, err := config.ParseFromLookup(func(key string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}

	if err := config.ValidateOneAuth(env); err == nil {
		t.Fatal("expected ValidateOneAuth to fail when auth token is missing")
	}
}
