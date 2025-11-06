package actions

import "reflect"

// actionRegistry maps YAML action names to their concrete Go types
// This enables polymorphic unmarshaling of ActionStep interfaces from YAML
//
// To add a new action:
// 1. Create a struct that implements the ActionStep interface (Validate & Build methods)
// 2. Add it to this registry with the name that will be used in YAML files
var actionRegistry = map[string]reflect.Type{
	"click":           reflect.TypeOf(Click{}),
	"swipe":           reflect.TypeOf(Swipe{}),
	"input":           reflect.TypeOf(Input{}),
	"send_key":        reflect.TypeOf(SendKey{}),
	"sleep":           reflect.TypeOf(Sleep{}),
	"delay":           reflect.TypeOf(Delay{}),
	"FindImage":       reflect.TypeOf(FindImage{}),
	"whileimagefound": reflect.TypeOf(WhileImageFound{}),
	"untilanyfound":   reflect.TypeOf(UntilAnyFound{}),
	"repeat":          reflect.TypeOf(Repeat{}),
	// "UntilTemplateAppears": reflect.TypeOf(UntilTemplateAppears{}),
	// "ClickIfFoundOffset":   reflect.TypeOf(ClickIfFoundOffset{}),
}
