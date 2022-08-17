package builtin

var (
	BuiltinFunc = map[string]interface{}{
		"json": builtinJSONFunction,
	}
)
