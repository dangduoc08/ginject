package core

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestModuleBuilderImports(t *testing.T) {
	child := ModuleBuilder().Build()
	b := ModuleBuilder().Imports(child)
	if len(b.imports) != 1 {
		t.Error(test.DiffMessage(len(b.imports), 1, "Imports length"))
	}
}

func TestModuleBuilderProviders(t *testing.T) {
	p := &mockProvider{}
	b := ModuleBuilder().Providers(p)
	if len(b.providers) != 1 {
		t.Error(test.DiffMessage(len(b.providers), 1, "Providers length"))
	}
}

func TestModuleBuilderControllers(t *testing.T) {
	c := &mockController{}
	b := ModuleBuilder().Controllers(c)
	if len(b.controllers) != 1 {
		t.Error(test.DiffMessage(len(b.controllers), 1, "Controllers length"))
	}
}

func TestModuleBuilderBuild(t *testing.T) {
	p := &mockProvider{}
	c := &mockController{}
	child := ModuleBuilder().Build()

	m := ModuleBuilder().
		Imports(child).
		Providers(p).
		Controllers(c).
		Build()

	if m == nil {
		t.Fatal(test.DiffMessage(nil, "*Module", "Build returned nil"))
		return
	}
	if m.ID() == "" {
		t.Error(test.DiffMessage("", "non-empty id", "module ID should not be empty"))
	}
	if len(m.providers) != 1 {
		t.Error(test.DiffMessage(len(m.providers), 1, "module providers length"))
	}
	if len(m.controllers) != 1 {
		t.Error(test.DiffMessage(len(m.controllers), 1, "module controllers length"))
	}
	if len(m.staticModules) != 1 {
		t.Error(test.DiffMessage(len(m.staticModules), 1, "module staticModules length"))
	}
}

func TestGetModuleTypeStaticOnly(t *testing.T) {
	child1 := ModuleBuilder().Build()
	child2 := ModuleBuilder().Build()
	b := ModuleBuilder().Imports(child1, child2)

	statics, dynamics := b.getModuleType()
	if len(statics) != 2 {
		t.Error(test.DiffMessage(len(statics), 2, "static modules count"))
	}
	if len(dynamics) != 0 {
		t.Error(test.DiffMessage(len(dynamics), 0, "dynamic modules count"))
	}
}

func TestGetModuleTypeInvalidPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error(test.DiffMessage(nil, "panic", "invalid import should panic"))
		}
	}()
	b := ModuleBuilder().Imports("not a module")
	b.getModuleType()
}

var testStaticModule = func() *Module {
	return ModuleBuilder().Build()
}

var (
	testGroupedModuleA = func() *Module { return ModuleBuilder().Build() }
	testGroupedModuleB = func() *Module { return ModuleBuilder().Build() }
)

func TestModuleBuilderBuildResolvesNameFromVarFuncLiteral(t *testing.T) {
	m := testStaticModule()
	if m.Name != "core.testStaticModule" {
		t.Error(test.DiffMessage(m.Name, "core.testStaticModule", "resolved module name"))
	}
}

func TestModuleBuilderBuildResolvesNameInGroupedVarDecl(t *testing.T) {
	a := testGroupedModuleA()
	b := testGroupedModuleB()
	if a.Name != "core.testGroupedModuleA" {
		t.Error(test.DiffMessage(a.Name, "core.testGroupedModuleA", "resolved module name"))
	}
	if b.Name != "core.testGroupedModuleB" {
		t.Error(test.DiffMessage(b.Name, "core.testGroupedModuleB", "resolved module name"))
	}
}

func testModuleFactory() *Module {
	return ModuleBuilder().Build()
}

var testFactoryBackedModule = testModuleFactory()

func TestModuleBuilderBuildResolvesNameFromVarAssignedFactoryCall(t *testing.T) {
	if testFactoryBackedModule.Name != "core.testFactoryBackedModule" {
		t.Error(test.DiffMessage(testFactoryBackedModule.Name, "core.testFactoryBackedModule", "resolved module name"))
	}
}

