package altdss

import (
	"errors"
	"fmt"
	"unsafe"
)

/*
#cgo LDFLAGS: -ldss_capi -Wl,-rpath,$ORIGIN
#include <stdlib.h>
#include "dss_capi_ctx.h"
*/
import "C"

const DSS_CAPI_VERSION = "0.14.0b1"

type ActionCodes int32

const (
	ActionCodes_none    ActionCodes = 0 // No action
	ActionCodes_Open    ActionCodes = 1 // Open a switch
	ActionCodes_Close   ActionCodes = 2 // Close a switch
	ActionCodes_Reset   ActionCodes = 3 // Reset to the shelf state (unlocked, closed for a switch)
	ActionCodes_Lock    ActionCodes = 4 // Lock a switch, preventing both manual and automatic operation
	ActionCodes_Unlock  ActionCodes = 5 // Unlock a switch, permitting both manual and automatic operation
	ActionCodes_TapUp   ActionCodes = 6 // Move a regulator tap up
	ActionCodes_TapDown ActionCodes = 7 // Move a regulator tap down
)

// Event codes used by the event callback system
//
// Legacy events are the events present the classic OpenDSS COM implementation,
// while the rest are extensions added here.
type AltDSSEvent int32

const (
	AltDSSEvent_Legacy_InitControls  AltDSSEvent = 0
	AltDSSEvent_Legacy_CheckControls AltDSSEvent = 1
	AltDSSEvent_Legacy_StepControls  AltDSSEvent = 2
	AltDSSEvent_Clear                AltDSSEvent = 3
	AltDSSEvent_ReprocessBuses       AltDSSEvent = 4
	AltDSSEvent_BuildSystemY         AltDSSEvent = 5
)

type AutoAddTypes int32

const (
	AutoAddTypes_AddGen AutoAddTypes = 1 // Add generators in AutoAdd mode
	AutoAddTypes_AddCap AutoAddTypes = 2 // Add capacitors in AutoAdd mode
)

type CapControlModes int32

const (
	CapControlModes_Current CapControlModes = 0 // Current control, ON and OFF settings on CT secondary
	CapControlModes_Voltage CapControlModes = 1 // Voltage control, ON and OFF settings on the PT secondary base
	CapControlModes_KVAR    CapControlModes = 2 // kVAR control, ON and OFF settings on PT / CT base
	CapControlModes_Time    CapControlModes = 3 // Time control, ON and OFF settings are seconds from midnight
	CapControlModes_PF      CapControlModes = 4 // ON and OFF settings are power factor, negative for leading
)

type CktModels int32

const (
	CktModels_Multiphase  CktModels = 0 // Circuit model is multiphase (default)
	CktModels_PositiveSeq CktModels = 1 // Circuit model is positive sequence model only
)

type ControlModes int32

const (
	ControlModes_Static    ControlModes = 0  // Control Mode option - Static
	ControlModes_Event     ControlModes = 1  // Control Mode Option - Event driven solution mode
	ControlModes_Time      ControlModes = 2  // Control mode option - Time driven mode
	ControlModes_Multirate ControlModes = 3  // Control mode option - Multirate mode
	ControlModes_Off       ControlModes = -1 // Control Mode OFF
)

// Transformer Core Type
type CoreType int32

const (
	CoreType_shell        CoreType = 0
	CoreType_one_phase    CoreType = 1
	CoreType_three_leg    CoreType = 3
	CoreType_four_leg     CoreType = 4
	CoreType_five_leg     CoreType = 5
	CoreType_core_1_phase CoreType = 9
)

const (
	// If enabled, don't check for NaNs in the inner solution loop.
	// This can lead to various errors.
	// This flag is useful for legacy applications that don't handle OpenDSS API errors properly.
	// Through the development of DSS-Extensions, we noticed this is actually a quite common issue.
	DSSCompatFlags_NoSolverFloatChecks = 1

	// If enabled, toggle worse precision for certain aspects of the engine. For
	// example, the sequence-to-phase (`As2p`) and sequence-to-phase (`Ap2s`)
	// transform matrices. On DSS C-API, we fill the matrix explicitly using
	// higher precision, while numerical inversion of an initially worse precision
	// matrix is used in the official OpenDSS. We will introduce better precision
	// for other aspects of the engine in the future, so this flag can be used to
	// toggle the old/bad values where feasible.
	DSSCompatFlags_BadPrecision = 2

	// Toggle some InvControl behavior introduced in OpenDSS 9.6.1.1. It could be a regression
	// but needs further investigation, so we added this flag in the time being.
	DSSCompatFlags_InvControl9611 = 4

	// When using "save circuit", the official OpenDSS always includes the "CalcVoltageBases" command in the
	// saved script. We found that it is not always a good idea, so we removed the command (leaving it commented).
	// Use this flag to enable the command in the saved script.
	DSSCompatFlags_SaveCalcVoltageBases = 8

	// In the official OpenDSS implementation, the Lines API use the active circuit element instead of the
	// active line. This can lead to unexpected behavior if the user is not aware of this detail.
	// For example, if the user accidentally enables any other circuit element, the next time they use
	// the Lines API, the line object that was previously enabled is overwritten with another unrelated
	// object.
	// This flag enables this behavior above if compatibility at this level is required. On DSS-Extensions,
	// we changed the behavior to follow what most of the other APIs do: use the active object in the internal
	// list. This change was done for DSS C-API v0.13.5, as well as the introduction of this flag.
	DSSCompatFlags_ActiveLine = 16

	// On DSS-Extensions/AltDSS, when setting a property invalidates a previous input value, the engine
	// will try to mark the invalidated data as unset. This allows for better exports and tracking of
	// the current state of DSS objects.
	// Set this flag to disable this behavior, following the original OpenDSS implementation for potential
	// compatibility with older software that may require the original behavior; note that may lead to
	// errorneous interpretation of the data in the DSS properties. This was introduced in DSS C-API v0.14.0
	// and will be further developed for future versions.
	DSSCompatFlags_NoPropertyTracking = 32

	// Some specific functions on the official OpenDSS APIs skip important side-effects.
	// By default, on DSS-Extensions/AltDSS, those side-effects are enabled. Use this flag
	// to try to follow the behavior of the official APIs. Beware that some side-effects are
	// important and skipping them may result in incorrect results.
	// This flag only affects some of the classic API functions, especially Loads and Generators.
	DSSCompatFlags_SkipSideEffects = 64
)

const (
	// Return all properties, regardless of order or if the property was filled by the user
	DSSJSONFlags_Full = 1

	// Skip redundant properties
	DSSJSONFlags_SkipRedundant = 2

	// Return enums as integers instead of strings
	DSSJSONFlags_EnumAsInt = 4

	// Use full names for the elements, including the class name
	DSSJSONFlags_FullNames = 8

	// Try to "pretty" format the JSON output
	DSSJSONFlags_Pretty = 16

	// Exclude disabled elements (only valid when exporting a collection)
	DSSJSONFlags_ExcludeDisabled = 32

	// Do not add the "DSSClass" property to the output
	DSSJSONFlags_SkipDSSClass = 64

	// Use lowercase representation for the property names (and other keys) instead of the internal variants.
	DSSJSONFlags_LowercaseKeys = 128

	// Include default unchanged objects in the exports.
	// Any default object that has been edited is always exported. Affects whole circuit and batch exports.
	DSSJSONFlags_IncludeDefaultObjs = 256
)

// This enum is used in the PropertyNameStyle property to control the naming convention.
// Currently, this only affects capitalization, i.e., if you software already uses case
// insensitive string comparisons for the property names, this is not useful. Otherwise,
// you can use `Legacy` to use the older names.
type DSSPropertyNameStyle int32

const (
	// By default, the modern names are used. The names were reviewed to try to reach a convention across all components.
	DSSPropertyNameStyle_Modern DSSPropertyNameStyle = 0

	// Use all lowercase strings.
	DSSPropertyNameStyle_Lowercase DSSPropertyNameStyle = 1

	// Use the previous capitalization of the property names.
	DSSPropertyNameStyle_Legacy DSSPropertyNameStyle = 2
)

type GeneratorStatus int32

const (
	GeneratorStatus_Variable GeneratorStatus = 0
	GeneratorStatus_Fixed    GeneratorStatus = 1
)

type LineUnits int32

const (
	LineUnits_none  LineUnits = 0 // No line length unit
	LineUnits_Miles LineUnits = 1 // Line length units in miles
	LineUnits_kFt   LineUnits = 2 // Line length units are in thousand feet
	LineUnits_km    LineUnits = 3 // Line length units are km
	LineUnits_meter LineUnits = 4 // Line length units are meters
	LineUnits_ft    LineUnits = 5 // Line units in feet
	LineUnits_inch  LineUnits = 6 // Line length units are inches
	LineUnits_cm    LineUnits = 7 // Line units are cm
	LineUnits_mm    LineUnits = 8 // Line length units are mm
)

type LoadModels int32

const (
	LoadModels_ConstPQ      LoadModels = 1
	LoadModels_ConstZ       LoadModels = 2
	LoadModels_Motor        LoadModels = 3
	LoadModels_CVR          LoadModels = 4
	LoadModels_ConstI       LoadModels = 5
	LoadModels_ConstPFixedQ LoadModels = 6
	LoadModels_ConstPFixedX LoadModels = 7
	LoadModels_ZIPV         LoadModels = 8
)

type LoadStatus int32

const (
	LoadStatus_Variable LoadStatus = 0
	LoadStatus_Fixed    LoadStatus = 1
	LoadStatus_Exempt   LoadStatus = 2
)

type MonitorModes int32

const (
	MonitorModes_VI        MonitorModes = 0  // Monitor records Voltage and Current at the terminal (Default)
	MonitorModes_Power     MonitorModes = 1  // Monitor records kW, kvar or kVA, angle values, etc. at the terminal to which it is connected.
	MonitorModes_Taps      MonitorModes = 2  // For monitoring Regulator and Transformer taps
	MonitorModes_States    MonitorModes = 3  // For monitoring State Variables (for PC Elements only)
	MonitorModes_Sequence  MonitorModes = 16 // Reports the monitored quantities as sequence quantities
	MonitorModes_Magnitude MonitorModes = 32 // Reports the monitored quantities in Magnitude Only
	MonitorModes_PosOnly   MonitorModes = 64 // Reports the Positive Seq only or avg of all phases
)

// Overcurrent Protection Device Type
type OCPDevType int32

const (
	OCPDevType_none     OCPDevType = 0
	OCPDevType_Fuse     OCPDevType = 1
	OCPDevType_Recloser OCPDevType = 2
	OCPDevType_Relay    OCPDevType = 3
)

// Deprecated. Please use instead:
// - AutoAddTypes
// - CktModels
// - ControlModes
// - SolutionLoadModels
// - SolutionAlgorithms
// - RandomModes
type Options int32

const (
	Options_PowerFlow   Options = 1
	Options_Admittance  Options = 2
	Options_NormalSolve Options = 0
	Options_LogNormal   Options = 3
	Options_ControlOFF  Options = -1
)

type RandomModes int32

const (
	RandomModes_Gaussian  RandomModes = 1
	RandomModes_Uniform   RandomModes = 2
	RandomModes_LogNormal RandomModes = 3
)

type SolutionAlgorithms int32

const (
	SolutionAlgorithms_NormalSolve SolutionAlgorithms = 0 // Solution algorithm option - Normal solution
	SolutionAlgorithms_NewtonSolve SolutionAlgorithms = 1 // Solution algorithm option - Newton solution
)

type SolutionLoadModels int32

const (
	SolutionLoadModels_PowerFlow  SolutionLoadModels = 1 // Power Flow load model option
	SolutionLoadModels_Admittance SolutionLoadModels = 2 // Admittance load model option
)

type SolveModes int32

const (
	SolveModes_SnapShot   SolveModes = 0  // Solve a single snapshot power flow
	SolveModes_Daily      SolveModes = 1  // Solve following Daily load shapes
	SolveModes_Yearly     SolveModes = 2  // Solve following Yearly load shapes
	SolveModes_Monte1     SolveModes = 3  // Monte Carlo Mode 1
	SolveModes_LD1        SolveModes = 4  // Load-duration Mode 1
	SolveModes_PeakDay    SolveModes = 5  // Solves for Peak Day using Daily load curve
	SolveModes_DutyCycle  SolveModes = 6  // Solve following Duty Cycle load shapes
	SolveModes_Direct     SolveModes = 7  // Solve direct (forced admittance model)
	SolveModes_MonteFault SolveModes = 8  // Monte carlo Fault Study
	SolveModes_FaultStudy SolveModes = 9  // Fault study at all buses
	SolveModes_Monte2     SolveModes = 10 // Monte Carlo Mode 2
	SolveModes_Monte3     SolveModes = 11 // Monte Carlo Mode 3
	SolveModes_LD2        SolveModes = 12 // Load-Duration Mode 2
	SolveModes_AutoAdd    SolveModes = 13 // Auto add generators or capacitors
	SolveModes_Dynamic    SolveModes = 14 // Solve for dynamics
	SolveModes_Harmonic   SolveModes = 15 // Harmonic solution mode
	SolveModes_Time       SolveModes = 16
	SolveModes_HarmonicT  SolveModes = 17
)

type SparseSolverOptions int32

const (
	// Default behavior, following the official OpenDSS implementation.
	SparseSolverOptions_ReuseNothing SparseSolverOptions = 0

	// Reuse only the prepared CSC matrix. This should be numerically exact, but
	// may have some cost saving if the number of entries changed in the system Y
	// matrix are a small fraction of the total entries.
	SparseSolverOptions_ReuseCompressedMatrix SparseSolverOptions = 1

	// Reuse the symbolic factorization, implies ReuseCompressedMatrix
	SparseSolverOptions_ReuseSymbolicFactorization SparseSolverOptions = 2

	// Reuse the numeric factorization, implies ReuseSymbolicFactorization
	SparseSolverOptions_ReuseNumericFactorization SparseSolverOptions = 3

	// Bit flag, see CktElement.pas for details. Some components do not clear the
	// dirty flag after their YPrim is updated, so YPrim is updated every time the
	// system Y is changed, even if there are no changes to the component. This
	// flag forces clearing the dirty flag, keeping the YPrim matrix constant when
	// the component has not changed.
	SparseSolverOptions_AlwaysResetYPrimInvalid SparseSolverOptions = 268435456
)

type YMatrixModes int32

const (
	YMatrixModes_SeriesOnly  YMatrixModes = 1
	YMatrixModes_WholeMatrix YMatrixModes = 2
)

type DSSContextPtrs struct {
	// Pointer to the context
	ctxPtr unsafe.Pointer

	// Pointer to the error number
	errorNumberPtr *int32

	// Pointers for the GR buffers
	CountPtr_PDouble  *[4]int32
	CountPtr_PPChar   *[4]int32
	CountPtr_PInteger *[4]int32
	CountPtr_PByte    *[4]int32

	DataPtr_PDouble  **float64
	DataPtr_PInteger **int32
	DataPtr_PByte    **uint8
	DataPtr_PPChar   ***C.char
}

type ICommonData struct {
	// Shared across all interfaces, owned by IDSS

	ctxPtr unsafe.Pointer
	ctx    *DSSContextPtrs
}

// type FuncGetStrings func(unsafe.Pointer, ***C.char, *C.int32_t)

func (ctx *DSSContextPtrs) PrepareStringArray(value []string) **C.char {
	data := (**C.char)(C.malloc((C.size_t)(len(value))))
	cdata := unsafe.Slice(data, len(value))
	for i := 0; i < len(value); i++ {
		cdata[i] = C.CString(value[i])
	}
	return data
}

func (ctx *DSSContextPtrs) FreeStringArray(data **C.char, count int) {
	cdata := unsafe.Slice(data, count)
	for i := 0; i < count; i++ {
		C.free(unsafe.Pointer(cdata[i]))
	}
	C.free(unsafe.Pointer(data))
}

func (ctx *DSSContextPtrs) GetStringArray(data **C.char, cnt [4]int32) ([]string, error) {
	err := ctx.DSSError()
	res_cnt := cnt[0]
	cdata := unsafe.Slice(data, res_cnt)
	result := make([]string, res_cnt)
	for i := int32(0); i < res_cnt; i++ {
		result[i] = C.GoString(cdata[i])
	}
	C.DSS_Dispose_PPAnsiChar(&data, (C.int32_t)(cnt[0]))
	return result, err
}

func (ctx *DSSContextPtrs) GetFloat64ArrayGR() ([]float64, error) {
	err := ctx.DSSError()
	res_cnt := (*ctx.CountPtr_PDouble)[0]
	cdata := unsafe.Slice(*ctx.DataPtr_PDouble, res_cnt)
	result := make([]float64, res_cnt)
	copy(result, cdata)
	return result, err
}

func (ctx *DSSContextPtrs) GetComplexArrayGR() ([]complex128, error) {
	err := ctx.DSSError()
	res_cnt := (*ctx.CountPtr_PDouble)[0]
	if res_cnt == 1 {
		res_cnt = 0
	}
	res_cnt /= 2
	cdata := unsafe.Slice((*complex128)(unsafe.Pointer(*ctx.DataPtr_PDouble)), res_cnt)
	result := make([]complex128, res_cnt)
	copy(result, cdata)
	return result, err
}

func (ctx *DSSContextPtrs) GetComplexSimpleGR() (complex128, error) {
	err := ctx.DSSError()
	res_cnt := (*ctx.CountPtr_PDouble)[0]
	if (err == nil) && (res_cnt != 2) {
		err := errors.New("(DSSError) Got invalid data for a complex number.")
		return 0.0, err
	}
	cdata := (*complex128)(unsafe.Pointer(*ctx.DataPtr_PDouble))
	return *cdata, err
}

func (ctx *DSSContextPtrs) GetInt32ArrayGR() ([]int32, error) {
	err := ctx.DSSError()
	res_cnt := (*ctx.CountPtr_PInteger)[0]
	cdata := unsafe.Slice(*ctx.DataPtr_PInteger, res_cnt)
	result := make([]int32, res_cnt)
	copy(result, cdata)
	return result, err
}

func (ctx *DSSContextPtrs) GetUInt8ArrayGR() ([]uint8, error) {
	err := ctx.DSSError()
	res_cnt := (*ctx.CountPtr_PByte)[0]
	cdata := unsafe.Slice(*ctx.DataPtr_PByte, res_cnt)
	result := make([]uint8, res_cnt)
	copy(result, cdata)
	return result, err
}

// func (ctx *DSSContextPtrs) GetStringsFromFunc(funcRef FuncGetStrings) ([]string, error) {
// 	var cnt [4]int32
// 	var data **C.char
// 	funcRef(ctx.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
// 	return ctx.GetStrings(data, cnt)
// }

func (ctx *DSSContextPtrs) DSSError() error {
	if (*ctx.errorNumberPtr) != 0 {
		err_result := fmt.Errorf("(DSSError#%d) %s", *ctx.errorNumberPtr, C.GoString(C.ctx_Error_Get_Description(ctx.ctxPtr)))
		*ctx.errorNumberPtr = 0
		return err_result
	}
	return nil
}

func (common *ICommonData) InitCommon(ctx *DSSContextPtrs) {
	common.ctx = ctx
	common.ctxPtr = ctx.ctxPtr
}

func (ctx *DSSContextPtrs) Init(ctxPtr unsafe.Pointer) {
	ctx.ctxPtr = ctxPtr
	C.ctx_DSS_Start(ctxPtr, 0)

	ctx.errorNumberPtr = (*int32)(C.ctx_Error_Get_NumberPtr(ctxPtr))

	C.ctx_DSS_GetGRPointers(
		ctxPtr,
		&ctx.DataPtr_PPChar,
		(***C.double)(unsafe.Pointer(&ctx.DataPtr_PDouble)),
		(***C.int32_t)(unsafe.Pointer(&ctx.DataPtr_PInteger)),
		(***C.int8_t)(unsafe.Pointer(&ctx.DataPtr_PByte)),
		(**C.int32_t)(unsafe.Pointer(&ctx.CountPtr_PPChar)),
		(**C.int32_t)(unsafe.Pointer(&ctx.CountPtr_PDouble)),
		(**C.int32_t)(unsafe.Pointer(&ctx.CountPtr_PInteger)),
		(**C.int32_t)(unsafe.Pointer(&ctx.CountPtr_PByte)),
	)
}

func ToUint16(v bool) C.uint16_t {
	if v {
		return (C.uint16_t)(1)
	}
	return (C.uint16_t)(0)
}

type IBus struct {
	ICommonData
}

func (bus *IBus) Init(ctx *DSSContextPtrs) {
	bus.InitCommon(ctx)
}

// Returns an array with the names of all PCE connected to the active bus
func (bus *IBus) AllPCEatBus() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Bus_Get_AllPCEatBus(bus.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return bus.ctx.GetStringArray(data, cnt)
}

// Returns an array with the names of all PDE connected to the active bus
func (bus *IBus) AllPDEatBus() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Bus_Get_AllPDEatBus(bus.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return bus.ctx.GetStringArray(data, cnt)
}

func (bus *IBus) GetUniqueNodeNumber(StartNumber int32) (int32, error) {
	return (int32)(C.ctx_Bus_GetUniqueNodeNumber(bus.ctxPtr, (C.int32_t)(StartNumber))), bus.ctx.DSSError()
}

// Refreshes the Zsc matrix for the active bus.
func (bus *IBus) ZscRefresh() (bool, error) {
	return (C.ctx_Bus_ZscRefresh(bus.ctxPtr) != 0), bus.ctx.DSSError()
}

// Indicates whether a coordinate has been defined for this bus
func (bus *IBus) Coorddefined() (bool, error) {
	return (C.ctx_Bus_Get_Coorddefined(bus.ctxPtr) != 0), bus.ctx.DSSError()
}

// Complex Double array of Sequence Voltages (0, 1, 2) at this Bus.
func (bus *IBus) CplxSeqVoltages() ([]complex128, error) {
	C.ctx_Bus_Get_CplxSeqVoltages_GR(bus.ctxPtr)
	return bus.ctx.GetComplexArrayGR()
}

// Accumulated customer outage durations
func (bus *IBus) Cust_Duration() (float64, error) {
	return (float64)(C.ctx_Bus_Get_Cust_Duration(bus.ctxPtr)), bus.ctx.DSSError()
}

// Annual number of customer-interruptions from this bus
func (bus *IBus) Cust_Interrupts() (float64, error) {
	return (float64)(C.ctx_Bus_Get_Cust_Interrupts(bus.ctxPtr)), bus.ctx.DSSError()
}

// Distance from energymeter (if non-zero)
func (bus *IBus) Distance() (float64, error) {
	return (float64)(C.ctx_Bus_Get_Distance(bus.ctxPtr)), bus.ctx.DSSError()
}

// Average interruption duration, hr.
func (bus *IBus) Int_Duration() (float64, error) {
	return (float64)(C.ctx_Bus_Get_Int_Duration(bus.ctxPtr)), bus.ctx.DSSError()
}

// Short circuit currents at bus; Complex Array.
func (bus *IBus) Isc() ([]complex128, error) {
	C.ctx_Bus_Get_Isc_GR(bus.ctxPtr)
	return bus.ctx.GetComplexArrayGR()
}

// Accumulated failure rate downstream from this bus; faults per year
func (bus *IBus) Lambda() (float64, error) {
	return (float64)(C.ctx_Bus_Get_Lambda(bus.ctxPtr)), bus.ctx.DSSError()
}

// Total numbers of customers served downline from this bus
func (bus *IBus) N_Customers() (int32, error) {
	return (int32)(C.ctx_Bus_Get_N_Customers(bus.ctxPtr)), bus.ctx.DSSError()
}

// Number of interruptions this bus per year
func (bus *IBus) N_interrupts() (float64, error) {
	return (float64)(C.ctx_Bus_Get_N_interrupts(bus.ctxPtr)), bus.ctx.DSSError()
}

// Name of Bus
func (bus *IBus) Name() (string, error) {
	return C.GoString(C.ctx_Bus_Get_Name(bus.ctxPtr)), bus.ctx.DSSError()
}

// Integer Array of Node Numbers defined at the bus in same order as the voltages.
func (bus *IBus) Nodes() ([]int32, error) {
	C.ctx_Bus_Get_Nodes_GR(bus.ctxPtr)
	return bus.ctx.GetInt32ArrayGR()
}

// Number of Nodes this bus.
func (bus *IBus) NumNodes() (int32, error) {
	return (int32)(C.ctx_Bus_Get_NumNodes(bus.ctxPtr)), bus.ctx.DSSError()
}

// Integer ID of the feeder section in which this bus is located.
func (bus *IBus) SectionID() (int32, error) {
	return (int32)(C.ctx_Bus_Get_SectionID(bus.ctxPtr)), bus.ctx.DSSError()
}

// Double Array of sequence voltages at this bus. Magnitudes only.
func (bus *IBus) SeqVoltages() ([]float64, error) {
	C.ctx_Bus_Get_SeqVoltages_GR(bus.ctxPtr)
	return bus.ctx.GetFloat64ArrayGR()
}

// Total length of line downline from this bus, in miles. For recloser siting algorithm.
func (bus *IBus) TotalMiles() (float64, error) {
	return (float64)(C.ctx_Bus_Get_TotalMiles(bus.ctxPtr)), bus.ctx.DSSError()
}

// For 2- and 3-phase buses, returns array of complex numbers represetin L-L voltages in volts. Returns -1.0 for 1-phase bus. If more than 3 phases, returns only first 3.
func (bus *IBus) VLL() ([]complex128, error) {
	C.ctx_Bus_Get_VLL_GR(bus.ctxPtr)
	return bus.ctx.GetComplexArrayGR()
}

// Array of doubles containing voltages in Magnitude (VLN), angle (degrees)
func (bus *IBus) VMagAngle() ([]float64, error) {
	C.ctx_Bus_Get_VMagAngle_GR(bus.ctxPtr)
	return bus.ctx.GetFloat64ArrayGR()
}

// Open circuit voltage; Complex array.
func (bus *IBus) Voc() ([]complex128, error) {
	C.ctx_Bus_Get_Voc_GR(bus.ctxPtr)
	return bus.ctx.GetComplexArrayGR()
}

// Complex array of voltages at this bus.
func (bus *IBus) Voltages() ([]complex128, error) {
	C.ctx_Bus_Get_Voltages_GR(bus.ctxPtr)
	return bus.ctx.GetComplexArrayGR()
}

// Complex array of Ysc matrix at bus. Column by column.
func (bus *IBus) YscMatrix() ([]complex128, error) {
	C.ctx_Bus_Get_YscMatrix_GR(bus.ctxPtr)
	return bus.ctx.GetComplexArrayGR()
}

// Complex Zero-Sequence short circuit impedance at bus.
func (bus *IBus) Zsc0() (complex128, error) {
	C.ctx_Bus_Get_Zsc0_GR(bus.ctxPtr)
	return bus.ctx.GetComplexSimpleGR()
}

// Complex Positive-Sequence short circuit impedance at bus.
func (bus *IBus) Zsc1() (complex128, error) {
	C.ctx_Bus_Get_Zsc1_GR(bus.ctxPtr)
	return bus.ctx.GetComplexSimpleGR()
}

// Complex array of Zsc matrix at bus. Column by column.
func (bus *IBus) ZscMatrix() ([]complex128, error) {
	C.ctx_Bus_Get_ZscMatrix_GR(bus.ctxPtr)
	return bus.ctx.GetComplexArrayGR()
}

// Base voltage at bus in kV
func (bus *IBus) Get_kVBase() (float64, error) {
	return (float64)(C.ctx_Bus_Get_kVBase(bus.ctxPtr)), bus.ctx.DSSError()
}

// Returns Complex array of pu L-L voltages for 2- and 3-phase buses. Returns -1.0 for 1-phase bus. If more than 3 phases, returns only 3 phases.
func (bus *IBus) PUVLL() ([]complex128, error) {
	C.ctx_Bus_Get_puVLL_GR(bus.ctxPtr)
	return bus.ctx.GetComplexArrayGR()
}

// Array of doubles containing voltage magnitude, angle (degrees) pairs in per unit
func (bus *IBus) PUVMagAngle() ([]float64, error) {
	C.ctx_Bus_Get_puVmagAngle_GR(bus.ctxPtr)
	return bus.ctx.GetFloat64ArrayGR()
}

// Complex Array of pu voltages at the bus.
func (bus *IBus) PUVoltages() ([]complex128, error) {
	C.ctx_Bus_Get_puVoltages_GR(bus.ctxPtr)
	return bus.ctx.GetComplexArrayGR()
}

// Array of doubles (complex) containing the complete 012 Zsc matrix.
// Only available after Zsc is computed, either through the "ZscRefresh" command, or running a "FaultStudy" solution.
// Only available for buses with 3 nodes.
func (bus *IBus) ZSC012Matrix() ([]complex128, error) {
	C.ctx_Bus_Get_ZSC012Matrix_GR(bus.ctxPtr)
	return bus.ctx.GetComplexArrayGR()
}

// X Coordinate for bus (double)
func (bus *IBus) Get_x() (float64, error) {
	return (float64)(C.ctx_Bus_Get_x(bus.ctxPtr)), bus.ctx.DSSError()
}

func (bus *IBus) Set_x(value float64) error {
	C.ctx_Bus_Set_x(bus.ctxPtr, (C.double)(value))
	return bus.ctx.DSSError()
}

// Y coordinate for bus(double)
func (bus *IBus) Get_y() (float64, error) {
	return (float64)(C.ctx_Bus_Get_y(bus.ctxPtr)), bus.ctx.DSSError()
}

func (bus *IBus) Set_y(value float64) error {
	C.ctx_Bus_Set_y(bus.ctxPtr, (C.double)(value))
	return bus.ctx.DSSError()
}

// List of strings: Full Names of LOAD elements connected to the active bus.
func (bus *IBus) LoadList() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Bus_Get_LoadList(bus.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return bus.ctx.GetStringArray(data, cnt)
}

// List of strings: Full Names of LINE elements connected to the active bus.
func (bus *IBus) LineList() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Bus_Get_LineList(bus.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return bus.ctx.GetStringArray(data, cnt)
}

type ICNData struct {
	ICommonData
}

func (cndata *ICNData) Init(ctx *DSSContextPtrs) {
	cndata.InitCommon(ctx)
}

// Array of strings with all CNData names in the circuit.
func (cndata *ICNData) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_CNData_Get_AllNames(cndata.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return cndata.ctx.GetStringArray(data, cnt)
}

// Number of CNData objects in active circuit.
func (cndata *ICNData) Count() (int32, error) {
	return (int32)(C.ctx_CNData_Get_Count(cndata.ctxPtr)), cndata.ctx.DSSError()
}

// Sets the first CNData active. Returns 0 if no more.
func (cndata *ICNData) First() (int32, error) {
	return (int32)(C.ctx_CNData_Get_First(cndata.ctxPtr)), cndata.ctx.DSSError()
}

// Sets the active CNData by Name.
func (cndata *ICNData) Get_Name() (string, error) {
	result := C.GoString(C.ctx_CNData_Get_Name(cndata.ctxPtr))
	return result, cndata.ctx.DSSError()
}

// Gets the name of the active CNData.
func (cndata *ICNData) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_CNData_Set_Name(cndata.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return cndata.ctx.DSSError()
}

// Sets the next CNData active. Returns 0 if no more.
func (cndata *ICNData) Next() (int32, error) {
	return (int32)(C.ctx_CNData_Get_Next(cndata.ctxPtr)), cndata.ctx.DSSError()
}

// Get the index of the active CNData; index is 1-based: 1..count
func (cndata *ICNData) Get_idx() (int32, error) {
	return (int32)(C.ctx_CNData_Get_idx(cndata.ctxPtr)), cndata.ctx.DSSError()
}

// Set the active CNData by index; index is 1-based: 1..count
func (cndata *ICNData) Set_idx(value int32) error {
	C.ctx_CNData_Set_idx(cndata.ctxPtr, (C.int32_t)(value))
	return cndata.ctx.DSSError()
}

// Emergency ampere rating
func (cndata *ICNData) Get_EmergAmps() (float64, error) {
	return (float64)(C.ctx_CNData_Get_EmergAmps(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_EmergAmps(value float64) error {
	C.ctx_CNData_Set_EmergAmps(cndata.ctxPtr, (C.double)(value))
	return cndata.ctx.DSSError()
}

// Normal Ampere rating
func (cndata *ICNData) Get_NormAmps() (float64, error) {
	return (float64)(C.ctx_CNData_Get_NormAmps(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_NormAmps(value float64) error {
	C.ctx_CNData_Set_NormAmps(cndata.ctxPtr, (C.double)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_Rdc() (float64, error) {
	return (float64)(C.ctx_CNData_Get_Rdc(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_Rdc(value float64) error {
	C.ctx_CNData_Set_Rdc(cndata.ctxPtr, (C.double)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_Rac() (float64, error) {
	return (float64)(C.ctx_CNData_Get_Rac(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_Rac(value float64) error {
	C.ctx_CNData_Set_Rac(cndata.ctxPtr, (C.double)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_GMRac() (float64, error) {
	return (float64)(C.ctx_CNData_Get_GMRac(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_GMRac(value float64) error {
	C.ctx_CNData_Set_GMRac(cndata.ctxPtr, (C.double)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_GMRUnits() (LineUnits, error) {
	return (LineUnits)(C.ctx_CNData_Get_GMRUnits(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_GMRUnits(value LineUnits) error {
	C.ctx_CNData_Set_GMRUnits(cndata.ctxPtr, (C.int32_t)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_Radius() (float64, error) {
	return (float64)(C.ctx_CNData_Get_Radius(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_Radius(value float64) error {
	C.ctx_CNData_Set_Radius(cndata.ctxPtr, (C.double)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_RadiusUnits() (LineUnits, error) {
	return (LineUnits)(C.ctx_CNData_Get_RadiusUnits(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_RadiusUnits(value LineUnits) error {
	C.ctx_CNData_Set_RadiusUnits(cndata.ctxPtr, (C.int32_t)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_ResistanceUnits() (LineUnits, error) {
	return (LineUnits)(C.ctx_CNData_Get_ResistanceUnits(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_ResistanceUnits(value LineUnits) error {
	C.ctx_CNData_Set_ResistanceUnits(cndata.ctxPtr, (C.int32_t)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_Diameter() (float64, error) {
	return (float64)(C.ctx_CNData_Get_Diameter(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_Diameter(value float64) error {
	C.ctx_CNData_Set_Diameter(cndata.ctxPtr, (C.double)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_EpsR() (float64, error) {
	return (float64)(C.ctx_CNData_Get_EpsR(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_EpsR(value float64) error {
	C.ctx_CNData_Set_EpsR(cndata.ctxPtr, (C.double)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_InsLayer() (float64, error) {
	return (float64)(C.ctx_CNData_Get_InsLayer(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_InsLayer(value float64) error {
	C.ctx_CNData_Set_InsLayer(cndata.ctxPtr, (C.double)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_DiaIns() (float64, error) {
	return (float64)(C.ctx_CNData_Get_DiaIns(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_DiaIns(value float64) error {
	C.ctx_CNData_Set_DiaIns(cndata.ctxPtr, (C.double)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_DiaCable() (float64, error) {
	return (float64)(C.ctx_CNData_Get_DiaCable(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_DiaCable(value float64) error {
	C.ctx_CNData_Set_DiaCable(cndata.ctxPtr, (C.double)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_k() (int32, error) {
	return (int32)(C.ctx_CNData_Get_k(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_k(value int32) error {
	C.ctx_CNData_Set_k(cndata.ctxPtr, (C.int32_t)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_DiaStrand() (float64, error) {
	return (float64)(C.ctx_CNData_Get_DiaStrand(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_DiaStrand(value float64) error {
	C.ctx_CNData_Set_DiaStrand(cndata.ctxPtr, (C.double)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_GmrStrand() (float64, error) {
	return (float64)(C.ctx_CNData_Get_GmrStrand(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_GmrStrand(value float64) error {
	C.ctx_CNData_Set_GmrStrand(cndata.ctxPtr, (C.double)(value))
	return cndata.ctx.DSSError()
}

func (cndata *ICNData) Get_RStrand() (float64, error) {
	return (float64)(C.ctx_CNData_Get_RStrand(cndata.ctxPtr)), cndata.ctx.DSSError()
}

func (cndata *ICNData) Set_RStrand(value float64) error {
	C.ctx_CNData_Set_RStrand(cndata.ctxPtr, (C.double)(value))
	return cndata.ctx.DSSError()
}

type ICapacitors struct {
	ICommonData
}

func (capacitors *ICapacitors) Init(ctx *DSSContextPtrs) {
	capacitors.InitCommon(ctx)
}

// Array of strings with all Capacitor names in the circuit.
func (capacitors *ICapacitors) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Capacitors_Get_AllNames(capacitors.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return capacitors.ctx.GetStringArray(data, cnt)
}

// Number of Capacitor objects in active circuit.
func (capacitors *ICapacitors) Count() (int32, error) {
	return (int32)(C.ctx_Capacitors_Get_Count(capacitors.ctxPtr)), capacitors.ctx.DSSError()
}

// Sets the first Capacitor active. Returns 0 if no more.
func (capacitors *ICapacitors) First() (int32, error) {
	return (int32)(C.ctx_Capacitors_Get_First(capacitors.ctxPtr)), capacitors.ctx.DSSError()
}

// Sets the active Capacitor by Name.
func (capacitors *ICapacitors) Get_Name() (string, error) {
	result := C.GoString(C.ctx_Capacitors_Get_Name(capacitors.ctxPtr))
	return result, capacitors.ctx.DSSError()
}

// Gets the name of the active Capacitor.
func (capacitors *ICapacitors) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_Capacitors_Set_Name(capacitors.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return capacitors.ctx.DSSError()
}

// Sets the next Capacitor active. Returns 0 if no more.
func (capacitors *ICapacitors) Next() (int32, error) {
	return (int32)(C.ctx_Capacitors_Get_Next(capacitors.ctxPtr)), capacitors.ctx.DSSError()
}

// Get the index of the active Capacitor; index is 1-based: 1..count
func (capacitors *ICapacitors) Get_idx() (int32, error) {
	return (int32)(C.ctx_Capacitors_Get_idx(capacitors.ctxPtr)), capacitors.ctx.DSSError()
}

// Set the active Capacitor by index; index is 1-based: 1..count
func (capacitors *ICapacitors) Set_idx(value int32) error {
	C.ctx_Capacitors_Set_idx(capacitors.ctxPtr, (C.int32_t)(value))
	return capacitors.ctx.DSSError()
}

func (capacitors *ICapacitors) AddStep() (bool, error) {
	return (C.ctx_Capacitors_AddStep(capacitors.ctxPtr) != 0), capacitors.ctx.DSSError()
}

func (capacitors *ICapacitors) Close() error {
	C.ctx_Capacitors_Close(capacitors.ctxPtr)
	return capacitors.ctx.DSSError()
}

func (capacitors *ICapacitors) Open() error {
	C.ctx_Capacitors_Open(capacitors.ctxPtr)
	return capacitors.ctx.DSSError()
}

func (capacitors *ICapacitors) SubtractStep() (bool, error) {
	return (C.ctx_Capacitors_SubtractStep(capacitors.ctxPtr) != 0), capacitors.ctx.DSSError()
}

// Number of Steps available in cap bank to be switched ON.
func (capacitors *ICapacitors) AvailableSteps() (int32, error) {
	return (int32)(C.ctx_Capacitors_Get_AvailableSteps(capacitors.ctxPtr)), capacitors.ctx.DSSError()
}

// Delta connection or wye?
func (capacitors *ICapacitors) Get_IsDelta() (bool, error) {
	return (C.ctx_Capacitors_Get_IsDelta(capacitors.ctxPtr) != 0), capacitors.ctx.DSSError()
}

func (capacitors *ICapacitors) Set_IsDelta(value bool) error {
	C.ctx_Capacitors_Set_IsDelta(capacitors.ctxPtr, ToUint16(value))
	return capacitors.ctx.DSSError()
}

// Number of steps (default 1) for distributing and switching the total bank kVAR.
func (capacitors *ICapacitors) Get_NumSteps() (int32, error) {
	return (int32)(C.ctx_Capacitors_Get_NumSteps(capacitors.ctxPtr)), capacitors.ctx.DSSError()
}

func (capacitors *ICapacitors) Set_NumSteps(value int32) error {
	C.ctx_Capacitors_Set_NumSteps(capacitors.ctxPtr, (C.int32_t)(value))
	return capacitors.ctx.DSSError()
}

// A array of  integer [0..numsteps-1] indicating state of each step. If the read value is -1 an error has occurred.
func (capacitors *ICapacitors) Get_States() ([]int32, error) {
	C.ctx_Capacitors_Get_States_GR(capacitors.ctxPtr)
	return capacitors.ctx.GetInt32ArrayGR()
}

func (capacitors *ICapacitors) Set_States(value []int32) error {
	C.ctx_Capacitors_Set_States(capacitors.ctxPtr, (*C.int32_t)(&value[0]), (C.int32_t)(len(value)))
	return capacitors.ctx.DSSError()
}

// Bank kV rating. Use LL for 2 or 3 phases, or actual can rating for 1 phase.
func (capacitors *ICapacitors) Get_kV() (float64, error) {
	return (float64)(C.ctx_Capacitors_Get_kV(capacitors.ctxPtr)), capacitors.ctx.DSSError()
}

func (capacitors *ICapacitors) Set_kV(value float64) error {
	C.ctx_Capacitors_Set_kV(capacitors.ctxPtr, (C.double)(value))
	return capacitors.ctx.DSSError()
}

// Total bank KVAR, distributed equally among phases and steps.
func (capacitors *ICapacitors) Get_kvar() (float64, error) {
	return (float64)(C.ctx_Capacitors_Get_kvar(capacitors.ctxPtr)), capacitors.ctx.DSSError()
}

func (capacitors *ICapacitors) Set_kvar(value float64) error {
	C.ctx_Capacitors_Set_kvar(capacitors.ctxPtr, (C.double)(value))
	return capacitors.ctx.DSSError()
}

type ICktElement struct {
	ICommonData

	Properties IDSSProperty
}

func (cktelement *ICktElement) Init(ctx *DSSContextPtrs) {
	cktelement.InitCommon(ctx)
	cktelement.Properties.Init(ctx)
}

// Value as return and error code in Code parameter. For PCElement, get the value of a variable by name. If Code>0 then no variable by this name or not a PCelement.
func (cktelement *ICktElement) Get_Variable(varName string, Code *int32) (float64, error) {
	varName_c := C.CString(varName)
	result, err := (float64)(C.ctx_CktElement_Get_Variable(cktelement.ctxPtr, varName_c, (*C.int32_t)(Code))), cktelement.ctx.DSSError()
	C.free(unsafe.Pointer(varName_c))
	return result, err
}

// Value as return and error code in Code parameter. For PCElement, get the value of a variable by integer index. If Code>0 then no variable by this index or not a PCelement.
func (cktelement *ICktElement) Get_Variablei(Idx int32, Code *int32) (float64, error) {
	return (float64)(C.ctx_CktElement_Get_Variablei(cktelement.ctxPtr, (C.int32_t)(Idx), (*C.int32_t)(Code))), cktelement.ctx.DSSError()
}

// Value as return and error code in Code parameter. For PCElement, get the value of a variable by integer index. If Code>0 then no variable by this index or not a PCelement.
func (cktelement *ICktElement) Get_VariableByIndex(Idx int32, Code *int32) (float64, error) {
	return cktelement.Get_Variablei(Idx, Code)
}

// Value as return and error code in Code parameter. For PCElement, get the value of a variable by name. If Code>0 then no variable by this name or not a PCelement.
func (cktelement *ICktElement) Get_VariableByName(Name string, Code *int32) (float64, error) {
	return cktelement.Get_Variable(Name, Code)
}

// Set the Value of a variable by indx if a PCElement. If Code>0 then no variable by this index or not a PCelement.
func (cktelement *ICktElement) Set_VariableByIndex(Idx int32, Code *int32, Value float64) error {
	C.ctx_CktElement_Set_Variablei(cktelement.ctxPtr, (C.int32_t)(Idx), (*C.int32_t)(Code), (C.double)(Value))
	return cktelement.ctx.DSSError()
}

// Set the Value of a variable by name if a PCElement. If Code>0 then no variable by this name or not a PCelement.
func (cktelement *ICktElement) Set_VariableByName(varName string, Code *int32, Value float64) error {
	varName_c := C.CString(varName)
	C.ctx_CktElement_Set_Variable(cktelement.ctxPtr, varName_c, (*C.int32_t)(Code), (C.double)(Value))
	C.free(unsafe.Pointer(varName_c))
	return cktelement.ctx.DSSError()
}

func (cktelement *ICktElement) Close(Term int32, Phs int32) error {
	C.ctx_CktElement_Close(cktelement.ctxPtr, (C.int32_t)(Term), (C.int32_t)(Phs))
	return cktelement.ctx.DSSError()
}

// Full name of the i-th controller attached to this element. Ex: str = Controller(2).  See NumControls to determine valid index range
func (cktelement *ICktElement) Controller(idx int32) (string, error) {
	return C.GoString(C.ctx_CktElement_Get_Controller(cktelement.ctxPtr, (C.int32_t)(idx))), cktelement.ctx.DSSError()
}

func (cktelement *ICktElement) IsOpen(Term int32, Phs int32) (bool, error) {
	return (C.ctx_CktElement_IsOpen(cktelement.ctxPtr, (C.int32_t)(Term), (C.int32_t)(Phs)) != 0), cktelement.ctx.DSSError()
}

func (cktelement *ICktElement) Open(Term int32, Phs int32) error {
	C.ctx_CktElement_Open(cktelement.ctxPtr, (C.int32_t)(Term), (C.int32_t)(Phs))
	return cktelement.ctx.DSSError()
}

// Array containing all property names of the active device.
func (cktelement *ICktElement) AllPropertyNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_CktElement_Get_AllPropertyNames(cktelement.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return cktelement.ctx.GetStringArray(data, cnt)
}

// Array of strings listing all the published state variable names.
// Valid only for PCElements.
func (cktelement *ICktElement) AllVariableNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_CktElement_Get_AllVariableNames(cktelement.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return cktelement.ctx.GetStringArray(data, cnt)
}

// Array of doubles. Values of state variables of active element if PC element.
// Valid only for PCElements.
func (cktelement *ICktElement) AllVariableValues() ([]float64, error) {
	C.ctx_CktElement_Get_AllVariableValues_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetFloat64ArrayGR()
}

// Array of strings. Get  Bus definitions to which each terminal is connected.
func (cktelement *ICktElement) Get_BusNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_CktElement_Get_BusNames(cktelement.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return cktelement.ctx.GetStringArray(data, cnt)
}

func (cktelement *ICktElement) Set_BusNames(value []string) error {
	value_c := cktelement.ctx.PrepareStringArray(value)
	defer cktelement.ctx.FreeStringArray(value_c, len(value))
	C.ctx_CktElement_Set_BusNames(cktelement.ctxPtr, value_c, (C.int32_t)(len(value)))
	return cktelement.ctx.DSSError()
}

// Complex double array of Sequence Currents for all conductors of all terminals of active circuit element.
func (cktelement *ICktElement) CplxSeqCurrents() ([]complex128, error) {
	C.ctx_CktElement_Get_CplxSeqCurrents_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetComplexArrayGR()
}

// Complex double array of Sequence Voltage for all terminals of active circuit element.
func (cktelement *ICktElement) CplxSeqVoltages() ([]complex128, error) {
	C.ctx_CktElement_Get_CplxSeqVoltages_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetComplexArrayGR()
}

// Complex array of currents into each conductor of each terminal
func (cktelement *ICktElement) Currents() ([]complex128, error) {
	C.ctx_CktElement_Get_Currents_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetComplexArrayGR()
}

// Currents in magnitude, angle (degrees) format as a array of doubles.
func (cktelement *ICktElement) CurrentsMagAng() ([]float64, error) {
	C.ctx_CktElement_Get_CurrentsMagAng_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetFloat64ArrayGR()
}

// Display name of the object (not necessarily unique)
func (cktelement *ICktElement) Get_DisplayName() (string, error) {
	return C.GoString(C.ctx_CktElement_Get_DisplayName(cktelement.ctxPtr)), cktelement.ctx.DSSError()
}

func (cktelement *ICktElement) Set_DisplayName(value string) error {
	value_c := C.CString(value)
	C.ctx_CktElement_Set_DisplayName(cktelement.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return cktelement.ctx.DSSError()
}

// Emergency Ampere Rating for PD elements
func (cktelement *ICktElement) Get_EmergAmps() (float64, error) {
	return (float64)(C.ctx_CktElement_Get_EmergAmps(cktelement.ctxPtr)), cktelement.ctx.DSSError()
}

func (cktelement *ICktElement) Set_EmergAmps(value float64) error {
	C.ctx_CktElement_Set_EmergAmps(cktelement.ctxPtr, (C.double)(value))
	return cktelement.ctx.DSSError()
}

// Boolean indicating that element is currently in the circuit.
func (cktelement *ICktElement) Get_Enabled() (bool, error) {
	return (C.ctx_CktElement_Get_Enabled(cktelement.ctxPtr) != 0), cktelement.ctx.DSSError()
}

func (cktelement *ICktElement) Set_Enabled(value bool) error {
	C.ctx_CktElement_Set_Enabled(cktelement.ctxPtr, ToUint16(value))
	return cktelement.ctx.DSSError()
}

// Name of the Energy Meter this element is assigned to.
func (cktelement *ICktElement) EnergyMeter() (string, error) {
	return C.GoString(C.ctx_CktElement_Get_EnergyMeter(cktelement.ctxPtr)), cktelement.ctx.DSSError()
}

// globally unique identifier for this object
func (cktelement *ICktElement) GUID() (string, error) {
	return C.GoString(C.ctx_CktElement_Get_GUID(cktelement.ctxPtr)), cktelement.ctx.DSSError()
}

// Pointer to this object
func (cktelement *ICktElement) Handle() (int32, error) {
	return (int32)(C.ctx_CktElement_Get_Handle(cktelement.ctxPtr)), cktelement.ctx.DSSError()
}

// True if a recloser, relay, or fuse controlling this ckt element. OCP = Overcurrent Protection
func (cktelement *ICktElement) HasOCPDevice() (bool, error) {
	return (C.ctx_CktElement_Get_HasOCPDevice(cktelement.ctxPtr) != 0), cktelement.ctx.DSSError()
}

// This element has a SwtControl attached.
func (cktelement *ICktElement) HasSwitchControl() (bool, error) {
	return (C.ctx_CktElement_Get_HasSwitchControl(cktelement.ctxPtr) != 0), cktelement.ctx.DSSError()
}

// This element has a CapControl or RegControl attached.
func (cktelement *ICktElement) HasVoltControl() (bool, error) {
	return (C.ctx_CktElement_Get_HasVoltControl(cktelement.ctxPtr) != 0), cktelement.ctx.DSSError()
}

// Total losses in the element: two-element double array (complex), in VA (watts, vars)
func (cktelement *ICktElement) Losses() (complex128, error) {
	C.ctx_CktElement_Get_Losses_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetComplexSimpleGR()
}

// Full Name of Active Circuit Element
func (cktelement *ICktElement) Name() (string, error) {
	return C.GoString(C.ctx_CktElement_Get_Name(cktelement.ctxPtr)), cktelement.ctx.DSSError()
}

// Array of integer containing the node numbers (representing phases, for example) for each conductor of each terminal.
func (cktelement *ICktElement) NodeOrder() ([]int32, error) {
	C.ctx_CktElement_Get_NodeOrder_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetInt32ArrayGR()
}

// Normal ampere rating for PD Elements
func (cktelement *ICktElement) Get_NormalAmps() (float64, error) {
	return (float64)(C.ctx_CktElement_Get_NormalAmps(cktelement.ctxPtr)), cktelement.ctx.DSSError()
}

func (cktelement *ICktElement) Set_NormalAmps(value float64) error {
	C.ctx_CktElement_Set_NormalAmps(cktelement.ctxPtr, (C.double)(value))
	return cktelement.ctx.DSSError()
}

// Number of Conductors per Terminal
func (cktelement *ICktElement) NumConductors() (int32, error) {
	return (int32)(C.ctx_CktElement_Get_NumConductors(cktelement.ctxPtr)), cktelement.ctx.DSSError()
}

// Number of controls connected to this device.
// Use to determine valid range for index into Controller array.
func (cktelement *ICktElement) NumControls() (int32, error) {
	return (int32)(C.ctx_CktElement_Get_NumControls(cktelement.ctxPtr)), cktelement.ctx.DSSError()
}

// Number of Phases
func (cktelement *ICktElement) NumPhases() (int32, error) {
	return (int32)(C.ctx_CktElement_Get_NumPhases(cktelement.ctxPtr)), cktelement.ctx.DSSError()
}

// Number of Properties this Circuit Element.
func (cktelement *ICktElement) NumProperties() (int32, error) {
	return (int32)(C.ctx_CktElement_Get_NumProperties(cktelement.ctxPtr)), cktelement.ctx.DSSError()
}

// Number of Terminals this Circuit Element
func (cktelement *ICktElement) NumTerminals() (int32, error) {
	return (int32)(C.ctx_CktElement_Get_NumTerminals(cktelement.ctxPtr)), cktelement.ctx.DSSError()
}

// Index into Controller list of OCP Device controlling this CktElement
func (cktelement *ICktElement) OCPDevIndex() (int32, error) {
	return (int32)(C.ctx_CktElement_Get_OCPDevIndex(cktelement.ctxPtr)), cktelement.ctx.DSSError()
}

// 0=None; 1=Fuse; 2=Recloser; 3=Relay;  Type of OCP controller device
func (cktelement *ICktElement) OCPDevType() (OCPDevType, error) {
	return (OCPDevType)(C.ctx_CktElement_Get_OCPDevType(cktelement.ctxPtr)), cktelement.ctx.DSSError()
}

// Complex array of losses (kVA) by phase
func (cktelement *ICktElement) PhaseLosses() ([]complex128, error) {
	C.ctx_CktElement_Get_PhaseLosses_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetComplexArrayGR()
}

// Complex array of powers (kVA) into each conductor of each terminal
func (cktelement *ICktElement) Powers() ([]complex128, error) {
	C.ctx_CktElement_Get_Powers_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetComplexArrayGR()
}

// Residual currents for each terminal: (magnitude, angle in degrees)
func (cktelement *ICktElement) Residuals() ([]float64, error) {
	C.ctx_CktElement_Get_Residuals_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetFloat64ArrayGR()
}

// Double array of symmetrical component currents (magnitudes only) into each 3-phase terminal
func (cktelement *ICktElement) SeqCurrents() ([]float64, error) {
	C.ctx_CktElement_Get_SeqCurrents_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetFloat64ArrayGR()
}

// Complex array of sequence powers (kW, kvar) into each 3-phase teminal
func (cktelement *ICktElement) SeqPowers() ([]complex128, error) {
	C.ctx_CktElement_Get_SeqPowers_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetComplexArrayGR()
}

// Double array of symmetrical component voltages (magnitudes only) at each 3-phase terminal
func (cktelement *ICktElement) SeqVoltages() ([]float64, error) {
	C.ctx_CktElement_Get_SeqVoltages_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetFloat64ArrayGR()
}

// Complex array of voltages at terminals
func (cktelement *ICktElement) Voltages() ([]complex128, error) {
	C.ctx_CktElement_Get_Voltages_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetComplexArrayGR()
}

// Voltages at each conductor in magnitude, angle form as array of doubles.
func (cktelement *ICktElement) VoltagesMagAng() ([]float64, error) {
	C.ctx_CktElement_Get_VoltagesMagAng_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetFloat64ArrayGR()
}

// YPrim matrix, column order, complex numbers
func (cktelement *ICktElement) Yprim() ([]complex128, error) {
	C.ctx_CktElement_Get_Yprim_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetComplexArrayGR()
}

// Returns true if the current active element is isolated.
// Note that this only fetches the current value. See also the Topology interface.
//
// (API Extension)
func (cktelement *ICktElement) IsIsolated() (bool, error) {
	return (C.ctx_CktElement_Get_IsIsolated(cktelement.ctxPtr) != 0), cktelement.ctx.DSSError()
}

// Returns an array with the total powers (complex, kVA) at ALL terminals of the active circuit element.
func (cktelement *ICktElement) TotalPowers() ([]complex128, error) {
	C.ctx_CktElement_Get_TotalPowers_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetComplexArrayGR()
}

// Array of integers, a copy of the internal NodeRef of the CktElement.
func (cktelement *ICktElement) NodeRef() ([]int32, error) {
	C.ctx_CktElement_Get_NodeRef_GR(cktelement.ctxPtr)
	return cktelement.ctx.GetInt32ArrayGR()
}

type IGenerators struct {
	ICommonData
}

func (generators *IGenerators) Init(ctx *DSSContextPtrs) {
	generators.InitCommon(ctx)
}

// Array of strings with all Generator names in the circuit.
func (generators *IGenerators) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Generators_Get_AllNames(generators.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return generators.ctx.GetStringArray(data, cnt)
}

// Number of Generator objects in active circuit.
func (generators *IGenerators) Count() (int32, error) {
	return (int32)(C.ctx_Generators_Get_Count(generators.ctxPtr)), generators.ctx.DSSError()
}

// Sets the first Generator active. Returns 0 if no more.
func (generators *IGenerators) First() (int32, error) {
	return (int32)(C.ctx_Generators_Get_First(generators.ctxPtr)), generators.ctx.DSSError()
}

// Sets the active Generator by Name.
func (generators *IGenerators) Get_Name() (string, error) {
	result := C.GoString(C.ctx_Generators_Get_Name(generators.ctxPtr))
	return result, generators.ctx.DSSError()
}

// Gets the name of the active Generator.
func (generators *IGenerators) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_Generators_Set_Name(generators.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return generators.ctx.DSSError()
}

// Sets the next Generator active. Returns 0 if no more.
func (generators *IGenerators) Next() (int32, error) {
	return (int32)(C.ctx_Generators_Get_Next(generators.ctxPtr)), generators.ctx.DSSError()
}

// Get the index of the active Generator; index is 1-based: 1..count
func (generators *IGenerators) Get_idx() (int32, error) {
	return (int32)(C.ctx_Generators_Get_idx(generators.ctxPtr)), generators.ctx.DSSError()
}

// Set the active Generator by index; index is 1-based: 1..count
func (generators *IGenerators) Set_idx(value int32) error {
	C.ctx_Generators_Set_idx(generators.ctxPtr, (C.int32_t)(value))
	return generators.ctx.DSSError()
}

// Indicates whether the generator is forced ON regardles of other dispatch criteria.
func (generators *IGenerators) Get_ForcedON() (bool, error) {
	return (C.ctx_Generators_Get_ForcedON(generators.ctxPtr) != 0), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_ForcedON(value bool) error {
	C.ctx_Generators_Set_ForcedON(generators.ctxPtr, ToUint16(value))
	return generators.ctx.DSSError()
}

// Generator Model
func (generators *IGenerators) Get_Model() (int32, error) {
	return (int32)(C.ctx_Generators_Get_Model(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_Model(value int32) error {
	C.ctx_Generators_Set_Model(generators.ctxPtr, (C.int32_t)(value))
	return generators.ctx.DSSError()
}

// Power factor (pos. = producing vars). Updates kvar based on present kW value.
func (generators *IGenerators) Get_PF() (float64, error) {
	return (float64)(C.ctx_Generators_Get_PF(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_PF(value float64) error {
	C.ctx_Generators_Set_PF(generators.ctxPtr, (C.double)(value))
	return generators.ctx.DSSError()
}

// Number of phases
func (generators *IGenerators) Get_Phases() (int32, error) {
	return (int32)(C.ctx_Generators_Get_Phases(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_Phases(value int32) error {
	C.ctx_Generators_Set_Phases(generators.ctxPtr, (C.int32_t)(value))
	return generators.ctx.DSSError()
}

// Array of Names of all generator energy meter registers
func (generators *IGenerators) RegisterNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Generators_Get_RegisterNames(generators.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return generators.ctx.GetStringArray(data, cnt)
}

// Array of valus in generator energy meter registers.
func (generators *IGenerators) RegisterValues() ([]float64, error) {
	C.ctx_Generators_Get_RegisterValues_GR(generators.ctxPtr)
	return generators.ctx.GetFloat64ArrayGR()
}

// Vmaxpu for generator model
func (generators *IGenerators) Get_Vmaxpu() (float64, error) {
	return (float64)(C.ctx_Generators_Get_Vmaxpu(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_Vmaxpu(value float64) error {
	C.ctx_Generators_Set_Vmaxpu(generators.ctxPtr, (C.double)(value))
	return generators.ctx.DSSError()
}

// Vminpu for Generator model
func (generators *IGenerators) Get_Vminpu() (float64, error) {
	return (float64)(C.ctx_Generators_Get_Vminpu(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_Vminpu(value float64) error {
	C.ctx_Generators_Set_Vminpu(generators.ctxPtr, (C.double)(value))
	return generators.ctx.DSSError()
}

// Voltage base for the active generator, kV
func (generators *IGenerators) Get_kV() (float64, error) {
	return (float64)(C.ctx_Generators_Get_kV(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_kV(value float64) error {
	C.ctx_Generators_Set_kV(generators.ctxPtr, (C.double)(value))
	return generators.ctx.DSSError()
}

// kVA rating of the generator
func (generators *IGenerators) Get_kVArated() (float64, error) {
	return (float64)(C.ctx_Generators_Get_kVArated(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_kVArated(value float64) error {
	C.ctx_Generators_Set_kVArated(generators.ctxPtr, (C.double)(value))
	return generators.ctx.DSSError()
}

// kW output for the active generator. kvar is updated for current power factor.
func (generators *IGenerators) Get_kW() (float64, error) {
	return (float64)(C.ctx_Generators_Get_kW(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_kW(value float64) error {
	C.ctx_Generators_Set_kW(generators.ctxPtr, (C.double)(value))
	return generators.ctx.DSSError()
}

// kvar output for the active generator. Updates power factor based on present kW value.
func (generators *IGenerators) Get_kvar() (float64, error) {
	return (float64)(C.ctx_Generators_Get_kvar(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_kvar(value float64) error {
	C.ctx_Generators_Set_kvar(generators.ctxPtr, (C.double)(value))
	return generators.ctx.DSSError()
}

// Name of the loadshape for a daily generation profile.
//
// (API Extension)
func (generators *IGenerators) Get_daily() (string, error) {
	return C.GoString(C.ctx_Generators_Get_daily(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_daily(value string) error {
	value_c := C.CString(value)
	C.ctx_Generators_Set_daily(generators.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return generators.ctx.DSSError()
}

// Name of the loadshape for a duty cycle simulation.
//
// (API Extension)
func (generators *IGenerators) Get_duty() (string, error) {
	return C.GoString(C.ctx_Generators_Get_duty(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_duty(value string) error {
	value_c := C.CString(value)
	C.ctx_Generators_Set_duty(generators.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return generators.ctx.DSSError()
}

// Name of yearly loadshape
//
// (API Extension)
func (generators *IGenerators) Get_Yearly() (string, error) {
	return C.GoString(C.ctx_Generators_Get_Yearly(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_Yearly(value string) error {
	value_c := C.CString(value)
	C.ctx_Generators_Set_Yearly(generators.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return generators.ctx.DSSError()
}

// Response to dispatch multipliers: Fixed=1 (dispatch multipliers do not apply), Variable=0 (follows curves).
//
// Related enumeration: GeneratorStatus
//
// (API Extension)
func (generators *IGenerators) Get_Status() (GeneratorStatus, error) {
	return (GeneratorStatus)(C.ctx_Generators_Get_Status(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_Status(value GeneratorStatus) error {
	C.ctx_Generators_Set_Status(generators.ctxPtr, (C.int32_t)(value))
	return generators.ctx.DSSError()
}

// Generator connection. True/1 if delta connection, False/0 if wye.
//
// (API Extension)
func (generators *IGenerators) Get_IsDelta() (bool, error) {
	return (C.ctx_Generators_Get_IsDelta(generators.ctxPtr) != 0), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_IsDelta(value bool) error {
	C.ctx_Generators_Set_IsDelta(generators.ctxPtr, ToUint16(value))
	return generators.ctx.DSSError()
}

// kVA rating of electrical machine. Applied to machine or inverter definition for Dynamics mode solutions.
//
// (API Extension)
func (generators *IGenerators) Get_kva() (float64, error) {
	return (float64)(C.ctx_Generators_Get_kva(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_kva(value float64) error {
	C.ctx_Generators_Set_kva(generators.ctxPtr, (C.double)(value))
	return generators.ctx.DSSError()
}

// An arbitrary integer number representing the class of Generator so that Generator values may be segregated by class.
//
// (API Extension)
func (generators *IGenerators) Get_Class() (int32, error) {
	return (int32)(C.ctx_Generators_Get_Class_(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_Class(value int32) error {
	C.ctx_Generators_Set_Class_(generators.ctxPtr, (C.int32_t)(value))
	return generators.ctx.DSSError()
}

// Bus to which the Generator is connected. May include specific node specification.
//
// (API Extension)
func (generators *IGenerators) Get_Bus1() (string, error) {
	return C.GoString(C.ctx_Generators_Get_Bus1(generators.ctxPtr)), generators.ctx.DSSError()
}

func (generators *IGenerators) Set_Bus1(value string) error {
	value_c := C.CString(value)
	C.ctx_Generators_Set_Bus1(generators.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return generators.ctx.DSSError()
}

type ILines struct {
	ICommonData
}

func (lines *ILines) Init(ctx *DSSContextPtrs) {
	lines.InitCommon(ctx)
}

// Array of strings with all Line names in the circuit.
func (lines *ILines) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Lines_Get_AllNames(lines.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return lines.ctx.GetStringArray(data, cnt)
}

// Number of Line objects in active circuit.
func (lines *ILines) Count() (int32, error) {
	return (int32)(C.ctx_Lines_Get_Count(lines.ctxPtr)), lines.ctx.DSSError()
}

// Sets the first Line active. Returns 0 if no more.
func (lines *ILines) First() (int32, error) {
	return (int32)(C.ctx_Lines_Get_First(lines.ctxPtr)), lines.ctx.DSSError()
}

// Sets the active Line by Name.
func (lines *ILines) Get_Name() (string, error) {
	result := C.GoString(C.ctx_Lines_Get_Name(lines.ctxPtr))
	return result, lines.ctx.DSSError()
}

// Gets the name of the active Line.
func (lines *ILines) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_Lines_Set_Name(lines.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return lines.ctx.DSSError()
}

// Sets the next Line active. Returns 0 if no more.
func (lines *ILines) Next() (int32, error) {
	return (int32)(C.ctx_Lines_Get_Next(lines.ctxPtr)), lines.ctx.DSSError()
}

// Get the index of the active Line; index is 1-based: 1..count
func (lines *ILines) Get_idx() (int32, error) {
	return (int32)(C.ctx_Lines_Get_idx(lines.ctxPtr)), lines.ctx.DSSError()
}

// Set the active Line by index; index is 1-based: 1..count
func (lines *ILines) Set_idx(value int32) error {
	C.ctx_Lines_Set_idx(lines.ctxPtr, (C.int32_t)(value))
	return lines.ctx.DSSError()
}

func (lines *ILines) New(Name string) (int32, error) {
	Name_c := C.CString(Name)
	defer C.free(unsafe.Pointer(Name_c))
	return (int32)(C.ctx_Lines_New(lines.ctxPtr, Name_c)), lines.ctx.DSSError()
}

// Name of bus for terminal 1.
func (lines *ILines) Get_Bus1() (string, error) {
	return C.GoString(C.ctx_Lines_Get_Bus1(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_Bus1(value string) error {
	value_c := C.CString(value)
	C.ctx_Lines_Set_Bus1(lines.ctxPtr, value_c)
	defer C.free(unsafe.Pointer(value_c))
	return lines.ctx.DSSError()
}

// Name of bus for terminal 2.
func (lines *ILines) Get_Bus2() (string, error) {
	return C.GoString(C.ctx_Lines_Get_Bus2(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_Bus2(value string) error {
	value_c := C.CString(value)
	C.ctx_Lines_Set_Bus2(lines.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return lines.ctx.DSSError()
}

// Zero Sequence capacitance, nanofarads per unit length.
func (lines *ILines) Get_C0() (float64, error) {
	return (float64)(C.ctx_Lines_Get_C0(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_C0(value float64) error {
	C.ctx_Lines_Set_C0(lines.ctxPtr, (C.double)(value))
	return lines.ctx.DSSError()
}

// Positive Sequence capacitance, nanofarads per unit length.
func (lines *ILines) Get_C1() (float64, error) {
	return (float64)(C.ctx_Lines_Get_C1(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_C1(value float64) error {
	C.ctx_Lines_Set_C1(lines.ctxPtr, (C.double)(value))
	return lines.ctx.DSSError()
}

func (lines *ILines) Get_Cmatrix() ([]float64, error) {
	C.ctx_Lines_Get_Cmatrix_GR(lines.ctxPtr)
	return lines.ctx.GetFloat64ArrayGR()
}

func (lines *ILines) Set_Cmatrix(value []float64) error {
	C.ctx_Lines_Set_Cmatrix(lines.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return lines.ctx.DSSError()
}

// Emergency (maximum) ampere rating of Line.
func (lines *ILines) Get_EmergAmps() (float64, error) {
	return (float64)(C.ctx_Lines_Get_EmergAmps(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_EmergAmps(value float64) error {
	C.ctx_Lines_Set_EmergAmps(lines.ctxPtr, (C.double)(value))
	return lines.ctx.DSSError()
}

// Line geometry code
func (lines *ILines) Get_Geometry() (string, error) {
	return C.GoString(C.ctx_Lines_Get_Geometry(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_Geometry(value string) error {
	value_c := C.CString(value)
	C.ctx_Lines_Set_Geometry(lines.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return lines.ctx.DSSError()
}

// Length of line section in units compatible with the LineCode definition.
func (lines *ILines) Get_Length() (float64, error) {
	return (float64)(C.ctx_Lines_Get_Length(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_Length(value float64) error {
	C.ctx_Lines_Set_Length(lines.ctxPtr, (C.double)(value))
	return lines.ctx.DSSError()
}

// Name of LineCode object that defines the impedances.
func (lines *ILines) Get_LineCode() (string, error) {
	return C.GoString(C.ctx_Lines_Get_LineCode(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_LineCode(value string) error {
	value_c := C.CString(value)
	C.ctx_Lines_Set_LineCode(lines.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return lines.ctx.DSSError()
}

// Normal ampere rating of Line.
func (lines *ILines) Get_NormAmps() (float64, error) {
	return (float64)(C.ctx_Lines_Get_NormAmps(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_NormAmps(value float64) error {
	C.ctx_Lines_Set_NormAmps(lines.ctxPtr, (C.double)(value))
	return lines.ctx.DSSError()
}

// Number of customers on this line section.
func (lines *ILines) NumCust() (int32, error) {
	return (int32)(C.ctx_Lines_Get_NumCust(lines.ctxPtr)), lines.ctx.DSSError()
}

// Sets Parent of the active Line to be the active line. Returns 0 if no parent or action fails.
func (lines *ILines) Parent() (int32, error) {
	return (int32)(C.ctx_Lines_Get_Parent(lines.ctxPtr)), lines.ctx.DSSError()
}

// Number of Phases, this Line element.
func (lines *ILines) Get_Phases() (int32, error) {
	return (int32)(C.ctx_Lines_Get_Phases(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_Phases(value int32) error {
	C.ctx_Lines_Set_Phases(lines.ctxPtr, (C.int32_t)(value))
	return lines.ctx.DSSError()
}

// Zero Sequence resistance, ohms per unit length.
func (lines *ILines) Get_R0() (float64, error) {
	return (float64)(C.ctx_Lines_Get_R0(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_R0(value float64) error {
	C.ctx_Lines_Set_R0(lines.ctxPtr, (C.double)(value))
	return lines.ctx.DSSError()
}

// Positive Sequence resistance, ohms per unit length.
func (lines *ILines) Get_R1() (float64, error) {
	return (float64)(C.ctx_Lines_Get_R1(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_R1(value float64) error {
	C.ctx_Lines_Set_R1(lines.ctxPtr, (C.double)(value))
	return lines.ctx.DSSError()
}

// Earth return resistance value used to compute line impedances at power frequency
func (lines *ILines) Get_Rg() (float64, error) {
	return (float64)(C.ctx_Lines_Get_Rg(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_Rg(value float64) error {
	C.ctx_Lines_Set_Rg(lines.ctxPtr, (C.double)(value))
	return lines.ctx.DSSError()
}

// Earth Resistivity, m-ohms
func (lines *ILines) Get_Rho() (float64, error) {
	return (float64)(C.ctx_Lines_Get_Rho(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_Rho(value float64) error {
	C.ctx_Lines_Set_Rho(lines.ctxPtr, (C.double)(value))
	return lines.ctx.DSSError()
}

// Resistance matrix (full), ohms per unit length. Array of doubles.
func (lines *ILines) Get_Rmatrix() ([]float64, error) {
	C.ctx_Lines_Get_Rmatrix_GR(lines.ctxPtr)
	return lines.ctx.GetFloat64ArrayGR()
}

func (lines *ILines) Set_Rmatrix(value []float64) error {
	C.ctx_Lines_Set_Rmatrix(lines.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return lines.ctx.DSSError()
}

// Line spacing code
func (lines *ILines) Get_Spacing() (string, error) {
	return C.GoString(C.ctx_Lines_Get_Spacing(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_Spacing(value string) error {
	value_c := C.CString(value)
	C.ctx_Lines_Set_Spacing(lines.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return lines.ctx.DSSError()
}

// Total Number of customers served from this line section.
func (lines *ILines) TotalCust() (int32, error) {
	return (int32)(C.ctx_Lines_Get_TotalCust(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Get_Units() (LineUnits, error) {
	return (LineUnits)(C.ctx_Lines_Get_Units(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_Units(value LineUnits) error {
	C.ctx_Lines_Set_Units(lines.ctxPtr, (C.int32_t)(value))
	return lines.ctx.DSSError()
}

// Zero Sequence reactance ohms per unit length.
func (lines *ILines) Get_X0() (float64, error) {
	return (float64)(C.ctx_Lines_Get_X0(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_X0(value float64) error {
	C.ctx_Lines_Set_X0(lines.ctxPtr, (C.double)(value))
	return lines.ctx.DSSError()
}

// Positive Sequence reactance, ohms per unit length.
func (lines *ILines) Get_X1() (float64, error) {
	return (float64)(C.ctx_Lines_Get_X1(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_X1(value float64) error {
	C.ctx_Lines_Set_X1(lines.ctxPtr, (C.double)(value))
	return lines.ctx.DSSError()
}

// Earth return reactance value used to compute line impedances at power frequency
func (lines *ILines) Get_Xg() (float64, error) {
	return (float64)(C.ctx_Lines_Get_Xg(lines.ctxPtr)), lines.ctx.DSSError()
}

func (lines *ILines) Set_Xg(value float64) error {
	C.ctx_Lines_Set_Xg(lines.ctxPtr, (C.double)(value))
	return lines.ctx.DSSError()
}

// Reactance matrix (full), ohms per unit length. Array of doubles.
func (lines *ILines) Get_Xmatrix() ([]float64, error) {
	C.ctx_Lines_Get_Xmatrix_GR(lines.ctxPtr)
	return lines.ctx.GetFloat64ArrayGR()
}

func (lines *ILines) Set_Xmatrix(value []float64) error {
	C.ctx_Lines_Set_Xmatrix(lines.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return lines.ctx.DSSError()
}

// Yprimitive for the active line object (complex array).
func (lines *ILines) Get_Yprim() ([]complex128, error) {
	C.ctx_Lines_Get_Yprim_GR(lines.ctxPtr)
	return lines.ctx.GetComplexArrayGR()
}

func (lines *ILines) Set_Yprim(value []complex128) error {
	C.ctx_Lines_Set_Yprim(lines.ctxPtr, (*C.double)((unsafe.Pointer)(&value)), (C.int32_t)(2*len(value)))
	return lines.ctx.DSSError()
}

// Delivers the rating for the current season (in Amps)  if the "SeasonalRatings" option is active
func (lines *ILines) SeasonRating() (float64, error) {
	return (float64)(C.ctx_Lines_Get_SeasonRating(lines.ctxPtr)), lines.ctx.DSSError()
}

// Sets/gets the Line element switch status. Setting it has side-effects to the line parameters.
//
// (API Extension)
func (lines *ILines) Get_IsSwitch() (bool, error) {
	return (C.ctx_Lines_Get_IsSwitch(lines.ctxPtr) != 0), lines.ctx.DSSError()
}

func (lines *ILines) Set_IsSwitch(value bool) error {
	C.ctx_Lines_Set_IsSwitch(lines.ctxPtr, ToUint16(value))
	return lines.ctx.DSSError()
}

type ISettings struct {
	ICommonData
}

func (settings *ISettings) Init(ctx *DSSContextPtrs) {
	settings.InitCommon(ctx)
}

// {True | False*} Designates whether to allow duplicate names of objects
//
// **NOTE**: for DSS-Extensions, we are considering removing this option in a future
// release since it has performance impacts even when not used.
func (settings *ISettings) Get_AllowDuplicates() (bool, error) {
	return (C.ctx_Settings_Get_AllowDuplicates(settings.ctxPtr) != 0), settings.ctx.DSSError()
}

func (settings *ISettings) Set_AllowDuplicates(value bool) error {
	C.ctx_Settings_Set_AllowDuplicates(settings.ctxPtr, ToUint16(value))
	return settings.ctx.DSSError()
}

// List of Buses or (File=xxxx) syntax for the AutoAdd solution mode.
func (settings *ISettings) Get_AutoBusList() (string, error) {
	return C.GoString(C.ctx_Settings_Get_AutoBusList(settings.ctxPtr)), settings.ctx.DSSError()
}

func (settings *ISettings) Set_AutoBusList(value string) error {
	value_c := C.CString(value)
	C.ctx_Settings_Set_AutoBusList(settings.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return settings.ctx.DSSError()
}

// {dssMultiphase (0) * | dssPositiveSeq (1) } Indicate if the circuit model is positive sequence.
func (settings *ISettings) Get_CktModel() (int32, error) {
	return (int32)(C.ctx_Settings_Get_CktModel(settings.ctxPtr)), settings.ctx.DSSError()
}

func (settings *ISettings) Set_CktModel(value int32) error {
	C.ctx_Settings_Set_CktModel(settings.ctxPtr, (C.int32_t)(value))
	return settings.ctx.DSSError()
}

// {True | False*} Denotes whether to trace the control actions to a file.
func (settings *ISettings) Get_ControlTrace() (bool, error) {
	return (C.ctx_Settings_Get_ControlTrace(settings.ctxPtr) != 0), settings.ctx.DSSError()
}

func (settings *ISettings) Set_ControlTrace(value bool) error {
	C.ctx_Settings_Set_ControlTrace(settings.ctxPtr, ToUint16(value))
	return settings.ctx.DSSError()
}

// Per Unit maximum voltage for Emergency conditions.
func (settings *ISettings) Get_EmergVmaxpu() (float64, error) {
	return (float64)(C.ctx_Settings_Get_EmergVmaxpu(settings.ctxPtr)), settings.ctx.DSSError()
}

func (settings *ISettings) Set_EmergVmaxpu(value float64) error {
	C.ctx_Settings_Set_EmergVmaxpu(settings.ctxPtr, (C.double)(value))
	return settings.ctx.DSSError()
}

// Per Unit minimum voltage for Emergency conditions.
func (settings *ISettings) Get_EmergVminpu() (float64, error) {
	return (float64)(C.ctx_Settings_Get_EmergVminpu(settings.ctxPtr)), settings.ctx.DSSError()
}

func (settings *ISettings) Set_EmergVminpu(value float64) error {
	C.ctx_Settings_Set_EmergVminpu(settings.ctxPtr, (C.double)(value))
	return settings.ctx.DSSError()
}

// Integer array defining which energy meter registers to use for computing losses
func (settings *ISettings) Get_LossRegs() ([]int32, error) {
	C.ctx_Settings_Get_LossRegs_GR(settings.ctxPtr)
	return settings.ctx.GetInt32ArrayGR()
}

func (settings *ISettings) Set_LossRegs(value []int32) error {
	C.ctx_Settings_Set_LossRegs(settings.ctxPtr, (*C.int32_t)(&value[0]), (C.int32_t)(len(value)))
	return settings.ctx.DSSError()
}

// Weighting factor applied to Loss register values.
func (settings *ISettings) Get_LossWeight() (float64, error) {
	return (float64)(C.ctx_Settings_Get_LossWeight(settings.ctxPtr)), settings.ctx.DSSError()
}

func (settings *ISettings) Set_LossWeight(value float64) error {
	C.ctx_Settings_Set_LossWeight(settings.ctxPtr, (C.double)(value))
	return settings.ctx.DSSError()
}

// Per Unit maximum voltage for Normal conditions.
func (settings *ISettings) Get_NormVmaxpu() (float64, error) {
	return (float64)(C.ctx_Settings_Get_NormVmaxpu(settings.ctxPtr)), settings.ctx.DSSError()
}

func (settings *ISettings) Set_NormVmaxpu(value float64) error {
	C.ctx_Settings_Set_NormVmaxpu(settings.ctxPtr, (C.double)(value))
	return settings.ctx.DSSError()
}

// Per Unit minimum voltage for Normal conditions.
func (settings *ISettings) Get_NormVminpu() (float64, error) {
	return (float64)(C.ctx_Settings_Get_NormVminpu(settings.ctxPtr)), settings.ctx.DSSError()
}

func (settings *ISettings) Set_NormVminpu(value float64) error {
	C.ctx_Settings_Set_NormVminpu(settings.ctxPtr, (C.double)(value))
	return settings.ctx.DSSError()
}

// Name of LoadShape object that serves as the source of price signal data for yearly simulations, etc.
func (settings *ISettings) Get_PriceCurve() (string, error) {
	return C.GoString(C.ctx_Settings_Get_PriceCurve(settings.ctxPtr)), settings.ctx.DSSError()
}

func (settings *ISettings) Set_PriceCurve(value string) error {
	value_c := C.CString(value)
	C.ctx_Settings_Set_PriceCurve(settings.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return settings.ctx.DSSError()
}

// Price Signal for the Circuit
func (settings *ISettings) Get_PriceSignal() (float64, error) {
	return (float64)(C.ctx_Settings_Get_PriceSignal(settings.ctxPtr)), settings.ctx.DSSError()
}

func (settings *ISettings) Set_PriceSignal(value float64) error {
	C.ctx_Settings_Set_PriceSignal(settings.ctxPtr, (C.double)(value))
	return settings.ctx.DSSError()
}

// Gets value of trapezoidal integration flag in energy meters. Defaults to `False`.
func (settings *ISettings) Get_Trapezoidal() (bool, error) {
	return (C.ctx_Settings_Get_Trapezoidal(settings.ctxPtr) != 0), settings.ctx.DSSError()
}

func (settings *ISettings) Set_Trapezoidal(value bool) error {
	C.ctx_Settings_Set_Trapezoidal(settings.ctxPtr, ToUint16(value))
	return settings.ctx.DSSError()
}

// Array of Integers defining energy meter registers to use for computing UE
func (settings *ISettings) Get_UEregs() ([]int32, error) {
	C.ctx_Settings_Get_UEregs_GR(settings.ctxPtr)
	return settings.ctx.GetInt32ArrayGR()
}

func (settings *ISettings) Set_UEregs(value []int32) error {
	C.ctx_Settings_Set_UEregs(settings.ctxPtr, (*C.int32_t)(&value[0]), (C.int32_t)(len(value)))
	return settings.ctx.DSSError()
}

// Weighting factor applied to UE register values.
func (settings *ISettings) Get_UEweight() (float64, error) {
	return (float64)(C.ctx_Settings_Get_UEweight(settings.ctxPtr)), settings.ctx.DSSError()
}

func (settings *ISettings) Set_UEweight(value float64) error {
	C.ctx_Settings_Set_UEweight(settings.ctxPtr, (C.double)(value))
	return settings.ctx.DSSError()
}

// Array of doubles defining the legal voltage bases in kV L-L
func (settings *ISettings) Get_VoltageBases() ([]float64, error) {
	C.ctx_Settings_Get_VoltageBases_GR(settings.ctxPtr)
	return settings.ctx.GetFloat64ArrayGR()
}

func (settings *ISettings) Set_VoltageBases(value []float64) error {
	C.ctx_Settings_Set_VoltageBases(settings.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return settings.ctx.DSSError()
}

// {True | False*}  Locks Zones on energy meters to prevent rebuilding if a circuit change occurs.
func (settings *ISettings) Get_ZoneLock() (bool, error) {
	return (C.ctx_Settings_Get_ZoneLock(settings.ctxPtr) != 0), settings.ctx.DSSError()
}

func (settings *ISettings) Set_ZoneLock(value bool) error {
	C.ctx_Settings_Set_ZoneLock(settings.ctxPtr, ToUint16(value))
	return settings.ctx.DSSError()
}

func (settings *ISettings) Set_AllocationFactors(value float64) error {
	C.ctx_Settings_Set_AllocationFactors(settings.ctxPtr, (C.double)(value))
	return settings.ctx.DSSError()
}

// Controls whether the terminals are checked when updating the currents in Load component. Defaults to True.
// If the loads are guaranteed to have their terminals closed throughout the simulation, this can be set to False to save some time.
//
// (API Extension)
func (settings *ISettings) Get_LoadsTerminalCheck() (bool, error) {
	return (C.ctx_Settings_Get_LoadsTerminalCheck(settings.ctxPtr) != 0), settings.ctx.DSSError()
}

func (settings *ISettings) Set_LoadsTerminalCheck(value bool) error {
	C.ctx_Settings_Set_LoadsTerminalCheck(settings.ctxPtr, ToUint16(value))
	return settings.ctx.DSSError()
}

// Controls whether `First`/`Next` iteration includes or skips disabled circuit elements.
// The default behavior from OpenDSS is to skip those. The user can still activate the element by name or index.
//
// The default value for IterateDisabled is 0, keeping the original behavior.
// Set it to 1 (or `True`) to include disabled elements.
// Other numeric values are reserved for other potential behaviors.
//
// (API Extension)
func (settings *ISettings) Get_IterateDisabled() (int32, error) {
	return (int32)(C.ctx_Settings_Get_IterateDisabled(settings.ctxPtr)), settings.ctx.DSSError()
}

func (settings *ISettings) Set_IterateDisabled(value int32) error {
	C.ctx_Settings_Set_IterateDisabled(settings.ctxPtr, (C.int32_t)(value))
	return settings.ctx.DSSError()
}

type IActiveClass struct {
	ICommonData
}

func (activeclass *IActiveClass) Init(ctx *DSSContextPtrs) {
	activeclass.InitCommon(ctx)
}

// Returns name of active class.
func (activeclass *IActiveClass) ActiveClassName() (string, error) {
	return C.GoString(C.ctx_ActiveClass_Get_ActiveClassName(activeclass.ctxPtr)), activeclass.ctx.DSSError()
}

// Array of strings consisting of all element names in the active class.
func (activeclass *IActiveClass) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_ActiveClass_Get_AllNames(activeclass.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return activeclass.ctx.GetStringArray(data, cnt)
}

// Number of elements in Active Class. Same as NumElements Property.
func (activeclass *IActiveClass) Count() (int32, error) {
	return (int32)(C.ctx_ActiveClass_Get_Count(activeclass.ctxPtr)), activeclass.ctx.DSSError()
}

// Sets first element in the active class to be the active DSS object. If object is a CktElement, ActiveCktELment also points to this element. Returns 0 if none.
func (activeclass *IActiveClass) First() (int32, error) {
	return (int32)(C.ctx_ActiveClass_Get_First(activeclass.ctxPtr)), activeclass.ctx.DSSError()
}

// Name of the Active Element of the Active Class
func (activeclass *IActiveClass) Get_Name() (string, error) {
	return C.GoString(C.ctx_ActiveClass_Get_Name(activeclass.ctxPtr)), activeclass.ctx.DSSError()
}

func (activeclass *IActiveClass) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_ActiveClass_Set_Name(activeclass.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return activeclass.ctx.DSSError()
}

// Sets next element in active class to be the active DSS object. If object is a CktElement, ActiveCktElement also points to this element.  Returns 0 if no more.
func (activeclass *IActiveClass) Next() (int32, error) {
	return (int32)(C.ctx_ActiveClass_Get_Next(activeclass.ctxPtr)), activeclass.ctx.DSSError()
}

// Number of elements in this class. Same as Count property.
func (activeclass *IActiveClass) NumElements() (int32, error) {
	return (int32)(C.ctx_ActiveClass_Get_NumElements(activeclass.ctxPtr)), activeclass.ctx.DSSError()
}

// Get the name of the parent class of the active class
func (activeclass *IActiveClass) ActiveClassParent() (string, error) {
	return C.GoString(C.ctx_ActiveClass_Get_ActiveClassParent(activeclass.ctxPtr)), activeclass.ctx.DSSError()
}

// Returns the data (as a list) of all elements from the active class as a JSON-encoded string.
//
// The `options` parameter contains bit-flags to toggle specific features.
// See `Obj_ToJSON` (C-API) for more.
//
// Additionally, the `ExcludeDisabled` flag can be used to excluded disabled elements from the output.
//
// (API Extension)
func (activeclass *IActiveClass) ToJSON(options int32) (string, error) {
	return C.GoString(C.ctx_ActiveClass_ToJSON(activeclass.ctxPtr, (C.int32_t)(options))), activeclass.ctx.DSSError()
}

type ICapControls struct {
	ICommonData
}

func (capcontrols *ICapControls) Init(ctx *DSSContextPtrs) {
	capcontrols.InitCommon(ctx)
}

// Array of strings with all CapControl names in the circuit.
func (capcontrols *ICapControls) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_CapControls_Get_AllNames(capcontrols.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return capcontrols.ctx.GetStringArray(data, cnt)
}

// Number of CapControl objects in active circuit.
func (capcontrols *ICapControls) Count() (int32, error) {
	return (int32)(C.ctx_CapControls_Get_Count(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

// Sets the first CapControl active. Returns 0 if no more.
func (capcontrols *ICapControls) First() (int32, error) {
	return (int32)(C.ctx_CapControls_Get_First(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

// Sets the active CapControl by Name.
func (capcontrols *ICapControls) Get_Name() (string, error) {
	result := C.GoString(C.ctx_CapControls_Get_Name(capcontrols.ctxPtr))
	return result, capcontrols.ctx.DSSError()
}

// Gets the name of the active CapControl.
func (capcontrols *ICapControls) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_CapControls_Set_Name(capcontrols.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return capcontrols.ctx.DSSError()
}

// Sets the next CapControl active. Returns 0 if no more.
func (capcontrols *ICapControls) Next() (int32, error) {
	return (int32)(C.ctx_CapControls_Get_Next(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

// Get the index of the active CapControl; index is 1-based: 1..count
func (capcontrols *ICapControls) Get_idx() (int32, error) {
	return (int32)(C.ctx_CapControls_Get_idx(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

// Set the active CapControl by index; index is 1-based: 1..count
func (capcontrols *ICapControls) Set_idx(value int32) error {
	C.ctx_CapControls_Set_idx(capcontrols.ctxPtr, (C.int32_t)(value))
	return capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Reset() error {
	C.ctx_CapControls_Reset(capcontrols.ctxPtr)
	return capcontrols.ctx.DSSError()
}

// Transducer ratio from pirmary current to control current.
func (capcontrols *ICapControls) Get_CTratio() (float64, error) {
	return (float64)(C.ctx_CapControls_Get_CTratio(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Set_CTratio(value float64) error {
	C.ctx_CapControls_Set_CTratio(capcontrols.ctxPtr, (C.double)(value))
	return capcontrols.ctx.DSSError()
}

// Name of the Capacitor that is controlled.
func (capcontrols *ICapControls) Get_Capacitor() (string, error) {
	return C.GoString(C.ctx_CapControls_Get_Capacitor(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Set_Capacitor(value string) error {
	value_c := C.CString(value)
	C.ctx_CapControls_Set_Capacitor(capcontrols.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Get_DeadTime() (float64, error) {
	return (float64)(C.ctx_CapControls_Get_DeadTime(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Set_DeadTime(value float64) error {
	C.ctx_CapControls_Set_DeadTime(capcontrols.ctxPtr, (C.double)(value))
	return capcontrols.ctx.DSSError()
}

// Time delay [s] to switch on after arming.  Control may reset before actually switching.
func (capcontrols *ICapControls) Get_Delay() (float64, error) {
	return (float64)(C.ctx_CapControls_Get_Delay(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Set_Delay(value float64) error {
	C.ctx_CapControls_Set_Delay(capcontrols.ctxPtr, (C.double)(value))
	return capcontrols.ctx.DSSError()
}

// Time delay [s] before switching off a step. Control may reset before actually switching.
func (capcontrols *ICapControls) Get_DelayOff() (float64, error) {
	return (float64)(C.ctx_CapControls_Get_DelayOff(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Set_DelayOff(value float64) error {
	C.ctx_CapControls_Set_DelayOff(capcontrols.ctxPtr, (C.double)(value))
	return capcontrols.ctx.DSSError()
}

// Type of automatic controller.
func (capcontrols *ICapControls) Get_Mode() (int32, error) {
	return (int32)(C.ctx_CapControls_Get_Mode(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Set_Mode(value int32) error {
	C.ctx_CapControls_Set_Mode(capcontrols.ctxPtr, (C.int32_t)(value))
	return capcontrols.ctx.DSSError()
}

// Full name of the element that PT and CT are connected to.
func (capcontrols *ICapControls) Get_MonitoredObj() (string, error) {
	return C.GoString(C.ctx_CapControls_Get_MonitoredObj(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Set_MonitoredObj(value string) error {
	value_c := C.CString(value)
	C.ctx_CapControls_Set_MonitoredObj(capcontrols.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return capcontrols.ctx.DSSError()
}

// Terminal number on the element that PT and CT are connected to.
func (capcontrols *ICapControls) Get_MonitoredTerm() (int32, error) {
	return (int32)(C.ctx_CapControls_Get_MonitoredTerm(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Set_MonitoredTerm(value int32) error {
	C.ctx_CapControls_Set_MonitoredTerm(capcontrols.ctxPtr, (C.int32_t)(value))
	return capcontrols.ctx.DSSError()
}

// Threshold to switch off a step. See Mode for units.
func (capcontrols *ICapControls) Get_OFFSetting() (float64, error) {
	return (float64)(C.ctx_CapControls_Get_OFFSetting(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Set_OFFSetting(value float64) error {
	C.ctx_CapControls_Set_OFFSetting(capcontrols.ctxPtr, (C.double)(value))
	return capcontrols.ctx.DSSError()
}

// Threshold to arm or switch on a step.  See Mode for units.
func (capcontrols *ICapControls) Get_ONSetting() (float64, error) {
	return (float64)(C.ctx_CapControls_Get_ONSetting(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Set_ONSetting(value float64) error {
	C.ctx_CapControls_Set_ONSetting(capcontrols.ctxPtr, (C.double)(value))
	return capcontrols.ctx.DSSError()
}

// Transducer ratio from primary feeder to control voltage.
func (capcontrols *ICapControls) Get_PTratio() (float64, error) {
	return (float64)(C.ctx_CapControls_Get_PTratio(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Set_PTratio(value float64) error {
	C.ctx_CapControls_Set_PTratio(capcontrols.ctxPtr, (C.double)(value))
	return capcontrols.ctx.DSSError()
}

// Enables Vmin and Vmax to override the control Mode
func (capcontrols *ICapControls) Get_UseVoltOverride() (bool, error) {
	return (C.ctx_CapControls_Get_UseVoltOverride(capcontrols.ctxPtr) != 0), capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Set_UseVoltOverride(value bool) error {
	C.ctx_CapControls_Set_UseVoltOverride(capcontrols.ctxPtr, ToUint16(value))
	return capcontrols.ctx.DSSError()
}

// With VoltOverride, swtich off whenever PT voltage exceeds this level.
func (capcontrols *ICapControls) Get_Vmax() (float64, error) {
	return (float64)(C.ctx_CapControls_Get_Vmax(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Set_Vmax(value float64) error {
	C.ctx_CapControls_Set_Vmax(capcontrols.ctxPtr, (C.double)(value))
	return capcontrols.ctx.DSSError()
}

// With VoltOverride, switch ON whenever PT voltage drops below this level.
func (capcontrols *ICapControls) Get_Vmin() (float64, error) {
	return (float64)(C.ctx_CapControls_Get_Vmin(capcontrols.ctxPtr)), capcontrols.ctx.DSSError()
}

func (capcontrols *ICapControls) Set_Vmin(value float64) error {
	C.ctx_CapControls_Set_Vmin(capcontrols.ctxPtr, (C.double)(value))
	return capcontrols.ctx.DSSError()
}

type ICircuit struct {
	ICommonData

	Buses            IBus
	CktElements      ICktElement
	ActiveElement    ICktElement
	Solution         ISolution
	ActiveBus        IBus
	Generators       IGenerators
	Meters           IMeters
	Monitors         IMonitors
	Settings         ISettings
	Lines            ILines
	CtrlQueue        ICtrlQueue
	Loads            ILoads
	ActiveCktElement ICktElement
	ActiveDSSElement IDSSElement
	ActiveClass      IActiveClass
	CapControls      ICapControls
	RegControls      IRegControls
	SwtControls      ISwtControls
	Transformers     ITransformers
	Capacitors       ICapacitors
	Topology         ITopology
	Sensors          ISensors
	XYCurves         IXYCurves
	PDElements       IPDElements
	Reclosers        IReclosers
	Relays           IRelays
	LoadShapes       ILoadShapes
	Fuses            IFuses
	// DSSim_Coms IDSSimComs
	PVSystems      IPVSystems
	Vsources       IVsources
	LineCodes      ILineCodes
	LineGeometries ILineGeometries
	LineSpacings   ILineSpacings
	WireData       IWireData
	CNData         ICNData
	TSData         ITSData
	Reactors       IReactors
	ReduceCkt      IReduceCkt
	Storages       IStorages
	GICSources     IGICSources
	Parallel       IParallel
}

func (circuit *ICircuit) Init(ctx *DSSContextPtrs) {
	circuit.InitCommon(ctx)
	circuit.Buses.Init(ctx)
	circuit.CktElements.Init(ctx)
	circuit.ActiveElement.Init(ctx)
	circuit.Solution.Init(ctx)
	circuit.ActiveBus.Init(ctx)
	circuit.Generators.Init(ctx)
	circuit.Meters.Init(ctx)
	circuit.Monitors.Init(ctx)
	circuit.Settings.Init(ctx)
	circuit.Lines.Init(ctx)
	circuit.CtrlQueue.Init(ctx)
	circuit.Loads.Init(ctx)
	circuit.ActiveCktElement.Init(ctx)
	circuit.ActiveDSSElement.Init(ctx)
	circuit.ActiveClass.Init(ctx)
	circuit.CapControls.Init(ctx)
	circuit.RegControls.Init(ctx)
	circuit.SwtControls.Init(ctx)
	circuit.Transformers.Init(ctx)
	circuit.Capacitors.Init(ctx)
	circuit.Topology.Init(ctx)
	circuit.Sensors.Init(ctx)
	circuit.XYCurves.Init(ctx)
	circuit.PDElements.Init(ctx)
	circuit.Reclosers.Init(ctx)
	circuit.Relays.Init(ctx)
	circuit.LoadShapes.Init(ctx)
	circuit.Fuses.Init(ctx)
	// circuit.DSSim_Coms.Init(ctx)
	circuit.PVSystems.Init(ctx)
	circuit.Vsources.Init(ctx)
	circuit.LineCodes.Init(ctx)
	circuit.LineGeometries.Init(ctx)
	circuit.LineSpacings.Init(ctx)
	circuit.WireData.Init(ctx)
	circuit.CNData.Init(ctx)
	circuit.TSData.Init(ctx)
	circuit.Reactors.Init(ctx)
	circuit.ReduceCkt.Init(ctx)
	circuit.Storages.Init(ctx)
	circuit.GICSources.Init(ctx)
	circuit.Parallel.Init(ctx)
}

// Activates and returns a bus by its (zero-based) index.
// Returns a reference to the existing ActiveBus.
func (circuit *ICircuit) Get_Buses(idx int32) (*IBus, error) {
	if C.ctx_Circuit_SetActiveBusi(circuit.ctxPtr, (C.int32_t)(idx)) < 0 {
		return nil, circuit.ctx.DSSError()
	}
	return &circuit.ActiveBus, circuit.ctx.DSSError()
}

// Activates and returns a bus by its name.
func (circuit *ICircuit) get_Buses(name string) (*IBus, error) {
	name_c := C.CString(name)
	if C.ctx_Circuit_SetActiveBus(circuit.ctxPtr, name_c) < 0 {
		C.free(unsafe.Pointer(name_c))
		return nil, circuit.ctx.DSSError()
	}
	C.free(unsafe.Pointer(name_c))
	return &circuit.ActiveBus, circuit.ctx.DSSError()
}

// Activates and returns a CktElement by its global (zero-based) index.
func (circuit *ICircuit) get_CktElementsi(idx int32) (*ICktElement, error) {
	C.ctx_Circuit_SetCktElementIndex(circuit.ctxPtr, (C.int32_t)(idx))
	return &circuit.ActiveCktElement, circuit.ctx.DSSError()
}

// Activates and returns a CktElement by its full name (e.g. "load.abc").
func (circuit *ICircuit) get_CktElements(fullName string) (*ICktElement, error) {
	fullName_c := C.CString(fullName)
	C.ctx_Circuit_SetCktElementName(circuit.ctxPtr, fullName_c)
	C.free(unsafe.Pointer(fullName_c))
	return &circuit.ActiveCktElement, circuit.ctx.DSSError()
}

func (circuit *ICircuit) Capacity(Start float64, Increment float64) (float64, error) {
	return (float64)(C.ctx_Circuit_Capacity(circuit.ctxPtr, (C.double)(Start), (C.double)(Increment))), circuit.ctx.DSSError()
}

func (circuit *ICircuit) Disable(Name string) error {
	Name_c := C.CString(Name)
	C.ctx_Circuit_Disable(circuit.ctxPtr, Name_c)
	C.free(unsafe.Pointer(Name_c))
	return circuit.ctx.DSSError()
}

func (circuit *ICircuit) Enable(Name string) error {
	Name_c := C.CString(Name)
	C.ctx_Circuit_Enable(circuit.ctxPtr, Name_c)
	C.free(unsafe.Pointer(Name_c))
	return circuit.ctx.DSSError()
}

func (circuit *ICircuit) EndOfTimeStepUpdate() error {
	C.ctx_Circuit_EndOfTimeStepUpdate(circuit.ctxPtr)
	return circuit.ctx.DSSError()
}

func (circuit *ICircuit) FirstElement() (int32, error) {
	return (int32)(C.ctx_Circuit_FirstElement(circuit.ctxPtr)), circuit.ctx.DSSError()
}

func (circuit *ICircuit) FirstPCElement() (int32, error) {
	return (int32)(C.ctx_Circuit_FirstPCElement(circuit.ctxPtr)), circuit.ctx.DSSError()
}

func (circuit *ICircuit) FirstPDElement() (int32, error) {
	return (int32)(C.ctx_Circuit_FirstPDElement(circuit.ctxPtr)), circuit.ctx.DSSError()
}

// Returns an array of doubles representing the distances to parent EnergyMeter. Sequence of array corresponds to other node ByPhase properties.
func (circuit *ICircuit) AllNodeDistancesByPhase(Phase int32) ([]float64, error) {
	C.ctx_Circuit_Get_AllNodeDistancesByPhase_GR(circuit.ctxPtr, (C.int32_t)(Phase))
	return circuit.ctx.GetFloat64ArrayGR()
}

// Return array of strings of the node names for the By Phase criteria. Sequence corresponds to other ByPhase properties.
func (circuit *ICircuit) AllNodeNamesByPhase(Phase int32) ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Circuit_Get_AllNodeNamesByPhase(circuit.ctxPtr, &data, (*C.int32_t)(&cnt[0]), (C.int32_t)(Phase))
	return circuit.ctx.GetStringArray(data, cnt)
}

// Returns Array of doubles represent voltage magnitudes for nodes on the specified phase.
func (circuit *ICircuit) AllNodeVmagByPhase(Phase int32) ([]float64, error) {
	C.ctx_Circuit_Get_AllNodeVmagByPhase_GR(circuit.ctxPtr, (C.int32_t)(Phase))
	return circuit.ctx.GetFloat64ArrayGR()
}

// Returns array of per unit voltage magnitudes for each node by phase
func (circuit *ICircuit) AllNodeVmagPUByPhase(Phase int32) ([]float64, error) {
	C.ctx_Circuit_Get_AllNodeVmagPUByPhase_GR(circuit.ctxPtr, (C.int32_t)(Phase))
	return circuit.ctx.GetFloat64ArrayGR()
}

func (circuit *ICircuit) NextElement() (int32, error) {
	return (int32)(C.ctx_Circuit_NextElement(circuit.ctxPtr)), circuit.ctx.DSSError()
}

func (circuit *ICircuit) NextPCElement() (int32, error) {
	return (int32)(C.ctx_Circuit_NextPCElement(circuit.ctxPtr)), circuit.ctx.DSSError()
}

func (circuit *ICircuit) NextPDElement() (int32, error) {
	return (int32)(C.ctx_Circuit_NextPDElement(circuit.ctxPtr)), circuit.ctx.DSSError()
}

func (circuit *ICircuit) Sample() error {
	C.ctx_Circuit_Sample(circuit.ctxPtr)
	return circuit.ctx.DSSError()
}

func (circuit *ICircuit) SaveSample() error {
	C.ctx_Circuit_SaveSample(circuit.ctxPtr)
	return circuit.ctx.DSSError()
}

func (circuit *ICircuit) SetActiveBus(BusName string) (int32, error) {
	BusName_c := C.CString(BusName)
	defer C.free(unsafe.Pointer(BusName_c))
	return (int32)(C.ctx_Circuit_SetActiveBus(circuit.ctxPtr, BusName_c)), circuit.ctx.DSSError()
}

func (circuit *ICircuit) SetActiveBusi(BusIndex int32) (int32, error) {
	return (int32)(C.ctx_Circuit_SetActiveBusi(circuit.ctxPtr, (C.int32_t)(BusIndex))), circuit.ctx.DSSError()
}

func (circuit *ICircuit) SetActiveClass(ClassName string) (int32, error) {
	ClassName_c := C.CString(ClassName)
	defer C.free(unsafe.Pointer(ClassName_c))
	return (int32)(C.ctx_Circuit_SetActiveClass(circuit.ctxPtr, ClassName_c)), circuit.ctx.DSSError()
}

func (circuit *ICircuit) SetActiveElement(FullName string) (int32, error) {
	FullName_c := C.CString(FullName)
	defer C.free(unsafe.Pointer(FullName_c))
	return (int32)(C.ctx_Circuit_SetActiveElement(circuit.ctxPtr, FullName_c)), circuit.ctx.DSSError()
}

func (circuit *ICircuit) UpdateStorage() error {
	C.ctx_Circuit_UpdateStorage(circuit.ctxPtr)
	return circuit.ctx.DSSError()
}

// Returns distance from each bus to parent EnergyMeter. Corresponds to sequence in AllBusNames.
func (circuit *ICircuit) AllBusDistances() ([]float64, error) {
	C.ctx_Circuit_Get_AllBusDistances_GR(circuit.ctxPtr)
	return circuit.ctx.GetFloat64ArrayGR()
}

// Array of strings containing names of all buses in circuit (see AllNodeNames).
func (circuit *ICircuit) AllBusNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Circuit_Get_AllBusNames(circuit.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return circuit.ctx.GetStringArray(data, cnt)
}

// Array of magnitudes (doubles) of voltages at all buses
func (circuit *ICircuit) AllBusVmag() ([]float64, error) {
	C.ctx_Circuit_Get_AllBusVmag_GR(circuit.ctxPtr)
	return circuit.ctx.GetFloat64ArrayGR()
}

// Double Array of all bus voltages (each node) magnitudes in Per unit
func (circuit *ICircuit) AllBusVmagPu() ([]float64, error) {
	C.ctx_Circuit_Get_AllBusVmagPu_GR(circuit.ctxPtr)
	return circuit.ctx.GetFloat64ArrayGR()
}

// Complex array of all bus, node voltages from most recent solution
func (circuit *ICircuit) AllBusVolts() ([]complex128, error) {
	C.ctx_Circuit_Get_AllBusVolts_GR(circuit.ctxPtr)
	return circuit.ctx.GetComplexArrayGR()
}

// Array of total losses (complex) in each circuit element
func (circuit *ICircuit) AllElementLosses() ([]complex128, error) {
	C.ctx_Circuit_Get_AllElementLosses_GR(circuit.ctxPtr)
	return circuit.ctx.GetComplexArrayGR()
}

// Array of strings containing Full Name of all elements.
func (circuit *ICircuit) AllElementNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Circuit_Get_AllElementNames(circuit.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return circuit.ctx.GetStringArray(data, cnt)
}

// Returns an array of distances from parent EnergyMeter for each Node. Corresponds to AllBusVMag sequence.
func (circuit *ICircuit) AllNodeDistances() ([]float64, error) {
	C.ctx_Circuit_Get_AllNodeDistances_GR(circuit.ctxPtr)
	return circuit.ctx.GetFloat64ArrayGR()
}

// Array of strings containing full name of each node in system in same order as returned by AllBusVolts, etc.
func (circuit *ICircuit) AllNodeNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Circuit_Get_AllNodeNames(circuit.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return circuit.ctx.GetStringArray(data, cnt)
}

// Complex total line losses in the circuit
func (circuit *ICircuit) LineLosses() (complex128, error) {
	C.ctx_Circuit_Get_LineLosses_GR(circuit.ctxPtr)
	return circuit.ctx.GetComplexSimpleGR()
}

// Total losses in active circuit, complex number (two-element array of double).
func (circuit *ICircuit) Losses() (complex128, error) {
	C.ctx_Circuit_Get_Losses_GR(circuit.ctxPtr)
	return circuit.ctx.GetComplexSimpleGR()
}

// Name of the active circuit.
func (circuit *ICircuit) Name() (string, error) {
	return C.GoString(C.ctx_Circuit_Get_Name(circuit.ctxPtr)), circuit.ctx.DSSError()
}

// Total number of Buses in the circuit.
func (circuit *ICircuit) NumBuses() (int32, error) {
	return (int32)(C.ctx_Circuit_Get_NumBuses(circuit.ctxPtr)), circuit.ctx.DSSError()
}

// Number of CktElements in the circuit.
func (circuit *ICircuit) NumCktElements() (int32, error) {
	return (int32)(C.ctx_Circuit_Get_NumCktElements(circuit.ctxPtr)), circuit.ctx.DSSError()
}

// Total number of nodes in the circuit.
func (circuit *ICircuit) NumNodes() (int32, error) {
	return (int32)(C.ctx_Circuit_Get_NumNodes(circuit.ctxPtr)), circuit.ctx.DSSError()
}

// Sets Parent PD element, if any, to be the active circuit element and returns index>0; Returns 0 if it fails or not applicable.
func (circuit *ICircuit) ParentPDElement() (int32, error) {
	return (int32)(C.ctx_Circuit_Get_ParentPDElement(circuit.ctxPtr)), circuit.ctx.DSSError()
}

// Complex losses in all transformers designated to substations.
func (circuit *ICircuit) SubstationLosses() (complex128, error) {
	C.ctx_Circuit_Get_SubstationLosses_GR(circuit.ctxPtr)
	return circuit.ctx.GetComplexSimpleGR()
}

// System Y matrix (after a solution has been performed).
// This is deprecated as it returns a dense matrix. Only use it for small systems.
// For large-scale systems, prefer YMatrix.GetCompressedYMatrix.
func (circuit *ICircuit) SystemY() ([]complex128, error) {
	C.ctx_Circuit_Get_SystemY_GR(circuit.ctxPtr)
	return circuit.ctx.GetComplexArrayGR()
}

// Total power (complex), kVA delivered to the circuit
func (circuit *ICircuit) TotalPower() (complex128, error) {
	C.ctx_Circuit_Get_TotalPower_GR(circuit.ctxPtr)
	return circuit.ctx.GetComplexSimpleGR()
}

// Array of doubles containing complex injection currents for the present solution. It is the "I" vector of I=YV
func (circuit *ICircuit) YCurrents() ([]complex128, error) {
	C.ctx_Circuit_Get_YCurrents_GR(circuit.ctxPtr)
	return circuit.ctx.GetComplexArrayGR()
}

// Array of strings containing the names of the nodes in the same order as the Y matrix
func (circuit *ICircuit) YNodeOrder() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Circuit_Get_YNodeOrder(circuit.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return circuit.ctx.GetStringArray(data, cnt)
}

// Complex array of actual node voltages in same order as SystemY matrix.
func (circuit *ICircuit) YNodeVarray() ([]complex128, error) {
	C.ctx_Circuit_Get_YNodeVarray_GR(circuit.ctxPtr)
	return circuit.ctx.GetComplexArrayGR()
}

// Returns data for all objects and basic circuit properties as a JSON-encoded string.
//
// The JSON data is organized using the JSON schema proposed at
// https://github.com/dss-extensions/AltDSS-Schema
//
// The `options` parameter contains bit-flags to toggle specific features.
// See the enum `DSSJSONFlags` or `Obj_ToJSON` (C-API) for more.
//
// (API Extension)
func (circuit *ICircuit) ToJSON(options int32) (string, error) {
	return C.GoString(C.ctx_Circuit_ToJSON(circuit.ctxPtr, (C.int32_t)(options))), circuit.ctx.DSSError()
}

type ICtrlQueue struct {
	ICommonData
}

func (ctrlqueue *ICtrlQueue) Init(ctx *DSSContextPtrs) {
	ctrlqueue.InitCommon(ctx)
}

func (ctrlqueue *ICtrlQueue) ClearActions() error {
	C.ctx_CtrlQueue_ClearActions(ctrlqueue.ctxPtr)
	return ctrlqueue.ctx.DSSError()
}

func (ctrlqueue *ICtrlQueue) ClearQueue() error {
	C.ctx_CtrlQueue_ClearQueue(ctrlqueue.ctxPtr)
	return ctrlqueue.ctx.DSSError()
}

func (ctrlqueue *ICtrlQueue) Delete(ActionHandle int32) error {
	C.ctx_CtrlQueue_Delete(ctrlqueue.ctxPtr, (C.int32_t)(ActionHandle))
	return ctrlqueue.ctx.DSSError()
}

func (ctrlqueue *ICtrlQueue) DoAllQueue() error {
	C.ctx_CtrlQueue_DoAllQueue(ctrlqueue.ctxPtr)
	return ctrlqueue.ctx.DSSError()
}

func (ctrlqueue *ICtrlQueue) Show() error {
	C.ctx_CtrlQueue_Show(ctrlqueue.ctxPtr)
	return ctrlqueue.ctx.DSSError()
}

// Code for the active action. Long integer code to tell the control device what to do
func (ctrlqueue *ICtrlQueue) ActionCode() (int32, error) {
	return (int32)(C.ctx_CtrlQueue_Get_ActionCode(ctrlqueue.ctxPtr)), ctrlqueue.ctx.DSSError()
}

// Handle (User defined) to device that must act on the pending action.
func (ctrlqueue *ICtrlQueue) DeviceHandle() (int32, error) {
	return (int32)(C.ctx_CtrlQueue_Get_DeviceHandle(ctrlqueue.ctxPtr)), ctrlqueue.ctx.DSSError()
}

// Number of Actions on the current actionlist (that have been popped off the control queue by CheckControlActions)
func (ctrlqueue *ICtrlQueue) NumActions() (int32, error) {
	return (int32)(C.ctx_CtrlQueue_Get_NumActions(ctrlqueue.ctxPtr)), ctrlqueue.ctx.DSSError()
}

// Push a control action onto the DSS control queue by time, action code, and device handle (user defined). Returns Control Queue handle.
func (ctrlqueue *ICtrlQueue) Push(Hour int32, Seconds float64, ActionCode int32, DeviceHandle int32) (int32, error) {
	return (int32)(C.ctx_CtrlQueue_Push(ctrlqueue.ctxPtr, (C.int32_t)(Hour), (C.double)(Seconds), (C.int32_t)(ActionCode), (C.int32_t)(DeviceHandle))), ctrlqueue.ctx.DSSError()
}

// Pops next action off the action list and makes it the active action. Returns zero if none.
func (ctrlqueue *ICtrlQueue) PopAction() (int32, error) {
	return (int32)(C.ctx_CtrlQueue_Get_PopAction(ctrlqueue.ctxPtr)), ctrlqueue.ctx.DSSError()
}

// Array of strings containing the entire queue in CSV format
func (ctrlqueue *ICtrlQueue) Queue() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_CtrlQueue_Get_Queue(ctrlqueue.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return ctrlqueue.ctx.GetStringArray(data, cnt)
}

// Number of items on the OpenDSS control Queue
func (ctrlqueue *ICtrlQueue) QueueSize() (int32, error) {
	return (int32)(C.ctx_CtrlQueue_Get_QueueSize(ctrlqueue.ctxPtr)), ctrlqueue.ctx.DSSError()
}

func (ctrlqueue *ICtrlQueue) Set_Action(value int32) error {
	C.ctx_CtrlQueue_Set_Action(ctrlqueue.ctxPtr, (C.int32_t)(value))
	return ctrlqueue.ctx.DSSError()
}

type IDSSElement struct {
	ICommonData

	Properties IDSSProperty
}

func (dsselement *IDSSElement) Init(ctx *DSSContextPtrs) {
	dsselement.InitCommon(ctx)
	dsselement.Properties.Init(ctx)
}

// Array of strings containing the names of all properties for the active DSS object.
func (dsselement *IDSSElement) AllPropertyNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_DSSElement_Get_AllPropertyNames(dsselement.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return dsselement.ctx.GetStringArray(data, cnt)
}

// Full Name of Active DSS Object (general element or circuit element).
func (dsselement *IDSSElement) Name() (string, error) {
	return C.GoString(C.ctx_DSSElement_Get_Name(dsselement.ctxPtr)), dsselement.ctx.DSSError()
}

// Number of Properties for the active DSS object.
func (dsselement *IDSSElement) NumProperties() (int32, error) {
	return (int32)(C.ctx_DSSElement_Get_NumProperties(dsselement.ctxPtr)), dsselement.ctx.DSSError()
}

// Returns the properties of the active DSS object as a JSON-encoded string.
//
// The `options` parameter contains bit-flags to toggle specific features.
// See `Obj_ToJSON` (C-API) for more.
//
// (API Extension)
func (dsselement *IDSSElement) ToJSON(options int32) (string, error) {
	return C.GoString(C.ctx_DSSElement_ToJSON(dsselement.ctxPtr, (C.int32_t)(options))), dsselement.ctx.DSSError()
}

type IDSSProgress struct {
	ICommonData
}

func (dssprogress *IDSSProgress) Init(ctx *DSSContextPtrs) {
	dssprogress.InitCommon(ctx)
}

func (dssprogress *IDSSProgress) Close() error {
	C.ctx_DSSProgress_Close(dssprogress.ctxPtr)
	return dssprogress.ctx.DSSError()
}

func (dssprogress *IDSSProgress) Show() error {
	C.ctx_DSSProgress_Show(dssprogress.ctxPtr)
	return dssprogress.ctx.DSSError()
}

func (dssprogress *IDSSProgress) Set_Caption(value string) error {
	value_c := C.CString(value)
	C.ctx_DSSProgress_Set_Caption(dssprogress.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return dssprogress.ctx.DSSError()
}

func (dssprogress *IDSSProgress) Set_PctProgress(value int32) error {
	C.ctx_DSSProgress_Set_PctProgress(dssprogress.ctxPtr, (C.int32_t)(value))
	return dssprogress.ctx.DSSError()
}

type IDSSProperty struct {
	ICommonData
}

func (dssproperty *IDSSProperty) Init(ctx *DSSContextPtrs) {
	dssproperty.InitCommon(ctx)
}

func (dssproperty *IDSSProperty) Set_idx(key int32) error {
	C.ctx_DSSProperty_Set_Index(dssproperty.ctxPtr, (C.int32_t)(key))
	return dssproperty.ctx.DSSError()
}

func (dssproperty *IDSSProperty) Set_Name(key string) error {
	key_c := C.CString(key)
	C.ctx_DSSProperty_Set_Name(dssproperty.ctxPtr, key_c)
	C.free(unsafe.Pointer(key_c))
	return dssproperty.ctx.DSSError()
}

// Description of the property.
func (dssproperty *IDSSProperty) Description() (string, error) {
	return C.GoString(C.ctx_DSSProperty_Get_Description(dssproperty.ctxPtr)), dssproperty.ctx.DSSError()
}

// Name of Property
func (dssproperty *IDSSProperty) Name() (string, error) {
	return C.GoString(C.ctx_DSSProperty_Get_Name(dssproperty.ctxPtr)), dssproperty.ctx.DSSError()
}

func (dssproperty *IDSSProperty) Get_Val() (string, error) {
	return C.GoString(C.ctx_DSSProperty_Get_Val(dssproperty.ctxPtr)), dssproperty.ctx.DSSError()
}

func (dssproperty *IDSSProperty) Set_Val(value string) error {
	value_c := C.CString(value)
	C.ctx_DSSProperty_Set_Val(dssproperty.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return dssproperty.ctx.DSSError()
}

type IDSS_Executive struct {
	ICommonData
}

func (dss_executive *IDSS_Executive) Init(ctx *DSSContextPtrs) {
	dss_executive.InitCommon(ctx)
}

// Get i-th command
func (dss_executive *IDSS_Executive) Command(i int32) (string, error) {
	return C.GoString(C.ctx_DSS_Executive_Get_Command(dss_executive.ctxPtr, (C.int32_t)(i))), dss_executive.ctx.DSSError()
}

// Get help string for i-th command
func (dss_executive *IDSS_Executive) CommandHelp(i int32) (string, error) {
	return C.GoString(C.ctx_DSS_Executive_Get_CommandHelp(dss_executive.ctxPtr, (C.int32_t)(i))), dss_executive.ctx.DSSError()
}

// Get i-th option
func (dss_executive *IDSS_Executive) Option(i int32) (string, error) {
	return C.GoString(C.ctx_DSS_Executive_Get_Option(dss_executive.ctxPtr, (C.int32_t)(i))), dss_executive.ctx.DSSError()
}

// Get help string for i-th option
func (dss_executive *IDSS_Executive) OptionHelp(i int32) (string, error) {
	return C.GoString(C.ctx_DSS_Executive_Get_OptionHelp(dss_executive.ctxPtr, (C.int32_t)(i))), dss_executive.ctx.DSSError()
}

// Get present value of i-th option
func (dss_executive *IDSS_Executive) OptionValue(i int32) (string, error) {
	return C.GoString(C.ctx_DSS_Executive_Get_OptionValue(dss_executive.ctxPtr, (C.int32_t)(i))), dss_executive.ctx.DSSError()
}

// Number of DSS Executive Commands
func (dss_executive *IDSS_Executive) NumCommands() (int32, error) {
	return (int32)(C.ctx_DSS_Executive_Get_NumCommands(dss_executive.ctxPtr)), dss_executive.ctx.DSSError()
}

// Number of DSS Executive Options
func (dss_executive *IDSS_Executive) NumOptions() (int32, error) {
	return (int32)(C.ctx_DSS_Executive_Get_NumOptions(dss_executive.ctxPtr)), dss_executive.ctx.DSSError()
}

type IError struct {
	ICommonData
}

func (error *IError) Init(ctx *DSSContextPtrs) {
	error.InitCommon(ctx)
}

// Description of error for last operation
func (error *IError) Description() (string, error) {
	return C.GoString(C.ctx_Error_Get_Description(error.ctxPtr)), error.ctx.DSSError()
}

// Error Number (returns current value and then resets to zero)
func (error *IError) Number() (int32, error) {
	return (int32)(C.ctx_Error_Get_Number(error.ctxPtr)), error.ctx.DSSError()
}

// EarlyAbort controls whether all errors halts the DSS script processing (Compile/Redirect), defaults to True.
//
// (API Extension)
func (error *IError) Get_EarlyAbort() (bool, error) {
	return (C.ctx_Error_Get_EarlyAbort(error.ctxPtr) != 0), error.ctx.DSSError()
}

func (error *IError) Set_EarlyAbort(value bool) error {
	C.ctx_Error_Set_EarlyAbort(error.ctxPtr, ToUint16(value))
	return error.ctx.DSSError()
}

// Controls whether the extended error mechanism is used. Defaults to True.
//
// Extended errors are errors derived from checks across the API to ensure
// a valid state. Although many of these checks are already present in the
// original/official COM interface, the checks do not produce any error
// message. An error value can be returned by a function but this value
// can, for many of the functions, be a valid value. As such, the user
// has no means to detect an invalid API call.
//
// Extended errors use the Error interface to provide a more clear message
// and should help users, especially new users, to find usage issues earlier.
//
// At Go level, the errors from the Error interface at mapped to Golang errors
// for most function calls, hence the user does not need to use the DSS Error
// interface directly.
//
// The current default state is ON. For compatibility, the user can turn it
// off to restore the previous behavior.
//
// (API Extension)
func (error *IError) Get_ExtendedErrors() (bool, error) {
	return (C.ctx_Error_Get_ExtendedErrors(error.ctxPtr) != 0), error.ctx.DSSError()
}

func (error *IError) Set_ExtendedErrors(value bool) error {
	C.ctx_Error_Set_ExtendedErrors(error.ctxPtr, ToUint16(value))
	return error.ctx.DSSError()
}

type IFuses struct {
	ICommonData
}

func (fuses *IFuses) Init(ctx *DSSContextPtrs) {
	fuses.InitCommon(ctx)
}

// Array of strings with all Fuse names in the circuit.
func (fuses *IFuses) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Fuses_Get_AllNames(fuses.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return fuses.ctx.GetStringArray(data, cnt)
}

// Number of Fuse objects in active circuit.
func (fuses *IFuses) Count() (int32, error) {
	return (int32)(C.ctx_Fuses_Get_Count(fuses.ctxPtr)), fuses.ctx.DSSError()
}

// Sets the first Fuse active. Returns 0 if no more.
func (fuses *IFuses) First() (int32, error) {
	return (int32)(C.ctx_Fuses_Get_First(fuses.ctxPtr)), fuses.ctx.DSSError()
}

// Sets the active Fuse by Name.
func (fuses *IFuses) Get_Name() (string, error) {
	result := C.GoString(C.ctx_Fuses_Get_Name(fuses.ctxPtr))
	return result, fuses.ctx.DSSError()
}

// Gets the name of the active Fuse.
func (fuses *IFuses) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_Fuses_Set_Name(fuses.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return fuses.ctx.DSSError()
}

// Sets the next Fuse active. Returns 0 if no more.
func (fuses *IFuses) Next() (int32, error) {
	return (int32)(C.ctx_Fuses_Get_Next(fuses.ctxPtr)), fuses.ctx.DSSError()
}

// Get the index of the active Fuse; index is 1-based: 1..count
func (fuses *IFuses) Get_idx() (int32, error) {
	return (int32)(C.ctx_Fuses_Get_idx(fuses.ctxPtr)), fuses.ctx.DSSError()
}

// Set the active Fuse by index; index is 1-based: 1..count
func (fuses *IFuses) Set_idx(value int32) error {
	C.ctx_Fuses_Set_idx(fuses.ctxPtr, (C.int32_t)(value))
	return fuses.ctx.DSSError()
}

// Close all phases of the fuse.
func (fuses *IFuses) Close() error {
	C.ctx_Fuses_Close(fuses.ctxPtr)
	return fuses.ctx.DSSError()
}

// Current state of the fuses. TRUE if any fuse on any phase is blown. Else FALSE.
func (fuses *IFuses) IsBlown() (bool, error) {
	return (C.ctx_Fuses_IsBlown(fuses.ctxPtr) != 0), fuses.ctx.DSSError()
}

// Manual opening of all phases of the fuse.
func (fuses *IFuses) Open() error {
	C.ctx_Fuses_Open(fuses.ctxPtr)
	return fuses.ctx.DSSError()
}

// Reset fuse to normal state.
func (fuses *IFuses) Reset() error {
	C.ctx_Fuses_Reset(fuses.ctxPtr)
	return fuses.ctx.DSSError()
}

// A fixed delay time in seconds added to the fuse blowing time determined by the TCC curve. Default is 0.
// This represents a fuse clear or other delay.
func (fuses *IFuses) Get_Delay() (float64, error) {
	return (float64)(C.ctx_Fuses_Get_Delay(fuses.ctxPtr)), fuses.ctx.DSSError()
}

func (fuses *IFuses) Set_Delay(value float64) error {
	C.ctx_Fuses_Set_Delay(fuses.ctxPtr, (C.double)(value))
	return fuses.ctx.DSSError()
}

// Full name of the circuit element to which the fuse is connected.
func (fuses *IFuses) Get_MonitoredObj() (string, error) {
	return C.GoString(C.ctx_Fuses_Get_MonitoredObj(fuses.ctxPtr)), fuses.ctx.DSSError()
}

func (fuses *IFuses) Set_MonitoredObj(value string) error {
	value_c := C.CString(value)
	C.ctx_Fuses_Set_MonitoredObj(fuses.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return fuses.ctx.DSSError()
}

// Terminal number to which the fuse is connected.
func (fuses *IFuses) Get_MonitoredTerm() (int32, error) {
	return (int32)(C.ctx_Fuses_Get_MonitoredTerm(fuses.ctxPtr)), fuses.ctx.DSSError()
}

func (fuses *IFuses) Set_MonitoredTerm(value int32) error {
	C.ctx_Fuses_Set_MonitoredTerm(fuses.ctxPtr, (C.int32_t)(value))
	return fuses.ctx.DSSError()
}

// Number of phases, this fuse.
func (fuses *IFuses) NumPhases() (int32, error) {
	return (int32)(C.ctx_Fuses_Get_NumPhases(fuses.ctxPtr)), fuses.ctx.DSSError()
}

// Multiplier or actual amps for the TCCcurve object. Defaults to 1.0.
// Multiply current values of TCC curve by this to get actual amps.
func (fuses *IFuses) Get_RatedCurrent() (float64, error) {
	return (float64)(C.ctx_Fuses_Get_RatedCurrent(fuses.ctxPtr)), fuses.ctx.DSSError()
}

func (fuses *IFuses) Set_RatedCurrent(value float64) error {
	C.ctx_Fuses_Set_RatedCurrent(fuses.ctxPtr, (C.double)(value))
	return fuses.ctx.DSSError()
}

// Full name of the circuit element switch that the fuse controls.
// Defaults to the MonitoredObj.
func (fuses *IFuses) Get_SwitchedObj() (string, error) {
	return C.GoString(C.ctx_Fuses_Get_SwitchedObj(fuses.ctxPtr)), fuses.ctx.DSSError()
}

func (fuses *IFuses) Set_SwitchedObj(value string) error {
	value_c := C.CString(value)
	C.ctx_Fuses_Set_SwitchedObj(fuses.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return fuses.ctx.DSSError()
}

// Number of the terminal of the controlled element containing the switch controlled by the fuse.
func (fuses *IFuses) Get_SwitchedTerm() (int32, error) {
	return (int32)(C.ctx_Fuses_Get_SwitchedTerm(fuses.ctxPtr)), fuses.ctx.DSSError()
}

func (fuses *IFuses) Set_SwitchedTerm(value int32) error {
	C.ctx_Fuses_Set_SwitchedTerm(fuses.ctxPtr, (C.int32_t)(value))
	return fuses.ctx.DSSError()
}

// Name of the TCCcurve object that determines fuse blowing.
func (fuses *IFuses) Get_TCCcurve() (string, error) {
	return C.GoString(C.ctx_Fuses_Get_TCCcurve(fuses.ctxPtr)), fuses.ctx.DSSError()
}

func (fuses *IFuses) Set_TCCcurve(value string) error {
	value_c := C.CString(value)
	C.ctx_Fuses_Set_TCCcurve(fuses.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return fuses.ctx.DSSError()
}

// Array of strings indicating the state of each phase of the fuse.
func (fuses *IFuses) Get_State() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Fuses_Get_State(fuses.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return fuses.ctx.GetStringArray(data, cnt)
}

func (fuses *IFuses) Set_State(value []string) error {
	value_c := fuses.ctx.PrepareStringArray(value)
	defer fuses.ctx.FreeStringArray(value_c, len(value))
	C.ctx_Fuses_Set_State(fuses.ctxPtr, value_c, (C.int32_t)(len(value)))
	return fuses.ctx.DSSError()
}

// Array of strings indicating the normal state of each phase of the fuse.
func (fuses *IFuses) Get_NormalState() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Fuses_Get_NormalState(fuses.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return fuses.ctx.GetStringArray(data, cnt)
}

func (fuses *IFuses) Set_NormalState(value []string) error {
	value_c := fuses.ctx.PrepareStringArray(value)
	defer fuses.ctx.FreeStringArray(value_c, len(value))
	C.ctx_Fuses_Set_NormalState(fuses.ctxPtr, value_c, (C.int32_t)(len(value)))
	return fuses.ctx.DSSError()
}

type IISources struct {
	ICommonData
}

func (isources *IISources) Init(ctx *DSSContextPtrs) {
	isources.InitCommon(ctx)
}

// Array of strings with all ISource names in the circuit.
func (isources *IISources) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_ISources_Get_AllNames(isources.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return isources.ctx.GetStringArray(data, cnt)
}

// Number of ISource objects in active circuit.
func (isources *IISources) Count() (int32, error) {
	return (int32)(C.ctx_ISources_Get_Count(isources.ctxPtr)), isources.ctx.DSSError()
}

// Sets the first ISource active. Returns 0 if no more.
func (isources *IISources) First() (int32, error) {
	return (int32)(C.ctx_ISources_Get_First(isources.ctxPtr)), isources.ctx.DSSError()
}

// Sets the active ISource by Name.
func (isources *IISources) Get_Name() (string, error) {
	result := C.GoString(C.ctx_ISources_Get_Name(isources.ctxPtr))
	return result, isources.ctx.DSSError()
}

// Gets the name of the active ISource.
func (isources *IISources) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_ISources_Set_Name(isources.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return isources.ctx.DSSError()
}

// Sets the next ISource active. Returns 0 if no more.
func (isources *IISources) Next() (int32, error) {
	return (int32)(C.ctx_ISources_Get_Next(isources.ctxPtr)), isources.ctx.DSSError()
}

// Get the index of the active ISource; index is 1-based: 1..count
func (isources *IISources) Get_idx() (int32, error) {
	return (int32)(C.ctx_ISources_Get_idx(isources.ctxPtr)), isources.ctx.DSSError()
}

// Set the active ISource by index; index is 1-based: 1..count
func (isources *IISources) Set_idx(value int32) error {
	C.ctx_ISources_Set_idx(isources.ctxPtr, (C.int32_t)(value))
	return isources.ctx.DSSError()
}

// Magnitude of the ISource in amps
func (isources *IISources) Get_Amps() (float64, error) {
	return (float64)(C.ctx_ISources_Get_Amps(isources.ctxPtr)), isources.ctx.DSSError()
}

func (isources *IISources) Set_Amps(value float64) error {
	C.ctx_ISources_Set_Amps(isources.ctxPtr, (C.double)(value))
	return isources.ctx.DSSError()
}

// Phase angle for ISource, degrees
func (isources *IISources) Get_AngleDeg() (float64, error) {
	return (float64)(C.ctx_ISources_Get_AngleDeg(isources.ctxPtr)), isources.ctx.DSSError()
}

func (isources *IISources) Set_AngleDeg(value float64) error {
	C.ctx_ISources_Set_AngleDeg(isources.ctxPtr, (C.double)(value))
	return isources.ctx.DSSError()
}

// The present frequency of the ISource, Hz
func (isources *IISources) Get_Frequency() (float64, error) {
	return (float64)(C.ctx_ISources_Get_Frequency(isources.ctxPtr)), isources.ctx.DSSError()
}

func (isources *IISources) Set_Frequency(value float64) error {
	C.ctx_ISources_Set_Frequency(isources.ctxPtr, (C.double)(value))
	return isources.ctx.DSSError()
}

type ILineCodes struct {
	ICommonData
}

func (linecodes *ILineCodes) Init(ctx *DSSContextPtrs) {
	linecodes.InitCommon(ctx)
}

// Array of strings with all LineCode names in the circuit.
func (linecodes *ILineCodes) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_LineCodes_Get_AllNames(linecodes.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return linecodes.ctx.GetStringArray(data, cnt)
}

// Number of LineCode objects in active circuit.
func (linecodes *ILineCodes) Count() (int32, error) {
	return (int32)(C.ctx_LineCodes_Get_Count(linecodes.ctxPtr)), linecodes.ctx.DSSError()
}

// Sets the first LineCode active. Returns 0 if no more.
func (linecodes *ILineCodes) First() (int32, error) {
	return (int32)(C.ctx_LineCodes_Get_First(linecodes.ctxPtr)), linecodes.ctx.DSSError()
}

// Sets the active LineCode by Name.
func (linecodes *ILineCodes) Get_Name() (string, error) {
	result := C.GoString(C.ctx_LineCodes_Get_Name(linecodes.ctxPtr))
	return result, linecodes.ctx.DSSError()
}

// Gets the name of the active LineCode.
func (linecodes *ILineCodes) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_LineCodes_Set_Name(linecodes.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return linecodes.ctx.DSSError()
}

// Sets the next LineCode active. Returns 0 if no more.
func (linecodes *ILineCodes) Next() (int32, error) {
	return (int32)(C.ctx_LineCodes_Get_Next(linecodes.ctxPtr)), linecodes.ctx.DSSError()
}

// Get the index of the active LineCode; index is 1-based: 1..count
func (linecodes *ILineCodes) Get_idx() (int32, error) {
	return (int32)(C.ctx_LineCodes_Get_idx(linecodes.ctxPtr)), linecodes.ctx.DSSError()
}

// Set the active LineCode by index; index is 1-based: 1..count
func (linecodes *ILineCodes) Set_idx(value int32) error {
	C.ctx_LineCodes_Set_idx(linecodes.ctxPtr, (C.int32_t)(value))
	return linecodes.ctx.DSSError()
}

// Zero-sequence capacitance, nF per unit length
func (linecodes *ILineCodes) Get_C0() (float64, error) {
	return (float64)(C.ctx_LineCodes_Get_C0(linecodes.ctxPtr)), linecodes.ctx.DSSError()
}

func (linecodes *ILineCodes) Set_C0(value float64) error {
	C.ctx_LineCodes_Set_C0(linecodes.ctxPtr, (C.double)(value))
	return linecodes.ctx.DSSError()
}

// Positive-sequence capacitance, nF per unit length
func (linecodes *ILineCodes) Get_C1() (float64, error) {
	return (float64)(C.ctx_LineCodes_Get_C1(linecodes.ctxPtr)), linecodes.ctx.DSSError()
}

func (linecodes *ILineCodes) Set_C1(value float64) error {
	C.ctx_LineCodes_Set_C1(linecodes.ctxPtr, (C.double)(value))
	return linecodes.ctx.DSSError()
}

// Capacitance matrix, nF per unit length
func (linecodes *ILineCodes) Get_Cmatrix() ([]float64, error) {
	C.ctx_LineCodes_Get_Cmatrix_GR(linecodes.ctxPtr)
	return linecodes.ctx.GetFloat64ArrayGR()
}

func (linecodes *ILineCodes) Set_Cmatrix(value []float64) error {
	C.ctx_LineCodes_Set_Cmatrix(linecodes.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return linecodes.ctx.DSSError()
}

// Emergency ampere rating
func (linecodes *ILineCodes) Get_EmergAmps() (float64, error) {
	return (float64)(C.ctx_LineCodes_Get_EmergAmps(linecodes.ctxPtr)), linecodes.ctx.DSSError()
}

func (linecodes *ILineCodes) Set_EmergAmps(value float64) error {
	C.ctx_LineCodes_Set_EmergAmps(linecodes.ctxPtr, (C.double)(value))
	return linecodes.ctx.DSSError()
}

// Flag denoting whether impedance data were entered in symmetrical components
func (linecodes *ILineCodes) IsZ1Z0() (bool, error) {
	return (C.ctx_LineCodes_Get_IsZ1Z0(linecodes.ctxPtr) != 0), linecodes.ctx.DSSError()
}

// Normal Ampere rating
func (linecodes *ILineCodes) Get_NormAmps() (float64, error) {
	return (float64)(C.ctx_LineCodes_Get_NormAmps(linecodes.ctxPtr)), linecodes.ctx.DSSError()
}

func (linecodes *ILineCodes) Set_NormAmps(value float64) error {
	C.ctx_LineCodes_Set_NormAmps(linecodes.ctxPtr, (C.double)(value))
	return linecodes.ctx.DSSError()
}

// Number of Phases
func (linecodes *ILineCodes) Get_Phases() (int32, error) {
	return (int32)(C.ctx_LineCodes_Get_Phases(linecodes.ctxPtr)), linecodes.ctx.DSSError()
}

func (linecodes *ILineCodes) Set_Phases(value int32) error {
	C.ctx_LineCodes_Set_Phases(linecodes.ctxPtr, (C.int32_t)(value))
	return linecodes.ctx.DSSError()
}

// Zero-Sequence Resistance, ohms per unit length
func (linecodes *ILineCodes) Get_R0() (float64, error) {
	return (float64)(C.ctx_LineCodes_Get_R0(linecodes.ctxPtr)), linecodes.ctx.DSSError()
}

func (linecodes *ILineCodes) Set_R0(value float64) error {
	C.ctx_LineCodes_Set_R0(linecodes.ctxPtr, (C.double)(value))
	return linecodes.ctx.DSSError()
}

// Positive-sequence resistance ohms per unit length
func (linecodes *ILineCodes) Get_R1() (float64, error) {
	return (float64)(C.ctx_LineCodes_Get_R1(linecodes.ctxPtr)), linecodes.ctx.DSSError()
}

func (linecodes *ILineCodes) Set_R1(value float64) error {
	C.ctx_LineCodes_Set_R1(linecodes.ctxPtr, (C.double)(value))
	return linecodes.ctx.DSSError()
}

// Resistance matrix, ohms per unit length
func (linecodes *ILineCodes) Get_Rmatrix() ([]float64, error) {
	C.ctx_LineCodes_Get_Rmatrix_GR(linecodes.ctxPtr)
	return linecodes.ctx.GetFloat64ArrayGR()
}

func (linecodes *ILineCodes) Set_Rmatrix(value []float64) error {
	C.ctx_LineCodes_Set_Rmatrix(linecodes.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return linecodes.ctx.DSSError()
}

func (linecodes *ILineCodes) Get_Units() (LineUnits, error) {
	return (LineUnits)(C.ctx_LineCodes_Get_Units(linecodes.ctxPtr)), linecodes.ctx.DSSError()
}

func (linecodes *ILineCodes) Set_Units(value LineUnits) error {
	C.ctx_LineCodes_Set_Units(linecodes.ctxPtr, (C.int32_t)(value))
	return linecodes.ctx.DSSError()
}

// Zero Sequence Reactance, Ohms per unit length
func (linecodes *ILineCodes) Get_X0() (float64, error) {
	return (float64)(C.ctx_LineCodes_Get_X0(linecodes.ctxPtr)), linecodes.ctx.DSSError()
}

func (linecodes *ILineCodes) Set_X0(value float64) error {
	C.ctx_LineCodes_Set_X0(linecodes.ctxPtr, (C.double)(value))
	return linecodes.ctx.DSSError()
}

// Posiive-sequence reactance, ohms per unit length
func (linecodes *ILineCodes) Get_X1() (float64, error) {
	return (float64)(C.ctx_LineCodes_Get_X1(linecodes.ctxPtr)), linecodes.ctx.DSSError()
}

func (linecodes *ILineCodes) Set_X1(value float64) error {
	C.ctx_LineCodes_Set_X1(linecodes.ctxPtr, (C.double)(value))
	return linecodes.ctx.DSSError()
}

// Reactance matrix, ohms per unit length
func (linecodes *ILineCodes) Get_Xmatrix() ([]float64, error) {
	C.ctx_LineCodes_Get_Xmatrix_GR(linecodes.ctxPtr)
	return linecodes.ctx.GetFloat64ArrayGR()
}

func (linecodes *ILineCodes) Set_Xmatrix(value []float64) error {
	C.ctx_LineCodes_Set_Xmatrix(linecodes.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return linecodes.ctx.DSSError()
}

type IMonitors struct {
	ICommonData
}

func (monitors *IMonitors) Init(ctx *DSSContextPtrs) {
	monitors.InitCommon(ctx)
}

// TODO: Implement AsMatrix someday

// Array of float64 for the specified channel (usage: MyArray = DSSMonitor.Channel(i)).
// A Save or SaveAll should be executed first. Done automatically by most standard solution modes.
// Channels start at index 1.
func (monitors *IMonitors) Channel(index int32) ([]float64, error) {
	//TODO: use the better implementation
	C.ctx_Monitors_Get_Channel_GR(monitors.ctxPtr, (C.int32_t)(index))
	return monitors.ctx.GetFloat64ArrayGR()
}

// Array of strings with all Monitor names in the circuit.
func (monitors *IMonitors) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Monitors_Get_AllNames(monitors.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return monitors.ctx.GetStringArray(data, cnt)
}

// Number of Monitor objects in active circuit.
func (monitors *IMonitors) Count() (int32, error) {
	return (int32)(C.ctx_Monitors_Get_Count(monitors.ctxPtr)), monitors.ctx.DSSError()
}

// Sets the first Monitor active. Returns 0 if no more.
func (monitors *IMonitors) First() (int32, error) {
	return (int32)(C.ctx_Monitors_Get_First(monitors.ctxPtr)), monitors.ctx.DSSError()
}

// Sets the active Monitor by Name.
func (monitors *IMonitors) Get_Name() (string, error) {
	result := C.GoString(C.ctx_Monitors_Get_Name(monitors.ctxPtr))
	return result, monitors.ctx.DSSError()
}

// Gets the name of the active Monitor.
func (monitors *IMonitors) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_Monitors_Set_Name(monitors.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return monitors.ctx.DSSError()
}

// Sets the next Monitor active. Returns 0 if no more.
func (monitors *IMonitors) Next() (int32, error) {
	return (int32)(C.ctx_Monitors_Get_Next(monitors.ctxPtr)), monitors.ctx.DSSError()
}

// Get the index of the active Monitor; index is 1-based: 1..count
func (monitors *IMonitors) Get_idx() (int32, error) {
	return (int32)(C.ctx_Monitors_Get_idx(monitors.ctxPtr)), monitors.ctx.DSSError()
}

// Set the active Monitor by index; index is 1-based: 1..count
func (monitors *IMonitors) Set_idx(value int32) error {
	C.ctx_Monitors_Set_idx(monitors.ctxPtr, (C.int32_t)(value))
	return monitors.ctx.DSSError()
}

func (monitors *IMonitors) Process() error {
	C.ctx_Monitors_Process(monitors.ctxPtr)
	return monitors.ctx.DSSError()
}

func (monitors *IMonitors) ProcessAll() error {
	C.ctx_Monitors_ProcessAll(monitors.ctxPtr)
	return monitors.ctx.DSSError()
}

func (monitors *IMonitors) Reset() error {
	C.ctx_Monitors_Reset(monitors.ctxPtr)
	return monitors.ctx.DSSError()
}

func (monitors *IMonitors) ResetAll() error {
	C.ctx_Monitors_ResetAll(monitors.ctxPtr)
	return monitors.ctx.DSSError()
}

func (monitors *IMonitors) Sample() error {
	C.ctx_Monitors_Sample(monitors.ctxPtr)
	return monitors.ctx.DSSError()
}

func (monitors *IMonitors) SampleAll() error {
	C.ctx_Monitors_SampleAll(monitors.ctxPtr)
	return monitors.ctx.DSSError()
}

func (monitors *IMonitors) Save() error {
	C.ctx_Monitors_Save(monitors.ctxPtr)
	return monitors.ctx.DSSError()
}

func (monitors *IMonitors) SaveAll() error {
	C.ctx_Monitors_SaveAll(monitors.ctxPtr)
	return monitors.ctx.DSSError()
}

func (monitors *IMonitors) Show() error {
	C.ctx_Monitors_Show(monitors.ctxPtr)
	return monitors.ctx.DSSError()
}

// Byte Array containing monitor stream values. Make sure a "save" is done first (standard solution modes do this automatically)
func (monitors *IMonitors) ByteStream() ([]byte, error) {
	C.ctx_Monitors_Get_ByteStream_GR(monitors.ctxPtr)
	return monitors.ctx.GetUInt8ArrayGR()
}

// Full object name of element being monitored.
func (monitors *IMonitors) Get_Element() (string, error) {
	return C.GoString(C.ctx_Monitors_Get_Element(monitors.ctxPtr)), monitors.ctx.DSSError()
}

func (monitors *IMonitors) Set_Element(value string) error {
	value_c := C.CString(value)
	C.ctx_Monitors_Set_Element(monitors.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return monitors.ctx.DSSError()
}

// Name of CSV file associated with active Monitor.
func (monitors *IMonitors) FileName() (string, error) {
	return C.GoString(C.ctx_Monitors_Get_FileName(monitors.ctxPtr)), monitors.ctx.DSSError()
}

// Monitor File Version (integer)
func (monitors *IMonitors) FileVersion() (int32, error) {
	return (int32)(C.ctx_Monitors_Get_FileVersion(monitors.ctxPtr)), monitors.ctx.DSSError()
}

// Header string;  Array of strings containing Channel names
func (monitors *IMonitors) Header() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Monitors_Get_Header(monitors.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return monitors.ctx.GetStringArray(data, cnt)
}

// Set Monitor mode (bitmask integer - see DSS Help)
func (monitors *IMonitors) Get_Mode() (int32, error) {
	return (int32)(C.ctx_Monitors_Get_Mode(monitors.ctxPtr)), monitors.ctx.DSSError()
}

func (monitors *IMonitors) Set_Mode(value int32) error {
	C.ctx_Monitors_Set_Mode(monitors.ctxPtr, (C.int32_t)(value))
	return monitors.ctx.DSSError()
}

// Number of Channels in the active Monitor
func (monitors *IMonitors) NumChannels() (int32, error) {
	return (int32)(C.ctx_Monitors_Get_NumChannels(monitors.ctxPtr)), monitors.ctx.DSSError()
}

// Size of each record in ByteStream (Integer). Same as NumChannels.
func (monitors *IMonitors) RecordSize() (int32, error) {
	return (int32)(C.ctx_Monitors_Get_RecordSize(monitors.ctxPtr)), monitors.ctx.DSSError()
}

// Number of Samples in Monitor at Present
func (monitors *IMonitors) SampleCount() (int32, error) {
	return (int32)(C.ctx_Monitors_Get_SampleCount(monitors.ctxPtr)), monitors.ctx.DSSError()
}

// Terminal number of element being monitored.
func (monitors *IMonitors) Get_Terminal() (int32, error) {
	return (int32)(C.ctx_Monitors_Get_Terminal(monitors.ctxPtr)), monitors.ctx.DSSError()
}

func (monitors *IMonitors) Set_Terminal(value int32) error {
	C.ctx_Monitors_Set_Terminal(monitors.ctxPtr, (C.int32_t)(value))
	return monitors.ctx.DSSError()
}

// Array of doubles containing frequency values for harmonics mode solutions; Empty for time mode solutions (use dblHour)
func (monitors *IMonitors) DblFreq() ([]float64, error) {
	C.ctx_Monitors_Get_dblFreq_GR(monitors.ctxPtr)
	return monitors.ctx.GetFloat64ArrayGR()
}

// Array of doubles containing time value in hours for time-sampled monitor values; Empty if frequency-sampled values for harmonics solution (see dblFreq)
func (monitors *IMonitors) DblHour() ([]float64, error) {
	C.ctx_Monitors_Get_dblHour_GR(monitors.ctxPtr)
	return monitors.ctx.GetFloat64ArrayGR()
}

type IParser struct {
	ICommonData
}

func (parser *IParser) Init(ctx *DSSContextPtrs) {
	parser.InitCommon(ctx)
}

// Use this property to parse a Matrix token in OpenDSS format.  Returns square matrix of order specified. Order same as default Fortran order: column by column.
func (parser *IParser) Matrix(ExpectedOrder int32) ([]float64, error) {
	C.ctx_Parser_Get_Matrix_GR(parser.ctxPtr, (C.int32_t)(ExpectedOrder))
	return parser.ctx.GetFloat64ArrayGR()
}

// Use this property to parse a matrix token specified in lower triangle form. Symmetry is forced.
func (parser *IParser) SymMatrix(ExpectedOrder int32) ([]float64, error) {
	C.ctx_Parser_Get_SymMatrix_GR(parser.ctxPtr, (C.int32_t)(ExpectedOrder))
	return parser.ctx.GetFloat64ArrayGR()
}

// Returns token as array of doubles. For parsing quoted array syntax.
func (parser *IParser) Vector(ExpectedSize int32) ([]float64, error) {
	C.ctx_Parser_Get_Vector_GR(parser.ctxPtr, (C.int32_t)(ExpectedSize))
	return parser.ctx.GetFloat64ArrayGR()
}

func (parser *IParser) ResetDelimiters() error {
	C.ctx_Parser_ResetDelimiters(parser.ctxPtr)
	return parser.ctx.DSSError()
}

// Default is FALSE. If TRUE parser automatically advances to next token after DblValue, IntValue, or StrValue. Simpler when you don't need to check for parameter names.
func (parser *IParser) Get_AutoIncrement() (bool, error) {
	return (C.ctx_Parser_Get_AutoIncrement(parser.ctxPtr) != 0), parser.ctx.DSSError()
}

func (parser *IParser) Set_AutoIncrement(value bool) error {
	C.ctx_Parser_Set_AutoIncrement(parser.ctxPtr, ToUint16(value))
	return parser.ctx.DSSError()
}

// Get/Set String containing the the characters for Quoting in OpenDSS scripts. Matching pairs defined in EndQuote. Default is "'([{.
func (parser *IParser) Get_BeginQuote() (string, error) {
	return C.GoString(C.ctx_Parser_Get_BeginQuote(parser.ctxPtr)), parser.ctx.DSSError()
}

func (parser *IParser) Set_BeginQuote(value string) error {
	value_c := C.CString(value)
	C.ctx_Parser_Set_BeginQuote(parser.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return parser.ctx.DSSError()
}

// String to be parsed. Loading this string resets the Parser to the beginning of the line. Then parse off the tokens in sequence.
func (parser *IParser) Get_CmdString() (string, error) {
	return C.GoString(C.ctx_Parser_Get_CmdString(parser.ctxPtr)), parser.ctx.DSSError()
}

func (parser *IParser) Set_CmdString(value string) error {
	value_c := C.CString(value)
	C.ctx_Parser_Set_CmdString(parser.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return parser.ctx.DSSError()
}

// Return next parameter as a double.
func (parser *IParser) DblValue() (float64, error) {
	return (float64)(C.ctx_Parser_Get_DblValue(parser.ctxPtr)), parser.ctx.DSSError()
}

// String defining hard delimiters used to separate token on the command string. Default is , and =. The = separates token name from token value. These override whitesspace to separate tokens.
func (parser *IParser) Get_Delimiters() (string, error) {
	return C.GoString(C.ctx_Parser_Get_Delimiters(parser.ctxPtr)), parser.ctx.DSSError()
}

func (parser *IParser) Set_Delimiters(value string) error {
	value_c := C.CString(value)
	C.ctx_Parser_Set_Delimiters(parser.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return parser.ctx.DSSError()
}

// String containing characters, in order, that match the beginning quote characters in BeginQuote. Default is "')]}
func (parser *IParser) Get_EndQuote() (string, error) {
	return C.GoString(C.ctx_Parser_Get_EndQuote(parser.ctxPtr)), parser.ctx.DSSError()
}

func (parser *IParser) Set_EndQuote(value string) error {
	value_c := C.CString(value)
	C.ctx_Parser_Set_EndQuote(parser.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return parser.ctx.DSSError()
}

// Return next parameter as a long integer.
func (parser *IParser) IntValue() (int32, error) {
	return (int32)(C.ctx_Parser_Get_IntValue(parser.ctxPtr)), parser.ctx.DSSError()
}

// Get next token and return tag name (before = sign) if any. See AutoIncrement.
func (parser *IParser) NextParam() (string, error) {
	return C.GoString(C.ctx_Parser_Get_NextParam(parser.ctxPtr)), parser.ctx.DSSError()
}

// Return next parameter as a string
func (parser *IParser) StrValue() (string, error) {
	return C.GoString(C.ctx_Parser_Get_StrValue(parser.ctxPtr)), parser.ctx.DSSError()
}

// Get/set the characters used for White space in the command string.  Default is blank and Tab.
func (parser *IParser) Get_WhiteSpace() (string, error) {
	return C.GoString(C.ctx_Parser_Get_WhiteSpace(parser.ctxPtr)), parser.ctx.DSSError()
}

func (parser *IParser) Set_WhiteSpace(value string) error {
	value_c := C.CString(value)
	C.ctx_Parser_Set_WhiteSpace(parser.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return parser.ctx.DSSError()
}

type IReduceCkt struct {
	ICommonData
}

func (reduceckt *IReduceCkt) Init(ctx *DSSContextPtrs) {
	reduceckt.InitCommon(ctx)
}

// Zmag (ohms) for Reduce Option for Z of short lines
func (reduceckt *IReduceCkt) Get_Zmag() (float64, error) {
	return (float64)(C.ctx_ReduceCkt_Get_Zmag(reduceckt.ctxPtr)), reduceckt.ctx.DSSError()
}

func (reduceckt *IReduceCkt) Set_Zmag(value float64) error {
	C.ctx_ReduceCkt_Set_Zmag(reduceckt.ctxPtr, (C.double)(value))
	return reduceckt.ctx.DSSError()
}

// Keep load flag for Reduction options that remove branches
func (reduceckt *IReduceCkt) Get_KeepLoad() (bool, error) {
	return (C.ctx_ReduceCkt_Get_KeepLoad(reduceckt.ctxPtr) != 0), reduceckt.ctx.DSSError()
}

func (reduceckt *IReduceCkt) Set_KeepLoad(value bool) error {
	C.ctx_ReduceCkt_Set_KeepLoad(reduceckt.ctxPtr, ToUint16(value))
	return reduceckt.ctx.DSSError()
}

// Edit String for RemoveBranches functions
func (reduceckt *IReduceCkt) Get_EditString() (string, error) {
	return C.GoString(C.ctx_ReduceCkt_Get_EditString(reduceckt.ctxPtr)), reduceckt.ctx.DSSError()
}

func (reduceckt *IReduceCkt) Set_EditString(value string) error {
	value_c := C.CString(value)
	C.ctx_ReduceCkt_Set_EditString(reduceckt.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return reduceckt.ctx.DSSError()
}

// Start element for Remove Branch function
func (reduceckt *IReduceCkt) Get_StartPDElement() (string, error) {
	return C.GoString(C.ctx_ReduceCkt_Get_StartPDElement(reduceckt.ctxPtr)), reduceckt.ctx.DSSError()
}

func (reduceckt *IReduceCkt) Set_StartPDElement(value string) error {
	value_c := C.CString(value)
	C.ctx_ReduceCkt_Set_StartPDElement(reduceckt.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return reduceckt.ctx.DSSError()
}

// Name of Energymeter to use for reduction
func (reduceckt *IReduceCkt) Get_EnergyMeter() (string, error) {
	return C.GoString(C.ctx_ReduceCkt_Get_EnergyMeter(reduceckt.ctxPtr)), reduceckt.ctx.DSSError()
}

func (reduceckt *IReduceCkt) Set_EnergyMeter(value string) error {
	value_c := C.CString(value)
	C.ctx_ReduceCkt_Set_EnergyMeter(reduceckt.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return reduceckt.ctx.DSSError()
}

// Save present (reduced) circuit
// Filename is listed in the Text Result interface
func (reduceckt *IReduceCkt) SaveCircuit(CktName string) error {
	CktName_c := C.CString(CktName)
	C.ctx_ReduceCkt_SaveCircuit(reduceckt.ctxPtr, CktName_c)
	C.free(unsafe.Pointer(CktName_c))
	return reduceckt.ctx.DSSError()
}

// Do Default Reduction algorithm
func (reduceckt *IReduceCkt) DoDefault() error {
	C.ctx_ReduceCkt_DoDefault(reduceckt.ctxPtr)
	return reduceckt.ctx.DSSError()
}

// Do ShortLines algorithm: Set Zmag first if you don't want the default
func (reduceckt *IReduceCkt) DoShortLines() error {
	C.ctx_ReduceCkt_DoShortLines(reduceckt.ctxPtr)
	return reduceckt.ctx.DSSError()
}

// Reduce Dangling Algorithm; branches with nothing connected
func (reduceckt *IReduceCkt) DoDangling() error {
	C.ctx_ReduceCkt_DoDangling(reduceckt.ctxPtr)
	return reduceckt.ctx.DSSError()
}

func (reduceckt *IReduceCkt) DoLoopBreak() error {
	C.ctx_ReduceCkt_DoLoopBreak(reduceckt.ctxPtr)
	return reduceckt.ctx.DSSError()
}

func (reduceckt *IReduceCkt) DoParallelLines() error {
	C.ctx_ReduceCkt_DoParallelLines(reduceckt.ctxPtr)
	return reduceckt.ctx.DSSError()
}

func (reduceckt *IReduceCkt) DoSwitches() error {
	C.ctx_ReduceCkt_DoSwitches(reduceckt.ctxPtr)
	return reduceckt.ctx.DSSError()
}

func (reduceckt *IReduceCkt) Do1phLaterals() error {
	C.ctx_ReduceCkt_Do1phLaterals(reduceckt.ctxPtr)
	return reduceckt.ctx.DSSError()
}

func (reduceckt *IReduceCkt) DoBranchRemove() error {
	C.ctx_ReduceCkt_DoBranchRemove(reduceckt.ctxPtr)
	return reduceckt.ctx.DSSError()
}

type ISolution struct {
	ICommonData
}

func (solution *ISolution) Init(ctx *DSSContextPtrs) {
	solution.InitCommon(ctx)
}

func (solution *ISolution) BuildYMatrix(BuildOption int32, AllocateVI int32) error {
	C.ctx_Solution_BuildYMatrix(solution.ctxPtr, (C.int32_t)(BuildOption), (C.int32_t)(AllocateVI))
	return solution.ctx.DSSError()
}

func (solution *ISolution) CheckControls() error {
	C.ctx_Solution_CheckControls(solution.ctxPtr)
	return solution.ctx.DSSError()
}

func (solution *ISolution) CheckFaultStatus() error {
	C.ctx_Solution_CheckFaultStatus(solution.ctxPtr)
	return solution.ctx.DSSError()
}

func (solution *ISolution) Cleanup() error {
	C.ctx_Solution_Cleanup(solution.ctxPtr)
	return solution.ctx.DSSError()
}

func (solution *ISolution) DoControlActions() error {
	C.ctx_Solution_DoControlActions(solution.ctxPtr)
	return solution.ctx.DSSError()
}

func (solution *ISolution) FinishTimeStep() error {
	C.ctx_Solution_FinishTimeStep(solution.ctxPtr)
	return solution.ctx.DSSError()
}

func (solution *ISolution) InitSnap() error {
	C.ctx_Solution_InitSnap(solution.ctxPtr)
	return solution.ctx.DSSError()
}

func (solution *ISolution) SampleControlDevices() error {
	C.ctx_Solution_SampleControlDevices(solution.ctxPtr)
	return solution.ctx.DSSError()
}

func (solution *ISolution) Sample_DoControlActions() error {
	C.ctx_Solution_Sample_DoControlActions(solution.ctxPtr)
	return solution.ctx.DSSError()
}

func (solution *ISolution) Solve() error {
	C.ctx_Solution_Solve(solution.ctxPtr)
	return solution.ctx.DSSError()
}

func (solution *ISolution) SolveDirect() error {
	C.ctx_Solution_SolveDirect(solution.ctxPtr)
	return solution.ctx.DSSError()
}

func (solution *ISolution) SolveNoControl() error {
	C.ctx_Solution_SolveNoControl(solution.ctxPtr)
	return solution.ctx.DSSError()
}

func (solution *ISolution) SolvePflow() error {
	C.ctx_Solution_SolvePflow(solution.ctxPtr)
	return solution.ctx.DSSError()
}

func (solution *ISolution) SolvePlusControl() error {
	C.ctx_Solution_SolvePlusControl(solution.ctxPtr)
	return solution.ctx.DSSError()
}

func (solution *ISolution) SolveSnap() error {
	C.ctx_Solution_SolveSnap(solution.ctxPtr)
	return solution.ctx.DSSError()
}

// Type of device to add in AutoAdd Mode: {dssGen (Default) | dssCap}
func (solution *ISolution) Get_AddType() (int32, error) {
	return (int32)(C.ctx_Solution_Get_AddType(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_AddType(value int32) error {
	C.ctx_Solution_Set_AddType(solution.ctxPtr, (C.int32_t)(value))
	return solution.ctx.DSSError()
}

// Base Solution algorithm: {dssNormalSolve | dssNewtonSolve}
func (solution *ISolution) Get_Algorithm() (SolutionAlgorithms, error) {
	return (SolutionAlgorithms)(C.ctx_Solution_Get_Algorithm(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_Algorithm(value SolutionAlgorithms) error {
	C.ctx_Solution_Set_Algorithm(solution.ctxPtr, (C.int32_t)(value))
	return solution.ctx.DSSError()
}

// Capacitor kvar for adding capacitors in AutoAdd mode
func (solution *ISolution) Get_Capkvar() (float64, error) {
	return (float64)(C.ctx_Solution_Get_Capkvar(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_Capkvar(value float64) error {
	C.ctx_Solution_Set_Capkvar(solution.ctxPtr, (C.double)(value))
	return solution.ctx.DSSError()
}

// Flag indicating the control actions are done.
func (solution *ISolution) Get_ControlActionsDone() (bool, error) {
	return (C.ctx_Solution_Get_ControlActionsDone(solution.ctxPtr) != 0), solution.ctx.DSSError()
}

func (solution *ISolution) Set_ControlActionsDone(value bool) error {
	C.ctx_Solution_Set_ControlActionsDone(solution.ctxPtr, ToUint16(value))
	return solution.ctx.DSSError()
}

// Value of the control iteration counter
func (solution *ISolution) Get_ControlIterations() (int32, error) {
	return (int32)(C.ctx_Solution_Get_ControlIterations(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_ControlIterations(value int32) error {
	C.ctx_Solution_Set_ControlIterations(solution.ctxPtr, (C.int32_t)(value))
	return solution.ctx.DSSError()
}

// {dssStatic* | dssEvent | dssTime}  Modes for control devices
func (solution *ISolution) Get_ControlMode() (ControlModes, error) {
	return (ControlModes)(C.ctx_Solution_Get_ControlMode(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_ControlMode(value ControlModes) error {
	C.ctx_Solution_Set_ControlMode(solution.ctxPtr, (C.int32_t)(value))
	return solution.ctx.DSSError()
}

// Flag to indicate whether the circuit solution converged
func (solution *ISolution) Get_Converged() (bool, error) {
	return (C.ctx_Solution_Get_Converged(solution.ctxPtr) != 0), solution.ctx.DSSError()
}

func (solution *ISolution) Set_Converged(value bool) error {
	C.ctx_Solution_Set_Converged(solution.ctxPtr, ToUint16(value))
	return solution.ctx.DSSError()
}

// Default daily load shape (defaults to "Default")
func (solution *ISolution) Get_DefaultDaily() (string, error) {
	return C.GoString(C.ctx_Solution_Get_DefaultDaily(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_DefaultDaily(value string) error {
	value_c := C.CString(value)
	C.ctx_Solution_Set_DefaultDaily(solution.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return solution.ctx.DSSError()
}

// Default Yearly load shape (defaults to "Default")
func (solution *ISolution) Get_DefaultYearly() (string, error) {
	return C.GoString(C.ctx_Solution_Get_DefaultYearly(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_DefaultYearly(value string) error {
	value_c := C.CString(value)
	C.ctx_Solution_Set_DefaultYearly(solution.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return solution.ctx.DSSError()
}

// Array of strings containing the Event Log
func (solution *ISolution) EventLog() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Solution_Get_EventLog(solution.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return solution.ctx.GetStringArray(data, cnt)
}

// Set the Frequency for next solution
func (solution *ISolution) Get_Frequency() (float64, error) {
	return (float64)(C.ctx_Solution_Get_Frequency(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_Frequency(value float64) error {
	C.ctx_Solution_Set_Frequency(solution.ctxPtr, (C.double)(value))
	return solution.ctx.DSSError()
}

// Default Multiplier applied to generators (like LoadMult)
func (solution *ISolution) Get_GenMult() (float64, error) {
	return (float64)(C.ctx_Solution_Get_GenMult(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_GenMult(value float64) error {
	C.ctx_Solution_Set_GenMult(solution.ctxPtr, (C.double)(value))
	return solution.ctx.DSSError()
}

// PF for generators in AutoAdd mode
func (solution *ISolution) Get_GenPF() (float64, error) {
	return (float64)(C.ctx_Solution_Get_GenPF(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_GenPF(value float64) error {
	C.ctx_Solution_Set_GenPF(solution.ctxPtr, (C.double)(value))
	return solution.ctx.DSSError()
}

// Generator kW for AutoAdd mode
func (solution *ISolution) Get_GenkW() (float64, error) {
	return (float64)(C.ctx_Solution_Get_GenkW(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_GenkW(value float64) error {
	C.ctx_Solution_Set_GenkW(solution.ctxPtr, (C.double)(value))
	return solution.ctx.DSSError()
}

// Set Hour for time series solutions.
func (solution *ISolution) Get_Hour() (int32, error) {
	return (int32)(C.ctx_Solution_Get_Hour(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_Hour(value int32) error {
	C.ctx_Solution_Set_Hour(solution.ctxPtr, (C.int32_t)(value))
	return solution.ctx.DSSError()
}

// Get/Set the Solution.IntervalHrs variable used for devices that integrate / custom solution algorithms
func (solution *ISolution) Get_IntervalHrs() (float64, error) {
	return (float64)(C.ctx_Solution_Get_IntervalHrs(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_IntervalHrs(value float64) error {
	C.ctx_Solution_Set_IntervalHrs(solution.ctxPtr, (C.double)(value))
	return solution.ctx.DSSError()
}

// Number of iterations taken for last solution. (Same as Totaliterations)
func (solution *ISolution) Iterations() (int32, error) {
	return (int32)(C.ctx_Solution_Get_Iterations(solution.ctxPtr)), solution.ctx.DSSError()
}

// Load-Duration Curve name for LD modes
func (solution *ISolution) Get_LDCurve() (string, error) {
	return C.GoString(C.ctx_Solution_Get_LDCurve(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_LDCurve(value string) error {
	value_c := C.CString(value)
	C.ctx_Solution_Set_LDCurve(solution.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return solution.ctx.DSSError()
}

// Load Model: {dssPowerFlow (default) | dssAdmittance}
func (solution *ISolution) Get_LoadModel() (int32, error) {
	return (int32)(C.ctx_Solution_Get_LoadModel(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_LoadModel(value int32) error {
	C.ctx_Solution_Set_LoadModel(solution.ctxPtr, (C.int32_t)(value))
	return solution.ctx.DSSError()
}

// Default load multiplier applied to all non-fixed loads
func (solution *ISolution) Get_LoadMult() (float64, error) {
	return (float64)(C.ctx_Solution_Get_LoadMult(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_LoadMult(value float64) error {
	C.ctx_Solution_Set_LoadMult(solution.ctxPtr, (C.double)(value))
	return solution.ctx.DSSError()
}

// Maximum allowable control iterations
func (solution *ISolution) Get_MaxControlIterations() (int32, error) {
	return (int32)(C.ctx_Solution_Get_MaxControlIterations(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_MaxControlIterations(value int32) error {
	C.ctx_Solution_Set_MaxControlIterations(solution.ctxPtr, (C.int32_t)(value))
	return solution.ctx.DSSError()
}

// Max allowable iterations.
func (solution *ISolution) Get_MaxIterations() (int32, error) {
	return (int32)(C.ctx_Solution_Get_MaxIterations(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_MaxIterations(value int32) error {
	C.ctx_Solution_Set_MaxIterations(solution.ctxPtr, (C.int32_t)(value))
	return solution.ctx.DSSError()
}

// Minimum number of iterations required for a power flow solution.
func (solution *ISolution) Get_MinIterations() (int32, error) {
	return (int32)(C.ctx_Solution_Get_MinIterations(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_MinIterations(value int32) error {
	C.ctx_Solution_Set_MinIterations(solution.ctxPtr, (C.int32_t)(value))
	return solution.ctx.DSSError()
}

// Set present solution mode
func (solution *ISolution) Get_Mode() (SolveModes, error) {
	return (SolveModes)(C.ctx_Solution_Get_Mode(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_Mode(value SolveModes) error {
	C.ctx_Solution_Set_Mode(solution.ctxPtr, (C.int32_t)(value))
	return solution.ctx.DSSError()
}

// ID (text) of the present solution mode
func (solution *ISolution) ModeID() (string, error) {
	return C.GoString(C.ctx_Solution_Get_ModeID(solution.ctxPtr)), solution.ctx.DSSError()
}

// Max number of iterations required to converge at any control iteration of the most recent solution.
func (solution *ISolution) MostIterationsDone() (int32, error) {
	return (int32)(C.ctx_Solution_Get_MostIterationsDone(solution.ctxPtr)), solution.ctx.DSSError()
}

// Number of solutions to perform for Monte Carlo and time series simulations
func (solution *ISolution) Get_Number() (int32, error) {
	return (int32)(C.ctx_Solution_Get_Number(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_Number(value int32) error {
	C.ctx_Solution_Set_Number(solution.ctxPtr, (C.int32_t)(value))
	return solution.ctx.DSSError()
}

// Gets the time required to perform the latest solution (Read only)
func (solution *ISolution) Process_Time() (float64, error) {
	return (float64)(C.ctx_Solution_Get_Process_Time(solution.ctxPtr)), solution.ctx.DSSError()
}

// Randomization mode for random variables "Gaussian" or "Uniform"
func (solution *ISolution) Get_Random() (int32, error) {
	return (int32)(C.ctx_Solution_Get_Random(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_Random(value int32) error {
	C.ctx_Solution_Set_Random(solution.ctxPtr, (C.int32_t)(value))
	return solution.ctx.DSSError()
}

// Seconds from top of the hour.
func (solution *ISolution) Get_Seconds() (float64, error) {
	return (float64)(C.ctx_Solution_Get_Seconds(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_Seconds(value float64) error {
	C.ctx_Solution_Set_Seconds(solution.ctxPtr, (C.double)(value))
	return solution.ctx.DSSError()
}

// Time step size in sec
func (solution *ISolution) Get_StepSize() (float64, error) {
	return (float64)(C.ctx_Solution_Get_StepSize(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_StepSize(value float64) error {
	C.ctx_Solution_Set_StepSize(solution.ctxPtr, (C.double)(value))
	return solution.ctx.DSSError()
}

// Flag that indicates if elements of the System Y have been changed by recent activity.
func (solution *ISolution) SystemYChanged() (bool, error) {
	return (C.ctx_Solution_Get_SystemYChanged(solution.ctxPtr) != 0), solution.ctx.DSSError()
}

// Get the solution process time + sample time for time step
func (solution *ISolution) Time_of_Step() (float64, error) {
	return (float64)(C.ctx_Solution_Get_Time_of_Step(solution.ctxPtr)), solution.ctx.DSSError()
}

// Solution convergence tolerance.
func (solution *ISolution) Get_Tolerance() (float64, error) {
	return (float64)(C.ctx_Solution_Get_Tolerance(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_Tolerance(value float64) error {
	C.ctx_Solution_Set_Tolerance(solution.ctxPtr, (C.double)(value))
	return solution.ctx.DSSError()
}

// Gets/sets the accumulated time of the simulation
func (solution *ISolution) Get_Total_Time() (float64, error) {
	return (float64)(C.ctx_Solution_Get_Total_Time(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_Total_Time(value float64) error {
	C.ctx_Solution_Set_Total_Time(solution.ctxPtr, (C.double)(value))
	return solution.ctx.DSSError()
}

// Total iterations including control iterations for most recent solution.
func (solution *ISolution) Totaliterations() (int32, error) {
	return (int32)(C.ctx_Solution_Get_Totaliterations(solution.ctxPtr)), solution.ctx.DSSError()
}

// Set year for planning studies
func (solution *ISolution) Get_Year() (int32, error) {
	return (int32)(C.ctx_Solution_Get_Year(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_Year(value int32) error {
	C.ctx_Solution_Set_Year(solution.ctxPtr, (C.int32_t)(value))
	return solution.ctx.DSSError()
}

// Hour as a double, including fractional part
func (solution *ISolution) Get_dblHour() (float64, error) {
	return (float64)(C.ctx_Solution_Get_dblHour(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_dblHour(value float64) error {
	C.ctx_Solution_Set_dblHour(solution.ctxPtr, (C.double)(value))
	return solution.ctx.DSSError()
}

// Percent default  annual load growth rate
func (solution *ISolution) Get_pctGrowth() (float64, error) {
	return (float64)(C.ctx_Solution_Get_pctGrowth(solution.ctxPtr)), solution.ctx.DSSError()
}

func (solution *ISolution) Set_pctGrowth(value float64) error {
	C.ctx_Solution_Set_pctGrowth(solution.ctxPtr, (C.double)(value))
	return solution.ctx.DSSError()
}

func (solution *ISolution) Set_StepsizeHr(value float64) error {
	C.ctx_Solution_Set_StepsizeHr(solution.ctxPtr, (C.double)(value))
	return solution.ctx.DSSError()
}

func (solution *ISolution) Set_StepsizeMin(value float64) error {
	C.ctx_Solution_Set_StepsizeMin(solution.ctxPtr, (C.double)(value))
	return solution.ctx.DSSError()
}

func (solution *ISolution) BusLevels() ([]int32, error) {
	C.ctx_Solution_Get_BusLevels_GR(solution.ctxPtr)
	return solution.ctx.GetInt32ArrayGR()
}

func (solution *ISolution) IncMatrix() ([]int32, error) {
	C.ctx_Solution_Get_IncMatrix_GR(solution.ctxPtr)
	return solution.ctx.GetInt32ArrayGR()
}

func (solution *ISolution) IncMatrixCols() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Solution_Get_IncMatrixCols(solution.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return solution.ctx.GetStringArray(data, cnt)
}

func (solution *ISolution) IncMatrixRows() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Solution_Get_IncMatrixRows(solution.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return solution.ctx.GetStringArray(data, cnt)
}

func (solution *ISolution) Laplacian() ([]int32, error) {
	C.ctx_Solution_Get_Laplacian_GR(solution.ctxPtr)
	return solution.ctx.GetInt32ArrayGR()
}
func (solution *ISolution) SolveAll() error {
	C.ctx_Solution_SolveAll(solution.ctxPtr)
	return solution.ctx.DSSError()
}

type ILineGeometries struct {
	ICommonData
}

func (linegeometries *ILineGeometries) Init(ctx *DSSContextPtrs) {
	linegeometries.InitCommon(ctx)
}

// Array of strings with all LineGeometrie names in the circuit.
func (linegeometries *ILineGeometries) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_LineGeometries_Get_AllNames(linegeometries.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return linegeometries.ctx.GetStringArray(data, cnt)
}

// Number of LineGeometrie objects in active circuit.
func (linegeometries *ILineGeometries) Count() (int32, error) {
	return (int32)(C.ctx_LineGeometries_Get_Count(linegeometries.ctxPtr)), linegeometries.ctx.DSSError()
}

// Sets the first LineGeometrie active. Returns 0 if no more.
func (linegeometries *ILineGeometries) First() (int32, error) {
	return (int32)(C.ctx_LineGeometries_Get_First(linegeometries.ctxPtr)), linegeometries.ctx.DSSError()
}

// Sets the active LineGeometrie by Name.
func (linegeometries *ILineGeometries) Get_Name() (string, error) {
	result := C.GoString(C.ctx_LineGeometries_Get_Name(linegeometries.ctxPtr))
	return result, linegeometries.ctx.DSSError()
}

// Gets the name of the active LineGeometrie.
func (linegeometries *ILineGeometries) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_LineGeometries_Set_Name(linegeometries.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return linegeometries.ctx.DSSError()
}

// Sets the next LineGeometrie active. Returns 0 if no more.
func (linegeometries *ILineGeometries) Next() (int32, error) {
	return (int32)(C.ctx_LineGeometries_Get_Next(linegeometries.ctxPtr)), linegeometries.ctx.DSSError()
}

// Get the index of the active LineGeometrie; index is 1-based: 1..count
func (linegeometries *ILineGeometries) Get_idx() (int32, error) {
	return (int32)(C.ctx_LineGeometries_Get_idx(linegeometries.ctxPtr)), linegeometries.ctx.DSSError()
}

// Set the active LineGeometrie by index; index is 1-based: 1..count
func (linegeometries *ILineGeometries) Set_idx(value int32) error {
	C.ctx_LineGeometries_Set_idx(linegeometries.ctxPtr, (C.int32_t)(value))
	return linegeometries.ctx.DSSError()
}

// Array of strings with names of all conductors in the active LineGeometry object
func (linegeometries *ILineGeometries) Conductors() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_LineGeometries_Get_Conductors(linegeometries.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return linegeometries.ctx.GetStringArray(data, cnt)
}

// Emergency ampere rating
func (linegeometries *ILineGeometries) Get_EmergAmps() (float64, error) {
	return (float64)(C.ctx_LineGeometries_Get_EmergAmps(linegeometries.ctxPtr)), linegeometries.ctx.DSSError()
}

func (linegeometries *ILineGeometries) Set_EmergAmps(value float64) error {
	C.ctx_LineGeometries_Set_EmergAmps(linegeometries.ctxPtr, (C.double)(value))
	return linegeometries.ctx.DSSError()
}

// Normal ampere rating
func (linegeometries *ILineGeometries) Get_NormAmps() (float64, error) {
	return (float64)(C.ctx_LineGeometries_Get_NormAmps(linegeometries.ctxPtr)), linegeometries.ctx.DSSError()
}

func (linegeometries *ILineGeometries) Set_NormAmps(value float64) error {
	C.ctx_LineGeometries_Set_NormAmps(linegeometries.ctxPtr, (C.double)(value))
	return linegeometries.ctx.DSSError()
}

func (linegeometries *ILineGeometries) Get_RhoEarth() (float64, error) {
	return (float64)(C.ctx_LineGeometries_Get_RhoEarth(linegeometries.ctxPtr)), linegeometries.ctx.DSSError()
}

func (linegeometries *ILineGeometries) Set_RhoEarth(value float64) error {
	C.ctx_LineGeometries_Set_RhoEarth(linegeometries.ctxPtr, (C.double)(value))
	return linegeometries.ctx.DSSError()
}

func (linegeometries *ILineGeometries) Get_Reduce() (bool, error) {
	return (C.ctx_LineGeometries_Get_Reduce(linegeometries.ctxPtr) != 0), linegeometries.ctx.DSSError()
}

func (linegeometries *ILineGeometries) Set_Reduce(value bool) error {
	C.ctx_LineGeometries_Set_Reduce(linegeometries.ctxPtr, ToUint16(value))
	return linegeometries.ctx.DSSError()
}

// Number of Phases
func (linegeometries *ILineGeometries) Get_Phases() (int32, error) {
	return (int32)(C.ctx_LineGeometries_Get_Phases(linegeometries.ctxPtr)), linegeometries.ctx.DSSError()
}

func (linegeometries *ILineGeometries) Set_Phases(value int32) error {
	C.ctx_LineGeometries_Set_Phases(linegeometries.ctxPtr, (C.int32_t)(value))
	return linegeometries.ctx.DSSError()
}

// Resistance matrix, ohms
func (linegeometries *ILineGeometries) Rmatrix(Frequency float64, Length float64, Units int32) ([]float64, error) {
	C.ctx_LineGeometries_Get_Rmatrix_GR(linegeometries.ctxPtr, (C.double)(Frequency), (C.double)(Length), (C.int32_t)(Units))
	return linegeometries.ctx.GetFloat64ArrayGR()
}

// Reactance matrix, ohms
func (linegeometries *ILineGeometries) Xmatrix(Frequency float64, Length float64, Units int32) ([]float64, error) {
	C.ctx_LineGeometries_Get_Xmatrix_GR(linegeometries.ctxPtr, (C.double)(Frequency), (C.double)(Length), (C.int32_t)(Units))
	return linegeometries.ctx.GetFloat64ArrayGR()
}

// Complex impedance matrix, ohms
func (linegeometries *ILineGeometries) Zmatrix(Frequency float64, Length float64, Units int32) ([]complex128, error) {
	C.ctx_LineGeometries_Get_Zmatrix_GR(linegeometries.ctxPtr, (C.double)(Frequency), (C.double)(Length), (C.int32_t)(Units))
	return linegeometries.ctx.GetComplexArrayGR()
}

// Capacitance matrix, nF
func (linegeometries *ILineGeometries) Cmatrix(Frequency float64, Length float64, Units int32) ([]float64, error) {
	C.ctx_LineGeometries_Get_Cmatrix_GR(linegeometries.ctxPtr, (C.double)(Frequency), (C.double)(Length), (C.int32_t)(Units))
	return linegeometries.ctx.GetFloat64ArrayGR()
}

func (linegeometries *ILineGeometries) Get_Units() ([]LineUnits, error) {
	C.ctx_LineGeometries_Get_Units_GR(linegeometries.ctxPtr)
	tmp, err := (linegeometries.ctx.GetInt32ArrayGR())
	res := make([]LineUnits, len(tmp))
	for i := 0; i < len(tmp); i++ {
		res[i] = (LineUnits)(tmp[i])
	}
	return res, err
}

func (linegeometries *ILineGeometries) Set_Units(value []LineUnits) error {
	C.ctx_LineGeometries_Set_Units(linegeometries.ctxPtr, (*C.int32_t)(&value[0]), (C.int32_t)(len(value)))
	return linegeometries.ctx.DSSError()
}

// Get/Set the X (horizontal) coordinates of the conductors
func (linegeometries *ILineGeometries) Get_Xcoords() ([]float64, error) {
	C.ctx_LineGeometries_Get_Xcoords_GR(linegeometries.ctxPtr)
	return linegeometries.ctx.GetFloat64ArrayGR()
}

func (linegeometries *ILineGeometries) Set_Xcoords(value []float64) error {
	C.ctx_LineGeometries_Set_Xcoords(linegeometries.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return linegeometries.ctx.DSSError()
}

// Get/Set the Y (vertical/height) coordinates of the conductors
func (linegeometries *ILineGeometries) Get_Ycoords() ([]float64, error) {
	C.ctx_LineGeometries_Get_Ycoords_GR(linegeometries.ctxPtr)
	return linegeometries.ctx.GetFloat64ArrayGR()
}

func (linegeometries *ILineGeometries) Set_Ycoords(value []float64) error {
	C.ctx_LineGeometries_Set_Ycoords(linegeometries.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return linegeometries.ctx.DSSError()
}

// Number of conductors in this geometry. Default is 3. Triggers memory allocations. Define first!
func (linegeometries *ILineGeometries) Get_Nconds() (int32, error) {
	return (int32)(C.ctx_LineGeometries_Get_Nconds(linegeometries.ctxPtr)), linegeometries.ctx.DSSError()
}

func (linegeometries *ILineGeometries) Set_Nconds(value int32) error {
	C.ctx_LineGeometries_Set_Nconds(linegeometries.ctxPtr, (C.int32_t)(value))
	return linegeometries.ctx.DSSError()
}

type ILineSpacings struct {
	ICommonData
}

func (linespacings *ILineSpacings) Init(ctx *DSSContextPtrs) {
	linespacings.InitCommon(ctx)
}

// Array of strings with all LineSpacing names in the circuit.
func (linespacings *ILineSpacings) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_LineSpacings_Get_AllNames(linespacings.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return linespacings.ctx.GetStringArray(data, cnt)
}

// Number of LineSpacing objects in active circuit.
func (linespacings *ILineSpacings) Count() (int32, error) {
	return (int32)(C.ctx_LineSpacings_Get_Count(linespacings.ctxPtr)), linespacings.ctx.DSSError()
}

// Sets the first LineSpacing active. Returns 0 if no more.
func (linespacings *ILineSpacings) First() (int32, error) {
	return (int32)(C.ctx_LineSpacings_Get_First(linespacings.ctxPtr)), linespacings.ctx.DSSError()
}

// Sets the active LineSpacing by Name.
func (linespacings *ILineSpacings) Get_Name() (string, error) {
	result := C.GoString(C.ctx_LineSpacings_Get_Name(linespacings.ctxPtr))
	return result, linespacings.ctx.DSSError()
}

// Gets the name of the active LineSpacing.
func (linespacings *ILineSpacings) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_LineSpacings_Set_Name(linespacings.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return linespacings.ctx.DSSError()
}

// Sets the next LineSpacing active. Returns 0 if no more.
func (linespacings *ILineSpacings) Next() (int32, error) {
	return (int32)(C.ctx_LineSpacings_Get_Next(linespacings.ctxPtr)), linespacings.ctx.DSSError()
}

// Get the index of the active LineSpacing; index is 1-based: 1..count
func (linespacings *ILineSpacings) Get_idx() (int32, error) {
	return (int32)(C.ctx_LineSpacings_Get_idx(linespacings.ctxPtr)), linespacings.ctx.DSSError()
}

// Set the active LineSpacing by index; index is 1-based: 1..count
func (linespacings *ILineSpacings) Set_idx(value int32) error {
	C.ctx_LineSpacings_Set_idx(linespacings.ctxPtr, (C.int32_t)(value))
	return linespacings.ctx.DSSError()
}

// Number of Phases
func (linespacings *ILineSpacings) Get_Phases() (int32, error) {
	return (int32)(C.ctx_LineSpacings_Get_Phases(linespacings.ctxPtr)), linespacings.ctx.DSSError()
}

func (linespacings *ILineSpacings) Set_Phases(value int32) error {
	C.ctx_LineSpacings_Set_Phases(linespacings.ctxPtr, (C.int32_t)(value))
	return linespacings.ctx.DSSError()
}

func (linespacings *ILineSpacings) Get_Nconds() (int32, error) {
	return (int32)(C.ctx_LineSpacings_Get_Nconds(linespacings.ctxPtr)), linespacings.ctx.DSSError()
}

func (linespacings *ILineSpacings) Set_Nconds(value int32) error {
	C.ctx_LineSpacings_Set_Nconds(linespacings.ctxPtr, (C.int32_t)(value))
	return linespacings.ctx.DSSError()
}

func (linespacings *ILineSpacings) Get_Units() (LineUnits, error) {
	return (LineUnits)(C.ctx_LineSpacings_Get_Units(linespacings.ctxPtr)), linespacings.ctx.DSSError()
}

func (linespacings *ILineSpacings) Set_Units(value LineUnits) error {
	C.ctx_LineSpacings_Set_Units(linespacings.ctxPtr, (C.int32_t)(value))
	return linespacings.ctx.DSSError()
}

// Get/Set the X (horizontal) coordinates of the conductors
func (linespacings *ILineSpacings) Get_Xcoords() ([]float64, error) {
	C.ctx_LineSpacings_Get_Xcoords_GR(linespacings.ctxPtr)
	return linespacings.ctx.GetFloat64ArrayGR()
}

func (linespacings *ILineSpacings) Set_Xcoords(value []float64) error {
	C.ctx_LineSpacings_Set_Xcoords(linespacings.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return linespacings.ctx.DSSError()
}

// Get/Set the Y (vertical/height) coordinates of the conductors
func (linespacings *ILineSpacings) Get_Ycoords() ([]float64, error) {
	C.ctx_LineSpacings_Get_Ycoords_GR(linespacings.ctxPtr)
	return linespacings.ctx.GetFloat64ArrayGR()
}

func (linespacings *ILineSpacings) Set_Ycoords(value []float64) error {
	C.ctx_LineSpacings_Set_Ycoords(linespacings.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return linespacings.ctx.DSSError()
}

type ILoadShapes struct {
	ICommonData
}

func (loadshapes *ILoadShapes) Init(ctx *DSSContextPtrs) {
	loadshapes.InitCommon(ctx)
}

// Array of strings with all LoadShape names in the circuit.
func (loadshapes *ILoadShapes) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_LoadShapes_Get_AllNames(loadshapes.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return loadshapes.ctx.GetStringArray(data, cnt)
}

// Number of LoadShape objects in active circuit.
func (loadshapes *ILoadShapes) Count() (int32, error) {
	return (int32)(C.ctx_LoadShapes_Get_Count(loadshapes.ctxPtr)), loadshapes.ctx.DSSError()
}

// Sets the first LoadShape active. Returns 0 if no more.
func (loadshapes *ILoadShapes) First() (int32, error) {
	return (int32)(C.ctx_LoadShapes_Get_First(loadshapes.ctxPtr)), loadshapes.ctx.DSSError()
}

// Sets the active LoadShape by Name.
func (loadshapes *ILoadShapes) Get_Name() (string, error) {
	result := C.GoString(C.ctx_LoadShapes_Get_Name(loadshapes.ctxPtr))
	return result, loadshapes.ctx.DSSError()
}

// Gets the name of the active LoadShape.
func (loadshapes *ILoadShapes) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_LoadShapes_Set_Name(loadshapes.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return loadshapes.ctx.DSSError()
}

// Sets the next LoadShape active. Returns 0 if no more.
func (loadshapes *ILoadShapes) Next() (int32, error) {
	return (int32)(C.ctx_LoadShapes_Get_Next(loadshapes.ctxPtr)), loadshapes.ctx.DSSError()
}

// Get the index of the active LoadShape; index is 1-based: 1..count
func (loadshapes *ILoadShapes) Get_idx() (int32, error) {
	return (int32)(C.ctx_LoadShapes_Get_idx(loadshapes.ctxPtr)), loadshapes.ctx.DSSError()
}

// Set the active LoadShape by index; index is 1-based: 1..count
func (loadshapes *ILoadShapes) Set_idx(value int32) error {
	C.ctx_LoadShapes_Set_idx(loadshapes.ctxPtr, (C.int32_t)(value))
	return loadshapes.ctx.DSSError()
}

func (loadshapes *ILoadShapes) New(Name string) (int32, error) {
	Name_c := C.CString(Name)
	defer C.free(unsafe.Pointer(Name_c))
	return (int32)(C.ctx_LoadShapes_New(loadshapes.ctxPtr, Name_c)), loadshapes.ctx.DSSError()
}

func (loadshapes *ILoadShapes) Normalize() error {
	C.ctx_LoadShapes_Normalize(loadshapes.ctxPtr)
	return loadshapes.ctx.DSSError()
}

// Fixed interval time value, hours.
func (loadshapes *ILoadShapes) Get_HrInterval() (float64, error) {
	return (float64)(C.ctx_LoadShapes_Get_HrInterval(loadshapes.ctxPtr)), loadshapes.ctx.DSSError()
}

func (loadshapes *ILoadShapes) Set_HrInterval(value float64) error {
	C.ctx_LoadShapes_Set_HrInterval(loadshapes.ctxPtr, (C.double)(value))
	return loadshapes.ctx.DSSError()
}

// Fixed Interval time value, in minutes
func (loadshapes *ILoadShapes) Get_MinInterval() (float64, error) {
	return (float64)(C.ctx_LoadShapes_Get_MinInterval(loadshapes.ctxPtr)), loadshapes.ctx.DSSError()
}

func (loadshapes *ILoadShapes) Set_MinInterval(value float64) error {
	C.ctx_LoadShapes_Set_MinInterval(loadshapes.ctxPtr, (C.double)(value))
	return loadshapes.ctx.DSSError()
}

// Get/set Number of points in active Loadshape.
func (loadshapes *ILoadShapes) Get_Npts() (int32, error) {
	return (int32)(C.ctx_LoadShapes_Get_Npts(loadshapes.ctxPtr)), loadshapes.ctx.DSSError()
}

func (loadshapes *ILoadShapes) Set_Npts(value int32) error {
	C.ctx_LoadShapes_Set_Npts(loadshapes.ctxPtr, (C.int32_t)(value))
	return loadshapes.ctx.DSSError()
}

func (loadshapes *ILoadShapes) Get_PBase() (float64, error) {
	return (float64)(C.ctx_LoadShapes_Get_PBase(loadshapes.ctxPtr)), loadshapes.ctx.DSSError()
}

func (loadshapes *ILoadShapes) Set_PBase(value float64) error {
	C.ctx_LoadShapes_Set_PBase(loadshapes.ctxPtr, (C.double)(value))
	return loadshapes.ctx.DSSError()
}

// Array of doubles for the P multiplier in the Loadshape.
func (loadshapes *ILoadShapes) Get_Pmult() ([]float64, error) {
	C.ctx_LoadShapes_Get_Pmult_GR(loadshapes.ctxPtr)
	return loadshapes.ctx.GetFloat64ArrayGR()
}

func (loadshapes *ILoadShapes) Set_Pmult(value []float64) error {
	C.ctx_LoadShapes_Set_Pmult(loadshapes.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return loadshapes.ctx.DSSError()
}

// Base for normalizing Q curve. If left at zero, the peak value is used.
func (loadshapes *ILoadShapes) Get_QBase() (float64, error) {
	return (float64)(C.ctx_LoadShapes_Get_Qbase(loadshapes.ctxPtr)), loadshapes.ctx.DSSError()
}

func (loadshapes *ILoadShapes) Set_QBase(value float64) error {
	C.ctx_LoadShapes_Set_Qbase(loadshapes.ctxPtr, (C.double)(value))
	return loadshapes.ctx.DSSError()
}

// Array of doubles containing the Q multipliers.
func (loadshapes *ILoadShapes) Get_Qmult() ([]float64, error) {
	C.ctx_LoadShapes_Get_Qmult_GR(loadshapes.ctxPtr)
	return loadshapes.ctx.GetFloat64ArrayGR()
}

func (loadshapes *ILoadShapes) Set_Qmult(value []float64) error {
	C.ctx_LoadShapes_Set_Qmult(loadshapes.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return loadshapes.ctx.DSSError()
}

// Time array in hours correscponding to P and Q multipliers when the Interval=0.
func (loadshapes *ILoadShapes) Get_TimeArray() ([]float64, error) {
	C.ctx_LoadShapes_Get_TimeArray_GR(loadshapes.ctxPtr)
	return loadshapes.ctx.GetFloat64ArrayGR()
}

func (loadshapes *ILoadShapes) Set_TimeArray(value []float64) error {
	C.ctx_LoadShapes_Set_TimeArray(loadshapes.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return loadshapes.ctx.DSSError()
}

// Boolean flag to let Loads know to use the actual value in the curve rather than use the value as a multiplier.
func (loadshapes *ILoadShapes) Get_UseActual() (bool, error) {
	return (C.ctx_LoadShapes_Get_UseActual(loadshapes.ctxPtr) != 0), loadshapes.ctx.DSSError()
}

func (loadshapes *ILoadShapes) Set_UseActual(value bool) error {
	C.ctx_LoadShapes_Set_UseActual(loadshapes.ctxPtr, ToUint16(value))
	return loadshapes.ctx.DSSError()
}

func (loadshapes *ILoadShapes) Get_sInterval() (float64, error) {
	return (float64)(C.ctx_LoadShapes_Get_SInterval(loadshapes.ctxPtr)), loadshapes.ctx.DSSError()
}

func (loadshapes *ILoadShapes) Set_sInterval(value float64) error {
	C.ctx_LoadShapes_Set_SInterval(loadshapes.ctxPtr, (C.double)(value))
	return loadshapes.ctx.DSSError()
}

// Converts the current LoadShape data to float32/single precision.
// If there is no data or the data is already represented using float32, nothing is done.
//
// (API Extension)
func (loadshapes *ILoadShapes) UseFloat32() error {
	C.ctx_LoadShapes_UseFloat32(loadshapes.ctxPtr)
	return loadshapes.ctx.DSSError()
}

// Converts the current LoadShape data to float64/double precision.
// If there is no data or the data is already represented using float64, nothing is done.
//
// (API Extension)
func (loadshapes *ILoadShapes) UseFloat64() error {
	C.ctx_LoadShapes_UseFloat64(loadshapes.ctxPtr)
	return loadshapes.ctx.DSSError()
}

type ILoads struct {
	ICommonData
}

func (loads *ILoads) Init(ctx *DSSContextPtrs) {
	loads.InitCommon(ctx)
}

// Array of strings with all Load names in the circuit.
func (loads *ILoads) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Loads_Get_AllNames(loads.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return loads.ctx.GetStringArray(data, cnt)
}

// Number of Load objects in active circuit.
func (loads *ILoads) Count() (int32, error) {
	return (int32)(C.ctx_Loads_Get_Count(loads.ctxPtr)), loads.ctx.DSSError()
}

// Sets the first Load active. Returns 0 if no more.
func (loads *ILoads) First() (int32, error) {
	return (int32)(C.ctx_Loads_Get_First(loads.ctxPtr)), loads.ctx.DSSError()
}

// Sets the active Load by Name.
func (loads *ILoads) Get_Name() (string, error) {
	result := C.GoString(C.ctx_Loads_Get_Name(loads.ctxPtr))
	return result, loads.ctx.DSSError()
}

// Gets the name of the active Load.
func (loads *ILoads) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_Loads_Set_Name(loads.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return loads.ctx.DSSError()
}

// Sets the next Load active. Returns 0 if no more.
func (loads *ILoads) Next() (int32, error) {
	return (int32)(C.ctx_Loads_Get_Next(loads.ctxPtr)), loads.ctx.DSSError()
}

// Get the index of the active Load; index is 1-based: 1..count
func (loads *ILoads) Get_idx() (int32, error) {
	return (int32)(C.ctx_Loads_Get_idx(loads.ctxPtr)), loads.ctx.DSSError()
}

// Set the active Load by index; index is 1-based: 1..count
func (loads *ILoads) Set_idx(value int32) error {
	C.ctx_Loads_Set_idx(loads.ctxPtr, (C.int32_t)(value))
	return loads.ctx.DSSError()
}

// Factor for allocating loads by connected xfkva
func (loads *ILoads) Get_AllocationFactor() (float64, error) {
	return (float64)(C.ctx_Loads_Get_AllocationFactor(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_AllocationFactor(value float64) error {
	C.ctx_Loads_Set_AllocationFactor(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Name of a loadshape with both Mult and Qmult, for CVR factors as a function of time.
func (loads *ILoads) Get_CVRcurve() (string, error) {
	return C.GoString(C.ctx_Loads_Get_CVRcurve(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_CVRcurve(value string) error {
	value_c := C.CString(value)
	C.ctx_Loads_Set_CVRcurve(loads.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return loads.ctx.DSSError()
}

// Percent reduction in Q for percent reduction in V. Must be used with dssLoadModelCVR.
func (loads *ILoads) Get_CVRvars() (float64, error) {
	return (float64)(C.ctx_Loads_Get_CVRvars(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_CVRvars(value float64) error {
	C.ctx_Loads_Set_CVRvars(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Percent reduction in P for percent reduction in V. Must be used with dssLoadModelCVR.
func (loads *ILoads) Get_CVRwatts() (float64, error) {
	return (float64)(C.ctx_Loads_Get_CVRwatts(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_CVRwatts(value float64) error {
	C.ctx_Loads_Set_CVRwatts(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Factor relates average to peak kw.  Used for allocation with kwh and kwhdays
func (loads *ILoads) Get_Cfactor() (float64, error) {
	return (float64)(C.ctx_Loads_Get_Cfactor(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_Cfactor(value float64) error {
	C.ctx_Loads_Set_Cfactor(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

func (loads *ILoads) Get_Class() (int32, error) {
	return (int32)(C.ctx_Loads_Get_Class_(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_Class(value int32) error {
	C.ctx_Loads_Set_Class_(loads.ctxPtr, (C.int32_t)(value))
	return loads.ctx.DSSError()
}

// Name of the growthshape curve for yearly load growth factors.
func (loads *ILoads) Get_Growth() (string, error) {
	return C.GoString(C.ctx_Loads_Get_Growth(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_Growth(value string) error {
	value_c := C.CString(value)
	C.ctx_Loads_Set_Growth(loads.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return loads.ctx.DSSError()
}

// Delta loads are connected line-to-line.
func (loads *ILoads) Get_IsDelta() (bool, error) {
	return (C.ctx_Loads_Get_IsDelta(loads.ctxPtr) != 0), loads.ctx.DSSError()
}

func (loads *ILoads) Set_IsDelta(value bool) error {
	C.ctx_Loads_Set_IsDelta(loads.ctxPtr, ToUint16(value))
	return loads.ctx.DSSError()
}

// The Load Model defines variation of P and Q with voltage.
func (loads *ILoads) Get_Model() (LoadModels, error) {
	return (LoadModels)(C.ctx_Loads_Get_Model(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_Model(value LoadModels) error {
	C.ctx_Loads_Set_Model(loads.ctxPtr, (C.int32_t)(value))
	return loads.ctx.DSSError()
}

// Number of customers in this load, defaults to one.
func (loads *ILoads) Get_NumCust() (int32, error) {
	return (int32)(C.ctx_Loads_Get_NumCust(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_NumCust(value int32) error {
	C.ctx_Loads_Set_NumCust(loads.ctxPtr, (C.int32_t)(value))
	return loads.ctx.DSSError()
}

// Get or set Power Factor for Active Load. Specify leading PF as negative. Updates kvar based on present value of kW
func (loads *ILoads) Get_PF() (float64, error) {
	return (float64)(C.ctx_Loads_Get_PF(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_PF(value float64) error {
	C.ctx_Loads_Set_PF(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Average percent of nominal load in Monte Carlo studies; only if no loadshape defined for this load.
func (loads *ILoads) Get_PctMean() (float64, error) {
	return (float64)(C.ctx_Loads_Get_PctMean(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_PctMean(value float64) error {
	C.ctx_Loads_Set_PctMean(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Percent standard deviation for Monte Carlo load studies; if there is no loadshape assigned to this load.
func (loads *ILoads) Get_PctStdDev() (float64, error) {
	return (float64)(C.ctx_Loads_Get_PctStdDev(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_PctStdDev(value float64) error {
	C.ctx_Loads_Set_PctStdDev(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Relative Weighting factor for the active LOAD
func (loads *ILoads) Get_RelWeight() (float64, error) {
	return (float64)(C.ctx_Loads_Get_RelWeight(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_RelWeight(value float64) error {
	C.ctx_Loads_Set_RelWeight(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Neutral resistance for wye-connected loads.
func (loads *ILoads) Get_Rneut() (float64, error) {
	return (float64)(C.ctx_Loads_Get_Rneut(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_Rneut(value float64) error {
	C.ctx_Loads_Set_Rneut(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Name of harmonic current spectrrum shape.
func (loads *ILoads) Get_Spectrum() (string, error) {
	return C.GoString(C.ctx_Loads_Get_Spectrum(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_Spectrum(value string) error {
	value_c := C.CString(value)
	C.ctx_Loads_Set_Spectrum(loads.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return loads.ctx.DSSError()
}

// Response to load multipliers: Fixed (growth only), Exempt (no LD curve), Variable (all).
func (loads *ILoads) Get_Status() (LoadStatus, error) {
	return (LoadStatus)(C.ctx_Loads_Get_Status(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_Status(value LoadStatus) error {
	C.ctx_Loads_Set_Status(loads.ctxPtr, (C.int32_t)(value))
	return loads.ctx.DSSError()
}

// Maximum per-unit voltage to use the load model. Above this, constant Z applies.
func (loads *ILoads) Get_Vmaxpu() (float64, error) {
	return (float64)(C.ctx_Loads_Get_Vmaxpu(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_Vmaxpu(value float64) error {
	C.ctx_Loads_Set_Vmaxpu(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Minimum voltage for unserved energy (UE) evaluation.
func (loads *ILoads) Get_Vminemerg() (float64, error) {
	return (float64)(C.ctx_Loads_Get_Vminemerg(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_Vminemerg(value float64) error {
	C.ctx_Loads_Set_Vminemerg(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Minimum voltage for energy exceeding normal (EEN) evaluations.
func (loads *ILoads) Get_Vminnorm() (float64, error) {
	return (float64)(C.ctx_Loads_Get_Vminnorm(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_Vminnorm(value float64) error {
	C.ctx_Loads_Set_Vminnorm(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Minimum voltage to apply the load model. Below this, constant Z is used.
func (loads *ILoads) Get_Vminpu() (float64, error) {
	return (float64)(C.ctx_Loads_Get_Vminpu(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_Vminpu(value float64) error {
	C.ctx_Loads_Set_Vminpu(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Neutral reactance for wye-connected loads.
func (loads *ILoads) Get_Xneut() (float64, error) {
	return (float64)(C.ctx_Loads_Get_Xneut(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_Xneut(value float64) error {
	C.ctx_Loads_Set_Xneut(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Name of yearly duration loadshape
func (loads *ILoads) Get_Yearly() (string, error) {
	return C.GoString(C.ctx_Loads_Get_Yearly(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_Yearly(value string) error {
	value_c := C.CString(value)
	C.ctx_Loads_Set_Yearly(loads.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return loads.ctx.DSSError()
}

// Array of 7 doubles with values for ZIPV property of the load object
func (loads *ILoads) Get_ZIPV() ([]float64, error) {
	C.ctx_Loads_Get_ZIPV_GR(loads.ctxPtr)
	return loads.ctx.GetFloat64ArrayGR()
}

func (loads *ILoads) Set_ZIPV(value []float64) error {
	C.ctx_Loads_Set_ZIPV(loads.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return loads.ctx.DSSError()
}

// Name of the loadshape for a daily load profile.
func (loads *ILoads) Get_daily() (string, error) {
	return C.GoString(C.ctx_Loads_Get_daily(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_daily(value string) error {
	value_c := C.CString(value)
	C.ctx_Loads_Set_daily(loads.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return loads.ctx.DSSError()
}

// Name of the loadshape for a duty cycle simulation.
func (loads *ILoads) Get_duty() (string, error) {
	return C.GoString(C.ctx_Loads_Get_duty(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_duty(value string) error {
	value_c := C.CString(value)
	C.ctx_Loads_Set_duty(loads.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return loads.ctx.DSSError()
}

// Set kV rating for active Load. For 2 or more phases set Line-Line kV. Else actual kV across terminals.
func (loads *ILoads) Get_kV() (float64, error) {
	return (float64)(C.ctx_Loads_Get_kV(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_kV(value float64) error {
	C.ctx_Loads_Set_kV(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Set kW for active Load. Updates kvar based on present PF.
func (loads *ILoads) Get_kW() (float64, error) {
	return (float64)(C.ctx_Loads_Get_kW(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_kW(value float64) error {
	C.ctx_Loads_Set_kW(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Base load kva. Also defined kw and kvar or pf input, or load allocation by kwh or xfkva.
func (loads *ILoads) Get_kva() (float64, error) {
	return (float64)(C.ctx_Loads_Get_kva(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_kva(value float64) error {
	C.ctx_Loads_Set_kva(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Get/set kvar for active Load. If set, updates PF based on present kW.
func (loads *ILoads) Get_kvar() (float64, error) {
	return (float64)(C.ctx_Loads_Get_kvar(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_kvar(value float64) error {
	C.ctx_Loads_Set_kvar(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// kwh billed for this period. Can be used with Cfactor for load allocation.
func (loads *ILoads) Get_kwh() (float64, error) {
	return (float64)(C.ctx_Loads_Get_kwh(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_kwh(value float64) error {
	C.ctx_Loads_Set_kwh(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Length of kwh billing period for average demand calculation. Default 30.
func (loads *ILoads) Get_kwhdays() (float64, error) {
	return (float64)(C.ctx_Loads_Get_kwhdays(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_kwhdays(value float64) error {
	C.ctx_Loads_Set_kwhdays(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Percent of Load that is modeled as series R-L for harmonics studies
func (loads *ILoads) Get_pctSeriesRL() (float64, error) {
	return (float64)(C.ctx_Loads_Get_pctSeriesRL(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_pctSeriesRL(value float64) error {
	C.ctx_Loads_Set_pctSeriesRL(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Rated service transformer kVA for load allocation, using AllocationFactor. Affects kW, kvar, and pf.
func (loads *ILoads) Get_xfkVA() (float64, error) {
	return (float64)(C.ctx_Loads_Get_xfkVA(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_xfkVA(value float64) error {
	C.ctx_Loads_Set_xfkVA(loads.ctxPtr, (C.double)(value))
	return loads.ctx.DSSError()
}

// Name of the sensor monitoring this load.
func (loads *ILoads) Sensor() (string, error) {
	return C.GoString(C.ctx_Loads_Get_Sensor(loads.ctxPtr)), loads.ctx.DSSError()
}

// Number of phases
//
// (API Extension)
func (loads *ILoads) Get_Phases() (int32, error) {
	return (int32)(C.ctx_Loads_Get_Phases(loads.ctxPtr)), loads.ctx.DSSError()
}

func (loads *ILoads) Set_Phases(value int32) error {
	C.ctx_Loads_Set_Phases(loads.ctxPtr, (C.int32_t)(value))
	return loads.ctx.DSSError()
}

type IMeters struct {
	ICommonData
}

func (meters *IMeters) Init(ctx *DSSContextPtrs) {
	meters.InitCommon(ctx)
}

// Returns the list of all PCE within the area covered by the energy meter
func (meters *IMeters) ZonePCE() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Meters_Get_ZonePCE(meters.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return meters.ctx.GetStringArray(data, cnt)
}

// Array of strings with all Meter names in the circuit.
func (meters *IMeters) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Meters_Get_AllNames(meters.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return meters.ctx.GetStringArray(data, cnt)
}

// Number of Meter objects in active circuit.
func (meters *IMeters) Count() (int32, error) {
	return (int32)(C.ctx_Meters_Get_Count(meters.ctxPtr)), meters.ctx.DSSError()
}

// Sets the first Meter active. Returns 0 if no more.
func (meters *IMeters) First() (int32, error) {
	return (int32)(C.ctx_Meters_Get_First(meters.ctxPtr)), meters.ctx.DSSError()
}

// Sets the active Meter by Name.
func (meters *IMeters) Get_Name() (string, error) {
	result := C.GoString(C.ctx_Meters_Get_Name(meters.ctxPtr))
	return result, meters.ctx.DSSError()
}

// Gets the name of the active Meter.
func (meters *IMeters) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_Meters_Set_Name(meters.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return meters.ctx.DSSError()
}

// Sets the next Meter active. Returns 0 if no more.
func (meters *IMeters) Next() (int32, error) {
	return (int32)(C.ctx_Meters_Get_Next(meters.ctxPtr)), meters.ctx.DSSError()
}

// Get the index of the active Meter; index is 1-based: 1..count
func (meters *IMeters) Get_idx() (int32, error) {
	return (int32)(C.ctx_Meters_Get_idx(meters.ctxPtr)), meters.ctx.DSSError()
}

// Set the active Meter by index; index is 1-based: 1..count
func (meters *IMeters) Set_idx(value int32) error {
	C.ctx_Meters_Set_idx(meters.ctxPtr, (C.int32_t)(value))
	return meters.ctx.DSSError()
}

// Close All Demand Interval Files. Users are required to close the DI files at the end of a run.
func (meters *IMeters) CloseAllDIFiles() error {
	C.ctx_Meters_CloseAllDIFiles(meters.ctxPtr)
	return meters.ctx.DSSError()
}

// Calculate reliability indices
func (meters *IMeters) DoReliabilityCalc(AssumeRestoration bool) error {
	C.ctx_Meters_DoReliabilityCalc(meters.ctxPtr, ToUint16(AssumeRestoration))
	return meters.ctx.DSSError()
}

// Open Demand Interval (DI) files
func (meters *IMeters) OpenAllDIFiles() error {
	C.ctx_Meters_OpenAllDIFiles(meters.ctxPtr)
	return meters.ctx.DSSError()
}

// Resets registers of active meter.
func (meters *IMeters) Reset() error {
	C.ctx_Meters_Reset(meters.ctxPtr)
	return meters.ctx.DSSError()
}

// Resets registers of all meter objects.
func (meters *IMeters) ResetAll() error {
	C.ctx_Meters_ResetAll(meters.ctxPtr)
	return meters.ctx.DSSError()
}

// Forces active Meter to take a sample.
func (meters *IMeters) Sample() error {
	C.ctx_Meters_Sample(meters.ctxPtr)
	return meters.ctx.DSSError()
}

// Causes all EnergyMeter objects to take a sample at the present time.
func (meters *IMeters) SampleAll() error {
	C.ctx_Meters_SampleAll(meters.ctxPtr)
	return meters.ctx.DSSError()
}

// Saves meter register values.
func (meters *IMeters) Save() error {
	C.ctx_Meters_Save(meters.ctxPtr)
	return meters.ctx.DSSError()
}

// Save All EnergyMeter objects
func (meters *IMeters) SaveAll() error {
	C.ctx_Meters_SaveAll(meters.ctxPtr)
	return meters.ctx.DSSError()
}

func (meters *IMeters) SetActiveSection(SectIdx int32) error {
	C.ctx_Meters_SetActiveSection(meters.ctxPtr, (C.int32_t)(SectIdx))
	return meters.ctx.DSSError()
}

// Wide string list of all branches in zone of the active EnergyMeter object.
func (meters *IMeters) AllBranchesInZone() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Meters_Get_AllBranchesInZone(meters.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return meters.ctx.GetStringArray(data, cnt)
}

// Array of names of all zone end elements.
func (meters *IMeters) AllEndElements() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Meters_Get_AllEndElements(meters.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return meters.ctx.GetStringArray(data, cnt)
}

// Array of doubles: set the phase allocation factors for the active meter.
func (meters *IMeters) Get_AllocFactors() ([]float64, error) {
	C.ctx_Meters_Get_AllocFactors_GR(meters.ctxPtr)
	return meters.ctx.GetFloat64ArrayGR()
}

func (meters *IMeters) Set_AllocFactors(value []float64) error {
	C.ctx_Meters_Set_AllocFactors(meters.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return meters.ctx.DSSError()
}

// Average Repair time in this section of the meter zone
func (meters *IMeters) AvgRepairTime() (float64, error) {
	return (float64)(C.ctx_Meters_Get_AvgRepairTime(meters.ctxPtr)), meters.ctx.DSSError()
}

// Set the magnitude of the real part of the Calculated Current (normally determined by solution) for the Meter to force some behavior on Load Allocation
func (meters *IMeters) Get_CalcCurrent() ([]float64, error) {
	C.ctx_Meters_Get_CalcCurrent_GR(meters.ctxPtr)
	return meters.ctx.GetFloat64ArrayGR()
}

func (meters *IMeters) Set_CalcCurrent(value []float64) error {
	C.ctx_Meters_Set_CalcCurrent(meters.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return meters.ctx.DSSError()
}

// Number of branches in Active energymeter zone. (Same as sequencelist size)
func (meters *IMeters) CountBranches() (int32, error) {
	return (int32)(C.ctx_Meters_Get_CountBranches(meters.ctxPtr)), meters.ctx.DSSError()
}

// Number of zone end elements in the active meter zone.
func (meters *IMeters) CountEndElements() (int32, error) {
	return (int32)(C.ctx_Meters_Get_CountEndElements(meters.ctxPtr)), meters.ctx.DSSError()
}

// Total customer interruptions for this Meter zone based on reliability calcs.
func (meters *IMeters) CustInterrupts() (float64, error) {
	return (float64)(C.ctx_Meters_Get_CustInterrupts(meters.ctxPtr)), meters.ctx.DSSError()
}

// Global Flag in the DSS to indicate if Demand Interval (DI) files have been properly opened.
func (meters *IMeters) DIFilesAreOpen() (bool, error) {
	return (C.ctx_Meters_Get_DIFilesAreOpen(meters.ctxPtr) != 0), meters.ctx.DSSError()
}

// Sum of Fault Rate time Repair Hrs in this section of the meter zone
func (meters *IMeters) FaultRateXRepairHrs() (float64, error) {
	return (float64)(C.ctx_Meters_Get_FaultRateXRepairHrs(meters.ctxPtr)), meters.ctx.DSSError()
}

// Set Name of metered element
func (meters *IMeters) Get_MeteredElement() (string, error) {
	return C.GoString(C.ctx_Meters_Get_MeteredElement(meters.ctxPtr)), meters.ctx.DSSError()
}

func (meters *IMeters) Set_MeteredElement(value string) error {
	value_c := C.CString(value)
	C.ctx_Meters_Set_MeteredElement(meters.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return meters.ctx.DSSError()
}

// set Number of Metered Terminal
func (meters *IMeters) Get_MeteredTerminal() (int32, error) {
	return (int32)(C.ctx_Meters_Get_MeteredTerminal(meters.ctxPtr)), meters.ctx.DSSError()
}

func (meters *IMeters) Set_MeteredTerminal(value int32) error {
	C.ctx_Meters_Set_MeteredTerminal(meters.ctxPtr, (C.int32_t)(value))
	return meters.ctx.DSSError()
}

// Number of branches (lines) in this section
func (meters *IMeters) NumSectionBranches() (int32, error) {
	return (int32)(C.ctx_Meters_Get_NumSectionBranches(meters.ctxPtr)), meters.ctx.DSSError()
}

// Number of Customers in the active section.
func (meters *IMeters) NumSectionCustomers() (int32, error) {
	return (int32)(C.ctx_Meters_Get_NumSectionCustomers(meters.ctxPtr)), meters.ctx.DSSError()
}

// Number of feeder sections in this meter's zone
func (meters *IMeters) NumSections() (int32, error) {
	return (int32)(C.ctx_Meters_Get_NumSections(meters.ctxPtr)), meters.ctx.DSSError()
}

// Type of OCP device. 1=Fuse; 2=Recloser; 3=Relay
func (meters *IMeters) OCPDeviceType() (int32, error) {
	return (int32)(C.ctx_Meters_Get_OCPDeviceType(meters.ctxPtr)), meters.ctx.DSSError()
}

// Array of doubles to set values of Peak Current property
func (meters *IMeters) Get_Peakcurrent() ([]float64, error) {
	C.ctx_Meters_Get_Peakcurrent_GR(meters.ctxPtr)
	return meters.ctx.GetFloat64ArrayGR()
}

func (meters *IMeters) Set_Peakcurrent(value []float64) error {
	C.ctx_Meters_Set_Peakcurrent(meters.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return meters.ctx.DSSError()
}

// Array of strings containing the names of the registers.
func (meters *IMeters) RegisterNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Meters_Get_RegisterNames(meters.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return meters.ctx.GetStringArray(data, cnt)
}

// Array of all the values contained in the Meter registers for the active Meter.
func (meters *IMeters) RegisterValues() ([]float64, error) {
	C.ctx_Meters_Get_RegisterValues_GR(meters.ctxPtr)
	return meters.ctx.GetFloat64ArrayGR()
}

// SAIDI for this meter's zone. Execute DoReliabilityCalc first.
func (meters *IMeters) SAIDI() (float64, error) {
	return (float64)(C.ctx_Meters_Get_SAIDI(meters.ctxPtr)), meters.ctx.DSSError()
}

// Returns SAIFI for this meter's Zone. Execute Reliability Calc method first.
func (meters *IMeters) SAIFI() (float64, error) {
	return (float64)(C.ctx_Meters_Get_SAIFI(meters.ctxPtr)), meters.ctx.DSSError()
}

// SAIFI based on kW rather than number of customers. Get after reliability calcs.
func (meters *IMeters) SAIFIKW() (float64, error) {
	return (float64)(C.ctx_Meters_Get_SAIFIKW(meters.ctxPtr)), meters.ctx.DSSError()
}

// SequenceIndex of the branch at the head of this section
func (meters *IMeters) SectSeqIdx() (int32, error) {
	return (int32)(C.ctx_Meters_Get_SectSeqIdx(meters.ctxPtr)), meters.ctx.DSSError()
}

// Total Customers downline from this section
func (meters *IMeters) SectTotalCust() (int32, error) {
	return (int32)(C.ctx_Meters_Get_SectTotalCust(meters.ctxPtr)), meters.ctx.DSSError()
}

// Size of Sequence List
func (meters *IMeters) SeqListSize() (int32, error) {
	return (int32)(C.ctx_Meters_Get_SeqListSize(meters.ctxPtr)), meters.ctx.DSSError()
}

// Get/set Index into Meter's SequenceList that contains branch pointers in lexical order. Earlier index guaranteed to be upline from later index. Sets PDelement active.
func (meters *IMeters) Get_SequenceIndex() (int32, error) {
	return (int32)(C.ctx_Meters_Get_SequenceIndex(meters.ctxPtr)), meters.ctx.DSSError()
}

func (meters *IMeters) Set_SequenceIndex(value int32) error {
	C.ctx_Meters_Set_SequenceIndex(meters.ctxPtr, (C.int32_t)(value))
	return meters.ctx.DSSError()
}

// Sum of the branch fault rates in this section of the meter's zone
func (meters *IMeters) SumBranchFltRates() (float64, error) {
	return (float64)(C.ctx_Meters_Get_SumBranchFltRates(meters.ctxPtr)), meters.ctx.DSSError()
}

// Total Number of customers in this zone (downline from the EnergyMeter)
func (meters *IMeters) TotalCustomers() (int32, error) {
	return (int32)(C.ctx_Meters_Get_TotalCustomers(meters.ctxPtr)), meters.ctx.DSSError()
}

// Totals of all registers of all meters
func (meters *IMeters) Totals() ([]float64, error) {
	C.ctx_Meters_Get_Totals_GR(meters.ctxPtr)
	return meters.ctx.GetFloat64ArrayGR()
}

type IPDElements struct {
	ICommonData
}

func (pdelements *IPDElements) Init(ctx *DSSContextPtrs) {
	pdelements.InitCommon(ctx)
}

// accummulated failure rate for this branch on downline
func (pdelements *IPDElements) AccumulatedL() (float64, error) {
	return (float64)(C.ctx_PDElements_Get_AccumulatedL(pdelements.ctxPtr)), pdelements.ctx.DSSError()
}

// Number of PD elements (including disabled elements)
func (pdelements *IPDElements) Count() (int32, error) {
	return (int32)(C.ctx_PDElements_Get_Count(pdelements.ctxPtr)), pdelements.ctx.DSSError()
}

// Get/Set Number of failures per year.
// For LINE elements: Number of failures per unit length per year.
func (pdelements *IPDElements) Get_FaultRate() (float64, error) {
	return (float64)(C.ctx_PDElements_Get_FaultRate(pdelements.ctxPtr)), pdelements.ctx.DSSError()
}

func (pdelements *IPDElements) Set_FaultRate(value float64) error {
	C.ctx_PDElements_Set_FaultRate(pdelements.ctxPtr, (C.double)(value))
	return pdelements.ctx.DSSError()
}

// Set the first enabled PD element to be the active element.
// Returns 0 if none found.
func (pdelements *IPDElements) First() (int32, error) {
	return (int32)(C.ctx_PDElements_Get_First(pdelements.ctxPtr)), pdelements.ctx.DSSError()
}

// Number of the terminal of active PD element that is on the "from"
// side. This is set after the meter zone is determined.
func (pdelements *IPDElements) FromTerminal() (int32, error) {
	return (int32)(C.ctx_PDElements_Get_FromTerminal(pdelements.ctxPtr)), pdelements.ctx.DSSError()
}

// Boolean indicating of PD element should be treated as a shunt
// element rather than a series element. Applies to Capacitor and Reactor
// elements in particular.
func (pdelements *IPDElements) IsShunt() (bool, error) {
	return (C.ctx_PDElements_Get_IsShunt(pdelements.ctxPtr) != 0), pdelements.ctx.DSSError()
}

// Failure rate for this branch. Faults per year including length of line.
func (pdelements *IPDElements) Lambda() (float64, error) {
	return (float64)(C.ctx_PDElements_Get_Lambda(pdelements.ctxPtr)), pdelements.ctx.DSSError()
}

// Get/Set name of active PD Element. Returns null string if active element
// is not PDElement type.
func (pdelements *IPDElements) Get_Name() (string, error) {
	return C.GoString(C.ctx_PDElements_Get_Name(pdelements.ctxPtr)), pdelements.ctx.DSSError()
}

func (pdelements *IPDElements) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_PDElements_Set_Name(pdelements.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return pdelements.ctx.DSSError()
}

// Advance to the next PD element in the circuit. Enabled elements
// only. Returns 0 when no more elements.
func (pdelements *IPDElements) Next() (int32, error) {
	return (int32)(C.ctx_PDElements_Get_Next(pdelements.ctxPtr)), pdelements.ctx.DSSError()
}

// Number of customers, this branch
func (pdelements *IPDElements) Numcustomers() (int32, error) {
	return (int32)(C.ctx_PDElements_Get_Numcustomers(pdelements.ctxPtr)), pdelements.ctx.DSSError()
}

// Sets the parent PD element to be the active circuit element.
// Returns 0 if no more elements upline.
func (pdelements *IPDElements) ParentPDElement() (int32, error) {
	return (int32)(C.ctx_PDElements_Get_ParentPDElement(pdelements.ctxPtr)), pdelements.ctx.DSSError()
}

// Average repair time for this element in hours
func (pdelements *IPDElements) Get_RepairTime() (float64, error) {
	return (float64)(C.ctx_PDElements_Get_RepairTime(pdelements.ctxPtr)), pdelements.ctx.DSSError()
}

func (pdelements *IPDElements) Set_RepairTime(value float64) error {
	C.ctx_PDElements_Set_RepairTime(pdelements.ctxPtr, (C.double)(value))
	return pdelements.ctx.DSSError()
}

// Integer ID of the feeder section that this PDElement branch is part of
func (pdelements *IPDElements) SectionID() (int32, error) {
	return (int32)(C.ctx_PDElements_Get_SectionID(pdelements.ctxPtr)), pdelements.ctx.DSSError()
}

// Total miles of line from this element to the end of the zone. For recloser siting algorithm.
func (pdelements *IPDElements) TotalMiles() (float64, error) {
	return (float64)(C.ctx_PDElements_Get_TotalMiles(pdelements.ctxPtr)), pdelements.ctx.DSSError()
}

// Total number of customers from this branch to the end of the zone
func (pdelements *IPDElements) Totalcustomers() (int32, error) {
	return (int32)(C.ctx_PDElements_Get_Totalcustomers(pdelements.ctxPtr)), pdelements.ctx.DSSError()
}

// Get/Set percent of faults that are permanent (require repair). Otherwise, fault is assumed to be transient/temporary.
func (pdelements *IPDElements) Get_pctPermanent() (float64, error) {
	return (float64)(C.ctx_PDElements_Get_pctPermanent(pdelements.ctxPtr)), pdelements.ctx.DSSError()
}

func (pdelements *IPDElements) Set_pctPermanent(value float64) error {
	C.ctx_PDElements_Set_pctPermanent(pdelements.ctxPtr, (C.double)(value))
	return pdelements.ctx.DSSError()
}

// Array of strings consisting of all PD element names.
//
// (API Extension)
func (pdelements *IPDElements) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_PDElements_Get_AllNames(pdelements.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return pdelements.ctx.GetStringArray(data, cnt)
}

// Array of doubles with the maximum current across the conductors, for each PD
// element.
//
// By default, only the *first terminal* is used for the maximum current, matching
// the behavior of the "export capacity" command. Pass `true` to
// force the analysis to all terminals.
//
// See also:
// https://sourceforge.net/p/electricdss/discussion/beginners/thread/da5b93ca/
//
// (API Extension)
func (pdelements *IPDElements) AllMaxCurrents(AllNodes bool) ([]float64, error) {
	C.ctx_PDElements_Get_AllMaxCurrents_GR(pdelements.ctxPtr, ToUint16(AllNodes))
	return pdelements.ctx.GetFloat64ArrayGR()
}

// Array of doubles with the maximum current across the conductors as a percentage
// of the Normal Ampere Rating, for each PD element.
//
// By default, only the *first terminal* is used for the maximum current, matching
// the behavior of the "export capacity" command. Pass `true` to
// force the analysis to all terminals.
//
// See also:
// https://sourceforge.net/p/electricdss/discussion/beginners/thread/da5b93ca/
//
// (API Extension)
func (pdelements *IPDElements) AllPctNorm(AllNodes bool) ([]float64, error) {
	C.ctx_PDElements_Get_AllPctNorm_GR(pdelements.ctxPtr, ToUint16(AllNodes))
	return pdelements.ctx.GetFloat64ArrayGR()
}

// Array of doubles with the maximum current across the conductors as a percentage
// of the Emergency Ampere Rating, for each PD element.
//
// By default, only the *first terminal* is used for the maximum current, matching
// the behavior of the "export capacity" command. Pass `true` to
// force the analysis to all terminals.
//
// See also:
// https://sourceforge.net/p/electricdss/discussion/beginners/thread/da5b93ca/
//
// (API Extension)
func (pdelements *IPDElements) AllPctEmerg(AllNodes bool) ([]float64, error) {
	C.ctx_PDElements_Get_AllPctEmerg_GR(pdelements.ctxPtr, ToUint16(AllNodes))
	return pdelements.ctx.GetFloat64ArrayGR()
}

// Complex array of currents for all conductors, all terminals, for each PD element.
//
// (API Extension)
func (pdelements *IPDElements) AllCurrents() ([]complex128, error) {
	C.ctx_PDElements_Get_AllCurrents_GR(pdelements.ctxPtr)
	return pdelements.ctx.GetComplexArrayGR()
}

// Complex array (magnitude and angle format) of currents for all conductors, all terminals, for each PD element.
//
// (API Extension)
func (pdelements *IPDElements) AllCurrentsMagAng() ([]float64, error) {
	C.ctx_PDElements_Get_AllCurrentsMagAng_GR(pdelements.ctxPtr)
	return pdelements.ctx.GetFloat64ArrayGR()
}

// Complex double array of Sequence Currents for all conductors of all terminals, for each PD elements.
//
// (API Extension)
func (pdelements *IPDElements) AllCplxSeqCurrents() ([]complex128, error) {
	C.ctx_PDElements_Get_AllCplxSeqCurrents_GR(pdelements.ctxPtr)
	return pdelements.ctx.GetComplexArrayGR()
}

// Double array of the symmetrical component currents (magnitudes only) into each 3-phase terminal, for each PD element.
//
// (API Extension)
func (pdelements *IPDElements) AllSeqCurrents() ([]float64, error) {
	C.ctx_PDElements_Get_AllSeqCurrents_GR(pdelements.ctxPtr)
	return pdelements.ctx.GetFloat64ArrayGR()
}

// Complex array of powers into each conductor of each terminal, for each PD element.
//
// (API Extension)
func (pdelements *IPDElements) AllPowers() ([]complex128, error) {
	C.ctx_PDElements_Get_AllPowers_GR(pdelements.ctxPtr)
	return pdelements.ctx.GetComplexArrayGR()
}

// Complex array of sequence powers into each 3-phase teminal, for each PD element
//
// (API Extension)
func (pdelements *IPDElements) AllSeqPowers() ([]complex128, error) {
	C.ctx_PDElements_Get_AllSeqPowers_GR(pdelements.ctxPtr)
	return pdelements.ctx.GetComplexArrayGR()
}

// Integer array listing the number of phases of all PD elements
//
// (API Extension)
func (pdelements *IPDElements) AllNumPhases() ([]int32, error) {
	C.ctx_PDElements_Get_AllNumPhases_GR(pdelements.ctxPtr)
	return pdelements.ctx.GetInt32ArrayGR()
}

// Integer array listing the number of conductors of all PD elements
//
// (API Extension)
func (pdelements *IPDElements) AllNumConductors() ([]int32, error) {
	C.ctx_PDElements_Get_AllNumConductors_GR(pdelements.ctxPtr)
	return pdelements.ctx.GetInt32ArrayGR()
}

// Integer array listing the number of terminals of all PD elements
//
// (API Extension)
func (pdelements *IPDElements) AllNumTerminals() ([]int32, error) {
	C.ctx_PDElements_Get_AllNumTerminals_GR(pdelements.ctxPtr)
	return pdelements.ctx.GetInt32ArrayGR()
}

type IPVSystems struct {
	ICommonData
}

func (pvsystems *IPVSystems) Init(ctx *DSSContextPtrs) {
	pvsystems.InitCommon(ctx)
}

// Array of strings with all PVSystem names in the circuit.
func (pvsystems *IPVSystems) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_PVSystems_Get_AllNames(pvsystems.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return pvsystems.ctx.GetStringArray(data, cnt)
}

// Number of PVSystem objects in active circuit.
func (pvsystems *IPVSystems) Count() (int32, error) {
	return (int32)(C.ctx_PVSystems_Get_Count(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

// Sets the first PVSystem active. Returns 0 if no more.
func (pvsystems *IPVSystems) First() (int32, error) {
	return (int32)(C.ctx_PVSystems_Get_First(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

// Sets the active PVSystem by Name.
func (pvsystems *IPVSystems) Get_Name() (string, error) {
	result := C.GoString(C.ctx_PVSystems_Get_Name(pvsystems.ctxPtr))
	return result, pvsystems.ctx.DSSError()
}

// Gets the name of the active PVSystem.
func (pvsystems *IPVSystems) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_PVSystems_Set_Name(pvsystems.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return pvsystems.ctx.DSSError()
}

// Sets the next PVSystem active. Returns 0 if no more.
func (pvsystems *IPVSystems) Next() (int32, error) {
	return (int32)(C.ctx_PVSystems_Get_Next(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

// Get the index of the active PVSystem; index is 1-based: 1..count
func (pvsystems *IPVSystems) Get_idx() (int32, error) {
	return (int32)(C.ctx_PVSystems_Get_idx(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

// Set the active PVSystem by index; index is 1-based: 1..count
func (pvsystems *IPVSystems) Set_idx(value int32) error {
	C.ctx_PVSystems_Set_idx(pvsystems.ctxPtr, (C.int32_t)(value))
	return pvsystems.ctx.DSSError()
}

// Get/set the present value of the Irradiance property in kW/m²
func (pvsystems *IPVSystems) Get_Irradiance() (float64, error) {
	return (float64)(C.ctx_PVSystems_Get_Irradiance(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

func (pvsystems *IPVSystems) Set_Irradiance(value float64) error {
	C.ctx_PVSystems_Set_Irradiance(pvsystems.ctxPtr, (C.double)(value))
	return pvsystems.ctx.DSSError()
}

// Get/set the power factor for the active PVSystem
func (pvsystems *IPVSystems) Get_PF() (float64, error) {
	return (float64)(C.ctx_PVSystems_Get_PF(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

func (pvsystems *IPVSystems) Set_PF(value float64) error {
	C.ctx_PVSystems_Set_PF(pvsystems.ctxPtr, (C.double)(value))
	return pvsystems.ctx.DSSError()
}

// Array of PVSYSTEM energy meter register names
func (pvsystems *IPVSystems) RegisterNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_PVSystems_Get_RegisterNames(pvsystems.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return pvsystems.ctx.GetStringArray(data, cnt)
}

// Array of doubles containing values in PVSystem registers.
func (pvsystems *IPVSystems) RegisterValues() ([]float64, error) {
	C.ctx_PVSystems_Get_RegisterValues_GR(pvsystems.ctxPtr)
	return pvsystems.ctx.GetFloat64ArrayGR()
}

// Get/set Rated kVA of the PVSystem
func (pvsystems *IPVSystems) Get_kVArated() (float64, error) {
	return (float64)(C.ctx_PVSystems_Get_kVArated(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

func (pvsystems *IPVSystems) Set_kVArated(value float64) error {
	C.ctx_PVSystems_Set_kVArated(pvsystems.ctxPtr, (C.double)(value))
	return pvsystems.ctx.DSSError()
}

// Get kW output
func (pvsystems *IPVSystems) Get_kW() (float64, error) {
	return (float64)(C.ctx_PVSystems_Get_kW(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

// Get/set kvar output value
func (pvsystems *IPVSystems) Get_kvar() (float64, error) {
	return (float64)(C.ctx_PVSystems_Get_kvar(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

func (pvsystems *IPVSystems) Set_kvar(value float64) error {
	C.ctx_PVSystems_Set_kvar(pvsystems.ctxPtr, (C.double)(value))
	return pvsystems.ctx.DSSError()
}

// Name of the dispatch shape to use for daily simulations. Must be previously
// defined as a Loadshape object of 24 hrs, typically. In the default dispatch
// mode, the PVSystem element uses this loadshape to trigger State changes.
//
// (API Extension)
func (pvsystems *IPVSystems) Get_daily() (string, error) {
	return C.GoString(C.ctx_PVSystems_Get_daily(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

func (pvsystems *IPVSystems) Set_daily(value string) error {
	value_c := C.CString(value)
	C.ctx_PVSystems_Set_daily(pvsystems.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return pvsystems.ctx.DSSError()
}

// Name of the load shape to use for duty cycle dispatch simulations such as
// for solar ramp rate studies. Must be previously defined as a Loadshape
// object. Typically would have time intervals of 1-5 seconds.
//
// (API Extension)
func (pvsystems *IPVSystems) Get_duty() (string, error) {
	return C.GoString(C.ctx_PVSystems_Get_duty(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

func (pvsystems *IPVSystems) Set_duty(value string) error {
	value_c := C.CString(value)
	C.ctx_PVSystems_Set_duty(pvsystems.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return pvsystems.ctx.DSSError()
}

// Dispatch shape to use for yearly simulations. Must be previously defined
// as a Loadshape object. If this is not specified, the Daily dispatch shape,
// if any, is repeated during Yearly solution modes. In the default dispatch
// mode, the PVSystem element uses this loadshape to trigger State changes.
//
// (API Extension)
func (pvsystems *IPVSystems) Get_yearly() (string, error) {
	return C.GoString(C.ctx_PVSystems_Get_yearly(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

func (pvsystems *IPVSystems) Set_yearly(value string) error {
	value_c := C.CString(value)
	C.ctx_PVSystems_Set_yearly(pvsystems.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return pvsystems.ctx.DSSError()
}

// Temperature shape to use for daily simulations. Must be previously defined
// as a TShape object of 24 hrs, typically. The PVSystem element uses this
// TShape to determine the Pmpp from the Pmpp vs T curve. Units must agree
// with the Pmpp vs T curve.
//
// (API Extension)
func (pvsystems *IPVSystems) Get_Tdaily() (string, error) {
	return C.GoString(C.ctx_PVSystems_Get_Tdaily(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

func (pvsystems *IPVSystems) Set_Tdaily(value string) error {
	value_c := C.CString(value)
	C.ctx_PVSystems_Set_Tdaily(pvsystems.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return pvsystems.ctx.DSSError()
}

// Temperature shape to use for duty cycle dispatch simulations such as for
// solar ramp rate studies. Must be previously defined as a TShape object.
// Typically would have time intervals of 1-5 seconds. Designate the number
// of points to solve using the Set Number=xxxx command. If there are fewer
// points in the actual shape, the shape is assumed to repeat. The PVSystem
// model uses this TShape to determine the Pmpp from the Pmpp vs T curve.
// Units must agree with the Pmpp vs T curve.
//
// (API Extension)
func (pvsystems *IPVSystems) Get_Tduty() (string, error) {
	return C.GoString(C.ctx_PVSystems_Get_Tduty(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

func (pvsystems *IPVSystems) Set_Tduty(value string) error {
	value_c := C.CString(value)
	C.ctx_PVSystems_Set_Tduty(pvsystems.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return pvsystems.ctx.DSSError()
}

// Temperature shape to use for yearly simulations. Must be previously defined
// as a TShape object. If this is not specified, the Daily dispatch shape, if
// any, is repeated during Yearly solution modes. The PVSystem element uses
// this TShape to determine the Pmpp from the Pmpp vs T curve. Units must
// agree with the Pmpp vs T curve.
//
// (API Extension)
func (pvsystems *IPVSystems) Get_Tyearly() (string, error) {
	return C.GoString(C.ctx_PVSystems_Get_Tyearly(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

func (pvsystems *IPVSystems) Set_Tyearly(value string) error {
	value_c := C.CString(value)
	C.ctx_PVSystems_Set_Tyearly(pvsystems.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return pvsystems.ctx.DSSError()
}

// Returns the current irradiance value for the active PVSystem. Use it to
// know what's the current irradiance value for the PV during a simulation.
func (pvsystems *IPVSystems) IrradianceNow() (float64, error) {
	return (float64)(C.ctx_PVSystems_Get_IrradianceNow(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

// Gets/sets the rated max power of the PV array for 1.0 kW/sq-m irradiance
// and a user-selected array temperature of the active PVSystem.
func (pvsystems *IPVSystems) Get_Pmpp() (float64, error) {
	return (float64)(C.ctx_PVSystems_Get_Pmpp(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

func (pvsystems *IPVSystems) Set_Pmpp(value float64) error {
	C.ctx_PVSystems_Set_Pmpp(pvsystems.ctxPtr, (C.double)(value))
	return pvsystems.ctx.DSSError()
}

// Name of the sensor monitoring this element.
func (pvsystems *IPVSystems) Sensor() (string, error) {
	return C.GoString(C.ctx_PVSystems_Get_Sensor(pvsystems.ctxPtr)), pvsystems.ctx.DSSError()
}

type IReactors struct {
	ICommonData
}

func (reactors *IReactors) Init(ctx *DSSContextPtrs) {
	reactors.InitCommon(ctx)
}

// Array of strings with all Reactor names in the circuit.
func (reactors *IReactors) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Reactors_Get_AllNames(reactors.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return reactors.ctx.GetStringArray(data, cnt)
}

// Number of Reactor objects in active circuit.
func (reactors *IReactors) Count() (int32, error) {
	return (int32)(C.ctx_Reactors_Get_Count(reactors.ctxPtr)), reactors.ctx.DSSError()
}

// Sets the first Reactor active. Returns 0 if no more.
func (reactors *IReactors) First() (int32, error) {
	return (int32)(C.ctx_Reactors_Get_First(reactors.ctxPtr)), reactors.ctx.DSSError()
}

// Sets the active Reactor by Name.
func (reactors *IReactors) Get_Name() (string, error) {
	result := C.GoString(C.ctx_Reactors_Get_Name(reactors.ctxPtr))
	return result, reactors.ctx.DSSError()
}

// Gets the name of the active Reactor.
func (reactors *IReactors) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_Reactors_Set_Name(reactors.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return reactors.ctx.DSSError()
}

// Sets the next Reactor active. Returns 0 if no more.
func (reactors *IReactors) Next() (int32, error) {
	return (int32)(C.ctx_Reactors_Get_Next(reactors.ctxPtr)), reactors.ctx.DSSError()
}

// Get the index of the active Reactor; index is 1-based: 1..count
func (reactors *IReactors) Get_idx() (int32, error) {
	return (int32)(C.ctx_Reactors_Get_idx(reactors.ctxPtr)), reactors.ctx.DSSError()
}

// Set the active Reactor by index; index is 1-based: 1..count
func (reactors *IReactors) Set_idx(value int32) error {
	C.ctx_Reactors_Set_idx(reactors.ctxPtr, (C.int32_t)(value))
	return reactors.ctx.DSSError()
}

// How the reactor data was provided: 1=kvar, 2=R+jX, 3=R and X matrices, 4=sym components.
// Depending on this value, only some properties are filled or make sense in the context.
func (reactors *IReactors) SpecType() (int32, error) {
	return (int32)(C.ctx_Reactors_Get_SpecType(reactors.ctxPtr)), reactors.ctx.DSSError()
}

// Delta connection or wye?
func (reactors *IReactors) Get_IsDelta() (bool, error) {
	return (C.ctx_Reactors_Get_IsDelta(reactors.ctxPtr) != 0), reactors.ctx.DSSError()
}

func (reactors *IReactors) Set_IsDelta(value bool) error {
	C.ctx_Reactors_Set_IsDelta(reactors.ctxPtr, ToUint16(value))
	return reactors.ctx.DSSError()
}

// Indicates whether Rmatrix and Xmatrix are to be considered in parallel.
func (reactors *IReactors) Get_Parallel() (bool, error) {
	return (C.ctx_Reactors_Get_Parallel(reactors.ctxPtr) != 0), reactors.ctx.DSSError()
}

func (reactors *IReactors) Set_Parallel(value bool) error {
	C.ctx_Reactors_Set_Parallel(reactors.ctxPtr, ToUint16(value))
	return reactors.ctx.DSSError()
}

// Inductance, mH. Alternate way to define the reactance, X, property.
func (reactors *IReactors) Get_LmH() (float64, error) {
	return (float64)(C.ctx_Reactors_Get_LmH(reactors.ctxPtr)), reactors.ctx.DSSError()
}

func (reactors *IReactors) Set_LmH(value float64) error {
	C.ctx_Reactors_Set_LmH(reactors.ctxPtr, (C.double)(value))
	return reactors.ctx.DSSError()
}

// For 2, 3-phase, kV phase-phase. Otherwise specify actual coil rating.
func (reactors *IReactors) Get_kV() (float64, error) {
	return (float64)(C.ctx_Reactors_Get_kV(reactors.ctxPtr)), reactors.ctx.DSSError()
}

func (reactors *IReactors) Set_kV(value float64) error {
	C.ctx_Reactors_Set_kV(reactors.ctxPtr, (C.double)(value))
	return reactors.ctx.DSSError()
}

// Total kvar, all phases.  Evenly divided among phases. Only determines X. Specify R separately
func (reactors *IReactors) Get_kvar() (float64, error) {
	return (float64)(C.ctx_Reactors_Get_kvar(reactors.ctxPtr)), reactors.ctx.DSSError()
}

func (reactors *IReactors) Set_kvar(value float64) error {
	C.ctx_Reactors_Set_kvar(reactors.ctxPtr, (C.double)(value))
	return reactors.ctx.DSSError()
}

// Number of phases.
func (reactors *IReactors) Get_Phases() (int32, error) {
	return (int32)(C.ctx_Reactors_Get_Phases(reactors.ctxPtr)), reactors.ctx.DSSError()
}

func (reactors *IReactors) Set_Phases(value int32) error {
	C.ctx_Reactors_Set_Phases(reactors.ctxPtr, (C.int32_t)(value))
	return reactors.ctx.DSSError()
}

// Name of first bus.
// Bus2 property will default to this bus, node 0, unless previously specified.
// Only Bus1 need be specified for a Yg shunt reactor.
func (reactors *IReactors) Get_Bus1() (string, error) {
	return C.GoString(C.ctx_Reactors_Get_Bus1(reactors.ctxPtr)), reactors.ctx.DSSError()
}

func (reactors *IReactors) Set_Bus1(value string) error {
	value_c := C.CString(value)
	C.ctx_Reactors_Set_Bus1(reactors.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return reactors.ctx.DSSError()
}

// Name of 2nd bus. Defaults to all phases connected to first bus, node 0, (Shunt Wye Connection) except when Bus2 is specifically defined.
// Not necessary to specify for delta (LL) connection
func (reactors *IReactors) Get_Bus2() (string, error) {
	return C.GoString(C.ctx_Reactors_Get_Bus2(reactors.ctxPtr)), reactors.ctx.DSSError()
}

func (reactors *IReactors) Set_Bus2(value string) error {
	value_c := C.CString(value)
	C.ctx_Reactors_Set_Bus2(reactors.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return reactors.ctx.DSSError()
}

// Name of XYCurve object, previously defined, describing per-unit variation of phase inductance, L=X/w, vs. frequency. Applies to reactance specified by X, LmH, Z, or kvar property. L generally decreases somewhat with frequency above the base frequency, approaching a limit at a few kHz.
func (reactors *IReactors) Get_LCurve() (string, error) {
	return C.GoString(C.ctx_Reactors_Get_LCurve(reactors.ctxPtr)), reactors.ctx.DSSError()
}

func (reactors *IReactors) Set_LCurve(value string) error {
	value_c := C.CString(value)
	C.ctx_Reactors_Set_LCurve(reactors.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return reactors.ctx.DSSError()
}

// Name of XYCurve object, previously defined, describing per-unit variation of phase resistance, R, vs. frequency. Applies to resistance specified by R or Z property. If actual values are not known, R often increases by approximately the square root of frequency.
func (reactors *IReactors) Get_RCurve() (string, error) {
	return C.GoString(C.ctx_Reactors_Get_RCurve(reactors.ctxPtr)), reactors.ctx.DSSError()
}

func (reactors *IReactors) Set_RCurve(value string) error {
	value_c := C.CString(value)
	C.ctx_Reactors_Set_RCurve(reactors.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return reactors.ctx.DSSError()
}

// Resistance (in series with reactance), each phase, ohms. This property applies to REACTOR specified by either kvar or X. See also help on Z.
func (reactors *IReactors) Get_R() (float64, error) {
	return (float64)(C.ctx_Reactors_Get_R(reactors.ctxPtr)), reactors.ctx.DSSError()
}

func (reactors *IReactors) Set_R(value float64) error {
	C.ctx_Reactors_Set_R(reactors.ctxPtr, (C.double)(value))
	return reactors.ctx.DSSError()
}

// Reactance, each phase, ohms at base frequency. See also help on Z and LmH properties.
func (reactors *IReactors) Get_X() (float64, error) {
	return (float64)(C.ctx_Reactors_Get_X(reactors.ctxPtr)), reactors.ctx.DSSError()
}

func (reactors *IReactors) Set_X(value float64) error {
	C.ctx_Reactors_Set_X(reactors.ctxPtr, (C.double)(value))
	return reactors.ctx.DSSError()
}

// Resistance in parallel with R and X (the entire branch). Assumed infinite if not specified.
func (reactors *IReactors) Get_Rp() (float64, error) {
	return (float64)(C.ctx_Reactors_Get_Rp(reactors.ctxPtr)), reactors.ctx.DSSError()
}

func (reactors *IReactors) Set_Rp(value float64) error {
	C.ctx_Reactors_Set_Rp(reactors.ctxPtr, (C.double)(value))
	return reactors.ctx.DSSError()
}

// Resistance matrix, ohms at base frequency. Order of the matrix is the number of phases. Mutually exclusive to specifying parameters by kvar or X.
func (reactors *IReactors) Get_Rmatrix() ([]float64, error) {
	C.ctx_Reactors_Get_Rmatrix_GR(reactors.ctxPtr)
	return reactors.ctx.GetFloat64ArrayGR()
}

func (reactors *IReactors) Set_Rmatrix(value []float64) error {
	C.ctx_Reactors_Set_Rmatrix(reactors.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return reactors.ctx.DSSError()
}

// Reactance matrix, ohms at base frequency. Order of the matrix is the number of phases. Mutually exclusive to specifying parameters by kvar or X.
func (reactors *IReactors) Get_Xmatrix() ([]float64, error) {
	C.ctx_Reactors_Get_Xmatrix_GR(reactors.ctxPtr)
	return reactors.ctx.GetFloat64ArrayGR()
}

func (reactors *IReactors) Set_Xmatrix(value []float64) error {
	C.ctx_Reactors_Set_Xmatrix(reactors.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return reactors.ctx.DSSError()
}

// Alternative way of defining R and X properties. Enter a 2-element array representing R +jX in ohms.
func (reactors *IReactors) Get_Z() (complex128, error) {
	C.ctx_Reactors_Get_Z_GR(reactors.ctxPtr)
	return reactors.ctx.GetComplexSimpleGR()
}

func (reactors *IReactors) Set_Z(value complex128) error {
	C.ctx_Reactors_Set_Z(reactors.ctxPtr, (*C.double)((unsafe.Pointer)(&value)), (C.int32_t)(2))
	return reactors.ctx.DSSError()
}

// Positive-sequence impedance, ohms, as a 2-element array representing a complex number.
//
// If defined, Z1, Z2, and Z0 are used to define the impedance matrix of the REACTOR.
//
// Z1 MUST BE DEFINED TO USE THIS OPTION FOR DEFINING THE MATRIX.
//
// Side Effect: Sets Z2 and Z0 to same values unless they were previously defined.
func (reactors *IReactors) Get_Z1() (complex128, error) {
	C.ctx_Reactors_Get_Z1_GR(reactors.ctxPtr)
	return reactors.ctx.GetComplexSimpleGR()
}

func (reactors *IReactors) Set_Z1(value complex128) error {
	C.ctx_Reactors_Set_Z1(reactors.ctxPtr, (*C.double)((unsafe.Pointer)(&value)), (C.int32_t)(2))
	return reactors.ctx.DSSError()
}

// Negative-sequence impedance, ohms, as a 2-element array representing a complex number.
//
// Used to define the impedance matrix of the REACTOR if Z1 is also specified.
//
// Note: Z2 defaults to Z1 if it is not specifically defined. If Z2 is not equal to Z1, the impedance matrix is asymmetrical.
func (reactors *IReactors) Get_Z2() (complex128, error) {
	C.ctx_Reactors_Get_Z2_GR(reactors.ctxPtr)
	return reactors.ctx.GetComplexSimpleGR()
}

func (reactors *IReactors) Set_Z2(value complex128) error {
	C.ctx_Reactors_Set_Z2(reactors.ctxPtr, (*C.double)((unsafe.Pointer)(&value)), (C.int32_t)(2))
	return reactors.ctx.DSSError()
}

// Zero-sequence impedance, ohms, as a 2-element array representing a complex number.
//
// Used to define the impedance matrix of the REACTOR if Z1 is also specified.
//
// Note: Z0 defaults to Z1 if it is not specifically defined.
func (reactors *IReactors) Get_Z0() (complex128, error) {
	C.ctx_Reactors_Get_Z0_GR(reactors.ctxPtr)
	return reactors.ctx.GetComplexSimpleGR()
}

func (reactors *IReactors) Set_Z0(value complex128) error {
	C.ctx_Reactors_Set_Z0(reactors.ctxPtr, (*C.double)((unsafe.Pointer)(&value)), (C.int32_t)(2))
	return reactors.ctx.DSSError()
}

type IReclosers struct {
	ICommonData
}

func (reclosers *IReclosers) Init(ctx *DSSContextPtrs) {
	reclosers.InitCommon(ctx)
}

// Array of strings with all Recloser names in the circuit.
func (reclosers *IReclosers) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Reclosers_Get_AllNames(reclosers.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return reclosers.ctx.GetStringArray(data, cnt)
}

// Number of Recloser objects in active circuit.
func (reclosers *IReclosers) Count() (int32, error) {
	return (int32)(C.ctx_Reclosers_Get_Count(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

// Sets the first Recloser active. Returns 0 if no more.
func (reclosers *IReclosers) First() (int32, error) {
	return (int32)(C.ctx_Reclosers_Get_First(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

// Sets the active Recloser by Name.
func (reclosers *IReclosers) Get_Name() (string, error) {
	result := C.GoString(C.ctx_Reclosers_Get_Name(reclosers.ctxPtr))
	return result, reclosers.ctx.DSSError()
}

// Gets the name of the active Recloser.
func (reclosers *IReclosers) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_Reclosers_Set_Name(reclosers.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return reclosers.ctx.DSSError()
}

// Sets the next Recloser active. Returns 0 if no more.
func (reclosers *IReclosers) Next() (int32, error) {
	return (int32)(C.ctx_Reclosers_Get_Next(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

// Get the index of the active Recloser; index is 1-based: 1..count
func (reclosers *IReclosers) Get_idx() (int32, error) {
	return (int32)(C.ctx_Reclosers_Get_idx(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

// Set the active Recloser by index; index is 1-based: 1..count
func (reclosers *IReclosers) Set_idx(value int32) error {
	C.ctx_Reclosers_Set_idx(reclosers.ctxPtr, (C.int32_t)(value))
	return reclosers.ctx.DSSError()
}

func (reclosers *IReclosers) Close() error {
	C.ctx_Reclosers_Close(reclosers.ctxPtr)
	return reclosers.ctx.DSSError()
}

func (reclosers *IReclosers) Open() error {
	C.ctx_Reclosers_Open(reclosers.ctxPtr)
	return reclosers.ctx.DSSError()
}

// Ground (3I0) instantaneous trip setting - curve multipler or actual amps.
func (reclosers *IReclosers) Get_GroundInst() (float64, error) {
	return (float64)(C.ctx_Reclosers_Get_GroundInst(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

func (reclosers *IReclosers) Set_GroundInst(value float64) error {
	C.ctx_Reclosers_Set_GroundInst(reclosers.ctxPtr, (C.double)(value))
	return reclosers.ctx.DSSError()
}

// Ground (3I0) trip multiplier or actual amps
func (reclosers *IReclosers) Get_GroundTrip() (float64, error) {
	return (float64)(C.ctx_Reclosers_Get_GroundTrip(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

func (reclosers *IReclosers) Set_GroundTrip(value float64) error {
	C.ctx_Reclosers_Set_GroundTrip(reclosers.ctxPtr, (C.double)(value))
	return reclosers.ctx.DSSError()
}

// Full name of object this Recloser to be monitored.
func (reclosers *IReclosers) Get_MonitoredObj() (string, error) {
	return C.GoString(C.ctx_Reclosers_Get_MonitoredObj(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

func (reclosers *IReclosers) Set_MonitoredObj(value string) error {
	value_c := C.CString(value)
	C.ctx_Reclosers_Set_MonitoredObj(reclosers.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return reclosers.ctx.DSSError()
}

// Terminal number of Monitored object for the Recloser
func (reclosers *IReclosers) Get_MonitoredTerm() (int32, error) {
	return (int32)(C.ctx_Reclosers_Get_MonitoredTerm(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

func (reclosers *IReclosers) Set_MonitoredTerm(value int32) error {
	C.ctx_Reclosers_Set_MonitoredTerm(reclosers.ctxPtr, (C.int32_t)(value))
	return reclosers.ctx.DSSError()
}

// Number of fast shots
func (reclosers *IReclosers) Get_NumFast() (int32, error) {
	return (int32)(C.ctx_Reclosers_Get_NumFast(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

func (reclosers *IReclosers) Set_NumFast(value int32) error {
	C.ctx_Reclosers_Set_NumFast(reclosers.ctxPtr, (C.int32_t)(value))
	return reclosers.ctx.DSSError()
}

// Phase instantaneous curve multipler or actual amps
func (reclosers *IReclosers) Get_PhaseInst() (float64, error) {
	return (float64)(C.ctx_Reclosers_Get_PhaseInst(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

func (reclosers *IReclosers) Set_PhaseInst(value float64) error {
	C.ctx_Reclosers_Set_PhaseInst(reclosers.ctxPtr, (C.double)(value))
	return reclosers.ctx.DSSError()
}

// Phase trip curve multiplier or actual amps
func (reclosers *IReclosers) Get_PhaseTrip() (float64, error) {
	return (float64)(C.ctx_Reclosers_Get_PhaseTrip(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

func (reclosers *IReclosers) Set_PhaseTrip(value float64) error {
	C.ctx_Reclosers_Set_PhaseTrip(reclosers.ctxPtr, (C.double)(value))
	return reclosers.ctx.DSSError()
}

// Array of Doubles: reclose intervals, s, between shots.
func (reclosers *IReclosers) RecloseIntervals() ([]float64, error) {
	C.ctx_Reclosers_Get_RecloseIntervals_GR(reclosers.ctxPtr)
	return reclosers.ctx.GetFloat64ArrayGR()
}

// Number of shots to lockout (fast + delayed)
func (reclosers *IReclosers) Get_Shots() (int32, error) {
	return (int32)(C.ctx_Reclosers_Get_Shots(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

func (reclosers *IReclosers) Set_Shots(value int32) error {
	C.ctx_Reclosers_Set_Shots(reclosers.ctxPtr, (C.int32_t)(value))
	return reclosers.ctx.DSSError()
}

// Full name of the circuit element that is being switched by the Recloser.
func (reclosers *IReclosers) Get_SwitchedObj() (string, error) {
	return C.GoString(C.ctx_Reclosers_Get_SwitchedObj(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

func (reclosers *IReclosers) Set_SwitchedObj(value string) error {
	value_c := C.CString(value)
	C.ctx_Reclosers_Set_SwitchedObj(reclosers.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return reclosers.ctx.DSSError()
}

// Terminal number of the controlled device being switched by the Recloser
func (reclosers *IReclosers) Get_SwitchedTerm() (int32, error) {
	return (int32)(C.ctx_Reclosers_Get_SwitchedTerm(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

func (reclosers *IReclosers) Set_SwitchedTerm(value int32) error {
	C.ctx_Reclosers_Set_SwitchedTerm(reclosers.ctxPtr, (C.int32_t)(value))
	return reclosers.ctx.DSSError()
}

// Reset recloser to normal state.
// If open, lock out the recloser.
// If closed, resets recloser to first operation.
func (reclosers *IReclosers) Reset() error {
	C.ctx_Reclosers_Reset(reclosers.ctxPtr)
	return reclosers.ctx.DSSError()
}

// Get/Set present state of recloser.
// If set to open (ActionCodes.Open=1), open recloser's controlled element and lock out the recloser.
// If set to close (ActionCodes.Close=2), close recloser's controlled element and resets recloser to first operation.
func (reclosers *IReclosers) Get_State() (int32, error) {
	return (int32)(C.ctx_Reclosers_Get_State(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

func (reclosers *IReclosers) Set_State(value int32) error {
	C.ctx_Reclosers_Set_State(reclosers.ctxPtr, (C.int32_t)(value))
	return reclosers.ctx.DSSError()
}

// Get/set normal state (ActionCodes.Open=1, ActionCodes.Close=2) of the recloser.
func (reclosers *IReclosers) Get_NormalState() (int32, error) {
	return (int32)(C.ctx_Reclosers_Get_NormalState(reclosers.ctxPtr)), reclosers.ctx.DSSError()
}

func (reclosers *IReclosers) Set_NormalState(value int32) error {
	C.ctx_Reclosers_Set_NormalState(reclosers.ctxPtr, (C.int32_t)(value))
	return reclosers.ctx.DSSError()
}

type IRegControls struct {
	ICommonData
}

func (regcontrols *IRegControls) Init(ctx *DSSContextPtrs) {
	regcontrols.InitCommon(ctx)
}

// Array of strings with all RegControl names in the circuit.
func (regcontrols *IRegControls) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_RegControls_Get_AllNames(regcontrols.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return regcontrols.ctx.GetStringArray(data, cnt)
}

// Number of RegControl objects in active circuit.
func (regcontrols *IRegControls) Count() (int32, error) {
	return (int32)(C.ctx_RegControls_Get_Count(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

// Sets the first RegControl active. Returns 0 if no more.
func (regcontrols *IRegControls) First() (int32, error) {
	return (int32)(C.ctx_RegControls_Get_First(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

// Sets the active RegControl by Name.
func (regcontrols *IRegControls) Get_Name() (string, error) {
	result := C.GoString(C.ctx_RegControls_Get_Name(regcontrols.ctxPtr))
	return result, regcontrols.ctx.DSSError()
}

// Gets the name of the active RegControl.
func (regcontrols *IRegControls) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_RegControls_Set_Name(regcontrols.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return regcontrols.ctx.DSSError()
}

// Sets the next RegControl active. Returns 0 if no more.
func (regcontrols *IRegControls) Next() (int32, error) {
	return (int32)(C.ctx_RegControls_Get_Next(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

// Get the index of the active RegControl; index is 1-based: 1..count
func (regcontrols *IRegControls) Get_idx() (int32, error) {
	return (int32)(C.ctx_RegControls_Get_idx(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

// Set the active RegControl by index; index is 1-based: 1..count
func (regcontrols *IRegControls) Set_idx(value int32) error {
	C.ctx_RegControls_Set_idx(regcontrols.ctxPtr, (C.int32_t)(value))
	return regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Reset() error {
	C.ctx_RegControls_Reset(regcontrols.ctxPtr)
	return regcontrols.ctx.DSSError()
}

// CT primary ampere rating (secondary is 0.2 amperes)
func (regcontrols *IRegControls) Get_CTPrimary() (float64, error) {
	return (float64)(C.ctx_RegControls_Get_CTPrimary(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_CTPrimary(value float64) error {
	C.ctx_RegControls_Set_CTPrimary(regcontrols.ctxPtr, (C.double)(value))
	return regcontrols.ctx.DSSError()
}

// Time delay [s] after arming before the first tap change. Control may reset before actually changing taps.
func (regcontrols *IRegControls) Get_Delay() (float64, error) {
	return (float64)(C.ctx_RegControls_Get_Delay(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_Delay(value float64) error {
	C.ctx_RegControls_Set_Delay(regcontrols.ctxPtr, (C.double)(value))
	return regcontrols.ctx.DSSError()
}

// Regulation bandwidth in forward direciton, centered on Vreg
func (regcontrols *IRegControls) Get_ForwardBand() (float64, error) {
	return (float64)(C.ctx_RegControls_Get_ForwardBand(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_ForwardBand(value float64) error {
	C.ctx_RegControls_Set_ForwardBand(regcontrols.ctxPtr, (C.double)(value))
	return regcontrols.ctx.DSSError()
}

// LDC R setting in Volts
func (regcontrols *IRegControls) Get_ForwardR() (float64, error) {
	return (float64)(C.ctx_RegControls_Get_ForwardR(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_ForwardR(value float64) error {
	C.ctx_RegControls_Set_ForwardR(regcontrols.ctxPtr, (C.double)(value))
	return regcontrols.ctx.DSSError()
}

// Target voltage in the forward direction, on PT secondary base.
func (regcontrols *IRegControls) Get_ForwardVreg() (float64, error) {
	return (float64)(C.ctx_RegControls_Get_ForwardVreg(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_ForwardVreg(value float64) error {
	C.ctx_RegControls_Set_ForwardVreg(regcontrols.ctxPtr, (C.double)(value))
	return regcontrols.ctx.DSSError()
}

// LDC X setting in Volts
func (regcontrols *IRegControls) Get_ForwardX() (float64, error) {
	return (float64)(C.ctx_RegControls_Get_ForwardX(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_ForwardX(value float64) error {
	C.ctx_RegControls_Set_ForwardX(regcontrols.ctxPtr, (C.double)(value))
	return regcontrols.ctx.DSSError()
}

// Time delay is inversely adjsuted, proportinal to the amount of voltage outside the regulating band.
func (regcontrols *IRegControls) Get_IsInverseTime() (bool, error) {
	return (C.ctx_RegControls_Get_IsInverseTime(regcontrols.ctxPtr) != 0), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_IsInverseTime(value bool) error {
	C.ctx_RegControls_Set_IsInverseTime(regcontrols.ctxPtr, ToUint16(value))
	return regcontrols.ctx.DSSError()
}

// Regulator can use different settings in the reverse direction.  Usually not applicable to substation transformers.
func (regcontrols *IRegControls) Get_IsReversible() (bool, error) {
	return (C.ctx_RegControls_Get_IsReversible(regcontrols.ctxPtr) != 0), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_IsReversible(value bool) error {
	C.ctx_RegControls_Set_IsReversible(regcontrols.ctxPtr, ToUint16(value))
	return regcontrols.ctx.DSSError()
}

// Maximum tap change per iteration in STATIC solution mode. 1 is more realistic, 16 is the default for a faster soluiton.
func (regcontrols *IRegControls) Get_MaxTapChange() (int32, error) {
	return (int32)(C.ctx_RegControls_Get_MaxTapChange(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_MaxTapChange(value int32) error {
	C.ctx_RegControls_Set_MaxTapChange(regcontrols.ctxPtr, (C.int32_t)(value))
	return regcontrols.ctx.DSSError()
}

// Name of a remote regulated bus, in lieu of LDC settings
func (regcontrols *IRegControls) Get_MonitoredBus() (string, error) {
	return C.GoString(C.ctx_RegControls_Get_MonitoredBus(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_MonitoredBus(value string) error {
	value_c := C.CString(value)
	C.ctx_RegControls_Set_MonitoredBus(regcontrols.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return regcontrols.ctx.DSSError()
}

// PT ratio for voltage control settings
func (regcontrols *IRegControls) Get_PTratio() (float64, error) {
	return (float64)(C.ctx_RegControls_Get_PTratio(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_PTratio(value float64) error {
	C.ctx_RegControls_Set_PTratio(regcontrols.ctxPtr, (C.double)(value))
	return regcontrols.ctx.DSSError()
}

// Bandwidth in reverse direction, centered on reverse Vreg.
func (regcontrols *IRegControls) Get_ReverseBand() (float64, error) {
	return (float64)(C.ctx_RegControls_Get_ReverseBand(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_ReverseBand(value float64) error {
	C.ctx_RegControls_Set_ReverseBand(regcontrols.ctxPtr, (C.double)(value))
	return regcontrols.ctx.DSSError()
}

// Reverse LDC R setting in Volts.
func (regcontrols *IRegControls) Get_ReverseR() (float64, error) {
	return (float64)(C.ctx_RegControls_Get_ReverseR(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_ReverseR(value float64) error {
	C.ctx_RegControls_Set_ReverseR(regcontrols.ctxPtr, (C.double)(value))
	return regcontrols.ctx.DSSError()
}

// Target voltage in the revese direction, on PT secondary base.
func (regcontrols *IRegControls) Get_ReverseVreg() (float64, error) {
	return (float64)(C.ctx_RegControls_Get_ReverseVreg(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_ReverseVreg(value float64) error {
	C.ctx_RegControls_Set_ReverseVreg(regcontrols.ctxPtr, (C.double)(value))
	return regcontrols.ctx.DSSError()
}

// Reverse LDC X setting in volts.
func (regcontrols *IRegControls) Get_ReverseX() (float64, error) {
	return (float64)(C.ctx_RegControls_Get_ReverseX(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_ReverseX(value float64) error {
	C.ctx_RegControls_Set_ReverseX(regcontrols.ctxPtr, (C.double)(value))
	return regcontrols.ctx.DSSError()
}

// Time delay [s] for subsequent tap changes in a set. Control may reset before actually changing taps.
func (regcontrols *IRegControls) Get_TapDelay() (float64, error) {
	return (float64)(C.ctx_RegControls_Get_TapDelay(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_TapDelay(value float64) error {
	C.ctx_RegControls_Set_TapDelay(regcontrols.ctxPtr, (C.double)(value))
	return regcontrols.ctx.DSSError()
}

// Integer number of the tap that the controlled transformer winding is currentliy on.
func (regcontrols *IRegControls) Get_TapNumber() (int32, error) {
	return (int32)(C.ctx_RegControls_Get_TapNumber(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_TapNumber(value int32) error {
	C.ctx_RegControls_Set_TapNumber(regcontrols.ctxPtr, (C.int32_t)(value))
	return regcontrols.ctx.DSSError()
}

// Tapped winding number
func (regcontrols *IRegControls) Get_TapWinding() (int32, error) {
	return (int32)(C.ctx_RegControls_Get_TapWinding(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_TapWinding(value int32) error {
	C.ctx_RegControls_Set_TapWinding(regcontrols.ctxPtr, (C.int32_t)(value))
	return regcontrols.ctx.DSSError()
}

// Name of the transformer this regulator controls
func (regcontrols *IRegControls) Get_Transformer() (string, error) {
	return C.GoString(C.ctx_RegControls_Get_Transformer(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_Transformer(value string) error {
	value_c := C.CString(value)
	C.ctx_RegControls_Set_Transformer(regcontrols.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return regcontrols.ctx.DSSError()
}

// First house voltage limit on PT secondary base.  Setting to 0 disables this function.
func (regcontrols *IRegControls) Get_VoltageLimit() (float64, error) {
	return (float64)(C.ctx_RegControls_Get_VoltageLimit(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_VoltageLimit(value float64) error {
	C.ctx_RegControls_Set_VoltageLimit(regcontrols.ctxPtr, (C.double)(value))
	return regcontrols.ctx.DSSError()
}

// Winding number for PT and CT connections
func (regcontrols *IRegControls) Get_Winding() (int32, error) {
	return (int32)(C.ctx_RegControls_Get_Winding(regcontrols.ctxPtr)), regcontrols.ctx.DSSError()
}

func (regcontrols *IRegControls) Set_Winding(value int32) error {
	C.ctx_RegControls_Set_Winding(regcontrols.ctxPtr, (C.int32_t)(value))
	return regcontrols.ctx.DSSError()
}

type IRelays struct {
	ICommonData
}

func (relays *IRelays) Init(ctx *DSSContextPtrs) {
	relays.InitCommon(ctx)
}

// Array of strings with all Relay names in the circuit.
func (relays *IRelays) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Relays_Get_AllNames(relays.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return relays.ctx.GetStringArray(data, cnt)
}

// Number of Relay objects in active circuit.
func (relays *IRelays) Count() (int32, error) {
	return (int32)(C.ctx_Relays_Get_Count(relays.ctxPtr)), relays.ctx.DSSError()
}

// Sets the first Relay active. Returns 0 if no more.
func (relays *IRelays) First() (int32, error) {
	return (int32)(C.ctx_Relays_Get_First(relays.ctxPtr)), relays.ctx.DSSError()
}

// Sets the active Relay by Name.
func (relays *IRelays) Get_Name() (string, error) {
	result := C.GoString(C.ctx_Relays_Get_Name(relays.ctxPtr))
	return result, relays.ctx.DSSError()
}

// Gets the name of the active Relay.
func (relays *IRelays) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_Relays_Set_Name(relays.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return relays.ctx.DSSError()
}

// Sets the next Relay active. Returns 0 if no more.
func (relays *IRelays) Next() (int32, error) {
	return (int32)(C.ctx_Relays_Get_Next(relays.ctxPtr)), relays.ctx.DSSError()
}

// Get the index of the active Relay; index is 1-based: 1..count
func (relays *IRelays) Get_idx() (int32, error) {
	return (int32)(C.ctx_Relays_Get_idx(relays.ctxPtr)), relays.ctx.DSSError()
}

// Set the active Relay by index; index is 1-based: 1..count
func (relays *IRelays) Set_idx(value int32) error {
	C.ctx_Relays_Set_idx(relays.ctxPtr, (C.int32_t)(value))
	return relays.ctx.DSSError()
}

// Full name of object this Relay is monitoring.
func (relays *IRelays) Get_MonitoredObj() (string, error) {
	return C.GoString(C.ctx_Relays_Get_MonitoredObj(relays.ctxPtr)), relays.ctx.DSSError()
}

func (relays *IRelays) Set_MonitoredObj(value string) error {
	value_c := C.CString(value)
	C.ctx_Relays_Set_MonitoredObj(relays.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return relays.ctx.DSSError()
}

// Number of terminal of monitored element that this Relay is monitoring.
func (relays *IRelays) Get_MonitoredTerm() (int32, error) {
	return (int32)(C.ctx_Relays_Get_MonitoredTerm(relays.ctxPtr)), relays.ctx.DSSError()
}

func (relays *IRelays) Set_MonitoredTerm(value int32) error {
	C.ctx_Relays_Set_MonitoredTerm(relays.ctxPtr, (C.int32_t)(value))
	return relays.ctx.DSSError()
}

// Full name of element that will be switched when relay trips.
func (relays *IRelays) Get_SwitchedObj() (string, error) {
	return C.GoString(C.ctx_Relays_Get_SwitchedObj(relays.ctxPtr)), relays.ctx.DSSError()
}

func (relays *IRelays) Set_SwitchedObj(value string) error {
	value_c := C.CString(value)
	C.ctx_Relays_Set_SwitchedObj(relays.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return relays.ctx.DSSError()
}

// Terminal number of the switched object that will be opened when the relay trips.
func (relays *IRelays) Get_SwitchedTerm() (int32, error) {
	return (int32)(C.ctx_Relays_Get_SwitchedTerm(relays.ctxPtr)), relays.ctx.DSSError()
}

func (relays *IRelays) Set_SwitchedTerm(value int32) error {
	C.ctx_Relays_Set_SwitchedTerm(relays.ctxPtr, (C.int32_t)(value))
	return relays.ctx.DSSError()
}

// Open relay's controlled element and lock out the relay.
func (relays *IRelays) Open() error {
	C.ctx_Relays_Open(relays.ctxPtr)
	return relays.ctx.DSSError()
}

// Close the switched object controlled by the relay. Resets relay to first operation.
func (relays *IRelays) Close() error {
	C.ctx_Relays_Close(relays.ctxPtr)
	return relays.ctx.DSSError()
}

// Reset relay to normal state.
// If open, lock out the relay.
// If closed, resets relay to first operation.
func (relays *IRelays) Reset() error {
	C.ctx_Relays_Reset(relays.ctxPtr)
	return relays.ctx.DSSError()
}

// Get/Set present state of relay.
// If set to open, open relay's controlled element and lock out the relay.
// If set to close, close relay's controlled element and resets relay to first operation.
func (relays *IRelays) Get_State() (int32, error) {
	return (int32)(C.ctx_Relays_Get_State(relays.ctxPtr)), relays.ctx.DSSError()
}

func (relays *IRelays) Set_State(value int32) error {
	C.ctx_Relays_Set_State(relays.ctxPtr, (C.int32_t)(value))
	return relays.ctx.DSSError()
}

// Normal state of relay.
func (relays *IRelays) Get_NormalState() (int32, error) {
	return (int32)(C.ctx_Relays_Get_NormalState(relays.ctxPtr)), relays.ctx.DSSError()
}

func (relays *IRelays) Set_NormalState(value int32) error {
	C.ctx_Relays_Set_NormalState(relays.ctxPtr, (C.int32_t)(value))
	return relays.ctx.DSSError()
}

type ISensors struct {
	ICommonData
}

func (sensors *ISensors) Init(ctx *DSSContextPtrs) {
	sensors.InitCommon(ctx)
}

// Array of strings with all Sensor names in the circuit.
func (sensors *ISensors) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Sensors_Get_AllNames(sensors.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return sensors.ctx.GetStringArray(data, cnt)
}

// Number of Sensor objects in active circuit.
func (sensors *ISensors) Count() (int32, error) {
	return (int32)(C.ctx_Sensors_Get_Count(sensors.ctxPtr)), sensors.ctx.DSSError()
}

// Sets the first Sensor active. Returns 0 if no more.
func (sensors *ISensors) First() (int32, error) {
	return (int32)(C.ctx_Sensors_Get_First(sensors.ctxPtr)), sensors.ctx.DSSError()
}

// Sets the active Sensor by Name.
func (sensors *ISensors) Get_Name() (string, error) {
	result := C.GoString(C.ctx_Sensors_Get_Name(sensors.ctxPtr))
	return result, sensors.ctx.DSSError()
}

// Gets the name of the active Sensor.
func (sensors *ISensors) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_Sensors_Set_Name(sensors.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return sensors.ctx.DSSError()
}

// Sets the next Sensor active. Returns 0 if no more.
func (sensors *ISensors) Next() (int32, error) {
	return (int32)(C.ctx_Sensors_Get_Next(sensors.ctxPtr)), sensors.ctx.DSSError()
}

// Get the index of the active Sensor; index is 1-based: 1..count
func (sensors *ISensors) Get_idx() (int32, error) {
	return (int32)(C.ctx_Sensors_Get_idx(sensors.ctxPtr)), sensors.ctx.DSSError()
}

// Set the active Sensor by index; index is 1-based: 1..count
func (sensors *ISensors) Set_idx(value int32) error {
	C.ctx_Sensors_Set_idx(sensors.ctxPtr, (C.int32_t)(value))
	return sensors.ctx.DSSError()
}

func (sensors *ISensors) Reset() error {
	C.ctx_Sensors_Reset(sensors.ctxPtr)
	return sensors.ctx.DSSError()
}

func (sensors *ISensors) ResetAll() error {
	C.ctx_Sensors_ResetAll(sensors.ctxPtr)
	return sensors.ctx.DSSError()
}

// Array of doubles for the line current measurements; don't use with kWS and kVARS.
func (sensors *ISensors) Get_Currents() ([]float64, error) {
	C.ctx_Sensors_Get_Currents_GR(sensors.ctxPtr)
	return sensors.ctx.GetFloat64ArrayGR()
}

func (sensors *ISensors) Set_Currents(value []float64) error {
	C.ctx_Sensors_Set_Currents(sensors.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return sensors.ctx.DSSError()
}

// True if measured voltages are line-line. Currents are always line currents.
func (sensors *ISensors) Get_IsDelta() (bool, error) {
	return (C.ctx_Sensors_Get_IsDelta(sensors.ctxPtr) != 0), sensors.ctx.DSSError()
}

func (sensors *ISensors) Set_IsDelta(value bool) error {
	C.ctx_Sensors_Set_IsDelta(sensors.ctxPtr, ToUint16(value))
	return sensors.ctx.DSSError()
}

// Full Name of the measured element
func (sensors *ISensors) Get_MeteredElement() (string, error) {
	return C.GoString(C.ctx_Sensors_Get_MeteredElement(sensors.ctxPtr)), sensors.ctx.DSSError()
}

func (sensors *ISensors) Set_MeteredElement(value string) error {
	value_c := C.CString(value)
	C.ctx_Sensors_Set_MeteredElement(sensors.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return sensors.ctx.DSSError()
}

// Number of the measured terminal in the measured element.
func (sensors *ISensors) Get_MeteredTerminal() (int32, error) {
	return (int32)(C.ctx_Sensors_Get_MeteredTerminal(sensors.ctxPtr)), sensors.ctx.DSSError()
}

func (sensors *ISensors) Set_MeteredTerminal(value int32) error {
	C.ctx_Sensors_Set_MeteredTerminal(sensors.ctxPtr, (C.int32_t)(value))
	return sensors.ctx.DSSError()
}

// Assumed percent error in the Sensor measurement. Default is 1.
func (sensors *ISensors) Get_PctError() (float64, error) {
	return (float64)(C.ctx_Sensors_Get_PctError(sensors.ctxPtr)), sensors.ctx.DSSError()
}

func (sensors *ISensors) Set_PctError(value float64) error {
	C.ctx_Sensors_Set_PctError(sensors.ctxPtr, (C.double)(value))
	return sensors.ctx.DSSError()
}

// True if voltage measurements are 1-3, 3-2, 2-1.
func (sensors *ISensors) Get_ReverseDelta() (bool, error) {
	return (C.ctx_Sensors_Get_ReverseDelta(sensors.ctxPtr) != 0), sensors.ctx.DSSError()
}

func (sensors *ISensors) Set_ReverseDelta(value bool) error {
	C.ctx_Sensors_Set_ReverseDelta(sensors.ctxPtr, ToUint16(value))
	return sensors.ctx.DSSError()
}

// Weighting factor for this Sensor measurement with respect to other Sensors. Default is 1.
func (sensors *ISensors) Get_Weight() (float64, error) {
	return (float64)(C.ctx_Sensors_Get_Weight(sensors.ctxPtr)), sensors.ctx.DSSError()
}

func (sensors *ISensors) Set_Weight(value float64) error {
	C.ctx_Sensors_Set_Weight(sensors.ctxPtr, (C.double)(value))
	return sensors.ctx.DSSError()
}

// Array of doubles for Q measurements. Overwrites Currents with a new estimate using kWS.
func (sensors *ISensors) Get_kVARS() ([]float64, error) {
	C.ctx_Sensors_Get_kVARS_GR(sensors.ctxPtr)
	return sensors.ctx.GetFloat64ArrayGR()
}

func (sensors *ISensors) Set_kVARS(value []float64) error {
	C.ctx_Sensors_Set_kVARS(sensors.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return sensors.ctx.DSSError()
}

// Array of doubles for the LL or LN (depending on Delta connection) voltage measurements.
func (sensors *ISensors) Get_kVS() ([]float64, error) {
	C.ctx_Sensors_Get_kVS_GR(sensors.ctxPtr)
	return sensors.ctx.GetFloat64ArrayGR()
}

func (sensors *ISensors) Set_kVS(value []float64) error {
	C.ctx_Sensors_Set_kVS(sensors.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return sensors.ctx.DSSError()
}

// Voltage base for the sensor measurements. LL for 2 and 3-phase sensors, LN for 1-phase sensors.
func (sensors *ISensors) Get_kVbase() (float64, error) {
	return (float64)(C.ctx_Sensors_Get_kVbase(sensors.ctxPtr)), sensors.ctx.DSSError()
}

func (sensors *ISensors) Set_kVbase(value float64) error {
	C.ctx_Sensors_Set_kVbase(sensors.ctxPtr, (C.double)(value))
	return sensors.ctx.DSSError()
}

// Array of doubles for P measurements. Overwrites Currents with a new estimate using kVARS.
func (sensors *ISensors) Get_kWS() ([]float64, error) {
	C.ctx_Sensors_Get_kWS_GR(sensors.ctxPtr)
	return sensors.ctx.GetFloat64ArrayGR()
}

func (sensors *ISensors) Set_kWS(value []float64) error {
	C.ctx_Sensors_Set_kWS(sensors.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return sensors.ctx.DSSError()
}

// Array of doubles for the allocation factors for each phase.
func (sensors *ISensors) AllocationFactor() ([]float64, error) {
	C.ctx_Sensors_Get_AllocationFactor_GR(sensors.ctxPtr)
	return sensors.ctx.GetFloat64ArrayGR()
}

type ISwtControls struct {
	ICommonData
}

func (swtcontrols *ISwtControls) Init(ctx *DSSContextPtrs) {
	swtcontrols.InitCommon(ctx)
}

// Array of strings with all SwtControl names in the circuit.
func (swtcontrols *ISwtControls) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_SwtControls_Get_AllNames(swtcontrols.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return swtcontrols.ctx.GetStringArray(data, cnt)
}

// Number of SwtControl objects in active circuit.
func (swtcontrols *ISwtControls) Count() (int32, error) {
	return (int32)(C.ctx_SwtControls_Get_Count(swtcontrols.ctxPtr)), swtcontrols.ctx.DSSError()
}

// Sets the first SwtControl active. Returns 0 if no more.
func (swtcontrols *ISwtControls) First() (int32, error) {
	return (int32)(C.ctx_SwtControls_Get_First(swtcontrols.ctxPtr)), swtcontrols.ctx.DSSError()
}

// Sets the active SwtControl by Name.
func (swtcontrols *ISwtControls) Get_Name() (string, error) {
	result := C.GoString(C.ctx_SwtControls_Get_Name(swtcontrols.ctxPtr))
	return result, swtcontrols.ctx.DSSError()
}

// Gets the name of the active SwtControl.
func (swtcontrols *ISwtControls) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_SwtControls_Set_Name(swtcontrols.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return swtcontrols.ctx.DSSError()
}

// Sets the next SwtControl active. Returns 0 if no more.
func (swtcontrols *ISwtControls) Next() (int32, error) {
	return (int32)(C.ctx_SwtControls_Get_Next(swtcontrols.ctxPtr)), swtcontrols.ctx.DSSError()
}

// Get the index of the active SwtControl; index is 1-based: 1..count
func (swtcontrols *ISwtControls) Get_idx() (int32, error) {
	return (int32)(C.ctx_SwtControls_Get_idx(swtcontrols.ctxPtr)), swtcontrols.ctx.DSSError()
}

// Set the active SwtControl by index; index is 1-based: 1..count
func (swtcontrols *ISwtControls) Set_idx(value int32) error {
	C.ctx_SwtControls_Set_idx(swtcontrols.ctxPtr, (C.int32_t)(value))
	return swtcontrols.ctx.DSSError()
}

func (swtcontrols *ISwtControls) Reset() error {
	C.ctx_SwtControls_Reset(swtcontrols.ctxPtr)
	return swtcontrols.ctx.DSSError()
}

// Open or Close the switch. No effect if switch is locked.  However, Reset removes any lock and then closes the switch (shelf state).
func (swtcontrols *ISwtControls) Get_Action() (int32, error) {
	return (int32)(C.ctx_SwtControls_Get_Action(swtcontrols.ctxPtr)), swtcontrols.ctx.DSSError()
}

func (swtcontrols *ISwtControls) Set_Action(value int32) error {
	C.ctx_SwtControls_Set_Action(swtcontrols.ctxPtr, (C.int32_t)(value))
	return swtcontrols.ctx.DSSError()
}

// Time delay [s] betwen arming and opening or closing the switch.  Control may reset before actually operating the switch.
func (swtcontrols *ISwtControls) Get_Delay() (float64, error) {
	return (float64)(C.ctx_SwtControls_Get_Delay(swtcontrols.ctxPtr)), swtcontrols.ctx.DSSError()
}

func (swtcontrols *ISwtControls) Set_Delay(value float64) error {
	C.ctx_SwtControls_Set_Delay(swtcontrols.ctxPtr, (C.double)(value))
	return swtcontrols.ctx.DSSError()
}

// The lock prevents both manual and automatic switch operation.
func (swtcontrols *ISwtControls) Get_IsLocked() (bool, error) {
	return (C.ctx_SwtControls_Get_IsLocked(swtcontrols.ctxPtr) != 0), swtcontrols.ctx.DSSError()
}

func (swtcontrols *ISwtControls) Set_IsLocked(value bool) error {
	C.ctx_SwtControls_Set_IsLocked(swtcontrols.ctxPtr, ToUint16(value))
	return swtcontrols.ctx.DSSError()
}

// Get/set Normal state of switch (see actioncodes) dssActionOpen or dssActionClose
func (swtcontrols *ISwtControls) Get_NormalState() (ActionCodes, error) {
	return (ActionCodes)(C.ctx_SwtControls_Get_NormalState(swtcontrols.ctxPtr)), swtcontrols.ctx.DSSError()
}

func (swtcontrols *ISwtControls) Set_NormalState(value ActionCodes) error {
	C.ctx_SwtControls_Set_NormalState(swtcontrols.ctxPtr, (C.int32_t)(value))
	return swtcontrols.ctx.DSSError()
}

// Set it to force the switch to a specified state, otherwise read its present state.
func (swtcontrols *ISwtControls) Get_State() (int32, error) {
	return (int32)(C.ctx_SwtControls_Get_State(swtcontrols.ctxPtr)), swtcontrols.ctx.DSSError()
}

func (swtcontrols *ISwtControls) Set_State(value int32) error {
	C.ctx_SwtControls_Set_State(swtcontrols.ctxPtr, (C.int32_t)(value))
	return swtcontrols.ctx.DSSError()
}

// Full name of the switched element.
func (swtcontrols *ISwtControls) Get_SwitchedObj() (string, error) {
	return C.GoString(C.ctx_SwtControls_Get_SwitchedObj(swtcontrols.ctxPtr)), swtcontrols.ctx.DSSError()
}

func (swtcontrols *ISwtControls) Set_SwitchedObj(value string) error {
	value_c := C.CString(value)
	C.ctx_SwtControls_Set_SwitchedObj(swtcontrols.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return swtcontrols.ctx.DSSError()
}

// Terminal number where the switch is located on the SwitchedObj
func (swtcontrols *ISwtControls) Get_SwitchedTerm() (int32, error) {
	return (int32)(C.ctx_SwtControls_Get_SwitchedTerm(swtcontrols.ctxPtr)), swtcontrols.ctx.DSSError()
}

func (swtcontrols *ISwtControls) Set_SwitchedTerm(value int32) error {
	C.ctx_SwtControls_Set_SwitchedTerm(swtcontrols.ctxPtr, (C.int32_t)(value))
	return swtcontrols.ctx.DSSError()
}

type ITSData struct {
	ICommonData
}

func (tsdata *ITSData) Init(ctx *DSSContextPtrs) {
	tsdata.InitCommon(ctx)
}

// Array of strings with all TSData names in the circuit.
func (tsdata *ITSData) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_TSData_Get_AllNames(tsdata.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return tsdata.ctx.GetStringArray(data, cnt)
}

// Number of TSData objects in active circuit.
func (tsdata *ITSData) Count() (int32, error) {
	return (int32)(C.ctx_TSData_Get_Count(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

// Sets the first TSData active. Returns 0 if no more.
func (tsdata *ITSData) First() (int32, error) {
	return (int32)(C.ctx_TSData_Get_First(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

// Sets the active TSData by Name.
func (tsdata *ITSData) Get_Name() (string, error) {
	result := C.GoString(C.ctx_TSData_Get_Name(tsdata.ctxPtr))
	return result, tsdata.ctx.DSSError()
}

// Gets the name of the active TSData.
func (tsdata *ITSData) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_TSData_Set_Name(tsdata.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return tsdata.ctx.DSSError()
}

// Sets the next TSData active. Returns 0 if no more.
func (tsdata *ITSData) Next() (int32, error) {
	return (int32)(C.ctx_TSData_Get_Next(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

// Get the index of the active TSData; index is 1-based: 1..count
func (tsdata *ITSData) Get_idx() (int32, error) {
	return (int32)(C.ctx_TSData_Get_idx(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

// Set the active TSData by index; index is 1-based: 1..count
func (tsdata *ITSData) Set_idx(value int32) error {
	C.ctx_TSData_Set_idx(tsdata.ctxPtr, (C.int32_t)(value))
	return tsdata.ctx.DSSError()
}

// Emergency ampere rating
func (tsdata *ITSData) Get_EmergAmps() (float64, error) {
	return (float64)(C.ctx_TSData_Get_EmergAmps(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_EmergAmps(value float64) error {
	C.ctx_TSData_Set_EmergAmps(tsdata.ctxPtr, (C.double)(value))
	return tsdata.ctx.DSSError()
}

// Normal Ampere rating
func (tsdata *ITSData) Get_NormAmps() (float64, error) {
	return (float64)(C.ctx_TSData_Get_NormAmps(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_NormAmps(value float64) error {
	C.ctx_TSData_Set_NormAmps(tsdata.ctxPtr, (C.double)(value))
	return tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Get_Rdc() (float64, error) {
	return (float64)(C.ctx_TSData_Get_Rdc(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_Rdc(value float64) error {
	C.ctx_TSData_Set_Rdc(tsdata.ctxPtr, (C.double)(value))
	return tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Get_Rac() (float64, error) {
	return (float64)(C.ctx_TSData_Get_Rac(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_Rac(value float64) error {
	C.ctx_TSData_Set_Rac(tsdata.ctxPtr, (C.double)(value))
	return tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Get_GMRac() (float64, error) {
	return (float64)(C.ctx_TSData_Get_GMRac(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_GMRac(value float64) error {
	C.ctx_TSData_Set_GMRac(tsdata.ctxPtr, (C.double)(value))
	return tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Get_GMRUnits() (int32, error) {
	return (int32)(C.ctx_TSData_Get_GMRUnits(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_GMRUnits(value int32) error {
	C.ctx_TSData_Set_GMRUnits(tsdata.ctxPtr, (C.int32_t)(value))
	return tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Get_Radius() (float64, error) {
	return (float64)(C.ctx_TSData_Get_Radius(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_Radius(value float64) error {
	C.ctx_TSData_Set_Radius(tsdata.ctxPtr, (C.double)(value))
	return tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Get_RadiusUnits() (int32, error) {
	return (int32)(C.ctx_TSData_Get_RadiusUnits(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_RadiusUnits(value int32) error {
	C.ctx_TSData_Set_RadiusUnits(tsdata.ctxPtr, (C.int32_t)(value))
	return tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Get_ResistanceUnits() (int32, error) {
	return (int32)(C.ctx_TSData_Get_ResistanceUnits(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_ResistanceUnits(value int32) error {
	C.ctx_TSData_Set_ResistanceUnits(tsdata.ctxPtr, (C.int32_t)(value))
	return tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Get_Diameter() (float64, error) {
	return (float64)(C.ctx_TSData_Get_Diameter(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_Diameter(value float64) error {
	C.ctx_TSData_Set_Diameter(tsdata.ctxPtr, (C.double)(value))
	return tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Get_EpsR() (float64, error) {
	return (float64)(C.ctx_TSData_Get_EpsR(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_EpsR(value float64) error {
	C.ctx_TSData_Set_EpsR(tsdata.ctxPtr, (C.double)(value))
	return tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Get_InsLayer() (float64, error) {
	return (float64)(C.ctx_TSData_Get_InsLayer(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_InsLayer(value float64) error {
	C.ctx_TSData_Set_InsLayer(tsdata.ctxPtr, (C.double)(value))
	return tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Get_DiaIns() (float64, error) {
	return (float64)(C.ctx_TSData_Get_DiaIns(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_DiaIns(value float64) error {
	C.ctx_TSData_Set_DiaIns(tsdata.ctxPtr, (C.double)(value))
	return tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Get_DiaCable() (float64, error) {
	return (float64)(C.ctx_TSData_Get_DiaCable(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_DiaCable(value float64) error {
	C.ctx_TSData_Set_DiaCable(tsdata.ctxPtr, (C.double)(value))
	return tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Get_DiaShield() (float64, error) {
	return (float64)(C.ctx_TSData_Get_DiaShield(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_DiaShield(value float64) error {
	C.ctx_TSData_Set_DiaShield(tsdata.ctxPtr, (C.double)(value))
	return tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Get_TapeLayer() (float64, error) {
	return (float64)(C.ctx_TSData_Get_TapeLayer(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_TapeLayer(value float64) error {
	C.ctx_TSData_Set_TapeLayer(tsdata.ctxPtr, (C.double)(value))
	return tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Get_TapeLap() (float64, error) {
	return (float64)(C.ctx_TSData_Get_TapeLap(tsdata.ctxPtr)), tsdata.ctx.DSSError()
}

func (tsdata *ITSData) Set_TapeLap(value float64) error {
	C.ctx_TSData_Set_TapeLap(tsdata.ctxPtr, (C.double)(value))
	return tsdata.ctx.DSSError()
}

type IText struct {
	ICommonData
}

func (text *IText) Init(ctx *DSSContextPtrs) {
	text.InitCommon(ctx)
}

// Runs a list of strings as commands directly in the DSS engine.
// Intermediate results are ignored.
//
// (API Extension)
func (text *IText) Commands(value []string) error {
	value_c := text.ctx.PrepareStringArray(value)
	defer text.ctx.FreeStringArray(value_c, len(value))
	C.ctx_Text_CommandArray(text.ctxPtr, value_c, (C.int32_t)(len(value)))
	return text.ctx.DSSError()
}

// Runs a large string as commands directly in the DSS engine.
// Intermediate results are ignored.
//
// (API Extension)
func (text *IText) CommandBlock(value string) error {
	value_c := C.CString(value)
	C.ctx_Text_CommandBlock(text.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return text.ctx.DSSError()
}

// Input command string for the DSS.
func (text *IText) Get_Command() (string, error) {
	return C.GoString(C.ctx_Text_Get_Command(text.ctxPtr)), text.ctx.DSSError()
}

func (text *IText) Set_Command(value string) error {
	value_c := C.CString(value)
	C.ctx_Text_Set_Command(text.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return text.ctx.DSSError()
}

// Result string for the last command.
func (text *IText) Result() (string, error) {
	return C.GoString(C.ctx_Text_Get_Result(text.ctxPtr)), text.ctx.DSSError()
}

type ITopology struct {
	ICommonData
}

func (topology *ITopology) Init(ctx *DSSContextPtrs) {
	topology.InitCommon(ctx)
}

// Returns index of the active branch
func (topology *ITopology) ActiveBranch() (int32, error) {
	return (int32)(C.ctx_Topology_Get_ActiveBranch(topology.ctxPtr)), topology.ctx.DSSError()
}

// Topological depth of the active branch
func (topology *ITopology) ActiveLevel() (int32, error) {
	return (int32)(C.ctx_Topology_Get_ActiveLevel(topology.ctxPtr)), topology.ctx.DSSError()
}

// Array of all isolated branch names.
func (topology *ITopology) AllIsolatedBranches() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Topology_Get_AllIsolatedBranches(topology.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return topology.ctx.GetStringArray(data, cnt)
}

// Array of all isolated load names.
func (topology *ITopology) AllIsolatedLoads() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Topology_Get_AllIsolatedLoads(topology.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return topology.ctx.GetStringArray(data, cnt)
}

// Array of all looped element names, by pairs.
func (topology *ITopology) AllLoopedPairs() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Topology_Get_AllLoopedPairs(topology.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return topology.ctx.GetStringArray(data, cnt)
}

// Move back toward the source, return index of new active branch, or 0 if no more.
func (topology *ITopology) BackwardBranch() (int32, error) {
	return (int32)(C.ctx_Topology_Get_BackwardBranch(topology.ctxPtr)), topology.ctx.DSSError()
}

// Name of the active branch.
func (topology *ITopology) Get_BranchName() (string, error) {
	return C.GoString(C.ctx_Topology_Get_BranchName(topology.ctxPtr)), topology.ctx.DSSError()
}

func (topology *ITopology) Set_BranchName(value string) error {
	value_c := C.CString(value)
	C.ctx_Topology_Set_BranchName(topology.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return topology.ctx.DSSError()
}

// Set the active branch to one containing this bus, return index or 0 if not found
func (topology *ITopology) Get_BusName() (string, error) {
	return C.GoString(C.ctx_Topology_Get_BusName(topology.ctxPtr)), topology.ctx.DSSError()
}

func (topology *ITopology) Set_BusName(value string) error {
	value_c := C.CString(value)
	C.ctx_Topology_Set_BusName(topology.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return topology.ctx.DSSError()
}

// Sets the first branch active, returns 0 if none.
func (topology *ITopology) First() (int32, error) {
	return (int32)(C.ctx_Topology_Get_First(topology.ctxPtr)), topology.ctx.DSSError()
}

// First load at the active branch, return index or 0 if none.
func (topology *ITopology) FirstLoad() (int32, error) {
	return (int32)(C.ctx_Topology_Get_FirstLoad(topology.ctxPtr)), topology.ctx.DSSError()
}

// Move forward in the tree, return index of new active branch or 0 if no more
func (topology *ITopology) ForwardBranch() (int32, error) {
	return (int32)(C.ctx_Topology_Get_ForwardBranch(topology.ctxPtr)), topology.ctx.DSSError()
}

// Move to looped branch, return index or 0 if none.
func (topology *ITopology) LoopedBranch() (int32, error) {
	return (int32)(C.ctx_Topology_Get_LoopedBranch(topology.ctxPtr)), topology.ctx.DSSError()
}

// Sets the next branch active, returns 0 if no more.
func (topology *ITopology) Next() (int32, error) {
	return (int32)(C.ctx_Topology_Get_Next(topology.ctxPtr)), topology.ctx.DSSError()
}

// Next load at the active branch, return index or 0 if no more.
func (topology *ITopology) NextLoad() (int32, error) {
	return (int32)(C.ctx_Topology_Get_NextLoad(topology.ctxPtr)), topology.ctx.DSSError()
}

// Number of isolated branches (PD elements and capacitors).
func (topology *ITopology) NumIsolatedBranches() (int32, error) {
	return (int32)(C.ctx_Topology_Get_NumIsolatedBranches(topology.ctxPtr)), topology.ctx.DSSError()
}

// Number of isolated loads
func (topology *ITopology) NumIsolatedLoads() (int32, error) {
	return (int32)(C.ctx_Topology_Get_NumIsolatedLoads(topology.ctxPtr)), topology.ctx.DSSError()
}

// Number of loops
func (topology *ITopology) NumLoops() (int32, error) {
	return (int32)(C.ctx_Topology_Get_NumLoops(topology.ctxPtr)), topology.ctx.DSSError()
}

// Move to directly parallel branch, return index or 0 if none.
func (topology *ITopology) ParallelBranch() (int32, error) {
	return (int32)(C.ctx_Topology_Get_ParallelBranch(topology.ctxPtr)), topology.ctx.DSSError()
}

type ITransformers struct {
	ICommonData
}

func (transformers *ITransformers) Init(ctx *DSSContextPtrs) {
	transformers.InitCommon(ctx)
}

// Array of strings with all Transformer names in the circuit.
func (transformers *ITransformers) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Transformers_Get_AllNames(transformers.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return transformers.ctx.GetStringArray(data, cnt)
}

// Number of Transformer objects in active circuit.
func (transformers *ITransformers) Count() (int32, error) {
	return (int32)(C.ctx_Transformers_Get_Count(transformers.ctxPtr)), transformers.ctx.DSSError()
}

// Sets the first Transformer active. Returns 0 if no more.
func (transformers *ITransformers) First() (int32, error) {
	return (int32)(C.ctx_Transformers_Get_First(transformers.ctxPtr)), transformers.ctx.DSSError()
}

// Sets the active Transformer by Name.
func (transformers *ITransformers) Get_Name() (string, error) {
	result := C.GoString(C.ctx_Transformers_Get_Name(transformers.ctxPtr))
	return result, transformers.ctx.DSSError()
}

// Gets the name of the active Transformer.
func (transformers *ITransformers) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_Transformers_Set_Name(transformers.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return transformers.ctx.DSSError()
}

// Sets the next Transformer active. Returns 0 if no more.
func (transformers *ITransformers) Next() (int32, error) {
	return (int32)(C.ctx_Transformers_Get_Next(transformers.ctxPtr)), transformers.ctx.DSSError()
}

// Get the index of the active Transformer; index is 1-based: 1..count
func (transformers *ITransformers) Get_idx() (int32, error) {
	return (int32)(C.ctx_Transformers_Get_idx(transformers.ctxPtr)), transformers.ctx.DSSError()
}

// Set the active Transformer by index; index is 1-based: 1..count
func (transformers *ITransformers) Set_idx(value int32) error {
	C.ctx_Transformers_Set_idx(transformers.ctxPtr, (C.int32_t)(value))
	return transformers.ctx.DSSError()
}

// Active Winding delta or wye connection?
func (transformers *ITransformers) Get_IsDelta() (bool, error) {
	return (C.ctx_Transformers_Get_IsDelta(transformers.ctxPtr) != 0), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_IsDelta(value bool) error {
	C.ctx_Transformers_Set_IsDelta(transformers.ctxPtr, ToUint16(value))
	return transformers.ctx.DSSError()
}

// Active Winding maximum tap in per-unit.
func (transformers *ITransformers) Get_MaxTap() (float64, error) {
	return (float64)(C.ctx_Transformers_Get_MaxTap(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_MaxTap(value float64) error {
	C.ctx_Transformers_Set_MaxTap(transformers.ctxPtr, (C.double)(value))
	return transformers.ctx.DSSError()
}

// Active Winding minimum tap in per-unit.
func (transformers *ITransformers) Get_MinTap() (float64, error) {
	return (float64)(C.ctx_Transformers_Get_MinTap(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_MinTap(value float64) error {
	C.ctx_Transformers_Set_MinTap(transformers.ctxPtr, (C.double)(value))
	return transformers.ctx.DSSError()
}

// Active Winding number of tap steps betwein MinTap and MaxTap.
func (transformers *ITransformers) Get_NumTaps() (int32, error) {
	return (int32)(C.ctx_Transformers_Get_NumTaps(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_NumTaps(value int32) error {
	C.ctx_Transformers_Set_NumTaps(transformers.ctxPtr, (C.int32_t)(value))
	return transformers.ctx.DSSError()
}

// Number of windings on this transformer. Allocates memory; set or change this property first.
func (transformers *ITransformers) Get_NumWindings() (int32, error) {
	return (int32)(C.ctx_Transformers_Get_NumWindings(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_NumWindings(value int32) error {
	C.ctx_Transformers_Set_NumWindings(transformers.ctxPtr, (C.int32_t)(value))
	return transformers.ctx.DSSError()
}

// Active Winding resistance in %
func (transformers *ITransformers) Get_R() (float64, error) {
	return (float64)(C.ctx_Transformers_Get_R(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_R(value float64) error {
	C.ctx_Transformers_Set_R(transformers.ctxPtr, (C.double)(value))
	return transformers.ctx.DSSError()
}

// Active Winding neutral resistance [ohms] for wye connections. Set less than zero for ungrounded wye.
func (transformers *ITransformers) Get_Rneut() (float64, error) {
	return (float64)(C.ctx_Transformers_Get_Rneut(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_Rneut(value float64) error {
	C.ctx_Transformers_Set_Rneut(transformers.ctxPtr, (C.double)(value))
	return transformers.ctx.DSSError()
}

// Active Winding tap in per-unit.
func (transformers *ITransformers) Get_Tap() (float64, error) {
	return (float64)(C.ctx_Transformers_Get_Tap(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_Tap(value float64) error {
	C.ctx_Transformers_Set_Tap(transformers.ctxPtr, (C.double)(value))
	return transformers.ctx.DSSError()
}

// Active Winding Number from 1..NumWindings. Update this before reading or setting a sequence of winding properties (R, Tap, kV, kVA, etc.)
func (transformers *ITransformers) Get_Wdg() (int32, error) {
	return (int32)(C.ctx_Transformers_Get_Wdg(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_Wdg(value int32) error {
	C.ctx_Transformers_Set_Wdg(transformers.ctxPtr, (C.int32_t)(value))
	return transformers.ctx.DSSError()
}

// Name of an XfrmCode that supplies electircal parameters for this Transformer.
func (transformers *ITransformers) Get_XfmrCode() (string, error) {
	return C.GoString(C.ctx_Transformers_Get_XfmrCode(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_XfmrCode(value string) error {
	value_c := C.CString(value)
	C.ctx_Transformers_Set_XfmrCode(transformers.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return transformers.ctx.DSSError()
}

// Percent reactance between windings 1 and 2, on winding 1 kVA base. Use for 2-winding or 3-winding transformers.
func (transformers *ITransformers) Get_Xhl() (float64, error) {
	return (float64)(C.ctx_Transformers_Get_Xhl(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_Xhl(value float64) error {
	C.ctx_Transformers_Set_Xhl(transformers.ctxPtr, (C.double)(value))
	return transformers.ctx.DSSError()
}

// Percent reactance between windigns 1 and 3, on winding 1 kVA base.  Use for 3-winding transformers only.
func (transformers *ITransformers) Get_Xht() (float64, error) {
	return (float64)(C.ctx_Transformers_Get_Xht(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_Xht(value float64) error {
	C.ctx_Transformers_Set_Xht(transformers.ctxPtr, (C.double)(value))
	return transformers.ctx.DSSError()
}

// Percent reactance between windings 2 and 3, on winding 1 kVA base. Use for 3-winding transformers only.
func (transformers *ITransformers) Get_Xlt() (float64, error) {
	return (float64)(C.ctx_Transformers_Get_Xlt(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_Xlt(value float64) error {
	C.ctx_Transformers_Set_Xlt(transformers.ctxPtr, (C.double)(value))
	return transformers.ctx.DSSError()
}

// Active Winding neutral reactance [ohms] for wye connections.
func (transformers *ITransformers) Get_Xneut() (float64, error) {
	return (float64)(C.ctx_Transformers_Get_Xneut(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_Xneut(value float64) error {
	C.ctx_Transformers_Set_Xneut(transformers.ctxPtr, (C.double)(value))
	return transformers.ctx.DSSError()
}

// Active Winding kV rating.  Phase-phase for 2 or 3 phases, actual winding kV for 1 phase transformer.
func (transformers *ITransformers) Get_kV() (float64, error) {
	return (float64)(C.ctx_Transformers_Get_kV(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_kV(value float64) error {
	C.ctx_Transformers_Set_kV(transformers.ctxPtr, (C.double)(value))
	return transformers.ctx.DSSError()
}

// Active Winding kVA rating. On winding 1, this also determines normal and emergency current ratings for all windings.
func (transformers *ITransformers) Get_kVA() (float64, error) {
	return (float64)(C.ctx_Transformers_Get_kVA(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_kVA(value float64) error {
	C.ctx_Transformers_Set_kVA(transformers.ctxPtr, (C.double)(value))
	return transformers.ctx.DSSError()
}

// Complex array of voltages for active winding
//
// WARNING: If the transformer has open terminal(s), results may be wrong, i.e. avoid using this
// in those situations. For more information, see https://github.com/dss-extensions/dss-extensions/issues/24
func (transformers *ITransformers) WdgVoltages() ([]complex128, error) {
	C.ctx_Transformers_Get_WdgVoltages_GR(transformers.ctxPtr)
	return transformers.ctx.GetComplexArrayGR()
}

// All Winding currents (ph1, wdg1, wdg2,... ph2, wdg1, wdg2 ...)
//
// WARNING: If the transformer has open terminal(s), results may be wrong, i.e. avoid using this
// in those situations. For more information, see https://github.com/dss-extensions/dss-extensions/issues/24
func (transformers *ITransformers) WdgCurrents() ([]complex128, error) {
	C.ctx_Transformers_Get_WdgCurrents_GR(transformers.ctxPtr)
	return transformers.ctx.GetComplexArrayGR()
}

// All winding currents in CSV string form like the WdgCurrents property
//
// WARNING: If the transformer has open terminal(s), results may be wrong, i.e. avoid using this
// in those situations. For more information, see https://github.com/dss-extensions/dss-extensions/issues/24
func (transformers *ITransformers) StrWdgCurrents() (string, error) {
	return C.GoString(C.ctx_Transformers_Get_strWdgCurrents(transformers.ctxPtr)), transformers.ctx.DSSError()
}

// Transformer Core Type: 0=Shell; 1=1ph; 3-3leg; 4=4-Leg; 5=5-leg; 9=Core-1-phase
func (transformers *ITransformers) Get_CoreType() (CoreType, error) {
	return (CoreType)(C.ctx_Transformers_Get_CoreType(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_CoreType(value CoreType) error {
	C.ctx_Transformers_Set_CoreType(transformers.ctxPtr, (C.int32_t)(value))
	return transformers.ctx.DSSError()
}

// dc Resistance of active winding in ohms for GIC analysis
func (transformers *ITransformers) Get_RdcOhms() (float64, error) {
	return (float64)(C.ctx_Transformers_Get_RdcOhms(transformers.ctxPtr)), transformers.ctx.DSSError()
}

func (transformers *ITransformers) Set_RdcOhms(value float64) error {
	C.ctx_Transformers_Set_RdcOhms(transformers.ctxPtr, (C.double)(value))
	return transformers.ctx.DSSError()
}

// Complex array with the losses by type (total losses, load losses, no-load losses), in VA
//
// (API Extension)
func (transformers *ITransformers) LossesByType() ([]complex128, error) {
	C.ctx_Transformers_Get_LossesByType_GR(transformers.ctxPtr)
	return transformers.ctx.GetComplexArrayGR()
}

// Complex array with the losses by type (total losses, load losses, no-load losses), in VA, concatenated for ALL transformers
//
// (API Extension)
func (transformers *ITransformers) AllLossesByType() ([]complex128, error) {
	C.ctx_Transformers_Get_AllLossesByType_GR(transformers.ctxPtr)
	return transformers.ctx.GetComplexArrayGR()
}

type IVsources struct {
	ICommonData
}

func (vsources *IVsources) Init(ctx *DSSContextPtrs) {
	vsources.InitCommon(ctx)
}

// Array of strings with all Vsource names in the circuit.
func (vsources *IVsources) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Vsources_Get_AllNames(vsources.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return vsources.ctx.GetStringArray(data, cnt)
}

// Number of Vsource objects in active circuit.
func (vsources *IVsources) Count() (int32, error) {
	return (int32)(C.ctx_Vsources_Get_Count(vsources.ctxPtr)), vsources.ctx.DSSError()
}

// Sets the first Vsource active. Returns 0 if no more.
func (vsources *IVsources) First() (int32, error) {
	return (int32)(C.ctx_Vsources_Get_First(vsources.ctxPtr)), vsources.ctx.DSSError()
}

// Sets the active Vsource by Name.
func (vsources *IVsources) Get_Name() (string, error) {
	result := C.GoString(C.ctx_Vsources_Get_Name(vsources.ctxPtr))
	return result, vsources.ctx.DSSError()
}

// Gets the name of the active Vsource.
func (vsources *IVsources) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_Vsources_Set_Name(vsources.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return vsources.ctx.DSSError()
}

// Sets the next Vsource active. Returns 0 if no more.
func (vsources *IVsources) Next() (int32, error) {
	return (int32)(C.ctx_Vsources_Get_Next(vsources.ctxPtr)), vsources.ctx.DSSError()
}

// Get the index of the active Vsource; index is 1-based: 1..count
func (vsources *IVsources) Get_idx() (int32, error) {
	return (int32)(C.ctx_Vsources_Get_idx(vsources.ctxPtr)), vsources.ctx.DSSError()
}

// Set the active Vsource by index; index is 1-based: 1..count
func (vsources *IVsources) Set_idx(value int32) error {
	C.ctx_Vsources_Set_idx(vsources.ctxPtr, (C.int32_t)(value))
	return vsources.ctx.DSSError()
}

// Phase angle of first phase in degrees
func (vsources *IVsources) Get_AngleDeg() (float64, error) {
	return (float64)(C.ctx_Vsources_Get_AngleDeg(vsources.ctxPtr)), vsources.ctx.DSSError()
}

func (vsources *IVsources) Set_AngleDeg(value float64) error {
	C.ctx_Vsources_Set_AngleDeg(vsources.ctxPtr, (C.double)(value))
	return vsources.ctx.DSSError()
}

// Source voltage in kV
func (vsources *IVsources) Get_BasekV() (float64, error) {
	return (float64)(C.ctx_Vsources_Get_BasekV(vsources.ctxPtr)), vsources.ctx.DSSError()
}

func (vsources *IVsources) Set_BasekV(value float64) error {
	C.ctx_Vsources_Set_BasekV(vsources.ctxPtr, (C.double)(value))
	return vsources.ctx.DSSError()
}

// Source frequency in Hz
func (vsources *IVsources) Get_Frequency() (float64, error) {
	return (float64)(C.ctx_Vsources_Get_Frequency(vsources.ctxPtr)), vsources.ctx.DSSError()
}

func (vsources *IVsources) Set_Frequency(value float64) error {
	C.ctx_Vsources_Set_Frequency(vsources.ctxPtr, (C.double)(value))
	return vsources.ctx.DSSError()
}

// Number of phases
func (vsources *IVsources) Get_Phases() (int32, error) {
	return (int32)(C.ctx_Vsources_Get_Phases(vsources.ctxPtr)), vsources.ctx.DSSError()
}

func (vsources *IVsources) Set_Phases(value int32) error {
	C.ctx_Vsources_Set_Phases(vsources.ctxPtr, (C.int32_t)(value))
	return vsources.ctx.DSSError()
}

// Per-unit value of source voltage
func (vsources *IVsources) Get_pu() (float64, error) {
	return (float64)(C.ctx_Vsources_Get_pu(vsources.ctxPtr)), vsources.ctx.DSSError()
}

func (vsources *IVsources) Set_pu(value float64) error {
	C.ctx_Vsources_Set_pu(vsources.ctxPtr, (C.double)(value))
	return vsources.ctx.DSSError()
}

type IWireData struct {
	ICommonData
}

func (wiredata *IWireData) Init(ctx *DSSContextPtrs) {
	wiredata.InitCommon(ctx)
}

// Array of strings with all WireData names in the circuit.
func (wiredata *IWireData) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_WireData_Get_AllNames(wiredata.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return wiredata.ctx.GetStringArray(data, cnt)
}

// Number of WireData objects in active circuit.
func (wiredata *IWireData) Count() (int32, error) {
	return (int32)(C.ctx_WireData_Get_Count(wiredata.ctxPtr)), wiredata.ctx.DSSError()
}

// Sets the first WireData active. Returns 0 if no more.
func (wiredata *IWireData) First() (int32, error) {
	return (int32)(C.ctx_WireData_Get_First(wiredata.ctxPtr)), wiredata.ctx.DSSError()
}

// Sets the active WireData by Name.
func (wiredata *IWireData) Get_Name() (string, error) {
	result := C.GoString(C.ctx_WireData_Get_Name(wiredata.ctxPtr))
	return result, wiredata.ctx.DSSError()
}

// Gets the name of the active WireData.
func (wiredata *IWireData) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_WireData_Set_Name(wiredata.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return wiredata.ctx.DSSError()
}

// Sets the next WireData active. Returns 0 if no more.
func (wiredata *IWireData) Next() (int32, error) {
	return (int32)(C.ctx_WireData_Get_Next(wiredata.ctxPtr)), wiredata.ctx.DSSError()
}

// Get the index of the active WireData; index is 1-based: 1..count
func (wiredata *IWireData) Get_idx() (int32, error) {
	return (int32)(C.ctx_WireData_Get_idx(wiredata.ctxPtr)), wiredata.ctx.DSSError()
}

// Set the active WireData by index; index is 1-based: 1..count
func (wiredata *IWireData) Set_idx(value int32) error {
	C.ctx_WireData_Set_idx(wiredata.ctxPtr, (C.int32_t)(value))
	return wiredata.ctx.DSSError()
}

// Emergency ampere rating
func (wiredata *IWireData) Get_EmergAmps() (float64, error) {
	return (float64)(C.ctx_WireData_Get_EmergAmps(wiredata.ctxPtr)), wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Set_EmergAmps(value float64) error {
	C.ctx_WireData_Set_EmergAmps(wiredata.ctxPtr, (C.double)(value))
	return wiredata.ctx.DSSError()
}

// Normal Ampere rating
func (wiredata *IWireData) Get_NormAmps() (float64, error) {
	return (float64)(C.ctx_WireData_Get_NormAmps(wiredata.ctxPtr)), wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Set_NormAmps(value float64) error {
	C.ctx_WireData_Set_NormAmps(wiredata.ctxPtr, (C.double)(value))
	return wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Get_Rdc() (float64, error) {
	return (float64)(C.ctx_WireData_Get_Rdc(wiredata.ctxPtr)), wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Set_Rdc(value float64) error {
	C.ctx_WireData_Set_Rdc(wiredata.ctxPtr, (C.double)(value))
	return wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Get_Rac() (float64, error) {
	return (float64)(C.ctx_WireData_Get_Rac(wiredata.ctxPtr)), wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Set_Rac(value float64) error {
	C.ctx_WireData_Set_Rac(wiredata.ctxPtr, (C.double)(value))
	return wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Get_GMRac() (float64, error) {
	return (float64)(C.ctx_WireData_Get_GMRac(wiredata.ctxPtr)), wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Set_GMRac(value float64) error {
	C.ctx_WireData_Set_GMRac(wiredata.ctxPtr, (C.double)(value))
	return wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Get_GMRUnits() (LineUnits, error) {
	return (LineUnits)(C.ctx_WireData_Get_GMRUnits(wiredata.ctxPtr)), wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Set_GMRUnits(value LineUnits) error {
	C.ctx_WireData_Set_GMRUnits(wiredata.ctxPtr, (C.int32_t)(value))
	return wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Get_Radius() (float64, error) {
	return (float64)(C.ctx_WireData_Get_Radius(wiredata.ctxPtr)), wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Set_Radius(value float64) error {
	C.ctx_WireData_Set_Radius(wiredata.ctxPtr, (C.double)(value))
	return wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Get_RadiusUnits() (int32, error) {
	return (int32)(C.ctx_WireData_Get_RadiusUnits(wiredata.ctxPtr)), wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Set_RadiusUnits(value int32) error {
	C.ctx_WireData_Set_RadiusUnits(wiredata.ctxPtr, (C.int32_t)(value))
	return wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Get_ResistanceUnits() (LineUnits, error) {
	return (LineUnits)(C.ctx_WireData_Get_ResistanceUnits(wiredata.ctxPtr)), wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Set_ResistanceUnits(value LineUnits) error {
	C.ctx_WireData_Set_ResistanceUnits(wiredata.ctxPtr, (C.int32_t)(value))
	return wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Get_Diameter() (float64, error) {
	return (float64)(C.ctx_WireData_Get_Diameter(wiredata.ctxPtr)), wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Set_Diameter(value float64) error {
	C.ctx_WireData_Set_Diameter(wiredata.ctxPtr, (C.double)(value))
	return wiredata.ctx.DSSError()
}

// Equivalent conductor radius for capacitance calcs. Specify this for bundled conductors. Defaults to same value as radius.
func (wiredata *IWireData) Get_CapRadius() (float64, error) {
	return (float64)(C.ctx_WireData_Get_CapRadius(wiredata.ctxPtr)), wiredata.ctx.DSSError()
}

func (wiredata *IWireData) Set_CapRadius(value float64) error {
	C.ctx_WireData_Set_CapRadius(wiredata.ctxPtr, (C.double)(value))
	return wiredata.ctx.DSSError()
}

type IXYCurves struct {
	ICommonData
}

func (xycurves *IXYCurves) Init(ctx *DSSContextPtrs) {
	xycurves.InitCommon(ctx)
}

// Array of strings with all XYCurve names in the circuit.
func (xycurves *IXYCurves) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_XYCurves_Get_AllNames(xycurves.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return xycurves.ctx.GetStringArray(data, cnt)
}

// Number of XYCurve objects in active circuit.
func (xycurves *IXYCurves) Count() (int32, error) {
	return (int32)(C.ctx_XYCurves_Get_Count(xycurves.ctxPtr)), xycurves.ctx.DSSError()
}

// Sets the first XYCurve active. Returns 0 if no more.
func (xycurves *IXYCurves) First() (int32, error) {
	return (int32)(C.ctx_XYCurves_Get_First(xycurves.ctxPtr)), xycurves.ctx.DSSError()
}

// Sets the active XYCurve by Name.
func (xycurves *IXYCurves) Get_Name() (string, error) {
	result := C.GoString(C.ctx_XYCurves_Get_Name(xycurves.ctxPtr))
	return result, xycurves.ctx.DSSError()
}

// Gets the name of the active XYCurve.
func (xycurves *IXYCurves) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_XYCurves_Set_Name(xycurves.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return xycurves.ctx.DSSError()
}

// Sets the next XYCurve active. Returns 0 if no more.
func (xycurves *IXYCurves) Next() (int32, error) {
	return (int32)(C.ctx_XYCurves_Get_Next(xycurves.ctxPtr)), xycurves.ctx.DSSError()
}

// Get the index of the active XYCurve; index is 1-based: 1..count
func (xycurves *IXYCurves) Get_idx() (int32, error) {
	return (int32)(C.ctx_XYCurves_Get_idx(xycurves.ctxPtr)), xycurves.ctx.DSSError()
}

// Set the active XYCurve by index; index is 1-based: 1..count
func (xycurves *IXYCurves) Set_idx(value int32) error {
	C.ctx_XYCurves_Set_idx(xycurves.ctxPtr, (C.int32_t)(value))
	return xycurves.ctx.DSSError()
}

// Get/Set Number of points in X-Y curve
func (xycurves *IXYCurves) Get_Npts() (int32, error) {
	return (int32)(C.ctx_XYCurves_Get_Npts(xycurves.ctxPtr)), xycurves.ctx.DSSError()
}

func (xycurves *IXYCurves) Set_Npts(value int32) error {
	C.ctx_XYCurves_Set_Npts(xycurves.ctxPtr, (C.int32_t)(value))
	return xycurves.ctx.DSSError()
}

// Get/set X values as a Array of doubles. Set Npts to max number expected if setting
func (xycurves *IXYCurves) Get_Xarray() ([]float64, error) {
	C.ctx_XYCurves_Get_Xarray_GR(xycurves.ctxPtr)
	return xycurves.ctx.GetFloat64ArrayGR()
}

func (xycurves *IXYCurves) Set_Xarray(value []float64) error {
	C.ctx_XYCurves_Set_Xarray(xycurves.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return xycurves.ctx.DSSError()
}

// Factor to scale X values from original curve
func (xycurves *IXYCurves) Get_Xscale() (float64, error) {
	return (float64)(C.ctx_XYCurves_Get_Xscale(xycurves.ctxPtr)), xycurves.ctx.DSSError()
}

func (xycurves *IXYCurves) Set_Xscale(value float64) error {
	C.ctx_XYCurves_Set_Xscale(xycurves.ctxPtr, (C.double)(value))
	return xycurves.ctx.DSSError()
}

// Amount to shift X value from original curve
func (xycurves *IXYCurves) Get_Xshift() (float64, error) {
	return (float64)(C.ctx_XYCurves_Get_Xshift(xycurves.ctxPtr)), xycurves.ctx.DSSError()
}

func (xycurves *IXYCurves) Set_Xshift(value float64) error {
	C.ctx_XYCurves_Set_Xshift(xycurves.ctxPtr, (C.double)(value))
	return xycurves.ctx.DSSError()
}

// Get/Set Y values in curve; Set Npts to max number expected if setting
func (xycurves *IXYCurves) Get_Yarray() ([]float64, error) {
	C.ctx_XYCurves_Get_Yarray_GR(xycurves.ctxPtr)
	return xycurves.ctx.GetFloat64ArrayGR()
}

func (xycurves *IXYCurves) Set_Yarray(value []float64) error {
	C.ctx_XYCurves_Set_Yarray(xycurves.ctxPtr, (*C.double)(&value[0]), (C.int32_t)(len(value)))
	return xycurves.ctx.DSSError()
}

// Factor to scale Y values from original curve
func (xycurves *IXYCurves) Get_Yscale() (float64, error) {
	return (float64)(C.ctx_XYCurves_Get_Yscale(xycurves.ctxPtr)), xycurves.ctx.DSSError()
}

func (xycurves *IXYCurves) Set_Yscale(value float64) error {
	C.ctx_XYCurves_Set_Yscale(xycurves.ctxPtr, (C.double)(value))
	return xycurves.ctx.DSSError()
}

// Amount to shift Y value from original curve
func (xycurves *IXYCurves) Get_Yshift() (float64, error) {
	return (float64)(C.ctx_XYCurves_Get_Yshift(xycurves.ctxPtr)), xycurves.ctx.DSSError()
}

func (xycurves *IXYCurves) Set_Yshift(value float64) error {
	C.ctx_XYCurves_Set_Yshift(xycurves.ctxPtr, (C.double)(value))
	return xycurves.ctx.DSSError()
}

// Set X value or get interpolated value after setting Y
func (xycurves *IXYCurves) Get_x() (float64, error) {
	return (float64)(C.ctx_XYCurves_Get_x(xycurves.ctxPtr)), xycurves.ctx.DSSError()
}

func (xycurves *IXYCurves) Set_x(value float64) error {
	C.ctx_XYCurves_Set_x(xycurves.ctxPtr, (C.double)(value))
	return xycurves.ctx.DSSError()
}

// Set Y value or get interpolated Y value after setting X
func (xycurves *IXYCurves) Get_y() (float64, error) {
	return (float64)(C.ctx_XYCurves_Get_y(xycurves.ctxPtr)), xycurves.ctx.DSSError()
}

func (xycurves *IXYCurves) Set_y(value float64) error {
	C.ctx_XYCurves_Set_y(xycurves.ctxPtr, (C.double)(value))
	return xycurves.ctx.DSSError()
}

type IYMatrix struct {
	ICommonData
}

func (ymatrix *IYMatrix) Init(ctx *DSSContextPtrs) {
	ymatrix.InitCommon(ctx)
}

func (ymatrix *IYMatrix) ZeroInjCurr() error {
	C.ctx_YMatrix_ZeroInjCurr(ymatrix.ctxPtr)
	return ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) GetSourceInjCurrents() error {
	C.ctx_YMatrix_GetSourceInjCurrents(ymatrix.ctxPtr)
	return ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) GetPCInjCurr() error {
	C.ctx_YMatrix_GetPCInjCurr(ymatrix.ctxPtr)
	return ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) BuildYMatrixD(BuildOps int32, AllocateVI int32) error {
	C.ctx_YMatrix_BuildYMatrixD(ymatrix.ctxPtr, (C.int32_t)(BuildOps), (C.int32_t)(AllocateVI))
	return ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) AddInAuxCurrents(SType int32) error {
	C.ctx_YMatrix_AddInAuxCurrents(ymatrix.ctxPtr, (C.int32_t)(SType))
	return ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) Get_SystemYChanged() (bool, error) {
	return (C.ctx_YMatrix_Get_SystemYChanged(ymatrix.ctxPtr) != 0), ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) Set_SystemYChanged(value bool) error {
	C.ctx_YMatrix_Set_SystemYChanged(ymatrix.ctxPtr, ToUint16(value))
	return ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) Get_UseAuxCurrents() (bool, error) {
	return (C.ctx_YMatrix_Get_UseAuxCurrents(ymatrix.ctxPtr) != 0), ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) Set_UseAuxCurrents(value bool) error {
	C.ctx_YMatrix_Set_UseAuxCurrents(ymatrix.ctxPtr, ToUint16(value))
	return ymatrix.ctx.DSSError()
}

// Sparse solver options. See the enumeration SparseSolverOptions
func (ymatrix *IYMatrix) Get_SolverOptions() (uint64, error) {
	return (uint64)(C.ctx_YMatrix_Get_SolverOptions(ymatrix.ctxPtr)), ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) Set_SolverOptions(value uint64) error {
	C.ctx_YMatrix_Set_SolverOptions(ymatrix.ctxPtr, (C.uint64_t)(value))
	return ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) CheckConvergence() (bool, error) {
	return (C.ctx_YMatrix_CheckConvergence(ymatrix.ctxPtr) != 0), ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) SetGeneratordQdV() error {
	C.ctx_YMatrix_SetGeneratordQdV(ymatrix.ctxPtr)
	return ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) Get_LoadsNeedUpdating() (bool, error) {
	return (C.ctx_YMatrix_Get_LoadsNeedUpdating(ymatrix.ctxPtr) != 0), ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) Set_LoadsNeedUpdating(value bool) error {
	C.ctx_YMatrix_Set_LoadsNeedUpdating(ymatrix.ctxPtr, ToUint16(value))
	return ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) Get_SolutionInitialized() (bool, error) {
	return (C.ctx_YMatrix_Get_SolutionInitialized(ymatrix.ctxPtr) != 0), ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) Set_SolutionInitialized(value bool) error {
	C.ctx_YMatrix_Set_SolutionInitialized(ymatrix.ctxPtr, ToUint16(value))
	return ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) Get_Iteration() (int32, error) {
	return (int32)(C.ctx_YMatrix_Get_Iteration(ymatrix.ctxPtr)), ymatrix.ctx.DSSError()
}

func (ymatrix *IYMatrix) Set_Iteration(value int32) error {
	C.ctx_YMatrix_Set_Iteration(ymatrix.ctxPtr, (C.int32_t)(value))
	return ymatrix.ctx.DSSError()
}

type IZIP struct {
	ICommonData
}

func (zip *IZIP) Init(ctx *DSSContextPtrs) {
	zip.InitCommon(ctx)
}

// Extracts the contents of the file "FileName" from the current (open) ZIP file.
// Returns a byte-string.
//
// (API Extension)
func (zip *IZIP) Extract(fileName string) ([]byte, error) {
	fileName_c := C.CString(fileName)
	C.ctx_ZIP_Extract_GR(zip.ctxPtr, fileName_c)
	C.free(unsafe.Pointer(fileName_c))
	return zip.ctx.GetUInt8ArrayGR()
}

// List of strings consisting of all names match the regular expression provided in regexp.
// If no expression is provided (empty string), all names in the current open ZIP are returned.
//
// See https://regex.sorokin.engineer/en/latest/regular_expressions.html for information on
// the expression syntax and options.
//
// (API Extension)
func (zip *IZIP) List(regexp string) ([]string, error) {
	var cnt [4]int32
	var data **C.char
	regexp_c := C.CString(regexp)
	C.ctx_ZIP_List(zip.ctxPtr, &data, (*C.int32_t)(&cnt[0]), regexp_c)
	C.free(unsafe.Pointer(regexp_c))
	return zip.ctx.GetStringArray(data, cnt)
}

// Opens and prepares a ZIP file to be used by the DSS text parser.
// Currently, the ZIP format support is limited by what is provided in the Free Pascal distribution.
// Besides that, the full filenames inside the ZIP must be shorter than 256 characters.
// The limitations should be removed in a future revision.
//
// (API Extension)
func (zip *IZIP) Open(FileName string) error {
	FileName_c := C.CString(FileName)
	C.ctx_ZIP_Open(zip.ctxPtr, FileName_c)
	C.free(unsafe.Pointer(FileName_c))
	return zip.ctx.DSSError()
}

// Closes the current open ZIP file
//
// (API Extension)
func (zip *IZIP) Close() error {
	C.ctx_ZIP_Close(zip.ctxPtr)
	return zip.ctx.DSSError()
}

// Runs a "Redirect" command inside the current (open) ZIP file.
// In the current implementation, all files required by the script must
// be present inside the ZIP, using relative paths. The only exceptions are
// memory-mapped files.
//
// (API Extension)
func (zip *IZIP) Redirect(FileInZip string) error {
	FileInZip_c := C.CString(FileInZip)
	C.ctx_ZIP_Redirect(zip.ctxPtr, FileInZip_c)
	C.free(unsafe.Pointer(FileInZip_c))
	return zip.ctx.DSSError()
}

// Check if the given path name is present in the current ZIP file.
//
// (API Extension)
func (zip *IZIP) Contains(Name string) (bool, error) {
	Name_c := C.CString(Name)
	defer C.free(unsafe.Pointer(Name_c))
	return (C.ctx_ZIP_Contains(zip.ctxPtr, Name_c) != 0), zip.ctx.DSSError()
}

type IGICSources struct {
	ICommonData
}

func (gicsources *IGICSources) Init(ctx *DSSContextPtrs) {
	gicsources.InitCommon(ctx)
}

// Array of strings with all GICSource names in the circuit.
func (gicsources *IGICSources) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_GICSources_Get_AllNames(gicsources.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return gicsources.ctx.GetStringArray(data, cnt)
}

// Number of GICSource objects in active circuit.
func (gicsources *IGICSources) Count() (int32, error) {
	return (int32)(C.ctx_GICSources_Get_Count(gicsources.ctxPtr)), gicsources.ctx.DSSError()
}

// Sets the first GICSource active. Returns 0 if no more.
func (gicsources *IGICSources) First() (int32, error) {
	return (int32)(C.ctx_GICSources_Get_First(gicsources.ctxPtr)), gicsources.ctx.DSSError()
}

// Sets the active GICSource by Name.
func (gicsources *IGICSources) Get_Name() (string, error) {
	result := C.GoString(C.ctx_GICSources_Get_Name(gicsources.ctxPtr))
	return result, gicsources.ctx.DSSError()
}

// Gets the name of the active GICSource.
func (gicsources *IGICSources) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_GICSources_Set_Name(gicsources.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return gicsources.ctx.DSSError()
}

// Sets the next GICSource active. Returns 0 if no more.
func (gicsources *IGICSources) Next() (int32, error) {
	return (int32)(C.ctx_GICSources_Get_Next(gicsources.ctxPtr)), gicsources.ctx.DSSError()
}

// Get the index of the active GICSource; index is 1-based: 1..count
func (gicsources *IGICSources) Get_idx() (int32, error) {
	return (int32)(C.ctx_GICSources_Get_idx(gicsources.ctxPtr)), gicsources.ctx.DSSError()
}

// Set the active GICSource by index; index is 1-based: 1..count
func (gicsources *IGICSources) Set_idx(value int32) error {
	C.ctx_GICSources_Set_idx(gicsources.ctxPtr, (C.int32_t)(value))
	return gicsources.ctx.DSSError()
}

// First bus name of GICSource (Created name)
func (gicsources *IGICSources) Bus1() (string, error) {
	return C.GoString(C.ctx_GICSources_Get_Bus1(gicsources.ctxPtr)), gicsources.ctx.DSSError()
}

// Second bus name
func (gicsources *IGICSources) Bus2() (string, error) {
	return C.GoString(C.ctx_GICSources_Get_Bus2(gicsources.ctxPtr)), gicsources.ctx.DSSError()
}

// Number of Phases, this GICSource element.
func (gicsources *IGICSources) Get_Phases() (int32, error) {
	return (int32)(C.ctx_GICSources_Get_Phases(gicsources.ctxPtr)), gicsources.ctx.DSSError()
}

func (gicsources *IGICSources) Set_Phases(value int32) error {
	C.ctx_GICSources_Set_Phases(gicsources.ctxPtr, (C.int32_t)(value))
	return gicsources.ctx.DSSError()
}

// Northward E Field V/km
func (gicsources *IGICSources) Get_EN() (float64, error) {
	return (float64)(C.ctx_GICSources_Get_EN(gicsources.ctxPtr)), gicsources.ctx.DSSError()
}

func (gicsources *IGICSources) Set_EN(value float64) error {
	C.ctx_GICSources_Set_EN(gicsources.ctxPtr, (C.double)(value))
	return gicsources.ctx.DSSError()
}

// Eastward E Field, V/km
func (gicsources *IGICSources) Get_EE() (float64, error) {
	return (float64)(C.ctx_GICSources_Get_EE(gicsources.ctxPtr)), gicsources.ctx.DSSError()
}

func (gicsources *IGICSources) Set_EE(value float64) error {
	C.ctx_GICSources_Set_EE(gicsources.ctxPtr, (C.double)(value))
	return gicsources.ctx.DSSError()
}

// Latitude of Bus1 (degrees)
func (gicsources *IGICSources) Get_Lat1() (float64, error) {
	return (float64)(C.ctx_GICSources_Get_Lat1(gicsources.ctxPtr)), gicsources.ctx.DSSError()
}

func (gicsources *IGICSources) Set_Lat1(value float64) error {
	C.ctx_GICSources_Set_Lat1(gicsources.ctxPtr, (C.double)(value))
	return gicsources.ctx.DSSError()
}

// Latitude of Bus2 (degrees)
func (gicsources *IGICSources) Get_Lat2() (float64, error) {
	return (float64)(C.ctx_GICSources_Get_Lat2(gicsources.ctxPtr)), gicsources.ctx.DSSError()
}

func (gicsources *IGICSources) Set_Lat2(value float64) error {
	C.ctx_GICSources_Set_Lat2(gicsources.ctxPtr, (C.double)(value))
	return gicsources.ctx.DSSError()
}

// Longitude of Bus1 (Degrees)
func (gicsources *IGICSources) Get_Lon1() (float64, error) {
	return (float64)(C.ctx_GICSources_Get_Lon1(gicsources.ctxPtr)), gicsources.ctx.DSSError()
}

func (gicsources *IGICSources) Set_Lon1(value float64) error {
	C.ctx_GICSources_Set_Lon1(gicsources.ctxPtr, (C.double)(value))
	return gicsources.ctx.DSSError()
}

// Longitude of Bus2 (Degrees)
func (gicsources *IGICSources) Get_Lon2() (float64, error) {
	return (float64)(C.ctx_GICSources_Get_Lon2(gicsources.ctxPtr)), gicsources.ctx.DSSError()
}

func (gicsources *IGICSources) Set_Lon2(value float64) error {
	C.ctx_GICSources_Set_Lon2(gicsources.ctxPtr, (C.double)(value))
	return gicsources.ctx.DSSError()
}

// Specify dc voltage directly
func (gicsources *IGICSources) Get_Volts() (float64, error) {
	return (float64)(C.ctx_GICSources_Get_Volts(gicsources.ctxPtr)), gicsources.ctx.DSSError()
}

func (gicsources *IGICSources) Set_Volts(value float64) error {
	C.ctx_GICSources_Set_Volts(gicsources.ctxPtr, (C.double)(value))
	return gicsources.ctx.DSSError()
}

type IParallel struct {
	ICommonData
}

func (parallel *IParallel) Init(ctx *DSSContextPtrs) {
	parallel.InitCommon(ctx)
}

func (parallel *IParallel) CreateActor() error {
	C.ctx_Parallel_CreateActor(parallel.ctxPtr)
	return parallel.ctx.DSSError()
}

func (parallel *IParallel) Wait() error {
	C.ctx_Parallel_Wait(parallel.ctxPtr)
	return parallel.ctx.DSSError()
}

// Gets/sets the ID of the Active Actor
func (parallel *IParallel) Get_ActiveActor() (int32, error) {
	return (int32)(C.ctx_Parallel_Get_ActiveActor(parallel.ctxPtr)), parallel.ctx.DSSError()
}

func (parallel *IParallel) Set_ActiveActor(value int32) error {
	C.ctx_Parallel_Set_ActiveActor(parallel.ctxPtr, (C.int32_t)(value))
	return parallel.ctx.DSSError()
}

// (read) Sets ON/OFF (1/0) Parallel features of the Engine
// (write) Delivers if the Parallel features of the Engine are Active
func (parallel *IParallel) Get_ActiveParallel() (int32, error) {
	return (int32)(C.ctx_Parallel_Get_ActiveParallel(parallel.ctxPtr)), parallel.ctx.DSSError()
}

func (parallel *IParallel) Set_ActiveParallel(value int32) error {
	C.ctx_Parallel_Set_ActiveParallel(parallel.ctxPtr, (C.int32_t)(value))
	return parallel.ctx.DSSError()
}

// Gets/sets the CPU of the Active Actor
func (parallel *IParallel) Get_ActorCPU() (int32, error) {
	return (int32)(C.ctx_Parallel_Get_ActorCPU(parallel.ctxPtr)), parallel.ctx.DSSError()
}

func (parallel *IParallel) Set_ActorCPU(value int32) error {
	C.ctx_Parallel_Set_ActorCPU(parallel.ctxPtr, (C.int32_t)(value))
	return parallel.ctx.DSSError()
}

// Gets the progress of all existing actors in pct
func (parallel *IParallel) ActorProgress() ([]int32, error) {
	C.ctx_Parallel_Get_ActorProgress_GR(parallel.ctxPtr)
	return parallel.ctx.GetInt32ArrayGR()
}

// Gets the status of each actor
func (parallel *IParallel) ActorStatus() ([]int32, error) {
	C.ctx_Parallel_Get_ActorStatus_GR(parallel.ctxPtr)
	return parallel.ctx.GetInt32ArrayGR()
}

// (read) Reads the values of the ConcatenateReports option (1=enabled, 0=disabled)
// (write) Enable/Disable (1/0) the ConcatenateReports option for extracting monitors data
func (parallel *IParallel) Get_ConcatenateReports() (int32, error) {
	return (int32)(C.ctx_Parallel_Get_ConcatenateReports(parallel.ctxPtr)), parallel.ctx.DSSError()
}

func (parallel *IParallel) Set_ConcatenateReports(value int32) error {
	C.ctx_Parallel_Set_ConcatenateReports(parallel.ctxPtr, (C.int32_t)(value))
	return parallel.ctx.DSSError()
}

// Delivers the number of CPUs on the current PC
func (parallel *IParallel) NumCPUs() (int32, error) {
	return (int32)(C.ctx_Parallel_Get_NumCPUs(parallel.ctxPtr)), parallel.ctx.DSSError()
}

// Delivers the number of Cores of the local PC
func (parallel *IParallel) NumCores() (int32, error) {
	return (int32)(C.ctx_Parallel_Get_NumCores(parallel.ctxPtr)), parallel.ctx.DSSError()
}

// Gets the number of Actors created
func (parallel *IParallel) NumOfActors() (int32, error) {
	return (int32)(C.ctx_Parallel_Get_NumOfActors(parallel.ctxPtr)), parallel.ctx.DSSError()
}

type IStorages struct {
	ICommonData
}

func (storages *IStorages) Init(ctx *DSSContextPtrs) {
	storages.InitCommon(ctx)
}

// Array of strings with all Storage names in the circuit.
func (storages *IStorages) AllNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Storages_Get_AllNames(storages.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return storages.ctx.GetStringArray(data, cnt)
}

// Number of Storage objects in active circuit.
func (storages *IStorages) Count() (int32, error) {
	return (int32)(C.ctx_Storages_Get_Count(storages.ctxPtr)), storages.ctx.DSSError()
}

// Sets the first Storage active. Returns 0 if no more.
func (storages *IStorages) First() (int32, error) {
	return (int32)(C.ctx_Storages_Get_First(storages.ctxPtr)), storages.ctx.DSSError()
}

// Sets the active Storage by Name.
func (storages *IStorages) Get_Name() (string, error) {
	result := C.GoString(C.ctx_Storages_Get_Name(storages.ctxPtr))
	return result, storages.ctx.DSSError()
}

// Gets the name of the active Storage.
func (storages *IStorages) Set_Name(value string) error {
	value_c := C.CString(value)
	C.ctx_Storages_Set_Name(storages.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return storages.ctx.DSSError()
}

// Sets the next Storage active. Returns 0 if no more.
func (storages *IStorages) Next() (int32, error) {
	return (int32)(C.ctx_Storages_Get_Next(storages.ctxPtr)), storages.ctx.DSSError()
}

// Get the index of the active Storage; index is 1-based: 1..count
func (storages *IStorages) Get_idx() (int32, error) {
	return (int32)(C.ctx_Storages_Get_idx(storages.ctxPtr)), storages.ctx.DSSError()
}

// Set the active Storage by index; index is 1-based: 1..count
func (storages *IStorages) Set_idx(value int32) error {
	C.ctx_Storages_Set_idx(storages.ctxPtr, (C.int32_t)(value))
	return storages.ctx.DSSError()
}

// Per unit state of charge
func (storages *IStorages) Get_puSOC() (float64, error) {
	return (float64)(C.ctx_Storages_Get_puSOC(storages.ctxPtr)), storages.ctx.DSSError()
}

func (storages *IStorages) Set_puSOC(value float64) error {
	C.ctx_Storages_Set_puSOC(storages.ctxPtr, (C.double)(value))
	return storages.ctx.DSSError()
}

// Get/set state: 0=Idling; 1=Discharging; -1=Charging;
//
// Related enumeration: StorageStates
func (storages *IStorages) Get_State() (int32, error) {
	return (int32)(C.ctx_Storages_Get_State(storages.ctxPtr)), storages.ctx.DSSError()
}

func (storages *IStorages) Set_State(value int32) error {
	C.ctx_Storages_Set_State(storages.ctxPtr, (C.int32_t)(value))
	return storages.ctx.DSSError()
}

// Array of Names of all Storage energy meter registers
func (storages *IStorages) RegisterNames() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_Storages_Get_RegisterNames(storages.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return storages.ctx.GetStringArray(data, cnt)
}

// Array of values in Storage registers.
func (storages *IStorages) RegisterValues() ([]float64, error) {
	C.ctx_Storages_Get_RegisterValues_GR(storages.ctxPtr)
	return storages.ctx.GetFloat64ArrayGR()
}

type IDSS struct {
	ICommonData

	ActiveCircuit ICircuit
	Circuits      ICircuit
	Error         IError
	Text          IText
	DSSProgress   IDSSProgress
	ActiveClass   IActiveClass
	Executive     IDSS_Executive
	// Events IDSSEvents
	Parser IParser
	// DSSim_Coms IDSSimComs
	YMatrix IYMatrix
	ZIP     IZIP
}

// Initialize all structures of the classic DSS API.
//
// For creating new independent DSS instances, use the function NewContext.
func (dss *IDSS) Init(ctxPtr unsafe.Pointer) {
	if ctxPtr == nil {
		ctxPtr = C.ctx_Get_Prime()
		C.ctx_DSS_Start(ctxPtr, 0)
	}
	dss.ctx = &DSSContextPtrs{}
	dss.ctxPtr = ctxPtr
	dss.ctx.Init(dss.ctxPtr)
	ctx := dss.ctx
	dss.ActiveCircuit.Init(ctx)
	dss.Circuits.Init(ctx)
	dss.Error.Init(ctx)
	dss.Text.Init(ctx)
	dss.DSSProgress.Init(ctx)
	dss.ActiveClass.Init(ctx)
	dss.Executive.Init(ctx)
	// dss.Events.Init(ctx)
	dss.Parser.Init(ctx)
	// dss.DSSim_Coms.Init(ctx)
	dss.YMatrix.Init(ctx)
	dss.ZIP.Init(ctx)
}

// Creates a new DSS engine context.
// A DSS Context encapsulates most of the global state of the original OpenDSS engine,
// allowing the user to create multiple instances in the same process. By creating contexts
// manually, the management of threads and potential issues should be handled by the user.
//
// (API Extension)
func (dss *IDSS) NewContext() (*IDSS, error) {
	newCtxPtr := C.ctx_New()
	dssNew := &IDSS{}
	dssNew.Init(newCtxPtr)
	if newCtxPtr == nil {
		return dssNew, errors.New("(DSSError) Could not create a new DSS Context")
	}
	return dssNew, nil
}

func (dss *IDSS) NewCircuit(name string) (*ICircuit, error) {
	name_c := C.CString(name)
	C.ctx_DSS_NewCircuit(dss.ctxPtr, name_c)
	C.free(unsafe.Pointer(name_c))
	return &dss.ActiveCircuit, dss.ctx.DSSError()
}

func (dss *IDSS) ClearAll() error {
	C.ctx_DSS_ClearAll(dss.ctxPtr)
	return dss.ctx.DSSError()
}

// This is a no-op function, does nothing. Left for compatibility.
func (dss *IDSS) Reset() error {
	C.ctx_DSS_Reset(dss.ctxPtr)
	return dss.ctx.DSSError()
}

func (dss *IDSS) SetActiveClass(ClassName string) (int32, error) {
	ClassName_c := C.CString(ClassName)
	defer C.free(unsafe.Pointer(ClassName_c))
	return (int32)(C.ctx_DSS_SetActiveClass(dss.ctxPtr, ClassName_c)), dss.ctx.DSSError()
}

// This is a no-op function, does nothing. Left for compatibility.
//
// Calling `Start` in AltDSS/DSS-Extensions is required but that is already
// handled automatically, so the users do not need to call it manually.
//
// On the official OpenDSS, `Start` also does nothing at all in the current
// versions.
func (dss *IDSS) Start(code int32) (bool, error) {
	return (C.ctx_DSS_Start(dss.ctxPtr, (C.int32_t)(code)) != 0), dss.ctx.DSSError()
}

// List of DSS intrinsic classes (names of the classes)
func (dss *IDSS) Classes() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_DSS_Get_Classes(dss.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return dss.ctx.GetStringArray(data, cnt)
}

// DSS Data File Path.  Default path for reports, etc. from DSS
func (dss *IDSS) Get_DataPath() (string, error) {
	return C.GoString(C.ctx_DSS_Get_DataPath(dss.ctxPtr)), dss.ctx.DSSError()
}

func (dss *IDSS) Set_DataPath(value string) error {
	value_c := C.CString(value)
	C.ctx_DSS_Set_DataPath(dss.ctxPtr, value_c)
	C.free(unsafe.Pointer(value_c))
	return dss.ctx.DSSError()
}

// Returns the path name for the default text editor.
func (dss *IDSS) DefaultEditor() (string, error) {
	return C.GoString(C.ctx_DSS_Get_DefaultEditor(dss.ctxPtr)), dss.ctx.DSSError()
}

// Number of Circuits currently defined
func (dss *IDSS) NumCircuits() (int32, error) {
	return (int32)(C.ctx_DSS_Get_NumCircuits(dss.ctxPtr)), dss.ctx.DSSError()
}

// Number of DSS intrinsic classes
func (dss *IDSS) NumClasses() (int32, error) {
	return (int32)(C.ctx_DSS_Get_NumClasses(dss.ctxPtr)), dss.ctx.DSSError()
}

// Number of user-defined classes
func (dss *IDSS) NumUserClasses() (int32, error) {
	return (int32)(C.ctx_DSS_Get_NumUserClasses(dss.ctxPtr)), dss.ctx.DSSError()
}

// List of user-defined classes
func (dss *IDSS) UserClasses() ([]string, error) {
	var cnt [4]int32
	var data **C.char
	C.ctx_DSS_Get_UserClasses(dss.ctxPtr, &data, (*C.int32_t)(&cnt[0]))
	return dss.ctx.GetStringArray(data, cnt)
}

// Get version string for the DSS.
func (dss *IDSS) Version() (string, error) {
	return C.GoString(C.ctx_DSS_Get_Version(dss.ctxPtr)), dss.ctx.DSSError()
}

// Gets/sets whether text output is allowed
func (dss *IDSS) Get_AllowForms() (bool, error) {
	return (C.ctx_DSS_Get_AllowForms(dss.ctxPtr) != 0), dss.ctx.DSSError()
}

func (dss *IDSS) Set_AllowForms(value bool) error {
	C.ctx_DSS_Set_AllowForms(dss.ctxPtr, ToUint16(value))
	return dss.ctx.DSSError()
}

// Gets/sets whether running the external editor for "Show" is allowed
//
// AllowEditor controls whether the external editor is used in commands like "Show".
// If you set to 0 (false), the editor is not executed. Note that other side effects,
// such as the creation of files, are not affected.
//
// (API Extension)
func (dss *IDSS) Get_AllowEditor() (bool, error) {
	return (C.ctx_DSS_Get_AllowEditor(dss.ctxPtr) != 0), dss.ctx.DSSError()
}

func (dss *IDSS) Set_AllowEditor(value bool) error {
	C.ctx_DSS_Set_AllowEditor(dss.ctxPtr, ToUint16(value))
	return dss.ctx.DSSError()
}

// LegacyModels was a flag used to toggle legacy (pre-2019) models for PVSystem, InvControl, Storage and
// StorageControl.
// In the official OpenDSS version 9.0, the old models were removed. They were temporarily present here
// but were also removed in DSS C-API v0.13.0.
//
// **NOTE**: this property will be removed for v1.0. It is left to avoid breaking the current API too soon.
//
// (API Extension)
func (dss *IDSS) Get_LegacyModels() (bool, error) {
	return (C.ctx_DSS_Get_LegacyModels(dss.ctxPtr) != 0), dss.ctx.DSSError()
}

func (dss *IDSS) Set_LegacyModels(value bool) error {
	C.ctx_DSS_Set_LegacyModels(dss.ctxPtr, ToUint16(value))
	return dss.ctx.DSSError()
}

// If disabled, the engine will not change the active working directory during execution. E.g. a "compile"
// command will not "chdir" to the file path.
//
// If you have issues with long paths, enabling this might help in some scenarios.
//
// Defaults to True (allow changes, backwards compatible) in the 0.10.x versions of DSS C-API.
// This might change to False in future versions.
//
// This can also be set through the environment variable DSS_CAPI_ALLOW_CHANGE_DIR. Set it to 0 to
// disallow changing the active working directory.
//
// (API Extension)
func (dss *IDSS) Get_AllowChangeDir() (bool, error) {
	return (C.ctx_DSS_Get_AllowChangeDir(dss.ctxPtr) != 0), dss.ctx.DSSError()
}

func (dss *IDSS) Set_AllowChangeDir(value bool) error {
	C.ctx_DSS_Set_AllowChangeDir(dss.ctxPtr, ToUint16(value))
	return dss.ctx.DSSError()
}

// If enabled, the `DOScmd` command is allowed. Otherwise, an error is reported if the user tries to use it.
//
// Defaults to False/0 (disabled state). Users should consider DOScmd deprecated on DSS-Extensions.
//
// This can also be set through the environment variable DSS_CAPI_ALLOW_DOSCMD. Setting it to 1 enables
// the command.
//
// (API Extension)
func (dss *IDSS) Get_AllowDOScmd() (bool, error) {
	return (C.ctx_DSS_Get_AllowDOScmd(dss.ctxPtr) != 0), dss.ctx.DSSError()
}

func (dss *IDSS) Set_AllowDOScmd(value bool) error {
	C.ctx_DSS_Set_AllowDOScmd(dss.ctxPtr, ToUint16(value))
	return dss.ctx.DSSError()
}

// If enabled, in case of errors or empty arrays, the API returns arrays with values compatible with the
// official OpenDSS COM interface.
//
// For example, consider the function `Loads_Get_ZIPV`. If there is no active circuit or active load element:
// - In the disabled state (COMErrorResults=False), the function will return "[]", an array with 0 elements.
// - In the enabled state (COMErrorResults=True), the function will return "[0.0]" instead. This should
// be compatible with the return value of the official COM interface.
//
// Defaults to True/1 (enabled state) in the v0.12.x series. This will change to false in future series.
//
// This can also be set through the environment variable DSS_CAPI_COM_DEFAULTS. Setting it to 0 disables
// the legacy/COM behavior. The value can be toggled through the API at any time.
//
// (API Extension)
func (dss *IDSS) Get_COMErrorResults() (bool, error) {
	return (C.ctx_DSS_Get_COMErrorResults(dss.ctxPtr) != 0), dss.ctx.DSSError()
}

func (dss *IDSS) Set_COMErrorResults(value bool) error {
	C.ctx_DSS_Set_COMErrorResults(dss.ctxPtr, ToUint16(value))
	return dss.ctx.DSSError()
}

// Controls some compatibility flags introduced to toggle some behavior from the official OpenDSS.
//
// **THESE FLAGS ARE GLOBAL, affecting all DSS engines in the process.**
//
// The current bit flags are:
//
//   - 0x1 (bit 0): If enabled, don't check for NaNs in the inner solution loop. This can lead to various errors.
//     This flag is useful for legacy applications that don't handle OpenDSS API errors properly. Through the
//     development of DSS-Extensions, we noticed this is actually a quite common issue.
//   - 0x2 (bit 1): Toggle worse precision for certain aspects of the engine. For example, the sequence-to-phase
//     (`As2p`) and sequence-to-phase (`Ap2s`) transform matrices. On DSS C-API, we fill the matrix explicitly
//     using higher precision, while numerical inversion of an initially worse precision matrix is used in the
//     official OpenDSS. We will introduce better precision for other aspects of the engine in the future,
//     so this flag can be used to toggle the old/bad values where feasible.
//   - 0x4 (bit 2): Toggle some InvControl behavior introduced in OpenDSS 9.6.1.1. It could be a regression
//     but needs further investigation, so we added this flag in the time being.
//
// These flags may change for each version of DSS C-API, but the same value will not be reused. That is,
// when we remove a compatibility flag, it will have no effect but will also not affect anything else
// besides raising an error if the user tries to toggle a flag that was available in a previous version.
//
// We expect to keep a very limited number of flags. Since the flags are more transient than the other
// options/flags, it was preferred to add this generic function instead of a separate function per
// flag.
//
// Related enumeration: DSSCompatFlags
//
// (API Extension)
func (dss *IDSS) Get_CompatFlags() (uint32, error) {
	return (uint32)(C.ctx_DSS_Get_CompatFlags(dss.ctxPtr)), dss.ctx.DSSError()
}

func (dss *IDSS) Set_CompatFlags(value uint32) error {
	C.ctx_DSS_Set_CompatFlags(dss.ctxPtr, (C.uint32_t)(value))
	return dss.ctx.DSSError()
}
