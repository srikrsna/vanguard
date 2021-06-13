package fuzz

var ff = [...]interface{}{
	FuzzExample,
	FuzzListExamplesRequest,
	FuzzListExamplesResponse,
	FuzzGetExampleRequest,
	FuzzCreateExampleRequest,
	FuzzUpdateExampleRequest,
	FuzzDeleteExampleRequest,
}

func FuzzFuncs() []interface{} {
	return ff[:]
}
