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
