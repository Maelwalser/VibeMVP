package dag

import (
	"testing"

	"github.com/vibe-menu/internal/manifest"
)

// minimalMonolith returns a manifest with a single monolith service.
func minimalMonolith() *manifest.Manifest {
	return &manifest.Manifest{
		Backend: manifest.BackendPillar{
			ArchPattern: manifest.ArchMonolith,
			Services: []manifest.ServiceDef{
				{Name: "monolith", Language: "Go", Framework: "Gin"},
			},
		},
	}
}

func buildDAG(t *testing.T, m *manifest.Manifest) *DAG {
	t.Helper()
	d, err := (&Builder{}).Build(m)
	if err != nil {
		t.Fatalf("Builder.Build() unexpected error: %v", err)
	}
	return d
}

func assertTaskPresent(t *testing.T, d *DAG, id string) {
	t.Helper()
	if _, ok := d.Tasks[id]; !ok {
		t.Errorf("expected task %q in DAG, not found; tasks=%v", id, taskIDs(d))
	}
}

func assertTaskAbsent(t *testing.T, d *DAG, id string) {
	t.Helper()
	if _, ok := d.Tasks[id]; ok {
		t.Errorf("unexpected task %q found in DAG", id)
	}
}

func taskIDs(d *DAG) []string {
	ids := make([]string, 0, len(d.Tasks))
	for id := range d.Tasks {
		ids = append(ids, id)
	}
	return ids
}

func TestBuild_DataTasksAlwaysPresent(t *testing.T) {
	d := buildDAG(t, minimalMonolith())
	assertTaskPresent(t, d, "data.schemas")
	assertTaskPresent(t, d, "data.migrations")
}

func TestBuild_Monolith_ServiceChain(t *testing.T) {
	d := buildDAG(t, minimalMonolith())
	for _, id := range []string{
		"svc.monolith.plan",
		"svc.monolith.deps",
		"svc.monolith.repository",
		"svc.monolith.service",
		"svc.monolith.handler",
		"svc.monolith.bootstrap",
	} {
		assertTaskPresent(t, d, id)
	}
}

func TestBuild_Monolith_ServiceChain_DependencyOrder(t *testing.T) {
	d := buildDAG(t, minimalMonolith())
	order := d.Order()
	pos := make(map[string]int, len(order))
	for i, id := range order {
		pos[id] = i
	}
	chain := []string{
		"svc.monolith.plan",
		"svc.monolith.deps",
		"svc.monolith.repository",
		"svc.monolith.service",
		"svc.monolith.handler",
		"svc.monolith.bootstrap",
	}
	for i := 1; i < len(chain); i++ {
		if pos[chain[i-1]] >= pos[chain[i]] {
			t.Errorf("%s must precede %s in topological order", chain[i-1], chain[i])
		}
	}
}

func TestBuild_Microservices_TwoServices(t *testing.T) {
	m := &manifest.Manifest{
		Backend: manifest.BackendPillar{
			ArchPattern: manifest.ArchMicroservices,
			Services: []manifest.ServiceDef{
				{Name: "api", Language: "Go", Framework: "Gin"},
				{Name: "worker", Language: "Go", Framework: "Gin"},
			},
		},
	}
	d := buildDAG(t, m)

	for _, svc := range []string{"api", "worker"} {
		assertTaskPresent(t, d, "svc."+svc+".bootstrap")
		assertTaskPresent(t, d, "svc."+svc+".plan")
	}
}

func TestBuild_WithAuth_AddsAuthTask(t *testing.T) {
	m := minimalMonolith()
	m.Backend.Auth = &manifest.AuthConfig{Strategy: "JWT"}
	d := buildDAG(t, m)
	assertTaskPresent(t, d, "backend.auth")
}

func TestBuild_WithoutAuth_NoAuthTask(t *testing.T) {
	d := buildDAG(t, minimalMonolith())
	assertTaskAbsent(t, d, "backend.auth")
}

func TestBuild_WithMessaging_AddsMessagingTask(t *testing.T) {
	m := minimalMonolith()
	m.Backend.Messaging = &manifest.MessagingConfig{BrokerTech: "Kafka"}
	d := buildDAG(t, m)
	assertTaskPresent(t, d, "backend.messaging")
}

func TestBuild_WithoutMessaging_NoMessagingTask(t *testing.T) {
	d := buildDAG(t, minimalMonolith())
	assertTaskAbsent(t, d, "backend.messaging")
}

