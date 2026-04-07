package manifest

import "testing"

// helper to build a minimal manifest with infra environments.
func baseManifest() *Manifest {
	return &Manifest{
		Infra: InfraPillar{
			Environments: []ServerEnvironmentDef{
				{Name: "dev"},
				{Name: "prod"},
			},
		},
		Backend: BackendPillar{
			Services: []ServiceDef{
				{Name: "users-svc"},
				{Name: "orders-svc"},
			},
			Auth: &AuthConfig{
				Strategy: "JWT",
				Roles: []RoleDef{
					{Name: "admin"},
					{Name: "user"},
				},
			},
		},
		Data: DataPillar{
			Databases: []DBSourceDef{
				{Alias: "primary-pg"},
				{Alias: "redis-cache", IsCache: true},
			},
			Domains: []DomainDef{
				{Name: "User"},
				{Name: "Order"},
			},
		},
		Contracts: ContractsPillar{
			DTOs: []DTODef{
				{Name: "UserDTO"},
				{Name: "OrderDTO"},
			},
		},
	}
}

func hasCode(errs []ValidationError, code string) bool {
	for _, e := range errs {
		if e.Code == code {
			return true
		}
	}
	return false
}

func countCode(errs []ValidationError, code string) int {
	n := 0
	for _, e := range errs {
		if e.Code == code {
			n++
		}
	}
	return n
}

func TestValidate_CleanManifest(t *testing.T) {
	m := baseManifest()
	errs := Validate(m)
	if len(errs) != 0 {
		t.Fatalf("expected no errors for clean manifest, got %d: %v", len(errs), errs)
	}
}

func TestValidateServiceEnvRefs(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		wantErr bool
	}{
		{"valid env", "dev", false},
		{"empty env", "", false},
		{"stale env", "staging", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := baseManifest()
			m.Backend.Services[0].Environment = tt.env
			errs := validateServiceEnvRefs(m)
			if tt.wantErr && !hasCode(errs, "stale_env_ref") {
				t.Error("expected stale_env_ref error")
			}
			if !tt.wantErr && len(errs) > 0 {
				t.Errorf("unexpected errors: %v", errs)
			}
		})
	}
}

