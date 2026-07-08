package core

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/dangduoc08/ginject/common"
)

type benchPrefixTargetController struct{ common.REST }

func (c benchPrefixTargetController) NewController() Controller      { return c }
func (c benchPrefixTargetController) READ_benchprefixtarget() string { return "ok" }

// seedGlobalPrefixArrNoise populates globalPrefixArr with n entries that never
// match benchPrefixTargetController, simulating a large real-world app where
// most registered controllers are unrelated to the one being looked up.
func seedGlobalPrefixArrNoise(n int) {
	noiseKey := genFieldKey(reflect.TypeOf(struct{ x int }{}))
	for i := 0; i < n; i++ {
		key := "[" + strconv.Itoa(i) + "]" + noiseKey
		globalPrefixArr[key] = []string{"/noise"}
	}
}

func BenchmarkControllerModulePrefixes(b *testing.B) {
	resetModuleGlobals()
	defer resetModuleGlobals()

	targetType := reflect.TypeOf(benchPrefixTargetController{})
	seedGlobalPrefixArrNoise(2000)
	globalPrefixArr[genFieldKey(targetType)] = []string{"/v1"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		controllerModulePrefixes(targetType)
	}
}

func BenchmarkRegisterControllerPrefixes(b *testing.B) {
	controllers := make([]Controller, 100)
	for i := range controllers {
		controllers[i] = benchPrefixTargetController{}
	}
	m := ModuleBuilder().Controllers(controllers...).Build()
	m.Prefix("v1")

	resetModuleGlobals()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.registerControllerPrefixes()
	}
}

type benchModuleController0 struct{ common.REST }

func (c benchModuleController0) NewController() Controller         { return c }
func (c benchModuleController0) READ_benchmoduleresource0() string { return "ok" }

type benchModuleController1 struct{ common.REST }

func (c benchModuleController1) NewController() Controller         { return c }
func (c benchModuleController1) READ_benchmoduleresource1() string { return "ok" }

type benchModuleController2 struct{ common.REST }

func (c benchModuleController2) NewController() Controller         { return c }
func (c benchModuleController2) READ_benchmoduleresource2() string { return "ok" }

type benchModuleController3 struct{ common.REST }

func (c benchModuleController3) NewController() Controller         { return c }
func (c benchModuleController3) READ_benchmoduleresource3() string { return "ok" }

type benchModuleController4 struct{ common.REST }

func (c benchModuleController4) NewController() Controller         { return c }
func (c benchModuleController4) READ_benchmoduleresource4() string { return "ok" }

// BenchmarkNewModule_FiveControllers exercises the full NewModule bootstrap
// pipeline (provider hoisting, prefix registration, controller binding) for a
// module with a realistic handful of controllers. mainModulePtr and friends
// are package-level latches, so each iteration must reset them - only timing
// the bootstrap itself, not the reset.
func BenchmarkNewModule_FiveControllers(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		resetModuleGlobals()
		m := ModuleBuilder().
			Controllers(
				benchModuleController0{},
				benchModuleController1{},
				benchModuleController2{},
				benchModuleController3{},
				benchModuleController4{},
			).
			Build()
		b.StartTimer()

		m.NewModule()
	}
}