func TestBuild_NoContractsWhenEmpty(t *testing.T) {
	// No DTOs and no endpoints → no contracts task
	d := buildDAG(t, minimalMonolith())
	assertTaskAbsent(t, d, "contracts")
}

func TestBuild_ContractsPresent_WhenDTOsDefined(t *testing.T) {
	m := minimalMonolith()
	m.Contracts.DTOs = []manifest.DTODef{{Name: "UserDTO"}}
	d := buildDAG(t, m)
	assertTaskPresent(t, d, "contracts")
}

func TestBuild_ContractsDependsOnServices(t *testing.T) {
	m := minimalMonolith()
	m.Contracts.DTOs = []manifest.DTODef{{Name: "UserDTO"}}
	d := buildDAG(t, m)

	contracts := d.Tasks["contracts"]
	if contracts == nil {
		t.Fatal("contracts task not found")
	}

	found := false
	for _, dep := range contracts.Dependencies {
		if dep == "svc.monolith.bootstrap" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("contracts should depend on svc.monolith.bootstrap, deps=%v", contracts.Dependencies)
	}
}

func TestBuild_NoFrontend_WhenTechEmpty(t *testing.T) {
	d := buildDAG(t, minimalMonolith())
	assertTaskAbsent(t, d, "frontend")
}

func TestBuild_FrontendPresent_WhenFrameworkSet(t *testing.T) {
	m := minimalMonolith()
	m.Frontend.Tech = &manifest.FrontendTechConfig{Framework: "Next.js", Language: "TypeScript"}
	d := buildDAG(t, m)
	assertTaskPresent(t, d, "frontend")
}

func TestBuild_InfraDockerAlwaysPresent(t *testing.T) {
	// infra.docker is always added regardless of manifest content
	d := buildDAG(t, minimalMonolith())
	assertTaskPresent(t, d, "infra.docker")
}

func TestBuild_InfraTerraform_OnlyWhenIaCToolSet(t *testing.T) {
	m := minimalMonolith()
	d := buildDAG(t, m)
	assertTaskAbsent(t, d, "infra.terraform")

	m.Infra.CICD = &manifest.CICDConfig{IaCTool: "Terraform"}
	d = buildDAG(t, m)
	assertTaskPresent(t, d, "infra.terraform")
}

func TestBuild_InfraCI_OnlyWhenPlatformSet(t *testing.T) {
	m := minimalMonolith()
	d := buildDAG(t, m)
	assertTaskAbsent(t, d, "infra.cicd")

	m.Infra.CICD = &manifest.CICDConfig{Platform: "GitHub Actions"}
	d = buildDAG(t, m)
	assertTaskPresent(t, d, "infra.cicd")
}

func TestBuild_CrossCutTesting_WhenUnitSet(t *testing.T) {
	m := minimalMonolith()
	m.CrossCut.Testing = &manifest.TestingConfig{Unit: "go test"}
	d := buildDAG(t, m)
	assertTaskPresent(t, d, "crosscut.testing")
}

func TestBuild_CrossCutDocs_WhenAPIDocsSet(t *testing.T) {
	m := minimalMonolith()
	m.CrossCut.Docs = &manifest.DocsConfig{APIDocs: "OpenAPI"}
	d := buildDAG(t, m)
	assertTaskPresent(t, d, "crosscut.docs")
}

func TestBuild_CrossCutDocs_WhenPerProtocolFormats(t *testing.T) {
	m := minimalMonolith()
	m.CrossCut.Docs = &manifest.DocsConfig{
		PerProtocolFormats: map[string]string{"REST": "OpenAPI 3.1"},
	}
	d := buildDAG(t, m)
	assertTaskPresent(t, d, "crosscut.docs")
}

// ── ConfigRef resolution ─────────────────────────────────────────────────────

func TestBuild_ConfigRef_ResolvesLanguageFramework(t *testing.T) {
	m := &manifest.Manifest{
		Backend: manifest.BackendPillar{
			ArchPattern: manifest.ArchMicroservices,
			StackConfigs: []manifest.StackConfig{
				{Name: "go-fiber", Language: "Go", LanguageVersion: "1.26", Framework: "Fiber", FrameworkVersion: "v2"},
			},
			Services: []manifest.ServiceDef{
				{Name: "api", ConfigRef: "go-fiber"},
			},
		},
	}
	d := buildDAG(t, m)

	task := d.Tasks["svc.api.plan"]
	if task == nil {
		t.Fatal("svc.api.plan not found")
	}
	if task.Payload.Service.Language != "Go" {
		t.Errorf("Language = %q, want %q", task.Payload.Service.Language, "Go")
	}
	if task.Payload.Service.Framework != "Fiber" {
		t.Errorf("Framework = %q, want %q", task.Payload.Service.Framework, "Fiber")
	}
}

