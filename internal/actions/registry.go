package actions

import "reflect"

// actionRegistry maps YAML action names to their concrete Go types
// This enables polymorphic unmarshaling of ActionStep interfaces from YAML
// Actions are mapped lowercase to allow for fuzzy script writing
//
// To add a new action:
// 1. Create a struct that implements the ActionStep interface (Validate & Build methods)
// 2. Add it to this registry with the name that will be used in YAML files
var actionRegistry = map[string]reflect.Type{
	"click":                reflect.TypeOf(Click{}),
	"swipe":                reflect.TypeOf(Swipe{}),
	"input":                reflect.TypeOf(Input{}),
	"send_key":             reflect.TypeOf(SendKey{}),
	"sleep":                reflect.TypeOf(Sleep{}),
	"delay":                reflect.TypeOf(Delay{}),
	"findimage":            reflect.TypeOf(FindImage{}),
	"clickifimagefound":    reflect.TypeOf(ClickIfImageFound{}),
	"clickifimagenotfound": reflect.TypeOf(ClickIfImageNotFound{}),
	"whileimagefound":      reflect.TypeOf(WhileImageFound{}),
	"untilimagefound":      reflect.TypeOf(UntilImageFound{}),
	"whileanyimagesfound":  reflect.TypeOf(WhileAnyImagesFound{}),
	"untilanyimagesfound":  reflect.TypeOf(UntilAnyImagesFound{}),
	"waitforimage":         reflect.TypeOf(WaitForImage{}), // Unlikely to be used
	"repeat":               reflect.TypeOf(Repeat{}),
	"ifimagefound":         reflect.TypeOf(IfImageFound{}),
	"ifimagenotfound":      reflect.TypeOf(IfImageNotFound{}),
	"ifanyimagesfound":     reflect.TypeOf(IfAnyImagesFound{}),
	"ifallimagesfound":     reflect.TypeOf(IfAllImagesFound{}),
	"ifnoimagesfound":      reflect.TypeOf(IfNoImagesFound{}),
	"runroutine":           reflect.TypeOf(RunRoutine{}),
	// Generic control flow with conditions
	"if":    reflect.TypeOf(If{}),
	"while": reflect.TypeOf(While{}),
	"until": reflect.TypeOf(Until{}),
	"break": reflect.TypeOf(Break{}),
	// Variable actions
	"setvariable": reflect.TypeOf(SetVariable{}),
	"getvariable": reflect.TypeOf(GetVariable{}),
	"increment":   reflect.TypeOf(Increment{}),
	"decrement":   reflect.TypeOf(Decrement{}),
	// Account pool actions
	"injectnextaccount":  reflect.TypeOf(InjectNextAccount{}),
	"completeaccount":    reflect.TypeOf(CompleteAccount{}),
	"returnaccount":      reflect.TypeOf(ReturnAccount{}),
	"markaccountfailed":  reflect.TypeOf(MarkAccountFailed{}),
}
