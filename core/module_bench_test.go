package core

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/dangduoc08/ginject/common"
)

type benchPrefixTargetController struct{ common.HTTP }

func (c benchPrefixTargetController) NewController() Controller      { return c }
func (c benchPrefixTargetController) READ_benchprefixtarget() string { return "ok" }

// seedGlobalPrefixesByControllerNoise populates globalPrefixesByController
// with n entries that never match benchPrefixTargetController, simulating a
// large real-world app where most registered controllers are unrelated to
// the one being looked up.
func seedGlobalPrefixesByControllerNoise(n int) {
	noiseKey := genFieldKey(reflect.TypeOf(struct{ x int }{}))
	for i := 0; i < n; i++ {
		key := "[" + strconv.Itoa(i) + "]" + noiseKey
		globalPrefixesByController[key] = []string{"/noise"}
	}
}

func BenchmarkControllerModulePrefixes(b *testing.B) {
	resetModuleGlobals()
	defer resetModuleGlobals()

	targetType := reflect.TypeOf(benchPrefixTargetController{})
	seedGlobalPrefixesByControllerNoise(2000)
	globalPrefixesByController[genFieldKey(targetType)] = []string{"/v1"}

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

type benchModuleController0 struct{ common.HTTP }

func (c benchModuleController0) NewController() Controller         { return c }
func (c benchModuleController0) READ_benchmoduleresource0() string { return "ok" }

type benchModuleController1 struct{ common.HTTP }

func (c benchModuleController1) NewController() Controller         { return c }
func (c benchModuleController1) READ_benchmoduleresource1() string { return "ok" }

type benchModuleController2 struct{ common.HTTP }

func (c benchModuleController2) NewController() Controller         { return c }
func (c benchModuleController2) READ_benchmoduleresource2() string { return "ok" }

type benchModuleController3 struct{ common.HTTP }

func (c benchModuleController3) NewController() Controller         { return c }
func (c benchModuleController3) READ_benchmoduleresource3() string { return "ok" }

type benchModuleController4 struct{ common.HTTP }

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

type benchChainProvider0 struct{}

func (p benchChainProvider0) NewProvider() Provider { return p }

type benchChainProvider1 struct{ Prev benchChainProvider0 }

func (p benchChainProvider1) NewProvider() Provider { return p }

type benchChainProvider2 struct{ Prev benchChainProvider1 }

func (p benchChainProvider2) NewProvider() Provider { return p }

type benchChainProvider3 struct{ Prev benchChainProvider2 }

func (p benchChainProvider3) NewProvider() Provider { return p }

type benchChainProvider4 struct{ Prev benchChainProvider3 }

func (p benchChainProvider4) NewProvider() Provider { return p }

type benchChainProvider5 struct{ Prev benchChainProvider4 }

func (p benchChainProvider5) NewProvider() Provider { return p }

type benchChainProvider6 struct{ Prev benchChainProvider5 }

func (p benchChainProvider6) NewProvider() Provider { return p }

type benchChainProvider7 struct{ Prev benchChainProvider6 }

func (p benchChainProvider7) NewProvider() Provider { return p }

type benchChainProvider8 struct{ Prev benchChainProvider7 }

func (p benchChainProvider8) NewProvider() Provider { return p }

type benchChainProvider9 struct{ Prev benchChainProvider8 }

func (p benchChainProvider9) NewProvider() Provider { return p }

type benchChainProvider10 struct{ Prev benchChainProvider9 }

func (p benchChainProvider10) NewProvider() Provider { return p }

type benchChainProvider11 struct{ Prev benchChainProvider10 }

func (p benchChainProvider11) NewProvider() Provider { return p }

type benchChainProvider12 struct{ Prev benchChainProvider11 }

func (p benchChainProvider12) NewProvider() Provider { return p }

type benchChainProvider13 struct{ Prev benchChainProvider12 }

func (p benchChainProvider13) NewProvider() Provider { return p }

type benchChainProvider14 struct{ Prev benchChainProvider13 }

func (p benchChainProvider14) NewProvider() Provider { return p }

type benchChainProvider15 struct{ Prev benchChainProvider14 }

func (p benchChainProvider15) NewProvider() Provider { return p }

type benchChainProvider16 struct{ Prev benchChainProvider15 }

func (p benchChainProvider16) NewProvider() Provider { return p }

type benchChainProvider17 struct{ Prev benchChainProvider16 }

func (p benchChainProvider17) NewProvider() Provider { return p }

type benchChainProvider18 struct{ Prev benchChainProvider17 }

func (p benchChainProvider18) NewProvider() Provider { return p }

type benchChainProvider19 struct{ Prev benchChainProvider18 }

func (p benchChainProvider19) NewProvider() Provider { return p }

type benchChainProvider20 struct{ Prev benchChainProvider19 }

func (p benchChainProvider20) NewProvider() Provider { return p }

type benchChainProvider21 struct{ Prev benchChainProvider20 }

func (p benchChainProvider21) NewProvider() Provider { return p }

type benchChainProvider22 struct{ Prev benchChainProvider21 }

func (p benchChainProvider22) NewProvider() Provider { return p }

type benchChainProvider23 struct{ Prev benchChainProvider22 }

func (p benchChainProvider23) NewProvider() Provider { return p }

type benchChainProvider24 struct{ Prev benchChainProvider23 }

func (p benchChainProvider24) NewProvider() Provider { return p }

type benchChainProvider25 struct{ Prev benchChainProvider24 }

func (p benchChainProvider25) NewProvider() Provider { return p }

type benchChainProvider26 struct{ Prev benchChainProvider25 }

func (p benchChainProvider26) NewProvider() Provider { return p }

type benchChainProvider27 struct{ Prev benchChainProvider26 }

func (p benchChainProvider27) NewProvider() Provider { return p }

type benchChainProvider28 struct{ Prev benchChainProvider27 }

func (p benchChainProvider28) NewProvider() Provider { return p }

type benchChainProvider29 struct{ Prev benchChainProvider28 }

func (p benchChainProvider29) NewProvider() Provider { return p }

type benchChainProvider30 struct{ Prev benchChainProvider29 }

func (p benchChainProvider30) NewProvider() Provider { return p }

type benchChainProvider31 struct{ Prev benchChainProvider30 }

func (p benchChainProvider31) NewProvider() Provider { return p }

type benchChainProvider32 struct{ Prev benchChainProvider31 }

func (p benchChainProvider32) NewProvider() Provider { return p }

type benchChainProvider33 struct{ Prev benchChainProvider32 }

func (p benchChainProvider33) NewProvider() Provider { return p }

type benchChainProvider34 struct{ Prev benchChainProvider33 }

func (p benchChainProvider34) NewProvider() Provider { return p }

type benchChainProvider35 struct{ Prev benchChainProvider34 }

func (p benchChainProvider35) NewProvider() Provider { return p }

type benchChainProvider36 struct{ Prev benchChainProvider35 }

func (p benchChainProvider36) NewProvider() Provider { return p }

type benchChainProvider37 struct{ Prev benchChainProvider36 }

func (p benchChainProvider37) NewProvider() Provider { return p }

type benchChainProvider38 struct{ Prev benchChainProvider37 }

func (p benchChainProvider38) NewProvider() Provider { return p }

type benchChainProvider39 struct{ Prev benchChainProvider38 }

func (p benchChainProvider39) NewProvider() Provider { return p }

type benchChainProvider40 struct{ Prev benchChainProvider39 }

func (p benchChainProvider40) NewProvider() Provider { return p }

type benchChainProvider41 struct{ Prev benchChainProvider40 }

func (p benchChainProvider41) NewProvider() Provider { return p }

type benchChainProvider42 struct{ Prev benchChainProvider41 }

func (p benchChainProvider42) NewProvider() Provider { return p }

type benchChainProvider43 struct{ Prev benchChainProvider42 }

func (p benchChainProvider43) NewProvider() Provider { return p }

type benchChainProvider44 struct{ Prev benchChainProvider43 }

func (p benchChainProvider44) NewProvider() Provider { return p }

type benchChainProvider45 struct{ Prev benchChainProvider44 }

func (p benchChainProvider45) NewProvider() Provider { return p }

type benchChainProvider46 struct{ Prev benchChainProvider45 }

func (p benchChainProvider46) NewProvider() Provider { return p }

type benchChainProvider47 struct{ Prev benchChainProvider46 }

func (p benchChainProvider47) NewProvider() Provider { return p }

type benchChainProvider48 struct{ Prev benchChainProvider47 }

func (p benchChainProvider48) NewProvider() Provider { return p }

type benchChainProvider49 struct{ Prev benchChainProvider48 }

func (p benchChainProvider49) NewProvider() Provider { return p }

type benchChainProvider50 struct{ Prev benchChainProvider49 }

func (p benchChainProvider50) NewProvider() Provider { return p }

type benchChainProvider51 struct{ Prev benchChainProvider50 }

func (p benchChainProvider51) NewProvider() Provider { return p }

type benchChainProvider52 struct{ Prev benchChainProvider51 }

func (p benchChainProvider52) NewProvider() Provider { return p }

type benchChainProvider53 struct{ Prev benchChainProvider52 }

func (p benchChainProvider53) NewProvider() Provider { return p }

type benchChainProvider54 struct{ Prev benchChainProvider53 }

func (p benchChainProvider54) NewProvider() Provider { return p }

type benchChainProvider55 struct{ Prev benchChainProvider54 }

func (p benchChainProvider55) NewProvider() Provider { return p }

type benchChainProvider56 struct{ Prev benchChainProvider55 }

func (p benchChainProvider56) NewProvider() Provider { return p }

type benchChainProvider57 struct{ Prev benchChainProvider56 }

func (p benchChainProvider57) NewProvider() Provider { return p }

type benchChainProvider58 struct{ Prev benchChainProvider57 }

func (p benchChainProvider58) NewProvider() Provider { return p }

type benchChainProvider59 struct{ Prev benchChainProvider58 }

func (p benchChainProvider59) NewProvider() Provider { return p }

type benchChainProvider60 struct{ Prev benchChainProvider59 }

func (p benchChainProvider60) NewProvider() Provider { return p }

type benchChainProvider61 struct{ Prev benchChainProvider60 }

func (p benchChainProvider61) NewProvider() Provider { return p }

type benchChainProvider62 struct{ Prev benchChainProvider61 }

func (p benchChainProvider62) NewProvider() Provider { return p }

type benchChainProvider63 struct{ Prev benchChainProvider62 }

func (p benchChainProvider63) NewProvider() Provider { return p }

type benchChainProvider64 struct{ Prev benchChainProvider63 }

func (p benchChainProvider64) NewProvider() Provider { return p }

type benchChainProvider65 struct{ Prev benchChainProvider64 }

func (p benchChainProvider65) NewProvider() Provider { return p }

type benchChainProvider66 struct{ Prev benchChainProvider65 }

func (p benchChainProvider66) NewProvider() Provider { return p }

type benchChainProvider67 struct{ Prev benchChainProvider66 }

func (p benchChainProvider67) NewProvider() Provider { return p }

type benchChainProvider68 struct{ Prev benchChainProvider67 }

func (p benchChainProvider68) NewProvider() Provider { return p }

type benchChainProvider69 struct{ Prev benchChainProvider68 }

func (p benchChainProvider69) NewProvider() Provider { return p }

type benchChainProvider70 struct{ Prev benchChainProvider69 }

func (p benchChainProvider70) NewProvider() Provider { return p }

type benchChainProvider71 struct{ Prev benchChainProvider70 }

func (p benchChainProvider71) NewProvider() Provider { return p }

type benchChainProvider72 struct{ Prev benchChainProvider71 }

func (p benchChainProvider72) NewProvider() Provider { return p }

type benchChainProvider73 struct{ Prev benchChainProvider72 }

func (p benchChainProvider73) NewProvider() Provider { return p }

type benchChainProvider74 struct{ Prev benchChainProvider73 }

func (p benchChainProvider74) NewProvider() Provider { return p }

type benchChainProvider75 struct{ Prev benchChainProvider74 }

func (p benchChainProvider75) NewProvider() Provider { return p }

type benchChainProvider76 struct{ Prev benchChainProvider75 }

func (p benchChainProvider76) NewProvider() Provider { return p }

type benchChainProvider77 struct{ Prev benchChainProvider76 }

func (p benchChainProvider77) NewProvider() Provider { return p }

type benchChainProvider78 struct{ Prev benchChainProvider77 }

func (p benchChainProvider78) NewProvider() Provider { return p }

type benchChainProvider79 struct{ Prev benchChainProvider78 }

func (p benchChainProvider79) NewProvider() Provider { return p }

type benchChainProvider80 struct{ Prev benchChainProvider79 }

func (p benchChainProvider80) NewProvider() Provider { return p }

type benchChainProvider81 struct{ Prev benchChainProvider80 }

func (p benchChainProvider81) NewProvider() Provider { return p }

type benchChainProvider82 struct{ Prev benchChainProvider81 }

func (p benchChainProvider82) NewProvider() Provider { return p }

type benchChainProvider83 struct{ Prev benchChainProvider82 }

func (p benchChainProvider83) NewProvider() Provider { return p }

type benchChainProvider84 struct{ Prev benchChainProvider83 }

func (p benchChainProvider84) NewProvider() Provider { return p }

type benchChainProvider85 struct{ Prev benchChainProvider84 }

func (p benchChainProvider85) NewProvider() Provider { return p }

type benchChainProvider86 struct{ Prev benchChainProvider85 }

func (p benchChainProvider86) NewProvider() Provider { return p }

type benchChainProvider87 struct{ Prev benchChainProvider86 }

func (p benchChainProvider87) NewProvider() Provider { return p }

type benchChainProvider88 struct{ Prev benchChainProvider87 }

func (p benchChainProvider88) NewProvider() Provider { return p }

type benchChainProvider89 struct{ Prev benchChainProvider88 }

func (p benchChainProvider89) NewProvider() Provider { return p }

type benchChainProvider90 struct{ Prev benchChainProvider89 }

func (p benchChainProvider90) NewProvider() Provider { return p }

type benchChainProvider91 struct{ Prev benchChainProvider90 }

func (p benchChainProvider91) NewProvider() Provider { return p }

type benchChainProvider92 struct{ Prev benchChainProvider91 }

func (p benchChainProvider92) NewProvider() Provider { return p }

type benchChainProvider93 struct{ Prev benchChainProvider92 }

func (p benchChainProvider93) NewProvider() Provider { return p }

type benchChainProvider94 struct{ Prev benchChainProvider93 }

func (p benchChainProvider94) NewProvider() Provider { return p }

type benchChainProvider95 struct{ Prev benchChainProvider94 }

func (p benchChainProvider95) NewProvider() Provider { return p }

type benchChainProvider96 struct{ Prev benchChainProvider95 }

func (p benchChainProvider96) NewProvider() Provider { return p }

type benchChainProvider97 struct{ Prev benchChainProvider96 }

func (p benchChainProvider97) NewProvider() Provider { return p }

type benchChainProvider98 struct{ Prev benchChainProvider97 }

func (p benchChainProvider98) NewProvider() Provider { return p }

type benchChainProvider99 struct{ Prev benchChainProvider98 }

func (p benchChainProvider99) NewProvider() Provider { return p }

func benchChainProviders() []Provider {
	providers := make([]Provider, 100)
	providers[0] = benchChainProvider0{}
	providers[1] = benchChainProvider1{}
	providers[2] = benchChainProvider2{}
	providers[3] = benchChainProvider3{}
	providers[4] = benchChainProvider4{}
	providers[5] = benchChainProvider5{}
	providers[6] = benchChainProvider6{}
	providers[7] = benchChainProvider7{}
	providers[8] = benchChainProvider8{}
	providers[9] = benchChainProvider9{}
	providers[10] = benchChainProvider10{}
	providers[11] = benchChainProvider11{}
	providers[12] = benchChainProvider12{}
	providers[13] = benchChainProvider13{}
	providers[14] = benchChainProvider14{}
	providers[15] = benchChainProvider15{}
	providers[16] = benchChainProvider16{}
	providers[17] = benchChainProvider17{}
	providers[18] = benchChainProvider18{}
	providers[19] = benchChainProvider19{}
	providers[20] = benchChainProvider20{}
	providers[21] = benchChainProvider21{}
	providers[22] = benchChainProvider22{}
	providers[23] = benchChainProvider23{}
	providers[24] = benchChainProvider24{}
	providers[25] = benchChainProvider25{}
	providers[26] = benchChainProvider26{}
	providers[27] = benchChainProvider27{}
	providers[28] = benchChainProvider28{}
	providers[29] = benchChainProvider29{}
	providers[30] = benchChainProvider30{}
	providers[31] = benchChainProvider31{}
	providers[32] = benchChainProvider32{}
	providers[33] = benchChainProvider33{}
	providers[34] = benchChainProvider34{}
	providers[35] = benchChainProvider35{}
	providers[36] = benchChainProvider36{}
	providers[37] = benchChainProvider37{}
	providers[38] = benchChainProvider38{}
	providers[39] = benchChainProvider39{}
	providers[40] = benchChainProvider40{}
	providers[41] = benchChainProvider41{}
	providers[42] = benchChainProvider42{}
	providers[43] = benchChainProvider43{}
	providers[44] = benchChainProvider44{}
	providers[45] = benchChainProvider45{}
	providers[46] = benchChainProvider46{}
	providers[47] = benchChainProvider47{}
	providers[48] = benchChainProvider48{}
	providers[49] = benchChainProvider49{}
	providers[50] = benchChainProvider50{}
	providers[51] = benchChainProvider51{}
	providers[52] = benchChainProvider52{}
	providers[53] = benchChainProvider53{}
	providers[54] = benchChainProvider54{}
	providers[55] = benchChainProvider55{}
	providers[56] = benchChainProvider56{}
	providers[57] = benchChainProvider57{}
	providers[58] = benchChainProvider58{}
	providers[59] = benchChainProvider59{}
	providers[60] = benchChainProvider60{}
	providers[61] = benchChainProvider61{}
	providers[62] = benchChainProvider62{}
	providers[63] = benchChainProvider63{}
	providers[64] = benchChainProvider64{}
	providers[65] = benchChainProvider65{}
	providers[66] = benchChainProvider66{}
	providers[67] = benchChainProvider67{}
	providers[68] = benchChainProvider68{}
	providers[69] = benchChainProvider69{}
	providers[70] = benchChainProvider70{}
	providers[71] = benchChainProvider71{}
	providers[72] = benchChainProvider72{}
	providers[73] = benchChainProvider73{}
	providers[74] = benchChainProvider74{}
	providers[75] = benchChainProvider75{}
	providers[76] = benchChainProvider76{}
	providers[77] = benchChainProvider77{}
	providers[78] = benchChainProvider78{}
	providers[79] = benchChainProvider79{}
	providers[80] = benchChainProvider80{}
	providers[81] = benchChainProvider81{}
	providers[82] = benchChainProvider82{}
	providers[83] = benchChainProvider83{}
	providers[84] = benchChainProvider84{}
	providers[85] = benchChainProvider85{}
	providers[86] = benchChainProvider86{}
	providers[87] = benchChainProvider87{}
	providers[88] = benchChainProvider88{}
	providers[89] = benchChainProvider89{}
	providers[90] = benchChainProvider90{}
	providers[91] = benchChainProvider91{}
	providers[92] = benchChainProvider92{}
	providers[93] = benchChainProvider93{}
	providers[94] = benchChainProvider94{}
	providers[95] = benchChainProvider95{}
	providers[96] = benchChainProvider96{}
	providers[97] = benchChainProvider97{}
	providers[98] = benchChainProvider98{}
	providers[99] = benchChainProvider99{}
	return providers
}

func BenchmarkInjectProviders_ChainedDependencies(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		resetModuleGlobals()
		m := ModuleBuilder().Providers(benchChainProviders()...).Build()
		b.StartTimer()

		m.injectProviders()
	}
}

type benchStaticSubProvider struct{}

func (p benchStaticSubProvider) NewProvider() Provider { return p }

func makeBenchStaticSubmodules(n int) []any {
	imports := make([]any, n)
	for i := range imports {
		imports[i] = ModuleBuilder().Providers(benchStaticSubProvider{}).Build()
	}
	return imports
}

func BenchmarkInjectStaticModules(b *testing.B) {
	const n = 100
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		resetModuleGlobals()
		root := ModuleBuilder().Imports(makeBenchStaticSubmodules(n)...).Build()
		b.StartTimer()

		root.injectStaticModules()
	}
}

func benchCollectModulesDynamicFactory() *Module {
	return ModuleBuilder().Build()
}

func BenchmarkCollectModules(b *testing.B) {
	resetModuleGlobals()
	const n = 100
	imports := make([]any, n)
	for i := range imports {
		imports[i] = benchCollectModulesDynamicFactory
	}
	root := ModuleBuilder().Imports(imports...).Build()
	root.NewModule()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.collectModules()
	}
}