func TestModuleBuilderBuildFallsBackToFunctionNameWhenNoEnclosingVar(t *testing.T) {
	m := testModuleFactory()
	if m.Name != "core.testModuleFactory" {
		t.Error(test.DiffMessage(m.Name, "core.testModuleFactory", "resolved module name"))
	}
}

// Regression: an outer `var Outer = func() *core.Module {...}` closure that
// imports another module built via a factory call in its own body (mirrors
// sample/benchmarks/module.go's `Imports(httpclient.Register(nil))`) must not
// have that nested module's Name misattributed to Outer, since the nested
// call's frame is lexically inside Outer's closure body too.
var testOuterModuleWithNestedFactoryImport = func() *Module {
	return ModuleBuilder().Imports(testModuleFactory()).Build()
}

func TestModuleBuilderBuildDoesNotMisattributeNestedFactoryCallToOuterVar(t *testing.T) {
	outer := testOuterModuleWithNestedFactoryImport()
	if outer.Name != "core.testOuterModuleWithNestedFactoryImport" {
		t.Error(test.DiffMessage(outer.Name, "core.testOuterModuleWithNestedFactoryImport", "resolved outer module name"))
	}
	if len(outer.staticModules) != 1 {
		t.Fatalf("expected 1 static submodule, got %d", len(outer.staticModules))
	}

	nested := outer.staticModules[0]
	if nested.Name != "core.testModuleFactory" {
		t.Error(test.DiffMessage(nested.Name, "core.testModuleFactory", "nested module built as an import argument must resolve to its own factory, not the outer var"))
	}
	if nested.Name == outer.Name {
		t.Error(test.DiffMessage(nested.Name, "<different from outer.Name>", "outer and nested modules must not collapse to the same name"))
	}
}

func TestModuleBuilderBuildResolvesNameFromEnclosingTestFunc(t *testing.T) {
	m := ModuleBuilder().Build()
	if m.Name != "core.TestModuleBuilderBuildResolvesNameFromEnclosingTestFunc" {
		t.Error(test.DiffMessage(m.Name, "core.TestModuleBuilderBuildResolvesNameFromEnclosingTestFunc", "resolved module name"))
	}
}

func TestModuleNameCandidatesAtLine(t *testing.T) {
	src := `package sample

var Module = func() *core.Module {
	return nil
}

var (
	A = func() *core.Module { return nil }
	B = func() *core.Module { return nil }
)

var ConfModule = Register(&Options{})

func Direct() *core.Module {
	return nil
}
`

	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, "sample.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		line              int
		allowFuncLitMatch bool
		wantVarName       string
		wantFuncName      string
	}{
		{4, true, "Module", ""},
		{4, false, "", ""},
		{8, true, "A", ""},
		{8, false, "", ""},
		{9, true, "B", ""},
		{12, true, "ConfModule", ""},
		{12, false, "ConfModule", ""},
		{16, true, "", "Direct"},
		{16, false, "", "Direct"},
	}
	for _, c := range cases {
		gotVar, gotFunc := moduleNameCandidatesAtLine(fset, astFile, c.line, c.allowFuncLitMatch)
		if gotVar != c.wantVarName || gotFunc != c.wantFuncName {
			t.Error(test.DiffMessage([2]string{gotVar, gotFunc}, [2]string{c.wantVarName, c.wantFuncName}, "moduleNameCandidatesAtLine"))
		}
	}
}

func TestParseModuleFileUnreadableFile(t *testing.T) {
	_, _, ok := parseModuleFile("/nonexistent/file/does/not/exist.go")
	if ok {
		t.Error(test.DiffMessage(ok, false, "parseModuleFile with unreadable file"))
	}
}

func TestModuleBuilderChaining(t *testing.T) {
	p1 := &mockProvider{}
	p2 := &mockProvider{}
	c1 := &mockController{}

	b := ModuleBuilder().Providers(p1).Providers(p2).Controllers(c1)
	if len(b.providers) != 2 {
		t.Error(test.DiffMessage(len(b.providers), 2, "chained providers"))
	}
	if len(b.controllers) != 1 {
		t.Error(test.DiffMessage(len(b.controllers), 1, "chained controllers"))
	}
}
