package manifest

import "fmt"

// ValidationError describes a cross-pillar incompatibility in a Manifest.
type ValidationError struct {
	Pillar  string // originating pillar: "backend", "frontend", etc.
	Path    string // human-readable field path, e.g. "backend.services[0].environment"
	Code    string // machine-readable code, e.g. "stale_env_ref"
	Message string // user-facing description
}

// Validate checks a Manifest for cross-pillar reference integrity.
// It returns nil when the manifest is consistent.
func Validate(m *Manifest) []ValidationError {
	var errs []ValidationError
	errs = append(errs, validateServiceEnvRefs(m)...)
	errs = append(errs, validateMonolithEnvRef(m)...)
	errs = append(errs, validateMessagingEnvRef(m)...)
	errs = append(errs, validateAPIGWEnvRef(m)...)
	errs = append(errs, validateEndpointServiceRefs(m)...)
	errs = append(errs, validateEndpointDTORefs(m)...)
	errs = append(errs, validateEndpointAuthRoles(m)...)
	errs = append(errs, validateFrontendAuth(m)...)
	errs = append(errs, validatePageAuthRoles(m)...)
	errs = append(errs, validateCommLinkServiceRefs(m)...)
	errs = append(errs, validateEventServiceRefs(m)...)
	errs = append(errs, validateRepoEntityRefs(m)...)
	errs = append(errs, validateRepoTargetDB(m)...)
	errs = append(errs, validateCacheRefs(m)...)
	errs = append(errs, validateFileStorageServiceRef(m)...)
	errs = append(errs, validateGovernanceDBRefs(m)...)
	errs = append(errs, validateDBEnvRefs(m)...)
	return errs
}

// ── helpers ──────────────────────────────────────────────────────────────────

func envNameSet(m *Manifest) map[string]bool {
	s := make(map[string]bool, len(m.Infra.Environments))
	for _, e := range m.Infra.Environments {
		s[e.Name] = true
	}
	return s
}

func serviceNameSet(m *Manifest) map[string]bool {
	s := make(map[string]bool, len(m.Backend.Services))
	for _, svc := range m.Backend.Services {
		s[svc.Name] = true
	}
	return s
}

func dtoNameSet(m *Manifest) map[string]bool {
	s := make(map[string]bool, len(m.Contracts.DTOs))
	for _, d := range m.Contracts.DTOs {
		s[d.Name] = true
	}
	return s
}

func domainNameSet(m *Manifest) map[string]bool {
	s := make(map[string]bool, len(m.Data.Domains))
	for _, d := range m.Data.Domains {
		s[d.Name] = true
	}
	return s
}

func dbAliasSet(m *Manifest) map[string]bool {
	s := make(map[string]bool, len(m.Data.Databases))
	for _, db := range m.Data.Databases {
		s[db.Alias] = true
	}
	return s
}

func authRoleSet(m *Manifest) map[string]bool {
	if m.Backend.Auth == nil {
		return nil
	}
	s := make(map[string]bool, len(m.Backend.Auth.Roles))
	for _, r := range m.Backend.Auth.Roles {
		s[r.Name] = true
	}
	return s
}

// ── validation rules ─────────────────────────────────────────────────────────

func validateServiceEnvRefs(m *Manifest) []ValidationError {
	envs := envNameSet(m)
	var errs []ValidationError
	for i, svc := range m.Backend.Services {
		if svc.Environment != "" && !envs[svc.Environment] {
			errs = append(errs, ValidationError{
				Pillar:  "backend",
				Path:    fmt.Sprintf("backend.services[%d].environment", i),
				Code:    "stale_env_ref",
				Message: fmt.Sprintf("service %q references unknown environment %q", svc.Name, svc.Environment),
			})
		}
	}
	return errs
}

func validateMonolithEnvRef(m *Manifest) []ValidationError {
	if m.Backend.MonolithEnvironment == "" {
		return nil
	}
	envs := envNameSet(m)
	if !envs[m.Backend.MonolithEnvironment] {
		return []ValidationError{{
			Pillar:  "backend",
			Path:    "backend.monolith_environment",
			Code:    "stale_env_ref",
			Message: fmt.Sprintf("monolith environment %q not found in infrastructure environments", m.Backend.MonolithEnvironment),
		}}
	}
	return nil
}

func validateMessagingEnvRef(m *Manifest) []ValidationError {
	if m.Backend.Messaging == nil || m.Backend.Messaging.Environment == "" {
		return nil
	}
	envs := envNameSet(m)
	if !envs[m.Backend.Messaging.Environment] {
		return []ValidationError{{
			Pillar:  "backend",
			Path:    "backend.messaging.environment",
			Code:    "stale_env_ref",
			Message: fmt.Sprintf("messaging environment %q not found in infrastructure environments", m.Backend.Messaging.Environment),
		}}
	}
	return nil
}

