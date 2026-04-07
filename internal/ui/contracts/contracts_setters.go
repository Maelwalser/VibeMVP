package contracts

// ── stale reference clearing ─────────────────────────────────────────────────

// ClearStaleServiceRefs clears service_unit on committed endpoints when the
// referenced service no longer exists.
func (ce *ContractsEditor) ClearStaleServiceRefs(services []string) {
	svcSet := make(map[string]bool, len(services))
	for _, s := range services {
		svcSet[s] = true
	}
	for i := range ce.endpoints {
		if ce.endpoints[i].ServiceUnit != "" && !svcSet[ce.endpoints[i].ServiceUnit] {
			ce.endpoints[i].ServiceUnit = ""
		}
	}
	// Also clear external API called_by_service references.
	for i := range ce.externalAPIs {
		if ce.externalAPIs[i].CalledByService != "" && !svcSet[ce.externalAPIs[i].CalledByService] {
			ce.externalAPIs[i].CalledByService = ""
		}
	}
}

// ClearStaleDTORefs clears request_dto and response_dto on committed endpoints
// when the referenced DTO no longer exists.
func (ce *ContractsEditor) ClearStaleDTORefs() {
	dtoSet := make(map[string]bool, len(ce.dtos))
	for _, d := range ce.dtos {
		if d.Name != "" {
			dtoSet[d.Name] = true
		}
	}
	for i := range ce.endpoints {
		if ce.endpoints[i].RequestDTO != "" && !dtoSet[ce.endpoints[i].RequestDTO] {
			ce.endpoints[i].RequestDTO = ""
		}
		if ce.endpoints[i].ResponseDTO != "" && !dtoSet[ce.endpoints[i].ResponseDTO] {
			ce.endpoints[i].ResponseDTO = ""
		}
	}
}