func TestValidateMonolithEnvRef(t *testing.T) {
	m := baseManifest()
	m.Backend.MonolithEnvironment = "staging"
	errs := validateMonolithEnvRef(m)
	if !hasCode(errs, "stale_env_ref") {
		t.Error("expected stale_env_ref for monolith environment")
	}

	m.Backend.MonolithEnvironment = "dev"
	errs = validateMonolithEnvRef(m)
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestValidateMessagingEnvRef(t *testing.T) {
	m := baseManifest()
	m.Backend.Messaging = &MessagingConfig{Environment: "gone"}
	errs := validateMessagingEnvRef(m)
	if !hasCode(errs, "stale_env_ref") {
		t.Error("expected stale_env_ref for messaging")
	}

	m.Backend.Messaging.Environment = "prod"
	errs = validateMessagingEnvRef(m)
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestValidateAPIGWEnvRef(t *testing.T) {
	m := baseManifest()
	m.Backend.APIGateway = &APIGatewayConfig{Environment: "gone"}
	errs := validateAPIGWEnvRef(m)
	if !hasCode(errs, "stale_env_ref") {
		t.Error("expected stale_env_ref for API gateway")
	}
}

func TestValidateEndpointServiceRefs(t *testing.T) {
	m := baseManifest()
	m.Contracts.Endpoints = []EndpointDef{
		{ServiceUnit: "users-svc", NamePath: "/users", Protocol: "REST"},
		{ServiceUnit: "deleted-svc", NamePath: "/gone", Protocol: "REST"},
	}
	errs := validateEndpointServiceRefs(m)
	if countCode(errs, "stale_svc_ref") != 1 {
		t.Errorf("expected 1 stale_svc_ref, got %d", countCode(errs, "stale_svc_ref"))
	}
}

func TestValidateEndpointDTORefs(t *testing.T) {
	m := baseManifest()
	m.Contracts.Endpoints = []EndpointDef{
		{NamePath: "/users", RequestDTO: "UserDTO", ResponseDTO: "GhostDTO"},
	}
	errs := validateEndpointDTORefs(m)
	if countCode(errs, "stale_dto_ref") != 1 {
		t.Errorf("expected 1 stale_dto_ref, got %d", countCode(errs, "stale_dto_ref"))
	}
}

func TestValidateEndpointDTORefs_NoDTOs(t *testing.T) {
	m := baseManifest()
	m.Contracts.DTOs = nil
	m.Contracts.Endpoints = []EndpointDef{
		{NamePath: "/users", RequestDTO: "Anything"},
	}
	errs := validateEndpointDTORefs(m)
	if len(errs) > 0 {
		t.Error("should skip validation when no DTOs defined")
	}
}

func TestValidateEndpointAuthRoles(t *testing.T) {
	m := baseManifest()
	m.Contracts.Endpoints = []EndpointDef{
		{NamePath: "/admin", AuthRequired: "true", AuthRoles: "admin"},
		{NamePath: "/super", AuthRequired: "true", AuthRoles: "superadmin"},
	}
	errs := validateEndpointAuthRoles(m)
	if countCode(errs, "auth_role_mismatch") != 1 {
		t.Errorf("expected 1 auth_role_mismatch, got %d", countCode(errs, "auth_role_mismatch"))
	}
}

func TestValidateFrontendAuth_NoBackendAuth(t *testing.T) {
	m := baseManifest()
	m.Backend.Auth = nil
	m.Frontend.Pages = []PageDef{
		{Name: "Dashboard", AuthRequired: "true"},
		{Name: "Home", AuthRequired: "false"},
	}
	errs := validateFrontendAuth(m)
	if countCode(errs, "frontend_auth_no_backend") != 1 {
		t.Errorf("expected 1 frontend_auth_no_backend, got %d", countCode(errs, "frontend_auth_no_backend"))
	}
}

func TestValidatePageAuthRoles(t *testing.T) {
	m := baseManifest()
	m.Frontend.Pages = []PageDef{
		{Name: "Admin", AuthRoles: "admin"},
		{Name: "Super", AuthRoles: "superadmin"},
	}
	errs := validatePageAuthRoles(m)
	if countCode(errs, "page_auth_role_mismatch") != 1 {
		t.Errorf("expected 1 page_auth_role_mismatch, got %d", countCode(errs, "page_auth_role_mismatch"))
	}
}

func TestValidateCommLinkServiceRefs(t *testing.T) {
	m := baseManifest()
	m.Backend.CommLinks = []CommLink{
		{From: "users-svc", To: "orders-svc"},
		{From: "users-svc", To: "deleted-svc"},
		{From: "deleted-svc", To: "orders-svc"},
	}
	errs := validateCommLinkServiceRefs(m)
	if countCode(errs, "stale_comm_svc") != 2 {
		t.Errorf("expected 2 stale_comm_svc, got %d", countCode(errs, "stale_comm_svc"))
	}
}

func TestValidateEventServiceRefs(t *testing.T) {
	m := baseManifest()
	m.Backend.Events = []EventDef{
		{Name: "UserCreated", PublisherService: "users-svc", ConsumerService: "deleted-svc"},
	}
	errs := validateEventServiceRefs(m)
	if countCode(errs, "stale_event_svc") != 1 {
		t.Errorf("expected 1 stale_event_svc, got %d", countCode(errs, "stale_event_svc"))
	}
}

func TestValidateRepoEntityRefs(t *testing.T) {
	m := baseManifest()
	m.Backend.Services[0].Repositories = []RepositoryDef{
		{Name: "UserRepo", EntityRef: "User"},
		{Name: "GhostRepo", EntityRef: "Ghost"},
	}
	errs := validateRepoEntityRefs(m)
	if countCode(errs, "stale_entity_ref") != 1 {
		t.Errorf("expected 1 stale_entity_ref, got %d", countCode(errs, "stale_entity_ref"))
	}
}

func TestValidateRepoTargetDB(t *testing.T) {
	m := baseManifest()
	m.Backend.Services[0].Repositories = []RepositoryDef{
		{Name: "UserRepo", TargetDB: "primary-pg"},
		{Name: "GhostRepo", TargetDB: "deleted-db"},
	}
	errs := validateRepoTargetDB(m)
	if countCode(errs, "stale_db_ref") != 1 {
		t.Errorf("expected 1 stale_db_ref, got %d", countCode(errs, "stale_db_ref"))
	}
}

func TestValidateCacheRefs(t *testing.T) {
	m := baseManifest()
	m.Backend.WAF = &WAFConfig{RateLimitBackend: "deleted-cache"}
	errs := validateCacheRefs(m)
	if !hasCode(errs, "stale_cache_ref") {
		t.Error("expected stale_cache_ref")
	}

	m.Backend.WAF.RateLimitBackend = "redis-cache"
	errs = validateCacheRefs(m)
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestValidateFileStorageServiceRef(t *testing.T) {
	m := baseManifest()
	m.Data.FileStorages = []FileStorageDef{
		{UsedByService: "users-svc"},
		{UsedByService: "deleted-svc"},
	}
	errs := validateFileStorageServiceRef(m)
	if countCode(errs, "stale_fs_svc") != 1 {
		t.Errorf("expected 1 stale_fs_svc, got %d", countCode(errs, "stale_fs_svc"))
	}
}

func TestValidateGovernanceDBRefs(t *testing.T) {
	m := baseManifest()
	m.Data.Governances = []DataGovernanceConfig{
		{Name: "retention", Databases: []string{"primary-pg", "deleted-db"}},
	}
	errs := validateGovernanceDBRefs(m)
	if countCode(errs, "stale_gov_db") != 1 {
		t.Errorf("expected 1 stale_gov_db, got %d", countCode(errs, "stale_gov_db"))
	}
}

func TestValidateDBEnvRefs(t *testing.T) {
	m := baseManifest()
	m.Data.Databases[0].Environment = "staging"
	errs := validateDBEnvRefs(m)
	if !hasCode(errs, "stale_env_ref") {
		t.Error("expected stale_env_ref for database environment")
	}

	m.Data.Databases[0].Environment = "dev"
	errs = validateDBEnvRefs(m)
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}