func validateAPIGWEnvRef(m *Manifest) []ValidationError {
	if m.Backend.APIGateway == nil || m.Backend.APIGateway.Environment == "" {
		return nil
	}
	envs := envNameSet(m)
	if !envs[m.Backend.APIGateway.Environment] {
		return []ValidationError{{
			Pillar:  "backend",
			Path:    "backend.api_gateway.environment",
			Code:    "stale_env_ref",
			Message: fmt.Sprintf("API gateway environment %q not found in infrastructure environments", m.Backend.APIGateway.Environment),
		}}
	}
	return nil
}

func validateEndpointServiceRefs(m *Manifest) []ValidationError {
	svcs := serviceNameSet(m)
	var errs []ValidationError
	for i, ep := range m.Contracts.Endpoints {
		if ep.ServiceUnit != "" && !svcs[ep.ServiceUnit] {
			errs = append(errs, ValidationError{
				Pillar:  "contracts",
				Path:    fmt.Sprintf("contracts.endpoints[%d].service_unit", i),
				Code:    "stale_svc_ref",
				Message: fmt.Sprintf("endpoint %q references unknown service %q", ep.NamePath, ep.ServiceUnit),
			})
		}
	}
	return errs
}

func validateEndpointDTORefs(m *Manifest) []ValidationError {
	dtos := dtoNameSet(m)
	if len(dtos) == 0 {
		return nil // no DTOs defined — nothing to validate against
	}
	var errs []ValidationError
	for i, ep := range m.Contracts.Endpoints {
		if ep.RequestDTO != "" && !dtos[ep.RequestDTO] {
			errs = append(errs, ValidationError{
				Pillar:  "contracts",
				Path:    fmt.Sprintf("contracts.endpoints[%d].request_dto", i),
				Code:    "stale_dto_ref",
				Message: fmt.Sprintf("endpoint %q references unknown request DTO %q", ep.NamePath, ep.RequestDTO),
			})
		}
		if ep.ResponseDTO != "" && !dtos[ep.ResponseDTO] {
			errs = append(errs, ValidationError{
				Pillar:  "contracts",
				Path:    fmt.Sprintf("contracts.endpoints[%d].response_dto", i),
				Code:    "stale_dto_ref",
				Message: fmt.Sprintf("endpoint %q references unknown response DTO %q", ep.NamePath, ep.ResponseDTO),
			})
		}
	}
	return errs
}

func validateEndpointAuthRoles(m *Manifest) []ValidationError {
	if m.Backend.Auth == nil {
		return nil
	}
	roles := authRoleSet(m)
	var errs []ValidationError
	for i, ep := range m.Contracts.Endpoints {
		if ep.AuthRequired != "true" || ep.AuthRoles == "" {
			continue
		}
		if !roles[ep.AuthRoles] {
			errs = append(errs, ValidationError{
				Pillar:  "contracts",
				Path:    fmt.Sprintf("contracts.endpoints[%d].auth_roles", i),
				Code:    "auth_role_mismatch",
				Message: fmt.Sprintf("endpoint %q requires role %q not defined in backend auth", ep.NamePath, ep.AuthRoles),
			})
		}
	}
	return errs
}

func validateFrontendAuth(m *Manifest) []ValidationError {
	if m.Backend.Auth != nil {
		return nil // backend auth exists — OK
	}
	var errs []ValidationError
	for i, p := range m.Frontend.Pages {
		if p.AuthRequired == "true" {
			errs = append(errs, ValidationError{
				Pillar:  "frontend",
				Path:    fmt.Sprintf("frontend.pages[%d].auth_required", i),
				Code:    "frontend_auth_no_backend",
				Message: fmt.Sprintf("page %q requires auth but no backend auth strategy is configured", p.Name),
			})
		}
	}
	return errs
}

func validatePageAuthRoles(m *Manifest) []ValidationError {
	if m.Backend.Auth == nil {
		return nil
	}
	roles := authRoleSet(m)
	var errs []ValidationError
	for i, p := range m.Frontend.Pages {
		if p.AuthRoles == "" {
			continue
		}
		if !roles[p.AuthRoles] {
			errs = append(errs, ValidationError{
				Pillar:  "frontend",
				Path:    fmt.Sprintf("frontend.pages[%d].auth_roles", i),
				Code:    "page_auth_role_mismatch",
				Message: fmt.Sprintf("page %q requires role %q not defined in backend auth", p.Name, p.AuthRoles),
			})
		}
	}
	return errs
}

func validateCommLinkServiceRefs(m *Manifest) []ValidationError {
	svcs := serviceNameSet(m)
	var errs []ValidationError
	for i, cl := range m.Backend.CommLinks {
		if cl.From != "" && !svcs[cl.From] {
			errs = append(errs, ValidationError{
				Pillar:  "backend",
				Path:    fmt.Sprintf("backend.comm_links[%d].from", i),
				Code:    "stale_comm_svc",
				Message: fmt.Sprintf("comm link references unknown source service %q", cl.From),
			})
		}
		if cl.To != "" && !svcs[cl.To] {
			errs = append(errs, ValidationError{
				Pillar:  "backend",
				Path:    fmt.Sprintf("backend.comm_links[%d].to", i),
				Code:    "stale_comm_svc",
				Message: fmt.Sprintf("comm link references unknown target service %q", cl.To),
			})
		}
	}
	return errs
}

