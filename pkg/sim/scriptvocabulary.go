package sim

func ScriptOperationSupported(condition bool, op int32) bool {
	if condition {
		return scriptCheckSupported(op)
	}
	return scriptInstantSupported(ScriptInstant{Op: op, Args: [10]int32{1}})
}
