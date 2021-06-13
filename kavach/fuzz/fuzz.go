package fuzz

var ff = [...]interface{}{
	FuzzPermission,
}

func FuzzFuncs() []interface{} {
	return ff[:]
}