func TestBuild_ConfigRef_InlineOverridesStackConfig(t *testing.T) {
	m := &manifest.Manifest{
		Backend: manifest.BackendPillar{
			ArchPattern: manifest.ArchMicroservices,
			StackConfigs: []manifest.StackConfig{
				{Name: "go-fiber", Language: "Go", Framework: "Fiber"},
			},
			Services: []manifest.ServiceDef{
				{Name: "api", ConfigRef: "go-fiber", Language: "Go", Framework: "Gin"},
			},
		},
	}
	d := buildDAG(t, m)

	task := d.Tasks["svc.api.plan"]
	if task.Payload.Service.Framework != "Gin" {
		t.Errorf("inline Framework should take precedence, got %q", task.Payload.Service.Framework)
	}
}

func TestBuild_ConfigRef_UnknownRef_NoOp(t *testing.T) {
	m := &manifest.Manifest{
		Backend: manifest.BackendPillar{
			ArchPattern: manifest.ArchMicroservices,
			Services: []manifest.ServiceDef{
				{Name: "api", ConfigRef: "nonexistent"},
			},
		},
	}
	d := buildDAG(t, m)

	task := d.Tasks["svc.api.plan"]
	if task.Payload.Service.Language != "" {
		t.Errorf("Language should remain empty for unknown ConfigRef, got %q", task.Payload.Service.Language)
	}
}

// ── Description injection ───────────────────────────────────────────────────

func TestBuild_Description_InjectedIntoKeyTasks(t *testing.T) {
	m := minimalMonolith()
	m.Description = "A collaborative blogging platform"
	m.Frontend.Tech = &manifest.FrontendTechConfig{Framework: "Next.js", Language: "TypeScript"}
	d := buildDAG(t, m)

	for _, id := range []string{"data.schemas", "svc.monolith.plan", "svc.monolith.service", "frontend"} {
		task := d.Tasks[id]
		if task == nil {
			t.Errorf("task %q not found", id)
			continue
		}
		if task.Payload.Description != "A collaborative blogging platform" {
			t.Errorf("task %q Description = %q, want project description", id, task.Payload.Description)
		}
	}
}

func TestBuild_Description_OmittedFromInfraTasks(t *testing.T) {
	m := minimalMonolith()
	m.Description = "Some project"
	d := buildDAG(t, m)

	task := d.Tasks["infra.docker"]
	if task == nil {
		t.Fatal("infra.docker not found")
	}
	if task.Payload.Description != "" {
		t.Errorf("infra.docker should not have Description, got %q", task.Payload.Description)
	}
}

// ── Events injection ─────────────────────────────────────────────────────────

func TestBuild_Events_InjectedIntoServiceTasks(t *testing.T) {
	m := &manifest.Manifest{
		Backend: manifest.BackendPillar{
			ArchPattern: manifest.ArchMicroservices,
			Services: []manifest.ServiceDef{
				{Name: "users", Language: "Go", Framework: "Fiber"},
				{Name: "orders", Language: "Go", Framework: "Fiber"},
			},
			Events: []manifest.EventDef{
				{Name: "UserCreated", PublisherService: "users", DTO: "UserCreatedEvent"},
				{Name: "OrderPlaced", PublisherService: "orders", ConsumerService: "users", DTO: "OrderPlacedEvent"},
			},
		},
	}
	d := buildDAG(t, m)

	// users service should see both events (publisher of UserCreated, consumer of OrderPlaced)
	usersLogic := d.Tasks["svc.users.service"]
	if usersLogic == nil {
		t.Fatal("svc.users.service not found")
	}
	if len(usersLogic.Payload.Events) != 2 {
		t.Errorf("users service should have 2 events, got %d", len(usersLogic.Payload.Events))
	}

	// orders service should see only OrderPlaced (publisher)
	ordersLogic := d.Tasks["svc.orders.service"]
	if ordersLogic == nil {
		t.Fatal("svc.orders.service not found")
	}
	if len(ordersLogic.Payload.Events) != 1 {
		t.Errorf("orders service should have 1 event, got %d", len(ordersLogic.Payload.Events))
	}

	// Events should also be on plan and handler tasks
	usersPlan := d.Tasks["svc.users.plan"]
	if usersPlan == nil || len(usersPlan.Payload.Events) != 2 {
		t.Errorf("users plan should have 2 events")
	}
	usersHandler := d.Tasks["svc.users.handler"]
	if usersHandler == nil || len(usersHandler.Payload.Events) != 2 {
		t.Errorf("users handler should have 2 events")
	}
}

