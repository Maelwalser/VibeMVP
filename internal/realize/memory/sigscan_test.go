package memory

import (
	"strings"
	"testing"
)

// ── ExtractGoExportedTypeNames ────────────────────────────────────────────────

func TestExtractGoExportedTypeNames_ExportedStructs(t *testing.T) {
	content := `package foo

type User struct {
	ID   int
	Name string
}

type Order struct{}
`
	result := ExtractGoExportedTypeNames("models.go", content)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	for _, name := range []string{"User", "Order"} {
		if _, ok := result[name]; !ok {
			t.Errorf("expected exported type %q to be present", name)
		}
	}
}

func TestExtractGoExportedTypeNames_SkipsUnexportedTypes(t *testing.T) {
	content := `package foo

type user struct{ ID int }
type internalState int
`
	result := ExtractGoExportedTypeNames("models.go", content)
	if len(result) != 0 {
		t.Errorf("expected no exported types, got %v", result)
	}
}

func TestExtractGoExportedTypeNames_SkipsTestFiles(t *testing.T) {
	content := `package foo

type Exported struct{}
`
	result := ExtractGoExportedTypeNames("models_test.go", content)
	if result != nil {
		t.Errorf("expected nil for _test.go file, got %v", result)
	}
}

func TestExtractGoExportedTypeNames_SkipsNonGoFiles(t *testing.T) {
	result := ExtractGoExportedTypeNames("models.ts", "type Foo = {};")
	if result != nil {
		t.Errorf("expected nil for non-.go file, got %v", result)
	}
}

func TestExtractGoExportedTypeNames_Interface(t *testing.T) {
	content := `package foo

type Repository interface {
	FindByID(id int) (*User, error)
}
`
	result := ExtractGoExportedTypeNames("repo.go", content)
	if _, ok := result["Repository"]; !ok {
		t.Error("expected interface 'Repository' to be extracted")
	}
}

func TestExtractGoExportedTypeNames_TypeAlias(t *testing.T) {
	content := `package foo

type UserID = int
type Status = string
`
	result := ExtractGoExportedTypeNames("types.go", content)
	for _, name := range []string{"UserID", "Status"} {
		if _, ok := result[name]; !ok {
			t.Errorf("expected type alias %q to be present", name)
		}
	}
}

func TestExtractGoExportedTypeNames_Empty(t *testing.T) {
	result := ExtractGoExportedTypeNames("empty.go", "package foo\n")
	if len(result) != 0 {
		t.Errorf("expected empty map for file with no type declarations, got %v", result)
	}
}

func TestExtractGoExportedTypeNames_MixedExportedUnexported(t *testing.T) {
	content := `package foo

type Exported struct{}
type unexported struct{}
type AlsoExported interface{}
`
	result := ExtractGoExportedTypeNames("types.go", content)
	if _, ok := result["Exported"]; !ok {
		t.Error("expected 'Exported' to be present")
	}
	if _, ok := result["AlsoExported"]; !ok {
		t.Error("expected 'AlsoExported' to be present")
	}
	if _, ok := result["unexported"]; ok {
		t.Error("unexpected 'unexported' found in result")
	}
}

// ── ExtractServiceMethodSigs ──────────────────────────────────────────────────

func TestExtractServiceMethodSigs_Go(t *testing.T) {
	content := `package service

import "context"

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.repo.FindByEmail(ctx, email)
}

func (s *UserService) ListActive(ctx context.Context) ([]*User, error) {
	return s.repo.ListActive(ctx)
}

func (s *UserService) helper() {
	// unexported — should be skipped
}
`
	sigs := ExtractServiceMethodSigs("internal/service/user.go", content)
	if len(sigs) != 2 {
		t.Fatalf("expected 2 service method sigs, got %d: %v", len(sigs), sigs)
	}
	if !strings.Contains(sigs[0], "GetByEmail") {
		t.Errorf("expected GetByEmail, got %q", sigs[0])
	}
	if !strings.Contains(sigs[1], "ListActive") {
		t.Errorf("expected ListActive, got %q", sigs[1])
	}
}

func TestExtractServiceMethodSigs_ExcludesConstructors(t *testing.T) {
	content := `package service

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) NewSomething() *Thing {
	return &Thing{}
}

func (s *UserService) Update(ctx context.Context) error {
	return nil
}
`
	sigs := ExtractServiceMethodSigs("internal/service/user.go", content)
	if len(sigs) != 1 {
		t.Fatalf("expected 1 method sig (Update only), got %d: %v", len(sigs), sigs)
	}
	if !strings.Contains(sigs[0], "Update") {
		t.Errorf("expected Update, got %q", sigs[0])
	}
}

func TestExtractServiceMethodSigs_SkipsTestFiles(t *testing.T) {
	content := `package service

func (s *UserService) GetByEmail(ctx context.Context, email string) (*User, error) {
	return nil, nil
}
`
	sigs := ExtractServiceMethodSigs("internal/service/user_test.go", content)
	if len(sigs) != 0 {
		t.Errorf("expected 0 sigs for test file, got %d", len(sigs))
	}
}

// ── Multi-line signature assembly ────────────────────────────────────────────

func TestExtractGoCtorSigs_MultiLine(t *testing.T) {
	content := `package service

func NewUserService(
	repo UserRepository,
	logger Logger,
	cache CacheProvider,
) (*UserService, error) {
	return &UserService{}, nil
}

func NewSimple(repo Repo) *Simple {
	return &Simple{}
}
`
	sigs := ExtractConstructorSigs("service.go", content)
	if len(sigs) != 2 {
		t.Fatalf("expected 2 constructor sigs, got %d: %v", len(sigs), sigs)
	}
	// Multi-line sig should be assembled into one string.
	if !strings.Contains(sigs[0], "repo UserRepository") || !strings.Contains(sigs[0], "cache CacheProvider") {
		t.Errorf("multi-line sig not assembled correctly: %q", sigs[0])
	}
	// Single-line sig should still work.
	if !strings.Contains(sigs[1], "NewSimple") {
		t.Errorf("single-line sig broken: %q", sigs[1])
	}
}

