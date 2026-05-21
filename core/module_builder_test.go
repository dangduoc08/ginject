package core

import (
	"testing"

	"github.com/dangduoc08/ginject/testutils"
)

func TestModuleBuilderImports(t *testing.T) {
	child := ModuleBuilder().Build()
	b := ModuleBuilder().Imports(child)
	if len(b.imports) != 1 {
		t.Error(testutils.DiffMessage(len(b.imports), 1, "Imports length"))
	}
}

func TestModuleBuilderProviders(t *testing.T) {
	p := &mockProvider{}
	b := ModuleBuilder().Providers(p)
	if len(b.providers) != 1 {
		t.Error(testutils.DiffMessage(len(b.providers), 1, "Providers length"))
	}
}

func TestModuleBuilderControllers(t *testing.T) {
	c := &mockController{}
	b := ModuleBuilder().Controllers(c)
	if len(b.controllers) != 1 {
		t.Error(testutils.DiffMessage(len(b.controllers), 1, "Controllers length"))
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
		t.Fatal(testutils.DiffMessage(nil, "*Module", "Build returned nil"))
		return
	}
	if m.ID() == "" {
		t.Error(testutils.DiffMessage("", "non-empty id", "module ID should not be empty"))
	}
	if len(m.providers) != 1 {
		t.Error(testutils.DiffMessage(len(m.providers), 1, "module providers length"))
	}
	if len(m.controllers) != 1 {
		t.Error(testutils.DiffMessage(len(m.controllers), 1, "module controllers length"))
	}
	if len(m.staticModules) != 1 {
		t.Error(testutils.DiffMessage(len(m.staticModules), 1, "module staticModules length"))
	}
}

func TestGetModuleTypeStaticOnly(t *testing.T) {
	child1 := ModuleBuilder().Build()
	child2 := ModuleBuilder().Build()
	b := ModuleBuilder().Imports(child1, child2)

	statics, dynamics := b.getModuleType()
	if len(statics) != 2 {
		t.Error(testutils.DiffMessage(len(statics), 2, "static modules count"))
	}
	if len(dynamics) != 0 {
		t.Error(testutils.DiffMessage(len(dynamics), 0, "dynamic modules count"))
	}
}

func TestGetModuleTypeInvalidPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error(testutils.DiffMessage(nil, "panic", "invalid import should panic"))
		}
	}()
	b := ModuleBuilder().Imports("not a module")
	b.getModuleType()
}

func TestModuleBuilderChaining(t *testing.T) {
	p1 := &mockProvider{}
	p2 := &mockProvider{}
	c1 := &mockController{}

	b := ModuleBuilder().Providers(p1).Providers(p2).Controllers(c1)
	if len(b.providers) != 2 {
		t.Error(testutils.DiffMessage(len(b.providers), 2, "chained providers"))
	}
	if len(b.controllers) != 1 {
		t.Error(testutils.DiffMessage(len(b.controllers), 1, "chained controllers"))
	}
}