func validateEventServiceRefs(m *Manifest) []ValidationError {
	svcs := serviceNameSet(m)
	var errs []ValidationError
	for i, ev := range m.Backend.Events {
		if ev.PublisherService != "" && !svcs[ev.PublisherService] {
			errs = append(errs, ValidationError{
				Pillar:  "backend",
				Path:    fmt.Sprintf("backend.events[%d].publisher_service", i),
				Code:    "stale_event_svc",
				Message: fmt.Sprintf("event %q references unknown publisher service %q", ev.Name, ev.PublisherService),
			})
		}
		if ev.ConsumerService != "" && !svcs[ev.ConsumerService] {
			errs = append(errs, ValidationError{
				Pillar:  "backend",
				Path:    fmt.Sprintf("backend.events[%d].consumer_service", i),
				Code:    "stale_event_svc",
				Message: fmt.Sprintf("event %q references unknown consumer service %q", ev.Name, ev.ConsumerService),
			})
		}
	}
	return errs
}

func validateRepoEntityRefs(m *Manifest) []ValidationError {
	domains := domainNameSet(m)
	if len(domains) == 0 {
		return nil
	}
	var errs []ValidationError
	for i, svc := range m.Backend.Services {
		for j, repo := range svc.Repositories {
			if repo.EntityRef != "" && !domains[repo.EntityRef] {
				errs = append(errs, ValidationError{
					Pillar:  "backend",
					Path:    fmt.Sprintf("backend.services[%d].repositories[%d].entity_ref", i, j),
					Code:    "stale_entity_ref",
					Message: fmt.Sprintf("repository %q in service %q references unknown domain %q", repo.Name, svc.Name, repo.EntityRef),
				})
			}
		}
	}
	return errs
}

func validateRepoTargetDB(m *Manifest) []ValidationError {
	dbs := dbAliasSet(m)
	if len(dbs) == 0 {
		return nil
	}
	var errs []ValidationError
	for i, svc := range m.Backend.Services {
		for j, repo := range svc.Repositories {
			if repo.TargetDB != "" && !dbs[repo.TargetDB] {
				errs = append(errs, ValidationError{
					Pillar:  "backend",
					Path:    fmt.Sprintf("backend.services[%d].repositories[%d].target_db", i, j),
					Code:    "stale_db_ref",
					Message: fmt.Sprintf("repository %q in service %q references unknown database %q", repo.Name, svc.Name, repo.TargetDB),
				})
			}
		}
	}
	return errs
}

func validateCacheRefs(m *Manifest) []ValidationError {
	if m.Backend.WAF == nil || m.Backend.WAF.RateLimitBackend == "" {
		return nil
	}
	dbs := dbAliasSet(m)
	if !dbs[m.Backend.WAF.RateLimitBackend] {
		return []ValidationError{{
			Pillar:  "backend",
			Path:    "backend.waf.rate_limit_backend",
			Code:    "stale_cache_ref",
			Message: fmt.Sprintf("WAF rate_limit_backend references unknown database %q", m.Backend.WAF.RateLimitBackend),
		}}
	}
	return nil
}

func validateFileStorageServiceRef(m *Manifest) []ValidationError {
	svcs := serviceNameSet(m)
	var errs []ValidationError
	for i, fs := range m.Data.FileStorages {
		if fs.UsedByService != "" && !svcs[fs.UsedByService] {
			errs = append(errs, ValidationError{
				Pillar:  "data",
				Path:    fmt.Sprintf("data.file_storages[%d].used_by_service", i),
				Code:    "stale_fs_svc",
				Message: fmt.Sprintf("file storage references unknown service %q", fs.UsedByService),
			})
		}
	}
	return errs
}

func validateGovernanceDBRefs(m *Manifest) []ValidationError {
	dbs := dbAliasSet(m)
	if len(dbs) == 0 {
		return nil
	}
	var errs []ValidationError
	for i, gov := range m.Data.Governances {
		for _, dbName := range gov.Databases {
			if dbName != "" && !dbs[dbName] {
				errs = append(errs, ValidationError{
					Pillar:  "data",
					Path:    fmt.Sprintf("data.governances[%d].databases", i),
					Code:    "stale_gov_db",
					Message: fmt.Sprintf("governance %q references unknown database %q", gov.Name, dbName),
				})
			}
		}
	}
	return errs
}

func validateDBEnvRefs(m *Manifest) []ValidationError {
	envs := envNameSet(m)
	var errs []ValidationError
	for i, db := range m.Data.Databases {
		if db.Environment != "" && !envs[db.Environment] {
			errs = append(errs, ValidationError{
				Pillar:  "data",
				Path:    fmt.Sprintf("data.databases[%d].environment", i),
				Code:    "stale_env_ref",
				Message: fmt.Sprintf("database %q references unknown environment %q", db.Alias, db.Environment),
			})
		}
	}
	return errs
}