func TestExtractServiceMethodSigs_MultiLine(t *testing.T) {
	content := `package service

func (s *UserService) GetByFilters(
	ctx context.Context,
	filters FilterOpts,
	pagination PaginationOpts,
) ([]*User, int, error) {
	return nil, 0, nil
}
`
	sigs := ExtractServiceMethodSigs("service.go", content)
	if len(sigs) != 1 {
		t.Fatalf("expected 1 method sig, got %d: %v", len(sigs), sigs)
	}
	if !strings.Contains(sigs[0], "GetByFilters") || !strings.Contains(sigs[0], "pagination PaginationOpts") {
		t.Errorf("multi-line method sig not assembled: %q", sigs[0])
	}
}

// ── extractGoSignatures ────────────────────────────────────────────────────────

func TestExtractGoSignatures_KeepsPackage(t *testing.T) {
	content := "package mypackage\n"
	got := extractGoSignatures(content)
	if !strings.Contains(got, "package mypackage") {
		t.Errorf("expected package declaration in output, got: %q", got)
	}
}

func TestExtractGoSignatures_KeepsImportBlock(t *testing.T) {
	content := `package foo

import (
	"fmt"
	"os"
)
`
	got := extractGoSignatures(content)
	if !strings.Contains(got, `import (`) {
		t.Errorf("expected import block in output, got: %q", got)
	}
	if !strings.Contains(got, `"fmt"`) {
		t.Errorf("expected 'fmt' import in output, got: %q", got)
	}
}

func TestExtractGoSignatures_PreservesTypeBodies(t *testing.T) {
	content := `package foo

type User struct {
	ID   int
	Name string
}
`
	got := extractGoSignatures(content)
	if !strings.Contains(got, "type User struct {") {
		t.Errorf("expected type declaration in output")
	}
	// Struct field declarations must be preserved — downstream agents need the
	// exact field layout to generate compatible code.
	if !strings.Contains(got, "ID   int") {
		t.Errorf("struct body fields should be preserved, got: %q", got)
	}
	if !strings.Contains(got, "Name string") {
		t.Errorf("struct body fields should be preserved, got: %q", got)
	}
}

func TestExtractGoSignatures_StripsFuncBodies(t *testing.T) {
	content := `package foo

func Greet(name string) string {
	return "hello " + name
}
`
	got := extractGoSignatures(content)
	if !strings.Contains(got, "func Greet") {
		t.Errorf("expected function signature in output")
	}
	if strings.Contains(got, `"hello "`) {
		t.Errorf("function body should be stripped, got: %q", got)
	}
}

func TestExtractGoSignatures_KeepsVarBlock(t *testing.T) {
	content := `package foo

import "errors"

var (
	ErrNotFound    = errors.New("not found")
	ErrConflict    = errors.New("already exists")
)
`
	got := extractGoSignatures(content)
	if !strings.Contains(got, "var (") {
		t.Errorf("expected var block in output, got: %q", got)
	}
	if !strings.Contains(got, "ErrNotFound") {
		t.Errorf("expected sentinel error in output, got: %q", got)
	}
	if !strings.Contains(got, "ErrConflict") {
		t.Errorf("expected sentinel error in output, got: %q", got)
	}
}

func TestExtractGoSignatures_KeepsConstBlock(t *testing.T) {
	content := `package foo

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)
`
	got := extractGoSignatures(content)
	if !strings.Contains(got, "const (") {
		t.Errorf("expected const block in output")
	}
	if !strings.Contains(got, `RoleAdmin`) {
		t.Errorf("expected const value in output")
	}
}

// ── extractTSSignatures ────────────────────────────────────────────────────────

func TestExtractTSSignatures_KeepsInterface(t *testing.T) {
	content := `export interface User {
  id: number;
  name: string;
}
`
	got := extractTSSignatures(content)
	if !strings.Contains(got, "export interface User {") {
		t.Errorf("expected interface declaration in output, got: %q", got)
	}
	// Interface field declarations must be preserved — downstream agents need
	// the exact field types to generate compatible TypeScript code.
	if !strings.Contains(got, "id: number") {
		t.Errorf("interface body fields should be preserved, got: %q", got)
	}
	if !strings.Contains(got, "name: string") {
		t.Errorf("interface body fields should be preserved, got: %q", got)
	}
}

func TestExtractTSSignatures_KeepsExportedFunction(t *testing.T) {
	content := `export function fetchUser(id: number): Promise<User> {
  return fetch('/users/' + id).then(r => r.json());
}
`
	got := extractTSSignatures(content)
	if !strings.Contains(got, "export function fetchUser") {
		t.Errorf("expected function signature in output, got: %q", got)
	}
	if strings.Contains(got, "fetch('/users/") {
		t.Errorf("function body should be stripped, got: %q", got)
	}
}

func TestExtractTSSignatures_KeepsTypeAlias(t *testing.T) {
	content := "export type UserID = number;\n"
	got := extractTSSignatures(content)
	if !strings.Contains(got, "export type UserID") {
		t.Errorf("expected type alias in output, got: %q", got)
	}
}

func TestExtractTSSignatures_KeepsImport(t *testing.T) {
	content := `import { User } from './types';

export function getUser(): User { return {}; }
`
	got := extractTSSignatures(content)
	if !strings.Contains(got, "import { User }") {
		t.Errorf("expected import statement in output, got: %q", got)
	}
}
