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
}