// ── Governances injection ───────────────────────────────────────────────────

func TestBuild_Governances_InjectedIntoDataTasks(t *testing.T) {
	m := minimalMonolith()
	m.Data.Governances = []manifest.DataGovernanceConfig{
		{Name: "GDPR", PIIEncryption: "AES-256", RetentionPolicy: "7 years"},
	}
	d := buildDAG(t, m)

	for _, id := range []string{"data.schemas", "data.migrations"} {
		task := d.Tasks[id]
		if task == nil {
			t.Errorf("task %q not found", id)
			continue
		}
		if len(task.Payload.Governances) != 1 {
			t.Errorf("task %q should have 1 governance config, got %d", id, len(task.Payload.Governances))
		}
	}
}

// ── WAF injection ───────────────────────────────────────────────────────────

func TestBuild_WAF_InjectedIntoHandlerAndGateway(t *testing.T) {
	m := minimalMonolith()
	m.Backend.WAF = &manifest.WAFConfig{RateLimitStrategy: "token-bucket"}
	m.Backend.APIGateway = &manifest.APIGatewayConfig{Technology: "Kong"}
	d := buildDAG(t, m)

	handler := d.Tasks["svc.monolith.handler"]
	if handler == nil || handler.Payload.WAF == nil {
		t.Error("handler should have WAF config")
	}

	gateway := d.Tasks["backend.gateway"]
	if gateway == nil || gateway.Payload.WAF == nil {
		t.Error("gateway should have WAF config")
	}

	bootstrap := d.Tasks["svc.monolith.bootstrap"]
	if bootstrap == nil || bootstrap.Payload.WAF == nil {
		t.Error("bootstrap should have WAF config")
	}
}

// ── CommLinks in service logic ──────────────────────────────────────────────

func TestBuild_CommLinks_InjectedIntoServiceLogic(t *testing.T) {
	m := &manifest.Manifest{
		Backend: manifest.BackendPillar{
			ArchPattern: manifest.ArchMicroservices,
			Services: []manifest.ServiceDef{
				{Name: "users", Language: "Go", Framework: "Fiber"},
				{Name: "orders", Language: "Go", Framework: "Fiber"},
			},
			CommLinks: []manifest.CommLink{
				{From: "orders", To: "users", Protocol: "REST", SyncAsync: "sync"},
			},
		},
	}
	d := buildDAG(t, m)

	ordersLogic := d.Tasks["svc.orders.service"]
	if ordersLogic == nil {
		t.Fatal("svc.orders.service not found")
	}
	if len(ordersLogic.Payload.CommLinks) != 1 {
		t.Errorf("orders service logic should have 1 comm link, got %d", len(ordersLogic.Payload.CommLinks))
	}

	usersLogic := d.Tasks["svc.users.service"]
	if usersLogic == nil {
		t.Fatal("svc.users.service not found")
	}
	if len(usersLogic.Payload.CommLinks) != 1 {
		t.Errorf("users service logic should have 1 comm link (is the 'To' side), got %d", len(usersLogic.Payload.CommLinks))
	}
}

func TestBuild_ValidDAG_NoErrors(t *testing.T) {
	// Full-featured manifest should produce a valid DAG with no build errors
	m := &manifest.Manifest{
		Backend: manifest.BackendPillar{
			ArchPattern: manifest.ArchMicroservices,
			Services: []manifest.ServiceDef{
				{Name: "users", Language: "Go", Framework: "Gin"},
				{Name: "orders", Language: "Go", Framework: "Gin"},
			},
			Auth:      &manifest.AuthConfig{Strategy: "JWT"},
			Messaging: &manifest.MessagingConfig{BrokerTech: "Kafka"},
		},
		Contracts: manifest.ContractsPillar{
			DTOs:      []manifest.DTODef{{Name: "UserDTO"}},
			Endpoints: []manifest.EndpointDef{{NamePath: "/users/:id", Protocol: "REST"}},
		},
		Frontend: manifest.FrontendPillar{
			Tech: &manifest.FrontendTechConfig{Framework: "Next.js", Language: "TypeScript"},
		},
	}
	_, err := (&Builder{}).Build(m)
	if err != nil {
		t.Errorf("Build() returned error for valid full manifest: %v", err)
	}
}
